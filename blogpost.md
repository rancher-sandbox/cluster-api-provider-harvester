# Provisioning Production RKE2 Clusters on Harvester HCI with Cluster API

*How I forked an inactive CAPI infrastructure provider for Harvester and brought it to production-grade quality -- validated by a formal SUSE engineering review.*

---

## The Problem

[Harvester](https://harvesterhci.io/) is SUSE's open-source HCI solution built on KubeVirt and Longhorn. It's a compelling platform for running VMs on bare metal Kubernetes. [Cluster API](https://cluster-api.sigs.k8s.io/) (CAPI) is the standard for declarative Kubernetes cluster lifecycle management. In theory, combining them should enable fully automated provisioning of RKE2 clusters on Harvester.

The upstream [cluster-api-provider-harvester](https://github.com/rancher-sandbox/cluster-api-provider-harvester) by the Rancher team at SUSE provided an excellent foundation for this. The project's architecture -- two reconcilers, clean CRD design, IPPool integration, cloud-init generation -- was well thought out and gave me a solid base to build on. Credit to the original authors for laying this groundwork.

That said, the upstream project had been inactive for over a year, with no significant updates since v0.1.6. It hadn't been updated for recent Harvester versions and had several blockers that prevented production use:

- **IP allocation was broken** -- every VM got the same first IP from the pool
- **VM image names with underscores failed** -- a regex didn't account for `x86_64` in image names
- **Nil pointer crashes** -- Harvester moved from `spec.running` to `spec.runStrategy`, causing panics
- **Cloud-init keys were case-sensitive** -- KubeVirt silently ignores camelCase secret keys
- **Single-disk only** -- no multi-disk or boot order support
- **No cloud-provider bootstrap** -- a chicken-and-egg problem between Calico and the CCM left nodes uninitialized
- **No etcd cleanup** -- deleting a control plane node left orphaned etcd members
- **No DHCP support** -- VMs required static IPs in all cases

I forked the project, fixed these issues, and added a number of features with the goal of bringing it to production-grade quality. The result is [CAPHV v0.2.8](https://github.com/rancher-sandbox/cluster-api-provider-harvester), which can provision a 3 control-plane + N worker RKE2 cluster on Harvester in about 16 minutes, with zero manual intervention. The code has been formally reviewed by the SUSE CAPI engineering team and received a **Go recommendation** for adoption as the official Harvester infrastructure provider.

---

## Architecture Overview

The overall stack involves three layers: Rancher for centralized multi-cluster management, CAPI for declarative cluster lifecycle, and Harvester as the VM infrastructure.

```
Rancher Manager (Multi-cluster management UI)
  |
  +-- Rancher Turtles (CAPI integration)
  |     Auto-imports CAPI clusters into Rancher
  |     Deploys cattle-cluster-agent on workload clusters
  |     Provides single pane of glass for all clusters
  |
  +-- CAPI Core + RKE2 Bootstrap/ControlPlane providers
  |
  +-- CAPHV (Infrastructure Provider)
  |     |
  |     +-- HarvesterClusterReconciler
  |     |     Manages: Harvester connectivity, IPPool, cloud-provider config
  |     |     Creates: LoadBalancer service, ClusterResourceSets
  |     |
  |     +-- HarvesterMachineReconciler
  |           Manages: VM lifecycle, IP allocation, cloud-init, node init
  |           Creates: PVCs, KubeVirt VMs, cloud-init Secrets
  |           Cleanup: etcd members, PVCs, VM resources
  |
  v
Harvester HCI (KubeVirt + Longhorn)
  |
  +-- KubeVirt VMs (RKE2 nodes)
  +-- Longhorn PVCs (boot + data disks)
  +-- Harvester IPPool (IP allocation)
```

[Rancher](https://www.rancher.com/) sits at the top of the stack as the centralized management plane. It provides a unified UI and API for managing all clusters -- both the management cluster itself and the workload clusters provisioned by CAPI. Through [Rancher Turtles](https://turtles.docs.rancher.com/), CAPI-created clusters are automatically imported into Rancher, giving operators a single pane of glass with RBAC, monitoring, and fleet management across all clusters.

CAPHV itself follows the standard CAPI infrastructure provider pattern with two controllers. The provider uses Harvester's kubeconfig (stored as an identity secret) to create and manage VMs on the target Harvester cluster. Each VM gets its own cloud-init secret with network configuration, RKE2 bootstrap scripts, and TLS certificates.

---

## Key Features

### 1. Automatic IP Allocation from Harvester IPPool

CAPHV integrates with Harvester's native IPPool CRD to automatically allocate IPs for VMs. When a machine is created, it reserves the next available IP from the pool. When deleted, the IP is released back. Multi-pool support allows fallback allocation across multiple IPPools.

The upstream had a critical bug where `Store.Reserve()` didn't update the `Status.Allocated` map, causing every machine to get the same IP. This was fixed with proper allocation tracking.

```yaml
# HarvesterCluster spec
loadBalancerConfig:
  ipamType: pool
vmNetworkConfig:
  ipPoolRef: capi-vm-pool
  gateway: 172.16.0.1
  subnetMask: "255.255.0.0"
  dnsServers:
    - 172.16.0.1
```

### 2. DHCP Support

Not every environment uses static IPs. CAPHV supports DHCP by simply omitting the `vmNetworkConfig`:

```yaml
# HarvesterCluster spec -- DHCP mode
loadBalancerConfig:
  ipamType: dhcp
# No vmNetworkConfig = VMs get DHCP
```

This was harder than expected. KubeVirt's bridge binding creates a virtualized network environment where DHCP traffic doesn't behave the same as on a physical network. Standard DHCP clients that rely on `SOCK_DGRAM` sockets and BPF filters can miss DHCP responses in this context, because the packet framing differs from what the filters expect.

The solution: cloud-init injects an inline ISC `dhclient` script that uses `SOCK_RAW` / LPF (which works reliably in KubeVirt's virtualized network stack) and runs `dhclient -1` for each NIC at boot.

### 3. Multi-NIC Cloud-Init

VMs can have multiple network interfaces connected to different Harvester networks (NADs). CAPHV generates cloud-init configuration for all NICs:

```yaml
# HarvesterMachineTemplate spec
networks:
  - default/production    # eth0
  - default/management    # eth1
```

In static IP mode, eth0 gets the allocated IP and additional NICs get DHCP. In full DHCP mode, all NICs get DHCP. The cloud-init bootcmd generates a `dhclient-script` and launches `dhclient -1` per interface.

### 4. Multi-Disk VMs

Machines can have multiple volumes with configurable types and boot order:

```yaml
volumes:
  - volumeType: image
    imageName: default/sles15-sp7.qcow2
    volumeSize: 40Gi
    bootOrder: 1        # Boot disk (vda)
  - volumeType: storageClass
    storageClass: longhorn
    volumeSize: 100Gi
    bootOrder: 0        # Data disk (vdb)
```

Each volume creates a separate PVC. Image-backed volumes use the Longhorn storage class associated with the Harvester image. StorageClass volumes create blank data disks.

### 5. Cloud Provider Bootstrap (Solving the Chicken-and-Egg)

When a fresh RKE2 node boots, it has no CNI (Calico) running. Without CNI, pods can't reach each other. But the cloud-provider-harvester pod (CCM) needs to run to initialize the node (set providerID, remove taints). Without the CCM, Calico won't start because the node is tainted.

CAPHV solves this with a multi-layered approach:

1. **CCM pod runs with `hostNetwork: true`** -- bypasses the need for CNI
2. **Tolerates the uninitialized taint** -- schedules on tainted nodes
3. **Node initialization from management cluster** -- as a fallback, the controller directly patches the workload node's providerID and removes the uninitialized taint using the management cluster's kubeconfig

This is deployed automatically via a `ClusterResourceSet` that propagates ConfigMaps containing the CCM + CSI manifests.

### 6. Automatic etcd Member Cleanup

When a control plane machine is deleted (scale down, remediation, or upgrade), the corresponding etcd member must be removed. RKE2's control plane controller handles most cases, but as a safety net, CAPHV also removes stale members:

1. Find a healthy etcd pod on a remaining CP node
2. Run `etcdctl member list` via `kubectl exec`
3. Match the deleted node name and remove the member

This prevents split-brain scenarios and ensures clean cluster state after node replacements.

### 7. MachineHealthCheck & Auto-Remediation

CAPHV works with standard CAPI `MachineHealthCheck` resources. A typical configuration:

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineHealthCheck
spec:
  clusterName: my-cluster
  maxUnhealthy: 34%
  nodeStartupTimeout: 20m
  unhealthyConditions:
    - type: Ready
      status: "False"
      timeout: 5m
    - type: Ready
      status: Unknown
      timeout: 5m
```

When a node is NotReady for 5 minutes, the full remediation cycle runs: VM deletion, etcd cleanup, new machine creation, cloud-init bootstrap, node initialization. End-to-end recovery takes about 9 minutes with zero manual intervention.

### 8. Validating Webhooks

Admission webhooks are included in the provider manifest (`infrastructure-components.yaml`) and deployed automatically alongside the controller. They catch configuration errors at admission time:

- CPU must be > 0, memory must be a valid Kubernetes resource quantity
- SSH user and key pair must be specified
- At least one volume and one network required
- Cluster-level: target namespace, identity secret, and IPAM configuration validated

Requires cert-manager for TLS certificate management. Turtles users benefit from automatic [no-cert-manager conversion](https://turtles.docs.rancher.com/turtles/stable/en/overview/features.html#_no_cert_manager) to wrangler-based TLS.

### 9. Fleet Addon Management

CAPHV supports two modes for deploying CNI and CSI drivers to workload clusters:

- **CRS mode** (default) -- ClusterResourceSets with ConfigMaps, simple and self-contained
- **Fleet mode** -- leverages Fleet via CAAPF (Cluster API Addon Provider Fleet) for GitOps-driven addon deployment

In Fleet mode, a GitRepo resource points to a repository containing Helm-based addon bundles (Calico, Canal, Cilium, Harvester CSI). Cluster labels (`cni: calico`, `csi: harvester`) drive Fleet's cluster selector to deploy the right addons to each cluster. The CCM stays in CRS mode due to bootstrap timing requirements.

---

## ClusterClass: From 200 Lines to 30

CAPHV ships with a `ClusterClass` named `harvester-rke2` that encapsulates the full topology. Instead of writing ~200 lines of YAML for each cluster, you write ~30:

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: my-cluster
  namespace: capi-clusters
spec:
  topology:
    class: harvester-rke2
    version: v1.31.6+rke2r1
    controlPlane:
      replicas: 3
    workers:
      machineDeployments:
        - class: default-worker
          name: worker
          replicas: 2
    variables:
      - name: cpuCount
        value: 2
      - name: memorySize
        value: "4Gi"
      - name: vmNetworks
        value: '["default/production"]'
      - name: volumes
        value: '[{"volumeType":"image","imageName":"default/sles15-sp7.qcow2","volumeSize":"40Gi","bootOrder":1}]'
      - name: sshUser
        value: sles
      - name: sshKeyPair
        value: default/my-ssh-key
      - name: identitySecretName
        value: hv-identity-secret
      - name: targetNamespace
        value: default
```

The ClusterClass has 13 variables and 13 patches that handle everything: VM spec, networking, CNI selection (calico, canal, cilium, or none), cloud provider, CSI, and RBAC.

### CLI Generator

For even faster setup, the `caphv-generate` CLI creates all required resources:

```bash
# Interactive mode -- prompts for all parameters
caphv-generate --interactive

# Scripted mode -- flags for CI/CD
caphv-generate \
  --name prod-cluster \
  --namespace capi-prod \
  --cp-replicas 3 \
  --worker-replicas 2 \
  --image "default/sles15-sp7.qcow2" \
  --ssh-keypair "default/capi-ssh-key" \
  --network "default/production" \
  --gateway 172.16.0.1 \
  --subnet-mask 255.255.0.0 \
  --ip-pool capi-vm-pool \
  --dns 172.16.3.6 \
  --apply
```

This generates and applies: Namespace, identity Secret, Cluster topology, cloud-provider ConfigMaps, ClusterResourceSets, and MachineHealthCheck. A fully operational 3CP+2W cluster in ~16 minutes.

---

## Rancher Integration via Turtles

[Rancher Turtles](https://turtles.docs.rancher.com/) bridges CAPI and Rancher. Adding one label to the CAPI Cluster resource automatically imports it into Rancher:

```yaml
metadata:
  labels:
    cluster-api.cattle.io/rancher-auto-import: "true"
```

Turtles creates the provisioning cluster, deploys the cattle-cluster-agent on the workload cluster, and the cluster appears in Rancher UI as Connected + Ready. The `caphv-generate` CLI adds this label by default.

CAPHV is deployed as a CAPIProvider resource, which Turtles manages automatically -- including upgrades, CRD installation, and RBAC setup:

```yaml
apiVersion: turtles-capi.cattle.io/v1alpha1
kind: CAPIProvider
metadata:
  name: harvester
  namespace: caphv-system
spec:
  name: harvester
  type: infrastructure
  version: v0.2.8
  fetchConfig:
    url: https://github.com/rancher-sandbox/cluster-api-provider-harvester/releases/download/v0.2.8/infrastructure-components.yaml
```

**One gotcha**: if Rancher uses external TLS termination (e.g., Traefik), the `cacerts` setting is empty by default. Turtles in strict mode (`agent-tls-mode=true`) rejects the import. Fix: set the `cacerts` setting to the full certificate chain (e.g., Let's Encrypt intermediate + root CA).

---

## Lessons Learned: KubeVirt and Harvester Gotchas

Building a production CAPI provider for KubeVirt/Harvester surfaced several non-obvious issues:

| Issue | Root Cause | Solution |
|-------|-----------|----------|
| Cloud-init keys silently ignored | KubeVirt expects lowercase `userdata`/`networkdata` in secrets, not camelCase | Always use lowercase keys |
| Network config v2 fails on SLES | SLES 15 SP7 uses network-config v1 | Generate v1 format |
| DHCP doesn't work inside VMs | KubeVirt bridge binding alters packet framing, breaking SOCK_DGRAM-based DHCP clients | Use ISC dhclient with SOCK_RAW/LPF |
| `spec.running` causes nil pointer | Harvester v1.7+ uses `spec.runStrategy: Always` instead | Use KubeVirt's `vm.RunStrategy()` API |
| SubnetMask is a string | Not an integer -- "255.255.0.0" not 16 | Use string type in CRD |
| CPU topology creates CPU^3 vCPUs | Default: sockets=cpu, cores=cpu, threads=cpu | Set sockets=1, threads=1, cores=cpu |
| `memory.guest` not set after upgrade | KubeVirt 1.2+ requires explicit `domain.resources.requests.memory` | Set on all VM specs |
| PVCs orphaned after VM deletion | Controller deleted PVC before VM finished terminating | Delete VM first, then PVCs by name prefix |

---

## Testing & Quality

CAPHV has three levels of testing:

**Unit tests (195+ tests, CI):** Cover IP allocation, utility functions, cloud-init generation, etcd helpers, node initialization, and controller logic. Controller coverage ~70%, utility coverage ~80%. Run on every push via GitHub Actions on openSUSE Tumbleweed containers.

**Integration tests (18 tests, live cluster):** Four test suites that run against a real Harvester + Rancher + CAPI environment:

| Suite | Tests | Duration | What it validates |
|-------|-------|----------|-------------------|
| webhook | 5 | ~10s | Reject invalid resources, accept valid ones |
| scale | 4 | ~7min | Scale CP up/down, verify VM and PVC lifecycle |
| multidisk | 6 | ~7min | Multi-disk VMs, boot order, lsblk check via SSH |
| remediation | 3 | ~14min | VM deletion, MHC detection, cluster recovery |

**Turtles certification tests:** Full lifecycle test using the official Turtles test framework -- ClusterClass topology, VM provisioning, RKE2 bootstrap, Calico CNI, Rancher auto-import, cluster deletion. Passed in ~12 minutes.

**Observability:** 17 Prometheus metrics (`caphv_*` namespace) with a Grafana dashboard (20 panels). ServiceMonitor available for clusters running the Prometheus Operator.

The project underwent a formal code review by the SUSE CAPI engineering team, comparing the implementation against the [reference CAPD provider](https://github.com/kubernetes-sigs/cluster-api/tree/release-1.10/test/infrastructure/docker) and verifying compliance with the [InfraCluster](https://release-1-10.cluster-api.sigs.k8s.io/developer/providers/contracts/infra-cluster) and [InfraMachine](https://release-1-10.cluster-api.sigs.k8s.io/developer/providers/contracts/infra-machine) contracts. The review concluded with a **Go recommendation** for adoption.

---

## Getting Started

### Prerequisites

- A management Kubernetes cluster (RKE2 recommended)
- CAPI Core, RKE2 Bootstrap, and RKE2 ControlPlane providers installed (e.g., via Rancher Turtles)
- A Harvester HCI cluster (v1.7.x)
- A VM image uploaded to Harvester (e.g., SLES 15 SP7)
- An SSH key pair created on Harvester
- cert-manager (for webhook TLS -- or use Turtles' no-cert-manager feature)

### Install CAPHV

The recommended installation is via CAPIProvider (see the Rancher Integration section above). Alternatively, using `clusterctl`:

```bash
clusterctl init --infrastructure harvester
```

Release artifacts follow [clusterctl conventions](https://cluster-api.sigs.k8s.io/developer/providers/contracts/clusterctl#workload-cluster-templates): `infrastructure-components.yaml`, `metadata.yaml`, cluster templates (`cluster-template-*.yaml`), and ClusterClass definition (`clusterclass-harvester-rke2.yaml`) are all included in each [release](https://github.com/rancher-sandbox/cluster-api-provider-harvester/releases).

### Create a Harvester Identity Secret

```bash
# Export kubeconfig from Harvester and create the secret
kubectl create secret generic hv-identity-secret \
  -n capi-clusters \
  --from-file=kubeconfig=./harvester-kubeconfig.yaml
```

### Create a Cluster

Use the ClusterClass approach (see the example above) or the CLI generator. In about 16 minutes, you'll have a fully functional RKE2 cluster running on Harvester, with automatic health checks, etcd safety nets, and optional Rancher integration.

---

## What's Next

- **CAPI IPAM Provider** -- decouple the IP allocation logic into a standalone [IPAM Provider](https://cluster-api.sigs.k8s.io/developer/providers/contracts/ipam) for cleaner separation of concerns and a standard CAPI-compliant API
- **CAPI v1beta2 migration** -- adopt `status.initialization.provisioned` and new condition types when the v1beta2 contract stabilizes
- **Harvester v1.8.x compatibility** -- validate against upcoming API changes
- **CAAPF evolution** -- adapt Fleet addon integration as the CAAPF API evolves ([RFD 0051](https://github.com/SUSE/rancher-architecture/pull/51))

The project is open source: [github.com/rancher-sandbox/cluster-api-provider-harvester](https://github.com/rancher-sandbox/cluster-api-provider-harvester)

Feedback and contributions welcome.

---

*Julien Niedergang -- March 2026*

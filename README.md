# Cluster API Provider Harvester (CAPHV)

> Fork of [rancher-sandbox/cluster-api-provider-harvester](https://github.com/rancher-sandbox/cluster-api-provider-harvester) with Harvester v1.7.1 compatibility and production-ready features.

## Overview

CAPHV is a [Cluster API](https://cluster-api.sigs.k8s.io/) Infrastructure Provider for provisioning Kubernetes clusters on [Harvester HCI](https://harvesterhci.io/).

This fork adds significant enhancements over upstream v0.1.6:

| Feature | Upstream v0.1.x | This fork (v0.2.x) |
|---------|----------------|-------------------|
| Harvester compatibility | v1.2.0 | v1.7.1 |
| Multi-disk VMs | Single disk only | Multiple disks (image + storageClass) |
| IP allocation | Manual / DHCP | Automatic from Harvester IPPool or DHCP |
| Cloud-init | Basic | Network-config v1 (SLES), multi-NIC, static IP + DHCP |
| Cloud provider bootstrap | Manual fixes needed | Automatic (hostNetwork, RBAC, tolerations) |
| Node initialization | Manual providerID | Automatic from management cluster |
| etcd cleanup | Manual | Automatic on CP machine deletion |
| Validating webhooks | None | HarvesterMachine + HarvesterCluster |
| Boot order | Not supported | Configurable per-disk |
| VM runStrategy | Deprecated `spec.running` | `spec.runStrategy: Always` |
| MachineHealthCheck | Untested | Tested, full auto-remediation |
| Rolling K8s upgrade | Untested | Tested (CP + workers) |
| E2E tests | Kubebuilder scaffold only | 18 integration tests (live cluster) |
| ClusterClass | Generic example only | Production-ready with vmNetworkConfig, IPPool, sshUser |
| CLI generator | None | `caphv-generate` script (~30-line clusters) |
| Fleet/CAAPF addons | Not supported | CSI/CNI via Fleet GitOps with per-cluster CNI tuning |
| Helm chart | None | Full chart with webhook + ClusterClass support |

## Prerequisites

- Harvester HCI v1.7.x cluster
- Management cluster (RKE2 recommended) with:
  - Cluster API Core (v1.10+)
  - RKE2 Bootstrap + ControlPlane providers (v0.21+)
  - Rancher Turtles (optional, for automatic Rancher import)
  - cert-manager (required if webhooks enabled)
- Harvester identity Secret (kubeconfig for the target Harvester cluster)
- SSH KeyPair created on Harvester
- VM image uploaded to Harvester (SLES 15 SP7 recommended)
- IPPool configured on Harvester (for automatic IP allocation)

## Installation

### Option 1: CAPIProvider via Rancher Turtles (recommended for production)

```yaml
apiVersion: turtles-capi.cattle.io/v1alpha1
kind: CAPIProvider
metadata:
  name: harvester
  namespace: caphv-system
spec:
  name: harvester
  type: infrastructure
  version: v0.2.7
  fetchConfig:
    url: https://github.com/rancher-sandbox/cluster-api-provider-harvester/releases/download/v0.2.7/infrastructure-components.yaml
  configSecret:
    name: caphv-variables
```

See [docs/operations.md](docs/operations.md) for full CAPIProvider deployment, upgrade, and migration instructions.

### Option 2: Helm Chart (legacy)

> **Note**: The Helm chart is maintained for compatibility but is not the recommended
> installation method. Prefer CAPIProvider via Rancher Turtles (Option 1) or `clusterctl init`
> for production deployments. The chart may be removed in a future release.

```bash
# Without webhooks
helm install caphv chart/caphv/ \
  -n caphv-system --create-namespace \
  --set image.repository=ghcr.io/rancher-sandbox/cluster-api-provider-harvester \
  --set image.tag=v0.2.7

# With webhooks (requires cert-manager)
helm install caphv chart/caphv/ \
  -n caphv-system --create-namespace \
  --set image.repository=ghcr.io/rancher-sandbox/cluster-api-provider-harvester \
  --set image.tag=v0.2.7 \
  --set webhooks.enabled=true \
  --set webhooks.certManager.enabled=true
```

### Option 3: Kustomize

```bash
# Build and push the image
make docker-build docker-push IMG=ghcr.io/rancher-sandbox/cluster-api-provider-harvester:v0.2.7

# Deploy
make deploy IMG=ghcr.io/rancher-sandbox/cluster-api-provider-harvester:v0.2.7
```

### Option 4: Manual (standalone manifests)

```bash
kubectl apply -f out/infrastructure-components.yaml
```

## Quick Start (ClusterClass — recommended)

Using ClusterClass reduces cluster creation from ~200 lines to ~30 lines of YAML.

### 1. Install the ClusterClass (once per management cluster)

```bash
# Via Helm (with controller)
helm install caphv chart/caphv/ \
  -n caphv-system --create-namespace \
  --set clusterClass.enabled=true

# Or standalone
kubectl apply -f templates/clusterclass/rke2/clusterclass-harvester-rke2.yaml
```

### 2. Generate cluster manifests with the CLI

```bash
# Generate all manifests
bin/caphv-generate \
  --name my-cluster \
  --image "default/my-vm-image.qcow2" \
  --ssh-keypair "default/my-ssh-key" \
  --network "default/my-vm-network" \
  --gateway 10.0.0.1 \
  --subnet-mask 255.255.255.0 \
  --ip-pool my-ip-pool \
  --dns 10.0.0.53 \
  --harvester-kubeconfig ~/.kube/harvester.yaml \
  > cluster.yaml

# Or interactive mode
bin/caphv-generate --interactive

# Apply
kubectl apply -f cluster.yaml
# Or directly: bin/caphv-generate [...] --apply
```

The CLI generates: Namespace, Secret, Cluster (topology), ConfigMaps (CCM/CSI/Calico), ClusterResourceSets, and MachineHealthCheck.

### Fleet Mode (optional — GitOps addon management)

With CAAPF installed, addons can be managed via Fleet instead of CRS:

```bash
bin/caphv-generate \
  --name my-cluster \
  --cni calico --cni-mtu 1450 --cni-encapsulation VXLAN \
  --pod-cidr 10.244.0.0/16 \
  --fleet-addon-repo https://my-gitea/org/caphv-fleet-addons.git \
  --image "default/my-vm-image.qcow2" \
  --ssh-keypair "default/my-ssh-key" \
  --network "default/my-vm-network" \
  --gateway 10.0.0.1 --subnet-mask 255.255.255.0 \
  --ip-pool my-ip-pool \
  --harvester-kubeconfig ~/.kube/harvester.yaml \
  --apply
```

See [docs/fleet-addons.md](docs/fleet-addons.md) for full documentation.

### 3. Monitor cluster creation

```bash
kubectl get cluster,machine,harvestermachine -n my-cluster
```

## User Experience Summary

### Prerequisites (one-time setup)

On the management cluster:
- Rancher + Turtles installed
- CAPHV deployed via Helm (`clusterClass.enabled=true`)
- Rancher `cacerts` setting configured (required for Turtles strict TLS mode with external TLS termination)
- Harvester kubeconfig available locally

### Create a cluster

**Interactive mode (guided):**
```bash
caphv-generate --interactive
```
The script asks ~15 questions with sensible defaults, then generates and applies everything.

**Flags mode (scriptable):**
```bash
caphv-generate \
  --name my-cluster \
  --cp-replicas 3 --worker-replicas 2 \
  --image "default/my-vm-image.qcow2" \
  --ssh-keypair "default/my-ssh-key" \
  --network "default/my-vm-network" \
  --gateway 10.0.0.1 --subnet-mask 255.255.255.0 \
  --ip-pool my-ip-pool --dns 10.0.0.53 \
  --harvester-kubeconfig ~/.kube/harvester.yaml \
  --apply
```

### What happens automatically (~16 min)

1. **Namespace** created
2. **Secret** with Harvester kubeconfig injected
3. **ClusterClass** + templates deployed in the namespace
4. **Cluster topology** created — CAPI orchestrates everything:
   - VMs created on Harvester (IPs allocated from IPPool)
   - RKE2 bootstrap (control plane then workers)
   - Cloud-init with static IP, iptables, SSH
   - Cloud provider + CSI Harvester installed via ClusterResourceSets
   - MachineHealthCheck active (auto-remediation)
5. **Rancher** detects the cluster (auto-import label) — deploys agent — cluster visible in the UI

### Result

- Fully functional Kubernetes cluster (RKE2)
- Visible and manageable in Rancher UI
- Auto-remediation: if a VM dies, it is automatically recreated (~9 min)
- Rolling upgrade: change the K8s version in the Cluster spec — rolling update CP then workers

### Day 2 Operations

- **Scale**: modify `replicas` in the Cluster spec
- **Upgrade K8s**: modify `version` in the Cluster spec
- **Delete**: `kubectl delete cluster my-cluster -n my-namespace` — everything is cleaned up (VMs, PVCs, secrets)

## Quick Start (manual — full control)

### 1. Create the identity Secret

```bash
kubectl create secret generic hv-identity-secret \
  -n <namespace> \
  --from-file=kubeconfig=<path-to-harvester-kubeconfig>
```

### 2. Create a Cluster

```yaml
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: my-cluster
  namespace: my-namespace
spec:
  clusterNetwork:
    pods:
      cidrBlocks: [10.52.0.0/16]
    services:
      cidrBlocks: [10.53.0.0/16]
  controlPlaneRef:
    apiVersion: controlplane.cluster.x-k8s.io/v1beta1
    kind: RKE2ControlPlane
    name: my-cluster-cp
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
    kind: HarvesterCluster
    name: my-cluster-hv
---
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: HarvesterCluster
metadata:
  name: my-cluster-hv
  namespace: my-namespace
spec:
  targetNamespace: default
  identitySecret:
    name: hv-identity-secret
    namespace: my-namespace
  loadBalancerConfig:
    ipamType: pool
  vmNetworkConfig:
    gateway: "10.0.0.1"
    subnetMask: "255.255.255.0"
    ipPoolRef: default/my-ip-pool
```

### 3. Define Machine Templates

```yaml
apiVersion: infrastructure.cluster.x-k8s.io/v1alpha1
kind: HarvesterMachineTemplate
metadata:
  name: my-cluster-machine
  namespace: my-namespace
spec:
  template:
    spec:
      cpu: 2
      memory: "4Gi"
      sshUser: sles
      sshKeyPair: default/my-ssh-key
      volumes:
        - volumeType: image
          imageName: default/my-vm-image.qcow2
          volumeSize: "40Gi"
          bootOrder: 1
        - volumeType: storageClass     # optional: additional data disk
          storageClass: longhorn
          volumeSize: "10Gi"
      networks:
        - default/production
```

### 4. Create Control Plane + Workers

See [templates/](templates/) for complete RKE2 cluster template examples.

## Architecture

```
Management Cluster (RKE2)
├── CAPI Core Controller
├── RKE2 Bootstrap Controller
├── RKE2 ControlPlane Controller
├── CAPHV Controller  ◄── this project
│   ├── HarvesterCluster reconciler
│   ├── HarvesterMachine reconciler
│   │   ├── IP allocation from IPPool
│   │   ├── VM creation (multi-disk, cloud-init, static IP)
│   │   ├── Cloud provider bootstrap (hostNetwork fix)
│   │   ├── Node init (providerID + taint removal)
│   │   └── etcd cleanup on CP deletion
│   └── Validating webhooks (optional)
└── Rancher Turtles (optional, auto-import)

Harvester HCI (target)
├── VMs (created by CAPHV)
├── IPPool (IP allocation)
├── VM Images (boot disks)
└── Longhorn (storage)
```

## Configuration Reference

### HarvesterCluster

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spec.targetNamespace` | string | Yes | Namespace on Harvester for VMs |
| `spec.identitySecret.name` | string | Yes | Secret containing Harvester kubeconfig |
| `spec.identitySecret.namespace` | string | Yes | Namespace of identity secret |
| `spec.loadBalancerConfig.ipamType` | string | Yes | `pool` or `dhcp` |
| `spec.vmNetworkConfig.gateway` | string | Yes* | Gateway IP (*required for pool IPAM) |
| `spec.vmNetworkConfig.subnetMask` | string | Yes* | Subnet mask (e.g. "255.255.0.0") |
| `spec.vmNetworkConfig.ipPoolRef` | string | No | Reference to Harvester IPPool |

> **DHCP mode**: If `vmNetworkConfig` is omitted and no machine-level `networkConfig` is set, all VM NICs will use DHCP automatically. No IPPool or static IP configuration is needed.

### HarvesterMachine

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `spec.cpu` | int | Yes | Number of CPU cores (must be > 0) |
| `spec.memory` | string | Yes | Memory (e.g. "4Gi") |
| `spec.sshUser` | string | Yes | SSH user for cloud-init |
| `spec.sshKeyPair` | string | Yes | Harvester SSH KeyPair reference |
| `spec.volumes` | []Volume | Yes | At least one volume required |
| `spec.networks` | []string | Yes | At least one network required |
| `spec.volumes[].volumeType` | string | Yes | `image` or `storageClass` |
| `spec.volumes[].imageName` | string | For image | Harvester VM image (namespace/name) |
| `spec.volumes[].storageClass` | string | For SC | Storage class for blank disk |
| `spec.volumes[].volumeSize` | string | Yes | Disk size (e.g. "40Gi") |
| `spec.volumes[].bootOrder` | int | No | Boot priority (1 = first) |

## Monitoring

CAPHV exposes custom Prometheus metrics (`caphv_*` namespace) via the controller-runtime metrics endpoint (port 8080, protected by kube-rbac-proxy).

A ServiceMonitor is included in the kustomize build. A ready-to-import Grafana dashboard is at `config/grafana/caphv-dashboard.json`.

Key metrics: `caphv_machine_create_total`, `caphv_machine_creation_duration_seconds`, `caphv_machine_status`, `caphv_ippool_allocations_total`, `caphv_cluster_ready`, `caphv_etcd_member_remove_total`, `caphv_node_init_duration_seconds`.

See [docs/operations.md](docs/operations.md) for the full metrics list and alerting recommendations.

## Documentation

- [Operations Guide](docs/operations.md) — installation via CAPIProvider, cluster lifecycle, monitoring, backup/DR
- [Fleet Addons Guide](docs/fleet-addons.md) — Fleet/CAAPF addon management for CSI and CNI
- [Troubleshooting](docs/troubleshooting.md) — IPPool, cloud-init, DHCP, Turtles/Rancher, VM creation, etcd

## E2E Tests

Integration tests run against a live Harvester + CAPI cluster:

```bash
./test/e2e/run-e2e.sh              # Run all (18 tests, ~30min)
./test/e2e/run-e2e.sh webhook      # Validation tests (~10s)
./test/e2e/run-e2e.sh scale        # Scale up/down (~7min)
./test/e2e/run-e2e.sh multidisk    # Multi-disk VM (~7min)
./test/e2e/run-e2e.sh remediation  # MHC auto-remediation (~14min)
```

## Building

```bash
# Build binary
make build

# Build container image
make docker-build IMG=ghcr.io/rancher-sandbox/cluster-api-provider-harvester:v0.2.7

# Run unit tests
make test
```

## Release History

| Version | Date | Key changes |
|---------|------|-------------|
| v0.2.7 | 2026-03-10 | Code quality fixes for SURE-11421 review: kustomize modernization, finalizer naming conventions, context propagation, error handling |
| v0.2.6 | 2026-03-09 | CSI decoupling, Fleet label automation, Fleet CSI bundle |
| v0.2.5 | 2026-03-08 | Fleet/CAAPF addon management, CNI configuration flags |
| v0.2.4 | 2026-03-08 | CAPIProvider in Turtles, P0 milestone complete |
| v0.2.3 | 2026-03-07 | DHCP VM support, multi-NIC cloud-init |
| v0.2.1 | 2026-03-06 | ClusterClass (harvester-rke2), CLI generator (caphv-generate), Helm ClusterClass option |
| v0.2.0 | 2026-03-06 | Harvester v1.7.1, multi-disk, IPPool, webhooks, auto-remediation, e2e tests |
| v0.1.6 | 2024-xx-xx | Upstream: initial CAPI contract, single disk, DHCP only |

## License

Apache License 2.0 - See [LICENSE](LICENSE) for details.

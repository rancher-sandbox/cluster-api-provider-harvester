# Fleet/CAAPF Addon Management

CAPHV supports two addon management modes for deploying CSI and CNI configuration to workload clusters:

1. **CRS mode** (default) — ClusterResourceSets with hardcoded ConfigMaps
2. **Fleet mode** — Fleet GitRepo via CAAPF (Cluster API Addon Provider Fleet)

## Architecture

```
                        Management Cluster
                    ┌─────────────────────────┐
                    │  CAPHV Controller        │
                    │  Turtles + CAAPF         │
                    │  Fleet Controller        │
                    └───────┬─────────────────┘
                            │
              ┌─────────────┼──────────────┐
              │ CRS Mode    │ Fleet Mode   │
              ▼             ▼              ▼
         ConfigMap     GitRepo ──► caphv-fleet-addons repo
         + CRS              │         ├── harvester-csi/
                            │         ├── calico-config/
                            │         ├── canal-config/
                            │         └── cilium-config/
                            ▼
                    Workload Cluster
                    ├── CCM (always CRS)
                    ├── CSI (CRS or Fleet)
                    └── CNI config (CRS or Fleet)
```

### Why CCM stays in CRS

The CAPHV controller (`reconcileCloudProviderConfig()`) injects the Harvester kubeconfig into the `cloud-config` Secret via the CSI ConfigMap CRS mechanism. The CCM depends on this Secret being present at bootstrap time, before Fleet agent can even connect. Decoupling CCM from CRS would require significant controller changes.

### Why not `cni: none` + Fleet

RKE2 deploys the CNI as a system chart during node bootstrap. Fleet needs a running agent on the workload cluster to deploy bundles — which requires pod networking — which requires a CNI. Setting `cni: none` and deploying CNI via Fleet creates a deadlock. Instead, RKE2 installs the CNI with defaults and Fleet deploys a `HelmChartConfig` to configure it.

## Prerequisites

### CAAPF Installation

Deploy the CAPIProvider for CAAPF on the management cluster:

```bash
kubectl apply -f manifests/caapf-provider.yaml
```

This creates a `CAPIProvider` resource that Turtles installs in `caapf-system`. Verify:

```bash
kubectl get capiprovider -A
# fleet   addon   fleet   v0.12.0   Ready
```

**Version compatibility**: CAAPF v0.12.0 works with CAPI v1.10.x (v1beta1). CAAPF v0.13+ requires CAPI v1.11+ (v1beta2).

### Fleet Addons Repository

The Fleet addon bundles are in a separate Git repository. For the homelab setup:
- Repository: `https://gitea.home.zypp.fr/jniedergang/caphv-fleet-addons.git`

## Usage

### Fleet Mode

```bash
caphv-generate \
  --name my-cluster \
  --cni calico \
  --cni-mtu 1450 \
  --cni-encapsulation VXLAN \
  --pod-cidr 10.244.0.0/16 \
  --fleet-addon-repo https://gitea.home.zypp.fr/jniedergang/caphv-fleet-addons.git \
  --fleet-addon-branch main \
  --image "default/sles15-sp7-minimal-vm.x86_64-cloud-qu2.qcow2" \
  --ssh-keypair "default/capi-ssh-key" \
  --network "default/production" \
  --gateway 172.16.0.1 \
  --subnet-mask 255.255.0.0 \
  --ip-pool capi-vm-pool \
  --apply
```

This generates:
- Cluster with `csi: harvester` and `cni: calico` labels (for Fleet targeting)
- CNI annotations (`caphv.io/cni-mtu`, etc.) for Fleet templating
- `spec.clusterNetwork.pods.cidrBlocks` for pod CIDR
- CCM ConfigMap + CRS (always)
- Fleet `GitRepo` targeting the addons repository
- MachineHealthCheck

### CRS Mode (default)

```bash
caphv-generate \
  --name my-cluster \
  --image "default/sles15-sp7-minimal-vm.x86_64-cloud-qu2.qcow2" \
  --ssh-keypair "default/capi-ssh-key" \
  --network "default/production" \
  --gateway 172.16.0.1 \
  --subnet-mask 255.255.0.0 \
  --ip-pool capi-vm-pool \
  --apply
```

Same as before — generates CCM + CSI + CNI ConfigMaps and 3 ClusterResourceSets.

## CNI Configuration Parameters

| Flag | Default | Description |
|------|---------|-------------|
| `--pod-cidr` | `10.42.0.0/16` | Pod CIDR |
| `--cni-mtu` | `1500` | MTU |
| `--cni-encapsulation` | `VXLANCrossSubnet` | Encapsulation mode |
| `--cni-bgp` | `Disabled` | BGP mode |

In Fleet mode, these are stored as cluster annotations and read by Fleet's templating engine when deploying the `HelmChartConfig`.

## How Fleet Targeting Works

1. CAAPF auto-creates a Fleet Cluster for each CAPI cluster
2. The `GitRepo` uses `clusterSelector` matching `cluster.x-k8s.io/cluster-name`
3. Each Fleet bundle has `targetCustomizations` matching labels (`csi: harvester`, `cni: calico`)
4. Fleet deploys matching bundles to the workload cluster

## Migration from CRS to Fleet

For existing clusters on CRS mode, migration is not automatic. To migrate:

1. Install CAAPF (`manifests/caapf-provider.yaml`)
2. Delete the CSI and CNI CRS resources from the cluster namespace
3. Add Fleet labels (`csi: harvester`, `cni: <plugin>`) and annotations to the Cluster
4. Create a `GitRepo` targeting the cluster

The CCM CRS remains unchanged in both modes.

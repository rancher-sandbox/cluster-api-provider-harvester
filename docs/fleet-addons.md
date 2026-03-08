# Fleet/CAAPF Addon Management

CAPHV supports two addon management modes for deploying CNI configuration to workload clusters:

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
         + CRS              │         ├── calico-config/
                            │         ├── canal-config/
                            │         └── cilium-config/
                            ▼
                    Workload Cluster
                    ├── CCM (always CRS)
                    ├── CSI (always CRS — controller-coupled)
                    └── CNI config (CRS or Fleet)
```

### Why CCM and CSI stay in CRS

The CAPHV controller (`reconcileCloudProviderConfig()`) injects the Harvester kubeconfig into the `cloud-config` Secret via the CSI ConfigMap CRS mechanism. Both the CCM and CSI depend on this Secret being present at bootstrap time, before Fleet agent can even connect. The controller is coupled to the CSI ConfigMap for kubeconfig injection — decoupling requires significant controller changes.

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

The Fleet addon bundles are in a separate Git repository containing HelmChartConfig manifests for each supported CNI plugin.

## Usage

### Fleet Mode

```bash
caphv-generate \
  --name my-cluster \
  --cni calico \
  --fleet-addon-repo https://my-gitea/org/caphv-fleet-addons.git \
  --image "default/my-vm-image.qcow2" \
  --ssh-keypair "default/my-ssh-key" \
  --network "default/my-network" \
  --gateway 10.0.0.1 \
  --subnet-mask 255.255.255.0 \
  --ip-pool my-ip-pool \
  --apply
```

This generates:
- Cluster with `cni: calico` label and CNI annotations
- `spec.clusterNetwork.pods.cidrBlocks` for pod CIDR
- CCM ConfigMap + CRS (always)
- CSI ConfigMap + CRS (always — controller-coupled)
- Fleet `GitRepo` targeting the CNI config bundle
- MachineHealthCheck

### CRS Mode (default)

```bash
caphv-generate \
  --name my-cluster \
  --image "default/my-vm-image.qcow2" \
  --ssh-keypair "default/my-ssh-key" \
  --network "default/my-network" \
  --gateway 10.0.0.1 \
  --subnet-mask 255.255.255.0 \
  --ip-pool my-ip-pool \
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

These are stored as cluster annotations (`caphv.io/cni-*`) for reference and future template support.

## How Fleet Targeting Works

1. Turtles auto-imports the CAPI cluster into Rancher
2. Rancher creates a management cluster, which needs `fleetWorkspaceName: fleet-default` for Fleet integration
3. A Fleet Cluster is created in `fleet-default` namespace
4. Labels must be added to the Fleet Cluster for bundle targeting (e.g., `cni: calico`)
5. The `GitRepo` bundle deploys matching CNI config to the workload cluster

**Important**: CAAPF v0.12.0 does not automatically propagate CAPI cluster labels to Fleet clusters. After cluster creation, you may need to:
- Set `fleetWorkspaceName: fleet-default` on the management cluster
- Add labels to the Fleet cluster for bundle matching

## Fleet Addon Repository Structure

```
caphv-fleet-addons/
  fleet/
    calico-config/
      fleet.yaml                    # Bundle config (clusterSelector: cni=calico)
      manifests/
        helmchartconfig.yaml        # HelmChartConfig for rke2-calico
    canal-config/
      fleet.yaml
      manifests/
        helmchartconfig.yaml
    cilium-config/
      fleet.yaml
      manifests/
        helmchartconfig.yaml
```

Each bundle deploys a `HelmChartConfig` that configures the RKE2 system chart for the CNI plugin. The HelmChartConfig uses static default values. To customize values per environment, fork the addons repository and modify the HelmChartConfig manifests.

## Migration from CRS to Fleet

For existing clusters on CRS mode, migration is not automatic. To migrate:

1. Install CAAPF (`manifests/caapf-provider.yaml`)
2. Delete the CNI CRS resource from the cluster namespace
3. Add Fleet labels (`cni: <plugin>`) to the Fleet Cluster
4. Create a `GitRepo` targeting the cluster

The CCM and CSI CRS remain unchanged in both modes.

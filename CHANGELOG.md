# Changelog

All notable changes to this project are documented in this file.

This fork diverges from [upstream](https://github.com/rancher-sandbox/cluster-api-provider-harvester) v0.1.6 with Harvester v1.7.1 compatibility and production-ready features.

## [v0.2.5] - 2026-03-08

### Added

- **Fleet/CAAPF addon management**: CNI configuration can now be deployed via Fleet GitOps instead of ClusterResourceSets — HelmChartConfig for Calico/Canal/Cilium tuning
- **CAAPF integration**: CAPIProvider manifest for Cluster API Addon Provider Fleet (v0.12.0, CAPI v1beta1 compatible)
- **CNI configuration flags**: `--pod-cidr`, `--cni-mtu`, `--cni-encapsulation`, `--cni-bgp` in caphv-generate
- **Fleet addon repository**: Separate Git repository (`caphv-fleet-addons`) with Fleet bundles for Calico, Canal, and Cilium HelmChartConfig
- **Fleet mode in caphv-generate**: `--fleet-addon-repo` generates a Fleet `GitRepo` with cluster-scoped targeting instead of CNI CRS
- **E2E fleet test suite**: 3 tests (CAAPF controller, Fleet mode generation, CRS retrocompatibility)

### Changed

- Cluster manifests now include `spec.clusterNetwork.pods.cidrBlocks` for pod CIDR configuration
- CCM and CSI remain in CRS mode regardless of addon mode (controller-coupled bootstrap dependency)

## [v0.2.3] - 2026-03-07

### Added

- **DHCP VM support**: VMs can now use DHCP instead of requiring static IP allocation from an IPPool — simply omit `vmNetworkConfig` and `networkConfig` to get DHCP on all NICs
- **Multi-NIC cloud-init**: Static mode network-config v1 generates entries for all NICs (eth0 static, eth1+ DHCP)

### Fixed

- **Wicked DHCP failure on KubeVirt**: Wicked's BPF filter uses link-layer offsets on `AF_PACKET SOCK_DGRAM`, but the kernel presents network-layer data to BPF, causing all DHCP responses to be silently dropped. Workaround: use ISC dhclient (`SOCK_RAW`/LPF) via cloud-init bootcmd
- **Cloud-init ordering**: dhclient-script created inline in `bootcmd` (not `write_files` which runs after `bootcmd`)
- **dhclient blocking**: Use `-1` flag (try once, fork to background) instead of `-d` (foreground) which blocked cloud-init

### Changed

- Split `buildNetworkData()` into `buildNetworkDataStatic()` (for static IP mode) and `buildDHCPCloudInit()` (for DHCP mode)
- Skip `networkdata` generation entirely in DHCP mode to prevent wicked nanny from overriding dhclient-assigned IPs

## [v0.2.1] - 2026-03-06

### Added

- **ClusterClass support**: Production-ready `harvester-rke2` ClusterClass with 12 variables and 12 patches, reducing cluster definitions from ~200 lines to ~30 lines of YAML
- **CLI generator** (`caphv-generate`): Interactive and flag-based tool to generate Cluster manifests from ClusterClass — outputs Namespace, Secret, Cluster topology, ConfigMaps, ClusterResourceSets, and MachineHealthCheck
- **Rancher auto-import via Turtles**: Label `cluster-api.cattle.io/rancher-auto-import=true` on generated Cluster manifests triggers automatic import into Rancher Manager through the Turtles operator
- **Helm ClusterClass integration**: `clusterClass.enabled=true` deploys the ClusterClass alongside the controller

### Fixed

- ClusterClass namespace scoping: ClusterClass must be in the same namespace as the Cluster (CAPI requirement)
- `controlPlaneEndpoint.host` must not be empty: use `"0.0.0.0"` as placeholder for dynamic IP allocation
- ClusterRole `configmaps` permission at cluster scope (controller cache requirement)

## [v0.2.0] - 2026-03-06

### Added

- **Harvester v1.7.1 compatibility**: Updated API clients, tested against Harvester v1.7.1 + Rancher v2.13.1
- **Multi-disk VM support**: Multiple volumes per VM (image-backed boot disk + storageClass data disks) with configurable boot order — fixes upstream limitation of single-disk only
- **IPPool-based IP allocation**: Automatic static IP assignment from Harvester IPPool resources, replacing manual or DHCP-only approaches
- **Network-config v1**: Cloud-init network configuration compatible with SLES/openSUSE (v2 not supported)
- **Cloud provider bootstrap**: Automatic `hostNetwork`, `dnsPolicy`, RBAC, and `not-ready` tolerations for the Harvester cloud controller manager — solves the Calico/cloud-provider chicken-and-egg problem
- **Node initialization from management cluster**: Automatic `providerID` and taint removal via management cluster, eliminating manual intervention
- **Automatic etcd member cleanup**: Removes stale etcd members when control-plane machines are deleted
- **Validating webhooks**: Admission webhooks for `HarvesterMachine` and `HarvesterCluster` resources — validates CPU, memory, sshUser, volumes, networks, targetNamespace, identitySecret, ipamType, vmNetworkConfig
- **E2E integration tests**: 18 tests across 4 suites (webhook, scale, multi-disk, remediation) running against a live cluster — addresses upstream issue [#91](https://github.com/rancher-sandbox/cluster-api-provider-harvester/issues/91)
- **Helm chart**: Full Helm chart with support for webhooks, cert-manager integration, ClusterClass, and configurable resources
- **MachineHealthCheck**: Tested full auto-remediation cycle (VM deletion → detection → etcd cleanup → replacement → Ready in ~9 min)
- **Rolling K8s upgrade**: Tested control plane + worker rolling upgrade (v1.31.6 → v1.31.14, ~35 min)

### Fixed

- **`memory.guest` on VM domain spec**: Set `memory.guest` for KubeVirt compatibility — fixes upstream issue [#139](https://github.com/rancher-sandbox/cluster-api-provider-harvester/issues/139)
- **`Store.Reserve()` not updating `Status.Allocated`**: Every machine got the same first IP from the pool
- **`CheckNamespacedName` regex missing underscore**: Image names containing `x86_64` failed parsing
- **CPU topology**: Set `sockets=1 threads=1` to prevent vCPU count multiplication
- **`spec.running` nil pointer dereference**: Harvester uses `runStrategy: Always` (not deprecated `running: true`); switched to `vm.RunStrategy()` method
- **Orphaned PVCs**: Delete VM first, then clean up PVCs by prefix
- **Cloud-init secret keys**: Must be lowercase (`userdata`/`networkdata`, not camelCase)

### Changed

- VM creation uses `spec.runStrategy: Always` instead of deprecated `spec.running: true`
- Webhooks use `admission.CustomValidator` interface (not `webhook.Validator`, removed in controller-runtime v0.21.0)

## [v0.1.6] - Upstream

See [upstream releases](https://github.com/rancher-sandbox/cluster-api-provider-harvester/releases) for v0.1.x changelog.

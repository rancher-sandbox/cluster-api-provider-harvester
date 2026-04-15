# Changelog

All notable changes to this project are documented in this file.

This fork diverges from [upstream](https://github.com/rancher-sandbox/cluster-api-provider-harvester) v0.1.6 with Harvester v1.7.1 compatibility and production-ready features.

## [v0.2.9] - 2026-04-15

### Security

- **Supply chain hardening (rancher/rancher-security#1667)**:
  - All GitHub Actions in `lint.yml`, `test.yml`, and `release.yml` pinned to full 40-char commit SHAs (no more tag references)
  - Replaced `curl HEAD/install.sh | bash` golangci-lint installer with the official `golangci/golangci-lint-action` (SHA-pinned to v9.2.0)
  - Added cosign keyless signing of multi-arch container images (OIDC via GitHub Actions identity, no secrets)
  - Added SLSA build provenance attestation (`actions/attest-build-provenance`) pushed to the registry
  - Enabled SBOM and provenance generation in `docker/build-push-action` (`sbom: true`, `provenance: mode=max`)
  - Added workflow-level and job-level least-privilege permissions across all CI workflows
  - Added `persist-credentials: false` to all `actions/checkout` invocations (defense-in-depth)
- **Build hygiene**: Restored missing `scripts/ci-lint-dockerfiles.sh` hadolint wrapper that the Makefile referenced but did not exist (pattern shared with cluster-api-provider-rke2 and rancher-turtles)

### Fixed

- **CI Lint workflow**: Upgraded `golangci-lint` from v2.2.1 to v2.11.1 to fix runtime panic on transitive dependencies declaring `go 1.26`
- **Workflow runtime**: Upgraded `azure/setup-helm` from v4.3.1 to v5.0.0 (Node.js 24, addresses GitHub deprecation warnings for Node 20 EOL 2026-09-16)

### Changed

- **Dependencies**: 8 Dependabot bumps merged (docker actions v3→v4/v5/v6/v7, ginkgo v2.28.1, gomega v1.39.1, prometheus/client_golang v1.23.2)
- **`.golangci.yml`**: Aligned `run.go` from `1.23` to `1.24` to match `go.mod`

### Verifying release artifacts

The container image and helm chart are signed with cosign using GitHub OIDC. Verify with:

```bash
cosign verify ghcr.io/rancher-sandbox/cluster-api-provider-harvester:v0.2.9 \
  --certificate-identity-regexp "^https://github.com/rancher-sandbox/cluster-api-provider-harvester" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
```

A SLSA provenance attestation is attached to the registry alongside the image.

## [v0.2.8] - 2026-03-16

### Fixed

- **CAPI contract compliance**: Added `ObjectMeta` to `HarvesterMachineTemplateResource` per InfraMachineTemplate contract; removed unused `HarvesterClusterTemplate.Status` field
- **Webhook clarity**: Removed standalone `examples/webhook-deployment.yaml` — webhooks are included in `infrastructure-components.yaml` and deployed automatically
- **Manifest verification**: Added `verify-manifests` target to `make verify` to catch out-of-sync generated manifests
- **Release artifacts**: Cluster templates (`cluster-template-*.yaml`) and ClusterClass (`clusterclass-harvester-rke2.yaml`) now included in release assets per clusterctl conventions
- **v1beta2 readiness**: Documented `status.initialization.provisioned` as forward-looking for CAPI v1beta2; `status.ready` remains the authoritative field for v1beta1
- **Helm chart deprecated**: Marked as legacy in favor of CAPIProvider/clusterctl installation
- **CAAPF note**: Added RFD 0051 advisory in Fleet addons documentation
- **Cleanup**: Removed unused `listImagesSelector` constant

## [v0.2.7] - 2026-03-10

### Fixed

- **Kustomize modernization**: Migrated config from deprecated `bases`, `patchesStrategicMerge`, `vars`, and `commonLabels` to `resources`, `patches`, `replacements`, and `labels` — eliminates 5 deprecation warnings from kustomize builds
- **Finalizer naming conventions**: Renamed finalizers to include `/finalizer` path segment per Kubernetes conventions (`harvestermachine.infrastructure.cluster.x-k8s.io/finalizer`, `harvestercluster.infrastructure.cluster.x-k8s.io/finalizer`); includes dual-finalizer migration for existing clusters
- **Context propagation**: Replaced all `context.TODO()` / `context.Background()` in production code with proper context propagation from reconciler scope
- **IP pool error handling**: Added error handling for `netip.AddrFromSlice()` calls in ippool.go
- **Typo fix**: Corrected "invalide" to "invalid" in ippool.go error message
- **Privileged container documentation**: Added comment explaining privileged container requirement in manager deployment
- **Regenerated infrastructure-components.yaml** without deprecation warnings

## [v0.2.6] - 2026-03-09

### Added

- **CSI decoupling**: Separated cloud-config Secret into its own ConfigMap (`cloud-config-addon-<name>`) and ClusterResourceSet, decoupling CSI driver from the controller — CSI can now be deployed via Fleet independently
- **Fleet label automation**: New `reconcileFleetIntegration()` automatically sets `fleetWorkspaceName: fleet-default` on Rancher management cluster and propagates CAPI labels (`cni`, `csi`, `cloud-config`, `cluster-name`) to Fleet cluster after Turtles import
- **Fleet CSI bundle**: CSI driver manifests in the Fleet addons repo (`/fleet/harvester-csi/`), deployed via Fleet GitRepo in Fleet mode
- **FleetIntegrationReady condition**: Tracks the status of Fleet label propagation (skipped for non-auto-import clusters)

### Changed

- `updateCloudProviderConfig` now points to `cloud-config-addon-<name>` (key: `cloud-config.yaml`) instead of `harvester-csi-driver-addon-<name>` (key: `harvester-cloud-provider-deploy.yaml`)
- CSI ConfigMap + CRS only generated in CRS mode (not Fleet mode); in Fleet mode, CSI is deployed via Fleet GitRepo
- Fleet mode GitRepo now includes `/fleet/harvester-csi` path in addition to CNI config
- Cluster labels: added `cloud-config: harvester` for CRS matching; Fleet mode uses `csi: harvester` (matches Fleet bundle selector)

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

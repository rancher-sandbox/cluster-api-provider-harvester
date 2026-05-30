# Migrating CAPHV from v0.2.x to v0.3.0

CAPHV v0.3.0 targets the **Cluster API v1.12 / v1beta2** ecosystem. It is
**not backward-compatible** with managers running CAPI v1.10 — installing
v0.3.0 on top of an unprepared manager will fail because the controller
looks up Cluster objects via `cluster.x-k8s.io/v1beta2`, which only
exists from CAPI core v1.11+.

This guide describes the upgrade in order.

## 1. Target ecosystem

| Component | Minimum | Tested against |
|-----------|---------|----------------|
| Kubernetes (management cluster) | 1.34 | 1.34.7 |
| Cluster API core | v1.12.x | v1.12.7 |
| cluster-api-provider-rke2 (bootstrap + control plane) | v0.24.x | v0.25.0 |
| Rancher Turtles | v0.26.x | v0.26.2 |
| Rancher Manager (optional, ships Turtles by default) | 2.14.x | 2.14.1 |
| cert-manager | v1.17+ | v1.18.2 |
| Harvester HCI | v1.7.x / v1.8.x | v1.8.0 |

Go 1.24+ is required when building CAPHV from source. CAPHV v0.3.0 declares
`go 1.25` in its `go.mod` and ships an image built with
`registry.suse.com/bci/golang:1.25`.

## 2. Pre-flight: backup

- Export your `HarvesterCluster`, `HarvesterMachine`, `HarvesterMachineTemplate`,
  `Cluster`, and related `*Template` resources:
  ```bash
  kubectl get -A harvestercluster,harvestermachine,harvestermachinetemplate,cluster,rke2controlplane,rke2configtemplate -o yaml > caphv-backup.yaml
  ```
- Snapshot the management cluster etcd if possible.

## 3. Upgrade the CAPI ecosystem first

CAPHV v0.3.0 will not reconcile against CAPI v1.10. Bump the dependent
providers in this order:

1. **CAPI core** v1.10.x → v1.12.x
2. **cluster-api-provider-rke2** v0.21.x → v0.24.x (bootstrap and control plane)
3. **Rancher Turtles** v0.18.x / v0.20.x → v0.26.x
4. (optional) **CAAPF** v0.12 → v0.13+

If you are running Rancher Turtles, the simplest path is to update the
Turtles Helm chart to v0.26.x and let it reconcile the bumped versions
via its `CAPIProvider` resources.

```bash
helm repo add turtles https://rancher.github.io/turtles
helm repo update
helm upgrade rancher-turtles turtles/rancher-turtles \
    --namespace cattle-turtles-system --version 0.26.2 --wait
```

After Turtles has converged, ensure the RKE2 providers exist explicitly
(Turtles does not enable them by default):

```yaml
apiVersion: turtles-capi.cattle.io/v1alpha1
kind: CAPIProvider
metadata:
  name: rke2-bootstrap
  namespace: rke2-bootstrap-system
spec:
  name: rke2
  type: bootstrap
---
apiVersion: turtles-capi.cattle.io/v1alpha1
kind: CAPIProvider
metadata:
  name: rke2-control-plane
  namespace: rke2-control-plane-system
spec:
  name: rke2
  type: controlPlane
```

Verify before continuing:

```bash
kubectl get capiproviders -A
# core cluster-api      Ready  v1.12.x
# bootstrap rke2        Ready  v0.24+ (validated v0.25.0)
# controlPlane rke2     Ready  v0.24+
kubectl api-resources --api-group=cluster.x-k8s.io | grep v1beta2
# clusters cl  cluster.x-k8s.io/v1beta2
```

## 4. Upgrade CAPHV

### Via Turtles `CAPIProvider`

Patch the spec of your existing `CAPIProvider/harvester` to bump the
version. Turtles will roll the controller deployment automatically.

```yaml
apiVersion: turtles-capi.cattle.io/v1alpha1
kind: CAPIProvider
metadata:
  name: harvester
  namespace: caphv-system
spec:
  name: harvester
  type: infrastructure
  version: v0.3.0
  fetchConfig:
    url: https://github.com/rancher-sandbox/cluster-api-provider-harvester/releases/download/v0.3.0/infrastructure-components.yaml
```

### Via `kubectl apply`

```bash
kubectl apply -f https://github.com/rancher-sandbox/cluster-api-provider-harvester/releases/download/v0.3.0/infrastructure-components.yaml
```

The release artifact updates the CRDs, RBAC, deployment, webhooks, and
cert-manager `Certificate`. Existing `HarvesterCluster` / `HarvesterMachine`
resources are preserved — only the `status.conditions` schema changes (see
next section).

## 5. Status changes you may notice

- `status.conditions` on `HarvesterCluster` and `HarvesterMachine` are
  now `[]metav1.Condition` (k8s-standard fields: `lastTransitionTime`,
  `message`, `reason`, `status`, `type`). The legacy `Severity` field is
  dropped (not part of the k8s contract).
- The Cluster-level `status.infrastructureReady` boolean is replaced
  by `status.initialization.infrastructureProvisioned` (`*bool`). This
  is a CAPI v1beta2 change, transparent if you only read
  `status.conditions[?(@.type=='InfrastructureReady')].status`.
- `Cluster.spec.infrastructureRef.namespace` is no longer set — the
  infra reference is always in the same namespace as the Cluster.

## 6. Update your ClusterClass and Cluster templates

If you ship custom `ClusterClass` or cluster templates that reference
RKE2 CRDs, bump the apiVersions:

```diff
- apiVersion: controlplane.cluster.x-k8s.io/v1beta1
+ apiVersion: controlplane.cluster.x-k8s.io/v1beta2
  kind: RKE2ControlPlaneTemplate
```

```diff
- apiVersion: bootstrap.cluster.x-k8s.io/v1beta1
+ apiVersion: bootstrap.cluster.x-k8s.io/v1beta2
  kind: RKE2ConfigTemplate
```

The core CAPI types (`Cluster`, `ClusterClass`, `ClusterResourceSet`,
`MachineHealthCheck`) may remain on `cluster.x-k8s.io/v1beta1` in your
rendered YAML — the CAPI v1.12 conversion webhook will transparently
upgrade them to v1beta2 internally. The shipped `caphv-generate` script
already uses the right mix.

## 7. Roll back

If anything goes wrong:

1. Revert the `CAPIProvider/harvester` `spec.version` to `v0.2.9`.
2. Delete `out/infrastructure-components.yaml` (the v0.3 artifact) and
   re-apply the v0.2.9 one.
3. Note that the CAPI ecosystem (CAPI v1.12 + RKE2 v0.24+) does **not**
   need to roll back — v0.2.9 happens to also work against v1.12 because
   v1.10/v1beta1 types are still served via the conversion webhook.

## 8. Validated end-to-end

CAPHV v0.3.0 release was validated on a clean management cluster:

- `mgmt-v030` VM on Harvester 1.8.0 (openSUSE Leap 15.6, RKE2
  v1.34.7+rke2r1)
- Rancher Manager 2.14.1, Rancher Turtles 0.26.2, cert-manager v1.18.2
- CAPI core v1.12.7, rke2-bootstrap v0.25.0, rke2-control-plane v0.25.0
- CAPHV v0.3.0 controller
- Webhooks: 5/5 e2e tests pass
- Unit tests: 195/195
- ClusterClass topology end-to-end: `HarvesterCluster` +
  `HarvesterMachine` + `RKE2ControlPlane` created, VM provisioned on
  Harvester 1.8.0 (KubeVirt 1.7.0), IP allocated from `capi-vm-pool`,
  `providerID` assigned, node Ready, scale-up worker 0→1→0 OK

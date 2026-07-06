# Migrating from v0.4.x to v0.5.x

v0.5.0 graduates the CAPHV API from `v1alpha1` to **`v1beta1`**: the API is now
covered by compatibility guarantees — no breaking change without a conversion path
and a deprecation window.

## What changes

- All four CRDs (`HarvesterCluster`, `HarvesterMachine`, `HarvesterClusterTemplate`,
  `HarvesterMachineTemplate`) serve **both** `v1alpha1` and `v1beta1`; the stored
  version is `v1beta1`. A conversion webhook translates between the two on the fly.
- `v1alpha1` is **deprecated**: it stays served for at least two more minor releases.
  Move your manifests, ClusterClasses and GitOps sources to
  `infrastructure.cluster.x-k8s.io/v1beta1` at your convenience (the schema is
  otherwise identical).
- `status.failureReason` / `status.failureMessage` do **not** exist in `v1beta1` and
  are not carried across conversion: they were deprecated in v0.4.0, the controller
  no longer writes them, and failures surface through the conditions
  (`HarvesterConnectionReady`, ...). Reading an object through `v1alpha1` returns
  these fields empty.

## What you need to do

- **Nothing for existing clusters**: objects are converted on the fly and rewritten
  to the new stored version lazily as they are updated. No downtime, no re-creation.
- **Recommended**: bump the `apiVersion` of your HarvesterCluster/HarvesterMachine
  manifests, templates and ClusterClasses to `v1beta1` (all in-repo templates and the
  Helm-shipped ClusterClass already are).
- If you pin RBAC or tooling to exact CRD versions, allow `v1beta1`.

## Rollback note

Downgrading the provider to v0.4.x after objects have been written as `v1beta1`
requires the CRDs to keep serving `v1beta1` as storage; do not remove the new CRD
version once applied.

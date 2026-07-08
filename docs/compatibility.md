# Compatibility matrix

Rancher drives the CAPI core version (through the Turtles system chart), and the CAPI
core version determines which provider contract is exercised — this matrix records
which CAPHV release works with what, and how each pairing is validated.

| CAPHV | Provider API | CAPI contract published | CAPI core | Rancher / Turtles | Validation |
|-------|--------------|-------------------------|-----------|-------------------|------------|
| v0.5.x | `v1beta1` (`v1alpha1` served, deprecated) | v1beta1 + v1beta2 | v1.12.x | Rancher 2.14.x / Turtles 0.26.x | nightly version-pairing (CAPI v1.12.7), Rancher-stack suite (2.14.1, Turtles 109.0.1+up0.26.1, core v1.12.2), Turtles integration suite on real Harvester |
| v0.4.x | `v1alpha1` | v1beta1 + v1beta2 | v1.12.x | Rancher 2.14.x / Turtles 0.26.x | nightly version-pairing, real-Harvester runs |
| v0.3.x | `v1alpha1` | v1beta1 | v1.12.x | Rancher 2.13–2.14 / Turtles 0.25–0.26 | real-Harvester runs (homelab) |
| v0.2.x | `v1alpha1` | v1beta1 | v1.10.x | Rancher 2.13.x / Turtles 0.25.x | Turtles integration suite (turtles/test v0.25.4, full lifecycle) |

Notes:

- **Contract vs API**: the contract labels on the CRDs (`cluster.x-k8s.io/v1beta1`,
  `v1beta2`) tell the CAPI core how to read the provider objects; the provider API
  version (`v1alpha1`/`v1beta1`) is the schema users write. They evolve independently.
- Turtles pins provider versions only for Rancher Prime (see the SUSE Prime
  documentation); with community Rancher, install CAPHV through a `CAPIProvider` with an
  explicit `version` and `fetchConfig` URL.
- Harvester side: validated against Harvester 1.8.x (VM provisioning, IP pools,
  multi-NIC). Older 1.6/1.7 pairings were exercised by the v0.2.x runs.
- The suites behind the "Validation" column live in `test/certification/` and run in CI
  (`certification.yml` nightly, `certification-tier-a.yml` and the integration suite on
  demand).

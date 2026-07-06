# Release process

Checklist for cutting a CAPHV release (`vX.Y.Z`).

## Before tagging

1. **CHANGELOG.md** — turn `[Unreleased]` into `[vX.Y.Z] - YYYY-MM-DD`.
2. **metadata.yaml** — on a new *minor* series only: add the `major/minor` entry with
   its CAPI contract version (clusterctl refuses to install a version missing from the
   release series).
3. Land the above through a PR (DCO sign-off required), then tag the merge commit:

   ```bash
   git tag vX.Y.Z && git push origin vX.Y.Z
   ```

## Release workflow

The `release.yml` workflow triggers on the tag and produces: the multi-arch image on
GHCR, the Helm OCI chart, `infrastructure-components.yaml` + `metadata.yaml` +
cluster templates as release assets, cosign signatures (verify with **cosign v3+**:
v2 wrongly reports "no signatures" on OCI 1.1 bundles) and build provenance.

## After the release is published

1. **Certification defaults** — bump `CAPHV_VERSION` and `CAPHV_COMPONENTS_URL` in
   `test/certification/config/config.yaml` so the nightly certifies the new release
   (only after the assets exist, otherwise the nightly fails on the download URL).
2. Spot-check the assets: the `infrastructure-components.yaml` URL resolves and
   references the `vX.Y.Z` image.
3. Optionally trigger the on-demand `certification-tier-a` workflow against the new
   version.

# CAPHV Version-Pairing Certification

This directory contains the version-pairing certification suite for CAPHV (Cluster API
Provider Harvester). It validates that a CAPHV release installs cleanly and stays
compatible with the targeted Cluster API ecosystem: the suite self-deploys a kind
management cluster with cert-manager and [cluster-api-operator], then installs the CAPI
core, the RKE2 bootstrap/control-plane providers and the CAPHV release under test, and
asserts the pairing is healthy.

**No Rancher, no Turtles and no Harvester endpoint are required**, so the suite runs
nightly on a standard GitHub-hosted runner (see `.github/workflows/certification.yml`).

> Why not Rancher Turtles? Turtles 0.26+ cannot run standalone: its controller watches
> Rancher's `management.cattle.io` CRDs and crashloops without a full Rancher install.
> This tier therefore certifies the deeper CAPHV↔CAPI contract through the
> Rancher-independent cluster-api-operator. The full Rancher + Turtles integration
> (real Harvester provisioning + auto-import) is a separate on-demand e2e tier; the
> `suites/data/cluster-templates/harvester-rke2-topology.yaml` template is retained
> for it.

[cluster-api-operator]: https://github.com/kubernetes-sigs/cluster-api-operator

## What the suite validates

1. The four providers (CAPI core, RKE2 bootstrap, RKE2 control-plane, CAPHV
   infrastructure) all reach `Ready=True`, with the targeted versions installed.
2. The `caphv-controller-manager` deployment becomes Available under the targeted CAPI
   core.
3. The CAPHV CRDs are registered and advertise a `cluster.x-k8s.io/<contract>` label the
   core accepts.

The bootstrap replicates `hack/tier-c-smoke.sh` — the manually validated recipe — step by
step. If you change one, keep the other in sync.

## Version matrix

All pins live in `config/config.yaml` (`variables:`) and are overridable from the
environment:

| Variable | Default | Meaning |
|----------|---------|---------|
| `CAPHV_VERSION` | `v0.3.1` | CAPHV release under certification |
| `CAPHV_COMPONENTS_URL` | release asset URL | `infrastructure-components.yaml` of that release |
| `CAPI_VERSION` | `v1.12.7` | Targeted Cluster API core |
| `CAPI_OPERATOR_VERSION` | `0.27.0` | cluster-api-operator helm chart |

The RKE2 providers deliberately carry no pin: the operator installs its latest known
release, matching the validated recipe. cert-manager's version is pinned by the turtles
`DeployCertManager` helper.

## Prerequisites

- A container runtime for kind: docker, or **rootful** podman (the kind library
  auto-detects docker → nerdctl → podman; the CLI-only `KIND_EXPERIMENTAL_PROVIDER`
  variable has no effect here). No kind binary is needed — kind is vendored as a Go
  library.
- `helm` (v3+) and `kubectl` on `PATH`.
- Go (version per `go.mod`).
- Internet access (chart repos, GitHub release assets, ghcr.io).

## Running

```bash
# Wrapper script (resolves helm path, creates artifacts folder)
./run.sh

# Keep the kind cluster for debugging
./run.sh --skip-cleanup

# Or via make (same defaults)
make test

# Certify a different pairing
CAPHV_VERSION=v0.3.2 \
CAPHV_COMPONENTS_URL=https://github.com/rancher-sandbox/cluster-api-provider-harvester/releases/download/v0.3.2/infrastructure-components.yaml \
./run.sh
```

From the project root: `make certification-test` / `make certification-build`.

On a podman host, build first and run the compiled binary as root (kind needs rootful
podman, and this keeps Go caches owned by your user):

```bash
make build
sudo -E env "PATH=$PATH" HOME=/root \
  E2E_CONFIG=$PWD/config/config.yaml ARTIFACTS_FOLDER=$PWD/_artifacts \
  HELM_BINARY_PATH=$(command -v helm) \
  ./bin/certification.test -test.timeout 60m -ginkgo.v -ginkgo.label-filter=short
```

## Structure

```
test/certification/
├── config/config.yaml            # E2E config: version matrix, intervals, kind settings
├── hack/tier-c-smoke.sh          # Manually validated recipe the suite replicates
├── suites/
│   ├── const.go                  # Embedded manifests (build tag: e2e)
│   ├── data/
│   │   ├── providers/            # cluster-api-operator provider CRs (envsubst templates)
│   │   │   ├── core.yaml         #   CoreProvider (CAPI ${CAPI_VERSION})
│   │   │   ├── rke2.yaml         #   RKE2 Bootstrap/ControlPlane providers
│   │   │   └── harvester.yaml    #   CAPHV InfrastructureProvider (${CAPHV_VERSION})
│   │   └── cluster-templates/    # ClusterClass template (on-demand e2e tier)
│   └── version-pairing/
│       ├── suite_test.go         # Bootstrap: kind -> cert-manager -> operator -> providers
│       └── version_pairing_test.go  # Certification assertions
├── Makefile
├── run.sh
├── go.mod                        # Standalone module (decoupled from the main CAPHV module)
└── README.md
```

## Certification submission

After the suite passes for a new pairing, a certification issue can be submitted on
[rancher/turtles](https://github.com/rancher/turtles/issues) with the test logs, the
CAPHV version and a link to this suite.

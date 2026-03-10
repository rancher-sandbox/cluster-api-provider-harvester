# CAPHV Turtles Certification Test

This directory contains the Rancher Turtles certification test suite for CAPHV
(Cluster API Provider Harvester). It validates that CAPHV works correctly with
the Turtles auto-import workflow: deploy via CAPIProvider, create a cluster via
ClusterClass topology, and auto-import into Rancher.

## Prerequisites

- Rancher Manager cluster with Turtles installed (v2.13+)
- CAPHV deployed as CAPIProvider (`harvester` in `caphv-system`)
- Harvester HCI cluster accessible from the management cluster
- kubectl and helm available on PATH
- Go 1.24+

## Configuration

Edit `config/config.yaml` to match your environment:

- `RANCHER_HOSTNAME`: Rancher Manager hostname
- `KUBERNETES_VERSION`: Target K8s version for workload cluster
- Harvester-specific variables (VM_NETWORK, VOLUME_IMAGE, etc.)

## Running

```bash
# Set the Harvester kubeconfig (base64-encoded)
export HARVESTER_KUBECONFIG_B64=$(base64 -w0 < /path/to/harvester-kubeconfig.yaml)

# Run with wrapper script
chmod +x run.sh
./run.sh

# Or via Makefile
make test

# Skip cleanup for debugging
./run.sh --skip-cleanup --skip-deletion
```

## From the project root

```bash
make certification-test KUBECONFIG=~/.kube/config
```

## Structure

```
test/certification/
├── config/config.yaml           # E2E config (intervals, variables)
├── suites/
│   ├── const.go                 # Embedded YAML data (build tag: e2e)
│   ├── data/
│   │   ├── providers/
│   │   │   └── harvester.yaml   # CAPIProvider manifest
│   │   └── cluster-templates/
│   │       └── harvester-rke2-topology.yaml  # Full ClusterClass + Cluster template
│   └── import-gitops/
│       ├── suite_test.go        # BeforeSuite / AfterSuite
│       └── import_gitops_test.go  # CreateUsingGitOpsSpec call
├── Makefile
├── run.sh
├── go.mod                       # Standalone module (separate deps from main CAPHV)
└── README.md
```

## What the test validates

1. ClusterClass + topology-based cluster creation on Harvester
2. CAPI machines become Ready (VM provisioning, IP allocation, RKE2 bootstrap)
3. Control plane Ready
4. Rancher auto-import via Turtles (namespace label)
5. Rancher cluster agent deployed and cluster Connected
6. Cluster deletion and cleanup

## Certification submission

After all tests pass, submit a certification issue on
[rancher/turtles](https://github.com/rancher/turtles/issues) with:

- Provider name: `harvester` (infrastructure)
- Test logs
- CAPHV version and Harvester version
- Link to this test suite

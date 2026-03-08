#!/bin/bash
# Wrapper script for running CAPHV Turtles certification tests.
# Usage: ./run.sh [--skip-cleanup] [--skip-deletion]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

# Defaults
export KUBECONFIG="${KUBECONFIG:-$HOME/.kube/config}"
export E2E_CONFIG="${E2E_CONFIG:-$SCRIPT_DIR/config/config.yaml}"
export ARTIFACTS_FOLDER="${ARTIFACTS_FOLDER:-$SCRIPT_DIR/_artifacts}"
export USE_EXISTING_CLUSTER=true

# Parse args
for arg in "$@"; do
    case $arg in
        --skip-cleanup)
            export SKIP_RESOURCE_CLEANUP=true
            ;;
        --skip-deletion)
            export SKIP_DELETION_TEST=true
            ;;
    esac
done

# Validate prerequisites
if [ ! -f "$KUBECONFIG" ]; then
    echo "ERROR: KUBECONFIG not found at $KUBECONFIG"
    exit 1
fi

if [ -z "${HARVESTER_KUBECONFIG_B64:-}" ]; then
    echo "WARNING: HARVESTER_KUBECONFIG_B64 is not set."
    echo "  Export it with: export HARVESTER_KUBECONFIG_B64=\$(base64 -w0 < /path/to/harvester-kubeconfig.yaml)"
fi

mkdir -p "$ARTIFACTS_FOLDER"

echo "=== CAPHV Turtles Certification Test ==="
echo "  KUBECONFIG:    $KUBECONFIG"
echo "  E2E_CONFIG:    $E2E_CONFIG"
echo "  ARTIFACTS:     $ARTIFACTS_FOLDER"
echo "  SKIP_CLEANUP:  ${SKIP_RESOURCE_CLEANUP:-false}"
echo "  SKIP_DELETION: ${SKIP_DELETION_TEST:-false}"
echo ""

exec go test -v -tags e2e -timeout 60m -count 1 \
    ./suites/import-gitops/ \
    -ginkgo.v \
    -ginkgo.label-filter="full"

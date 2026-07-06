#!/bin/bash
# Wrapper for the CAPHV version-pairing certification suite (tier C).
#
# Self-deploys a management cluster (kind) with cert-manager + cluster-api-operator, then the
# CAPI core, RKE2 and CAPHV providers, and validates that the CAPHV release under test is
# compatible with the targeted CAPI ecosystem. No Rancher, no Turtles and no Harvester
# endpoint are required. All version pins live in config/config.yaml and are overridable
# from the environment (CAPHV_VERSION, CAPHV_COMPONENTS_URL, CAPI_VERSION, ...).
#
# Usage: ./run.sh [--skip-cleanup]
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

export E2E_CONFIG="${E2E_CONFIG:-$SCRIPT_DIR/config/config.yaml}"
export ARTIFACTS_FOLDER="${ARTIFACTS_FOLDER:-$SCRIPT_DIR/_artifacts}"
# The config validation requires an existing file, so resolve the absolute helm path.
export HELM_BINARY_PATH="${HELM_BINARY_PATH:-$(command -v helm)}"

for arg in "$@"; do
    case $arg in
        --skip-cleanup)
            export SKIP_RESOURCE_CLEANUP=true
            ;;
    esac
done

mkdir -p "$ARTIFACTS_FOLDER"

echo "=== CAPHV version-pairing certification (tier C) ==="
echo "  CAPHV:        ${CAPHV_VERSION:-<config default>}"
echo "  CAPI:         ${CAPI_VERSION:-<config default>}"
echo "  E2E_CONFIG:   $E2E_CONFIG"
echo "  ARTIFACTS:    $ARTIFACTS_FOLDER"
echo "  SKIP_CLEANUP: ${SKIP_RESOURCE_CLEANUP:-false}"
echo ""

exec go test -v -tags e2e -timeout 60m -count 1 \
    ./suites/version-pairing/ \
    -ginkgo.v \
    -ginkgo.label-filter="short"

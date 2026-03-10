#!/bin/bash
# Fleet integration tests for CAPHV
# Tests Fleet mode GitRepo generation and CAAPF detection
#
# Prerequisites:
#   - CAAPF controller running in caapf-system
#   - Fleet controller running
#   - caphv-fleet-addons repo accessible
#   - Existing CAPI cluster in ${NAMESPACE}
#
# Usage: sourced by run-e2e.sh

test_fleet() {
    log_test "Test: Fleet/CAAPF Integration"

    # Test 1: CAAPF controller running
    log_info "Checking CAAPF controller"
    local caapf_ready
    caapf_ready=$(kubectl_rancher "get deploy -n caapf-system caapf-controller-manager -o jsonpath='{.status.readyReplicas}'" 2>/dev/null || echo 0)
    if [ "$caapf_ready" = "1" ]; then
        pass_test "Fleet: CAAPF controller running"
    else
        fail_test "Fleet: CAAPF controller not ready (readyReplicas=$caapf_ready)"
        return
    fi

    # Test 2: Fleet mode generates correct YAML (CSI in CRS, CNI in Fleet)
    log_info "Testing Fleet mode YAML generation"
    local test_output
    test_output=$(SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)" && \
        "$SCRIPT_DIR/bin/caphv-generate" \
        --name fleet-e2e-test \
        --image "default/sles15-sp7-minimal-vm.x86_64-cloud-qu2.qcow2" \
        --ssh-keypair "default/capi-ssh-key" \
        --network "default/production" \
        --gateway 172.16.0.1 \
        --subnet-mask 255.255.0.0 \
        --ip-pool capi-vm-pool \
        --cni calico \
        --cni-mtu 1450 \
        --fleet-addon-repo https://gitea.home.zypp.fr/rancher-sandbox/caphv-fleet-addons.git \
        --harvester-kubeconfig /dev/null 2>/dev/null || echo "GENERATE_FAILED")

    if echo "$test_output" | grep -q "GENERATE_FAILED"; then
        # Need a real kubeconfig for base64 — use a dummy
        local tmpkc
        tmpkc=$(mktemp)
        echo "dummy" > "$tmpkc"
        test_output=$("$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/bin/caphv-generate" \
            --name fleet-e2e-test \
            --image "default/sles15-sp7-minimal-vm.x86_64-cloud-qu2.qcow2" \
            --ssh-keypair "default/capi-ssh-key" \
            --network "default/production" \
            --gateway 172.16.0.1 \
            --subnet-mask 255.255.0.0 \
            --ip-pool capi-vm-pool \
            --cni calico \
            --cni-mtu 1450 \
            --fleet-addon-repo https://gitea.home.zypp.fr/rancher-sandbox/caphv-fleet-addons.git \
            --harvester-kubeconfig "$tmpkc" 2>/dev/null)
        rm -f "$tmpkc"
    fi

    local gitrepo_count crs_csi_count crs_cni_count
    gitrepo_count=$(echo "$test_output" | grep -c "kind: GitRepo" || true)
    gitrepo_count="${gitrepo_count:-0}"
    crs_csi_count=$(echo "$test_output" | grep -c "name: crs-harvester-csi" || true)
    crs_csi_count="${crs_csi_count:-0}"
    crs_cni_count=$(echo "$test_output" | grep -c "name: crs-calico-chart-config" || true)
    crs_cni_count="${crs_cni_count:-0}"

    if [ "$gitrepo_count" -eq 1 ] && [ "$crs_csi_count" -eq 1 ] && [ "$crs_cni_count" -eq 0 ]; then
        pass_test "Fleet: Fleet mode generates GitRepo + CSI CRS, no CNI CRS"
    else
        fail_test "Fleet: expected 1 GitRepo, 1 CSI CRS, 0 CNI CRS (got $gitrepo_count/$crs_csi_count/$crs_cni_count)"
    fi

    # Test 3: CRS mode retrocompatibility
    log_info "Testing CRS mode retrocompatibility"
    local tmpkc
    tmpkc=$(mktemp)
    echo "dummy" > "$tmpkc"
    local crs_output
    crs_output=$("$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)/bin/caphv-generate" \
        --name crs-e2e-test \
        --image "default/sles15-sp7-minimal-vm.x86_64-cloud-qu2.qcow2" \
        --ssh-keypair "default/capi-ssh-key" \
        --network "default/production" \
        --gateway 172.16.0.1 \
        --subnet-mask 255.255.0.0 \
        --ip-pool capi-vm-pool \
        --harvester-kubeconfig "$tmpkc" 2>/dev/null)
    rm -f "$tmpkc"

    local crs_count gitrepo_crs_count
    crs_count=$(echo "$crs_output" | grep -c "kind: ClusterResourceSet" || true)
    crs_count="${crs_count:-0}"
    gitrepo_crs_count=$(echo "$crs_output" | grep -c "kind: GitRepo" || true)
    gitrepo_crs_count="${gitrepo_crs_count:-0}"

    if [ "$crs_count" -eq 3 ] && [ "$gitrepo_crs_count" -eq 0 ]; then
        pass_test "Fleet: CRS mode retrocompatible (3 CRS, 0 GitRepo)"
    else
        fail_test "Fleet: CRS mode broken (expected 3 CRS / 0 GitRepo, got $crs_count/$gitrepo_crs_count)"
    fi
}

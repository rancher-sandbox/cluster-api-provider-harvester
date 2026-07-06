#!/usr/bin/env bash
#
# Manual reproduction of the CAPHV "version-pairing" certification — tier C.
#
# Validates that a CAPHV release is compatible with the targeted CAPI core + RKE2 providers,
# deployed via cluster-api-operator (which, unlike Rancher Turtles 0.26+, is Rancher-independent).
# NO Rancher, NO Turtles, NO Harvester endpoint required. This is the exact recipe the Go
# certification suite automates; keep them in sync.
#
# Validated end-to-end on kind 2026-07-06: all four providers reach READY=True and
# caphv-controller-manager becomes Available.
#
# Requirements: kind, helm (v3+), kubectl, a container runtime.
#   On a host where kind needs rootful podman, run kind under:
#     sudo -E env "PATH=$PATH" KIND_EXPERIMENTAL_PROVIDER=podman <full-path-to-kind> ...
#   (a `kind` network created with --ipv6=false may be required first).
#
# Findings that make this recipe non-obvious (all discovered by real runs):
#   - Rancher Turtles 0.26.2 standalone CANNOT be used without Rancher: its controller watches
#     management.cattle.io/v3 (Setting, Cluster) CRDs, crashloops without them, and cannot
#     initialise ClusterctlConfig (needs the Rancher system-default-registry Setting).
#   - cluster-api-operator's InfrastructureProvider needs spec.version set EXPLICITLY; otherwise
#     it tries to parse the version from the fetchConfig URL and picks "download" -> IncorrectVersionFormat.
#   - `kubectl create ns` takes a single name at a time.
#
set -euo pipefail

CAPHV_VERSION="${CAPHV_VERSION:-v0.3.1}"
CAPI_VERSION="${CAPI_VERSION:-v1.12.7}"
CERT_MANAGER_VERSION="${CERT_MANAGER_VERSION:-v1.16.3}"
CAPHV_COMPONENTS_URL="${CAPHV_COMPONENTS_URL:-https://github.com/rancher-sandbox/cluster-api-provider-harvester/releases/download/${CAPHV_VERSION}/infrastructure-components.yaml}"

echo "### cert-manager ${CERT_MANAGER_VERSION}"
helm repo add jetstack https://charts.jetstack.io --force-update >/dev/null
helm install cert-manager jetstack/cert-manager -n cert-manager --create-namespace \
  --set crds.enabled=true --version "${CERT_MANAGER_VERSION}" --wait --timeout 5m

echo "### cluster-api-operator (Rancher-independent; NO value overrides)"
helm repo add capi-operator https://kubernetes-sigs.github.io/cluster-api-operator --force-update >/dev/null
helm repo update >/dev/null
helm install capi-operator capi-operator/cluster-api-operator \
  -n capi-operator-system --create-namespace --wait --timeout 5m

echo "### provider config secret (CAPI experimental features) in every provider namespace"
for ns in capi-system caphv-system rke2-bootstrap-system rke2-control-plane-system; do
  kubectl create namespace "$ns" 2>/dev/null || true
  kubectl -n "$ns" create secret generic variables \
    --from-literal=CLUSTER_TOPOLOGY=true \
    --from-literal=EXP_CLUSTER_RESOURCE_SET=true \
    --from-literal=EXP_MACHINE_POOL=true 2>/dev/null || true
done

echo "### CoreProvider (CAPI ${CAPI_VERSION})"
kubectl apply -f - <<EOF
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: CoreProvider
metadata: {name: cluster-api, namespace: capi-system}
spec:
  version: ${CAPI_VERSION}
  configSecret: {name: variables}
EOF
kubectl wait --for=condition=Ready coreprovider/cluster-api -n capi-system --timeout=5m

echo "### RKE2 bootstrap + control-plane, and CAPHV ${CAPHV_VERSION} infrastructure provider"
kubectl apply -f - <<EOF
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: BootstrapProvider
metadata: {name: rke2, namespace: rke2-bootstrap-system}
spec: {configSecret: {name: variables}}
---
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: ControlPlaneProvider
metadata: {name: rke2, namespace: rke2-control-plane-system}
spec: {configSecret: {name: variables}}
---
apiVersion: operator.cluster.x-k8s.io/v1alpha2
kind: InfrastructureProvider
metadata: {name: harvester, namespace: caphv-system}
spec:
  version: ${CAPHV_VERSION}          # MUST be explicit (see findings above)
  configSecret: {name: variables}
  fetchConfig:
    url: ${CAPHV_COMPONENTS_URL}
EOF

echo "### wait for all providers + the CAPHV controller"
kubectl wait --for=condition=Ready infrastructureprovider/harvester -n caphv-system --timeout=5m
kubectl rollout status deploy/caphv-controller-manager -n caphv-system --timeout=5m

echo "### RESULT"
kubectl get coreprovider,bootstrapprovider,controlplaneprovider,infrastructureprovider -A
kubectl get crd harvesterclusters.infrastructure.cluster.x-k8s.io \
  -o jsonpath='{.metadata.labels}' && echo
echo "PASS: CAPHV ${CAPHV_VERSION} is compatible with CAPI ${CAPI_VERSION} + RKE2 (no Rancher)."

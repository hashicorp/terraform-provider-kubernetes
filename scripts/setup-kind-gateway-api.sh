#!/usr/bin/env bash
# Copyright IBM Corp. 2017, 2026
# SPDX-License-Identifier: MPL-2.0
#
# Sets up a Kind cluster with Gateway API CRDs and cloud-provider-kind
# for acceptance testing of Gateway API v1 resources.

set -euo pipefail

CLUSTER_NAME="${KIND_CLUSTER_NAME:-tf-gateway-api-test}"
GATEWAY_API_VERSION="${GATEWAY_API_VERSION:-v1.5.1}"

log() { echo "[INFO] $*"; }
warn() { echo "[WARN] $*"; }
fail() { echo "[FAIL] $*" >&2; exit 1; }

command -v kind >/dev/null 2>&1 || fail "kind not found"
command -v kubectl >/dev/null 2>&1 || fail "kubectl not found"
command -v docker >/dev/null 2>&1 || fail "docker not found"

if ! kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
    log "Creating Kind cluster: ${CLUSTER_NAME}"
    kind create cluster --name "${CLUSTER_NAME}" --wait 60s || fail "Failed to create cluster"
else
    log "Cluster ${CLUSTER_NAME} already exists"
fi

kind get kubeconfig --name "${CLUSTER_NAME}" > /tmp/kind-kubeconfig
export KUBECONFIG="/tmp/kind-kubeconfig"
log "Kubeconfig written to /tmp/kind-kubeconfig"

log "Installing Gateway API CRDs ${GATEWAY_API_VERSION} (experimental channel)"
kubectl apply --server-side -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/experimental-install.yaml" || \
    fail "Failed to install Gateway API CRDs"

log "Verifying Gateway API CRDs..."
for crd in gateways.gateway.networking.k8s.io httproutes.gateway.networking.k8s.io \
           grpcroutes.gateway.networking.k8s.io tlsroutes.gateway.networking.k8s.io \
           backendtlspolicies.gateway.networking.k8s.io referencegrants.gateway.networking.k8s.io \
           listenersets.gateway.networking.k8s.io gatewayclasses.gateway.networking.k8s.io; do
    if kubectl get crd "${crd}" >/dev/null 2>&1; then
        log "  OK: ${crd}"
    else
        warn "  MISSING: ${crd}"
    fi
done

log "Deploying cloud-provider-kind (LoadBalancer + Gateway API controller)"
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/cloud-provider-kind/master/install.yaml 2>/dev/null || \
    warn "cloud-provider-kind may already be deployed"

log "Waiting for cloud-provider-kind..."
kubectl wait --namespace cloud-provider-kind \
    --for=condition=ready pod \
    --selector=app=cloud-provider-kind \
    --timeout=120s 2>/dev/null || warn "cloud-provider-kind not ready (check manually)"

log "Creating test namespaces"
kubectl create namespace gateway-api-tests --dry-run=client -o yaml | kubectl apply -f - 2>/dev/null || true
kubectl create namespace backend-ns --dry-run=client -o yaml | kubectl apply -f - 2>/dev/null || true

log "Deploying sample backend services for cross-namespace tests"
kubectl apply -n backend-ns -f - <<'EOF'
apiVersion: v1
kind: Service
metadata:
  name: backend-service
spec:
  selector:
    app: backend
  ports:
    - port: 80
      targetPort: 8080
    - port: 443
      targetPort: 8443
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
        - name: agnhost
          image: registry.k8s.io/e2e-test-images/agnhost:2.43
          args: ["netexec", "--http-port=8080", "--https-port=8443"]
          ports:
            - containerPort: 8080
            - containerPort: 8443
EOF

kubectl apply -n gateway-api-tests -f - <<'EOF'
apiVersion: v1
kind: Service
metadata:
  name: local-service
spec:
  selector:
    app: local-backend
  ports:
    - port: 80
      targetPort: 8080
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: local-backend
spec:
  replicas: 1
  selector:
    matchLabels:
      app: local-backend
  template:
    metadata:
      labels:
        app: local-backend
    spec:
      containers:
        - name: agnhost
          image: registry.k8s.io/e2e-test-images/agnhost:2.43
          args: ["netexec", "--http-port=8080"]
          ports:
            - containerPort: 8080
EOF

log "Creating self-signed TLS certificates for tests"
openssl req -x509 -newkey rsa:2048 -keyout /tmp/tls.key -out /tmp/tls.crt \
    -days 365 -nodes -subj "/CN=*.example.com" 2>/dev/null || true
kubectl create secret tls test-tls-cert \
    --cert=/tmp/tls.crt --key=/tmp/tls.key \
    --namespace=gateway-api-tests --dry-run=client -o yaml | kubectl apply -f - 2>/dev/null || true

log ""
log "========================================"
log "Setup complete!"
log "  Cluster:    ${CLUSTER_NAME}"
log "  Kubeconfig: /tmp/kind-kubeconfig"
log "========================================"
log ""
log "To run acceptance tests:"
log "  export KUBE_CONFIG_PATH=/tmp/kind-kubeconfig"
log "  export TF_ACC=1"
log "  go test -v -run 'TestAcc.*Gateway.*|TestAcc.*Route.*|TestAcc.*BackendTLS.*|TestAcc.*ListenerSet.*|TestAcc.*ReferenceGrant.*' ./kubernetes/ -timeout 60m"
log ""
log "To run only integration tests:"
log "  go test -v -run 'TestAccGatewayAPIIntegration' ./kubernetes/ -timeout 60m"
log ""
log "To destroy the cluster:"
log "  kind delete cluster --name ${CLUSTER_NAME}"

#!/bin/bash

# Test script for NamespaceClassOperator
# This script runs the complete test suite

set -e

echo "🧪 Running NamespaceClassOperator Test Suite"
echo "=============================================="

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl is not installed or not in PATH"
    exit 1
fi

# Check if cluster is accessible
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot access Kubernetes cluster"
    exit 1
fi

echo ""
print_info "Step 1: Installing custom CRD..."
kubectl apply -f 01-crd-appconfig.yaml
print_status "Custom CRD installed"

echo ""
print_info "Step 2: Creating NamespaceClassItem..."
kubectl apply -f 02-namespaceclassitem-appconfig.yaml
print_status "NamespaceClassItem created"

echo ""
print_info "Step 3: Creating NamespaceClass..."
kubectl apply -f 03-namespaceclass-dev.yaml
print_status "NamespaceClass created"

echo ""
print_info "Step 4: Creating test namespace..."
kubectl apply -f 04-namespace-test-app.yaml
print_status "Test namespace created"

echo ""
print_info "Step 5: Verifying resources..."

# Wait for resources to be created
sleep 5

# Check if AppConfig was created in the test namespace
if kubectl get appconfig -n test-app-namespace &> /dev/null; then
    print_status "AppConfig resource created successfully"
    echo "AppConfig details:"
    kubectl get appconfig -n test-app-namespace -o yaml
else
    print_error "AppConfig resource was not created"
fi

echo ""
print_info "Step 6: Checking operator logs..."
kubectl logs -n namespaceclassoperator-system deployment/namespaceclassoperator-controller-manager --tail=20

echo ""
print_info "Test completed! Resources created:"
echo "  - CRD: appconfigs.example.com"
echo "  - NamespaceClassItem: appconfig-dev-item"
echo "  - NamespaceClass: development-class"
echo "  - Namespace: test-app-namespace"
echo "  - AppConfig: (should be auto-created in test-app-namespace)"

echo ""
print_info "To clean up: kubectl delete -f ."
#!/bin/bash

# Cleanup script for test resources
# Run this to clean up all test resources

echo "🧹 Cleaning up test resources..."
echo "================================="

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_info() {
    echo -e "${YELLOW}ℹ${NC} $1"
}

echo ""
print_info "Deleting test namespace..."
kubectl delete namespace test-app-namespace --ignore-not-found=true
print_status "Test namespace deleted"

echo ""
print_info "Deleting NamespaceClass..."
kubectl delete namespaceclass development-class -n namespaceclassoperator-system --ignore-not-found=true
print_status "NamespaceClass deleted"

echo ""
print_info "Deleting NamespaceClassItem..."
kubectl delete namespaceclassitem appconfig-dev-item -n namespaceclassoperator-system --ignore-not-found=true
print_status "NamespaceClassItem deleted"

echo ""
print_info "Deleting custom CRD..."
kubectl delete crd appconfigs.example.com --ignore-not-found=true
print_status "Custom CRD deleted"

echo ""
print_info "Cleanup complete! All test resources have been removed."
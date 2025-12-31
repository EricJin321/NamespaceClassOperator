#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Setting up kubectl for local k3s development environment${NC}"

# Function to print status messages
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Create kubeconfig directory if it doesn't exist
mkdir -p kubeconfig

# Copy kubeconfig from container
print_status "Copying kubeconfig from k3s container..."
docker cp k3s-server:/output/kubeconfig.yaml kubeconfig/config

# Set KUBECONFIG environment variable
export KUBECONFIG=$(pwd)/kubeconfig/config

# Verify kubectl connection
print_status "Verifying kubectl connection..."
kubectl get nodes

# Install CRDs
print_status "Installing CRDs..."
make install

# Install cert-manager
print_status "Installing cert-manager..."
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.12.0/cert-manager.yaml

# Wait for cert-manager to be ready
print_status "Waiting for cert-manager to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager -n cert-manager
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-webhook -n cert-manager
kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-cainjector -n cert-manager

kubectl create namespace namespaceclass-test

# Deploy the operator (assuming image is already built)
print_status "Deploying the operator..."
make deploy IMG=yijinregistry.azurecr.io/namespaceclass-operator:latest

# Wait for deployment to be ready
print_status "Waiting for operator deployment to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/namespaceclassoperator-controller-manager -n namespaceclassoperator-system

print_status "🎉 NamespaceClassOperator is now running locally!"
echo ""
echo -e "${BLUE}Useful commands:${NC}"
echo "  • View pods: kubectl get pods -n namespaceclassoperator-system"
echo "  • View logs: kubectl logs -f deployment/namespaceclassoperator-controller-manager -n namespaceclassoperator-system"
echo ""
echo -e "${YELLOW}Note: Your kubeconfig is available at: $(pwd)/kubeconfig/config${NC}"
echo -e "${YELLOW}You can set KUBECONFIG=$(pwd)/kubeconfig/config to use kubectl with this cluster${NC}"
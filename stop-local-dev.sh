#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🛑 Stopping NamespaceClassOperator${NC}"

# Function to print status messages
print_status() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if kubectl is configured
if kubectl cluster-info &> /dev/null; then
    # Undeploy the operator
    print_status "Undeploying the operator..."
    make undeploy

    # Uninstall CRDs
    print_status "Uninstalling CRDs..."
    make uninstall
else
    print_status "kubectl not configured, skipping operator cleanup..."
fi

print_status "NamespaceClassOperator stopped"
echo ""
echo -e "${YELLOW}Note: To stop the K3S cluster and registry, run:${NC}"
echo "  docker-compose down"
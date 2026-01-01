# Local Development Environment

This directory contains files to set up a local Kubernetes development environment using Docker Compose and k3s.

## Files

- `docker-compose.yml`: Docker Compose configuration for running k3s and local registry in containers
- `registries.yaml`: K3S registry configuration for local Docker registry
- `start-local-dev.sh`: Script to configure kubectl and deploy the NamespaceClassOperator
- `stop-local-dev.sh`: Script to stop the operator and clean up resources
- `LOCAL_DEV_README.md`: This documentation file

## Prerequisites

- Docker
- Docker Compose (or Docker Compose v2)
- make
- kubectl (optional, will be configured automatically)

## Quick Start

1. **Start the k3s cluster and registry:**
   ```bash
   docker-compose up -d
   ```

2. **Build and push the operator image:**
   ```bash
   make docker-build IMG=localhost:5000/namespaceclass-operator:latest
   make docker-push IMG=localhost:5000/namespaceclass-operator:latest
   ```

3. **Deploy the operator:**
   ```bash
   ./start-local-dev.sh
   ```

   This script will:
   - Copy kubeconfig from the k3s container
   - Configure kubectl to use the local cluster
   - Install CRDs
   - Deploy the operator (including the namespaceclass-test namespace)
   - Wait for the deployment to be ready

3. **Verify the deployment:**
   ```bash
   # Check if the cluster is running
   kubectl get nodes

   # Check operator pods
   kubectl get pods -n namespaceclassoperator-system

   # View operator logs
   kubectl logs -f deployment/namespaceclassoperator-controller-manager -n namespaceclassoperator-system
   ```

4. **Test the operator:**
   ```bash
   # Create sample resources
   kubectl apply -f config/samples/

   # Check created resources
   kubectl get namespaceclass
   kubectl get namespaceclassitem
   kubectl get namespacestate
   kubectl get namespaceiteminstance -n test-app-namespace
   kubectl get appconfig -n test-app-namespace
   ```

5. **Stop the operator:**
   ```bash
   ./stop-local-dev.sh
   ```

   This script will:
   - Undeploy the operator
   - Remove CRDs
   - Clean up the namespaceclass-test namespace

6. **Stop the k3s cluster:**
   ```bash
   docker-compose down
   # or: docker compose down
   ```
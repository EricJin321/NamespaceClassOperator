# Local Development Environment

This directory contains files to set up a local Kubernetes development environment using Docker Compose and k3s.

## Files

- `docker-compose.yml`: Docker Compose configuration for running k3s in a container
- `start-local-dev.sh`: Script to start the cluster and deploy the NamespaceClassOperator
- `stop-local-dev.sh`: Script to stop the local development environment
- `LOCAL_DEV_README.md`: This documentation file

## Prerequisites

- Docker
- Docker Compose (or Docker Compose v2)
- make
- kubectl (optional, will be configured automatically)

## Quick Start

1. **Start the k3s cluster:**
   ```bash
   docker-compose up -d
   ```

2. **Deploy the operator:**
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
   kubectl get namespaceiteminstance
   kubectl get namespacestate
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

## Useful Commands

### Cluster Management
```bash
# Start cluster
docker-compose up -d

# Deploy operator
./start-local-dev.sh

# Stop operator (keeps cluster running)
./stop-local-dev.sh

# Stop cluster
docker-compose down

# Stop and remove volumes (reset cluster)
docker-compose down -v

# View cluster logs
docker-compose logs -f k3scluster
```

### kubectl Configuration
The script automatically sets up kubectl configuration. If you need to configure it manually:

```bash
export KUBECONFIG=$(pwd)/kubeconfig/config
```

### Operator Management
```bash
# Rebuild and redeploy operator
make docker-build IMG=namespaceclassoperator:local
make deploy IMG=namespaceclassoperator:local

# View CRDs
kubectl get crd | grep policy.akuity.io

# Clean up
make undeploy
make uninstall
```

## Troubleshooting

1. **k3s fails to start:**
   - Check Docker resources (k3s needs ~2GB RAM)
   - Try: `docker-compose down -v` then restart

2. **Operator fails to deploy:**
   - Check pod status: `kubectl describe pod -n namespaceclassoperator-system`
   - Check logs: `kubectl logs deployment/namespaceclassoperator-controller-manager -n namespaceclassoperator-system`

3. **kubectl connection issues:**
   - Verify kubeconfig: `kubectl config current-context`
   - Check cluster status: `kubectl cluster-info`

## Architecture

- **k3s**: Lightweight Kubernetes distribution running in Docker
- **NamespaceClassOperator**: Custom controller managing namespace policies
- **Local Development**: Isolated environment for testing without affecting host system

The setup provides a complete Kubernetes environment for developing and testing the operator locally.
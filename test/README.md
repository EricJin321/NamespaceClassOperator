# Test Case: Application Namespace Configuration
# This test demonstrates how to configure a namespace with custom application settings

## Overview
This test creates a complete namespace configuration scenario including:
- A custom AppConfig CRD for application-specific settings
- NamespaceClassItem with custom CR spec
- NamespaceClass that references the item
- A test namespace that uses the configuration

## Test Files
- 01-crd-appconfig.yaml: Custom AppConfig CRD definition
- 02-namespaceclassitem-appconfig.yaml: NamespaceClassItem with AppConfig spec
- 03-namespaceclass-dev.yaml: NamespaceClass referencing the item
- 04-namespace-test-app.yaml: Test namespace for the configuration

## Running the Test

### Prerequisites
- Kubernetes cluster running
- NamespaceClassOperator deployed
- kubectl configured

### Execute Test
```bash
cd test
./run-test.sh
```

### Verify Results
```bash
./verify-test.sh
```

### Expected Behavior
1. CRD gets installed
2. NamespaceClassItem creates AppConfig resource
3. NamespaceClass aggregates configurations
4. Namespace gets properly configured with AppConfig

### Check Results
After running the test, you should see:
- Custom CRD `appconfigs.example.com` installed
- NamespaceClassItem `appconfig-dev-item` created
- NamespaceClass `development-class` created
- Namespace `test-app-namespace` created
- AppConfig resource auto-created in `test-app-namespace` with the specified spec

## Custom CR Details

The test uses a custom `AppConfig` CRD with the following schema:
- `replicas`: Number of replicas (1-10)
- `image`: Container image to use
- `environment`: Environment type (dev/staging/prod)
- `resources`: CPU and memory limits

The NamespaceClassItem provides a complete spec for this CR:
```yaml
replicas: 2
image: nginx:1.21
environment: dev
resources:
  cpu: "100m"
  memory: "128Mi"
```

**Important**: The `resource` field in NamespaceClassItem should contain the **resource name** ("appconfigs"), not the Kind ("AppConfig"). 

The `spec` field should contain a **complete custom resource definition** including:
- `apiVersion`: Will be overridden by controller
- `kind`: Will be overridden by controller  
- `metadata.name`: Will be preserved
- `metadata.namespace`: Will be overridden by controller
- `spec`: The actual spec content

Example:
```yaml
spec: |
  apiVersion: example.com/v1
  kind: AppConfig
  metadata:
    name: my-app-config
  spec:
    replicas: 2
    image: nginx:1.21
    environment: dev
    resources:
      cpu: "100m"
      memory: "128Mi"
```

⚠️ **Note**: The controller will override `apiVersion`, `kind`, and `metadata.namespace`, but preserve `metadata.name` and other metadata fields.

## Cleanup
```bash
./cleanup-test.sh
```

Or manually:
```bash
kubectl delete -f .
```
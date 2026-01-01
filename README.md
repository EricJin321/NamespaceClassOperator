# NamespaceClassOperator

![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)
![Go Version](https://img.shields.io/badge/Go-1.24-brightgreen)
![Kubernetes](https://img.shields.io/badge/Kubernetes-1.34+-blue)

A Kubernetes Operator for managing namespace-level resource templating and class-based configuration. Built with Kubebuilder v4.10.1.

## Overview

**NamespaceClassOperator** provides a declarative way to manage resources across multiple namespaces using a class-based templating system. It enables you to:

- Define reusable resource templates as **NamespaceClassItems**
- Group templates into **NamespaceClasses** (e.g., development, production)
- Automatically provision resources in namespaces based on their assigned class
- Maintain consistency across multiple namespaces with centralized configuration

### Key Concepts

```
                        Cluster-scoped Resources
┌─────────────────────────────────────────────────────────────────┐
│                                                                   │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │             NamespaceClass (development-class)           │   │
│   │  Items: [appconfig-item, networkpolicy-item, ...]       │   │
│   └────────────────┬────────────────────────────────────────┘   │
│                    │ References                                  │
│       ┌────────────┼────────────┬──────────────┐                │
│       ▼            ▼            ▼              ▼                │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐            │
│  │  Item:  │  │  Item:  │  │  Item:  │  │  Item:  │            │
│  │appconfig│  │network  │  │resource │  │  ...    │            │
│  │  -item  │  │policy-  │  │quota-   │  │         │            │
│  │         │  │  item   │  │  item   │  │         │            │
│  │Template:│  │Template:│  │Template:│  │Template:│            │
│  │AppConfig│  │Network  │  │Resource │  │  ...    │            │
│  │ YAML    │  │Policy   │  │Quota    │  │         │            │
│  └─────────┘  └─────────┘  └─────────┘  └─────────┘            │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
                               │
                               │ Label: namespaceclass.akuity.io/name=development-class
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│              Namespace: my-app (Core Kubernetes)                 │
│                     labels:                                      │
│              namespaceclass.akuity.io/name: development-class    │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           │ Namespace Controller creates
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│    NamespaceState (my-app) - in namespace: my-app               │
│                                                                   │
│    spec:                                                         │
│      namespaceClass: development-class                           │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           │ NamespaceState Controller creates instances
                           │ for each item in the class
                           │
        ┌──────────────────┼──────────────────┬──────────────┐
        ▼                  ▼                  ▼              ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐
│NamespaceItem │  │NamespaceItem │  │NamespaceItem │  │NamespaceItem
│  Instance    │  │  Instance    │  │  Instance    │  │  Instance│
│              │  │              │  │              │  │          │
│appconfig-item│  │networkpolicy │  │resourcequota │  │   ...    │
│in my-app ns  │  │ -item        │  │ -item        │  │          │
└──────┬───────┘  └──────┬───────┘  └──────┬───────┘  └──────┬───┘
       │                 │                 │                 │
       │ NamespaceItemInstance Controller creates actual resources
       │
       ▼                 ▼                 ▼                 ▼
┌──────────────┐  ┌──────────────┐  ┌──────────────┐  ┌──────────┐
│  AppConfig   │  │ NetworkPolicy│  │ ResourceQuota│  │   ...    │
│ my-app-config│  │   dev-policy │  │  dev-quota   │  │          │
│              │  │              │  │              │  │          │
│ (Actual K8s  │  │ (Actual K8s  │  │ (Actual K8s  │  │(Actual   │
│  Resource)   │  │  Resource)   │  │  Resource)   │  │Resource) │
└──────────────┘  └──────────────┘  └──────────────┘  └──────────┘

All resources in namespace: my-app are owned by NamespaceItemInstances
```

## Architecture

### Custom Resource Definitions

#### 1. **NamespaceClassItem** (Cluster-scoped)

Defines a reusable template for any Kubernetes resource (CRDs, ConfigMaps, NetworkPolicies, etc.).

```yaml
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceClassItem
metadata:
  name: appconfig-dev-item
spec:
  group: example.com
  version: v1
  resource: appconfigs
  spec: |
    apiVersion: example.com/v1
    kind: AppConfig
    metadata:
      name: my-app-config
    spec:
      replicas: 2
      environment: development
```

- **Validated on creation** via webhook using dry-run in `namespaceclass-test` namespace
- **Prevents deletion** if referenced by any NamespaceClass
- **Status tracks** which NamespaceClasses reference it

#### 2. **NamespaceClass** (Cluster-scoped)

Groups multiple NamespaceClassItems into a reusable configuration class.

```yaml
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceClass
metadata:
  name: development-class
spec:
  items:
    - appconfig-dev-item
    - networkpolicy-dev-item
    - resourcequota-dev-item
```

- **Validates** that all referenced items exist
- **Prevents name conflicts** between items (same resource name in same namespace)
- **Prevents deletion** if any namespace uses this class
- **Triggers reconciliation** when items list changes

#### 3. **NamespaceState** (Namespace-scoped)

Represents the desired state for a namespace (internal tracking resource).

```yaml
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceState
metadata:
  name: my-app
  namespace: my-app
spec:
  namespaceClass: development-class
```

- **Automatically created** when a namespace is labeled with `namespaceclass.akuity.io/name`
- **Manages lifecycle** of NamespaceItemInstances
- **Supports updates** via pending-update annotation

#### 4. **NamespaceItemInstance** (Namespace-scoped)

Represents an instance of a NamespaceClassItem in a specific namespace.

```yaml
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceItemInstance
metadata:
  name: appconfig-dev-item
  namespace: my-app
spec:
  namespaceClassItem: appconfig-dev-item
status:
  observedGeneration: 1
  currentResourceGVK:
    group: example.com
    version: v1
    kind: AppConfig
```

- **Creates the actual resource** from the template
- **Tracks generation** to detect updates
- **Manages ownership** via OwnerReferences
- **Handles GVK changes** (cleans up old resources)

### Controllers

#### 1. **NamespaceReconciler**
Watches core Kubernetes Namespaces and manages NamespaceState lifecycle.

- **Triggers**: Namespace creation/update/deletion
- **Actions**:
  - Creates NamespaceState when namespace has `namespaceclass.akuity.io/name` label
  - Deletes NamespaceState when label is removed
  - Updates NamespaceState when label value changes

#### 2. **NamespaceClassReconciler**
Manages NamespaceClass resources and triggers cascading updates.

- **Triggers**: NamespaceClass creation/update/deletion
- **Actions**:
  - Adds finalizer for safe deletion
  - Updates all NamespaceStates using this class (via annotations)
  - Maintains ReferencedBy status in NamespaceClassItems
  - Cleans up references on deletion

#### 3. **NamespaceClassItemReconciler**
Manages NamespaceClassItem resources and triggers class updates.

- **Triggers**: NamespaceClassItem creation/update/deletion
- **Actions**:
  - Notifies all referencing NamespaceClasses (via annotations)
  - Maintains bidirectional references
  - Validates no orphaned references exist

#### 4. **NamespaceStateReconciler**
Reconciles NamespaceState to ensure correct NamespaceItemInstances exist.

- **Triggers**: NamespaceState creation/update, annotation changes
- **Actions**:
  - Creates/updates NamespaceItemInstances for items in the class
  - Deletes NamespaceItemInstances for removed items
  - Handles pending class updates
  - Clears reconciliation trigger annotations

#### 5. **NamespaceItemInstanceReconciler**
Creates and manages actual Kubernetes resources from templates.

- **Triggers**: NamespaceItemInstance creation/update
- **Actions**:
  - Fetches NamespaceClassItem template
  - Creates/updates the actual resource (e.g., AppConfig)
  - Sets OwnerReference for garbage collection
  - Tracks generation to detect changes
  - Handles resource GVK changes

### Webhooks

#### 1. **NamespaceClassItem ValidatingWebhook**
Validates NamespaceClassItem specs before admission.

**Validation Rules:**
- **Dry-run validation**: Creates resource in `namespaceclass-test` namespace with DryRun=All
- **Spec parsing**: Ensures YAML can be converted to valid Kubernetes resource
- **Conflict prevention**: Validates no name conflicts with other items in same NamespaceClasses
- **Reference check**: Prevents deletion if any NamespaceClass references this item

**Why `namespaceclass-test` namespace?**
The webhook needs an actual namespace to perform dry-run validation against CRDs. This namespace must exist before deploying the operator.

#### 2. **NamespaceClass ValidatingWebhook**
Validates NamespaceClass specs before admission.

**Validation Rules:**
- **Items exist**: All referenced NamespaceClassItems must exist
- **Non-empty items**: At least one item must be specified
- **No conflicts**: Validates that items don't create resource name conflicts
- **Reference check**: Prevents deletion if any namespace uses this class (checks namespace labels)

## Installation

### Prerequisites

- Kubernetes 1.34+ cluster
- kubectl configured to access your cluster
- cert-manager (for webhook certificates)

### Deploy cert-manager

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.14.0/cert-manager.yaml
```

### Deploy NamespaceClassOperator

1. **Install CRDs and operator:**

```bash
make deploy IMG=<your-registry>/namespaceclass-operator:latest
```

This will:
- Create the `namespaceclass-test` namespace (required for webhook validation)
- Install all CRDs
- Deploy the operator with webhooks
- Set up RBAC

2. **Verify deployment:**

```bash
kubectl get pods -n namespaceclassoperator-system
kubectl get crd | grep policy.akuity.io
```

Expected output:
```
namespaceclassitems.policy.akuity.io
namespaceclasses.policy.akuity.io
namespaceiteminstances.policy.akuity.io
namespacestates.policy.akuity.io
```

## Usage

### Step 1: Create a Custom CRD (Optional)

If you want to template a custom resource, create its CRD first:

```yaml
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: appconfigs.example.com
spec:
  group: example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            spec:
              type: object
              properties:
                replicas:
                  type: integer
                environment:
                  type: string
  scope: Namespaced
  names:
    plural: appconfigs
    singular: appconfig
    kind: AppConfig
```

```bash
kubectl apply -f config/samples/01-crd-appconfig.yaml
```

### Step 2: Create NamespaceClassItems

Define resource templates:

```yaml
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceClassItem
metadata:
  name: appconfig-dev-item
spec:
  group: example.com
  version: v1
  resource: appconfigs
  spec: |
    apiVersion: example.com/v1
    kind: AppConfig
    metadata:
      name: my-app-config
    spec:
      replicas: 2
      environment: development
```

```bash
kubectl apply -f config/samples/02-namespaceclassitem-appconfig.yaml
```

### Step 3: Create a NamespaceClass

Group items into a class:

```yaml
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceClass
metadata:
  name: development-class
spec:
  items:
    - appconfig-dev-item
```

```bash
kubectl apply -f config/samples/03-namespaceclass-dev.yaml
```

### Step 4: Apply Class to Namespace

Label a namespace to apply the class:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: test-app-namespace
  labels:
    namespaceclass.akuity.io/name: development-class
```

```bash
kubectl apply -f config/samples/04-namespace-test-app.yaml
```

### Step 5: Verify Resources

```bash
# Check NamespaceState was created
kubectl get namespacestate -n test-app-namespace

# Check NamespaceItemInstances were created
kubectl get namespaceiteminstance -n test-app-namespace

# Check actual AppConfig resource was created
kubectl get appconfig -n test-app-namespace

# View the created resource
kubectl get appconfig my-app-config -n test-app-namespace -o yaml
```

Expected output:
```yaml
apiVersion: example.com/v1
kind: AppConfig
metadata:
  name: my-app-config
  namespace: test-app-namespace
  ownerReferences:
    - apiVersion: policy.akuity.io/v1alpha
      kind: NamespaceItemInstance
      name: appconfig-dev-item
      uid: ...
spec:
  replicas: 2
  environment: development
```

## Development

### Local Development with K3S

See [LOCAL_DEV_README.md](LOCAL_DEV_README.md) for detailed instructions.

Quick start:
```bash
# Start k3s cluster
docker-compose up -d

# Build and deploy
make docker-build IMG=localhost:5000/namespaceclass-operator:latest
make docker-push IMG=localhost:5000/namespaceclass-operator:latest
./start-local-dev.sh

# Test
kubectl apply -f config/samples/

# Stop
./stop-local-dev.sh
docker-compose down
```

### Run Tests

#### Unit Tests
```bash
make test
```

#### E2E Tests
```bash
make test-e2e
```

The e2e test suite:
- Deploys operator to a Kind cluster
- Verifies webhook CA injection
- Applies sample CRs (01-04 from config/samples/)
- Validates reconciliation and metrics
- Cleans up resources

### Build and Deploy

```bash
# Build manager binary
make build

# Build Docker image
make docker-build IMG=<registry>/namespaceclass-operator:<tag>

# Push Docker image
make docker-push IMG=<registry>/namespaceclass-operator:<tag>

# Deploy to cluster
make deploy IMG=<registry>/namespaceclass-operator:<tag>

# Undeploy from cluster
make undeploy
```

### Code Generation

```bash
# Generate CRD manifests
make manifests

# Generate DeepCopy methods
make generate
```

## RBAC Permissions

The operator requires the following permissions:

### Cluster-wide:
- **NamespaceClass, NamespaceClassItem**: Full CRUD
- **NamespaceState, NamespaceItemInstance**: Full CRUD
- **Namespaces**: Read (for label watching)

### Namespace-scoped:
- **All resources** (`*/*`): Full CRUD in all namespaces
  - Required to create arbitrary resources from templates

**Security Note**: The operator needs broad permissions to create any resource type from templates. Ensure webhook validation is properly configured to prevent privilege escalation.

## Troubleshooting

### Webhook Validation Fails

**Symptom**: NamespaceClassItem creation rejected with "namespace not found: namespaceclass-test"

**Solution**: The `namespaceclass-test` namespace is required for dry-run validation:
```bash
kubectl create namespace namespaceclass-test
```

Or redeploy:
```bash
make undeploy
make deploy IMG=<your-image>
```

### Resources Not Created in Namespace

**Check 1**: Verify namespace has correct label:
```bash
kubectl get namespace <namespace> -o yaml | grep namespaceclass
```

**Check 2**: Verify NamespaceState exists:
```bash
kubectl get namespacestate -n <namespace>
```

**Check 3**: Check NamespaceItemInstance status:
```bash
kubectl get namespaceiteminstance -n <namespace> -o yaml
```

**Check 4**: View controller logs:
```bash
kubectl logs -f deployment/namespaceclassoperator-controller-manager \
  -n namespaceclassoperator-system
```

### NamespaceClass Cannot Be Deleted

**Symptom**: NamespaceClass stuck in "Terminating" state

**Cause**: The class is still referenced by namespace labels

**Solution**: Remove label from all referencing namespaces:
```bash
# Find namespaces using this class
kubectl get namespace -l namespaceclass.akuity.io/name=<class-name>

# Remove label
kubectl label namespace <namespace> namespaceclass.akuity.io/name-
```

### NamespaceClassItem Cannot Be Deleted

**Symptom**: NamespaceClassItem stuck in "Terminating" state

**Cause**: The item is still referenced by a NamespaceClass

**Solution**: Remove item from all NamespaceClass specs:
```bash
# Find classes referencing this item
kubectl get namespaceclass -o yaml | grep <item-name>

# Edit each class and remove the item
kubectl edit namespaceclass <class-name>
```

## Annotations and Labels

### Labels

| Label | Applied To | Purpose |
|-------|-----------|---------|
| `namespaceclass.akuity.io/name` | Namespace | Assigns a NamespaceClass to a namespace |

### Annotations

| Annotation | Applied To | Purpose |
|------------|-----------|---------|
| `namespaceclassitem.akuity.io/updated` | NamespaceItemInstance | Triggers reconciliation when item changes |
| `namespaceclass.akuity.io/class-updated` | NamespaceState | Triggers reconciliation when class changes |
| `namespaceclass.akuity.io/pending-update` | NamespaceState | Marks a pending class update (for future use) |

## Use Cases

### 1. Development/Staging/Production Environments

Define different resource quotas, network policies, and configurations per environment:

```yaml
---
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceClass
metadata:
  name: development-class
spec:
  items:
    - dev-resource-quota
    - dev-network-policy
---
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceClass
metadata:
  name: production-class
spec:
  items:
    - prod-resource-quota
    - prod-network-policy
    - prod-pod-security-policy
```

### 2. Multi-Tenant Clusters

Standardize configurations across tenant namespaces:

```yaml
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceClass
metadata:
  name: tenant-standard
spec:
  items:
    - tenant-network-policy
    - tenant-resource-quota
    - tenant-limit-range
    - tenant-rbac-config
```

### 3. Compliance and Security

Enforce security policies and compliance requirements:

```yaml
apiVersion: policy.akuity.io/v1alpha
kind: NamespaceClass
metadata:
  name: pci-compliant
spec:
  items:
    - pci-network-policy
    - pci-pod-security-standard
    - pci-audit-config
    - pci-encryption-policy
```

## Project Structure

```
.
├── api/v1alpha/              # API definitions
│   ├── namespaceclass_types.go
│   ├── namespaceclassitem_types.go
│   ├── namespacestate_types.go
│   ├── namespaceiteminstance_types.go
│   └── constants.go
├── internal/
│   ├── controller/           # Controllers
│   │   ├── namespace_controller.go
│   │   ├── namespaceclass_controller.go
│   │   ├── namespaceclassitem_controller.go
│   │   ├── namespacestate_controller.go
│   │   └── namespaceiteminstance_controller.go
│   └── webhook/v1alpha/      # Webhooks
│       ├── namespaceclass_webhook.go
│       └── namespaceclassitem_webhook.go
├── config/                   # Kubernetes manifests
│   ├── crd/                  # CRD definitions
│   ├── rbac/                 # RBAC configurations
│   ├── manager/              # Operator deployment
│   ├── webhook/              # Webhook configurations
│   ├── certmanager/          # Certificate configs
│   └── samples/              # Example resources
├── test/
│   ├── e2e/                  # End-to-end tests
│   └── utils/                # Test utilities
├── cmd/main.go               # Operator entrypoint
├── Makefile                  # Build and deploy targets
├── Dockerfile                # Container image
├── docker-compose.yml        # Local dev environment
└── LOCAL_DEV_README.md       # Local dev guide
```

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Standards

- Run `make fmt` to format code
- Run `make vet` to check for common errors
- Run `make test` before submitting PRs
- Add unit tests for new features
- Update documentation as needed

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Resources

- [Kubebuilder Documentation](https://book.kubebuilder.io/)
- [Kubernetes Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubernetes Webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/)

## Contact

- **Author**: Eric Jin
- **Repository**: [github.com/EricJin321/NamespaceClassOperator](https://github.com/EricJin321/NamespaceClassOperator)
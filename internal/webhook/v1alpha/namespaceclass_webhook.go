/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

// nolint:unused
// log is for logging in this package.
var namespaceclasslog = logf.Log.WithName("namespaceclass-resource")

// SetupNamespaceClassWebhookWithManager registers the webhook for NamespaceClass in the manager.
func SetupNamespaceClassWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&policyv1alpha.NamespaceClass{}).
		WithValidator(&NamespaceClassCustomValidator{
			Client: mgr.GetClient(),
		}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-policy-akuity-io-v1alpha-namespaceclass,mutating=false,failurePolicy=fail,sideEffects=None,groups=policy.akuity.io,resources=namespaceclasses,verbs=create;update;delete,versions=v1alpha,name=vnamespaceclass-v1alpha.kb.io,admissionReviewVersions=v1

// NamespaceClassCustomValidator struct is responsible for validating the NamespaceClass resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NamespaceClassCustomValidator struct {
	Client client.Client
}

var _ webhook.CustomValidator = &NamespaceClassCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type NamespaceClass.
func (v *NamespaceClassCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	namespaceclass, ok := obj.(*policyv1alpha.NamespaceClass)
	if !ok {
		return nil, fmt.Errorf("expected a NamespaceClass object but got %T", obj)
	}
	namespaceclasslog.Info("Validation for NamespaceClass upon creation", "name", namespaceclass.GetName())

	// Validate the spec
	if err := v.validateSpec(namespaceclass.Spec); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type NamespaceClass.
func (v *NamespaceClassCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	namespaceclass, ok := newObj.(*policyv1alpha.NamespaceClass)
	if !ok {
		return nil, fmt.Errorf("expected a NamespaceClass object for the newObj but got %T", newObj)
	}
	namespaceclasslog.Info("Validation for NamespaceClass upon update", "name", namespaceclass.GetName())

	// Validate the spec
	if err := v.validateSpec(namespaceclass.Spec); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type NamespaceClass.
func (v *NamespaceClassCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	namespaceclass, ok := obj.(*policyv1alpha.NamespaceClass)
	if !ok {
		return nil, fmt.Errorf("expected a NamespaceClass object but got %T", obj)
	}
	namespaceclasslog.Info("Validation for NamespaceClass upon deletion", "name", namespaceclass.GetName())

	// Query all namespaces to check if any reference this NamespaceClass
	var nsList corev1.NamespaceList
	if err := v.Client.List(ctx, &nsList); err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	for _, ns := range nsList.Items {
		if ns.Labels != nil && ns.Labels[policyv1alpha.LabelNamespaceClassName] == namespaceclass.Name {
			return nil, fmt.Errorf("cannot delete NamespaceClass %s, it is referenced by namespace %s", namespaceclass.Name, ns.Name)
		}
	}

	return nil, nil
}

// validateSpec validates the spec of NamespaceClass
func (v *NamespaceClassCustomValidator) validateSpec(spec policyv1alpha.NamespaceClassSpec) error {
	if len(spec.Items) == 0 {
		return fmt.Errorf("items cannot be empty")
	}

	// Track unique resource identifiers to prevent conflicts
	resourceMap := make(map[string]string) // key: "Group/Version/Resource/Name", value: itemName

	for _, item := range spec.Items {
		// Check if the NamespaceClassItem exists
		var nci policyv1alpha.NamespaceClassItem
		err := v.Client.Get(context.TODO(), client.ObjectKey{Name: item}, &nci)
		if err != nil {
			return fmt.Errorf("namespaceclassitem %s does not exist: %w", item, err)
		}

		// Parse the spec to extract resource information
		var obj map[string]interface{}
		if err := yaml.Unmarshal([]byte(nci.Spec.Spec), &obj); err != nil {
			return fmt.Errorf("failed to parse spec for namespaceclassitem %s: %w", item, err)
		}

		// Extract name from the spec
		name, ok := obj["metadata"].(map[string]interface{})["name"].(string)
		if !ok || name == "" {
			return fmt.Errorf("namespaceclassitem %s spec does not contain a valid name in metadata", item)
		}

		// Create unique key for this resource
		resourceKey := fmt.Sprintf("%s/%s/%s/%s", nci.Spec.Group, nci.Spec.Version, nci.Spec.Resource, name)

		// Check for conflicts
		if existingItem, exists := resourceMap[resourceKey]; exists {
			return fmt.Errorf("resource conflict: namespaceclassitem %s and %s both define the same resource %s", existingItem, item, resourceKey)
		}

		resourceMap[resourceKey] = item
	}

	return nil
}

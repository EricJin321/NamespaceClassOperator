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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/yaml"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

// nolint:unused
// log is for logging in this package.
var namespaceclassitemlog = logf.Log.WithName("namespaceclassitem-resource")

// SetupNamespaceClassItemWebhookWithManager registers the webhook for NamespaceClassItem in the manager.
func SetupNamespaceClassItemWebhookWithManager(mgr ctrl.Manager) error {
	dynamicClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		return err
	}
	return ctrl.NewWebhookManagedBy(mgr).For(&policyv1alpha.NamespaceClassItem{}).
		WithValidator(&NamespaceClassItemCustomValidator{
			Client: dynamicClient,
			Reader: mgr.GetClient(),
		}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: If you want to customise the 'path', use the flags '--defaulting-path' or '--validation-path'.
// +kubebuilder:webhook:path=/validate-policy-akuity-io-v1alpha-namespaceclassitem,mutating=false,failurePolicy=fail,sideEffects=None,groups=policy.akuity.io,resources=namespaceclassitems,verbs=create;update,versions=v1alpha,name=vnamespaceclassitem-v1alpha.kb.io,admissionReviewVersions=v1

// NamespaceClassItemCustomValidator struct is responsible for validating the NamespaceClassItem resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type NamespaceClassItemCustomValidator struct {
	Client dynamic.Interface
	Reader client.Client
}

var _ webhook.CustomValidator = &NamespaceClassItemCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type NamespaceClassItem.
func (v *NamespaceClassItemCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	namespaceclassitem, ok := obj.(*policyv1alpha.NamespaceClassItem)
	if !ok {
		return nil, fmt.Errorf("expected a NamespaceClassItem object but got %T", obj)
	}
	namespaceclassitemlog.Info("Validation for NamespaceClassItem upon creation", "name", namespaceclassitem.GetName())

	// Validate the spec by attempting a dry-run create in namespaceclass-test
	if err := v.validateSpec(namespaceclassitem.Spec); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type NamespaceClassItem.
func (v *NamespaceClassItemCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	namespaceclassitem, ok := newObj.(*policyv1alpha.NamespaceClassItem)
	if !ok {
		return nil, fmt.Errorf("expected a NamespaceClassItem object for the newObj but got %T", newObj)
	}
	namespaceclassitemlog.Info("Validation for NamespaceClassItem upon update", "name", namespaceclassitem.GetName())

	// Validate the spec by attempting a dry-run create in namespaceclass-test
	if err := v.validateSpec(namespaceclassitem.Spec); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type NamespaceClassItem.
func (v *NamespaceClassItemCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	namespaceclassitem, ok := obj.(*policyv1alpha.NamespaceClassItem)
	if !ok {
		return nil, fmt.Errorf("expected a NamespaceClassItem object but got %T", obj)
	}
	namespaceclassitemlog.Info("Validation for NamespaceClassItem upon deletion", "name", namespaceclassitem.GetName())

	// Check if any NamespaceClass references this item
	var ncList policyv1alpha.NamespaceClassList
	if err := v.Reader.List(context.TODO(), &ncList); err != nil {
		return nil, fmt.Errorf("failed to list NamespaceClass: %w", err)
	}
	for _, nc := range ncList.Items {
		for _, item := range nc.Spec.Items {
			if item == namespaceclassitem.GetName() {
				return nil, fmt.Errorf("cannot delete NamespaceClassItem %s: referenced by NamespaceClass %s", namespaceclassitem.GetName(), nc.GetName())
			}
		}
	}

	return nil, nil
}

// validateSpec validates the spec by attempting a dry-run create in the namespaceclass-test namespace
func (v *NamespaceClassItemCustomValidator) validateSpec(spec policyv1alpha.NamespaceClassItemSpec) error {
	gvr := schema.GroupVersionResource{
		Group:    spec.Group,
		Version:  spec.Version,
		Resource: spec.Resource,
	}

	obj := &unstructured.Unstructured{}
	if err := yaml.Unmarshal([]byte(spec.Spec), obj); err != nil {
		return fmt.Errorf("failed to unmarshal spec: %w", err)
	}

	// Attempt dry-run create
	_, err := v.Client.Resource(gvr).Namespace("namespaceclass-test").Create(context.TODO(), obj, metav1.CreateOptions{DryRun: []string{metav1.DryRunAll}})
	if err != nil {
		return fmt.Errorf("dry-run create failed for GVR %v: %w", gvr, err)
	}

	return nil
}

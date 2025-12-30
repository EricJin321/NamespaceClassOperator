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

package controller

import (
	"context"

	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

// NamespaceItemInstanceReconciler reconciles a NamespaceItemInstance object
type NamespaceItemInstanceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceiteminstances,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceiteminstances/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceiteminstances/finalizers,verbs=update
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclassitems,verbs=get;list;watch
// +kubebuilder:rbac:groups=*,resources=*,verbs=get;list;watch;create;update;patch;delete,namespace=*

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceItemInstance object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *NamespaceItemInstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the NamespaceItemInstance instance
	var nii policyv1alpha.NamespaceItemInstance
	if err := r.Get(ctx, req.NamespacedName, &nii); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// NamespaceItemInstance not found, ignore
		return ctrl.Result{}, nil
	}

	// Skip reconciliation in system namespaces
	if nii.Namespace == "kube-system" || nii.Namespace == "kube-public" || nii.Namespace == "kube-node-lease" {
		log.Info("Skipping reconciliation in system namespace", "namespace", nii.Namespace)
		return ctrl.Result{}, nil
	}

	// Get the NamespaceClassItem
	var nci policyv1alpha.NamespaceClassItem
	if err := r.Get(ctx, client.ObjectKey{Name: nii.Spec.NamespaceClassItem}, &nci); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		log.Error(err, "NamespaceClassItem not found", "name", nii.Spec.NamespaceClassItem)
		return ctrl.Result{}, nil
	}

	currentGen := nci.Generation

	// Check if update is needed
	if nii.Status.ObservedGeneration != currentGen {
		// Unmarshal the spec into unstructured
		var obj unstructured.Unstructured
		if err := yaml.Unmarshal([]byte(nci.Spec.Spec), &obj); err != nil {
			log.Error(err, "Failed to unmarshal NamespaceClassItem spec")
			return ctrl.Result{}, err
		}

		// Set GVK from NamespaceClassItem
		obj.SetGroupVersionKind(schema.GroupVersionKind{
			Group:   nci.Spec.Group,
			Version: nci.Spec.Version,
			Kind:    nci.Spec.Resource,
		})

		// Set namespace and owner
		obj.SetNamespace(nii.Namespace)
		obj.SetOwnerReferences([]metav1.OwnerReference{
			{
				APIVersion: nii.APIVersion,
				Kind:       nii.Kind,
				Name:       nii.Name,
				UID:        nii.UID,
			},
		})

		// Check if the resource exists
		existing := &unstructured.Unstructured{}
		existing.SetGroupVersionKind(obj.GroupVersionKind())
		err := r.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, existing)
		if err != nil {
			if client.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, err
			}
			// Create the resource
			if err := r.Create(ctx, &obj); err != nil {
				log.Error(err, "Failed to create resource")
				return ctrl.Result{}, err
			}
		} else {
			// Update the resource
			obj.SetResourceVersion(existing.GetResourceVersion())
			if err := r.Update(ctx, &obj); err != nil {
				log.Error(err, "Failed to update resource")
				return ctrl.Result{}, err
			}
		}

		// Update status
		nii.Status.ObservedGeneration = currentGen
		if err := r.Status().Update(ctx, &nii); err != nil {
			log.Error(err, "Failed to update status")
			return ctrl.Result{}, err
		}
	}

	// Remove the updated annotation if present
	if nii.Annotations != nil && nii.Annotations[policyv1alpha.AnnotationNamespaceClassItemUpdated] != "" {
		delete(nii.Annotations, policyv1alpha.AnnotationNamespaceClassItemUpdated)
		if err := r.Update(ctx, &nii); err != nil {
			log.Error(err, "Failed to remove updated annotation")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceItemInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policyv1alpha.NamespaceItemInstance{}).
		Named("namespaceiteminstance").
		Complete(r)
}

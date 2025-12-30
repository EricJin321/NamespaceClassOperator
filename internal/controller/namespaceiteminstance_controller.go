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
			log.Error(err, "Failed to get NamespaceItemInstance", "namespaceItemInstance", req.NamespacedName)
			return ctrl.Result{}, err
		}
		// NamespaceItemInstance not found, ignore
		log.Info("NamespaceItemInstance not found, ignoring", "namespaceItemInstance", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling NamespaceItemInstance", "component", "controller", "controller", "namespaceiteminstance", "name", nii.Name, "namespace", nii.Namespace, "spec", nii.Spec, "status", nii.Status)

	// Skip reconciliation in system namespaces
	if nii.Namespace == "kube-system" || nii.Namespace == "kube-public" || nii.Namespace == "kube-node-lease" {
		log.Info("Skipping reconciliation in system namespace", "namespace", nii.Namespace, "name", nii.Name)
		return ctrl.Result{}, nil
	}

	log.Info("Processing NamespaceItemInstance", "name", nii.Name, "namespace", nii.Namespace, "namespaceClassItem", nii.Spec.NamespaceClassItem)

	// Get the NamespaceClassItem
	var nci policyv1alpha.NamespaceClassItem
	if err := r.Get(ctx, client.ObjectKey{Name: nii.Spec.NamespaceClassItem}, &nci); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get NamespaceClassItem", "namespaceItemInstance", nii.Name, "namespaceClassItem", nii.Spec.NamespaceClassItem)
			return ctrl.Result{}, err
		}
		log.Error(err, "NamespaceClassItem not found", "namespaceItemInstance", nii.Name, "namespaceClassItem", nii.Spec.NamespaceClassItem)
		return ctrl.Result{}, nil
	}

	log.Info("Found NamespaceClassItem", "namespaceItemInstance", nii.Name, "namespaceClassItem", nci.Name, "spec", nci.Spec)

	currentGen := nci.Generation
	log.Info("Checking generation", "namespaceItemInstance", nii.Name, "currentGen", currentGen, "observedGen", nii.Status.ObservedGeneration)

	// Check if update is needed
	if nii.Status.ObservedGeneration != currentGen {
		log.Info("Generation mismatch, updating resource", "namespaceItemInstance", nii.Name, "from", nii.Status.ObservedGeneration, "to", currentGen)

		log.Info("Spec to unmarshal", "spec", nci.Spec.Spec)

		// Unmarshal the spec into unstructured
		var obj unstructured.Unstructured
		jsonData, err := yaml.YAMLToJSON([]byte(nci.Spec.Spec))
		if err != nil {
			log.Error(err, "Failed to convert YAML to JSON", "namespaceItemInstance", nii.Name, "namespaceClassItem", nci.Name)
			return ctrl.Result{}, err
		}
		log.Info("Converted JSON", "component", "controller", "controller", "namespaceiteminstance", "namespaceItemInstance", nii.Name, "namespaceClassItem", nci.Name, "json", string(jsonData))
		_, _, err = unstructured.UnstructuredJSONScheme.Decode(jsonData, nil, &obj)
		if err != nil {
			log.Error(err, "Failed to decode JSON into unstructured", "namespaceItemInstance", nii.Name, "namespaceClassItem", nci.Name)
			return ctrl.Result{}, err
		}

		log.Info("Unmarshaled resource spec", "namespaceItemInstance", nii.Name, "resource", obj)

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

		log.Info("Prepared resource for creation/update", "component", "controller", "controller", "namespaceiteminstance", "namespaceItemInstance", nii.Name, "resourceName", obj.GetName(), "resourceNamespace", obj.GetNamespace(), "gvk", obj.GroupVersionKind())

		// Check if the resource exists
		existing := &unstructured.Unstructured{}
		existing.SetGroupVersionKind(obj.GroupVersionKind())
		err := r.Get(ctx, client.ObjectKey{Name: obj.GetName(), Namespace: obj.GetNamespace()}, existing)
		if err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Failed to check if resource exists", "namespaceItemInstance", nii.Name, "resource", obj.GetName())
				return ctrl.Result{}, err
			}
			// Create the resource
			log.Info("Creating new resource", "namespaceItemInstance", nii.Name, "resource", obj.GetName())
			if err := r.Create(ctx, &obj); err != nil {
				log.Error(err, "Failed to create resource", "namespaceItemInstance", nii.Name, "resource", obj.GetName())
				return ctrl.Result{}, err
			}
			log.Info("Successfully created resource", "namespaceItemInstance", nii.Name, "resource", obj.GetName())
		} else {
			// Update the resource
			log.Info("Updating existing resource", "namespaceItemInstance", nii.Name, "resource", obj.GetName())
			obj.SetResourceVersion(existing.GetResourceVersion())
			if err := r.Update(ctx, &obj); err != nil {
				log.Error(err, "Failed to update resource", "namespaceItemInstance", nii.Name, "resource", obj.GetName())
				return ctrl.Result{}, err
			}
			log.Info("Successfully updated resource", "namespaceItemInstance", nii.Name, "resource", obj.GetName())
		}

		// Update status
		log.Info("Updating NamespaceItemInstance status", "name", nii.Name, "observedGeneration", currentGen)
		nii.Status.ObservedGeneration = currentGen
		if err := r.Status().Update(ctx, &nii); err != nil {
			log.Error(err, "Failed to update status", "namespaceItemInstance", nii.Name)
			return ctrl.Result{}, err
		}
	} else {
		log.Info("Generation matches, no update needed", "namespaceItemInstance", nii.Name, "generation", currentGen)
	}

	// Remove the updated annotation if present
	if nii.Annotations != nil && nii.Annotations[policyv1alpha.AnnotationNamespaceClassItemUpdated] != "" {
		log.Info("Removing updated annotation from NamespaceItemInstance", "name", nii.Name)
		delete(nii.Annotations, policyv1alpha.AnnotationNamespaceClassItemUpdated)
		if err := r.Update(ctx, &nii); err != nil {
			log.Error(err, "Failed to remove updated annotation", "namespaceItemInstance", nii.Name)
			return ctrl.Result{}, err
		}
	}

	log.Info("NamespaceItemInstance reconciliation completed successfully", "name", nii.Name, "namespace", nii.Namespace)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceItemInstanceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policyv1alpha.NamespaceItemInstance{}).
		Named("namespaceiteminstance").
		Complete(r)
}

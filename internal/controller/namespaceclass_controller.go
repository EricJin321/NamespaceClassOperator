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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

// NamespaceClassReconciler reconciles a NamespaceClass object
type NamespaceClassReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclasses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclasses/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclasses/finalizers,verbs=update
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclassitems/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespacestates,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceClass object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *NamespaceClassReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// Fetch the NamespaceClass instance
	var nc policyv1alpha.NamespaceClass
	if err := r.Get(ctx, req.NamespacedName, &nc); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get NamespaceClass", "namespaceClass", req.NamespacedName)
			return ctrl.Result{}, err
		}
		// NamespaceClass not found, ignore
		log.Info("NamespaceClass not found, ignoring", "namespaceClass", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling NamespaceClass", "component", "controller", "controller", "namespaceclass", "name", nc.Name, "spec", nc.Spec, "status", nc.Status)

	// Handle deletion
	if !nc.DeletionTimestamp.IsZero() {
		log.Info("Handling NamespaceClass deletion", "name", nc.Name)
		return r.handleDeletion(ctx, &nc)
	}

	log.Info("Processing NamespaceClass reconciliation", "component", "controller", "controller", "namespaceclass", "name", nc.Name, "items", nc.Spec.Items)

	// Add finalizer if not present
	if !containsString(nc.Finalizers, policyv1alpha.FinalizerNamespaceClass) {
		log.Info("Adding finalizer to NamespaceClass", "name", nc.Name, "finalizer", policyv1alpha.FinalizerNamespaceClass)
		nc.Finalizers = append(nc.Finalizers, policyv1alpha.FinalizerNamespaceClass)
		if err := r.Update(ctx, &nc); err != nil {
			log.Error(err, "Failed to add finalizer to NamespaceClass", "name", nc.Name)
			return ctrl.Result{}, err
		}
	}

	// List all NamespaceState that reference this class
	var nsList policyv1alpha.NamespaceStateList
	if err := r.List(ctx, &nsList, client.MatchingFields{"spec.namespaceClass": nc.Name}); err != nil {
		log.Error(err, "Failed to list NamespaceStates for class", "class", nc.Name)
		return ctrl.Result{}, err
	}

	log.Info("Found NamespaceStates referencing this class", "class", nc.Name, "count", len(nsList.Items))

	// Add annotation to trigger reconcile for all matching NamespaceStates
	for _, nsState := range nsList.Items {
		if _, hasAnnotation := nsState.Annotations[policyv1alpha.AnnotationNamespaceClassUpdated]; !hasAnnotation {
			log.Info("Adding update annotation to NamespaceState", "class", nc.Name, "namespaceState", nsState.Name)
			// Add annotation to trigger NamespaceState reconcile
			if nsState.Annotations == nil {
				nsState.Annotations = make(map[string]string)
			}
			nsState.Annotations[policyv1alpha.AnnotationNamespaceClassUpdated] = policyv1alpha.AnnotationValueTrue
			if err := r.Update(ctx, &nsState); err != nil {
				log.Error(err, "Failed to add update annotation to NamespaceState", "class", nc.Name, "namespaceState", nsState.Name)
				return ctrl.Result{}, err
			}
		}
	}

	// Update ReferencedBy in NamespaceClassItems
	if err := r.updateNamespaceClassItemReferences(ctx, nc); err != nil {
		log.Error(err, "Failed to update NamespaceClassItem references", "class", nc.Name)
		return ctrl.Result{}, err
	}

	// Check if it has the trigger annotation and remove it
	if _, hasAnnotation := nc.Annotations[policyv1alpha.AnnotationNamespaceClassItemUpdated]; hasAnnotation {
		log.Info("Removing trigger annotation from NamespaceClass", "name", nc.Name)
		delete(nc.Annotations, policyv1alpha.AnnotationNamespaceClassItemUpdated)
		if err := r.Update(ctx, &nc); err != nil {
			log.Error(err, "Failed to remove trigger annotation from NamespaceClass", "name", nc.Name)
			return ctrl.Result{}, err
		}
	}

	log.Info("NamespaceClass reconciliation completed successfully", "name", nc.Name)
	return ctrl.Result{}, nil
}

// handleDeletion handles NamespaceClass deletion, cleaning up references
func (r *NamespaceClassReconciler) handleDeletion(ctx context.Context, nc *policyv1alpha.NamespaceClass) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Starting NamespaceClass deletion cleanup", "name", nc.Name)

	// Clean up references in NamespaceClassItems
	for _, itemName := range nc.Spec.Items {
		log.Info("Cleaning up reference in NamespaceClassItem", "class", nc.Name, "item", itemName)
		var nci policyv1alpha.NamespaceClassItem
		if err := r.Get(ctx, client.ObjectKey{Name: itemName}, &nci); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Failed to get NamespaceClassItem during deletion cleanup", "class", nc.Name, "item", itemName)
				return ctrl.Result{}, err
			}
			continue
		}

		// Remove this class from ReferencedBy
		var newRefs []string
		for _, ref := range nci.Status.ReferencedBy {
			if ref != nc.Name {
				newRefs = append(newRefs, ref)
			}
		}
		nci.Status.ReferencedBy = newRefs
		if err := r.Status().Update(ctx, &nci); err != nil {
			log.Error(err, "Failed to update NamespaceClassItem status during deletion cleanup", "class", nc.Name, "item", itemName)
			return ctrl.Result{}, err
		}
		log.Info("Successfully updated NamespaceClassItem reference", "class", nc.Name, "item", itemName)
	}

	// Remove finalizer
	log.Info("Removing finalizer from NamespaceClass", "name", nc.Name, "finalizer", policyv1alpha.FinalizerNamespaceClass)
	nc.Finalizers = removeString(nc.Finalizers, policyv1alpha.FinalizerNamespaceClass)
	if err := r.Update(ctx, nc); err != nil {
		log.Error(err, "Failed to remove finalizer from NamespaceClass", "name", nc.Name)
		return ctrl.Result{}, err
	}

	log.Info("NamespaceClass deletion cleanup completed", "name", nc.Name)
	return ctrl.Result{}, nil
}

// containsString checks if a string is in a slice
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// removeString removes a string from a slice
func removeString(slice []string, s string) []string {
	var result []string
	for _, item := range slice {
		if item != s {
			result = append(result, item)
		}
	}
	return result
}

// updateNamespaceClassItemReferences maintains the ReferencedBy field in NamespaceClassItems
func (r *NamespaceClassReconciler) updateNamespaceClassItemReferences(ctx context.Context, nc policyv1alpha.NamespaceClass) error {
	log := logf.FromContext(ctx)
	log.Info("Updating NamespaceClassItem references", "class", nc.Name, "items", nc.Spec.Items)

	// For each item in the class, add this class to its ReferencedBy
	for _, itemName := range nc.Spec.Items {
		log.Info("Processing NamespaceClassItem reference", "class", nc.Name, "item", itemName)
		var nci policyv1alpha.NamespaceClassItem
		if err := r.Get(ctx, client.ObjectKey{Name: itemName}, &nci); err != nil {
			if client.IgnoreNotFound(err) != nil {
				log.Error(err, "Failed to get NamespaceClassItem for reference update", "class", nc.Name, "item", itemName)
				return err
			}
			continue // Item doesn't exist, skip
		}

		// Check if already referenced
		alreadyReferenced := false
		for _, ref := range nci.Status.ReferencedBy {
			if ref == nc.Name {
				alreadyReferenced = true
				break
			}
		}

		if !alreadyReferenced {
			log.Info("Adding reference to NamespaceClassItem", "class", nc.Name, "item", itemName)
			// Add reference
			nci.Status.ReferencedBy = append(nci.Status.ReferencedBy, nc.Name)
			if err := r.Status().Update(ctx, &nci); err != nil {
				log.Error(err, "Failed to update NamespaceClassItem status with reference", "class", nc.Name, "item", itemName)
				return err
			}
		} else {
			log.Info("Reference already exists in NamespaceClassItem. No Update", "class", nc.Name, "item", itemName)
		}
	}

	log.Info("NamespaceClassItem references update completed", "class", nc.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policyv1alpha.NamespaceClass{}).
		Named("namespaceclass").
		Complete(r)
}

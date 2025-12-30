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

// NamespaceClassItemReconciler reconciles a NamespaceClassItem object
type NamespaceClassItemReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclassitems,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclassitems/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclassitems/finalizers,verbs=update
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclasses,verbs=get;list;watch;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceClassItem object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *NamespaceClassItemReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// Fetch the NamespaceClassItem instance
	var nci policyv1alpha.NamespaceClassItem
	if err := r.Get(ctx, req.NamespacedName, &nci); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// NamespaceClassItem not found, ignore
		return ctrl.Result{}, nil
	}

	// Check if spec has changed by comparing generation
	specChanged := nci.Generation != nci.Status.ObservedGeneration

	// If spec hasn't changed, nothing to do
	if !specChanged {
		return ctrl.Result{}, nil
	}

	// Update observed generation in status
	nci.Status.ObservedGeneration = nci.Generation
	if err := r.Status().Update(ctx, &nci); err != nil {
		return ctrl.Result{}, err
	}
	// Re-fetch after status update
	if err := r.Get(ctx, req.NamespacedName, &nci); err != nil {
		return ctrl.Result{}, err
	}

	// Trigger downstream reconcile since spec changed
	// Use ReferencedBy from status to find referencing NamespaceClasses
	referencingClasses := nci.Status.ReferencedBy

	// Add annotation to trigger reconcile for each referencing NamespaceClass
	for _, className := range referencingClasses {
		var nc policyv1alpha.NamespaceClass
		if err := r.Get(ctx, client.ObjectKey{Name: className}, &nc); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return ctrl.Result{}, err
			}
			continue // Class doesn't exist, skip
		}

		// Add annotation to trigger reconcile if not already present
		if nc.Annotations == nil {
			nc.Annotations = make(map[string]string)
		}
		if _, hasAnnotation := nc.Annotations[policyv1alpha.AnnotationNamespaceClassItemUpdated]; !hasAnnotation {
			nc.Annotations[policyv1alpha.AnnotationNamespaceClassItemUpdated] = "true"
			if err := r.Update(ctx, &nc); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceClassItemReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policyv1alpha.NamespaceClassItem{}).
		Named("namespaceclassitem").
		Complete(r)
}

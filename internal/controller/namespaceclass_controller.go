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
	_ = logf.FromContext(ctx)

	// Fetch the NamespaceClass instance
	var nc policyv1alpha.NamespaceClass
	if err := r.Get(ctx, req.NamespacedName, &nc); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// NamespaceClass not found, ignore
		return ctrl.Result{}, nil
	}

	// List all NamespaceState and ensure those that reference this class have the annotation
	var nsList policyv1alpha.NamespaceStateList
	if err := r.List(ctx, &nsList); err != nil {
		return ctrl.Result{}, err
	}

	for _, nsState := range nsList.Items {
		if nsState.Spec.NamespaceClass == nc.Name {
			if _, hasAnnotation := nsState.Annotations["namespaceclass.akuity.io/class-updated"]; !hasAnnotation {
				// Add annotation to trigger NamespaceState reconcile
				if nsState.Annotations == nil {
					nsState.Annotations = make(map[string]string)
				}
				nsState.Annotations["namespaceclass.akuity.io/class-updated"] = "true"
				if err := r.Update(ctx, &nsState); err != nil {
					return ctrl.Result{}, err
				}
			}
		}
	}

	// Check if it has the trigger annotation and remove it
	if _, hasAnnotation := nc.Annotations["namespaceclassitem.akuity.io/updated"]; hasAnnotation {
		delete(nc.Annotations, "namespaceclassitem.akuity.io/updated")
		if err := r.Update(ctx, &nc); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceClassReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policyv1alpha.NamespaceClass{}).
		Named("namespaceclass").
		Complete(r)
}

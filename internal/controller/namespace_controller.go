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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespacestates,verbs=get;list;watch;create;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Namespace object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// Fetch the Namespace instance
	var ns corev1.Namespace
	if err := r.Get(ctx, req.NamespacedName, &ns); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// Namespace not found, ignore
		return ctrl.Result{}, nil
	}

	// Check if the namespace has the label namespaceclass.akuity.io/name
	labelKey := "namespaceclass.akuity.io/name"
	className, exists := ns.Labels[labelKey]
	if !exists {
		// No label, check if NamespaceState exists and delete it
		var nsState policyv1alpha.NamespaceState
		err := r.Get(ctx, client.ObjectKey{Name: ns.Name}, &nsState)
		if err == nil {
			// Exists, delete it
			if err := r.Delete(ctx, &nsState); err != nil {
				return ctrl.Result{}, err
			}
		} else if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// No label and no state, nothing to do
		return ctrl.Result{}, nil
	}

	// Check if NamespaceState exists
	var nsState policyv1alpha.NamespaceState
	err := r.Get(ctx, client.ObjectKey{Name: ns.Name}, &nsState)
	if err != nil && client.IgnoreNotFound(err) != nil {
		return ctrl.Result{}, err
	}

	if client.IgnoreNotFound(err) == nil {
		// NamespaceState exists, check if it matches
		if nsState.Spec.NamespaceClass != className {
			// Mismatch, add annotation for pending update
			if nsState.Annotations == nil {
				nsState.Annotations = make(map[string]string)
			}
			nsState.Annotations["namespaceclass.akuity.io/pending-update"] = className
			if err := r.Update(ctx, &nsState); err != nil {
				return ctrl.Result{}, err
			}
		}
	} else {
		// NamespaceState does not exist, create it
		nsState = policyv1alpha.NamespaceState{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: ns.APIVersion,
						Kind:       ns.Kind,
						Name:       ns.Name,
						UID:        ns.UID,
					},
				},
			},
			Spec: policyv1alpha.NamespaceStateSpec{
				NamespaceClass: className,
			},
		}
		if err := r.Create(ctx, &nsState); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Named("namespace").
		Complete(r)
}

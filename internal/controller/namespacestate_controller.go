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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

// NamespaceStateReconciler reconciles a NamespaceState object
type NamespaceStateReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespacestates,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespacestates/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespacestates/finalizers,verbs=update
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceclassitems,verbs=get;list;watch
// +kubebuilder:rbac:groups=policy.akuity.io,resources=namespaceiteminstances,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NamespaceState object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *NamespaceStateReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// Fetch the NamespaceState instance
	var nsState policyv1alpha.NamespaceState
	if err := r.Get(ctx, req.NamespacedName, &nsState); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}
		// NamespaceState not found, ignore
		return ctrl.Result{}, nil
	}

	// Check if the spec class exists
	var specClassExists bool
	if nsState.Spec.NamespaceClass != "" {
		var specNc policyv1alpha.NamespaceClass
		err := r.Get(ctx, client.ObjectKey{Name: nsState.Spec.NamespaceClass}, &specNc)
		specClassExists = err == nil
	} else {
		specClassExists = false
	}

	// Check if pending update is set
	pending, hasPending := nsState.Annotations["namespaceclass.akuity.io/pending-update"]

	// If spec class does not exist or has pending update, delete all owned NamespaceItemInstances
	if !specClassExists || hasPending {
		var niiList policyv1alpha.NamespaceItemInstanceList
		if err := r.List(ctx, &niiList, client.InNamespace(nsState.Name)); err != nil {
			return ctrl.Result{}, err
		}
		for _, nii := range niiList.Items {
			if err := r.Delete(ctx, &nii); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Now determine target class name
	var targetClassName string
	if hasPending {
		targetClassName = pending
	} else {
		targetClassName = nsState.Spec.NamespaceClass
	}

	// Check if target class exists
	var nc policyv1alpha.NamespaceClass
	err := r.Get(ctx, client.ObjectKey{Name: targetClassName}, &nc)
	if err != nil {
		// Target class does not exist, nothing more to do
		return ctrl.Result{}, nil
	}

	// Now reconcile to the target class
	desiredItems := nc.Spec.Items

	// List existing NamespaceItemInstances in the namespace
	var existingNIIList policyv1alpha.NamespaceItemInstanceList
	if err := r.List(ctx, &existingNIIList, client.InNamespace(nsState.Name)); err != nil {
		return ctrl.Result{}, err
	}

	existingMap := make(map[string]policyv1alpha.NamespaceItemInstance)
	for _, nii := range existingNIIList.Items {
		existingMap[nii.Spec.NamespaceClassItem] = nii
	}

	// Create or update for desired items
	for _, item := range desiredItems {
		if nii, exists := existingMap[item]; exists {
			// Exists, add annotation to trigger reconcile
			if nii.Annotations == nil {
				nii.Annotations = make(map[string]string)
			}
			nii.Annotations["namespaceclassitem.akuity.io/updated"] = "true"
			if err := r.Update(ctx, &nii); err != nil {
				return ctrl.Result{}, err
			}
			delete(existingMap, item)
		} else {
			// Create new
			nii := policyv1alpha.NamespaceItemInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      item, // or generate unique name
					Namespace: nsState.Name,
					OwnerReferences: []metav1.OwnerReference{
						{
							APIVersion: nsState.APIVersion,
							Kind:       nsState.Kind,
							Name:       nsState.Name,
							UID:        nsState.UID,
						},
					},
				},
				Spec: policyv1alpha.NamespaceItemInstanceSpec{
					NamespaceClassItem: item,
				},
			}
			if err := r.Create(ctx, &nii); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Delete extra ones
	for _, nii := range existingMap {
		if err := r.Delete(ctx, &nii); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Update spec and remove annotations only if necessary
	needUpdate := hasPending || (nsState.Annotations != nil && nsState.Annotations["namespaceclassitem.akuity.io/updated"] != "") || nsState.Spec.NamespaceClass != targetClassName
	if needUpdate {
		nsState.Spec.NamespaceClass = targetClassName
		delete(nsState.Annotations, "namespaceclass.akuity.io/pending-update")
		delete(nsState.Annotations, "namespaceclassitem.akuity.io/updated")
		if err := r.Update(ctx, &nsState); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceStateReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&policyv1alpha.NamespaceState{}).
		Named("namespacestate").
		Complete(r)
}

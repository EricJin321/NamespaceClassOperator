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
	log := logf.FromContext(ctx)

	// Fetch the Namespace instance
	var ns corev1.Namespace
	if err := r.Get(ctx, req.NamespacedName, &ns); err != nil {
		if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get Namespace", "namespace", req.NamespacedName)
			return ctrl.Result{}, err
		}
		// Namespace not found, ignore
		log.Info("Namespace not found, ignoring", "namespace", req.NamespacedName)
		return ctrl.Result{}, nil
	}

	log.Info("Reconciling Namespace", "component", "controller", "controller", "namespace", "name", ns.Name, "labels", ns.Labels, "spec", ns.Spec)

	// Check if the namespace has the label namespaceclass.akuity.io/name
	labelKey := policyv1alpha.LabelNamespaceClassName
	className, exists := ns.Labels[labelKey]
	if !exists {
		log.Info("Namespace does not have required label, checking for existing NamespaceState", "namespace", ns.Name, "labelKey", labelKey)
		// No label, check if NamespaceState exists and delete it
		var nsState policyv1alpha.NamespaceState
		err := r.Get(ctx, client.ObjectKey{Name: ns.Name}, &nsState)
		if err == nil {
			log.Info("Deleting existing NamespaceState for namespace without label", "namespace", ns.Name, "namespaceState", nsState.Name)
			// Exists, delete it
			if err := r.Delete(ctx, &nsState); err != nil {
				log.Error(err, "Failed to delete NamespaceState", "namespace", ns.Name, "namespaceState", nsState.Name)
				return ctrl.Result{}, err
			}
		} else if client.IgnoreNotFound(err) != nil {
			log.Error(err, "Failed to get NamespaceState for deletion", "namespace", ns.Name)
			return ctrl.Result{}, err
		}
		// No label and no state, nothing to do
		log.Info("No label and no existing NamespaceState, reconciliation complete", "namespace", ns.Name)
		return ctrl.Result{}, nil
	}

	log.Info("Namespace has required label", "namespace", ns.Name, "className", className)

	// Check if NamespaceState exists
	var nsState policyv1alpha.NamespaceState
	err := r.Get(ctx, client.ObjectKey{Name: ns.Name}, &nsState)
	if client.IgnoreNotFound(err) != nil {
		log.Error(err, "Failed to get NamespaceState", "namespace", ns.Name)
		return ctrl.Result{}, err
	}

	log.Info("Get NamespaceState result", "namespace", ns.Name, "err", err, "ignoreNotFound", client.IgnoreNotFound(err), "nsState", nsState)
	if err == nil {
		// NamespaceState exists, check if it matches
		if nsState.Spec.NamespaceClass != className {
			log.Info("NamespaceState class mismatch, updating", "namespace", ns.Name, "from", nsState.Spec.NamespaceClass, "to", className)
			// Mismatch, add annotation for pending update
			if nsState.Annotations == nil {
				nsState.Annotations = make(map[string]string)
			}
			nsState.Annotations[policyv1alpha.AnnotationNamespaceClassPendingUpdate] = className
			if err := r.Update(ctx, &nsState); err != nil {
				log.Error(err, "Failed to update NamespaceState with pending update annotation", "namespace", ns.Name, "namespaceState", nsState.Name)
				return ctrl.Result{}, err
			}
		} else {
			log.Info("NamespaceState class matches, no action needed", "namespace", ns.Name, "class", className)
		}
	} else {
		log.Info("NamespaceState does not exist, creating new one", "namespace", ns.Name, "class", className)
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
		log.Info("Creating NamespaceState", "namespace", ns.Name, "spec", nsState.Spec)
		if err := r.Create(ctx, &nsState); err != nil {
			log.Error(err, "Failed to create NamespaceState", "namespace", ns.Name, "spec", nsState.Spec)
			return ctrl.Result{}, err
		}
		log.Info("Successfully created NamespaceState", "namespace", ns.Name, "namespaceState", nsState.Name)
	}

	log.Info("Namespace reconciliation completed successfully", "namespace", ns.Name)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Named("namespace").
		Complete(r)
}

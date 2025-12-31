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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

var _ = Describe("Namespace Controller", func() {
	Context("When reconciling a resource", func() {
		const namespaceName = "test-namespace"
		const namespaceKind = "Namespace"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name: namespaceName,
		}
		namespace := &corev1.Namespace{}

		BeforeEach(func() {
			By("creating the test namespace")
			err := k8sClient.Get(ctx, typeNamespacedName, namespace)
			if err != nil && errors.IsNotFound(err) {
				resource := &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: namespaceName,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// Cleanup logic after each test
			resource := &corev1.Namespace{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				By("Cleanup the test namespace")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}

			// Cleanup NamespaceState
			nsState := &policyv1alpha.NamespaceState{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, nsState)
			if err == nil {
				Expect(k8sClient.Delete(ctx, nsState)).To(Succeed())
			}
		})

		It("should create NamespaceState when namespace has the required label", func() {
			By("Adding the required label to the namespace")
			ns := &corev1.Namespace{}
			err := k8sClient.Get(ctx, typeNamespacedName, ns)
			Expect(err).NotTo(HaveOccurred())
			// Ensure APIVersion and Kind are set for testing
			ns.APIVersion = "v1"
			ns.Kind = namespaceKind
			if ns.Labels == nil {
				ns.Labels = make(map[string]string)
			}
			ns.Labels[policyv1alpha.LabelNamespaceClassName] = "test-class"
			Expect(k8sClient.Update(ctx, ns)).To(Succeed())

			By("Reconciling the namespace")
			controllerReconciler := &NamespaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that NamespaceState was created")
			nsState := &policyv1alpha.NamespaceState{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, nsState)
			Expect(err).NotTo(HaveOccurred())
			Expect(nsState.Spec.NamespaceClass).To(Equal("test-class"))
			Expect(nsState.OwnerReferences).To(HaveLen(1))
			Expect(nsState.OwnerReferences[0].Name).To(Equal(namespaceName))
		})

		It("should delete existing NamespaceState when namespace loses the label", func() {
			By("Creating a NamespaceState first")
			nsState := &policyv1alpha.NamespaceState{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
				Spec: policyv1alpha.NamespaceStateSpec{
					NamespaceClass: "old-class",
				},
			}
			Expect(k8sClient.Create(ctx, nsState)).To(Succeed())

			By("Ensuring namespace has no label")
			ns := &corev1.Namespace{}
			err := k8sClient.Get(ctx, typeNamespacedName, ns)
			Expect(err).NotTo(HaveOccurred())
			// Ensure APIVersion and Kind are set for testing
			ns.APIVersion = "v1"
			ns.Kind = namespaceKind
			ns.Labels = make(map[string]string) // Remove any labels
			Expect(k8sClient.Update(ctx, ns)).To(Succeed())

			By("Reconciling the namespace")
			controllerReconciler := &NamespaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that NamespaceState was deleted")
			nsStateCheck := &policyv1alpha.NamespaceState{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, nsStateCheck)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should update NamespaceState with pending annotation when class changes", func() {
			By("Creating a NamespaceState with different class")
			nsState := &policyv1alpha.NamespaceState{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
				Spec: policyv1alpha.NamespaceStateSpec{
					NamespaceClass: "old-class",
				},
			}
			Expect(k8sClient.Create(ctx, nsState)).To(Succeed())

			By("Adding the required label with different class")
			ns := &corev1.Namespace{}
			err := k8sClient.Get(ctx, typeNamespacedName, ns)
			Expect(err).NotTo(HaveOccurred())
			// Ensure APIVersion and Kind are set for testing
			ns.APIVersion = "v1"
			ns.Kind = namespaceKind
			if ns.Labels == nil {
				ns.Labels = make(map[string]string)
			}
			ns.Labels[policyv1alpha.LabelNamespaceClassName] = "new-class"
			Expect(k8sClient.Update(ctx, ns)).To(Succeed())

			By("Reconciling the namespace")
			controllerReconciler := &NamespaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that NamespaceState has pending update annotation")
			updatedNSState := &policyv1alpha.NamespaceState{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, updatedNSState)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedNSState.Spec.NamespaceClass).To(Equal("old-class"))
			Expect(updatedNSState.Annotations).To(HaveKey(policyv1alpha.AnnotationNamespaceClassPendingUpdate))
			Expect(updatedNSState.Annotations[policyv1alpha.AnnotationNamespaceClassPendingUpdate]).To(Equal("new-class"))
		})

		It("should do nothing when NamespaceState class matches the label", func() {
			By("Creating a NamespaceState with matching class")
			nsState := &policyv1alpha.NamespaceState{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespaceName,
				},
				Spec: policyv1alpha.NamespaceStateSpec{
					NamespaceClass: "test-class",
				},
			}
			Expect(k8sClient.Create(ctx, nsState)).To(Succeed())

			By("Adding the required label with matching class")
			ns := &corev1.Namespace{}
			err := k8sClient.Get(ctx, typeNamespacedName, ns)
			Expect(err).NotTo(HaveOccurred())
			// Ensure APIVersion and Kind are set for testing
			ns.APIVersion = "v1"
			ns.Kind = namespaceKind
			if ns.Labels == nil {
				ns.Labels = make(map[string]string)
			}
			ns.Labels[policyv1alpha.LabelNamespaceClassName] = "test-class"
			Expect(k8sClient.Update(ctx, ns)).To(Succeed())

			By("Reconciling the namespace")
			controllerReconciler := &NamespaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that NamespaceState remains unchanged")
			updatedNSState := &policyv1alpha.NamespaceState{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, updatedNSState)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedNSState.Spec.NamespaceClass).To(Equal("test-class"))
			Expect(updatedNSState.Annotations).NotTo(HaveKey(policyv1alpha.AnnotationNamespaceClassPendingUpdate))
		})

		It("should handle namespace not found", func() {
			By("Reconciling a non-existent namespace")
			controllerReconciler := &NamespaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "non-existent-namespace"},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should do nothing when namespace has no label and no NamespaceState exists", func() {
			By("Ensuring namespace has no label")
			ns := &corev1.Namespace{}
			err := k8sClient.Get(ctx, typeNamespacedName, ns)
			Expect(err).NotTo(HaveOccurred())
			// Ensure APIVersion and Kind are set for testing
			ns.APIVersion = "v1"
			ns.Kind = namespaceKind
			ns.Labels = make(map[string]string) // Remove any labels
			Expect(k8sClient.Update(ctx, ns)).To(Succeed())

			By("Reconciling the namespace")
			controllerReconciler := &NamespaceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that no NamespaceState was created")
			nsState := &policyv1alpha.NamespaceState{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: namespaceName}, nsState)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})

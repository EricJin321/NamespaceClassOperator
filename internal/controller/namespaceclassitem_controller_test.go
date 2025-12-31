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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

var _ = Describe("NamespaceClassItem Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		namespaceclassitem := &policyv1alpha.NamespaceClassItem{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind NamespaceClassItem")
			err := k8sClient.Get(ctx, typeNamespacedName, namespaceclassitem)
			if err != nil && errors.IsNotFound(err) {
				resource := &policyv1alpha.NamespaceClassItem{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: policyv1alpha.NamespaceClassItemSpec{
						Group:    "",
						Version:  "v1",
						Resource: "configmaps",
						Spec: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-configmap
data:
  key: value`,
					},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			// Cleanup logic after each test, like removing the resource instance.
			resource := &policyv1alpha.NamespaceClassItem{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				By("Cleanup the specific resource instance NamespaceClassItem")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}

			// Cleanup NamespaceClass
			nc := &policyv1alpha.NamespaceClass{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-namespaceclass"}, nc)
			if err == nil {
				Expect(k8sClient.Delete(ctx, nc)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource", func() {
			By("Creating a NamespaceClass")
			nc := &policyv1alpha.NamespaceClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespaceclass",
				},
				Spec: policyv1alpha.NamespaceClassSpec{
					Items: []string{resourceName},
				},
			}
			Expect(k8sClient.Create(ctx, nc)).To(Succeed())

			By("Reconciling the created resource")
			controllerReconciler := &NamespaceClassItemReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the status was updated")
			updatedNCI := &policyv1alpha.NamespaceClassItem{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedNCI)
			Expect(err).NotTo(HaveOccurred())
			// The controller updates status based on NamespaceClass references
		})

		It("should handle NamespaceClass not found", func() {
			By("Reconciling the created resource without NamespaceClass")
			controllerReconciler := &NamespaceClassItemReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			// Should not fail even if no NamespaceClass references it
		})

		It("should handle finalizer removal", func() {
			By("Adding finalizer to the resource")
			nci := &policyv1alpha.NamespaceClassItem{}
			err := k8sClient.Get(ctx, typeNamespacedName, nci)
			Expect(err).NotTo(HaveOccurred())

			nci.Finalizers = []string{"policy.akuity.io/finalizer"}
			Expect(k8sClient.Update(ctx, nci)).To(Succeed())

			By("Reconciling the resource")
			controllerReconciler := &NamespaceClassItemReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})

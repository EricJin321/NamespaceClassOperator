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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

var _ = Describe("NamespaceItemInstance Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}

		BeforeEach(func() {
			By("creating the custom resource for the Kind NamespaceItemInstance")
			resource := &policyv1alpha.NamespaceItemInstance{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "policy.akuity.io/v1alpha",
					Kind:       "NamespaceItemInstance",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: policyv1alpha.NamespaceItemInstanceSpec{
					NamespaceClassItem: "test-namespaceclassitem",
				},
			}
			// Try to create, ignore AlreadyExists error
			_ = k8sClient.Create(ctx, resource)
		})

		AfterEach(func() {
			// Cleanup logic after each test, like removing the resource instance.
			resource := &policyv1alpha.NamespaceItemInstance{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			if err == nil {
				By("Cleanup the specific resource instance NamespaceItemInstance")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			}

			// Cleanup NamespaceClassItem
			nci := &policyv1alpha.NamespaceClassItem{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-namespaceclassitem"}, nci)
			if err == nil {
				Expect(k8sClient.Delete(ctx, nci)).To(Succeed())
			}

			// Cleanup any created ConfigMap
			cm := &unstructured.Unstructured{}
			cm.SetAPIVersion("v1")
			cm.SetKind("ConfigMap")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-configmap", Namespace: "default"}, cm)
			if err == nil {
				Expect(k8sClient.Delete(ctx, cm)).To(Succeed())
			}
		})

		It("should successfully reconcile the resource when creating a new resource", func() {
			By("Creating a NamespaceClassItem")
			nci := &policyv1alpha.NamespaceClassItem{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "policy.akuity.io/v1alpha",
					Kind:       "NamespaceClassItem",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-namespaceclassitem",
					Generation: 1, // Simulate real k8s behavior where new resources have generation 1
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
			Expect(k8sClient.Create(ctx, nci)).To(Succeed())

			By("Reconciling the created resource")
			controllerReconciler := &NamespaceItemInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the ConfigMap was created")
			cm := &unstructured.Unstructured{}
			cm.SetAPIVersion("v1")
			cm.SetKind("ConfigMap")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-configmap", Namespace: "default"}, cm)
			Expect(err).NotTo(HaveOccurred())
			Expect(cm.Object["data"].(map[string]interface{})["key"]).To(Equal("value"))

			By("Checking that the status was updated")
			updatedNII := &policyv1alpha.NamespaceItemInstance{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedNII)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedNII.Status.ObservedGeneration).To(Equal(nci.Generation))
		})

		It("should successfully reconcile the resource when updating an existing resource", func() {
			By("Creating a NamespaceClassItem")
			nci := &policyv1alpha.NamespaceClassItem{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "policy.akuity.io/v1alpha",
					Kind:       "NamespaceClassItem",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-namespaceclassitem",
					Generation: 1, // Simulate real k8s behavior where new resources have generation 1
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
			Expect(k8sClient.Create(ctx, nci)).To(Succeed())

			By("Creating the ConfigMap manually first")
			cm := &unstructured.Unstructured{
				Object: map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "ConfigMap",
					"metadata": map[string]interface{}{
						"name":      "test-configmap",
						"namespace": "default",
					},
					"data": map[string]interface{}{
						"key": "oldvalue",
					},
				},
			}
			Expect(k8sClient.Create(ctx, cm)).To(Succeed())

			By("Reconciling the created resource")
			controllerReconciler := &NamespaceItemInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the ConfigMap was updated")
			updatedCM := &unstructured.Unstructured{}
			updatedCM.SetAPIVersion("v1")
			updatedCM.SetKind("ConfigMap")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-configmap", Namespace: "default"}, updatedCM)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedCM.Object["data"].(map[string]interface{})["key"]).To(Equal("value"))
		})

		It("should skip reconciliation in system namespaces", func() {
			By("Creating a NamespaceItemInstance in kube-system namespace")
			systemNII := &policyv1alpha.NamespaceItemInstance{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "policy.akuity.io/v1alpha",
					Kind:       "NamespaceItemInstance",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "system-test",
					Namespace: "kube-system",
				},
				Spec: policyv1alpha.NamespaceItemInstanceSpec{
					NamespaceClassItem: "test-namespaceclassitem",
				},
			}
			Expect(k8sClient.Create(ctx, systemNII)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, systemNII)
			}()

			By("Reconciling the resource")
			controllerReconciler := &NamespaceItemInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "system-test", Namespace: "kube-system"},
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that no ConfigMap was created")
			cm := &unstructured.Unstructured{}
			cm.SetAPIVersion("v1")
			cm.SetKind("ConfigMap")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-configmap", Namespace: "kube-system"}, cm)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})

		It("should handle NamespaceClassItem not found", func() {
			By("Reconciling the created resource without NamespaceClassItem")
			controllerReconciler := &NamespaceItemInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that status was not updated")
			updatedNII := &policyv1alpha.NamespaceItemInstance{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedNII)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedNII.Status.ObservedGeneration).To(Equal(int64(0)))
		})

		It("should handle resource not found", func() {
			By("Reconciling a non-existent resource")
			controllerReconciler := &NamespaceItemInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "non-existent", Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
		})

		It("should remove the updated annotation when present", func() {
			By("Creating a NamespaceClassItem")
			nci := &policyv1alpha.NamespaceClassItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-namespaceclassitem",
					Generation: 1, // Simulate real k8s behavior where new resources have generation 1
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
			Expect(k8sClient.Create(ctx, nci)).To(Succeed())

			By("Adding the updated annotation to the NamespaceItemInstance")
			nii := &policyv1alpha.NamespaceItemInstance{}
			err := k8sClient.Get(ctx, typeNamespacedName, nii)
			Expect(err).NotTo(HaveOccurred())
			if nii.Annotations == nil {
				nii.Annotations = make(map[string]string)
			}
			nii.Annotations[policyv1alpha.AnnotationNamespaceClassItemUpdated] = policyv1alpha.AnnotationValueTrue
			Expect(k8sClient.Update(ctx, nii)).To(Succeed())

			By("Reconciling the resource")
			controllerReconciler := &NamespaceItemInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that the annotation was removed")
			updatedNII := &policyv1alpha.NamespaceItemInstance{}
			err = k8sClient.Get(ctx, typeNamespacedName, updatedNII)
			Expect(err).NotTo(HaveOccurred())
			Expect(updatedNII.Annotations).NotTo(HaveKey(policyv1alpha.AnnotationNamespaceClassItemUpdated))
		})

		It("should not update when generation matches", func() {
			By("Creating a NamespaceClassItem")
			nci := &policyv1alpha.NamespaceClassItem{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-namespaceclassitem",
					Generation: 1, // Simulate real k8s behavior where new resources have generation 1
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
			Expect(k8sClient.Create(ctx, nci)).To(Succeed())

			By("Setting the observed generation to match")
			nii := &policyv1alpha.NamespaceItemInstance{}
			err := k8sClient.Get(ctx, typeNamespacedName, nii)
			Expect(err).NotTo(HaveOccurred())
			nii.Status.ObservedGeneration = nci.Generation // Both should be 1 now
			Expect(k8sClient.Status().Update(ctx, nii)).To(Succeed())

			By("Reconciling the resource")
			controllerReconciler := &NamespaceItemInstanceReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking that no ConfigMap was created")
			cm := &unstructured.Unstructured{}
			cm.SetAPIVersion("v1")
			cm.SetKind("ConfigMap")
			err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-configmap", Namespace: "default"}, cm)
			Expect(errors.IsNotFound(err)).To(BeTrue())
		})
	})
})

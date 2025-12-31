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

package v1alpha

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	policyv1alpha "github.com/EricJin321/NamespaceClassOperator/api/v1alpha"
)

var _ = Describe("NamespaceClass Webhook", func() {
	var (
		obj       *policyv1alpha.NamespaceClass
		oldObj    *policyv1alpha.NamespaceClass
		validator NamespaceClassCustomValidator
	)

	BeforeEach(func() {
		By("Initializing validator")
		validator = NamespaceClassCustomValidator{
			Client: fake.NewClientBuilder().WithScheme(scheme.Scheme).Build(),
		}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")

		By("Creating required NamespaceClassItems")
		nci1 := &policyv1alpha.NamespaceClassItem{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "policy.akuity.io/v1alpha",
				Kind:       "NamespaceClassItem",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-item1",
			},
			Spec: policyv1alpha.NamespaceClassItemSpec{
				Group:    "",
				Version:  "v1",
				Resource: "configmaps",
				Spec: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-configmap1
data:
  key: value1`,
			},
		}
		Expect(validator.Client.Create(context.Background(), nci1)).To(Succeed())

		nci2 := &policyv1alpha.NamespaceClassItem{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "policy.akuity.io/v1alpha",
				Kind:       "NamespaceClassItem",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-item2",
			},
			Spec: policyv1alpha.NamespaceClassItemSpec{
				Group:    "",
				Version:  "v1",
				Resource: "configmaps",
				Spec: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-configmap2
data:
  key: value2`,
			},
		}
		Expect(validator.Client.Create(context.Background(), nci2)).To(Succeed())

		obj = &policyv1alpha.NamespaceClass{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "policy.akuity.io/v1alpha",
				Kind:       "NamespaceClass",
			},
			Spec: policyv1alpha.NamespaceClassSpec{
				Items: []string{"test-item1", "test-item2"},
			},
		}
		oldObj = &policyv1alpha.NamespaceClass{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "policy.akuity.io/v1alpha",
				Kind:       "NamespaceClass",
			},
			Spec: policyv1alpha.NamespaceClassSpec{
				Items: []string{"test-item1"},
			},
		}
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	AfterEach(func() {
		// TODO (user): Add any teardown logic common to all tests
	})

	Context("When creating NamespaceClass", func() {
		It("should validate successfully for valid spec", func() {
			warnings, err := validator.ValidateCreate(context.Background(), obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeNil())
		})

		It("should fail validation for empty items list", func() {
			obj.Spec.Items = []string{}
			warnings, err := validator.ValidateCreate(context.Background(), obj)
			Expect(err).To(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})

	Context("When updating NamespaceClass", func() {
		It("should validate successfully for valid update", func() {
			warnings, err := validator.ValidateUpdate(context.Background(), oldObj, obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeNil())
		})

		It("should validate successfully when removing items", func() {
			oldObj.Spec.Items = []string{"test-item1", "test-item2"}
			obj.Spec.Items = []string{"test-item1"}
			warnings, err := validator.ValidateUpdate(context.Background(), oldObj, obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})

	Context("When deleting NamespaceClass", func() {
		It("should validate successfully", func() {
			warnings, err := validator.ValidateDelete(context.Background(), obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeNil())
		})
	})
})

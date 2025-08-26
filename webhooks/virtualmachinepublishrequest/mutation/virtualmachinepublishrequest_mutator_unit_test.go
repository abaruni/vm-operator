// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package mutation_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha5"
	pkgconst "github.com/vmware-tanzu/vm-operator/pkg/constants"
	"github.com/vmware-tanzu/vm-operator/pkg/constants/testlabels"
	"github.com/vmware-tanzu/vm-operator/test/builder"
	"github.com/vmware-tanzu/vm-operator/webhooks/virtualmachinepublishrequest/mutation"
)

func uniTests() {
	Describe(
		"Mutate",
		Label(
			testlabels.Create,
			testlabels.Update,
			testlabels.Delete,
			testlabels.API,
			testlabels.Mutation,
			testlabels.Webhook,
		),
		unitTestsMutating,
	)
}

type unitMutationWebhookContext struct {
	builder.UnitTestContextForMutatingWebhook
	vmPubReq *vmopv1.VirtualMachinePublishRequest
}

func newUnitTestContextForMutatingWebhook() *unitMutationWebhookContext {
	vmPubReq := builder.DummyVirtualMachinePublishRequest(
		"dummy-vm-publish-request",
		"dummy-ns",
		"dummy-vm",
		"dummy-template",
		"dummy-cl")

	obj, err := builder.ToUnstructured(vmPubReq)
	Expect(err).ToNot(HaveOccurred())

	return &unitMutationWebhookContext{
		UnitTestContextForMutatingWebhook: *suite.NewUnitTestContextForMutatingWebhook(obj),
		vmPubReq:                          vmPubReq,
	}
}

func unitTestsMutating() {
	var (
		ctx *unitMutationWebhookContext
	)

	BeforeEach(func() {
		ctx = newUnitTestContextForMutatingWebhook()
		ctx.Op = admissionv1.Create

		rawObj, err := json.Marshal(ctx.vmPubReq)
		Expect(err).ToNot(HaveOccurred())
		ctx.RawObj = rawObj
	})

	AfterEach(func() {
		ctx = nil
	})

	When("operation is not CREATE", func() {
		nonCreateOperation := func(op admissionv1.Operation) bool {
			ctx.Op = op
			return ctx.Mutate(&ctx.WebhookRequestContext).Allowed
		}
		It("should mutate VirtualMachinePublishRequest successfully", func() {
			Expect(nonCreateOperation(admissionv1.Update)).To(BeTrue())
			Expect(nonCreateOperation(admissionv1.Delete)).To(BeTrue())
			Expect(nonCreateOperation(admissionv1.Connect)).To(BeTrue())
		})
	})

	When("request object does not contain quota check annotation", func() {
		It("should not mutate VirtualMachinePublishRequest", func() {
			Expect(mutation.SetQuotaCheckAnnotation(&ctx.WebhookRequestContext, ctx.vmPubReq)).To(BeFalse())
			Expect(ctx.vmPubReq.Annotations).To(BeNil())
		})
	})

	When("request object contains quota check annotation", func() {
		When("quota check annotation is true", func() {
			BeforeEach(func() {
				ctx.vmPubReq.Annotations = map[string]string{
					pkgconst.AsyncQuotaPerformCheckAnnotationKey: "true",
				}
			})

			It("should apply the correct annotation", func() {
				Expect(mutation.SetQuotaCheckAnnotation(&ctx.WebhookRequestContext, ctx.vmPubReq)).To(BeTrue())
				Expect(ctx.vmPubReq.Annotations).NotTo(BeNil())

				_, ok := ctx.vmPubReq.Annotations[pkgconst.AsyncQuotaCheckRequestedCapacityAnnotationKey]
				Expect(ok).To(BeTrue())
			})
		})

		When("quota check annotation is false", func() {
			BeforeEach(func() {
				ctx.vmPubReq.Annotations = map[string]string{
					pkgconst.AsyncQuotaPerformCheckAnnotationKey: "false",
				}
			})

			It("should not apply any annotations", func() {
				Expect(mutation.SetQuotaCheckAnnotation(&ctx.WebhookRequestContext, ctx.vmPubReq)).To(BeFalse())
				Expect(ctx.vmPubReq.Annotations).NotTo(BeNil())

				_, ok := ctx.vmPubReq.Annotations[pkgconst.AsyncQuotaCheckRequestedCapacityAnnotationKey]
				Expect(ok).To(BeFalse())
			})
		})

		When("quota check annotation is empty", func() {
			BeforeEach(func() {
				ctx.vmPubReq.Annotations = map[string]string{
					pkgconst.AsyncQuotaPerformCheckAnnotationKey: "",
				}
			})

			It("should not apply any annotations", func() {
				Expect(mutation.SetQuotaCheckAnnotation(&ctx.WebhookRequestContext, ctx.vmPubReq)).To(BeFalse())
				Expect(ctx.vmPubReq.Annotations).NotTo(BeNil())

				_, ok := ctx.vmPubReq.Annotations[pkgconst.AsyncQuotaCheckRequestedCapacityAnnotationKey]
				Expect(ok).To(BeFalse())
			})
		})
	})
}

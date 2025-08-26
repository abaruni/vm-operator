// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package mutation_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha5"
	pkgconst "github.com/vmware-tanzu/vm-operator/pkg/constants"
	"github.com/vmware-tanzu/vm-operator/pkg/constants/testlabels"
	"github.com/vmware-tanzu/vm-operator/test/builder"
)

func intgTests() {
	Describe(
		"Mutate",
		Label(
			testlabels.Create,
			testlabels.Update,
			testlabels.Delete,
			testlabels.EnvTest,
			testlabels.API,
			testlabels.Mutation,
			testlabels.Webhook,
		),
		intgTestsMutating,
	)
}

type intgMutatingWebhookContext struct {
	builder.IntegrationTestContext
	vmPubReq *vmopv1.VirtualMachinePublishRequest
}

func newIntgMutatingWebhookContext() *intgMutatingWebhookContext {
	ctx := &intgMutatingWebhookContext{
		IntegrationTestContext: *suite.NewIntegrationTestContext(),
	}

	ctx.vmPubReq = builder.DummyVirtualMachinePublishRequest(
		"dummy-name",
		ctx.Namespace,
		"dummy-vm",
		"dummy-template",
		"dummy-cl")

	return ctx
}

func intgTestsMutating() {
	var (
		ctx               *intgMutatingWebhookContext
		mutatedReq        *vmopv1.VirtualMachinePublishRequest
		expectedCreateErr *errors.StatusError
	)

	BeforeEach(func() {
		ctx = newIntgMutatingWebhookContext()
		mutatedReq = &vmopv1.VirtualMachinePublishRequest{}
		expectedCreateErr = nil
	})

	JustBeforeEach(func() {
		actualCreateErr := ctx.Client.Create(ctx.Context, ctx.vmPubReq)
		if expectedCreateErr == nil {
			Expect(actualCreateErr).NotTo(HaveOccurred())
			Expect(ctx.Client.Get(ctx.Context, ctrlclient.ObjectKeyFromObject(ctx.vmPubReq), mutatedReq)).To(Succeed())
		} else {
			Expect(actualCreateErr).To(MatchError(expectedCreateErr))
		}
	})

	AfterEach(func() {
		if expectedCreateErr == nil {
			Expect(ctx.Client.Delete(ctx.Context, ctx.vmPubReq)).To(Succeed())
		}
		ctx.AfterEach()
		ctx = nil
		mutatedReq = nil
		expectedCreateErr = nil
	})

	When("request object does not contain quota check annotation", func() {
		It("should not mutate VirtualMachinePublishRequest", func() {
			Expect(mutatedReq.Annotations).To(BeNil())
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
				Expect(mutatedReq.Annotations).NotTo(BeNil())

				_, ok := mutatedReq.Annotations[pkgconst.AsyncQuotaCheckRequestedCapacityAnnotationKey]
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
				Expect(mutatedReq.Annotations).NotTo(BeNil())

				_, ok := mutatedReq.Annotations[pkgconst.AsyncQuotaCheckRequestedCapacityAnnotationKey]
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
				Expect(mutatedReq.Annotations).NotTo(BeNil())

				_, ok := mutatedReq.Annotations[pkgconst.AsyncQuotaCheckRequestedCapacityAnnotationKey]
				Expect(ok).To(BeFalse())
			})
		})
	})
}

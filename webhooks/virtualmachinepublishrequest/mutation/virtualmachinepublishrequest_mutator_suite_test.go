// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package mutation_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"

	pkgcfg "github.com/vmware-tanzu/vm-operator/pkg/config"
	"github.com/vmware-tanzu/vm-operator/test/builder"
	"github.com/vmware-tanzu/vm-operator/webhooks/virtualmachinepublishrequest/mutation"
)

const (
	WebhookName = "default.mutating.virtualmachinepublishrequest.v1alpha5.vmoperator.vmware.com"
)

var suite = builder.NewTestSuiteForMutatingWebhookWithContext(
	pkgcfg.NewContext(),
	mutation.AddToManager,
	mutation.NewMutator,
	WebhookName)

func TestWebhook(t *testing.T) {
	suite.Register(t, "Mutating webhook suite", intgTests, uniTests)
}

var _ = BeforeSuite(suite.BeforeSuite)

var _ = AfterSuite(suite.AfterSuite)

// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha2_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestV1Alpha2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "v1alpha2 Suite")
}

// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package v1alpha2_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/google/go-cmp/cmp"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlconversion "sigs.k8s.io/controller-runtime/pkg/conversion"

	vmopv1a2 "github.com/vmware-tanzu/vm-operator/api/v1alpha2"
	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha3"
)

func TestVirtualMachineClassConversion(t *testing.T) {

	hubSpokeHub := func(g *WithT, hub, hubAfter ctrlconversion.Hub, spoke ctrlconversion.Convertible) {
		hubBefore := hub.DeepCopyObject().(ctrlconversion.Hub)

		// First convert hub to spoke
		dstCopy := spoke.DeepCopyObject().(ctrlconversion.Convertible)
		g.Expect(dstCopy.ConvertFrom(hubBefore)).To(Succeed())

		// Convert spoke back to hub and check if the resulting hub is equal to the hub before the round trip
		g.Expect(dstCopy.ConvertTo(hubAfter)).To(Succeed())

		g.Expect(apiequality.Semantic.DeepEqual(hubBefore, hubAfter)).To(BeTrue(), cmp.Diff(hubBefore, hubAfter))
	}

	t.Run("VirtualMachineClass hub-spoke-hub", func(t *testing.T) {

		t.Run("empty class", func(t *testing.T) {
			g := NewWithT(t)
			hub := vmopv1.VirtualMachineClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-vm-class",
					Namespace: "my-namespace",
				},
			}
			hubSpokeHub(g, &hub, &vmopv1.VirtualMachineClass{}, &vmopv1a2.VirtualMachineClass{})
		})
		t.Run("empty class w some annotations", func(t *testing.T) {
			g := NewWithT(t)
			hub := vmopv1.VirtualMachineClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-vm-class",
					Namespace: "my-namespace",
					Annotations: map[string]string{
						"fizz": "buzz",
					},
				},
			}
			hubSpokeHub(g, &hub, &vmopv1.VirtualMachineClass{}, &vmopv1a2.VirtualMachineClass{})
		})
		t.Run("class w some annotations and reserved profile ID and slots", func(t *testing.T) {
			g := NewWithT(t)
			hub := vmopv1.VirtualMachineClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-vm-class",
					Namespace: "my-namespace",
					Annotations: map[string]string{
						"fizz": "buzz",
					},
				},
				Spec: vmopv1.VirtualMachineClassSpec{
					ReservedProfileID: "my-profile-id",
					ReservedSlots:     4,
				},
			}
			hubSpokeHub(g, &hub, &vmopv1.VirtualMachineClass{}, &vmopv1a2.VirtualMachineClass{})
		})
		t.Run("class w reserved profile ID and slots", func(t *testing.T) {
			g := NewWithT(t)
			hub := vmopv1.VirtualMachineClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "my-vm-class",
					Namespace: "my-namespace",
				},
				Spec: vmopv1.VirtualMachineClassSpec{
					ReservedProfileID: "my-profile-id",
					ReservedSlots:     4,
				},
			}
			hubSpokeHub(g, &hub, &vmopv1.VirtualMachineClass{}, &vmopv1a2.VirtualMachineClass{})
		})
	})
}

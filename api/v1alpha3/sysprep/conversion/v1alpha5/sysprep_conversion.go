// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

package v1alpha5

import (
	"unsafe"

	apiconversion "k8s.io/apimachinery/pkg/conversion"

	vmopv1a3sysprep "github.com/vmware-tanzu/vm-operator/api/v1alpha3/sysprep"
	vmopv1sysprep "github.com/vmware-tanzu/vm-operator/api/v1alpha5/sysprep"
)

// Convert_sysprep_Sysprep_To_sysprep_Sysprep converts the Sysprep from v1alpha5
// to v1alpha3.
// Please see https://github.com/kubernetes/code-generator/issues/172 for why
// this function exists in this directory structure.
func Convert_sysprep_Sysprep_To_sysprep_Sysprep(
	in *vmopv1sysprep.Sysprep, out *vmopv1a3sysprep.Sysprep, s apiconversion.Scope) error {

	out.GUIRunOnce = (*vmopv1a3sysprep.GUIRunOnce)(unsafe.Pointer(in.GUIRunOnce))
	out.GUIUnattended = (*vmopv1a3sysprep.GUIUnattended)(unsafe.Pointer(in.GUIUnattended))
	out.LicenseFilePrintData = (*vmopv1a3sysprep.LicenseFilePrintData)(unsafe.Pointer(in.LicenseFilePrintData))
	out.UserData = *(*vmopv1a3sysprep.UserData)(unsafe.Pointer(&in.UserData))

	if id := in.Identification; id != nil {
		out.Identification = &vmopv1a3sysprep.Identification{
			DomainAdmin:         id.DomainAdmin,
			DomainAdminPassword: (*vmopv1a3sysprep.DomainPasswordSecretKeySelector)(unsafe.Pointer(id.DomainAdminPassword)),
			DomainOU:            id.DomainOU,
			JoinWorkgroup:       id.JoinWorkgroup,
		}
	}

	return nil
}

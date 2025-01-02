//go:build !ignore_autogenerated

// © Broadcom. All Rights Reserved.
// The term “Broadcom” refers to Broadcom Inc. and/or its subsidiaries.
// SPDX-License-Identifier: Apache-2.0

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Capabilities) DeepCopyInto(out *Capabilities) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Capabilities.
func (in *Capabilities) DeepCopy() *Capabilities {
	if in == nil {
		return nil
	}
	out := new(Capabilities)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Capabilities) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapabilitiesList) DeepCopyInto(out *CapabilitiesList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Capabilities, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapabilitiesList.
func (in *CapabilitiesList) DeepCopy() *CapabilitiesList {
	if in == nil {
		return nil
	}
	out := new(CapabilitiesList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *CapabilitiesList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapabilitiesSpec) DeepCopyInto(out *CapabilitiesSpec) {
	*out = *in
	if in.InfraCapabilities != nil {
		in, out := &in.InfraCapabilities, &out.InfraCapabilities
		*out = make([]Capability, len(*in))
		copy(*out, *in)
	}
	if in.SupervisorCapabilities != nil {
		in, out := &in.SupervisorCapabilities, &out.SupervisorCapabilities
		*out = make([]Capability, len(*in))
		copy(*out, *in)
	}
	if in.ServiceCapabilities != nil {
		in, out := &in.ServiceCapabilities, &out.ServiceCapabilities
		*out = make([]ServiceCapabilitiesSpec, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapabilitiesSpec.
func (in *CapabilitiesSpec) DeepCopy() *CapabilitiesSpec {
	if in == nil {
		return nil
	}
	out := new(CapabilitiesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapabilitiesStatus) DeepCopyInto(out *CapabilitiesStatus) {
	*out = *in
	if in.Supervisor != nil {
		in, out := &in.Supervisor, &out.Supervisor
		*out = make(map[CapabilityName]CapabilityStatus, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Services != nil {
		in, out := &in.Services, &out.Services
		*out = make(map[ServiceID]map[CapabilityName]CapabilityStatus, len(*in))
		for key, val := range *in {
			var outVal map[CapabilityName]CapabilityStatus
			if val == nil {
				(*out)[key] = nil
			} else {
				inVal := (*in)[key]
				in, out := &inVal, &outVal
				*out = make(map[CapabilityName]CapabilityStatus, len(*in))
				for key, val := range *in {
					(*out)[key] = val
				}
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapabilitiesStatus.
func (in *CapabilitiesStatus) DeepCopy() *CapabilitiesStatus {
	if in == nil {
		return nil
	}
	out := new(CapabilitiesStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Capability) DeepCopyInto(out *Capability) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Capability.
func (in *Capability) DeepCopy() *Capability {
	if in == nil {
		return nil
	}
	out := new(Capability)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *CapabilityStatus) DeepCopyInto(out *CapabilityStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new CapabilityStatus.
func (in *CapabilityStatus) DeepCopy() *CapabilityStatus {
	if in == nil {
		return nil
	}
	out := new(CapabilityStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ServiceCapabilitiesSpec) DeepCopyInto(out *ServiceCapabilitiesSpec) {
	*out = *in
	if in.Capabilities != nil {
		in, out := &in.Capabilities, &out.Capabilities
		*out = make([]Capability, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ServiceCapabilitiesSpec.
func (in *ServiceCapabilitiesSpec) DeepCopy() *ServiceCapabilitiesSpec {
	if in == nil {
		return nil
	}
	out := new(ServiceCapabilitiesSpec)
	in.DeepCopyInto(out)
	return out
}


/* **********************************************************
 * Copyright 2018 VMware, Inc.  All rights reserved. -- VMware Confidential
 * **********************************************************/


package v1beta1

import (
	"log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"vmware.com/kubevsphere/pkg/apis/vmoperator"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// VirtualMachine
// +k8s:openapi-gen=true
// +resource:path=virtualmachines,strategy=VirtualMachineStrategy
type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec,omitempty"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

// VirtualMachineSpec defines the desired state of VirtualMachine
type VirtualMachineSpec struct {
}

// VirtualMachineStatus defines the observed state of VirtualMachine
type VirtualMachineStatus struct {
}

// Validate checks that an instance of VirtualMachine is well formed
func (VirtualMachineStrategy) Validate(ctx request.Context, obj runtime.Object) field.ErrorList {
	o := obj.(*vmoperator.VirtualMachine)
	log.Printf("Validating fields for VirtualMachine %s\n", o.Name)
	errors := field.ErrorList{}
	// perform validation here and add to errors using field.Invalid
	return errors
}

// DefaultingFunction sets default VirtualMachine field values
func (VirtualMachineSchemeFns) DefaultingFunction(o interface{}) {
	obj := o.(*VirtualMachine)
	// set default field values here
	log.Printf("Defaulting fields for VirtualMachine %s\n", obj.Name)
}

/* **********************************************************
 * Copyright 2018 VMware, Inc.  All rights reserved. -- VMware Confidential
 * **********************************************************/


package virtualmachine

import (
	"context"
	"github.com/golang/glog"
	"github.com/kubernetes-incubator/apiserver-builder/pkg/builders"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"time"
	"vmware.com/kubevsphere/pkg/apis/vmoperator/v1beta1"
	clientSet "vmware.com/kubevsphere/pkg/client/clientset_generated/clientset"
	vmclientSet "vmware.com/kubevsphere/pkg/client/clientset_generated/clientset/typed/vmoperator/v1beta1"
	listers "vmware.com/kubevsphere/pkg/client/listers_generated/vmoperator/v1beta1"
	"vmware.com/kubevsphere/pkg/controller/sharedinformers"
	"vmware.com/kubevsphere/pkg/vmprovider"
)

// +controller:group=vmoperator,version=v1beta1,kind=VirtualMachine,resource=virtualmachines
type VirtualMachineControllerImpl struct {
	builders.DefaultControllerFns

	// lister indexes properties about VirtualMachine
	lister listers.VirtualMachineLister

	clientSet clientSet.Interface
	vmClientSet vmclientSet.VirtualMachineInterface
}

// Init initializes the controller and is called by the generated code
// Register watches for additional resource types here.
func (c *VirtualMachineControllerImpl) Init(arguments sharedinformers.ControllerInitArguments) {
	// Use the lister for indexing virtualmachines labels
	c.lister = arguments.GetSharedInformers().Factory.Vmoperator().V1beta1().VirtualMachines().Lister()

	clientSet, err := clientSet.NewForConfig(arguments.GetRestConfig())
	if err != nil {
		glog.Fatalf("error creating virtual machine client: %v", err)
	}
	c.clientSet = clientSet

	c.vmClientSet = clientSet.VmoperatorV1beta1().VirtualMachines(corev1.NamespaceDefault)
}

func (c *VirtualMachineControllerImpl) filter(list []string, strToFilter string) (newList []string) {
	for _, item := range list {
		if item != strToFilter {
			newList = append(newList, item)
		}
	}
	return
}

func (c *VirtualMachineControllerImpl) contains(list []string, strToSearch string) bool {
	for _, item := range list {
		if item == strToSearch {
			return true
		}
	}
	return false
}

// Reconcile handles enqueued messages
func (c *VirtualMachineControllerImpl) Reconcile(u *v1beta1.VirtualMachine) error {
	// Implement controller logic here
	//glog.V(0).Infof("Running reconcile VirtualMachine for %s\n", u.Name)
	glog.Infof("Running reconcile VirtualMachine for %s\n", u.Name)

	startTime := time.Now()
	defer func() {
		glog.V(0).Infof("Finished syncing vm %q (%v)", u.Name, time.Since(startTime))
	}()

	// We hold a Finalizer on the VM, so it must be present
	if !u.ObjectMeta.DeletionTimestamp.IsZero() {
		// This VM has been deleted, sync with backend
		glog.Infof("Deletion timestamp is non-zero")

		// Noop if our finalizer is not present
		//if u.ObjectMeta.Finalizers()
		if !c.contains(u.ObjectMeta.Finalizers, v1beta1.VirtualMachineFinalizer) {
			glog.Infof("reconciling virtual machine object %v causes a no-op as there is no finalizer.", u.Name)
			return nil
		}

		glog.Infof("reconciling virtual machine object %v triggers delete.", u.Name)
		if err := c.processVmDeletion(u); err != nil {
			glog.Errorf("Error deleting machine object %v; %v", u.Name, err)
			return err
		}

		// Remove finalizer on successful deletion.
		glog.Infof("virtual machine object %v deletion successful, removing finalizer.", u.Name)
		u.ObjectMeta.Finalizers = c.filter(u.ObjectMeta.Finalizers, v1beta1.VirtualMachineFinalizer)
		if _, err := c.vmClientSet.Update(u); err != nil {
			glog.Errorf("Error removing finalizer from machine object %v; %v", u.Name, err)
			return err
		}
	}

	// vm holds the latest vm info from apiserver
	vm, err := c.lister.VirtualMachines(u.Namespace).Get(u.Name)
	switch {
	case err != nil:
		glog.Infof("Unable to retrieve vm %v from store: %v", u.Name, err)
	default:
		err = c.processVmCreateOrUpdate(vm)
	}

	return err
}

	//c.lister.VirtualMachines(u.Namespace).Get(u.Name)

	/*
	instance := &vmv1.VM{}
	ctx := context.TODO()
	vMan := r.manager

	vClient, err := vsphere.NewClient(ctx, r.manager.Config.VcUrl)
	if err != nil {
		return reconcile.Result{}, err
	}

	defer vClient.Logout(ctx)

	// Fetch the VM instance
	err = r.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.

			log.Printf("Deleting VM %s %s", request.Name, instance.Name)
			vMan.DeleteVm(ctx, r.Client, vClient, request)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	log.Printf("Acquired VM %s %s", request.Name, instance.Name)

	// Check if the VM already exists in the backing Sddc
	vm, err := vMan.LookupVm(ctx, r.Client, vClient, request)
	if err == nil {
		log.Printf("VM exists")

		// Noop for now
		// If so, difference the spec with the VM config and reconfigure if necessary
		_, err := vMan.UpdateVm(ctx, r.Client, vClient, request, instance, vm)
		if err != nil {
			return reconcile.Result{}, err
		}
		return reconcile.Result{}, nil
	}

	if err != nil {
		if _, ok := err.(*find.NotFoundError); !ok {
			log.Printf("VM Lookup failed: %s!", err.Error())
			return reconcile.Result{}, err
		}
	}

	// VM Doesn't exist, create it
	log.Printf("VM does not exist")

	_, err = vMan.CreateVm(ctx, r.Client, vClient, request, instance)
	if err != nil {
		log.Printf("Create VM failed %s!", err)
		return reconcile.Result{}, err
	}
	*/
	//return nil

func (c *VirtualMachineControllerImpl) processVmDeletion(u *v1beta1.VirtualMachine) error {
	glog.Infof("Process VM Deletion for vm %s", u.Name)

	vmprovider, err := vmprovider.NewVmProvider(u.Namespace)
	if err != nil {
		glog.Errorf("Failed to find vmprovider")
		return errors.NewBadRequest("Namespace is invalid")
	}

	vmsProvider, supported := vmprovider.VirtualMachines()
	if !supported {
		glog.Errorf("Provider doesn't support vms func")
		return errors.NewMethodNotSupported(schema.GroupResource{"vmoperator", "VirtualMachines"}, "list")
	}

	ctx := context.TODO()
	err = vmsProvider.DeleteVirtualMachine(ctx, u.Name)
	if err != nil {
		if errors.IsNotFound(err) {
			glog.Infof("Failed to delete vm %s, already deleted?", u.Name)
		} else {
			glog.Errorf("Failed to delete vm %s: %s", u.Name, err)
			return err
		}
	}

	glog.V(4).Infof("Deleted VM %s %s", u.Name)
	return nil
}

func (c *VirtualMachineControllerImpl) processVmCreateOrUpdate(u *v1beta1.VirtualMachine) error {
	glog.Infof("Process VM Update for vm %s", u.Name)

	vmprovider, err := vmprovider.NewVmProvider(u.Namespace)
	if err != nil {
		glog.Errorf("Failed to find vmprovider")
		return errors.NewBadRequest("Namespace is invalid")
	}

	vmsProvider, supported := vmprovider.VirtualMachines()
	if !supported {
		glog.Errorf("Provider doesn't support vms func")
		return errors.NewMethodNotSupported(schema.GroupResource{"vmoperator", "VirtualMachines"}, "list")
	}

	ctx := context.TODO()
	vm, err := vmsProvider.GetVirtualMachine(ctx, u.Name)
	switch {
		// For now, treat any error as not found
	case err != nil:
	//case errors.IsNotFound(err):
		glog.Infof("VM doesn't exist in backend provider.  Creating now")
		err = c.processVmCreate(ctx, vmprovider, vmsProvider, u)
	//case err != nil:
	//	glog.Infof("Unable to retrieve vm %v from store: %v", u.Name, err)
	default:
		glog.V(4).Infof("Acquired VM %s %s", vm.Name, vm.InternalId)
		glog.Infof("Updating Vm %s", vm.Name)
		err = c.processVmUpdate(ctx, vmprovider, vmsProvider, u)
	}

	return nil
}

func VirtualMachineApiToProvider(vm v1beta1.VirtualMachine) vmprovider.VirtualMachine {
	return vmprovider.VirtualMachine{
		Name: vm.Name,
		PowerState: vm.Spec.PowerState,
	}
}

func (c *VirtualMachineControllerImpl) processVmCreate(ctx context.Context, vmprovider vmprovider.VirtualMachineProviderInterface, vmsProvider vmprovider.VirtualMachines, vm *v1beta1.VirtualMachine) error {
	glog.Infof("Creating VM: %s", vm.Name)
	err := vmsProvider.CreateVirtualMachine(ctx, VirtualMachineApiToProvider(*vm))
	if err != nil {
		glog.Errorf("Provider Failed to Create VM %s: %s", vm.Name, err)
	}
	return err
}

func (c *VirtualMachineControllerImpl) processVmUpdate(ctx context.Context, vmprovider vmprovider.VirtualMachineProviderInterface, vmsProvider vmprovider.VirtualMachines, vm *v1beta1.VirtualMachine) error {
	return nil
}

func (c *VirtualMachineControllerImpl) Get(namespace, name string) (*v1beta1.VirtualMachine, error) {
	return c.lister.VirtualMachines(namespace).Get(name)
}

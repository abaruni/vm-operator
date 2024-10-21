// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package crypto

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"github.com/vmware/govmomi/crypto"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	vimtypes "github.com/vmware/govmomi/vim25/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	byokv1 "github.com/vmware-tanzu/vm-operator/external/byok/api/v1alpha1"
	"github.com/vmware-tanzu/vm-operator/pkg/conditions"
	kubeutil "github.com/vmware-tanzu/vm-operator/pkg/util/kube"
	"github.com/vmware-tanzu/vm-operator/pkg/util/paused"
	"github.com/vmware-tanzu/vm-operator/pkg/vmconfig/crypto/internal"

	vmopv1 "github.com/vmware-tanzu/vm-operator/api/v1alpha3"
)

type ptrCfgSpec = *vimtypes.VirtualMachineConfigSpec
type ptrDevCfgSpec = *vimtypes.VirtualDeviceConfigSpec

type cryptoKey struct {
	id                string
	provider          string
	isDefaultProvider bool
}

type reconcileArgs struct {
	k8sClient             ctrlclient.Client
	vimClient             *vim25.Client
	vm                    *vmopv1.VirtualMachine
	moVM                  mo.VirtualMachine
	configSpec            ptrCfgSpec
	curKey                cryptoKey
	newKey                cryptoKey
	isEncStorClass        bool
	hasVTPM               bool
	addVTPM               bool
	remVTPM               bool
	encryptionClassName   string
	useDefaultKeyProvider bool
}

var (
	ErrMustUseVTPMOrEncryptedStorageClass = errors.New(
		"must use encrypted StorageClass or have vTPM")

	ErrMustNotUseVTPMOrEncryptedStorageClass = errors.New(
		"vTPM and/or encrypted StorageClass require encryption")

	ErrNoDefaultKeyProvider = errors.New("no default key provider")

	ErrInvalidKeyProvider = errors.New("invalid key provider")
)

func (r reconciler) Reconcile(
	ctx context.Context,
	k8sClient ctrlclient.Client,
	vimClient *vim25.Client,
	vm *vmopv1.VirtualMachine,
	moVM mo.VirtualMachine,
	configSpec ptrCfgSpec) error {

	if ctx == nil {
		panic("context is nil")
	}
	if k8sClient == nil {
		panic("k8sClient is nil")
	}
	if vimClient == nil {
		panic("vimClient is nil")
	}
	if vm == nil {
		panic("vm is nil")
	}
	if configSpec == nil {
		panic("configSpec is nil")
	}

	var (
		err  error
		args = reconcileArgs{
			k8sClient:  k8sClient,
			vimClient:  vimClient,
			vm:         vm,
			moVM:       moVM,
			configSpec: configSpec,
		}
	)

	// Record the VM's desired EncryptionClassName and whether or not to use
	// the default provider for rekeying existing VMs.
	if c := vm.Spec.Crypto; c == nil {
		args.useDefaultKeyProvider = true
	} else {
		args.encryptionClassName = c.EncryptionClassName
		if c.UseDefaultKeyProvider == nil {
			args.useDefaultKeyProvider = true
		} else {
			args.useDefaultKeyProvider = *c.UseDefaultKeyProvider
		}
	}

	// Check to if the VM is currently encrypted and record the current provider
	// ID and key ID.
	args.curKey = getCurCryptoKey(moVM)

	// Check whether or not the storage class is encrypted.
	args.isEncStorClass, err = kubeutil.IsEncryptedStorageClass(
		ctx,
		k8sClient,
		vm.Spec.StorageClass)
	if err != nil {
		return err
	}

	if paused.ByAdmin(moVM) || paused.ByDevOps(vm) {
		// Do not proceed further if the VM is paused.
		return nil
	}

	// Record whether the VM has, is adding, or is removing a vTPM.
	args.hasVTPM, args.addVTPM, args.remVTPM = hasVTPM(moVM, configSpec)

	if args.moVM.Config == nil {
		// A new VM is being created.
		return r.reconcileCreate(ctx, args)
	}

	// An existing VM is being updated.
	return r.reconcileUpdate(ctx, args)
}

func (r reconciler) reconcileCreate(
	ctx context.Context,
	args reconcileArgs) error {

	var err error

	if args.encryptionClassName != "" {

		//
		// The new VM specifies an EncryptionClass.
		//

		// Get the provider and key from the EncryptionClass.
		args.newKey, err = getCryptoKeyFromEncryptionClass(ctx, args)
		if err != nil {
			return setConditionAndReturnErr(args, err, ReasonInternalError)
		}

		if !args.hasVTPM && !args.addVTPM && !args.isEncStorClass {
			// The VM does not meet the requirements for encryption.
			return setConditionAndReturnErr(
				args,
				ErrMustUseVTPMOrEncryptedStorageClass,
				ReasonInvalidState)
		}

		// Encrypt the VM with the provider & key from the EncryptionClass.
		return doOp(ctx, args, doEncrypt)
	}

	// Attempt to get the default key provider.
	args.newKey = getCryptoKeyFromDefaultProvider(ctx, args)

	if args.newKey.provider == "" {

		//
		// There is no default key provider.
		//

		if args.hasVTPM || args.addVTPM || args.isEncStorClass {

			// The VM has a configuration that requires encryption.
			return setConditionAndReturnErr(
				args,
				ErrMustNotUseVTPMOrEncryptedStorageClass,
				ReasonNoDefaultKeyProvider)
		}

	} else {

		//
		// There is a default key provider.
		//

		if args.hasVTPM || args.addVTPM || args.isEncStorClass {

			//
			// The new VM meets the requirements to be encrypted.
			//

			// Encrypt the VM with the default key provider.
			return doOp(ctx, args, doEncrypt)
		}
	}

	// The new VM does not use encryption.
	return nil
}

func (r reconciler) reconcileUpdate(
	ctx context.Context,
	args reconcileArgs) error {

	var (
		err     error
		changed bool
	)

	if args.encryptionClassName != "" {
		// The existing VM specifies an EncryptionClass.
		changed, err = r.reconcileUpdateEncryptionClass(ctx, args)

	} else if args.useDefaultKeyProvider {
		// The existing VM indicates the default key provider should be used in
		// absence of the EncryptionClass.
		changed, err = r.reconcileUpdateDefaultKeyProvider(ctx, args)
	}

	if err != nil {
		return err
	}

	if changed {
		// A change was made to the existing VM to update its encryption state.
		return nil
	}

	//
	// The existing VM is not subject to any changes, i.e. its observed
	// encryption state is synchronized with its desired encryption state.
	//

	// Update the VM's status.
	return updateStatus(ctx, args)
}

func (r reconciler) reconcileUpdateEncryptionClass(
	ctx context.Context,
	args reconcileArgs) (bool, error) {

	var err error

	// Get the provider and key from the EncryptionClass.
	args.newKey, err = getCryptoKeyFromEncryptionClass(ctx, args)
	if err != nil {
		return false, setConditionAndReturnErr(args, err, ReasonInternalError)
	}

	if args.curKey.provider == "" {

		//
		// The existing VM is not encrypted.
		//

		// Encrypt the existing VM.
		return true, doOp(ctx, args, doEncrypt)
	}

	//
	// The existing VM is encrypted.
	//

	if args.curKey.provider != args.newKey.provider {

		//
		// The existing VM's key provider is different than the
		// specified key provider.
		//

		// Recrypt the existing VM.
		return true, doOp(ctx, args, doRecrypt)

	}

	//
	// The existing VM's key provider is the same as the specified provider.
	//

	if args.newKey.id != "" {

		//
		// The specified key is non-empty. Please note, empty keys are ignored
		// as the keys *will* be empty when the default provider is used since
		// keys are generated on-demand by vSphere when a default provider is
		// used.
		//

		if args.newKey.id != args.curKey.id {

			//
			// The specified key is different than the existing key, which
			// means the existing VM should be recrypted.
			//

			// Recrypt the existing VM.
			return true, doOp(ctx, args, doRecrypt)
		}
	}

	return false, nil
}

func (r reconciler) reconcileUpdateDefaultKeyProvider(
	ctx context.Context,
	args reconcileArgs) (bool, error) {

	// Attempt to get the default key provider.
	args.newKey = getCryptoKeyFromDefaultProvider(ctx, args)

	if args.curKey.provider == "" {

		//
		// The existing VM is not encrypted.
		//

		if args.newKey.provider != "" {

			//
			// There is a default key provider.
			//

			// Encrypt the existing VM.
			return true, doOp(ctx, args, doEncrypt)
		}

	} else {

		//
		// The existing VM is encrypted.
		//

		if args.newKey.provider == "" {

			//
			// There is no default key provider.
			//

			return false, setConditionAndReturnErr(
				args,
				ErrNoDefaultKeyProvider,
				ReasonNoDefaultKeyProvider)
		}

		if args.curKey.provider != args.newKey.provider {

			//
			// The existing VM's key provider is different than the
			// specified key provider.
			//

			// Recrypt the existing VM.
			return true, doOp(ctx, args, doRecrypt)
		}
	}

	return false, nil
}

func setConditionAndReturnErr(args reconcileArgs, err error, r Reason) error {
	if err == ErrInvalidKeyProvider {
		r = ReasonEncryptionClassInvalid
	} else if apierrors.IsNotFound(err) {
		r = ReasonEncryptionClassNotFound
	}
	conditions.MarkFalse(
		args.vm,
		vmopv1.VirtualMachineEncryptionSynced,
		r.String(),
		err.Error())
	return err
}

func updateStatus(ctx context.Context, args reconcileArgs) error {

	if args.curKey.provider == "" {

		//
		// The existing VM is not encrypted.
		//

		args.vm.Status.Crypto = nil
		conditions.Delete(args.vm, vmopv1.VirtualMachineEncryptionSynced)
		return nil
	}

	//
	// The existing VM is encrypted.
	//

	conditions.MarkTrue(args.vm, vmopv1.VirtualMachineEncryptionSynced)

	if args.vm.Status.Crypto == nil {
		args.vm.Status.Crypto = &vmopv1.VirtualMachineCryptoStatus{}
	}
	args.vm.Status.Crypto.ProviderID = args.curKey.provider
	args.vm.Status.Crypto.KeyID = args.curKey.id

	if args.isEncStorClass {
		args.vm.Status.Crypto.Encrypted = []vmopv1.VirtualMachineEncryptionType{
			vmopv1.VirtualMachineEncryptionTypeConfig,
			vmopv1.VirtualMachineEncryptionTypeDisks,
		}
	} else if args.hasVTPM && !args.remVTPM {
		args.vm.Status.Crypto.Encrypted = []vmopv1.VirtualMachineEncryptionType{
			vmopv1.VirtualMachineEncryptionTypeConfig,
		}
	}

	return doOp(ctx, args, doUpdateEncrypted)
}

func doOp(
	ctx context.Context,
	args reconcileArgs,
	fn func(context.Context, reconcileArgs) (string, Reason, []string, error)) error {

	// Please note, the return err is ignored here since it is not used anywhere
	// in the various functions that can be "fn." The reason the pattern is
	// retained is to avoid any refactoring later if there *does* need to be an
	// error returned from one of these functions.
	op, reason, msgs, _ := fn(ctx, args)

	if reason == 0 && len(msgs) == 0 {
		internal.SetOperation(ctx, op)
	} else {
		markEncryptionStateNotSynced(args.vm, op, reason, msgs...)
	}

	return nil
}

func doEncrypt(
	ctx context.Context,
	args reconcileArgs) (string, Reason, []string, error) {

	op := "encrypting"
	r, m, err := onEncrypt(ctx, args)
	return op, r, m, err
}

func doRecrypt(
	ctx context.Context,
	args reconcileArgs) (string, Reason, []string, error) {

	op := "recrypting"
	r, m, err := onRecrypt(ctx, args)
	return op, r, m, err
}

func doUpdateEncrypted(
	ctx context.Context,
	args reconcileArgs) (string, Reason, []string, error) {

	op := "updating encrypted"
	r, m, err := validateUpdateEncrypted(args)
	return op, r, m, err
}

func getCurCryptoKey(moVM mo.VirtualMachine) cryptoKey {
	var curKey cryptoKey
	if moVM.Config == nil {
		return curKey
	}
	if kid := moVM.Config.KeyId; kid != nil {
		curKey.id = kid.KeyId
		if pid := kid.ProviderId; pid != nil {
			curKey.provider = pid.Id
		}
	}
	return curKey
}

func getCryptoKeyFromEncryptionClass(
	ctx context.Context,
	args reconcileArgs) (cryptoKey, error) {

	var (
		obj    byokv1.EncryptionClass
		objKey = ctrlclient.ObjectKey{
			Namespace: args.vm.Namespace,
			Name:      args.vm.Spec.Crypto.EncryptionClassName,
		}
	)

	if err := args.k8sClient.Get(ctx, objKey, &obj); err != nil {
		return cryptoKey{}, err

	}

	m := crypto.NewManagerKmip(args.vimClient)
	if ok, _ := m.IsValidProvider(ctx, obj.Spec.KeyProvider); !ok {
		return cryptoKey{}, ErrInvalidKeyProvider
	}

	return cryptoKey{
		id:       obj.Spec.KeyID,
		provider: obj.Spec.KeyProvider,
	}, nil
}

func getCryptoKeyFromDefaultProvider(
	ctx context.Context,
	args reconcileArgs) cryptoKey {

	m := crypto.NewManagerKmip(args.vimClient)
	providerID, _ := m.GetDefaultKmsClusterID(ctx, nil, true)
	return cryptoKey{
		id:                "",
		provider:          providerID,
		isDefaultProvider: true,
	}
}

func onEncrypt(
	ctx context.Context,
	args reconcileArgs) (Reason, []string, error) {

	logger := logr.FromContextOrDiscard(ctx)

	reason, msgs, err := validateEncrypt(args)
	if reason > 0 || len(msgs) > 0 || err != nil {
		return reason, msgs, err
	}

	args.configSpec.Crypto = &vimtypes.CryptoSpecEncrypt{
		CryptoKeyId: vimtypes.CryptoKeyId{
			ProviderId: &vimtypes.KeyProviderId{
				Id: args.newKey.provider,
			},
			KeyId: args.newKey.id,
		},
	}

	logger.Info(
		"Encrypt VM",
		"newKeyID", args.newKey.id,
		"newProviderID", args.newKey.provider,
		"newProviderIsDefault", args.newKey.isDefaultProvider)

	return 0, nil, nil
}

func onRecrypt(
	ctx context.Context,
	args reconcileArgs) (Reason, []string, error) {

	logger := logr.FromContextOrDiscard(ctx)

	reason, msgs, err := validateRecrypt(args)
	if reason > 0 || len(msgs) > 0 || err != nil {
		return reason, msgs, err
	}

	args.configSpec.Crypto = &vimtypes.CryptoSpecShallowRecrypt{
		NewKeyId: vimtypes.CryptoKeyId{
			ProviderId: &vimtypes.KeyProviderId{
				Id: args.newKey.provider,
			},
			KeyId: args.newKey.id,
		},
	}

	logger.Info(
		"Recrypt VM",
		"currentKeyID", args.curKey.id,
		"currentProviderID", args.curKey.provider,
		"newKeyID", args.newKey.id,
		"newProviderID", args.newKey.provider,
		"newProviderIsDefault", args.newKey.isDefaultProvider)

	return 0, nil, nil
}

//nolint:unparam
func validateEncrypt(args reconcileArgs) (Reason, []string, error) {
	var (
		msgs   []string
		reason Reason
	)
	if r, m := validateStorageClassAndVTPM(args); r != 0 || len(m) > 0 {
		reason |= r
		msgs = append(msgs, m...)
	}
	if r, m := validatePoweredOffNoSnapshots(args.moVM); len(m) > 0 {
		reason |= r
		msgs = append(msgs, m...)
	}
	if r, m := validateDeviceChanges(args.configSpec); len(m) > 0 {
		reason |= r
		msgs = append(msgs, m...)
	}
	return reason, msgs, nil
}

//nolint:unparam
func validateRecrypt(args reconcileArgs) (Reason, []string, error) {
	var (
		msgs   []string
		reason Reason
	)
	if r, m := validateStorageClassAndVTPM(args); r != 0 || len(m) > 0 {
		reason |= r
		msgs = append(msgs, m...)
	}
	if hasSnapshotTree(args.moVM) {
		msgs = append(msgs, "not have snapshot tree")
		reason |= ReasonInvalidState
	}
	if r, m := validateDeviceChanges(args.configSpec); len(m) > 0 {
		reason |= r
		msgs = append(msgs, m...)
	}
	return reason, msgs, nil
}

//nolint:unparam
func validateUpdateEncrypted(args reconcileArgs) (Reason, []string, error) {
	var (
		msgs   []string
		reason Reason
	)
	if r, m := validateStorageClassAndVTPM(args); r != 0 || len(m) > 0 {
		reason |= r
		msgs = append(msgs, m...)
	}
	if isChangingSecretKey(args.configSpec) {
		msgs = append(msgs, "not add/remove/modify secret key")
		reason |= ReasonInvalidChanges
	}
	if r, m := validateDeviceChanges(args.configSpec); len(m) > 0 {
		reason |= r
		msgs = append(msgs, m...)
	}
	return reason, msgs, nil
}

func validateStorageClassAndVTPM(args reconcileArgs) (Reason, []string) {
	var (
		msgs   []string
		reason Reason
	)
	if args.remVTPM {
		reason |= ReasonInvalidChanges
		msgs = append(msgs, "not remove vTPM")
	}
	if !args.hasVTPM && !args.addVTPM && !args.isEncStorClass {
		reason |= ReasonInvalidState
		msgs = append(msgs, "use encryption storage class or have vTPM")
	}
	return reason, msgs
}

func hasVTPM(moVM mo.VirtualMachine, configSpec ptrCfgSpec) (bool, bool, bool) {
	var (
		has bool
		add bool
		rem bool
	)
	if c := moVM.Config; c != nil {
		for i := range c.Hardware.Device {
			if _, ok := c.Hardware.Device[i].(*vimtypes.VirtualTPM); ok {
				has = true
				break
			}
		}
	}
	for i := range configSpec.DeviceChange {
		if devChange := configSpec.DeviceChange[i]; devChange != nil {
			if devSpec := devChange.GetVirtualDeviceConfigSpec(); devSpec != nil {
				if _, ok := devSpec.Device.(*vimtypes.VirtualTPM); ok {
					if devSpec.Operation == vimtypes.VirtualDeviceConfigSpecOperationAdd {
						add = true
					} else if devSpec.Operation == vimtypes.VirtualDeviceConfigSpecOperationRemove {
						rem = true
					}
					break
				}
			}
		}
	}
	return has, add, rem
}

func validatePoweredOffNoSnapshots(moVM mo.VirtualMachine) (Reason, []string) {
	var (
		msgs   []string
		reason Reason
	)
	if moVM.Summary.Runtime.PowerState != "" &&
		moVM.Summary.Runtime.PowerState != vimtypes.VirtualMachinePowerStatePoweredOff {

		msgs = append(msgs, "be powered off")
		reason |= ReasonInvalidState
	}
	if moVM.Snapshot != nil && moVM.Snapshot.CurrentSnapshot != nil {
		msgs = append(msgs, "not have snapshots")
		reason |= ReasonInvalidState
	}
	return reason, msgs
}

func validateDeviceChanges(configSpec ptrCfgSpec) (Reason, []string) {

	var (
		msgs   []string
		reason Reason
	)

	for i := range configSpec.DeviceChange {
		if devChange := configSpec.DeviceChange[i]; devChange != nil {
			devSpec := devChange.GetVirtualDeviceConfigSpec()
			if isAddEditDeviceSpecEncryptedSansPolicy(devSpec) {
				msgs = append(msgs, "specify policy when encrypting devices")
				reason |= ReasonInvalidChanges
			}
			if isEncryptedRawDiskMapping(devSpec) {
				msgs = append(msgs, "not encrypt raw disks")
				reason |= ReasonInvalidChanges
			}
			if isEncryptedDeviceNonDisk(devSpec) {
				msgs = append(msgs, "not encrypt non-disk devices")
				reason |= ReasonInvalidChanges
			}
			if isEncryptedDeviceWithMultipleBackings(devSpec) {
				msgs = append(msgs, "not encrypt devices with multiple backings")
				reason |= ReasonInvalidChanges
			}
		}
	}

	return reason, msgs
}

var secretKeys = map[string]struct{}{
	"ancestordatafilekeys":           {},
	"cryptostate":                    {},
	"datafilekey":                    {},
	"encryption.required":            {},
	"encryption.required.vtpm":       {},
	"encryption.unspecified.default": {},
}

func isChangingSecretKey(configSpec ptrCfgSpec) bool {
	for i := range configSpec.ExtraConfig {
		if bov := configSpec.ExtraConfig[i]; bov != nil {
			if ov := bov.GetOptionValue(); ov != nil {
				if _, ok := secretKeys[ov.Key]; ok {
					return true
				}
			}
		}
	}
	return false
}

func isAddEditDeviceSpecEncryptedSansPolicy(spec ptrDevCfgSpec) bool {
	if spec != nil {
		switch spec.Operation {
		case vimtypes.VirtualDeviceConfigSpecOperationAdd,
			vimtypes.VirtualDeviceConfigSpecOperationEdit:

			if backing := spec.Backing; backing != nil {
				switch backing.Crypto.(type) {
				case *vimtypes.CryptoSpecEncrypt,
					*vimtypes.CryptoSpecDeepRecrypt,
					*vimtypes.CryptoSpecShallowRecrypt:

					for i := range spec.Profile {
						if doesProfileHaveIOFilters(spec.Profile[i]) {
							return false
						}
					}
					return true // is encrypted/recrypted sans policy
				}
			}
		}
	}
	return false
}

var whiteSpaceRegex = regexp.MustCompile(`[\s\t\n\r]`)

func doesProfileHaveIOFilters(spec vimtypes.BaseVirtualMachineProfileSpec) bool {
	if profile, ok := spec.(*vimtypes.VirtualMachineDefinedProfileSpec); ok {
		if data := profile.ProfileData; data != nil {
			if data.ExtensionKey == "com.vmware.vim.sips" {
				return strings.Contains(
					whiteSpaceRegex.ReplaceAllString(data.ObjectData, ""),
					"<namespace>IOFILTERS</namespace>")
			}
		}
	}
	return false
}

func isEncryptedDeviceNonDisk(spec ptrDevCfgSpec) bool {
	if spec != nil {
		if backing := spec.Backing; backing != nil {
			switch backing.Crypto.(type) {
			case *vimtypes.CryptoSpecEncrypt,
				*vimtypes.CryptoSpecDeepRecrypt,
				*vimtypes.CryptoSpecShallowRecrypt:

				_, isDisk := spec.Device.(*vimtypes.VirtualDisk)
				if !isDisk {
					return true
				}
			}
		}
	}
	return false
}

func isEncryptedDeviceWithMultipleBackings(spec ptrDevCfgSpec) bool {
	if spec != nil {
		if backing := spec.Backing; backing != nil {
			switch backing.Crypto.(type) {
			case *vimtypes.CryptoSpecEncrypt,
				*vimtypes.CryptoSpecDeepRecrypt,
				*vimtypes.CryptoSpecShallowRecrypt:

				return spec.Backing.Parent != nil
			}
		}
	}
	return false
}

func isEncryptedRawDiskMapping(spec ptrDevCfgSpec) bool {
	if spec != nil {
		if backing := spec.Backing; backing != nil {
			switch backing.Crypto.(type) {
			case *vimtypes.CryptoSpecEncrypt,
				*vimtypes.CryptoSpecDeepRecrypt,
				*vimtypes.CryptoSpecShallowRecrypt:

				if disk, ok := spec.Device.(*vimtypes.VirtualDisk); ok {
					//nolint:gocritic
					switch disk.Backing.(type) {
					case *vimtypes.VirtualDiskRawDiskVer2BackingInfo:
						return true
					}
				}
			}
		}
	}
	return false
}

func hasSnapshotTree(moVM mo.VirtualMachine) bool {
	var snapshotTree []vimtypes.VirtualMachineSnapshotTree
	if si := moVM.Snapshot; si != nil {
		snapshotTree = si.RootSnapshotList
	}
	return hasSnapshotTreeInner(snapshotTree)
}

func hasSnapshotTreeInner(nodes []vimtypes.VirtualMachineSnapshotTree) bool {
	switch len(nodes) {
	case 0:
		return false
	case 1:
		return hasSnapshotTreeInner(nodes[0].ChildSnapshotList)
	default:
		return true
	}
}
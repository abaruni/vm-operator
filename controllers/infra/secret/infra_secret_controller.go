// Copyright (c) 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package secret

import (
	goctx "context"
	"fmt"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	ctrl "sigs.k8s.io/controller-runtime"
	ctrlbuilder "sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	pkgconfig "github.com/vmware-tanzu/vm-operator/pkg/config"
	"github.com/vmware-tanzu/vm-operator/pkg/context"
	"github.com/vmware-tanzu/vm-operator/pkg/record"
)

const (
	// VcCredsSecretName is the credential secret that stores the VM operator service provider user credentials.
	VcCredsSecretName = "wcp-vmop-sa-vc-auth" //nolint:gosec
)

type provider interface {
	ResetVcClient(ctx goctx.Context)
}

// AddToManager adds this package's controller to the provided manager.
func AddToManager(ctx *context.ControllerManagerContext, mgr manager.Manager) error {
	var (
		controllerName      = "infra-secret"
		controllerNameShort = fmt.Sprintf("%s-controller", controllerName)
		controllerNameLong  = fmt.Sprintf("%s/%s/%s", ctx.Namespace, ctx.Name, controllerNameShort)
	)

	r := NewReconciler(
		ctx,
		mgr.GetClient(),
		ctrl.Log.WithName("controllers").WithName(controllerName),
		record.New(mgr.GetEventRecorderFor(controllerNameLong)),
		ctx.Namespace,
		ctx.VMProviderA2,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(controllerName).
		Watches(
			&corev1.Secret{},
			&handler.EnqueueRequestForObject{},
			ctrlbuilder.WithPredicates(
				predicate.Funcs{
					CreateFunc: func(e event.CreateEvent) bool {
						return e.Object.GetName() == VcCredsSecretName
					},
					UpdateFunc: func(e event.UpdateEvent) bool {
						return e.ObjectOld.GetName() == VcCredsSecretName
					},
					DeleteFunc: func(e event.DeleteEvent) bool {
						return false
					},
					GenericFunc: func(e event.GenericEvent) bool {
						return false
					},
				},
				predicate.ResourceVersionChangedPredicate{},
			)).
		Complete(r)
}

func NewReconciler(
	ctx goctx.Context,
	client client.Client,
	logger logr.Logger,
	recorder record.Recorder,
	vmOpNamespace string,
	provider provider) *Reconciler {
	return &Reconciler{
		Context:       ctx,
		Client:        client,
		Logger:        logger,
		Recorder:      recorder,
		vmOpNamespace: vmOpNamespace,
		provider:      provider,
	}
}

type Reconciler struct {
	client.Client
	Context       goctx.Context
	Logger        logr.Logger
	Recorder      record.Recorder
	vmOpNamespace string
	provider      provider
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *Reconciler) Reconcile(ctx goctx.Context, req ctrl.Request) (ctrl.Result, error) {
	ctx = pkgconfig.JoinContext(ctx, r.Context)

	// This is totally wrong and we should break this controller apart so we're not
	// watching different types.

	if req.Name == VcCredsSecretName && req.Namespace == r.vmOpNamespace {
		r.reconcileVcCreds(ctx, req)
		return ctrl.Result{}, nil
	}

	r.Logger.Error(nil, "Reconciling unexpected object", "req", req.NamespacedName)
	return ctrl.Result{}, nil
}

func (r *Reconciler) reconcileVcCreds(ctx goctx.Context, req ctrl.Request) {
	r.Logger.Info("Reconciling updated VM Operator credentials", "secret", req.NamespacedName)
	r.provider.ResetVcClient(ctx)
}

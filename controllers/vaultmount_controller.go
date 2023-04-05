// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/api"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	secretsv1alpha1 "github.com/hashicorp/vault-secrets-operator/api/v1alpha1"
	"github.com/hashicorp/vault-secrets-operator/internal/consts"
	"github.com/hashicorp/vault-secrets-operator/internal/vault"
)

const vaultMountFinalizer = "vaultmount.secrets.hashicorp.com/finalizer"

// VaultMountReconciler reconciles a VaultMount object
type VaultMountReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	ClientFactory vault.ClientFactory
}

//+kubebuilder:rbac:groups=secrets.hashicorp.com,resources=vaultmounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=secrets.hashicorp.com,resources=vaultmounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=secrets.hashicorp.com,resources=vaultmounts/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *VaultMountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	o := &secretsv1alpha1.VaultMount{}
	if err := r.Client.Get(ctx, req.NamespacedName, o); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error getting resource from k8s", "kv mount", o)
		return ctrl.Result{}, err
	}

	c, err := r.ClientFactory.Get(ctx, r.Client, o)
	if err != nil {
		r.Recorder.Eventf(o, corev1.EventTypeWarning, consts.ReasonVaultClientConfigError,
			"Failed to get Vault auth login: %s", err)
		return ctrl.Result{}, err
	}

	o.Status.Valid = false

	if o.GetDeletionTimestamp() == nil {
		if err := r.addFinalizer(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	} else {
		logger.Info("Got deletion timestamp", "obj", o)
		return r.handleFinalizer(ctx, c, o)
	}

	mounts, err := c.ListMounts(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}

	config := api.MountConfigInput{
		DefaultLeaseTTL:           o.Spec.Config.DefaultLeaseTTL,
		Description:               &o.Spec.Description,
		MaxLeaseTTL:               o.Spec.Config.MaxLeaseTTL,
		ForceNoCache:              o.Spec.Config.ForceNoCache,
		AuditNonHMACRequestKeys:   o.Spec.Config.AuditNonHMACRequestKeys,
		AuditNonHMACResponseKeys:  o.Spec.Config.AuditNonHMACResponseKeys,
		ListingVisibility:         o.Spec.Config.ListingVisibility,
		PassthroughRequestHeaders: o.Spec.Config.PassthroughRequestHeaders,
		AllowedResponseHeaders:    o.Spec.Config.AllowedResponseHeaders,
		Options:                   o.Spec.Options,
		// TokenType:                 o.Spec.C,
		// AllowedManagedKeys:        o.Spec.Config.AM,
		// PluginVersion:             o.Spec.Config.Pl,
		// UserLockoutConfig:         nil,
	}

	path := fmt.Sprintf("%s/", o.Spec.Path)

	if _, ok := mounts[path]; ok {
		err = c.TuneMount(ctx, o.Spec.Path, config)

		if err != nil {
			return ctrl.Result{}, err
		}
	} else {
		err = c.Mount(ctx, o.Spec.Path, &api.MountInput{
			Type:                  o.Spec.Type,
			Description:           o.Spec.Description,
			Config:                config,
			Local:                 o.Spec.Local,
			SealWrap:              o.Spec.SealWrap,
			ExternalEntropyAccess: o.Spec.ExternalEntropyAccess,
			Options:               o.Spec.Options,
		})

		if err != nil {
			return ctrl.Result{}, err
		}
	}

	if o.Status.Path != "" && o.Status.Path != o.Spec.Path {
		if _, err := c.Delete(ctx, fmt.Sprintf("/sys/mounts/%s", o.Status.Path)); err != nil {
			return ctrl.Result{}, err
		}
	}

	o.Status.Valid = true
	o.Status.Path = o.Spec.Path

	if err := r.Status().Update(ctx, o); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VaultMountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1alpha1.VaultMount{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *VaultMountReconciler) addFinalizer(ctx context.Context, o *secretsv1alpha1.VaultMount) error {
	if !controllerutil.ContainsFinalizer(o, vaultMountFinalizer) {
		controllerutil.AddFinalizer(o, vaultMountFinalizer)
		if err := r.Client.Update(ctx, o); err != nil {
			return err
		}
	}

	return nil
}

func (r *VaultMountReconciler) handleFinalizer(ctx context.Context, c vault.Client, o *secretsv1alpha1.VaultMount) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(o, vaultMountFinalizer) {
		if _, err := c.Delete(ctx, o.Spec.Path); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(o, vaultMountFinalizer)
		if err := r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

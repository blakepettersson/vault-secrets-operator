// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"fmt"

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

const vaultKVMountFinalizer = "vaultkvmount.secrets.hashicorp.com/finalizer"

// VaultKVMountReconciler reconciles a VaultKVMount object
type VaultKVMountReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	ClientFactory vault.ClientFactory
}

//+kubebuilder:rbac:groups=secrets.hashicorp.com,resources=vaultkvmounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=secrets.hashicorp.com,resources=vaultkvmounts/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=secrets.hashicorp.com,resources=vaultkvmounts/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *VaultKVMountReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	o := &secretsv1alpha1.VaultKVMount{}
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

	path, err := c.Read(ctx, fmt.Sprintf("/sys/mounts/%s", o.Spec.Path))
	if err != nil {
		return ctrl.Result{}, err
	}

	if path == nil {
		_, err = c.Write(ctx, fmt.Sprintf("/sys/mounts/%s", o.Spec.Path), map[string]interface{}{
			"description": o.Spec.Description,
			"type":        o.Spec.Type,
			"config": map[string]interface{}{
				"default_lease_ttl":            o.Spec.Config.DefaultLeaseTTL,
				"max_lease_ttl":                o.Spec.Config.MaxLeaseTTL,
				"force_no_cache":               o.Spec.Config.ForceNoCache,
				"audit_non_hmac_request_keys":  o.Spec.Config.AuditNonHMACRequestKeys,
				"audit_non_hmac_response_keys": o.Spec.Config.AuditNonHMACResponseKeys,
				"listing_visibility":           o.Spec.Config.ListingVisibility,
				"passthrough_request_headers":  o.Spec.Config.PassthroughRequestHeaders,
				"allowed_response_headers":     o.Spec.Config.AllowedResponseHeaders,
				"local":                        o.Spec.Local,
				"seal_wrap":                    o.Spec.SealWrap,
				"external_entropy_access":      o.Spec.ExternalEntropyAccess,
			},
		})

		if err != nil {
			return ctrl.Result{}, err
		}

		if o.Status.Path != "" && o.Status.Path != o.Spec.Path {
			if _, err := c.Delete(ctx, fmt.Sprintf("/sys/mounts/%s", o.Status.Path)); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	o.Status.Valid = true
	o.Status.Path = o.Spec.Path

	if err := r.Status().Update(ctx, o); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VaultKVMountReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1alpha1.VaultKVMount{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *VaultKVMountReconciler) addFinalizer(ctx context.Context, o *secretsv1alpha1.VaultKVMount) error {
	if !controllerutil.ContainsFinalizer(o, vaultKVMountFinalizer) {
		controllerutil.AddFinalizer(o, vaultKVMountFinalizer)
		if err := r.Client.Update(ctx, o); err != nil {
			return err
		}
	}

	return nil
}

func (r *VaultKVMountReconciler) handleFinalizer(ctx context.Context, c vault.Client, o *secretsv1alpha1.VaultKVMount) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(o, vaultKVMountFinalizer) {
		if _, err := c.Delete(ctx, o.Spec.Path); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(o, vaultKVMountFinalizer)
		if err := r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

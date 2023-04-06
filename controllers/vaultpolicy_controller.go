// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"context"
	"encoding/json"
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

const vaultPolicyFinalizer = "vaultpolicy.secrets.hashicorp.com/finalizer"

// VaultPolicyReconciler reconciles a VaultPolicy object
type VaultPolicyReconciler struct {
	client.Client
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	ClientFactory vault.ClientFactory
}

type vaultPolicyPath struct {
	Capabilities       []string            `json:"capabilities,omitempty"`
	AllowedParameters  map[string][]string `json:"allowed_parameters,omitempty"`
	DeniedParameters   map[string][]string `json:"denied_parameters,omitempty"`
	RequiredParameters []string            `json:"required_parameters,omitempty"`
	MinWrappingTTL     string              `json:"min_wrapping_ttl,omitempty"`
	MaxWrappingTTL     string              `json:"max_wrapping_ttl,omitempty"`
}

//+kubebuilder:rbac:groups=secrets.hashicorp.com,resources=vaultpolicies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=secrets.hashicorp.com,resources=vaultpolicies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=secrets.hashicorp.com,resources=vaultpolicies/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=events,verbs=create;patch

func (r *VaultPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	o := &secretsv1alpha1.VaultPolicy{}
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

	path := map[string]vaultPolicyPath{}

	for k, v := range o.Spec.Path {
		path[k] = vaultPolicyPath{
			Capabilities:       v.Capabilities,
			AllowedParameters:  v.AllowedParameters,
			DeniedParameters:   v.DeniedParameters,
			RequiredParameters: v.RequiredParameters,
			MinWrappingTTL:     v.MinWrappingTTL,
			MaxWrappingTTL:     v.MaxWrappingTTL,
		}
	}

	policy, err := json.Marshal(map[string]any{
		"path": path,
	})
	if err != nil {
		return ctrl.Result{}, err
	}

	if _, err := c.Write(ctx, fmt.Sprintf("/sys/policies/acl/%s", o.Name), map[string]any{
		"policy": string(policy),
	}); err != nil {
		return ctrl.Result{}, err
	}

	if o.Status.Name != "" && o.Status.Name != o.Name {
		if _, err := c.Delete(ctx, fmt.Sprintf("/sys/policies/acl/%s", o.Status.Name)); err != nil {
			return ctrl.Result{}, err
		}
	}

	o.Status.Valid = true
	o.Status.Name = o.Name

	if err := r.Status().Update(ctx, o); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *VaultPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretsv1alpha1.VaultPolicy{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}

func (r *VaultPolicyReconciler) addFinalizer(ctx context.Context, o *secretsv1alpha1.VaultPolicy) error {
	if !controllerutil.ContainsFinalizer(o, vaultPolicyFinalizer) {
		controllerutil.AddFinalizer(o, vaultPolicyFinalizer)
		if err := r.Client.Update(ctx, o); err != nil {
			return err
		}
	}

	return nil
}

func (r *VaultPolicyReconciler) handleFinalizer(ctx context.Context, c vault.Client, o *secretsv1alpha1.VaultPolicy) (ctrl.Result, error) {
	if controllerutil.ContainsFinalizer(o, vaultPolicyFinalizer) {
		if _, err := c.Delete(ctx, fmt.Sprintf("/sys/policies/acl/%s", o.Name)); err != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(o, vaultPolicyFinalizer)
		if err := r.Update(ctx, o); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

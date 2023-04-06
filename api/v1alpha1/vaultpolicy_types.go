// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VaultPolicySpec defines the desired state of VaultPolicySpec
type VaultPolicySpec struct {
	// VaultAuthRef of the VaultAuth resource
	// If no value is specified the Operator will default to the `default` VaultAuth,
	// configured in its own Kubernetes namespace.
	VaultAuthRef string `json:"vaultAuthRef,omitempty"`

	// Namespace to auth to in Vault
	Namespace string `json:"namespace,omitempty"`

	// Mount to use when authenticating to auth method.
	// +kubebuilder:validation:Required
	Path map[string]VaultPolicyPath `json:"path"`
}

type VaultPolicyPath struct {
	// +kubebuilder:validation:Required
	Capabilities []string `json:"capabilities,omitempty"`
	// +kubebuilder:validation:Optional
	AllowedParameters map[string][]string `json:"allowedParameters,omitempty"`
	// +kubebuilder:validation:Optional
	DeniedParameters map[string][]string `json:"deniedParameters,omitempty"`
	// +kubebuilder:validation:Optional
	RequiredParameters []string `json:"requiredParameters,omitempty"`
	// +kubebuilder:validation:Optional
	MinWrappingTTL string `json:"minWrappingTTL,omitempty"`
	// +kubebuilder:validation:Optional
	MaxWrappingTTL string `json:"maxWrappingTTL,omitempty"`
}

// VaultPolicyStatus defines the observed state of VaultPolicySpec
type VaultPolicyStatus struct {
	// Valid auth mechanism.
	Valid bool   `json:"valid"`
	Error string `json:"error"`
	Name  string `json:"name"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VaultPolicy is the Schema for the vaultmounts API
type VaultPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultPolicySpec   `json:"spec,omitempty"`
	Status VaultPolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VaultPolicyList contains a list of VaultPolicy
type VaultPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VaultPolicy{}, &VaultPolicyList{})
}

// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VaultMountSpec defines the desired state of VaultMountSpec
type VaultMountSpec struct {
	// VaultAuthRef of the VaultAuth resource
	// If no value is specified the Operator will default to the `default` VaultAuth,
	// configured in its own Kubernetes namespace.
	VaultAuthRef string `json:"vaultAuthRef,omitempty"`

	// Namespace to auth to in Vault
	Namespace string `json:"namespace,omitempty"`

	// Mount to use when authenticating to auth method.
	Path string `json:"path"`

	// Description Specifies the human-friendly description of the mount.
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// Specifies configuration options for this mount; if set on a specific mount, values will override any global defaults (e.g. the system TTL/Max TTL)
	// +kubebuilder:validation:Optional
	Config MountConfig `json:"config,omitempty"`

	// Local Specifies if the secrets engine is a local mount only. Local mounts are not replicated nor (if a secondary) removed by replication.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	Local bool `json:"local,omitempty"`

	// SealWrap Enable seal wrapping for the mount, causing values stored by the mount to be wrapped by the seal's encryption capability.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	SealWrap bool `json:"sealWrap,omitempty"`

	// ExternalEntropyAccess Enable the secrets engine to access Vault's external entropy source.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	ExternalEntropyAccess bool `json:"externalEntropyAccess,omitempty"`

	// Options Specifies mount type specific options that are passed to the backend.
	// +kubebuilder:validation:Optional
	// +mapType=granular
	Options map[string]string `json:"options,omitempty"`
}

type MountConfig struct {
	// DefaultLeaseTTL  The default lease duration, specified as a string duration like "5s" or "30m".
	// +kubebuilder:validation:Optional
	DefaultLeaseTTL string `json:"defaultLeaseTTL"`

	// MaxLeaseTTL The maximum lease duration, specified as a string duration like "5s" or "30m".
	// +kubebuilder:validation:Optional
	MaxLeaseTTL string `json:"maxLeaseTTL"`

	// ForceNoCache Disable caching.
	// +kubebuilder:validation:Optional
	// +kubebuilder:default:=false
	ForceNoCache bool `json:"forceNoCache"`

	// AuditNonHMACRequestKeys list of keys that will not be HMAC'd by audit devices in the request data object.
	// +kubebuilder:validation:Optional
	// +listType=set
	// kubebuilder:validation:UniqueItems=true
	AuditNonHMACRequestKeys []string `json:"auditNonHMACRequestKeys,omitempty"`

	// AuditNonHMACResponseKeys list of keys that will not be HMAC'd by audit devices in the response data object.
	// +kubebuilder:validation:Optional
	// +listType=set
	// kubebuilder:validation:UniqueItems=true
	AuditNonHMACResponseKeys []string `json:"auditNonHMACResponseKeys,omitempty"`

	// ListingVisibility Specifies whether to show this mount in the UI-specific listing endpoint. Valid values are "unauth" or "hidden". If not set, behaves like "hidden"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:={"unauth","hidden"}
	// +kubebuilder:default:="hidden"
	ListingVisibility string `json:"listingVisibility,omitempty"`

	// PassthroughRequestHeaders list of headers to whitelist and pass from the request to the plugin.
	// +kubebuilder:validation:Optional
	// +listType=set
	// kubebuilder:validation:UniqueItems=true
	PassthroughRequestHeaders []string `json:"passthroughRequestHeaders,omitempty"`

	// AllowedResponseHeaders list of headers to whitelist, allowing a plugin to include them in the response.
	// +kubebuilder:validation:Optional
	// +listType=set
	// kubebuilder:validation:UniqueItems=true
	AllowedResponseHeaders []string `json:"allowedResponseHeaders,omitempty"`
}

type VaultKVMountSpec struct {
	VaultMountSpec `json:",inline"`

	// Type Specifies the type of the backend, such as "aws".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum={1,2}
	Version string `json:"version"`

	// Type Specifies the type of the backend, such as "aws".
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum={kv}
	// +kubebuilder:default="kv"
	Type string `json:"type"`
}

// VaultMountStatus defines the observed state of VaultMountSpec
type VaultMountStatus struct {
	// Valid auth mechanism.
	Valid bool   `json:"valid"`
	Error string `json:"error"`
	Path  string `json:"path"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// VaultMount is the Schema for the vaultkvmounts API
type VaultKVMount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VaultKVMountSpec `json:"spec,omitempty"`
	Status VaultMountStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// VaultMountList contains a list of VaultKVMount
type VaultKVMountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VaultKVMount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VaultKVMount{}, &VaultKVMountList{})
}

// Package v1alpha1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=claude.fn.upbound.io
// +versionName=v1alpha1
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// This isn't a custom resource, in the sense that we never install its CRD.
// It is a KRM-like object, so we generate a CRD to describe its schema.

// TODO: Add your input type here! It doesn't need to be called 'Input', you can
// rename it to anything you like.

// Prompt can be used to provide input to this Function.
// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:resource:categories=crossplane
type Prompt struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// SytemPrompt to send to Claude.
	SystemPrompt string `json:"systemPrompt"`
	// UserPrompt to send to Claude.
	UserPrompt string `json:"userPrompt"`
}

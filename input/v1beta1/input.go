// Package v1beta1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=claude.fn.crossplane.io
// +versionName=v1beta1
package v1beta1

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

	// Prompt to send to Claude.
	Prompt string `json:"prompt"`
	
	// ContextFields is a list of context field names to include in the prompt
	// (e.g., ["metricsResult", "otherData"] to access context.metricsResult and context.otherData)
	// +optional
	ContextFields []string `json:"contextFields,omitempty"`
}

// /*
// Copyright 2025 The Upbound Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */

// Package v1alpha1 contains the input type for this Function
// +kubebuilder:object:generate=true
// +groupName=claude.fn.upbound.io
// +versionName=v1alpha1
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	// ModelName is the optional Anthropic model name to use (e.g., "claude-sonnet-4-5-20250929").
	// If not specified, the default model will be used.
	// See https://docs.claude.com/en/docs/about-claude/models/overview for available models.
	// +optional
	ModelName string `json:"modelName,omitempty"`
}

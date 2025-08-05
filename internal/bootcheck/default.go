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

//go:build !custombootcheck
// +build !custombootcheck

// Package bootcheck supports mutually exclusive implementations of
// custom preflight checks for the composition function.
package bootcheck

// CheckEnv performs environment validation checks.
// In the default build configuration, this is a no-op implementation.
func CheckEnv() error {
	// No-op by default. Use build tags for build-time isolation of custom preflight checks.
	return nil
}

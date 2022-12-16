/*
 * Copyright 2022.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import corev1 "k8s.io/api/core/v1"

// ConsoleSpec defines the desired state of Console
// +k8s:openapi-gen=true
type ConsoleSpec struct {
	// +kubebuilder:validation:Required
	// Name of the console
	Name string `json:"name"`
	// Image is the image of the console
	Image string `json:"image"`
	// ImagePullPolicy defines how the image is pulled
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// The secrets used to pull image from private registry
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Affinity the pod's scheduling constraints
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Tolerations the pod's tolerations.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// NodeSelector is a selector which must be true for the pod to fit on a node
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// Resources describes the compute resource requirements
	Resources corev1.ResourceRequirements `json:"resources"`
}

// ConsoleStatus defines the observed state of Console
// +k8s:openapi-gen=true
type ConsoleStatus struct {
	Status ConditionStatus `json:"status,omitempty"`
}

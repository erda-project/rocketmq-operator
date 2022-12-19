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

// BrokerSpec defines the desired state of Broker
// +k8s:openapi-gen=true
type BrokerSpec struct {
	// +kubebuilder:validation:Required
	// Name of the broker
	Name string `json:"name"`
	// Size of broker clusters
	Size int32 `json:"size"`
	// Image is the broker image to use for the Pods
	Image string `json:"image"`
	// ImagePullPolicy defines how the image is pulled
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// HostNetwork can be true or false
	HostNetwork bool `json:"hostNetwork,omitempty"`
	// Resources describes the compute resource requirements
	Resources corev1.ResourceRequirements `json:"resources"`
	// StorageMode can be EmptyDir, HostPath, StorageClass
	StorageMode string `json:"storageMode"`
	// HostPath is the local path to store data
	HostPath string `json:"hostPath,omitempty"`
	// Env defines custom env, e.g. BROKER_MEM
	Env []corev1.EnvVar `json:"env,omitempty"`
	// Volumes define the broker.conf
	Volumes []corev1.Volume `json:"volumes,omitempty"`
	// VolumeClaimTemplates defines the StorageClass
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates"`
	// The name of pod where the metadata from
	ScalePodName string `json:"scalePodName,omitempty"`
	// Pod Security Context
	PodSecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`
	// Container Security Context
	ContainerSecurityContext *corev1.SecurityContext `json:"containerSecurityContext,omitempty"`
	// The secrets used to pull image from private registry
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Affinity the pod's scheduling constraints
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Tolerations the pod's tolerations.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// NodeSelector is a selector which must be true for the pod to fit on a node
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// PriorityClassName indicates the pod's priority
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// ServiceAccountName
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
	// Labels
	Labels map[string]string `json:"labels,omitempty"`
}

// BrokerStatus defines the observed state of Broker
// +k8s:openapi-gen=true
type BrokerStatus struct {
	// Running is the number of pods which reached phase Running
	Running int32 `json:"running,omitempty"`

	// Status describes the status of the broker
	Status ConditionStatus `json:"status,omitempty"`
}

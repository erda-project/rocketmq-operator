/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConditionStatus string

const (
	ConditionInitializing ConditionStatus = "Initializing"
	ConditionReady        ConditionStatus = "Ready"
	ConditionFailed       ConditionStatus = "Failed"
)

// RocketMQSpec defines the desired state of RocketMQ
type RocketMQSpec struct {
	// +kubebuilder:validation:Required
	// NameServiceSpec defines the name service spec
	NameServiceSpec NameServiceSpec `json:"nameServiceSpec,omitempty"`

	// +kubebuilder:validation:Required
	// BrokerSpec defines the broker spec
	BrokerSpec BrokerSpec `json:"brokerSpec,omitempty"`

	// ConsoleSpec defines the console spec
	ConsoleSpec ConsoleSpec `json:"consoleSpec,omitempty"`
}

// RocketMQStatus defines the observed state of RocketMQ
type RocketMQStatus struct {
	// NameServiceStatus defines the name service status
	NameServiceStatus NameServiceStatus `json:"nameServiceStatus,omitempty"`

	// BrokerStatus defines the broker status
	BrokerStatus BrokerStatus `json:"brokerStatus,omitempty"`

	// ConsoleStatus defines the console status
	ConsoleStatus ConsoleStatus `json:"consoleStatus,omitempty"`

	// Conditions defines the conditions of the RocketMQ
	ConditionStatus ConditionStatus `json:"conditionStatus,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditionStatus"

// RocketMQ is the Schema for the rocketmqs API
type RocketMQ struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RocketMQSpec   `json:"spec,omitempty"`
	Status RocketMQStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// RocketMQList contains a list of RocketMQ
type RocketMQList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RocketMQ `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RocketMQ{}, &RocketMQList{})
}

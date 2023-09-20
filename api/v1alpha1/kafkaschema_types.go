/*
Copyright 2023.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// KafkaSchemaSpec defines the desired state of KafkaSchema
type KafkaSchemaSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Kafka schema
	// +kubebuilder:validation:Required
	Schema string `json:"schema"`
	// Schema name
	Subject string `json:"subject,omitempty"`
	// Schema references
	References []Reference `json:"references,omitempty"`
}

type StatusPhase string

const (
	CreatedStatusPhase StatusPhase = "CREATED"
	PendingStatusPhase StatusPhase = "PENDING"
	ErrorStatusPhase   StatusPhase = "ERROR"
)

// KafkaSchemaStatus defines the observed state of KafkaSchema
type KafkaSchemaStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Status of schema
	Phase StatusPhase `json:"phase"`
	// Conditions of schema
	Conditions []metav1.Condition `json:"conditions"`
	// ID of schema in registry
	ID int `json:"id,omitempty"`
	// Version of schema in registry
	Version int `json:"version,omitempty"`
	// Schema name
	Subject string `json:"subject,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="ID",type=integer,JSONPath=`.status.id`
//+kubebuilder:printcolumn:name="Version",type=integer,JSONPath=`.status.version`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:resource:shortName="ks"

// KafkaSchema is the Schema for the kafkaschemas API
type KafkaSchema struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec   KafkaSchemaSpec   `json:"spec"`
	Status KafkaSchemaStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// KafkaSchemaList contains a list of KafkaSchema
type KafkaSchemaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KafkaSchema `json:"items"`
}

type Reference struct {
	// Subject of schema
	// +kubebuilder:validation:Required
	Subject string `json:"subject"`
	// Version of schema
	Version int `json:"version,omitempty"`
}

func init() {
	SchemeBuilder.Register(&KafkaSchema{}, &KafkaSchemaList{})
}

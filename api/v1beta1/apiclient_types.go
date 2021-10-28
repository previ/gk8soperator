/*
Copyright 2021.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type APISubscription struct {
	APIContextPath string `json:"api_context_path,omitempty"`
	APIPlanName    string `json:"api_plan_name,omitempty"`
}

// APIClientSpec defines the desired state of APIClient
type APIClientSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Name of the Client App
	Name string `json:"name,omitempty"`

	// Description of the Client App
	Description string `json:"description,omitempty"`

	// Type of the Client App
	Type string `json:"type,omitempty"`

	// Type of the Client App
	ClientID string `json:"client_id,omitempty"`

	// API subscription
	APISubscriptions []APISubscription `json:"api_subscriptions,omitempty"`
}

// APIClientStatus defines the observed state of APIClient
type APIClientStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Application's uuid.
	// Example: 00f8c9e7-78fc-4907-b8c9-e778fc790750
	ID string `json:"id,omitempty"`

	// The last date (as a timestamp) when the Application was updated.
	// Example: 1581256457163
	UpdatedAt int64 `json:"updated_at,omitempty"`

	// The last reconcyled generation.
	// Example: 1
	UpdatedGeneration int64 `json:"updated_generation,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// APIClient is the Schema for the apiclients API
type APIClient struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   APIClientSpec   `json:"spec,omitempty"`
	Status APIClientStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// APIClientList contains a list of APIClient
type APIClientList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []APIClient `json:"items"`
}

func init() {
	SchemeBuilder.Register(&APIClient{}, &APIClientList{})
}

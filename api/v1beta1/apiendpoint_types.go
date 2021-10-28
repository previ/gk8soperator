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

// swagger:model Cors
type Cors struct {

	// access control allow credentials
	AllowCredentials bool `json:"allowCredentials,omitempty"`

	// access control allow headers
	// Unique: true
	AllowHeaders []string `json:"allowHeaders"`

	// access control allow methods
	// Unique: true
	AllowMethods []string `json:"allowMethods"`

	// access control allow origin
	// Unique: true
	AllowOrigin []string `json:"allowOrigin"`

	// access control allow origin regex
	// Unique: true
	AllowOriginRegex []string `json:"allowOriginRegex"`

	// access control expose headers
	// Unique: true
	ExposeHeaders []string `json:"exposeHeaders"`

	// access control max age
	MaxAge int32 `json:"maxAge,omitempty"`

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// error status code
	ErrorStatusCode int32 `json:"errorStatusCode,omitempty"`

	// run policies
	RunPolicies bool `json:"runPolicies,omitempty"`
}

// Rule rule
//
// swagger:model Rule
type Rule struct {

	// description
	Description string `json:"description,omitempty"`

	// enabled
	Enabled bool `json:"enabled,omitempty"`

	// methods
	// Unique: true
	Methods []string `json:"methods"`

	// policy
	Policy *Policy `json:"policy,omitempty"`
}

// Policy policy
//
// swagger:model Policy
type Policy struct {

	// configuration
	Configuration string `json:"configuration,omitempty"`

	// name
	Name string `json:"name,omitempty"`
}

// Path path
//
// swagger:model Path
type Path struct {

	// path
	Path string `json:"path,omitempty"`

	// rules
	Rules []*Rule `json:"rules"`
}

type Plan struct {
	// description
	// Required: true
	Description string `json:"description"`

	// name
	// Required: true
	Name *string `json:"name"`

	// paths
	// Required: true
	Paths map[string]*Path `json:"paths"`

	// security
	// Required: true
	// Enum: [KEY_LESS API_KEY OAUTH2 JWT]
	Security *string `json:"security"`

	// security definition
	SecurityDefinition map[string]string `json:"securityDefinition,omitempty"`

	// tags
	// Unique: true
	Tags []string `json:"tags"`
}

// APIEndpointSpec defines the desired state of APIEndpoint
type APIEndpointSpec struct {

	// API's name. Duplicate names can exists.
	// Example: My API
	Name string `json:"name,omitempty"`

	// API's version
	Version string `json:"version,omitempty"`

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// API's context path.
	// Example: /my-awesome-api
	ContextPath string `json:"context_path,omitempty"`

	// API's description. A short description of your API.
	// Example: I can use a hundred characters to describe this API.
	Description string `json:"description,omitempty"`

	// target URI (mutually exclusive with TargetService property)
	Target string `json:"target,omitempty"`

	// target service name
	TargetService string `json:"target_service,omitempty"`

	// CORS
	Cors *Cors `json:"cors,omitempty"`

	// Plans
	Plans []*Plan `json:"plans"`

	// The status of the API regarding the gateway.
	// Example: STARTED
	// Enum: [INITIALIZED STOPPED STARTED CLOSED]
	State string `json:"state,omitempty"`

	// the list of sharding tags associated with this API.
	// Example: default, internet
	// Unique: true
	Tags []string `json:"tags"`

	// The visibility of the API regarding the portal.
	// Example: PUBLIC
	// Enum: [PUBLIC PRIVATE]
	Visibility string `json:"visibility,omitempty"`
}

// APIEndpointStatus defines the observed state of APIEndpoint
type APIEndpointStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// API's uuid.
	// Example: 00f8c9e7-78fc-4907-b8c9-e778fc790750
	ID string `json:"id,omitempty"`

	// The last date (as a timestamp) when the API was updated.
	// Example: 1581256457163
	UpdatedAt int64 `json:"updated_at,omitempty"`

	// The last reconcyled generation.
	// Example: 1
	UpdatedGeneration int64 `json:"updated_generation,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// APIEndpoint is the Schema for the api endpoint API
type APIEndpoint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   APIEndpointSpec   `json:"spec,omitempty"`
	Status APIEndpointStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// APIEndpointList contains a list of APIEndpoint
type APIEndpointList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []APIEndpoint `json:"items"`
}

func init() {
	SchemeBuilder.Register(&APIEndpoint{}, &APIEndpointList{})
}

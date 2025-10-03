/*
Copyright 2025 Keith McClellan

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BrokerSpec defines the desired state of Broker
type BrokerSpec struct {
	// Endpoint is the base URL of the broker API
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:Pattern=`^https?://.*`
	Endpoint string `json:"endpoint"`

	// Region is the cloud region this broker manages (e.g., "eastus", "us-west-2")
	// +optional
	Region string `json:"region,omitempty"`

	// CloudProvider specifies the cloud provider (azure, aws, gcp, on-prem)
	// +kubebuilder:validation:Enum=azure;aws;gcp;on-prem
	CloudProvider string `json:"cloudProvider"`

	// Capabilities describes what resources this broker can provision
	Capabilities []BrokerCapability `json:"capabilities"`

	// Authentication configuration for broker communication
	// +optional
	Authentication *BrokerAuthentication `json:"authentication,omitempty"`

	// HealthCheck configuration
	// +optional
	HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`

	// Priority for broker selection (higher number = higher priority)
	// +optional
	// +kubebuilder:default=100
	Priority int32 `json:"priority,omitempty"`

	// MaxConcurrentDeployments limits parallel deployments
	// +optional
	// +kubebuilder:default=10
	MaxConcurrentDeployments int32 `json:"maxConcurrentDeployments,omitempty"`
}

// BrokerCapability describes a resource type the broker can provision
type BrokerCapability struct {
	// ResourceType is the kind of resource (Database, Cache, Topic, etc.)
	ResourceType string `json:"resourceType"`

	// Providers lists the specific implementations available (e.g., ["postgresql", "mysql"])
	// +optional
	Providers []string `json:"providers,omitempty"`

	// Regions lists specific regions where this capability is available
	// +optional
	Regions []string `json:"regions,omitempty"`
}

// BrokerAuthentication defines how to authenticate with the broker
type BrokerAuthentication struct {
	// Type of authentication (jwt, mtls, api-key)
	// +kubebuilder:validation:Enum=jwt;mtls;api-key;none
	// +kubebuilder:default=jwt
	Type string `json:"type"`

	// SecretRef references a secret containing authentication credentials
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// HealthCheckConfig defines health check parameters
type HealthCheckConfig struct {
	// Endpoint is the health check path (e.g., "/health")
	// +kubebuilder:default="/health"
	Endpoint string `json:"endpoint,omitempty"`

	// IntervalSeconds is how often to check health
	// +kubebuilder:default=30
	// +kubebuilder:validation:Minimum=5
	IntervalSeconds int32 `json:"intervalSeconds,omitempty"`

	// TimeoutSeconds is the health check timeout
	// +kubebuilder:default=5
	// +kubebuilder:validation:Minimum=1
	TimeoutSeconds int32 `json:"timeoutSeconds,omitempty"`

	// FailureThreshold is consecutive failures before marking unhealthy
	// +kubebuilder:default=3
	// +kubebuilder:validation:Minimum=1
	FailureThreshold int32 `json:"failureThreshold,omitempty"`
}

// BrokerStatus defines the observed state of Broker
type BrokerStatus struct {
	// Phase represents the current state of the broker
	// +kubebuilder:validation:Enum=Pending;Ready;Unhealthy;Offline;Unknown
	Phase string `json:"phase,omitempty"`

	// LastHeartbeat is the timestamp of the last successful health check
	// +optional
	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`

	// ActiveDeployments is the current number of active deployments
	// +optional
	ActiveDeployments int32 `json:"activeDeployments,omitempty"`

	// Version is the broker software version
	// +optional
	Version string `json:"version,omitempty"`

	// Conditions represent the latest observations of the broker's state
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ObservedGeneration reflects the generation most recently observed by the controller
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Message provides additional context about the current phase
	// +optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=br
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.cloudProvider`
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Active",type=integer,JSONPath=`.status.activeDeployments`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Broker is the Schema for the brokers API
type Broker struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BrokerSpec   `json:"spec,omitempty"`
	Status BrokerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// BrokerList contains a list of Broker
type BrokerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Broker `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Broker{}, &BrokerList{})
}

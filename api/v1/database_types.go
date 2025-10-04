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

// DatabaseSpec defines the desired state of Database
type DatabaseSpec struct {
	// Owner reference to the owning Application or Team
	Owner OwnerReference `json:"owner"`

	// Engine specifies the database engine (postgresql, mysql, mongodb, etc.)
	// +kubebuilder:validation:Enum=postgresql;mysql;mongodb;redis;sqlserver
	Engine string `json:"engine"`

	// Version specifies the engine version
	// +kubebuilder:validation:MinLength=1
	Version string `json:"version"`

	// Size specifies the instance size
	// +kubebuilder:validation:Enum=small;medium;large;xlarge
	Size string `json:"size"`

	// Target specifies where to deploy (e.g., azure-westus2-prod)
	// +optional
	Target string `json:"target,omitempty"`

	// Backup configuration
	// +optional
	Backup *BackupConfig `json:"backup,omitempty"`

	// Encryption configuration
	// +optional
	Encryption *EncryptionConfig `json:"encryption,omitempty"`

	// HighAvailability enables HA configuration
	// +optional
	HighAvailability bool `json:"highAvailability,omitempty"`

	// Parameters for database-specific configuration
	// +optional
	Parameters map[string]string `json:"parameters,omitempty"`
}

// OwnerReference points to the owning resource
type OwnerReference struct {
	// Kind of the owner (Team, Application)
	// +kubebuilder:validation:Enum=Team;Application
	Kind string `json:"kind"`

	// Name of the owner
	Name string `json:"name"`

	// Namespace of the owner (if namespaced)
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// BackupConfig defines backup settings
type BackupConfig struct {
	// Enabled determines if backups are enabled
	Enabled bool `json:"enabled"`

	// Retention period (e.g., "7d", "30d")
	// +kubebuilder:validation:Pattern=`^\d+[dhm]$`
	Retention string `json:"retention"`

	// Schedule in cron format
	// +optional
	Schedule string `json:"schedule,omitempty"`

	// PointInTimeRestore enables PITR
	// +optional
	PointInTimeRestore bool `json:"pointInTimeRestore,omitempty"`
}

// EncryptionConfig defines encryption settings
type EncryptionConfig struct {
	// AtRest encryption configuration
	AtRest AtRestEncryption `json:"atRest"`

	// InTransit encryption configuration
	InTransit InTransitEncryption `json:"inTransit"`
}

// AtRestEncryption configures encryption at rest
type AtRestEncryption struct {
	// Enabled determines if encryption at rest is enabled
	Enabled bool `json:"enabled"`

	// KMSKeyID specifies the KMS key (optional, uses default if not specified)
	// +optional
	KMSKeyID string `json:"kmsKeyId,omitempty"`
}

// InTransitEncryption configures encryption in transit
type InTransitEncryption struct {
	// Enabled determines if TLS is required
	Enabled bool `json:"enabled"`

	// MinTLSVersion specifies minimum TLS version
	// +kubebuilder:validation:Enum=1.2;1.3
	// +optional
	MinTLSVersion string `json:"minTLSVersion,omitempty"`
}

// DatabaseStatus defines the observed state of Database
type DatabaseStatus struct {
	// Phase represents the current state
	// +kubebuilder:validation:Enum=Pending;Provisioning;Ready;Failed;Deleting
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Endpoint is the connection endpoint
	// +optional
	Endpoint string `json:"endpoint,omitempty"`

	// Port is the connection port
	// +optional
	Port int32 `json:"port,omitempty"`

	// ConnectionSecretRef references the secret containing connection details
	// +optional
	ConnectionSecretRef *SecretReference `json:"connectionSecretRef,omitempty"`

	// CloudResourceID is the cloud provider's resource identifier
	// +optional
	CloudResourceID string `json:"cloudResourceId,omitempty"`

	// DeploymentID from the broker
	// +optional
	DeploymentID string `json:"deploymentId,omitempty"`

	// BrokerRef references the Broker CR that handled this deployment
	// +optional
	BrokerRef *ObjectReference `json:"brokerRef,omitempty"`

	// Cost information
	// +optional
	Cost *CostInfo `json:"cost,omitempty"`

	// LastBackup timestamp
	// +optional
	LastBackup *metav1.Time `json:"lastBackup,omitempty"`

	// ObservedGeneration reflects the generation most recently observed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// SecretReference points to a Kubernetes secret
type SecretReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// CostInfo tracks resource costs
type CostInfo struct {
	// EstimatedMonthly in USD
	EstimatedMonthly float64 `json:"estimatedMonthly"`

	// Currency code
	Currency string `json:"currency"`

	// LastUpdated timestamp
	LastUpdated metav1.Time `json:"lastUpdated"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.engine`
// +kubebuilder:printcolumn:name="Version",type=string,JSONPath=`.spec.version`
// +kubebuilder:printcolumn:name="Size",type=string,JSONPath=`.spec.size`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Endpoint",type=string,JSONPath=`.status.endpoint`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Database is the Schema for the databases API
type Database struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DatabaseSpec   `json:"spec,omitempty"`
	Status DatabaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DatabaseList contains a list of Database
type DatabaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Database `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Database{}, &DatabaseList{})
}

// ObjectReference is a reference to another Kubernetes object
type ObjectReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

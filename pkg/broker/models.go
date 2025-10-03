package broker

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

import (
	"fmt"
	"time"
)

// ProvisionRequest represents a request to provision a resource
type ProvisionRequest struct {
	// Resource identification
	ResourceType string `json:"resourceType"` // database, cache, topic, etc.
	ResourceName string `json:"resourceName"`
	Namespace    string `json:"namespace"`

	// Ownership
	Team  string `json:"team"`
	Owner string `json:"owner"` // User who created the resource

	// Callback configuration
	CallbackURL string `json:"callbackUrl"` // URL to POST status updates

	// Resource specification
	Spec map[string]interface{} `json:"spec"` // Resource-specific configuration
}

// Validate checks if the provision request is valid
func (r *ProvisionRequest) Validate() error {
	if r.ResourceType == "" {
		return fmt.Errorf("resourceType is required")
	}
	if r.ResourceName == "" {
		return fmt.Errorf("resourceName is required")
	}
	if r.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if r.Team == "" {
		return fmt.Errorf("team is required")
	}
	if r.CallbackURL == "" {
		return fmt.Errorf("callbackUrl is required")
	}
	if r.Spec == nil {
		return fmt.Errorf("spec is required")
	}
	return nil
}

// DeprovisionRequest represents a request to deprovision a resource
type DeprovisionRequest struct {
	// Resource identification
	DeploymentID string `json:"deploymentId"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Namespace    string `json:"namespace"`

	// Callback configuration
	CallbackURL string `json:"callbackUrl"`
}

// Validate checks if the deprovision request is valid
func (r *DeprovisionRequest) Validate() error {
	if r.DeploymentID == "" {
		return fmt.Errorf("deploymentId is required")
	}
	if r.ResourceType == "" {
		return fmt.Errorf("resourceType is required")
	}
	if r.ResourceName == "" {
		return fmt.Errorf("resourceName is required")
	}
	if r.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if r.CallbackURL == "" {
		return fmt.Errorf("callbackUrl is required")
	}
	return nil
}

// ProvisionResponse is the immediate response to a provision request
type ProvisionResponse struct {
	Status       string `json:"status"`       // accepted
	DeploymentID string `json:"deploymentId"` // Unique ID for this deployment
	Message      string `json:"message"`
}

// DeprovisionResponse is the immediate response to a deprovision request
type DeprovisionResponse struct {
	Status  string `json:"status"` // accepted
	Message string `json:"message"`
}

// CallbackRequest is sent from the broker back to the manager with status updates
type CallbackRequest struct {
	// Deployment identification
	DeploymentID string `json:"deploymentId"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Namespace    string `json:"namespace"`

	// Status information
	Status  string    `json:"status"`          // success, failed, in-progress
	Phase   string    `json:"phase"`           // Provisioning, Ready, Failed, Deleting, Deleted
	Message string    `json:"message"`         // Human-readable status message
	Error   string    `json:"error,omitempty"` // Error message if status is failed
	Time    time.Time `json:"time"`            // Timestamp of status update

	// Resource details (populated when Ready)
	Endpoint           string                 `json:"endpoint,omitempty"`           // Connection endpoint
	Port               int32                  `json:"port,omitempty"`               // Connection port
	ConnectionSecret   string                 `json:"connectionSecret,omitempty"`   // Name of K8s secret with credentials
	Details            map[string]interface{} `json:"details,omitempty"`            // Additional details
	AdditionalMetadata map[string]string      `json:"additionalMetadata,omitempty"` // Resource-specific metadata

	// Cost tracking
	EstimatedMonthlyCost float64 `json:"estimatedMonthlyCost,omitempty"`
}

// StatusResponse is returned when querying the status of a deployment
type StatusResponse struct {
	DeploymentID string    `json:"deploymentId"`
	Phase        string    `json:"phase"`
	Message      string    `json:"message"`
	LastUpdated  time.Time `json:"lastUpdated"`
}

// ResourceStateRequest represents a request to get the actual state of a resource
type ResourceStateRequest struct {
	ResourceType string `json:"resourceType,omitempty"` // Optional filter
	ResourceName string `json:"resourceName,omitempty"` // Optional filter
	Namespace    string `json:"namespace"`              // Required
	DeploymentID string `json:"deploymentId,omitempty"` // Optional filter
}

// Validate checks if the resource state request is valid
func (r *ResourceStateRequest) Validate() error {
	if r.Namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	return nil
}

// ResourceState represents the actual state of a deployed resource
type ResourceState struct {
	// Identification
	DeploymentID string `json:"deploymentId"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Namespace    string `json:"namespace"`

	// Current state
	Phase        string    `json:"phase"`        // Ready, Failed, Provisioning, etc.
	HealthStatus string    `json:"healthStatus"` // Healthy, Degraded, Unhealthy
	Message      string    `json:"message"`
	LastChecked  time.Time `json:"lastChecked"`

	// Resource details
	Endpoint         string            `json:"endpoint,omitempty"`
	Port             int32             `json:"port,omitempty"`
	ConnectionSecret string            `json:"connectionSecret,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`

	// Kubernetes resource information
	ActualSpec    map[string]interface{} `json:"actualSpec,omitempty"`   // What's actually deployed
	DesiredSpec   map[string]interface{} `json:"desiredSpec,omitempty"`  // What should be deployed
	DriftDetected bool                   `json:"driftDetected"`          // True if actual != desired
	DriftDetails  []string               `json:"driftDetails,omitempty"` // Descriptions of drift

	// Resource metrics (if available)
	ResourceUsage *ResourceUsage `json:"resourceUsage,omitempty"`

	// Cost
	EstimatedMonthlyCost float64 `json:"estimatedMonthlyCost,omitempty"`
}

// ResourceUsage contains current resource utilization
type ResourceUsage struct {
	CPUUsage     string `json:"cpuUsage,omitempty"`     // e.g., "250m"
	MemoryUsage  string `json:"memoryUsage,omitempty"`  // e.g., "512Mi"
	StorageUsage string `json:"storageUsage,omitempty"` // e.g., "5Gi"
	Replicas     int32  `json:"replicas,omitempty"`
}

// ResourceStateResponse is returned when querying resource state
type ResourceStateResponse struct {
	Resources []ResourceState `json:"resources"`
	Total     int             `json:"total"`
	Namespace string          `json:"namespace"`
}

// ErrorResponse is returned when an error occurs
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

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

// TenantSpec defines the desired state of Tenant
type TenantSpec struct {
	// DisplayName is the human-readable name of the tenant
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`

	// Description provides additional context about the tenant
	// +optional
	Description string `json:"description,omitempty"`

	// Domain is an optional DNS or organizational domain for the tenant
	// +optional
	Domain string `json:"domain,omitempty"`

	// Contacts lists primary contacts for the tenant
	// +optional
	Contacts []Contact `json:"contacts,omitempty"`

	// BillingCode used for chargeback or invoicing
	// +optional
	BillingCode string `json:"billingCode,omitempty"`

	// Quotas define tenant-wide limits
	// +optional
	Quotas *TenantQuotas `json:"quotas,omitempty"`
}

// TenantQuotas defines resource quotas for a tenant
type TenantQuotas struct {
	// MaxTeams is the maximum number of teams allowed in this tenant
	// +optional
	MaxTeams *int32 `json:"maxTeams,omitempty"`

	// MaxApplications is the maximum number of applications across the tenant
	// +optional
	MaxApplications *int32 `json:"maxApplications,omitempty"`

	// MaxDatabases is the maximum number of databases across the tenant
	// +optional
	MaxDatabases *int32 `json:"maxDatabases,omitempty"`
}

// TenantStatus defines the observed state of Tenant
type TenantStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Active;Suspended;Archived
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ResourceCount tracks the number of resources owned by this tenant
	// +optional
	ResourceCount *TenantResourceCount `json:"resourceCount,omitempty"`

	// CurrentSpend tracks current monthly spending
	// +optional
	CurrentSpend float64 `json:"currentSpend,omitempty"`

	// LastUpdated is the last time the status was updated
	// +optional
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// TenantResourceCount tracks resource counts by type for a tenant
type TenantResourceCount struct {
	Teams        int32 `json:"teams"`
	Applications int32 `json:"applications"`
	Databases    int32 `json:"databases"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Teams",type=integer,JSONPath=`.status.resourceCount.teams`
// +kubebuilder:printcolumn:name="Spend",type=number,JSONPath=`.status.currentSpend`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Tenant is the Schema for the tenants API
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TenantList contains a list of Tenant
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}

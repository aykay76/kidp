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

// TeamSpec defines the desired state of Team
type TeamSpec struct {
	// DisplayName is the human-readable name of the team
	// +kubebuilder:validation:MinLength=1
	DisplayName string `json:"displayName"`

	// Description provides additional context about the team
	// +optional
	Description string `json:"description,omitempty"`

	// Contacts lists the primary contacts for this team
	// +optional
	Contacts []Contact `json:"contacts,omitempty"`

	// Owner references the owning Tenant
	// +optional
	Owner OwnerReference `json:"owner,omitempty"`

	// CostCenter is used for budget tracking and chargeback
	// +optional
	CostCenter string `json:"costCenter,omitempty"`

	// Budget defines spending limits for this team
	// +optional
	Budget *Budget `json:"budget,omitempty"`

	// Quotas define resource limits for this team
	// +optional
	Quotas *TeamQuotas `json:"quotas,omitempty"`
}

// Contact represents a team contact
type Contact struct {
	// Name of the contact person
	Name string `json:"name"`

	// Email address
	Email string `json:"email"`

	// Slack channel or handle
	// +optional
	Slack string `json:"slack,omitempty"`

	// Role of the contact (e.g., "Team Lead", "Tech Lead")
	// +optional
	Role string `json:"role,omitempty"`
}

// Budget defines spending limits
type Budget struct {
	// MonthlyLimit is the maximum monthly spend in USD
	MonthlyLimit float64 `json:"monthlyLimit"`

	// AlertThresholds define when to send alerts (e.g., 0.8 for 80%)
	// +optional
	AlertThresholds []float64 `json:"alertThresholds,omitempty"`
}

// TeamQuotas defines resource quotas for a team
type TeamQuotas struct {
	// MaxApplications is the maximum number of applications
	// +optional
	MaxApplications *int32 `json:"maxApplications,omitempty"`

	// MaxDatabases is the maximum number of databases
	// +optional
	MaxDatabases *int32 `json:"maxDatabases,omitempty"`

	// MaxServices is the maximum number of services
	// +optional
	MaxServices *int32 `json:"maxServices,omitempty"`

	// MaxCaches is the maximum number of caches
	// +optional
	MaxCaches *int32 `json:"maxCaches,omitempty"`
}

// TeamStatus defines the observed state of Team
type TeamStatus struct {
	// Phase represents the current lifecycle phase
	// +kubebuilder:validation:Enum=Active;Suspended;Archived
	// +optional
	Phase string `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// ResourceCount tracks the number of resources owned by this team
	// +optional
	ResourceCount *ResourceCount `json:"resourceCount,omitempty"`

	// CurrentSpend tracks current monthly spending
	// +optional
	CurrentSpend float64 `json:"currentSpend,omitempty"`

	// LastUpdated is the last time the status was updated
	// +optional
	LastUpdated metav1.Time `json:"lastUpdated,omitempty"`
}

// ResourceCount tracks resource counts by type
type ResourceCount struct {
	Applications int32 `json:"applications"`
	Databases    int32 `json:"databases"`
	Services     int32 `json:"services"`
	Caches       int32 `json:"caches"`
	Topics       int32 `json:"topics"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=team
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Applications",type=integer,JSONPath=`.status.resourceCount.applications`
// +kubebuilder:printcolumn:name="Spend",type=number,JSONPath=`.status.currentSpend`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Team is the Schema for the teams API
type Team struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TeamSpec   `json:"spec,omitempty"`
	Status TeamStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TeamList contains a list of Team
type TeamList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Team `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Team{}, &TeamList{})
}

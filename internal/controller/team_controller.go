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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

const teamFinalizerName = "platform.company.com/team-cleanup"

// TeamReconciler reconciles a Team object
type TeamReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.company.com,resources=teams,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.company.com,resources=teams/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.company.com,resources=teams/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *TeamReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Team instance
	team := &platformv1.Team{}
	err := r.Get(ctx, req.NamespacedName, team)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Team resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Team")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !team.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, team)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(team, teamFinalizerName) {
		log.Info("Adding finalizer to Team", "name", team.Name)
		controllerutil.AddFinalizer(team, teamFinalizerName)
		if err := r.Update(ctx, team); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		// Requeue to process with finalizer in place
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("Reconciling Team",
		"name", team.Name,
		"displayName", team.Spec.DisplayName)

	// Validate or infer tenant association for this Team.
	// Priority: explicit Spec.TenantRef -> namespace label set by Tenant controller
	var tenantName string
	if team.Spec.TenantRef != nil && team.Spec.TenantRef.Name != "" {
		tenantName = team.Spec.TenantRef.Name
		// verify Tenant exists
		tenant := &platformv1.Tenant{}
		if err := r.Get(ctx, client.ObjectKey{Name: tenantName}, tenant); err != nil {
			if errors.IsNotFound(err) {
				log.Info("Referenced tenant not found, suspending team", "team", team.Name, "tenant", tenantName)
				team.Status.Phase = "Suspended"
				if statusErr := UpdateStatusWithFallback(ctx, r.Client, team, log); statusErr != nil {
					log.Error(statusErr, "Failed to update Team status")
					return ctrl.Result{}, statusErr
				}
				return ctrl.Result{}, fmt.Errorf("referenced tenant %s not found", tenantName)
			}
			log.Error(err, "Failed to get Tenant for TenantRef")
			return ctrl.Result{}, err
		}
	} else {
		// Try to infer tenant from namespace label set by Tenant controller
		ns := &corev1.Namespace{}
		if err := r.Get(ctx, client.ObjectKey{Name: team.Namespace}, ns); err == nil {
			if tn, ok := ns.Labels["platform.company.com/tenant"]; ok && tn != "" {
				// set TenantRef on the Team for clarity
				team.Spec.TenantRef = &platformv1.ObjectReference{Name: tn, Namespace: ""}
				if err := r.Update(ctx, team); err != nil {
					log.Error(err, "Failed to set TenantRef on team", "team", team.Name, "tenant", tn)
					return ctrl.Result{}, err
				}
				// requeue to process with TenantRef set
				return ctrl.Result{Requeue: true}, nil
			}
		}
		// No TenantRef and no namespace label: mark suspended
		log.Info("No tenantRef set and no tenant label on namespace; suspending team", "team", team.Name)
		team.Status.Phase = "Suspended"
		if statusErr := UpdateStatusWithFallback(ctx, r.Client, team, log); statusErr != nil {
			log.Error(statusErr, "Failed to update Team status")
			return ctrl.Result{}, statusErr
		}
		return ctrl.Result{}, nil
	}

	// Initialize status if needed
	if team.Status.Phase == "" {
		team.Status.Phase = "Active"
		if team.Status.ResourceCount == nil {
			team.Status.ResourceCount = &platformv1.ResourceCount{}
		}
		if err := UpdateStatusWithFallback(ctx, r.Client, team, log); err != nil {
			log.Error(err, "Failed to update Team status")
			return ctrl.Result{}, err
		}

		log.Info("Team status initialized", "name", team.Name)
	}

	// TODO: Implement team management logic
	// 1. Count resources owned by this team
	// 2. Calculate current spend
	// 3. Check quota limits
	// 4. Update status

	log.Info("Team reconciliation complete", "name", team.Name)

	return ctrl.Result{}, nil
}

// handleDeletion performs cleanup and safety checks when a Team is being deleted
func (r *TeamReconciler) handleDeletion(ctx context.Context, team *platformv1.Team) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(team, teamFinalizerName) {
		// Finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	log.Info("Handling Team deletion", "name", team.Name)

	// Check for owned resources before allowing deletion
	if err := r.checkOwnedResources(ctx, team); err != nil {
		log.Error(err, "Cannot delete Team, it still owns resources")
		// Update status to indicate why deletion is blocked
		team.Status.Phase = "Deleting"
		if statusErr := r.Status().Update(ctx, team); statusErr != nil {
			log.Error(statusErr, "Failed to update Team status")
		}
		// Don't remove finalizer - user must delete owned resources first
		return ctrl.Result{}, err
	}

	// All checks passed, safe to delete
	log.Info("Team cleanup completed, removing finalizer", "name", team.Name)

	controllerutil.RemoveFinalizer(team, teamFinalizerName)
	if err := r.Update(ctx, team); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// checkOwnedResources verifies that no resources are owned by this team
func (r *TeamReconciler) checkOwnedResources(ctx context.Context, team *platformv1.Team) error {
	log := log.FromContext(ctx)

	// Check for databases owned by this team
	databaseList := &platformv1.DatabaseList{}
	if err := r.List(ctx, databaseList); err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	ownedDatabases := 0
	for _, db := range databaseList.Items {
		if db.Spec.Owner.Kind == "Team" && db.Spec.Owner.Name == team.Name {
			ownedDatabases++
			log.Info("Found database owned by team",
				"database", db.Name,
				"namespace", db.Namespace,
				"team", team.Name)
		}
	}

	if ownedDatabases > 0 {
		return fmt.Errorf("team %s still owns %d database(s), delete them first",
			team.Name, ownedDatabases)
	}

	// TODO: Check for other resource types when implemented:
	// - Applications
	// - Services
	// - Caches
	// - Topics

	log.Info("No owned resources found for team", "name", team.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TeamReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1.Team{}).
		Complete(r)
}

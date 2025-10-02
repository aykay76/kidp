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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

const databaseFinalizerName = "platform.company.com/database-cleanup"

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.company.com,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.company.com,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.company.com,resources=databases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Database instance
	database := &platformv1.Database{}
	err := r.Get(ctx, req.NamespacedName, database)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			log.Info("Database resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Database")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !database.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, database)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(database, databaseFinalizerName) {
		log.Info("Adding finalizer to Database", "name", database.Name, "namespace", database.Namespace)
		controllerutil.AddFinalizer(database, databaseFinalizerName)
		if err := r.Update(ctx, database); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		// Requeue to process with finalizer in place
		return ctrl.Result{Requeue: true}, nil
	}

	// Log the reconciliation
	log.Info("Reconciling Database",
		"name", database.Name,
		"namespace", database.Namespace,
		"engine", database.Spec.Engine,
		"size", database.Spec.Size)

	// TODO: Implement reconciliation logic
	// 1. Validate the database spec
	// 2. Call broker webhook to provision database
	// 3. Wait for broker callback with status
	// 4. Update database status with connection info

	// For now, just update status to show we're working
	if database.Status.Phase == "" {
		database.Status.Phase = "Pending"
		if err := r.Status().Update(ctx, database); err != nil {
			log.Error(err, "Failed to update Database status")
			return ctrl.Result{}, err
		}

		log.Info("Database status updated to Pending", "name", database.Name)
	}

	// TODO: This is a placeholder - will implement broker communication next
	log.Info("Database reconciliation complete (placeholder)", "name", database.Name)

	return ctrl.Result{}, nil
}

// handleDeletion performs cleanup when a Database is being deleted
func (r *DatabaseReconciler) handleDeletion(ctx context.Context, database *platformv1.Database) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(database, databaseFinalizerName) {
		// Finalizer already removed, nothing to do
		return ctrl.Result{}, nil
	}

	log.Info("Handling Database deletion",
		"name", database.Name,
		"namespace", database.Namespace,
		"deploymentId", database.Status.DeploymentID)

	// Perform cleanup operations
	if err := r.cleanupDatabase(ctx, database); err != nil {
		log.Error(err, "Failed to cleanup Database, will retry")
		// Don't remove finalizer yet, retry cleanup
		return ctrl.Result{}, err
	}

	// Cleanup successful, remove finalizer
	log.Info("Database cleanup completed, removing finalizer",
		"name", database.Name,
		"namespace", database.Namespace)

	controllerutil.RemoveFinalizer(database, databaseFinalizerName)
	if err := r.Update(ctx, database); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// cleanupDatabase performs the actual cleanup operations
func (r *DatabaseReconciler) cleanupDatabase(ctx context.Context, database *platformv1.Database) error {
	log := log.FromContext(ctx)

	// TODO: Implement broker webhook call to deprovision database
	// This is where we would call:
	// brokerClient.DeprovisionDatabase(ctx, database.Status.DeploymentID)

	if database.Status.DeploymentID != "" {
		log.Info("Would call broker to deprovision database",
			"deploymentId", database.Status.DeploymentID,
			"engine", database.Spec.Engine,
			"size", database.Spec.Size)

		// Placeholder for broker call:
		// brokerURL := fmt.Sprintf("https://broker-%s/v1/databases/%s",
		//     database.Spec.Target, database.Status.DeploymentID)
		// resp, err := http.Delete(brokerURL)
		// if err != nil {
		//     return fmt.Errorf("failed to deprovision database: %w", err)
		// }
	}

	// TODO: Delete any associated secrets
	if database.Status.ConnectionSecretRef != nil {
		log.Info("Would delete connection secret",
			"secretName", database.Status.ConnectionSecretRef.Name,
			"secretNamespace", database.Status.ConnectionSecretRef.Namespace)

		// Placeholder for secret deletion:
		// secret := &corev1.Secret{}
		// secret.Name = database.Status.ConnectionSecretRef.Name
		// secret.Namespace = database.Status.ConnectionSecretRef.Namespace
		// if err := r.Delete(ctx, secret); err != nil && !errors.IsNotFound(err) {
		//     return fmt.Errorf("failed to delete secret: %w", err)
		// }
	}

	log.Info("Database cleanup simulated successfully",
		"name", database.Name,
		"namespace", database.Namespace)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1.Database{}).
		Complete(r)
}

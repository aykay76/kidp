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
	"os"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1 "github.com/aykay76/kidp/api/v1"
	"github.com/aykay76/kidp/pkg/brokerclient"
	"github.com/aykay76/kidp/pkg/brokerregistry"
)

const databaseFinalizerName = "platform.company.com/database-cleanup"

// DatabaseReconciler reconciles a Database object
type DatabaseReconciler struct {
	client.Client
	Scheme         *runtime.Scheme
	BrokerRegistry *brokerregistry.Registry
	Recorder       record.EventRecorder
}

// +kubebuilder:rbac:groups=platform.company.com,resources=databases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.company.com,resources=databases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.company.com,resources=databases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *DatabaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Debug: trace reconcile entry (logged at V(1))
	log.V(1).Info("Reconcile called", "namespace", req.Namespace, "name", req.Name)

	// Fetch the Database instance
	database := &platformv1.Database{}
	err := r.Get(ctx, req.NamespacedName, database)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, could have been deleted after reconcile request.
			// Return and don't requeue
			log.Info("Database resource not found. Ignoring since object must be deleted")
			log.V(1).Info("initial Get returned NotFound", "namespace", req.Namespace, "name", req.Name)
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Database")
		log.V(1).Info("initial Get returned error", "error", err)
		return ctrl.Result{}, err
	}

	log.V(1).Info("fetched Database object", "namespace", database.Namespace, "name", database.Name, "finalizers", database.Finalizers)

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
			log.V(1).Info("Update(add finalizer) returned error", "error", err, "isNotFound", errors.IsNotFound(err))
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

	// Resolve tenant for this database
	tenant, terr := ResolveTenant(ctx, r.Client, database)
	if terr != nil {
		log.Info("Unable to resolve tenant for database, suspending until tenant is available", "database", database.Name, "err", terr)
		if r.Recorder != nil {
			r.Recorder.Eventf(database, "Warning", "TenantUnresolved", "tenant could not be resolved: %v", terr)
		}
		database.Status.Phase = "Suspended"
		if err := UpdateStatusWithFallback(ctx, r.Client, database, log); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Ensure DB has tenant label for easy querying by other controllers
	if database.Labels == nil {
		database.Labels = map[string]string{}
	}
	if database.Labels["platform.company.com/tenant"] != tenant.Name {
		database.Labels["platform.company.com/tenant"] = tenant.Name
		if err := r.Update(ctx, database); err != nil {
			log.Error(err, "Failed to label Database with tenant")
			log.V(1).Info("Update(label) returned error", "error", err, "isNotFound", errors.IsNotFound(err))
			return ctrl.Result{}, err
		}
		if r.Recorder != nil {
			r.Recorder.Eventf(database, "Normal", "TenantAssigned", "Assigned tenant %s to database %s", tenant.Name, database.Name)
		}
		// Requeue to proceed with provisioning after label update
		return ctrl.Result{Requeue: true}, nil
	}

	// If deploymentId exists, provisioning is in progress or complete
	// Status updates will come via webhook callbacks
	if database.Status.DeploymentID != "" {
		log.Info("Database already provisioned or in progress",
			"deploymentId", database.Status.DeploymentID,
			"phase", database.Status.Phase)
		return ctrl.Result{}, nil
	}

	// Update status to Provisioning
	if database.Status.Phase != "Provisioning" {
		database.Status.Phase = "Provisioning"
		if err := UpdateStatusWithFallback(ctx, r.Client, database, log); err != nil {
			return ctrl.Result{}, err
		}
		log.Info("Database status updated to Provisioning", "name", database.Name)
	}

	// Call broker to provision database
	if err := r.provisionDatabase(ctx, database); err != nil {
		log.Error(err, "Failed to provision database")
		database.Status.Phase = "Failed"
		if statusErr := UpdateStatusWithFallback(ctx, r.Client, database, log); statusErr != nil {
			log.Error(statusErr, "Failed to update status to Failed")
		}
		return ctrl.Result{}, err
	}

	log.Info("Database provisioning request sent to broker", "name", database.Name)

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

	// Call broker to deprovision database
	if database.Status.DeploymentID != "" && r.BrokerRegistry != nil {
		log.Info("Calling broker to deprovision database",
			"deploymentId", database.Status.DeploymentID,
			"engine", database.Spec.Engine)
		// Prefer the broker that handled provisioning if recorded in status
		var selectedBroker *platformv1.Broker
		var err error
		if database.Status.BrokerRef != nil && database.Status.BrokerRef.Name != "" {
			// Try to fetch the Broker CR directly
			broker := &platformv1.Broker{}
			ns := database.Status.BrokerRef.Namespace
			if ns == "" {
				ns = database.Namespace
			}
			if getErr := r.Get(ctx, client.ObjectKey{Namespace: ns, Name: database.Status.BrokerRef.Name}, broker); getErr == nil {
				selectedBroker = broker
			} else {
				log.Info("Recorded BrokerRef not found, falling back to registry selection",
					"brokerRefName", database.Status.BrokerRef.Name, "err", getErr)
			}
		}

		// If no broker from status, select one matching capabilities
		if selectedBroker == nil {
			criteria := brokerregistry.SelectionCriteria{
				ResourceType: "Database",
				Provider:     database.Spec.Engine,
			}

			selectedBroker, err = r.BrokerRegistry.SelectBroker(ctx, criteria)
			if err != nil {
				// Log error but continue; resource might have been provisioned by a broker that no longer exists
				log.Error(err, "Failed to select broker for deprovisioning, continuing anyway")
				selectedBroker = nil
			}
		}

		if selectedBroker != nil {
			// Create broker client for deprovisioning
			brokerClient := brokerclient.NewClient(selectedBroker.Spec.Endpoint)

			// Get callback URL from environment or use default
			callbackURL := os.Getenv("KIDP_CALLBACK_URL")
			if callbackURL == "" {
				callbackURL = "http://manager-webhook-service.kidp-system.svc.cluster.local:9090/v1/callback"
			}

			deprovReq := brokerclient.DeprovisionRequest{
				DeploymentID: database.Status.DeploymentID,
				ResourceType: "database",
				ResourceName: database.Name,
				Namespace:    database.Namespace,
				CallbackURL:  callbackURL,
			}

			if _, err := brokerClient.Deprovision(ctx, deprovReq); err != nil {
				return fmt.Errorf("failed to call broker deprovision: %w", err)
			}

			log.Info("Deprovisioning request sent to broker",
				"deploymentId", database.Status.DeploymentID,
				"broker", selectedBroker.Name)
		}
	}

	log.Info("Database cleanup completed",
		"name", database.Name,
		"namespace", database.Namespace)

	return nil
}

// provisionDatabase calls the broker to provision a new database
func (r *DatabaseReconciler) provisionDatabase(ctx context.Context, database *platformv1.Database) error {
	log := log.FromContext(ctx)

	if r.BrokerRegistry == nil {
		return fmt.Errorf("broker registry not configured")
	}

	// Select appropriate broker based on database spec
	criteria := brokerregistry.SelectionCriteria{
		ResourceType:  "Database",
		CloudProvider: "",                   // Could be extracted from database.Spec.Target or labels
		Region:        "",                   // Could be extracted from database.Spec.Target
		Provider:      database.Spec.Engine, // e.g., "postgresql", "mysql"
	}

	// TODO: Parse Target field to extract cloudProvider and region
	// For now, if Target is not empty, try to use it as cloudProvider hint
	if database.Spec.Target != "" {
		// Target format could be "azure-eastus", "aws-us-west-2", etc.
		// This is a simplified parsing - real implementation would be more robust
		log.Info("Database has target specified", "target", database.Spec.Target)
		// criteria.CloudProvider = parseCloudProvider(database.Spec.Target)
		// criteria.Region = parseRegion(database.Spec.Target)
	}

	// Select broker
	selectedBroker, err := r.BrokerRegistry.SelectBroker(ctx, criteria)
	if err != nil {
		return fmt.Errorf("failed to select broker: %w", err)
	}

	log.Info("Selected broker for provisioning",
		"broker", selectedBroker.Name,
		"endpoint", selectedBroker.Spec.Endpoint,
		"cloudProvider", selectedBroker.Spec.CloudProvider,
		"region", selectedBroker.Spec.Region)

	// Create broker client for the selected broker
	brokerClient := brokerclient.NewClient(selectedBroker.Spec.Endpoint)

	// Get callback URL from environment or use default
	callbackURL := os.Getenv("KIDP_CALLBACK_URL")
	if callbackURL == "" {
		callbackURL = "http://manager-webhook-service.kidp-system.svc.cluster.local:9090/v1/callback"
	}

	// Build provision request
	provReq := brokerclient.ProvisionRequest{
		ResourceType: "database",
		ResourceName: database.Name,
		Namespace:    database.Namespace,
		Team:         fmt.Sprintf("%s/%s", database.Spec.Owner.Kind, database.Spec.Owner.Name),
		Owner:        database.Spec.Owner.Name,
		CallbackURL:  callbackURL,
		Spec: map[string]interface{}{
			"engine":  database.Spec.Engine,
			"version": database.Spec.Version,
			"size":    database.Spec.Size,
		},
	}

	// Call broker
	log.Info("Calling broker to provision database",
		"engine", database.Spec.Engine,
		"size", database.Spec.Size,
		"callbackURL", callbackURL)

	resp, err := brokerClient.Provision(ctx, provReq)
	if err != nil {
		return fmt.Errorf("failed to call broker provision: %w", err)
	}

	log.Info("Broker accepted provisioning request",
		"deploymentId", resp.DeploymentID,
		"status", resp.Status)

	// Store deploymentID in status
	database.Status.DeploymentID = resp.DeploymentID

	// Persist which broker handled the provisioning so deprovision targets the same broker
	database.Status.BrokerRef = &platformv1.ObjectReference{
		Name:      selectedBroker.Name,
		Namespace: selectedBroker.Namespace,
	}
	if err := r.Status().Update(ctx, database); err != nil {
		// try fallback to full update for fake clients
		if errors.IsNotFound(err) {
			if uerr := r.Update(ctx, database); uerr != nil {
				return fmt.Errorf("failed to update status with deploymentId (fallback): %w", uerr)
			}
		} else {
			return fmt.Errorf("failed to update status with deploymentId: %w", err)
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DatabaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("database-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1.Database{}).
		Complete(r)
}

package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

const applicationFinalizerName = "platform.company.com/application-cleanup"

// ApplicationReconciler reconciles an Application object
type ApplicationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=platform.company.com,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.company.com,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.company.com,resources=applications/finalizers,verbs=update

func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	app := &platformv1.Application{}
	if err := r.Get(ctx, req.NamespacedName, app); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Application resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Application")
		return ctrl.Result{}, err
	}

	// Handle deletion
	if !app.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, app)
	}

	// Add finalizer
	if !controllerutil.ContainsFinalizer(app, applicationFinalizerName) {
		log.Info("Adding finalizer to Application", "name", app.Name)
		controllerutil.AddFinalizer(app, applicationFinalizerName)
		if err := r.Update(ctx, app); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("Reconciling Application", "name", app.Name, "displayName", app.Spec.DisplayName)

	if app.Status.Phase == "" {
		app.Status.Phase = "Draft"
		if err := r.Status().Update(ctx, app); err != nil {
			log.Error(err, "Failed to update Application status")
			return ctrl.Result{}, err
		}
		log.Info("Application status initialized", "name", app.Name)
	}

	// Resolve tenant for this application (via tenantRef, owner chain, or namespace label)
	tenant, terr := ResolveTenant(ctx, r.Client, app)
	if terr != nil {
		log.Info("Unable to resolve tenant for application, suspending until tenant is available", "application", app.Name, "err", terr)
		if r.Recorder != nil {
			r.Recorder.Eventf(app, "Warning", "TenantUnresolved", "tenant could not be resolved: %v", terr)
		}
		app.Status.Phase = "Suspended"
		if statusErr := r.Status().Update(ctx, app); statusErr != nil {
			log.Error(statusErr, "Failed to update Application status")
			return ctrl.Result{}, statusErr
		}
		// don't requeue aggressively here; tenant controller or owner updates will trigger reconciliation
		return ctrl.Result{}, nil
	}
	log.Info("Application resolved tenant", "application", app.Name, "tenant", tenant.Name)
	// Ensure resource has tenant label for easy querying by other controllers
	if app.Labels == nil {
		app.Labels = map[string]string{}
	}
	if app.Labels["platform.company.com/tenant"] != tenant.Name {
		app.Labels["platform.company.com/tenant"] = tenant.Name
		if err := r.Update(ctx, app); err != nil {
			log.Error(err, "Failed to label Application with tenant")
			return ctrl.Result{}, err
		}
		if r.Recorder != nil {
			r.Recorder.Eventf(app, "Normal", "TenantAssigned", "Assigned tenant %s to application %s", tenant.Name, app.Name)
		}
		// Requeue to continue processing with label in place
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("Application reconciliation complete", "name", app.Name)
	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) handleDeletion(ctx context.Context, app *platformv1.Application) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(app, applicationFinalizerName) {
		return ctrl.Result{}, nil
	}

	log.Info("Handling Application deletion", "name", app.Name)

	// Check for owned resources (databases) - if any exist, block deletion
	if err := r.checkOwnedResources(ctx, app); err != nil {
		log.Error(err, "Cannot delete Application, it still owns resources")
		app.Status.Phase = "Deleting"
		if statusErr := r.Status().Update(ctx, app); statusErr != nil {
			log.Error(statusErr, "Failed to update Application status")
		}
		return ctrl.Result{}, err
	}

	controllerutil.RemoveFinalizer(app, applicationFinalizerName)
	if err := r.Update(ctx, app); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

func (r *ApplicationReconciler) checkOwnedResources(ctx context.Context, app *platformv1.Application) error {
	// Currently only checks for Databases owned by this application
	dbList := &platformv1.DatabaseList{}
	if err := r.List(ctx, dbList); err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	owned := 0
	for _, db := range dbList.Items {
		if db.Spec.Owner.Kind == "Application" && db.Spec.Owner.Name == app.Name && db.Namespace == app.Namespace {
			owned++
		}
	}
	if owned > 0 {
		return fmt.Errorf("application %s still owns %d database(s), delete them first", app.Name, owned)
	}
	return nil
}

func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Wire event recorder
	r.Recorder = mgr.GetEventRecorderFor("application-controller")
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1.Application{}).
		Complete(r)
}

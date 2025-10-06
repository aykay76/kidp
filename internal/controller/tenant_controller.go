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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

const tenantFinalizerName = "platform.company.com/tenant-cleanup"

// TenantReconciler reconciles a Tenant object
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=platform.company.com,resources=tenants,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.company.com,resources=tenants/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.company.com,resources=tenants/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	tenant := &platformv1.Tenant{}
	if err := r.Get(ctx, req.NamespacedName, tenant); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Tenant resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Tenant")
		return ctrl.Result{}, err
	}

	// Tenants are cluster-scoped; NamespacedName will have empty Namespace

	// Handle deletion
	if !tenant.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, tenant)
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(tenant, tenantFinalizerName) {
		log.Info("Adding finalizer to Tenant", "name", tenant.Name)
		controllerutil.AddFinalizer(tenant, tenantFinalizerName)
		if err := r.Update(ctx, tenant); err != nil {
			log.Error(err, "Failed to add finalizer")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true}, nil
	}

	log.Info("Reconciling Tenant", "name", tenant.Name, "displayName", tenant.Spec.DisplayName)

	// Initialize status if needed
	if tenant.Status.Phase == "" {
		tenant.Status.Phase = "Active"
		if tenant.Status.ResourceCount == nil {
			tenant.Status.ResourceCount = &platformv1.TenantResourceCount{}
		}
		if err := r.Status().Update(ctx, tenant); err != nil {
			log.Error(err, "Failed to update Tenant status")
			return ctrl.Result{}, err
		}
		log.Info("Tenant status initialized", "name", tenant.Name)
	}

	// TODO: Implement tenant management logic
	// - Ensure namespaces for tenant exist
	// - Enforce tenant-level quotas
	// - Aggregate resource counts and spend across namespaces belonging to tenant

	// Ensure tenant namespace exists (namespace per tenant for boundary)
	nsName := "tenant-" + tenant.Name
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: nsName}, ns); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Creating namespace for tenant", "namespace", nsName)
			ns = &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: nsName,
					Labels: map[string]string{
						"platform.company.com/tenant": tenant.Name,
					},
				},
			}
			if err := r.Create(ctx, ns); err != nil {
				log.Error(err, "Failed to create namespace for tenant", "namespace", nsName)
				return ctrl.Result{}, err
			}
			log.Info("Namespace created for tenant", "namespace", nsName)
		} else {
			log.Error(err, "Failed to get namespace for tenant")
			return ctrl.Result{}, err
		}
	}

	log.Info("Tenant reconciliation complete", "name", tenant.Name)
	return ctrl.Result{}, nil
}

func (r *TenantReconciler) handleDeletion(ctx context.Context, tenant *platformv1.Tenant) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(tenant, tenantFinalizerName) {
		return ctrl.Result{}, nil
	}

	log.Info("Handling Tenant deletion", "name", tenant.Name)

	// Check for teams or other resources across namespaces before allowing deletion
	// TODO: Implement checks similar to TeamReconciler.checkOwnedResources but across all namespaces

	// Remove finalizer
	controllerutil.RemoveFinalizer(tenant, tenantFinalizerName)
	if err := r.Update(ctx, tenant); err != nil {
		log.Error(err, "Failed to remove finalizer")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1.Tenant{}).
		Complete(r)
}

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
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

// BrokerReconciler reconciles a Broker object
type BrokerReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	httpClient *http.Client
}

// +kubebuilder:rbac:groups=platform.company.com,resources=brokers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=platform.company.com,resources=brokers/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=platform.company.com,resources=brokers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop
func (r *BrokerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the Broker instance
	broker := &platformv1.Broker{}
	if err := r.Get(ctx, req.NamespacedName, broker); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Perform health check
	healthy, message := r.checkBrokerHealth(ctx, broker)

	// Update status based on health check
	broker.Status.ObservedGeneration = broker.Generation
	now := metav1.Now()

	if healthy {
		broker.Status.Phase = "Ready"
		broker.Status.LastHeartbeat = &now
		broker.Status.Message = "Broker is healthy and operational"

		// Set Ready condition
		meta.SetStatusCondition(&broker.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "BrokerHealthy",
			Message:            message,
			ObservedGeneration: broker.Generation,
		})
	} else {
		broker.Status.Phase = "Unhealthy"
		broker.Status.Message = message

		// Set Ready condition to false
		meta.SetStatusCondition(&broker.Status.Conditions, metav1.Condition{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "BrokerUnhealthy",
			Message:            message,
			ObservedGeneration: broker.Generation,
		})
	}

	// Update the status
	if err := r.Status().Update(ctx, broker); err != nil {
		log.Error(err, "Failed to update Broker status")
		return ctrl.Result{}, err
	}

	// Determine requeue interval based on health check config
	requeueInterval := 30 * time.Second
	if broker.Spec.HealthCheck != nil && broker.Spec.HealthCheck.IntervalSeconds > 0 {
		requeueInterval = time.Duration(broker.Spec.HealthCheck.IntervalSeconds) * time.Second
	}

	log.Info("Reconciled Broker", "phase", broker.Status.Phase, "requeue", requeueInterval)
	return ctrl.Result{RequeueAfter: requeueInterval}, nil
}

// checkBrokerHealth performs a health check against the broker endpoint
func (r *BrokerReconciler) checkBrokerHealth(ctx context.Context, broker *platformv1.Broker) (bool, string) {
	// Build health check URL
	healthEndpoint := "/health"
	if broker.Spec.HealthCheck != nil && broker.Spec.HealthCheck.Endpoint != "" {
		healthEndpoint = broker.Spec.HealthCheck.Endpoint
	}
	healthURL := fmt.Sprintf("%s%s", broker.Spec.Endpoint, healthEndpoint)

	// Determine timeout
	timeout := 5 * time.Second
	if broker.Spec.HealthCheck != nil && broker.Spec.HealthCheck.TimeoutSeconds > 0 {
		timeout = time.Duration(broker.Spec.HealthCheck.TimeoutSeconds) * time.Second
	}

	// Create request with timeout
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", healthURL, nil)
	if err != nil {
		return false, fmt.Sprintf("Failed to create health check request: %v", err)
	}

	// Execute request
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Health check failed: %v", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, "Health check passed"
	}

	return false, fmt.Sprintf("Health check returned status %d", resp.StatusCode)
}

// SetupWithManager sets up the controller with the Manager
func (r *BrokerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize HTTP client if not set
	if r.httpClient == nil {
		r.httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1.Broker{}).
		Complete(r)
}

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

package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

// CallbackRequest mirrors the broker's CallbackRequest structure
type CallbackRequest struct {
	DeploymentID         string                 `json:"deploymentId"`
	ResourceType         string                 `json:"resourceType"`
	ResourceName         string                 `json:"resourceName"`
	Namespace            string                 `json:"namespace"`
	Status               string                 `json:"status"`
	Phase                string                 `json:"phase"`
	Message              string                 `json:"message"`
	Error                string                 `json:"error,omitempty"`
	Time                 time.Time              `json:"time"`
	Endpoint             string                 `json:"endpoint,omitempty"`
	Port                 int32                  `json:"port,omitempty"`
	ConnectionSecret     string                 `json:"connectionSecret,omitempty"`
	Details              map[string]interface{} `json:"details,omitempty"`
	AdditionalMetadata   map[string]string      `json:"additionalMetadata,omitempty"`
	EstimatedMonthlyCost float64                `json:"estimatedMonthlyCost,omitempty"`
}

// Server handles webhook callbacks from the broker
type Server struct {
	client client.Client
	port   int
}

// NewServer creates a new webhook server
func NewServer(client client.Client, port int) *Server {
	return &Server{
		client: client,
		port:   port,
	}
}

// Start starts the webhook server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/callback", s.handleCallback)
	mux.HandleFunc("/health", s.handleHealth)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Webhook server listening on :%d", s.port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Webhook server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown gracefully
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdownCtx)
}

// handleCallback processes callbacks from the broker
func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var callback CallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&callback); err != nil {
		log.Printf("Failed to decode callback: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("Received callback: deploymentId=%s, resourceType=%s, status=%s, phase=%s",
		callback.DeploymentID, callback.ResourceType, callback.Status, callback.Phase)

	// Route to appropriate handler based on resource type
	var err error
	switch callback.ResourceType {
	case "database":
		err = s.handleDatabaseCallback(r.Context(), callback)
	default:
		log.Printf("Unknown resource type: %s", callback.ResourceType)
		http.Error(w, "Unknown resource type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("Failed to process callback: %v", err)
		http.Error(w, "Failed to process callback", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "accepted",
	})
}

// handleDatabaseCallback updates the Database CR based on the callback
func (s *Server) handleDatabaseCallback(ctx context.Context, callback CallbackRequest) error {
	// Find the Database CR by deploymentId
	// We need to list all databases and find the one with matching deploymentId
	var dbList platformv1.DatabaseList
	if err := s.client.List(ctx, &dbList, client.InNamespace(callback.Namespace)); err != nil {
		return fmt.Errorf("failed to list databases: %w", err)
	}

	var database *platformv1.Database
	for i := range dbList.Items {
		if dbList.Items[i].Status.DeploymentID == callback.DeploymentID {
			database = &dbList.Items[i]
			break
		}
	}

	if database == nil {
		return fmt.Errorf("database not found for deploymentId: %s", callback.DeploymentID)
	}

	// Update the database status
	database.Status.Phase = callback.Phase

	// Update resource details if provided
	if callback.Status == "success" && callback.Phase == "Ready" {
		database.Status.Endpoint = callback.Endpoint
		database.Status.Port = callback.Port

		// Set connection secret reference
		if callback.ConnectionSecret != "" {
			database.Status.ConnectionSecretRef = &platformv1.SecretReference{
				Name:      callback.ConnectionSecret,
				Namespace: callback.Namespace,
			}
		}

		// Set conditions
		now := metav1.NewTime(callback.Time)
		database.Status.Conditions = []metav1.Condition{
			{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: now,
				Reason:             "ProvisioningSucceeded",
				Message:            callback.Message,
			},
		}
	} else if callback.Status == "failed" {
		now := metav1.NewTime(callback.Time)
		database.Status.Conditions = []metav1.Condition{
			{
				Type:               "Ready",
				Status:             metav1.ConditionFalse,
				LastTransitionTime: now,
				Reason:             "ProvisioningFailed",
				Message:            callback.Error,
			},
		}
	}

	// Update the status
	if err := s.client.Status().Update(ctx, database); err != nil {
		return fmt.Errorf("failed to update database status: %w", err)
	}

	log.Printf("Updated database %s/%s: phase=%s, status=%s",
		database.Namespace, database.Name, database.Status.Phase, callback.Status)

	return nil
}

// handleHealth returns health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

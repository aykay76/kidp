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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"crypto/ed25519"

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

	// Read full body for signature verification
	var callback CallbackRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&callback); err != nil {
		log.Printf("Failed to decode callback: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Verify signature headers: Broker-Name, Timestamp, Signature
	brokerName := r.Header.Get("X-KIDP-Broker-Name")
	timestamp := r.Header.Get("X-KIDP-Timestamp")
	signature := r.Header.Get("X-KIDP-Signature")

	if brokerName == "" || timestamp == "" || signature == "" {
		log.Printf("Missing signature headers: broker=%s timestamp=%s signature=%s", brokerName, timestamp, signature)
		http.Error(w, "Missing signature headers", http.StatusUnauthorized)
		return
	}

	// Replay protection: allow small skew (5m)
	ts, terr := time.Parse(time.RFC3339, timestamp)
	if terr != nil {
		log.Printf("Invalid timestamp header: %v", terr)
		http.Error(w, "Invalid timestamp", http.StatusBadRequest)
		return
	}
	if time.Since(ts) > 5*time.Minute || time.Until(ts) > 1*time.Minute {
		log.Printf("Timestamp outside allowed skew: %v", ts)
		http.Error(w, "Timestamp outside allowed range", http.StatusUnauthorized)
		return
	}

	// Lookup Broker CR by name to get stored public key
	var brokerCR platformv1.Broker
	if getErr := s.client.Get(r.Context(), client.ObjectKey{Namespace: "default", Name: brokerName}, &brokerCR); getErr != nil {
		log.Printf("Failed to get Broker CR for %s: %v", brokerName, getErr)
		http.Error(w, "Unknown broker", http.StatusUnauthorized)
		return
	}

	pubB64 := brokerCR.Status.CallbackPublicKey
	if pubB64 == "" {
		// Accept public key from header only if CR doesn't have one (initial registration)
		pubB64 = r.Header.Get("X-KIDP-Public-Key")
		if pubB64 == "" {
			log.Printf("No public key available for broker %s", brokerName)
			http.Error(w, "No public key available", http.StatusUnauthorized)
			return
		}
		// Optionally, persist this key to the Broker CR (left as future work)
	}

	// Reconstruct the raw body for verification: we need the original JSON bytes.
	// Since we've already decoded into struct, re-marshal to get deterministic bytes.
	rawBody, mErr := json.Marshal(callback)
	if mErr != nil {
		log.Printf("Failed to re-marshal callback for verification: %v", mErr)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	if ok, vErr := verifySignature(rawBody, timestamp, signature, pubB64); !ok {
		log.Printf("Signature verification failed for broker %s: %v", brokerName, vErr)
		http.Error(w, "Invalid signature", http.StatusUnauthorized)
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

// verifySignature verifies an Ed25519 signature. signatureB64 and pubKeyB64 are base64 encoded.
// The message that was signed is timestamp + '.' + body
func verifySignature(body []byte, timestamp, signatureB64, pubKeyB64 string) (bool, error) {
	sigBytes, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}
	pubBytes, err := base64.StdEncoding.DecodeString(pubKeyB64)
	if err != nil {
		return false, fmt.Errorf("failed to decode public key: %w", err)
	}
	if len(pubBytes) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key size: %d", len(pubBytes))
	}
	msg := append([]byte(timestamp+"."), body...)
	ok := ed25519.Verify(ed25519.PublicKey(pubBytes), msg, sigBytes)
	if !ok {
		return false, fmt.Errorf("signature verification failed")
	}
	return true, nil
}

// handleHealth returns health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

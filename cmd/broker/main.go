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

package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aykay76/kidp/pkg/broker"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	version = "0.1.0"
)

// Server configuration
type Config struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
	LogLevel        string
}

// Server holds the HTTP server and dependencies
type Server struct {
	config    *Config
	router    *http.ServeMux
	logger    *log.Logger
	k8sClient *broker.K8sClient
	startTime time.Time
}

func main() {
	// Parse command-line flags
	config := &Config{}
	flag.IntVar(&config.Port, "port", 8080, "HTTP server port")
	flag.DurationVar(&config.ReadTimeout, "read-timeout", 15*time.Second, "HTTP read timeout")
	flag.DurationVar(&config.WriteTimeout, "write-timeout", 15*time.Second, "HTTP write timeout")
	flag.DurationVar(&config.ShutdownTimeout, "shutdown-timeout", 30*time.Second, "Graceful shutdown timeout")
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Create logger
	logger := log.New(os.Stdout, "[broker] ", log.LstdFlags|log.Lmsgprefix)
	logger.Printf("Starting KIDP Deployment Broker v%s", version)
	logger.Printf("Configuration: port=%d, read-timeout=%s, write-timeout=%s",
		config.Port, config.ReadTimeout, config.WriteTimeout)

	// Create Kubernetes client
	k8sClient, err := broker.NewK8sClient()
	if err != nil {
		logger.Fatalf("Failed to create Kubernetes client: %v", err)
	}
	logger.Println("Successfully connected to Kubernetes cluster")

	// Create server
	server := NewServer(config, logger, k8sClient)

	// Setup HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", config.Port),
		Handler:      server.router,
		ReadTimeout:  config.ReadTimeout,
		WriteTimeout: config.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Printf("HTTP server listening on port %d", config.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Println("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.ShutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Printf("Server forced to shutdown: %v", err)
	}

	logger.Println("Server exited")
}

// NewServer creates a new broker server instance
func NewServer(config *Config, logger *log.Logger, k8sClient *broker.K8sClient) *Server {
	s := &Server{
		config:    config,
		router:    http.NewServeMux(),
		logger:    logger,
		k8sClient: k8sClient,
		startTime: time.Now(),
	}

	// Register routes
	s.registerRoutes()

	return s
}

// registerRoutes sets up all HTTP endpoints
func (s *Server) registerRoutes() {
	// Health check
	s.router.HandleFunc("/health", s.handleHealth)
	s.router.HandleFunc("/readiness", s.handleReadiness)

	// API v1 routes
	s.router.HandleFunc("/v1/provision", s.handleProvision)
	s.router.HandleFunc("/v1/deprovision", s.handleDeprovision)
	s.router.HandleFunc("/v1/status", s.handleStatus)
	s.router.HandleFunc("/v1/resources", s.handleGetResources)

	// Root handler
	s.router.HandleFunc("/", s.handleRoot)
}

// handleHealth returns server health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":  "healthy",
		"version": version,
		"time":    time.Now().UTC().Format(time.RFC3339),
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleReadiness checks if broker is ready to accept requests
func (s *Server) handleReadiness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Check Kubernetes API connectivity
	ready := true
	reason := "ok"

	if s.k8sClient != nil {
		// Try to list namespaces as a health check
		ctx := r.Context()
		_, err := s.k8sClient.Clientset().CoreV1().Namespaces().List(ctx,
			metav1.ListOptions{Limit: 1})
		if err != nil {
			ready = false
			reason = fmt.Sprintf("kubernetes API not accessible: %v", err)
			s.logger.Printf("Readiness check failed: %s", reason)
		}
	} else {
		ready = false
		reason = "kubernetes client not initialized"
	}

	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"ready":   ready,
		"reason":  reason,
		"version": version,
	}

	s.respondJSON(w, status, response)
}

// handleProvision handles resource provisioning requests
func (s *Server) handleProvision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Printf("Received provision request from %s", r.RemoteAddr)

	// Parse request body
	var req broker.ProvisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Printf("Failed to decode provision request: %v", err)
		s.respondJSON(w, http.StatusBadRequest, broker.ErrorResponse{
			Error:   "invalid_request",
			Message: fmt.Sprintf("Failed to parse request body: %v", err),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		s.logger.Printf("Invalid provision request: %v", err)
		s.respondJSON(w, http.StatusBadRequest, broker.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Generate deployment ID
	deploymentID := generateDeploymentID()
	s.logger.Printf("Created deployment %s for %s/%s in namespace %s",
		deploymentID, req.ResourceType, req.ResourceName, req.Namespace)

	// TODO: Queue the provisioning task
	// TODO: Start async provisioning in a goroutine

	// Return accepted response
	response := broker.ProvisionResponse{
		Status:       "accepted",
		DeploymentID: deploymentID,
		Message:      fmt.Sprintf("Provisioning request accepted for %s/%s", req.ResourceType, req.ResourceName),
	}

	s.respondJSON(w, http.StatusAccepted, response)
}

// handleDeprovision handles resource deprovisioning requests
func (s *Server) handleDeprovision(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.logger.Printf("Received deprovision request from %s", r.RemoteAddr)

	// Parse request body
	var req broker.DeprovisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Printf("Failed to decode deprovision request: %v", err)
		s.respondJSON(w, http.StatusBadRequest, broker.ErrorResponse{
			Error:   "invalid_request",
			Message: fmt.Sprintf("Failed to parse request body: %v", err),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate request
	if err := req.Validate(); err != nil {
		s.logger.Printf("Invalid deprovision request: %v", err)
		s.respondJSON(w, http.StatusBadRequest, broker.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	s.logger.Printf("Deprovisioning %s for deployment %s", req.ResourceName, req.DeploymentID)

	// TODO: Queue the deprovisioning task
	// TODO: Start async deprovisioning in a goroutine

	// Return accepted response
	response := broker.DeprovisionResponse{
		Status:  "accepted",
		Message: fmt.Sprintf("Deprovisioning request accepted for deployment %s", req.DeploymentID),
	}

	s.respondJSON(w, http.StatusAccepted, response)
}

// handleStatus returns the status of a deployment
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deploymentID := r.URL.Query().Get("id")
	if deploymentID == "" {
		http.Error(w, "Missing deployment ID", http.StatusBadRequest)
		return
	}

	s.logger.Printf("Status query for deployment: %s", deploymentID)

	// TODO: Implement status lookup
	response := map[string]interface{}{
		"deploymentId": deploymentID,
		"status":       "unknown",
		"message":      "Status endpoint - implementation pending",
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleGetResources returns the actual state of resources managed by this broker
func (s *Server) handleGetResources(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req broker.ResourceStateRequest

	// Support both GET with query params and POST with JSON body
	if r.Method == http.MethodGet {
		req = broker.ResourceStateRequest{
			Namespace:    r.URL.Query().Get("namespace"),
			ResourceType: r.URL.Query().Get("resourceType"),
			ResourceName: r.URL.Query().Get("resourceName"),
			DeploymentID: r.URL.Query().Get("deploymentId"),
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.logger.Printf("Failed to decode resource state request: %v", err)
			s.respondJSON(w, http.StatusBadRequest, broker.ErrorResponse{
				Error:   "invalid_request",
				Message: fmt.Sprintf("Failed to parse request body: %v", err),
				Code:    http.StatusBadRequest,
			})
			return
		}
	}

	// Validate request
	if err := req.Validate(); err != nil {
		s.logger.Printf("Invalid resource state request: %v", err)
		s.respondJSON(w, http.StatusBadRequest, broker.ErrorResponse{
			Error:   "validation_failed",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	s.logger.Printf("Resource state query for namespace: %s, type: %s, name: %s",
		req.Namespace, req.ResourceType, req.ResourceName)

	// TODO: Implement actual resource state lookup from Kubernetes
	// For now, return a placeholder response
	response := broker.ResourceStateResponse{
		Resources: []broker.ResourceState{},
		Total:     0,
		Namespace: req.Namespace,
	}

	s.respondJSON(w, http.StatusOK, response)
}

// handleRoot handles requests to the root path
// This provides a self-documenting API discovery endpoint following REST HATEOAS principles
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	response := map[string]interface{}{
		// Service metadata
		"service":     "KIDP Deployment Broker",
		"description": "Stateless broker for provisioning and managing resources in Kubernetes clusters",
		"version":     version,
		"status":      "running",

		// Support information
		"documentation": "https://github.com/aykay76/kidp/blob/master/docs/BROKER_API.md",
		"repository":    "https://github.com/aykay76/kidp",
		"support":       "https://github.com/aykay76/kidp/issues",

		// Capabilities
		"capabilities": map[string]interface{}{
			"resourceTypes": []string{"database", "cache", "queue"},
			"databases":     []string{"postgresql", "mysql", "mongodb", "redis"},
			"features":      []string{"drift-detection", "async-provisioning", "health-monitoring"},
		},

		// API endpoints with detailed information
		"endpoints": map[string]interface{}{
			"health": map[string]interface{}{
				"method":      "GET",
				"path":        "/health",
				"description": "Returns broker service health status",
				"response":    map[string]string{"status": "healthy", "version": "0.1.0"},
			},
			"readiness": map[string]interface{}{
				"method":      "GET",
				"path":        "/readiness",
				"description": "Checks broker readiness (includes Kubernetes API connectivity)",
				"response":    map[string]interface{}{"ready": true, "reason": "ok"},
			},
			"provision": map[string]interface{}{
				"method":      "POST",
				"path":        "/v1/provision",
				"description": "Provision a new resource in the target Kubernetes cluster",
				"contentType": "application/json",
				"request": map[string]interface{}{
					"resourceType": "database",
					"resourceName": "my-db",
					"namespace":    "team-platform",
					"team":         "platform-team",
					"owner":        "user@example.com",
					"callbackUrl":  "http://manager:9090/v1/callback",
					"spec": map[string]interface{}{
						"engine":  "postgresql",
						"version": "15",
						"size":    "medium",
					},
				},
				"response": map[string]string{
					"status":       "accepted",
					"deploymentId": "deploy-abc123",
					"message":      "Provisioning request accepted",
				},
			},
			"deprovision": map[string]interface{}{
				"method":      "POST",
				"path":        "/v1/deprovision",
				"description": "Deprovision an existing resource",
				"contentType": "application/json",
				"request": map[string]interface{}{
					"deploymentId": "deploy-abc123",
					"resourceType": "database",
					"resourceName": "my-db",
					"namespace":    "team-platform",
					"callbackUrl":  "http://manager:9090/v1/callback",
				},
				"response": map[string]string{
					"status":  "accepted",
					"message": "Deprovisioning request accepted",
				},
			},
			"status": map[string]interface{}{
				"method":      "GET",
				"path":        "/v1/status",
				"description": "Get the status of a specific deployment",
				"parameters": map[string]string{
					"id": "deploymentId (required)",
				},
				"example":  "/v1/status?id=deploy-abc123",
				"response": map[string]string{"deploymentId": "deploy-abc123", "phase": "Ready"},
			},
			"resources": map[string]interface{}{
				"methods":     []string{"GET", "POST"},
				"path":        "/v1/resources",
				"description": "Query actual state of resources for drift detection",
				"parameters": map[string]string{
					"namespace":    "namespace (required)",
					"resourceType": "filter by type (optional)",
					"resourceName": "filter by name (optional)",
					"deploymentId": "filter by deployment (optional)",
				},
				"example": "/v1/resources?namespace=team-platform&resourceType=database",
				"features": []string{
					"drift-detection",
					"health-status",
					"resource-usage",
					"cost-tracking",
				},
			},
		},

		// Hypermedia links (HATEOAS)
		"_links": map[string]interface{}{
			"self": map[string]string{
				"href":   "/",
				"method": "GET",
			},
			"health": map[string]string{
				"href":   "/health",
				"method": "GET",
			},
			"readiness": map[string]string{
				"href":   "/readiness",
				"method": "GET",
			},
			"provision": map[string]string{
				"href":   "/v1/provision",
				"method": "POST",
			},
			"deprovision": map[string]string{
				"href":   "/v1/deprovision",
				"method": "POST",
			},
			"resources": map[string]string{
				"href":    "/v1/resources",
				"methods": "GET, POST",
			},
		},

		// API versioning and compatibility
		"api": map[string]interface{}{
			"version":        "v1",
			"minApiVersion":  "v1",
			"deprecatedApis": []string{},
		},

		// Runtime information
		"runtime": map[string]interface{}{
			"uptime":              time.Since(s.startTime).String(),
			"kubernetesConnected": s.k8sClient != nil,
		},
	}

	s.respondJSON(w, http.StatusOK, response)
}

// respondJSON sends a JSON response
func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Printf("Error encoding JSON response: %v", err)
	}
}

// generateDeploymentID creates a unique deployment identifier
func generateDeploymentID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if crypto/rand fails
		return fmt.Sprintf("deploy-%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("deploy-%s", hex.EncodeToString(b))
}

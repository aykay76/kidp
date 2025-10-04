package brokerclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Client is a client for the broker API
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new broker client
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ProvisionRequest represents a provision request to the broker
type ProvisionRequest struct {
	ResourceType string                 `json:"resourceType"`
	ResourceName string                 `json:"resourceName"`
	Namespace    string                 `json:"namespace"`
	Team         string                 `json:"team"`
	Owner        string                 `json:"owner"`
	CallbackURL  string                 `json:"callbackUrl"`
	Spec         map[string]interface{} `json:"spec"`
}

// ProvisionResponse is the broker's response to a provision request
type ProvisionResponse struct {
	Status       string `json:"status"`
	DeploymentID string `json:"deploymentId"`
	Message      string `json:"message"`
}

// DeprovisionRequest represents a deprovision request to the broker
type DeprovisionRequest struct {
	DeploymentID string `json:"deploymentId"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Namespace    string `json:"namespace"`
	CallbackURL  string `json:"callbackUrl"`
}

// DeprovisionResponse is the broker's response to a deprovision request
type DeprovisionResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// Provision requests the broker to provision a resource
func (c *Client) Provision(ctx context.Context, req ProvisionRequest) (*ProvisionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/provision", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "KIDP-Manager/0.1.0")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call broker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("broker returned status %d", resp.StatusCode)
	}

	var provResp ProvisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&provResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &provResp, nil
}

// Deprovision requests the broker to deprovision a resource
func (c *Client) Deprovision(ctx context.Context, req DeprovisionRequest) (*DeprovisionResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/deprovision", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "KIDP-Manager/0.1.0")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to call broker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("broker returned status %d", resp.StatusCode)
	}

	var deprovResp DeprovisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&deprovResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &deprovResp, nil
}

// Ping checks if the broker is reachable
func (c *Client) Ping(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to ping broker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("broker health check failed with status %d", resp.StatusCode)
	}

	return nil
}

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

package broker
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

package broker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// CallbackClient handles webhook callbacks to the manager
type CallbackClient struct {
	httpClient *http.Client
	maxRetries int
}

// NewCallbackClient creates a new callback client with default configuration
func NewCallbackClient() *CallbackClient {
	return &CallbackClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		maxRetries: 3,
	}
}

// NotifyStatus sends a status update to the manager via webhook
// Uses exponential backoff retry strategy: 1s, 2s, 4s
func (c *CallbackClient) NotifyStatus(ctx context.Context, callbackURL string, payload CallbackRequest) error {
	var lastErr error
	
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			backoff := time.Duration(1<<uint(attempt-1)) * time.Second
			log.Printf("Callback attempt %d/%d failed, retrying in %v", attempt, c.maxRetries, backoff)
			
			select {
			case <-time.After(backoff):
				// Continue to retry
			case <-ctx.Done():
				return fmt.Errorf("callback cancelled: %w", ctx.Err())
			}
		}
		
		// Marshal payload to JSON
		body, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal callback payload: %w", err)
		}
		
		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, "POST", callbackURL, bytes.NewBuffer(body))
		if err != nil {
			return fmt.Errorf("failed to create callback request: %w", err)
		}
		
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "KIDP-Broker/0.1.0")
		
		// Log the attempt
		log.Printf("Sending callback to %s (attempt %d/%d): deploymentId=%s, status=%s, phase=%s",
			callbackURL, attempt+1, c.maxRetries, payload.DeploymentID, payload.Status, payload.Phase)
		
		// Send request
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("callback request failed: %w", err)
			log.Printf("Callback request error: %v", lastErr)
			continue
		}
		
		// Check response status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			resp.Body.Close()
			log.Printf("Callback successful: deploymentId=%s, status=%d", payload.DeploymentID, resp.StatusCode)
			return nil
		}
		
		// Non-2xx response
		resp.Body.Close()
		lastErr = fmt.Errorf("callback returned status %d", resp.StatusCode)
		log.Printf("Callback failed with status %d", resp.StatusCode)
	}
	
	return fmt.Errorf("callback failed after %d attempts: %w", c.maxRetries, lastErr)
}

// NotifySuccess is a convenience method to send a success callback
func (c *CallbackClient) NotifySuccess(ctx context.Context, callbackURL, deploymentID, phase string, details map[string]interface{}) error {
	payload := CallbackRequest{
		DeploymentID: deploymentID,
		Status:       "success",
		Phase:        phase,
		Message:      fmt.Sprintf("Successfully completed: %s", phase),
		Details:      details,
	}
	return c.NotifyStatus(ctx, callbackURL, payload)
}

// NotifyFailure is a convenience method to send a failure callback
func (c *CallbackClient) NotifyFailure(ctx context.Context, callbackURL, deploymentID, phase, errorMsg string) error {
	payload := CallbackRequest{
		DeploymentID: deploymentID,
		Status:       "failed",
		Phase:        phase,
		Message:      errorMsg,
		Error:        errorMsg,
	}
	return c.NotifyStatus(ctx, callbackURL, payload)
}

// NotifyProgress is a convenience method to send a progress update
func (c *CallbackClient) NotifyProgress(ctx context.Context, callbackURL, deploymentID, phase, message string) error {
	payload := CallbackRequest{
		DeploymentID: deploymentID,
		Status:       "in-progress",
		Phase:        phase,
		Message:      message,
	}
	return c.NotifyStatus(ctx, callbackURL, payload)
}

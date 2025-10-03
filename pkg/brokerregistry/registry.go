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

package brokerregistry

import (
	"context"
	"fmt"
	"sync"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1 "github.com/aykay76/kidp/api/v1"
)

// Registry manages broker discovery and selection
type Registry struct {
	client       client.Client
	mu           sync.RWMutex
	brokerCache  map[string]*platformv1.Broker
	lastRefresh  time.Time
	cacheTimeout time.Duration
}

// SelectionCriteria defines requirements for broker selection
type SelectionCriteria struct {
	ResourceType  string
	CloudProvider string
	Region        string
	Provider      string // Specific provider (e.g., "postgresql", "azure-sql")
}

// NewRegistry creates a new broker registry
func NewRegistry(client client.Client) *Registry {
	return &Registry{
		client:       client,
		brokerCache:  make(map[string]*platformv1.Broker),
		cacheTimeout: 30 * time.Second,
	}
}

// SelectBroker chooses the best broker based on criteria
func (r *Registry) SelectBroker(ctx context.Context, criteria SelectionCriteria) (*platformv1.Broker, error) {
	log := log.FromContext(ctx)

	// Refresh cache if needed
	if err := r.refreshCacheIfNeeded(ctx); err != nil {
		return nil, fmt.Errorf("failed to refresh broker cache: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var candidates []*platformv1.Broker

	// Filter brokers by criteria
	for _, broker := range r.brokerCache {
		if r.matchesCriteria(broker, criteria) {
			candidates = append(candidates, broker)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no broker found matching criteria: resourceType=%s, cloudProvider=%s, region=%s, provider=%s",
			criteria.ResourceType, criteria.CloudProvider, criteria.Region, criteria.Provider)
	}

	// Select best broker
	selected := r.selectBest(candidates)
	log.Info("Selected broker", "broker", selected.Name, "endpoint", selected.Spec.Endpoint)

	return selected, nil
}

// matchesCriteria checks if a broker matches the selection criteria
func (r *Registry) matchesCriteria(broker *platformv1.Broker, criteria SelectionCriteria) bool {
	// Only consider healthy brokers
	if broker.Status.Phase != "Ready" {
		return false
	}

	// Check cloud provider
	if criteria.CloudProvider != "" && broker.Spec.CloudProvider != criteria.CloudProvider {
		return false
	}

	// Check region
	if criteria.Region != "" && broker.Spec.Region != "" && broker.Spec.Region != criteria.Region {
		return false
	}

	// Check capabilities
	if criteria.ResourceType != "" {
		hasCapability := false
		for _, cap := range broker.Spec.Capabilities {
			if cap.ResourceType == criteria.ResourceType {
				// If specific provider requested, check if broker supports it
				if criteria.Provider != "" {
					for _, p := range cap.Providers {
						if p == criteria.Provider {
							hasCapability = true
							break
						}
					}
				} else {
					hasCapability = true
				}
				if hasCapability {
					break
				}
			}
		}
		if !hasCapability {
			return false
		}
	}

	// Check if broker is at capacity
	if broker.Spec.MaxConcurrentDeployments > 0 &&
		broker.Status.ActiveDeployments >= broker.Spec.MaxConcurrentDeployments {
		return false
	}

	return true
}

// selectBest chooses the best broker from candidates
func (r *Registry) selectBest(candidates []*platformv1.Broker) *platformv1.Broker {
	if len(candidates) == 0 {
		return nil
	}

	best := candidates[0]
	bestScore := r.calculateScore(best)

	for _, broker := range candidates[1:] {
		score := r.calculateScore(broker)
		if score > bestScore {
			best = broker
			bestScore = score
		}
	}

	return best
}

// calculateScore assigns a score to a broker for selection
func (r *Registry) calculateScore(broker *platformv1.Broker) float64 {
	score := float64(0)

	// Higher priority gets higher score
	score += float64(broker.Spec.Priority)

	// Lower load gets higher score
	if broker.Spec.MaxConcurrentDeployments > 0 {
		loadPercentage := float64(broker.Status.ActiveDeployments) / float64(broker.Spec.MaxConcurrentDeployments)
		score += (1.0 - loadPercentage) * 100 // Scale to 0-100
	}

	// Recent heartbeat gets higher score
	if broker.Status.LastHeartbeat != nil {
		age := time.Since(broker.Status.LastHeartbeat.Time)
		if age < 1*time.Minute {
			score += 50
		} else if age < 5*time.Minute {
			score += 25
		}
	}

	return score
}

// refreshCacheIfNeeded refreshes the broker cache if it's expired
func (r *Registry) refreshCacheIfNeeded(ctx context.Context) error {
	r.mu.RLock()
	needsRefresh := time.Since(r.lastRefresh) > r.cacheTimeout
	r.mu.RUnlock()

	if !needsRefresh {
		return nil
	}

	return r.RefreshCache(ctx)
}

// RefreshCache forces a refresh of the broker cache
func (r *Registry) RefreshCache(ctx context.Context) error {
	log := log.FromContext(ctx)

	// List all brokers
	brokerList := &platformv1.BrokerList{}
	if err := r.client.List(ctx, brokerList, &client.ListOptions{}); err != nil {
		return fmt.Errorf("failed to list brokers: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Update cache
	r.brokerCache = make(map[string]*platformv1.Broker)
	for i := range brokerList.Items {
		broker := &brokerList.Items[i]
		key := fmt.Sprintf("%s/%s", broker.Namespace, broker.Name)
		r.brokerCache[key] = broker
	}

	r.lastRefresh = time.Now()
	log.Info("Refreshed broker cache", "count", len(r.brokerCache))

	return nil
}

// ListBrokers returns all cached brokers
func (r *Registry) ListBrokers() []*platformv1.Broker {
	r.mu.RLock()
	defer r.mu.RUnlock()

	brokers := make([]*platformv1.Broker, 0, len(r.brokerCache))
	for _, broker := range r.brokerCache {
		brokers = append(brokers, broker)
	}

	return brokers
}

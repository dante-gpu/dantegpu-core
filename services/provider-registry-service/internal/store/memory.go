package store

import (
	"context"
	"strings"
	"sync"

	"github.com/dante-gpu/dante-backend/provider-registry-service/internal/models"
	"github.com/google/uuid"
)

// InMemoryProviderStore is a simple in-memory store for providers.
// I need to make this thread-safe for concurrent access.
type InMemoryProviderStore struct {
	mu        sync.RWMutex
	providers map[uuid.UUID]*models.Provider
}

// NewInMemoryProviderStore creates a new in-memory provider store.
func NewInMemoryProviderStore() *InMemoryProviderStore {
	return &InMemoryProviderStore{
		providers: make(map[uuid.UUID]*models.Provider),
	}
}

// Initialize sets up the in-memory store. For this implementation,
// there's no need for real initialization since the map is created in the constructor.
func (s *InMemoryProviderStore) Initialize(ctx context.Context) error {
	// Nothing to do for in-memory store
	return nil
}

// Close releases any resources. For this implementation, there are no
// external resources to release.
func (s *InMemoryProviderStore) Close() error {
	// Nothing to close for in-memory store
	return nil
}

// AddProvider adds a new provider to the store.
func (s *InMemoryProviderStore) AddProvider(ctx context.Context, provider *models.Provider) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// I should check if a provider with this ID already exists, though UUIDs should be unique.
	if _, exists := s.providers[provider.ID]; exists {
		return models.ErrProviderAlreadyExists // I'll need to define this error type
	}
	s.providers[provider.ID] = provider
	return nil
}

// GetProvider retrieves a provider by its ID.
func (s *InMemoryProviderStore) GetProvider(ctx context.Context, id uuid.UUID) (*models.Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, exists := s.providers[id]
	if !exists {
		return nil, models.ErrProviderNotFound // I'll need to define this error type
	}
	return provider, nil
}

// ListProviders returns a list of all providers with optional filtering.
// For now, it returns all providers.
func (s *InMemoryProviderStore) ListProviders(ctx context.Context, filters map[string]interface{}) ([]*models.Provider, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Start with all providers
	var filtered []*models.Provider
	for _, provider := range s.providers {
		// Apply filters if any
		if passesFilters(provider, filters) {
			filtered = append(filtered, provider)
		}
	}

	return filtered, nil
}

// passesFilters checks if a provider passes all the provided filters.
func passesFilters(provider *models.Provider, filters map[string]interface{}) bool {
	if filters == nil || len(filters) == 0 {
		return true // No filters, pass everything
	}

	// Check status filter
	if statusFilter, ok := filters["status"].(string); ok && statusFilter != "" {
		if string(provider.Status) != statusFilter {
			return false
		}
	}

	// Check minimum VRAM
	if minVRAM, ok := filters["min_vram"].(uint64); ok && minVRAM > 0 {
		hasGPUWithEnoughVRAM := false
		for _, gpu := range provider.GPUs {
			if gpu.VRAM >= minVRAM {
				hasGPUWithEnoughVRAM = true
				break
			}
		}
		if !hasGPUWithEnoughVRAM {
			return false
		}
	}

	// Check GPU model
	if gpuModel, ok := filters["gpu_model"].(string); ok && gpuModel != "" {
		hasMatchingModel := false
		for _, gpu := range provider.GPUs {
			if strings.Contains(strings.ToLower(gpu.ModelName), strings.ToLower(gpuModel)) {
				hasMatchingModel = true
				break
			}
		}
		if !hasMatchingModel {
			return false
		}
	}

	// Check architecture
	if arch, ok := filters["architecture"].(string); ok && arch != "" {
		hasMatchingArch := false
		for _, gpu := range provider.GPUs {
			if strings.Contains(strings.ToLower(gpu.Architecture), strings.ToLower(arch)) {
				hasMatchingArch = true
				break
			}
		}
		if !hasMatchingArch {
			return false
		}
	}

	// Check for healthy GPUs only
	if healthyOnly, ok := filters["healthy_only"].(bool); ok && healthyOnly {
		for _, gpu := range provider.GPUs {
			if !gpu.IsHealthy {
				return false // At least one GPU is unhealthy
			}
		}
	}

	return true // Passed all filters
}

// UpdateProvider updates an existing provider in the store.
func (s *InMemoryProviderStore) UpdateProvider(ctx context.Context, id uuid.UUID, updatedProvider *models.Provider) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, exists := s.providers[id]
	if !exists {
		return models.ErrProviderNotFound
	}
	// I should ensure the ID is not changed during an update.
	updatedProvider.ID = id
	s.providers[id] = updatedProvider
	return nil
}

// DeleteProvider removes a provider from the store.
func (s *InMemoryProviderStore) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.providers[id]; !exists {
		return models.ErrProviderNotFound
	}
	delete(s.providers, id)
	return nil
}

// UpdateProviderStatus updates the status of a specific provider.
func (s *InMemoryProviderStore) UpdateProviderStatus(ctx context.Context, id uuid.UUID, status models.ProviderStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider, exists := s.providers[id]
	if !exists {
		return models.ErrProviderNotFound
	}
	provider.UpdateStatus(status)
	s.providers[id] = provider // Re-assign to map if provider is a copy (though it's a pointer here)
	return nil
}

// UpdateProviderHeartbeat updates the LastSeenAt timestamp for a provider
// and updates GPU metrics if provided.
func (s *InMemoryProviderStore) UpdateProviderHeartbeat(ctx context.Context, id uuid.UUID, gpuMetrics []models.GPUDetail) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider, exists := s.providers[id]
	if !exists {
		return models.ErrProviderNotFound
	}

	provider.Heartbeat()

	// Update GPU metrics if provided
	if len(gpuMetrics) > 0 && len(provider.GPUs) > 0 {
		// Update existing GPUs with new metrics
		// We only update metrics for GPUs that exist, up to the length of what's provided
		for i := 0; i < len(gpuMetrics) && i < len(provider.GPUs); i++ {
			// Only update the utilization and health metrics
			provider.GPUs[i].UtilizationGPU = gpuMetrics[i].UtilizationGPU
			provider.GPUs[i].UtilizationMem = gpuMetrics[i].UtilizationMem
			provider.GPUs[i].Temperature = gpuMetrics[i].Temperature
			provider.GPUs[i].PowerDraw = gpuMetrics[i].PowerDraw
			provider.GPUs[i].IsHealthy = gpuMetrics[i].IsHealthy
		}
	}

	s.providers[id] = provider
	return nil
}

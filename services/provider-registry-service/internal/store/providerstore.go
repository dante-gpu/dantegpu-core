package store

import (
	"context"

	"github.com/dante-gpu/dante-backend/provider-registry-service/internal/models"
	"github.com/google/uuid"
)

// ProviderStore defines the interface for provider storage operations.
// This allows different storage backends to be used interchangeably.
type ProviderStore interface {
	// Initialize sets up any necessary structures or connections
	Initialize(ctx context.Context) error

	// AddProvider stores a new provider in the system
	AddProvider(ctx context.Context, provider *models.Provider) error

	// GetProvider retrieves a provider by ID
	GetProvider(ctx context.Context, id uuid.UUID) (*models.Provider, error)

	// ListProviders retrieves all providers, with optional filtering
	// TODO: Add filtering parameters
	ListProviders(ctx context.Context, filters map[string]interface{}) ([]*models.Provider, error)

	// UpdateProvider updates an existing provider
	UpdateProvider(ctx context.Context, id uuid.UUID, updatedProvider *models.Provider) error

	// DeleteProvider removes a provider from the system
	DeleteProvider(ctx context.Context, id uuid.UUID) error

	// UpdateProviderStatus updates only the status of a provider
	UpdateProviderStatus(ctx context.Context, id uuid.UUID, status models.ProviderStatus) error

	// UpdateProviderHeartbeat updates the last_seen_at timestamp for a provider
	// and updates GPU utilization metrics if provided
	UpdateProviderHeartbeat(ctx context.Context, id uuid.UUID, gpuMetrics []models.GPUDetail) error

	// Close cleans up any resources used by the store
	Close() error
}

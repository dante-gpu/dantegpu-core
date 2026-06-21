package clients

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/config"
	consul_client "github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/consul"
	"github.com/google/uuid"
	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

// --- Structs to represent Provider data (must match provider-registry-service models) ---

// ProviderStatus represents the possible states of a GPU provider.
type ProviderStatus string

const (
	StatusIdle        ProviderStatus = "idle"
	StatusBusy        ProviderStatus = "busy"
	StatusOffline     ProviderStatus = "offline"
	StatusMaintenance ProviderStatus = "maintenance"
	StatusError       ProviderStatus = "error"
)

// GPUDetail holds specific information about a GPU.
// This must match the models.GPUDetail in provider-registry-service.
type GPUDetail struct {
	ModelName     string `json:"model_name"`
	VRAM          uint64 `json:"vram_mb"` // VRAM in Megabytes
	DriverVersion string `json:"driver_version"`
}

// Provider represents a registered GPU provider as returned by the provider-registry-service.
// This must match the models.Provider in provider-registry-service.
type Provider struct {
	ID           uuid.UUID              `json:"id"`
	OwnerID      string                 `json:"owner_id"`
	Name         string                 `json:"name"`
	Hostname     string                 `json:"hostname,omitempty"`
	IPAddress    string                 `json:"ip_address,omitempty"`
	Status       ProviderStatus         `json:"status"`
	GPUs         []GPUDetail            `json:"gpus"`
	Location     string                 `json:"location,omitempty"`
	RegisteredAt time.Time              `json:"registered_at"`
	LastSeenAt   time.Time              `json:"last_seen_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Client is an HTTP client for interacting with the Provider Registry service.
type Client struct {
	httpClient       *http.Client
	logger           *zap.Logger
	consulClient     *consulapi.Client
	cfg              *config.Config
	targetService    string // Name of the provider-registry service in Consul
	lastKnownAddress string
	mu               sync.RWMutex // To protect lastKnownAddress
}

// NewClient creates a new Provider Registry client.
func NewClient(cfg *config.Config, consulClient *consulapi.Client, logger *zap.Logger) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: cfg.ProviderQueryTimeout, // Use timeout from config
		},
		logger:        logger,
		consulClient:  consulClient,
		cfg:           cfg,
		targetService: cfg.ProviderRegistryServiceName,
	}
}

// getServiceAddress discovers the Provider Registry service using Consul and returns a base URL.
// It implements a simple cache for the last known address to reduce Consul lookups.
func (c *Client) getServiceAddress() (string, error) {
	c.mu.RLock()
	if c.lastKnownAddress != "" {
		c.mu.RUnlock()
		// TODO: Add a mechanism to invalidate this cache periodically or on error
		c.logger.Debug("Using cached address for provider registry service", zap.String("address", c.lastKnownAddress))
		return c.lastKnownAddress, nil
	}
	c.mu.RUnlock()

	c.logger.Info("Discovering provider registry service via Consul", zap.String("service_name", c.targetService))
	serviceEntries, err := consul_client.DiscoverService(c.consulClient, c.targetService, c.logger)
	if err != nil {
		return "", fmt.Errorf("failed to discover %s service: %w", c.targetService, err)
	}
	if len(serviceEntries) == 0 {
		return "", fmt.Errorf("no healthy instances found for %s service", c.targetService)
	}

	// Simple load balancing: pick a random healthy instance
	selected := serviceEntries[rand.Intn(len(serviceEntries))]
	address := selected.Service.Address
	if address == "" {
		address = selected.Node.Address // Fallback to node address
	}
	scheme := "http" // Assuming http for now, can be made configurable or detected from tags/meta

	serviceURL := fmt.Sprintf("%s://%s:%d", scheme, address, selected.Service.Port)

	c.mu.Lock()
	c.lastKnownAddress = serviceURL
	c.mu.Unlock()

	c.logger.Info("Discovered provider registry service instance", zap.String("url", serviceURL))
	return serviceURL, nil
}

// ListAvailableProviders fetches a list of all providers from the Provider Registry service.
// TODO: Add filtering capabilities based on job requirements.
func (c *Client) ListAvailableProviders() ([]Provider, error) {
	baseURL, err := c.getServiceAddress()
	if err != nil {
		return nil, err
	}

	// The endpoint for listing providers is typically "/providers"
	requestURL := baseURL + "/providers"

	c.logger.Debug("Fetching providers from registry", zap.String("url", requestURL))

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		c.logger.Error("Failed to create request for provider list", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to get provider list from registry service", zap.Error(err), zap.String("url", requestURL))
		c.invalidateCachedAddress() // Invalidate cache on error
		return nil, fmt.Errorf("failed to get provider list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("Provider registry service returned non-OK status for provider list",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", requestURL),
		)
		c.invalidateCachedAddress() // Invalidate cache on non-OK status
		return nil, fmt.Errorf("provider registry service returned status %d for provider list", resp.StatusCode)
	}

	var providers []Provider
	if err := json.NewDecoder(resp.Body).Decode(&providers); err != nil {
		c.logger.Error("Failed to decode provider list response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode provider list: %w", err)
	}

	c.logger.Info("Successfully fetched providers", zap.Int("count", len(providers)))
	return providers, nil
}

// invalidateCachedAddress clears the last known address, forcing a new Consul lookup on next call.
func (c *Client) invalidateCachedAddress() {
	c.mu.Lock()
	c.lastKnownAddress = ""
	c.mu.Unlock()
	c.logger.Info("Invalidated cached address for provider registry service")
}

// GetProviderByID fetches a specific provider by its ID.
// This is an example, implement if needed.
/*
func (c *Client) GetProviderByID(providerID uuid.UUID) (*Provider, error) {
	baseURL, err := c.getServiceAddress()
	if err != nil {
		return nil, err
	}

	requestURL := fmt.Sprintf("%s/providers/%s", baseURL, providerID.String())
	c.logger.Debug("Fetching provider by ID from registry", zap.String("url", requestURL))

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for get provider by id: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.invalidateCachedAddress()
		return nil, fmt.Errorf("failed to get provider by id: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("provider with ID %s not found", providerID.String()) // Or a specific error type
	}
	if resp.StatusCode != http.StatusOK {
		c.invalidateCachedAddress()
		return nil, fmt.Errorf("provider registry service returned status %d for get provider by id", resp.StatusCode)
	}

	var provider Provider
	if err := json.NewDecoder(resp.Body).Decode(&provider); err != nil {
		return nil, fmt.Errorf("failed to decode provider response: %w", err)
	}
	return &provider, nil
}
*/

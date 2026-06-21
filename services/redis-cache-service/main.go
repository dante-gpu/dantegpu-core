package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
)

// RedisCacheService provides optimized caching for GPU rental platform
type RedisCacheService struct {
	client *redis.Client
	logger *zap.Logger
	ctx    context.Context
}

// CacheConfig holds cache configuration
type CacheConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// GPUAvailabilityCache represents cached GPU availability data
type GPUAvailabilityCache struct {
	GPUID           string    `json:"gpu_id"`
	ProviderID      string    `json:"provider_id"`
	IsAvailable     bool      `json:"is_available"`
	UtilizationGPU  int       `json:"utilization_gpu"`
	UtilizationMem  int       `json:"utilization_memory"`
	MemoryFree      int64     `json:"memory_free_mb"`
	MemoryTotal     int64     `json:"memory_total_mb"`
	Temperature     int       `json:"temperature_c"`
	PowerDraw       int       `json:"power_draw_w"`
	HourlyRateDGPU  float64   `json:"hourly_rate_dgpu"`
	HourlyRateUSD   float64   `json:"hourly_rate_usd"`
	LastUpdated     time.Time `json:"last_updated"`
	ActiveJobs      int       `json:"active_jobs"`
	QueuedJobs      int       `json:"queued_jobs"`
	EstimatedWait   int       `json:"estimated_wait_minutes"`
}

// UserSessionCache represents cached user session data
type UserSessionCache struct {
	UserID      string                 `json:"user_id"`
	Username    string                 `json:"username"`
	Role        string                 `json:"role"`
	Permissions []string               `json:"permissions"`
	Preferences map[string]interface{} `json:"preferences"`
	LastActive  time.Time              `json:"last_active"`
}

// PricingCache represents cached pricing information
type PricingCache struct {
	GPUModel       string    `json:"gpu_model"`
	BaseRateDGPU   float64   `json:"base_rate_dgpu"`
	BaseRateUSD    float64   `json:"base_rate_usd"`
	DemandMultiple float64   `json:"demand_multiple"`
	SupplyFactor   float64   `json:"supply_factor"`
	PeakHourFactor float64   `json:"peak_hour_factor"`
	FinalRateDGPU  float64   `json:"final_rate_dgpu"`
	FinalRateUSD   float64   `json:"final_rate_usd"`
	LastUpdated    time.Time `json:"last_updated"`
}

// JobStatusCache represents cached job status
type JobStatusCache struct {
	JobID           string    `json:"job_id"`
	UserID          string    `json:"user_id"`
	Status          string    `json:"status"`
	Progress        int       `json:"progress_percent"`
	EstimatedEnd    time.Time `json:"estimated_end"`
	CurrentCostDGPU float64   `json:"current_cost_dgpu"`
	CurrentCostUSD  float64   `json:"current_cost_usd"`
	LastUpdated     time.Time `json:"last_updated"`
}

// Cache key constants
const (
	// GPU availability keys
	KeyGPUAvailability     = "gpu:availability:%s"           // gpu_id
	KeyProviderGPUs        = "provider:gpus:%s"              // provider_id
	KeyAvailableGPUs       = "gpus:available"                // list of available GPU IDs
	KeyGPUsByModel         = "gpus:model:%s"                 // gpu_model
	KeyGPUUtilization      = "gpu:utilization:%s"            // gpu_id
	
	// User session keys
	KeyUserSession         = "user:session:%s"               // user_id
	KeyUserPreferences     = "user:preferences:%s"           // user_id
	KeyUserActiveJobs      = "user:jobs:active:%s"           // user_id
	KeyUserWallet          = "user:wallet:%s"                // user_id
	
	// Pricing keys
	KeyPricingModel        = "pricing:model:%s"              // gpu_model
	KeyPricingDemand       = "pricing:demand:%s"             // gpu_model
	KeyPricingSupply       = "pricing:supply:%s"             // gpu_model
	KeyExchangeRates       = "pricing:exchange_rates"
	
	// Job status keys
	KeyJobStatus           = "job:status:%s"                 // job_id
	KeyJobLogs             = "job:logs:%s"                   // job_id
	KeyJobMetrics          = "job:metrics:%s"                // job_id
	KeyUserJobQueue        = "user:job_queue:%s"             // user_id
	
	// System statistics keys
	KeySystemStats         = "system:stats"
	KeyProviderStats       = "provider:stats:%s"             // provider_id
	KeyGlobalMetrics       = "metrics:global"
	KeyAlerts              = "system:alerts"
	
	// Rate limiting keys
	KeyRateLimit           = "rate_limit:%s:%s"              // user_id:endpoint
	KeyAPIQuota            = "api:quota:%s"                  // user_id
)

// Cache TTL constants (in seconds)
const (
	TTLGPUAvailability  = 5     // 5 seconds for real-time GPU data
	TTLUserSession      = 3600  // 1 hour for user sessions
	TTLPricing          = 300   // 5 minutes for pricing data
	TTLJobStatus        = 10    // 10 seconds for job status
	TTLSystemStats      = 30    // 30 seconds for system stats
	TTLRateLimit        = 60    // 1 minute for rate limiting
	TTLUserPreferences  = 1800  // 30 minutes for user preferences
	TTLExchangeRates    = 900   // 15 minutes for exchange rates
)

// NewRedisCacheService creates a new Redis cache service
func NewRedisCacheService(config CacheConfig, logger *zap.Logger) *RedisCacheService {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})
	
	return &RedisCacheService{
		client: rdb,
		logger: logger,
		ctx:    context.Background(),
	}
}

// GPU Availability Cache Methods

// SetGPUAvailability caches GPU availability data
func (rcs *RedisCacheService) SetGPUAvailability(gpuID string, data GPUAvailabilityCache) error {
	key := fmt.Sprintf(KeyGPUAvailability, gpuID)
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal GPU availability data: %w", err)
	}
	
	err = rcs.client.Set(rcs.ctx, key, jsonData, TTLGPUAvailability*time.Second).Err()
	if err != nil {
		return fmt.Errorf("failed to cache GPU availability: %w", err)
	}
	
	// Update available GPUs list if GPU is available
	if data.IsAvailable {
		rcs.client.SAdd(rcs.ctx, KeyAvailableGPUs, gpuID)
		rcs.client.Expire(rcs.ctx, KeyAvailableGPUs, TTLGPUAvailability*time.Second)
	} else {
		rcs.client.SRem(rcs.ctx, KeyAvailableGPUs, gpuID)
	}
	
	// Update GPU by model index
	modelKey := fmt.Sprintf(KeyGPUsByModel, data.ProviderID) // Using provider as model for now
	rcs.client.SAdd(rcs.ctx, modelKey, gpuID)
	rcs.client.Expire(rcs.ctx, modelKey, TTLGPUAvailability*time.Second)
	
	return nil
}

// GetGPUAvailability retrieves cached GPU availability data
func (rcs *RedisCacheService) GetGPUAvailability(gpuID string) (*GPUAvailabilityCache, error) {
	key := fmt.Sprintf(KeyGPUAvailability, gpuID)
	
	val, err := rcs.client.Get(rcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get GPU availability from cache: %w", err)
	}
	
	var data GPUAvailabilityCache
	err = json.Unmarshal([]byte(val), &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal GPU availability data: %w", err)
	}
	
	return &data, nil
}

// GetAvailableGPUs returns list of available GPU IDs
func (rcs *RedisCacheService) GetAvailableGPUs() ([]string, error) {
	gpuIDs, err := rcs.client.SMembers(rcs.ctx, KeyAvailableGPUs).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get available GPUs: %w", err)
	}
	return gpuIDs, nil
}

// User Session Cache Methods

// SetUserSession caches user session data
func (rcs *RedisCacheService) SetUserSession(userID string, session UserSessionCache) error {
	key := fmt.Sprintf(KeyUserSession, userID)
	
	jsonData, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal user session: %w", err)
	}
	
	return rcs.client.Set(rcs.ctx, key, jsonData, TTLUserSession*time.Second).Err()
}

// GetUserSession retrieves cached user session
func (rcs *RedisCacheService) GetUserSession(userID string) (*UserSessionCache, error) {
	key := fmt.Sprintf(KeyUserSession, userID)
	
	val, err := rcs.client.Get(rcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	
	var session UserSessionCache
	err = json.Unmarshal([]byte(val), &session)
	return &session, err
}

// Pricing Cache Methods

// SetPricing caches pricing information
func (rcs *RedisCacheService) SetPricing(gpuModel string, pricing PricingCache) error {
	key := fmt.Sprintf(KeyPricingModel, gpuModel)
	
	jsonData, err := json.Marshal(pricing)
	if err != nil {
		return err
	}
	
	return rcs.client.Set(rcs.ctx, key, jsonData, TTLPricing*time.Second).Err()
}

// GetPricing retrieves cached pricing
func (rcs *RedisCacheService) GetPricing(gpuModel string) (*PricingCache, error) {
	key := fmt.Sprintf(KeyPricingModel, gpuModel)
	
	val, err := rcs.client.Get(rcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	
	var pricing PricingCache
	err = json.Unmarshal([]byte(val), &pricing)
	return &pricing, err
}

// Job Status Cache Methods

// SetJobStatus caches job status
func (rcs *RedisCacheService) SetJobStatus(jobID string, status JobStatusCache) error {
	key := fmt.Sprintf(KeyJobStatus, jobID)
	
	jsonData, err := json.Marshal(status)
	if err != nil {
		return err
	}
	
	err = rcs.client.Set(rcs.ctx, key, jsonData, TTLJobStatus*time.Second).Err()
	if err != nil {
		return err
	}
	
	// Add to user's active jobs if status is active
	if status.Status == "running" || status.Status == "queued" {
		userJobsKey := fmt.Sprintf(KeyUserActiveJobs, status.UserID)
		rcs.client.SAdd(rcs.ctx, userJobsKey, jobID)
		rcs.client.Expire(rcs.ctx, userJobsKey, TTLJobStatus*time.Second)
	}
	
	return nil
}

// GetJobStatus retrieves cached job status
func (rcs *RedisCacheService) GetJobStatus(jobID string) (*JobStatusCache, error) {
	key := fmt.Sprintf(KeyJobStatus, jobID)
	
	val, err := rcs.client.Get(rcs.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	
	var status JobStatusCache
	err = json.Unmarshal([]byte(val), &status)
	return &status, err
}

// Rate Limiting Methods

// CheckRateLimit checks and updates rate limit for user/endpoint
func (rcs *RedisCacheService) CheckRateLimit(userID, endpoint string, limit int) (bool, int, error) {
	key := fmt.Sprintf(KeyRateLimit, userID, endpoint)
	
	// Get current count
	current, err := rcs.client.Get(rcs.ctx, key).Int()
	if err != nil && err != redis.Nil {
		return false, 0, err
	}
	
	if current >= limit {
		return false, current, nil
	}
	
	// Increment counter
	pipe := rcs.client.Pipeline()
	pipe.Incr(rcs.ctx, key)
	pipe.Expire(rcs.ctx, key, TTLRateLimit*time.Second)
	_, err = pipe.Exec(rcs.ctx)
	
	if err != nil {
		return false, current, err
	}
	
	return true, current + 1, nil
}

// System Statistics Cache Methods

// SetSystemStats caches system statistics
func (rcs *RedisCacheService) SetSystemStats(stats map[string]interface{}) error {
	jsonData, err := json.Marshal(stats)
	if err != nil {
		return err
	}
	
	return rcs.client.Set(rcs.ctx, KeySystemStats, jsonData, TTLSystemStats*time.Second).Err()
}

// GetSystemStats retrieves cached system statistics
func (rcs *RedisCacheService) GetSystemStats() (map[string]interface{}, error) {
	val, err := rcs.client.Get(rcs.ctx, KeySystemStats).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}
	
	var stats map[string]interface{}
	err = json.Unmarshal([]byte(val), &stats)
	return stats, err
}

// Utility Methods

// InvalidateUserCache invalidates all cache entries for a user
func (rcs *RedisCacheService) InvalidateUserCache(userID string) error {
	pattern := fmt.Sprintf("user:*:%s", userID)
	keys, err := rcs.client.Keys(rcs.ctx, pattern).Result()
	if err != nil {
		return err
	}
	
	if len(keys) > 0 {
		return rcs.client.Del(rcs.ctx, keys...).Err()
	}
	
	return nil
}

// InvalidateGPUCache invalidates cache entries for a GPU
func (rcs *RedisCacheService) InvalidateGPUCache(gpuID string) error {
	keys := []string{
		fmt.Sprintf(KeyGPUAvailability, gpuID),
		fmt.Sprintf(KeyGPUUtilization, gpuID),
	}
	
	return rcs.client.Del(rcs.ctx, keys...).Err()
}

// GetCacheStats returns cache statistics
func (rcs *RedisCacheService) GetCacheStats() (map[string]interface{}, error) {
	info, err := rcs.client.Info(rcs.ctx, "memory", "stats").Result()
	if err != nil {
		return nil, err
	}
	
	stats := make(map[string]interface{})
	lines := strings.Split(info, "\n")
	
	for _, line := range lines {
		if strings.Contains(line, ":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				
				// Try to convert to number
				if intVal, err := strconv.Atoi(value); err == nil {
					stats[key] = intVal
				} else if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					stats[key] = floatVal
				} else {
					stats[key] = value
				}
			}
		}
	}
	
	return stats, nil
}

// Health check
func (rcs *RedisCacheService) HealthCheck() error {
	return rcs.client.Ping(rcs.ctx).Err()
}

// Close closes the Redis connection
func (rcs *RedisCacheService) Close() error {
	return rcs.client.Close()
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	
	config := CacheConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       0,
	}
	
	cache := NewRedisCacheService(config, logger)
	defer cache.Close()
	
	// Test connection
	if err := cache.HealthCheck(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	
	logger.Info("Redis cache service initialized successfully")
	
	// Example usage
	gpuData := GPUAvailabilityCache{
		GPUID:          "gpu-001",
		ProviderID:     "provider-123",
		IsAvailable:    true,
		UtilizationGPU: 45,
		MemoryFree:     16384,
		MemoryTotal:    24576,
		HourlyRateDGPU: 2.5,
		LastUpdated:    time.Now(),
	}
	
	if err := cache.SetGPUAvailability("gpu-001", gpuData); err != nil {
		logger.Error("Failed to cache GPU data", zap.Error(err))
	} else {
		logger.Info("GPU data cached successfully")
	}
}

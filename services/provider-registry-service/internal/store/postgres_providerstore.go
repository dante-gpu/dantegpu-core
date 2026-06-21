package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dante-gpu/dante-backend/provider-registry-service/internal/models"
	"github.com/dante-gpu/dante-backend/provider-registry-service/internal/retryer"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// PostgresProviderStore is a PostgreSQL implementation of the ProviderStore interface.
type PostgresProviderStore struct {
	db          *pgxpool.Pool
	logger      *zap.Logger
	retryConfig retryer.DatabaseRetryConfig
}

// NewPostgresProviderStore creates a new PostgresProviderStore.
func NewPostgresProviderStore(db *pgxpool.Pool, logger *zap.Logger) *PostgresProviderStore {
	return &PostgresProviderStore{
		db:          db,
		logger:      logger,
		retryConfig: retryer.DefaultRetryConfig(),
	}
}

// Initialize sets up the PostgreSQL tables if they don't exist.
func (pps *PostgresProviderStore) Initialize(ctx context.Context) error {
	// Create providers table if it doesn't exist
	sqlProviders := `
	CREATE TABLE IF NOT EXISTS providers (
		id UUID PRIMARY KEY,
		owner_id TEXT NOT NULL,
		name TEXT NOT NULL,
		hostname TEXT,
		ip_address TEXT,
		status TEXT NOT NULL,
		location TEXT,
		registered_at TIMESTAMPTZ NOT NULL,
		last_seen_at TIMESTAMPTZ NOT NULL,
		metadata JSONB,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Create indexes on frequently queried columns
	CREATE INDEX IF NOT EXISTS idx_providers_owner_id ON providers(owner_id);
	CREATE INDEX IF NOT EXISTS idx_providers_status ON providers(status);
	CREATE INDEX IF NOT EXISTS idx_providers_name ON providers(name);
	CREATE INDEX IF NOT EXISTS idx_providers_last_seen_at ON providers(last_seen_at);
	`

	// Create GPU details table
	sqlGPU := `
	CREATE TABLE IF NOT EXISTS gpu_details (
		id SERIAL PRIMARY KEY,
		provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
		model_name TEXT NOT NULL,
		vram_mb BIGINT NOT NULL,
		driver_version TEXT,
		architecture TEXT,
		compute_capability TEXT,
		cuda_cores INTEGER,
		tensor_cores INTEGER,
		memory_bandwidth_gb_s INTEGER,
		power_consumption_w INTEGER,
		utilization_gpu_percent SMALLINT,
		utilization_memory_percent SMALLINT,
		temperature_c SMALLINT,
		power_draw_w INTEGER,
		is_healthy BOOLEAN DEFAULT TRUE,
		created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	);

	-- Create indexes for GPU details table
	CREATE INDEX IF NOT EXISTS idx_gpu_details_provider_id ON gpu_details(provider_id);
	CREATE INDEX IF NOT EXISTS idx_gpu_details_model_name ON gpu_details(model_name);
	CREATE INDEX IF NOT EXISTS idx_gpu_details_vram ON gpu_details(vram_mb);
	CREATE INDEX IF NOT EXISTS idx_gpu_details_architecture ON gpu_details(architecture);
	CREATE INDEX IF NOT EXISTS idx_gpu_details_is_healthy ON gpu_details(is_healthy);
	`

	// Execute the table creation queries with retry
	return retryer.WithRetry(ctx, pps.logger, pps.retryConfig, "initialize database tables", func() error {
		// Execute providers table creation
		if _, err := pps.db.Exec(ctx, sqlProviders); err != nil {
			return fmt.Errorf("failed to create providers table: %w", err)
		}

		// Execute GPU details table creation
		if _, err := pps.db.Exec(ctx, sqlGPU); err != nil {
			return fmt.Errorf("failed to create gpu_details table: %w", err)
		}

		pps.logger.Info("PostgreSQL tables initialized for provider store")
		return nil
	})
}

// AddProvider adds a new provider to the PostgreSQL database.
func (pps *PostgresProviderStore) AddProvider(ctx context.Context, provider *models.Provider) error {
	return retryer.WithRetry(ctx, pps.logger, pps.retryConfig, "add provider", func() error {
		// Start transaction
		tx, err := pps.db.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		// Ensure rollback on error
		defer func() {
			if err != nil {
				if rbErr := tx.Rollback(ctx); rbErr != nil {
					pps.logger.Error("Transaction rollback failed", zap.Error(rbErr))
				}
			}
		}()

		// Insert provider record
		sqlProvider := `
		INSERT INTO providers (
			id, owner_id, name, hostname, ip_address, status, location, 
			registered_at, last_seen_at, metadata
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`

		// Convert metadata to JSON if it exists
		var metadataJSON []byte
		var err1 error
		if provider.Metadata != nil {
			metadataJSON, err1 = json.Marshal(provider.Metadata)
			if err1 != nil {
				err = fmt.Errorf("failed to marshal metadata: %w", err1)
				return err
			}
		}

		_, err = tx.Exec(
			ctx,
			sqlProvider,
			provider.ID,
			provider.OwnerID,
			provider.Name,
			provider.Hostname,
			provider.IPAddress,
			provider.Status,
			provider.Location,
			provider.RegisteredAt,
			provider.LastSeenAt,
			metadataJSON,
		)
		if err != nil {
			err = fmt.Errorf("failed to insert provider: %w", err)
			return err
		}

		// Insert each GPU detail
		sqlGPU := `
		INSERT INTO gpu_details (
			provider_id, model_name, vram_mb, driver_version, 
			architecture, compute_capability, cuda_cores, tensor_cores, 
			memory_bandwidth_gb_s, power_consumption_w, 
			utilization_gpu_percent, utilization_memory_percent, 
			temperature_c, power_draw_w, is_healthy
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		`

		for _, gpu := range provider.GPUs {
			_, err = tx.Exec(ctx, sqlGPU,
				provider.ID,
				gpu.ModelName,
				gpu.VRAM,
				gpu.DriverVersion,
				gpu.Architecture,
				gpu.ComputeCapability,
				gpu.CudaCores,
				gpu.TensorCores,
				gpu.MemoryBandwidth,
				gpu.PowerConsumption,
				gpu.UtilizationGPU,
				gpu.UtilizationMem,
				gpu.Temperature,
				gpu.PowerDraw,
				gpu.IsHealthy,
			)
			if err != nil {
				err = fmt.Errorf("failed to insert GPU detail: %w", err)
				return err
			}
		}

		// Commit the transaction
		if err = tx.Commit(ctx); err != nil {
			err = fmt.Errorf("failed to commit transaction: %w", err)
			return err
		}

		return nil
	})
}

// GetProvider retrieves a provider by its ID.
func (pps *PostgresProviderStore) GetProvider(ctx context.Context, id uuid.UUID) (*models.Provider, error) {
	var provider models.Provider
	provider.ID = id

	return &provider, retryer.WithRetry(ctx, pps.logger, pps.retryConfig, "get provider", func() error {
		// Query the provider
		sqlProvider := `
		SELECT 
			id, owner_id, name, hostname, ip_address, status, location, 
			registered_at, last_seen_at, metadata
		FROM providers 
		WHERE id = $1
		`

		var metadataJSON []byte
		err := pps.db.QueryRow(ctx, sqlProvider, id).Scan(
			&provider.ID,
			&provider.OwnerID,
			&provider.Name,
			&provider.Hostname,
			&provider.IPAddress,
			&provider.Status,
			&provider.Location,
			&provider.RegisteredAt,
			&provider.LastSeenAt,
			&metadataJSON,
		)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("provider not found: %w", err)
			}
			return fmt.Errorf("failed to query provider: %w", err)
		}

		// Unmarshal metadata if present
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &provider.Metadata); err != nil {
				return fmt.Errorf("failed to unmarshal metadata: %w", err)
			}
		}

		// Query GPU details
		sqlGPU := `
		SELECT 
			model_name, vram_mb, driver_version, 
			architecture, compute_capability, cuda_cores, tensor_cores, 
			memory_bandwidth_gb_s, power_consumption_w, 
			utilization_gpu_percent, utilization_memory_percent, 
			temperature_c, power_draw_w, is_healthy
		FROM gpu_details 
		WHERE provider_id = $1
		`

		rows, err := pps.db.Query(ctx, sqlGPU, id)
		if err != nil {
			return fmt.Errorf("failed to query GPU details: %w", err)
		}
		defer rows.Close()

		provider.GPUs = []models.GPUDetail{}
		for rows.Next() {
			var gpu models.GPUDetail

			// Temporary variables for nullable fields
			var architecture, computeCapability sql.NullString
			var cudaCores, tensorCores, memoryBandwidth, powerConsumption sql.NullInt64
			var utilizationGPU, utilizationMem, temperature sql.NullInt16
			var powerDraw sql.NullInt64
			var isHealthy sql.NullBool

			err := rows.Scan(
				&gpu.ModelName,
				&gpu.VRAM,
				&gpu.DriverVersion,
				&architecture,
				&computeCapability,
				&cudaCores,
				&tensorCores,
				&memoryBandwidth,
				&powerConsumption,
				&utilizationGPU,
				&utilizationMem,
				&temperature,
				&powerDraw,
				&isHealthy,
			)
			if err != nil {
				return fmt.Errorf("failed to scan GPU detail: %w", err)
			}

			// Set values from nullable fields
			if architecture.Valid {
				gpu.Architecture = architecture.String
			}
			if computeCapability.Valid {
				gpu.ComputeCapability = computeCapability.String
			}
			if cudaCores.Valid {
				gpu.CudaCores = uint32(cudaCores.Int64)
			}
			if tensorCores.Valid {
				gpu.TensorCores = uint32(tensorCores.Int64)
			}
			if memoryBandwidth.Valid {
				gpu.MemoryBandwidth = uint64(memoryBandwidth.Int64)
			}
			if powerConsumption.Valid {
				gpu.PowerConsumption = uint32(powerConsumption.Int64)
			}
			if utilizationGPU.Valid {
				gpu.UtilizationGPU = uint8(utilizationGPU.Int16)
			}
			if utilizationMem.Valid {
				gpu.UtilizationMem = uint8(utilizationMem.Int16)
			}
			if temperature.Valid {
				gpu.Temperature = uint8(temperature.Int16)
			}
			if powerDraw.Valid {
				gpu.PowerDraw = uint32(powerDraw.Int64)
			}
			if isHealthy.Valid {
				gpu.IsHealthy = isHealthy.Bool
			} else {
				gpu.IsHealthy = true // Default to true if not set
			}

			provider.GPUs = append(provider.GPUs, gpu)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating GPU details rows: %w", err)
		}

		return nil
	})
}

// ListProviders returns a list of all providers, with optional filtering.
func (pps *PostgresProviderStore) ListProviders(ctx context.Context, filters map[string]interface{}) ([]*models.Provider, error) {
	var providers []*models.Provider

	// Define a function for the database operation to be retried
	operation := func() error {
		// Query both provider information and GPU details with a single query
		// Use LEFT JOIN to include providers without GPUs and JSON_AGG to aggregate GPU details
		sqlQuery := `
		SELECT p.id, p.owner_id, p.name, p.hostname, p.ip_address, p.status, p.location, 
		       p.registered_at, p.last_seen_at, p.metadata,
			COALESCE(
				JSON_AGG(
					JSON_BUILD_OBJECT(
						'model_name', g.model_name,
						'vram_mb', g.vram_mb,
						'driver_version', g.driver_version,
						'architecture', g.architecture,
						'compute_capability', g.compute_capability,
						'cuda_cores', g.cuda_cores,
						'tensor_cores', g.tensor_cores,
						'memory_bandwidth_gb_s', g.memory_bandwidth_gb_s,
						'power_consumption_w', g.power_consumption_w,
						'utilization_gpu_percent', g.utilization_gpu_percent,
						'utilization_memory_percent', g.utilization_memory_percent,
						'temperature_c', g.temperature_c,
						'power_draw_w', g.power_draw_w,
						'is_healthy', g.is_healthy
					)
				) FILTER (WHERE g.id IS NOT NULL),
				'[]'::JSON
			) AS gpus
		FROM providers p
		LEFT JOIN gpu_details g ON p.id = g.provider_id
		`

		// Base WHERE clause to be appended if filters exist
		var whereConditions []string
		var args []interface{}
		argIndex := 1

		// Apply filters
		if filters != nil {
			// Filter by status
			if statusFilter, ok := filters["status"].(string); ok && statusFilter != "" {
				whereConditions = append(whereConditions, fmt.Sprintf("p.status = $%d", argIndex))
				args = append(args, statusFilter)
				argIndex++
			}

			// Filter by GPU model
			if gpuModel, ok := filters["gpu_model"].(string); ok && gpuModel != "" {
				whereConditions = append(whereConditions, fmt.Sprintf("EXISTS (SELECT 1 FROM gpu_details gf WHERE gf.provider_id = p.id AND LOWER(gf.model_name) LIKE LOWER($%d))", argIndex))
				args = append(args, "%"+gpuModel+"%")
				argIndex++
			}

			// Filter by minimum VRAM
			if minVRAM, ok := filters["min_vram"].(uint64); ok && minVRAM > 0 {
				whereConditions = append(whereConditions, fmt.Sprintf("EXISTS (SELECT 1 FROM gpu_details gf WHERE gf.provider_id = p.id AND gf.vram_mb >= $%d)", argIndex))
				args = append(args, minVRAM)
				argIndex++
			}

			// Filter by architecture
			if arch, ok := filters["architecture"].(string); ok && arch != "" {
				whereConditions = append(whereConditions, fmt.Sprintf("EXISTS (SELECT 1 FROM gpu_details gf WHERE gf.provider_id = p.id AND LOWER(gf.architecture) LIKE LOWER($%d))", argIndex))
				args = append(args, "%"+arch+"%")
				argIndex++
			}

			// Filter for only healthy GPUs
			if healthyOnly, ok := filters["healthy_only"].(bool); ok && healthyOnly {
				whereConditions = append(whereConditions, fmt.Sprintf("NOT EXISTS (SELECT 1 FROM gpu_details gf WHERE gf.provider_id = p.id AND gf.is_healthy = false)"))
			}
		}

		// Append WHERE clause if any filters were added
		if len(whereConditions) > 0 {
			sqlQuery += " WHERE " + strings.Join(whereConditions, " AND ")
		}

		// Add GROUP BY and ORDER BY
		sqlQuery += " GROUP BY p.id ORDER BY p.name"

		// Execute the query
		rows, err := pps.db.Query(ctx, sqlQuery, args...)
		if err != nil {
			return fmt.Errorf("failed to list providers: %w", err)
		}
		defer rows.Close()

		tmpProviders := []*models.Provider{}
		for rows.Next() {
			var (
				provider     = &models.Provider{}
				metadataJSON []byte
				gpusJSON     []byte
				hostname     string
				ipAddress    string
				location     string
			)

			err := rows.Scan(
				&provider.ID,
				&provider.OwnerID,
				&provider.Name,
				&hostname,
				&provider.IPAddress,
				&provider.Status,
				&location,
				&provider.RegisteredAt,
				&provider.LastSeenAt,
				&metadataJSON,
				&gpusJSON,
			)

			if err != nil {
				return fmt.Errorf("failed to scan provider row: %w", err)
			}

			// Set the scanned values
			provider.Hostname = hostname
			provider.IPAddress = ipAddress
			provider.Location = location

			// Unmarshal metadata if it exists
			if len(metadataJSON) > 0 && !strings.EqualFold(string(metadataJSON), "null") {
				var metadata map[string]interface{}
				if err := json.Unmarshal(metadataJSON, &metadata); err != nil {
					return fmt.Errorf("failed to unmarshal metadata for provider %s: %w", provider.ID, err)
				}
				provider.Metadata = metadata
			} else {
				provider.Metadata = make(map[string]interface{})
			}

			// Unmarshal GPU details
			if len(gpusJSON) > 0 && !strings.EqualFold(string(gpusJSON), "null") && !strings.EqualFold(string(gpusJSON), "[]") {
				var gpuDetails []models.GPUDetail
				if err := json.Unmarshal(gpusJSON, &gpuDetails); err != nil {
					return fmt.Errorf("failed to unmarshal GPU details for provider %s: %w", provider.ID, err)
				}
				provider.GPUs = gpuDetails
			} else {
				provider.GPUs = []models.GPUDetail{} // Empty slice instead of nil
			}

			tmpProviders = append(tmpProviders, provider)
		}

		if err := rows.Err(); err != nil {
			return fmt.Errorf("error iterating provider rows: %w", err)
		}

		// Update output parameters only on success
		providers = tmpProviders
		return nil
	}

	// Execute the operation with retry
	err := WithRetry(ctx, pps.logger, "ListProviders", 3, 1*time.Second, operation)
	if err != nil {
		return nil, err
	}

	return providers, nil
}

// UpdateProvider updates an existing provider in the database.
func (pps *PostgresProviderStore) UpdateProvider(ctx context.Context, id uuid.UUID, updatedProvider *models.Provider) error {
	// Start a transaction
	tx, err := pps.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Rollback if not committed

	// Update the provider record
	metadataJSON, err := json.Marshal(updatedProvider.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	sqlProvider := `
	UPDATE providers SET 
		owner_id = $1,
		name = $2,
		hostname = $3,
		ip_address = $4,
		status = $5,
		location = $6,
		last_seen_at = $7,
		metadata = $8
	WHERE id = $9
	`

	result, err := tx.Exec(ctx, sqlProvider,
		updatedProvider.OwnerID,
		updatedProvider.Name,
		updatedProvider.Hostname,
		updatedProvider.IPAddress,
		updatedProvider.Status,
		updatedProvider.Location,
		updatedProvider.LastSeenAt,
		metadataJSON,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	if rowsAffected := result.RowsAffected(); rowsAffected == 0 {
		return models.ErrProviderNotFound
	}

	// Delete existing GPU details for this provider
	sqlDeleteGPUs := `DELETE FROM gpu_details WHERE provider_id = $1`
	_, err = tx.Exec(ctx, sqlDeleteGPUs, id)
	if err != nil {
		return fmt.Errorf("failed to delete existing GPU details: %w", err)
	}

	// Insert updated GPU details
	sqlGPU := `
	INSERT INTO gpu_details (
		provider_id, model_name, vram_mb, driver_version,
		architecture, compute_capability, cuda_cores, tensor_cores,
		memory_bandwidth_gb_s, power_consumption_w,
		utilization_gpu_percent, utilization_memory_percent,
		temperature_c, power_draw_w, is_healthy
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	for _, gpu := range updatedProvider.GPUs {
		_, err = tx.Exec(ctx, sqlGPU,
			id,
			gpu.ModelName,
			gpu.VRAM,
			gpu.DriverVersion,
			gpu.Architecture,
			gpu.ComputeCapability,
			gpu.CudaCores,
			gpu.TensorCores,
			gpu.MemoryBandwidth,
			gpu.PowerConsumption,
			gpu.UtilizationGPU,
			gpu.UtilizationMem,
			gpu.Temperature,
			gpu.PowerDraw,
			gpu.IsHealthy,
		)
		if err != nil {
			return fmt.Errorf("failed to insert updated GPU detail: %w", err)
		}
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// DeleteProvider removes a provider from the database.
func (pps *PostgresProviderStore) DeleteProvider(ctx context.Context, id uuid.UUID) error {
	// The GPU details will be automatically deleted due to ON DELETE CASCADE

	sqlProvider := `DELETE FROM providers WHERE id = $1`
	result, err := pps.db.Exec(ctx, sqlProvider, id)
	if err != nil {
		return fmt.Errorf("failed to delete provider: %w", err)
	}

	if rowsAffected := result.RowsAffected(); rowsAffected == 0 {
		return models.ErrProviderNotFound
	}

	return nil
}

// UpdateProviderStatus updates the status of a specific provider.
func (pps *PostgresProviderStore) UpdateProviderStatus(ctx context.Context, id uuid.UUID, status models.ProviderStatus) error {
	now := time.Now().UTC()
	sql := `
	UPDATE providers 
	SET status = $1, last_seen_at = $2
	WHERE id = $3
	`

	result, err := pps.db.Exec(ctx, sql, status, now, id)
	if err != nil {
		return fmt.Errorf("failed to update provider status: %w", err)
	}

	if rowsAffected := result.RowsAffected(); rowsAffected == 0 {
		return models.ErrProviderNotFound
	}

	return nil
}

// UpdateProviderHeartbeat updates the timestamp for the last heartbeat
// and also updates GPU utilization metrics if provided
func (pps *PostgresProviderStore) UpdateProviderHeartbeat(ctx context.Context, id uuid.UUID, gpuMetrics []models.GPUDetail) error {
	// Start a transaction
	tx, err := pps.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Update the provider's last_seen_at timestamp
	sql := `
	UPDATE providers 
	SET last_seen_at = NOW(), 
	    status = CASE WHEN status = 'offline' OR status = 'error' THEN 'idle' ELSE status END
	WHERE id = $1
	`
	result, err := tx.Exec(ctx, sql, id)
	if err != nil {
		tx.Rollback(ctx)
		return fmt.Errorf("failed to update provider heartbeat: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		tx.Rollback(ctx)
		return models.ErrProviderNotFound
	}

	// If GPU metrics are provided, update them
	if len(gpuMetrics) > 0 {
		// For each GPU, update its metrics
		for i, gpu := range gpuMetrics {
			// We need the GPU ID from the database
			var gpuID int
			err := tx.QueryRow(ctx,
				"SELECT id FROM gpu_details WHERE provider_id = $1 ORDER BY id LIMIT 1 OFFSET $2",
				id, i).Scan(&gpuID)

			if err != nil {
				tx.Rollback(ctx)
				if err == pgx.ErrNoRows {
					return fmt.Errorf("GPU at index %d not found for provider %s", i, id)
				}
				return fmt.Errorf("failed to get GPU ID: %w", err)
			}

			// Update the GPU metrics
			_, err = tx.Exec(ctx, `
				UPDATE gpu_details
				SET utilization_gpu_percent = $1,
					utilization_memory_percent = $2,
					temperature_c = $3,
					power_draw_w = $4,
					is_healthy = $5,
					updated_at = NOW()
				WHERE id = $6`,
				gpu.UtilizationGPU,
				gpu.UtilizationMem,
				gpu.Temperature,
				gpu.PowerDraw,
				gpu.IsHealthy,
				gpuID)

			if err != nil {
				tx.Rollback(ctx)
				return fmt.Errorf("failed to update GPU metrics: %w", err)
			}
		}
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close closes the database connection pool.
func (pps *PostgresProviderStore) Close() error {
	if pps.db != nil {
		pps.db.Close()
		pps.logger.Info("Closed PostgreSQL provider store connection pool")
	}
	return nil
}

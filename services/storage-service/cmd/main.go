package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dante-gpu/dante-backend/storage-service/internal/api"
	"github.com/dante-gpu/dante-backend/storage-service/internal/config"
	"github.com/dante-gpu/dante-backend/storage-service/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Version string = "dev"
) // Can be set during build

func main() {
	// Initialize Logger (basic one until config is loaded)
	interimLogger, _ := zap.NewDevelopment()
	logger := interimLogger // Will be replaced by configured logger

	// Load Configuration
	cfg, err := config.LoadConfig("./configs/config.yaml", interimLogger) // Pass interim logger
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// Setup final logger based on loaded config
	logger, err = initLogger(cfg.LogLevel) // Use LogLevel from cfg
	if err != nil {
		interimLogger.Fatal("Failed to initialize final logger", zap.Error(err))
	}
	defer logger.Sync() // Flushes buffer, if any
	cfg.Logger = logger // Store final logger in cfg if needed elsewhere

	logger.Info("Starting Storage Service",
		zap.String("version", Version),
		zap.String("instance_id", cfg.InstanceID),
		zap.String("log_level", cfg.LogLevel),
	)

	// Initialize MinIO Storage Client
	minioClient, err := storage.NewMinioClient(cfg.Minio, logger)
	if err != nil {
		logger.Fatal("Failed to initialize MinIO client", zap.Error(err))
	}
	logger.Info("MinIO client initialized")

	// Ensure default bucket exists if specified and enabled
	if cfg.Minio.DefaultBucket != "" && cfg.Minio.AutoCreateDefaultBucket {
		logger.Info("Ensuring default bucket exists", zap.String("bucket", cfg.Minio.DefaultBucket), zap.String("region", cfg.Minio.Region))
		if err := minioClient.EnsureBucket(context.Background(), cfg.Minio.DefaultBucket, cfg.Minio.Region); err != nil {
			logger.Error("Failed to ensure default MinIO bucket", zap.String("bucket", cfg.Minio.DefaultBucket), zap.Error(err))
		} else {
			logger.Info("Default bucket ensured successfully", zap.String("bucket", cfg.Minio.DefaultBucket))
		}
	}

	// Initialize Router and Handlers
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(newStructuredLogger(logger))            // Custom Zap logger middleware
	r.Use(middleware.Recoverer)                   // Recovers from panics
	r.Use(middleware.Timeout(cfg.RequestTimeout)) // Global request timeout

	// Health check endpoint
	healthPath := "/health"
	if cfg.Consul.Enabled && cfg.Consul.Registration.HealthCheckPath != "" {
		healthPath = cfg.Consul.Registration.HealthCheckPath
	}
	r.Get(healthPath, func(w http.ResponseWriter, r *http.Request) {
		// TODO: Add more detailed health checks (e.g., MinIO connectivity)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "{\"status\": \"UP\"}")
	})

	storageHandler := api.NewStorageHandler(minioClient, logger)
	storageHandler.RegisterRoutes(r)
	logger.Info("HTTP routes registered")

	// Consul Registration
	var consulServiceID string
	if cfg.Consul.Enabled {
		logger.Info("Consul registration enabled. Attempting to register service...")
		consulServiceID, err = registerServiceWithConsul(cfg, logger)
		if err != nil {
			logger.Error("Failed to register service with Consul. Proceeding without registration.", zap.Error(err))
		} else {
			logger.Info("Service registered with Consul successfully", zap.String("service_id", consulServiceID))
		}
	}

	// Start HTTP Server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         serverAddr,
		Handler:      r,
		ReadTimeout:  cfg.RequestTimeout, // Apply more specific timeouts
		WriteTimeout: cfg.RequestTimeout * 2,
		IdleTimeout:  120 * time.Second, // Standard idle timeout
	}

	go func() {
		logger.Info("Storage Service listening", zap.String("address", serverAddr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Could not listen on address", zap.String("address", serverAddr), zap.Error(err))
		}
	}()

	// Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	if cfg.Consul.Enabled && consulServiceID != "" {
		logger.Info("Deregistering service from Consul", zap.String("service_id", consulServiceID))
		if err := deregisterServiceFromConsul(cfg.Consul.Address, consulServiceID, logger); err != nil {
			logger.Error("Failed to deregister service from Consul", zap.Error(err))
		} else {
			logger.Info("Successfully deregistered from Consul", zap.String("service_id", consulServiceID))
		}
	}

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited gracefully")
}

// initLogger initializes a Zap logger based on the configured log level.
func initLogger(logLevelStr string) (*zap.Logger, error) {
	var level zapcore.Level
	if err := level.Set(logLevelStr); err != nil {
		// Default to info if parsing fails, and log a warning
		fmt.Fprintf(os.Stderr, "Warning: Invalid log level '%s', defaulting to 'info'. Error: %v\n", logLevelStr, err)
		level = zapcore.InfoLevel
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(level),
		Development: false, // Set to true for more readable console output
		Encoding:    "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	// If development mode (e.g. log level "debug"), use a console encoder for readability
	if level == zapcore.DebugLevel {
		config.Encoding = "console"
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	return config.Build()
}

// newStructuredLogger creates a chi middleware for logging requests using Zap.
func newStructuredLogger(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			wrapper := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrapper, r)
			duration := time.Since(start)

			logger.Info("Request completed",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status_code", wrapper.Status()),
				zap.Int("bytes_written", wrapper.BytesWritten()),
				zap.Duration("duration_ms", duration),
				zap.String("remote_addr", r.RemoteAddr),
				zap.String("user_agent", r.UserAgent()),
				zap.String("request_id", middleware.GetReqID(r.Context())),
			)
		}
		return http.HandlerFunc(fn)
	}
}

// registerServiceWithConsul attempts to register the service with Consul.
func registerServiceWithConsul(cfg *config.Config, logger *zap.Logger) (string, error) {
	consulClientConfig := consulapi.DefaultConfig()
	consulClientConfig.Address = cfg.Consul.Address
	client, err := consulapi.NewClient(consulClientConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create consul client: %w", err)
	}

	serviceReg := cfg.Consul.Registration
	serviceID := serviceReg.ServiceIDPrefix + cfg.InstanceID // Ensure unique ID using instance ID

	// Determine service address for registration (Consul defaults to agent's address if empty)
	regAddress := cfg.Server.Host
	if regAddress == "0.0.0.0" || regAddress == "::" { // Common ways to specify all interfaces
		regAddress = "" // Let Consul figure out the actual address
	}

	agentService := &consulapi.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceReg.ServiceName,
		Port:    cfg.Server.Port,
		Address: regAddress,
		Tags:    serviceReg.ServiceTags,
		Check: &consulapi.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d%s", getHealthCheckHost(cfg.Server.Host, logger), cfg.Server.Port, serviceReg.HealthCheckPath),
			Interval:                       serviceReg.HealthCheckInterval.String(),
			Timeout:                        serviceReg.HealthCheckTimeout.String(),
			DeregisterCriticalServiceAfter: "1m", // Example, make configurable if needed
		},
	}

	// Retry mechanism for registration can be added here if desired
	if err = client.Agent().ServiceRegister(agentService); err != nil {
		return "", fmt.Errorf("failed to register service '%s' (ID: %s) with consul: %w", serviceReg.ServiceName, serviceID, err)
	}
	return serviceID, nil
}

// getHealthCheckHost determines the host for Consul health check URL.
// If server host is "" or "0.0.0.0", use "127.0.0.1" for health check.
func getHealthCheckHost(serverHost string, logger *zap.Logger) string {
	if serverHost == "" || serverHost == "0.0.0.0" || serverHost == "::" {
		logger.Debug("Server host is unspecified for health check, defaulting to 127.0.0.1")
		return "127.0.0.1"
	}
	return serverHost
}

// deregisterServiceFromConsul deregisters the service from Consul.
func deregisterServiceFromConsul(consulAddr, serviceID string, logger *zap.Logger) error {
	consulConfig := consulapi.DefaultConfig()
	consulConfig.Address = consulAddr
	client, err := consulapi.NewClient(consulConfig)
	if err != nil {
		return fmt.Errorf("failed to create consul client for deregistration: %w", err)
	}

	if err := client.Agent().ServiceDeregister(serviceID); err != nil {
		return fmt.Errorf("failed to deregister service '%s': %w", serviceID, err)
	}
	return nil
}

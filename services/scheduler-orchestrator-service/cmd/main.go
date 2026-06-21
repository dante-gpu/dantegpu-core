package main

import (
	"context"
	"fmt"
	stlog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/billing"
	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/clients"
	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/config"
	consul_client "github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/consul"
	nats_client "github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/nats"
	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/scheduler"
	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/server"
	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/store"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// --- Configuration ---
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		stlog.Fatalf("Failed to load configuration: %v", err) // Use standard log before Zap is up
	}

	// --- Logger ---
	logger, err := setupLogger(cfg.LogLevel)
	if err != nil {
		stlog.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		_ = logger.Sync() // Flush logs before exiting
	}()

	logger.Info("Scheduler Orchestrator Service starting up...")

	// --- Database Connection (JobStore) ---
	dbCtx, dbCancel := context.WithTimeout(context.Background(), 10*time.Second) // Context for initial DB connection
	defer dbCancel()

	dbPool, err := pgxpool.New(dbCtx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to PostgreSQL database", zap.String("url", cfg.DatabaseURL), zap.Error(err))
	}
	defer dbPool.Close() // Ensure pool is closed on exit

	logger.Info("Successfully connected to PostgreSQL database", zap.String("url", cfg.DatabaseURL))

	jobStore := store.NewPostgresJobStore(dbPool, logger)
	if err := jobStore.Initialize(dbCtx); err != nil {
		logger.Fatal("Failed to initialize job store (create tables)", zap.Error(err))
	}
	logger.Info("PostgresJobStore initialized successfully")
	// Defer jobStore.Close() if it has resources other than dbPool to close.
	// Since PostgresJobStore.Close() just calls dbPool.Close(), and dbPool.Close() is already deferred,
	// an explicit defer jobStore.Close() might be redundant here IF its Close method only closes the pool.
	// However, for interface compliance and future changes, it's safer to defer it.
	// We'll ensure graceful shutdown handles it before dbPool.Close().

	// --- Consul Client & Service Registration ---
	consulClient, err := consul_client.Connect(cfg.ConsulAddress, logger)
	if err != nil {
		logger.Fatal("Failed to connect to Consul agent", zap.Error(err))
	}

	serviceID := config.GenerateServiceID(cfg.ServiceIDPrefix)
	logger.Info("Generated unique service ID for Consul", zap.String("service_id", serviceID))

	err = consul_client.RegisterService(consulClient, cfg, serviceID, logger)
	if err != nil {
		logger.Fatal("Failed to register service with Consul", zap.Error(err))
	}
	logger.Info("Successfully registered service with Consul",
		zap.String("service_name", cfg.ServiceName),
		zap.String("service_id", serviceID),
	)

	// --- NATS Client ---
	nc, err := nats_client.Connect(cfg.NatsAddress, logger)
	if err != nil {
		logger.Error("Failed to establish initial NATS connection. Service may be degraded.", zap.Error(err))
	}
	if nc != nil {
		// Deferring nc.Close() will be handled after jobConsumer.Stop() and nc.Drain() in the shutdown sequence
		logger.Info("Successfully connected to NATS", zap.String("address", cfg.NatsAddress))
	} else {
		logger.Warn("Running without NATS connection. Job processing will be unavailable.")
	}

	// --- Provider Registry Client ---
	// This client is needed by the JobConsumer to query for available providers.
	prClient := clients.NewClient(cfg, consulClient, logger)
	logger.Info("Provider Registry Service client initialized")

	// --- Billing Client ---
	billingConfig := &billing.Config{
		BaseURL: "http://localhost:8080", // Billing service URL
		Timeout: 30 * time.Second,
	}
	billingClient := billing.NewClient(billingConfig, logger)
	logger.Info("Billing Service client initialized")

	// --- Job Consumer ---
	var jobConsumer *scheduler.JobConsumer
	if nc != nil { // Only start consumer if NATS connection is available
		jc, consumerErr := scheduler.NewJobConsumer(nc, cfg, prClient, billingClient, logger, jobStore)
		if consumerErr != nil {
			logger.Error("Failed to create JobConsumer. Job processing will be unavailable.", zap.Error(consumerErr))
		} else {
			jobConsumer = jc
			if err := jobConsumer.StartConsuming(); err != nil {
				logger.Error("Failed to start JobConsumer. Job processing will be unavailable.", zap.Error(err))
				jobConsumer = nil // Ensure it's nil if starting failed
			} else {
				logger.Info("JobConsumer started and listening for jobs on NATS.")
			}
		}
	} else {
		logger.Warn("JobConsumer not started due to NATS connection failure.")
	}

	// --- Setup Router and HTTP Server ---
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewStructuredLogger(logger)) // Zap logging middleware
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.RequestTimeout))

	// Health Check endpoint (required by Consul registration)
	r.Get(cfg.HealthCheckPath, func(w http.ResponseWriter, r *http.Request) {
		healthStatus := http.StatusOK
		healthMsg := "Scheduler Orchestrator Service is healthy."

		// Check DB connection status (ping)
		dbPingCtx, dbPingCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer dbPingCancel()
		if err := dbPool.Ping(dbPingCtx); err != nil {
			healthStatus = http.StatusServiceUnavailable
			healthMsg = "Database connection is down."
			logger.Warn("Health check: Database ping failed", zap.Error(err))
		} else {
			healthMsg += " DB: OK."
		}

		// Check NATS connection status
		if nc == nil || nc.Status() != nats.CONNECTED {
			healthStatus = http.StatusServiceUnavailable
			healthMsg = "NATS connection is down."
			logger.Warn("Health check: NATS is not connected")
		} else {
			healthMsg += " NATS: OK."
		}

		// Check if JobConsumer is running (basic check)
		if jobConsumer == nil {
			healthStatus = http.StatusServiceUnavailable
			healthMsg += " JobConsumer: Not Running."
			logger.Warn("Health check: JobConsumer is not running")
		} else {
			healthMsg += " JobConsumer: Running."
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(healthStatus)
		fmt.Fprintln(w, healthMsg)
		logger.Debug("Health check endpoint hit", zap.Int("status", healthStatus), zap.String("message", healthMsg))
	})

	srv := server.NewServer(cfg, r, logger)

	// --- Start Server Goroutine ---
	go srv.Start()

	// --- Graceful Shutdown & Consul Deregistration ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received

	logger.Info("Shutdown signal received, starting graceful shutdown...")

	// Stop JobConsumer first
	if jobConsumer != nil {
		logger.Info("Stopping JobConsumer...")
		jobConsumer.Stop()
		logger.Info("JobConsumer stopped.")
	}

	// Deregister from Consul
	if err := consul_client.DeregisterService(consulClient, serviceID, logger); err != nil {
		logger.Error("Error deregistering service from Consul", zap.Error(err))
	} else {
		logger.Info("Successfully deregistered service from Consul")
	}

	// Shutdown HTTP server
	ctxHttp, cancelHttp := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelHttp()
	srv.Stop(ctxHttp)

	// Close NATS connection gracefully if it was established
	if nc != nil {
		logger.Info("Draining NATS connection...")
		if err := nc.Drain(); err != nil {
			logger.Error("Error draining NATS connection", zap.Error(err))
		}
		nc.Close() // Close the connection after draining
		logger.Info("NATS connection drained and closed")
	}

	// jobStore.Close() is implicitly handled by dbPool.Close() which is already deferred.
	// If jobStore.Close() did more than just closing the pool, it would be called here explicitly.
	logger.Info("Scheduler Orchestrator Service gracefully stopped")
}

// setupLogger configures Zap based on the log level string.
func setupLogger(levelString string) (*zap.Logger, error) {
	var logLevel zapcore.Level
	if err := logLevel.Set(levelString); err != nil {
		logLevel = zapcore.InfoLevel // Default to info if parsing fails
	}

	zapCfg := zap.Config{
		Level:       zap.NewAtomicLevelAt(logLevel),
		Development: false, // Set to true for more dev-friendly output (e.g. console encoder, stack traces on warn)
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

	logger, err := zapCfg.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return logger, nil
}

// NewStructuredLogger returns a middleware that logs request details using Zap.
func NewStructuredLogger(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				duration := time.Since(start)
				logger.Info("Request completed",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("remote_ip", r.RemoteAddr),
					zap.String("request_id", middleware.GetReqID(r.Context())),
					zap.Int("status", ww.Status()),
					zap.Int("bytes", ww.BytesWritten()),
					zap.Duration("duration", duration),
				)
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

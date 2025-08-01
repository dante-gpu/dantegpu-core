package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/dante-gpu/dante-backend/api-gateway/internal/billing"
	"github.com/dante-gpu/dante-backend/api-gateway/internal/config"
	consul_client "github.com/dante-gpu/dante-backend/api-gateway/internal/consul"
	"github.com/dante-gpu/dante-backend/api-gateway/internal/handlers"
	"github.com/dante-gpu/dante-backend/api-gateway/internal/loadbalancer"
	customMiddleware "github.com/dante-gpu/dante-backend/api-gateway/internal/middleware" // Alias to avoid conflict
	nats_client "github.com/dante-gpu/dante-backend/api-gateway/internal/nats"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware" // Import consul api
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// I should load the configuration first.
	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		// Using a temporary basic logger for config loading errors
		basicLogger, _ := zap.NewProduction()
		basicLogger.Fatal("Failed to load configuration", zap.Error(err))
	}

	// I need to set up the Zap logger based on config.
	logger, err := setupLogger(cfg.LogLevel)
	if err != nil {
		// Use standard log if Zap setup fails initially
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync() // Flushing buffer, important!

	// == Establish NATS Connection ==
	nc, err := nats_client.Connect(cfg.NatsAddress, logger)
	if err != nil {
		logger.Fatal("Failed to establish initial NATS connection", zap.Error(err))
	}
	defer nc.Close()

	// == Establish Consul Connection ==
	consulClient, err := consul_client.Connect(cfg.ConsulAddress, logger)
	if err != nil {
		// Log the error, but the gateway might still be partially useful (e.g., auth, direct NATS jobs)
		// Let's make it non-fatal for now, but health check should reflect this.
		logger.Error("Failed to establish initial Consul connection", zap.Error(err))
		// Consider adding a flag or status to indicate Consul failure.
	}

	// I need to set up the router.
	r := chi.NewRouter()

	// I should add basic middleware.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(NewStructuredLogger(logger))
	r.Use(middleware.Recoverer)
	r.Use(customMiddleware.CORS()) // Add CORS middleware
	r.Use(middleware.Timeout(cfg.RequestTimeout))

	// Create billing client
	billingConfig := &billing.Config{
		BaseURL: "http://localhost:8080", // Billing service URL
		Timeout: 30 * time.Second,
	}
	billingClient := billing.NewClient(billingConfig, logger)

	// I need to create instances of my handlers.
	authHandler := handlers.NewAuthHandler(logger, cfg)
	jobHandler := handlers.NewJobHandler(logger, cfg, nc)
	billingHandler := handlers.NewBillingHandler(billingClient, logger)
	// Create load balancer and proxy handler (even if Consul connection failed, to avoid nil pointers)
	lb := loadbalancer.NewRoundRobin()
	proxyHandler := handlers.NewProxyHandler(logger, cfg, consulClient, lb)
	logHandler := handlers.NewLogHandler(logger, cfg.LokiURL)

	// == Public Routes ==
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		healthStatus := http.StatusOK
		healthMsg := "API Gateway is healthy"
		consulOk := false

		// Check NATS
		if nc.Status() != nats.CONNECTED {
			healthStatus = http.StatusServiceUnavailable
			healthMsg = "NATS connection is down."
			logger.Warn("Health check: NATS is not connected", zap.String("nats_status", nc.Status().String()))
		} else {
			healthMsg += " NATS: OK."
		}

		// Check Consul (only if client was created)
		if consulClient != nil {
			_, err := consulClient.Agent().Self() // Ping Consul
			if err != nil {
				healthStatus = http.StatusServiceUnavailable
				healthMsg += " Consul connection failed."
				logger.Warn("Health check: Consul ping failed", zap.Error(err))
			} else {
				healthMsg += " Consul: OK."
				consulOk = true
			}
		} else {
			healthStatus = http.StatusServiceUnavailable
			healthMsg += " Consul client not initialized."
			logger.Warn("Health check: Consul client is nil")
		}

		logger.Info("Health check endpoint hit",
			zap.String("path", r.URL.Path),
			zap.String("nats_status", nc.Status().String()),
			zap.Bool("consul_ok", consulOk),
			zap.Int("overall_status", healthStatus),
		)
		w.WriteHeader(healthStatus)
		fmt.Fprintf(w, strings.TrimSpace(healthMsg))
	})

	// == Log Streaming Route ==
	r.Get("/logs/stream", logHandler.StreamLogs)

	// == Authentication Routes ==
	r.Route("/auth", func(r chi.Router) {
		r.Post("/login", authHandler.Login)
		r.Post("/register", authHandler.Register)

		// Routes requiring authentication
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.Authenticator(logger, cfg.JwtSecret))
			r.Get("/profile", authHandler.Profile)
		})
	})

	// == API V1 Routes ==
	r.Route("/api/v1", func(r chi.Router) {
		// Authentication routes (no auth required)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authHandler.Login)
			r.Post("/register", authHandler.Register)

			// Protected auth routes
			r.Group(func(r chi.Router) {
				r.Use(customMiddleware.Authenticator(logger, cfg.JwtSecret))
				r.Get("/me", authHandler.Profile)
			})
		})

		// Protected routes
		r.Group(func(r chi.Router) {
			r.Use(customMiddleware.Authenticator(logger, cfg.JwtSecret))

			// Job submission routes
			r.Post("/jobs", jobHandler.SubmitJob)
			r.Get("/jobs/{jobID}", jobHandler.GetJobStatus)
			r.Delete("/jobs/{jobID}", jobHandler.CancelJob)

			// Billing and wallet endpoints
			r.Route("/billing", func(r chi.Router) {
				// Wallet management
				r.Post("/wallet", billingHandler.CreateWallet)
				r.Get("/wallet/{walletID}/balance", billingHandler.GetWalletBalance)
				r.Post("/wallet/{walletID}/deposit", billingHandler.DepositTokens)
				r.Post("/wallet/{walletID}/withdraw", billingHandler.WithdrawTokens)
				r.Get("/wallet/{walletID}/transactions", billingHandler.GetTransactionHistory)

				// User-specific endpoints
				r.Get("/user/{userID}/wallet", billingHandler.GetUserWallet)
				r.Get("/user/{userID}/balance", billingHandler.GetUserBalance)

				// Pricing and marketplace
				r.Get("/pricing/rates", billingHandler.GetPricingRates)
				r.Post("/pricing/calculate", billingHandler.CalculatePricing)
				r.Post("/pricing/estimate", billingHandler.EstimateJobCost)
				r.Get("/marketplace", billingHandler.GetGPUMarketplace)
			})
		})

		// Admin routes (placeholder)
		// r.Group(func(r chi.Router) {
		//    r.Use(customMiddleware.RequireRole("admin")) // Role checking middleware needed
		//    r.Get("/admin-stats", handlers.GetAdminStats)
		// })
	})

	// == Service Proxy Route ==
	// HandleFunc matches all methods. Use Method specific handlers (e.g., r.Get) if needed.
	// The "*" in the pattern is crucial for matching subpaths.
	r.HandleFunc("/services/{serviceName}/*", proxyHandler.ServeHTTP)

	// == Admin Dashboard Route (Example - protected) ==
	// r.Group(func(r chi.Router) {
	//     r.Use(customMiddleware.Authenticator(logger, cfg.JwtSecret))
	//     r.Use(customMiddleware.RequireRole("admin")) // Role checking middleware needed
	//     r.Get("/admin", handlers.AdminDashboard)
	// })

	// I need to start the HTTP server.
	logger.Info("Starting API Gateway", zap.String("port", cfg.Port))
	if err := http.ListenAndServe(cfg.Port, r); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

// setupLogger configures Zap based on the log level string.
func setupLogger(levelString string) (*zap.Logger, error) {
	var logLevel zapcore.Level
	if err := logLevel.Set(levelString); err != nil {
		logLevel = zapcore.InfoLevel // Default to info if parsing fails
	}

	config := zap.Config{
		Level:       zap.NewAtomicLevelAt(logLevel),
		Development: false,
		Encoding:    "json", // Or "console" for more readable output during dev
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
		OutputPaths:      []string{"stdout"}, // Log to standard output
		ErrorOutputPaths: []string{"stderr"}, // Log errors to standard error
	}

	logger, err := config.Build()
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
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor) // To capture status code

			defer func() {
				duration := time.Since(start)
				// Log details for the request
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

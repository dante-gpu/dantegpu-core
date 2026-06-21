package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v3"

	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/config"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/handlers"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/pricing"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/service"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/solana"
	"github.com/dante-gpu/dante-backend/billing-payment-service/internal/store"
)

func main() {
	// Load configuration
	cfg, err := loadConfig("configs/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Setup logger
	logger, err := setupLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Billing & Payment Service starting up...")

	// Setup database connection
	dbPool, err := setupDatabase(cfg.Database.URL, logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer dbPool.Close()

	// Initialize database schema
	store := store.NewPostgresStore(dbPool, logger)
	if err := store.Initialize(context.Background()); err != nil {
		logger.Fatal("Failed to initialize database schema", zap.Error(err))
	}

	// Setup Solana client
	solanaClient, err := setupSolanaClient(&cfg.Solana, logger)
	if err != nil {
		logger.Fatal("Failed to initialize Solana client", zap.Error(err))
	}
	defer solanaClient.Close()

	// Setup pricing engine
	pricingEngine := pricing.NewEngine(&cfg.Pricing, logger)

	// Setup billing service
	billingService := service.NewBillingService(
		store,
		solanaClient,
		pricingEngine,
		&cfg.Billing,
		logger,
	)

	// Setup HTTP server
	server := setupHTTPServer(cfg, billingService, logger)

	// Setup graceful shutdown
	setupGracefulShutdown(server, logger)

	// Start server
	logger.Info("Starting HTTP server", zap.String("address", fmt.Sprintf(":%d", cfg.Server.Port)))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Failed to start HTTP server", zap.Error(err))
	}
}

// loadConfig loads configuration from file
func loadConfig(path string) (*config.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg config.Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

// setupLogger initializes the logger
func setupLogger(level string) (*zap.Logger, error) {
	var zapLevel zap.AtomicLevel
	switch level {
	case "debug":
		zapLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapLevel = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapLevel = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	config := zap.NewProductionConfig()
	config.Level = zapLevel
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return config.Build()
}

// setupDatabase initializes the database connection
func setupDatabase(databaseURL string, logger *zap.Logger) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	// Configure connection pool
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = time.Minute * 15

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established successfully")
	return pool, nil
}

// setupSolanaClient initializes the Solana client
func setupSolanaClient(cfg *solana.Config, logger *zap.Logger) (*solana.Client, error) {
	client, err := solana.NewClient(cfg, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana client: %w", err)
	}

	logger.Info("Solana client initialized successfully")
	return client, nil
}

// setupHTTPServer configures and returns the HTTP server
func setupHTTPServer(cfg *config.Config, billingService *service.BillingService, logger *zap.Logger) *http.Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.Server.ReadTimeout))

	// CORS middleware for development
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-CSRF-Token")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","service":"billing-payment-service"}`))
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Wallet management
		r.Route("/wallet", func(r chi.Router) {
			r.Post("/", handlers.CreateWallet(billingService, logger))
			r.Get("/{walletID}/balance", handlers.GetWalletBalance(billingService, logger))
			r.Post("/{walletID}/deposit", handlers.DepositTokens(billingService, logger))
			r.Post("/{walletID}/withdraw", handlers.WithdrawTokens(billingService, logger))
			r.Get("/{walletID}/transactions", handlers.GetTransactionHistory(billingService, logger))
		})

		// Billing and sessions
		r.Route("/billing", func(r chi.Router) {
			r.Post("/start-session", handlers.StartRentalSession(billingService, logger))
			r.Post("/end-session", handlers.EndRentalSession(billingService, logger))
			r.Post("/usage-update", handlers.ProcessUsageUpdate(billingService, logger))
			r.Get("/current-usage/{sessionID}", handlers.GetCurrentUsage(billingService, logger))
			r.Get("/history", handlers.GetBillingHistory(billingService, logger))
		})

		// Pricing
		r.Route("/pricing", func(r chi.Router) {
			r.Post("/calculate", handlers.CalculatePricing(billingService, logger))
			r.Get("/rates", handlers.GetPricingRates(billingService, logger))
		})

		// Provider operations
		r.Route("/provider", func(r chi.Router) {
			r.Get("/{providerID}/earnings", handlers.GetProviderEarnings(billingService, logger))
			r.Post("/{providerID}/payout", handlers.RequestPayout(billingService, logger))
			r.Get("/{providerID}/rates", handlers.GetProviderRates(billingService, logger))
			r.Put("/{providerID}/rates", handlers.SetProviderRates(billingService, logger))
		})
	})

	return &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
}

// setupGracefulShutdown configures graceful shutdown handling
func setupGracefulShutdown(server *http.Server, logger *zap.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logger.Info("Received shutdown signal, shutting down gracefully...")

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := server.Shutdown(ctx); err != nil {
			logger.Error("Failed to shutdown server gracefully", zap.Error(err))
		} else {
			logger.Info("Server shutdown completed")
		}
	}()
}

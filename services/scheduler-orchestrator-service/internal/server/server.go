package server

import (
	"context"
	"net/http"
	"time"

	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/config"
	"go.uber.org/zap"
)

// Server wraps an http.Server with a logger for convenience.
// This allows us to attach methods like Start and Stop to our custom type.
type Server struct {
	*http.Server
	Logger *zap.Logger
}

// NewServer creates and configures a new Server instance for the scheduler-orchestrator.
// This server will primarily be used for health checks and potentially a minimal admin/status API.
func NewServer(cfg *config.Config, handler http.Handler, logger *zap.Logger) *Server {
	logger.Info("Configuring HTTP server for scheduler-orchestrator",
		zap.String("port", cfg.Port),
		zap.Duration("request_timeout", cfg.RequestTimeout),
	)

	// TODO: Add more specific timeouts (ReadHeaderTimeout, etc.) if needed from config
	httpSrv := &http.Server{
		Addr:         cfg.Port,
		Handler:      handler,
		ReadTimeout:  cfg.RequestTimeout,     // Apply general request timeout
		WriteTimeout: cfg.RequestTimeout * 2, // Usually a bit longer than read
		IdleTimeout:  120 * time.Second,      // Standard idle timeout
		// Add TLS config later if needed, loaded from cfg
	}
	return &Server{Server: httpSrv, Logger: logger}
}

// Start initiates the HTTP server listening process.
func (s *Server) Start() {
	s.Logger.Info("Starting HTTP server", zap.String("address", s.Addr))
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		s.Logger.Fatal("HTTP server ListenAndServe error", zap.Error(err))
	}
}

// Stop gracefully shuts down the HTTP server.
func (s *Server) Stop(ctx context.Context) {
	s.Logger.Info("Attempting graceful shutdown of HTTP server...")
	if err := s.Shutdown(ctx); err != nil {
		s.Logger.Error("HTTP server graceful shutdown failed", zap.Error(err))
		// Fallback to Close if Shutdown fails or context times out
		if err := s.Close(); err != nil {
			s.Logger.Error("HTTP server close failed after shutdown attempt", zap.Error(err))
		}
	} else {
		s.Logger.Info("HTTP server gracefully stopped")
	}
}

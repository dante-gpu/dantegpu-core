package server

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// NewServer creates and configures an http.Server.
// It takes the port (e.g., ":8002"), the main router, and a logger.
func NewServer(port string, handler http.Handler, logger *zap.Logger) *http.Server {
	// TODO: Add configuration for Read/Write timeouts from config
	srv := &http.Server{
		Addr:         port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		// Add TLS config later if needed
	}
	logger.Info("HTTP server configured", zap.String("address", port))
	return srv
}

package handlers

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/dante-gpu/dante-backend/api-gateway/internal/config"
	consul_client "github.com/dante-gpu/dante-backend/api-gateway/internal/consul"
	"github.com/dante-gpu/dante-backend/api-gateway/internal/loadbalancer"
	"github.com/go-chi/chi/v5"
	consulapi "github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

// ProxyHandler holds dependencies for the reverse proxy.
// I need the logger, config, Consul client, and a load balancer.
type ProxyHandler struct {
	Logger       *zap.Logger
	Config       *config.Config
	ConsulClient *consulapi.Client
	Balancer     loadbalancer.LoadBalancer
}

// NewProxyHandler creates a new ProxyHandler.
func NewProxyHandler(logger *zap.Logger, cfg *config.Config, consul *consulapi.Client, lb loadbalancer.LoadBalancer) *ProxyHandler {
	return &ProxyHandler{
		Logger:       logger,
		Config:       cfg,
		ConsulClient: consul,
		Balancer:     lb,
	}
}

// ServeHTTP handles incoming requests and proxies them to the appropriate backend service.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// I need to extract the target service name from the URL path.
	// Example: /services/my-cool-service/some/path -> my-cool-service
	serviceName := chi.URLParam(r, "serviceName")
	if serviceName == "" {
		h.Logger.Error("Missing service name in proxy request path", zap.String("path", r.URL.Path))
		http.Error(w, "Service name missing in path", http.StatusBadRequest)
		return
	}

	// I need to discover healthy instances of the service using Consul.
	serviceEntries, err := consul_client.DiscoverService(h.ConsulClient, serviceName, h.Logger)
	if err != nil {
		// Log the error (already done in DiscoverService)
		http.Error(w, fmt.Sprintf("Service '%s' not found or unhealthy: %v", serviceName, err), http.StatusBadGateway) // 502
		return
	}

	// I should select a backend instance using the load balancer.
	targetURL, err := h.Balancer.Next(serviceEntries)
	if err != nil {
		h.Logger.Error("Load balancer failed to select a service instance", zap.String("service", serviceName), zap.Error(err))
		http.Error(w, fmt.Sprintf("Failed to select instance for service '%s'", serviceName), http.StatusBadGateway)
		return
	}

	h.Logger.Debug("Proxying request to service instance",
		zap.String("service", serviceName),
		zap.String("target_url", targetURL.String()),
		zap.String("original_path", r.URL.Path),
	)

	// I need to create the reverse proxy.
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// I should modify the request path before proxying.
	// Remove the "/services/{serviceName}" prefix.
	// Example: /services/my-cool-service/some/path -> /some/path
	originalPath := r.URL.Path
	r.URL.Path = strings.TrimPrefix(originalPath, "/services/"+serviceName)
	if !strings.HasPrefix(r.URL.Path, "/") {
		r.URL.Path = "/" + r.URL.Path // Ensure leading slash
	}
	// Also clear RawPath to prevent conflicts
	r.URL.RawPath = ""

	// I should set the Host header to the target's host.
	r.Host = targetURL.Host

	// I should clear X-Forwarded-For potentially set by upstream proxies if desired,
	// or let the proxy handle appending.
	// r.Header.Del("X-Forwarded-For")

	h.Logger.Debug("Modified request path for proxy",
		zap.String("original_path", originalPath),
		zap.String("new_path", r.URL.Path),
		zap.String("target_host", r.Host),
	)

	// Serve the request using the proxy.
	proxy.ServeHTTP(w, r)
}

// Package upstream provides small helpers for the gateway to forward requests to
// the real backend services instead of serving mock data.
package upstream

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

var client = &http.Client{Timeout: 30 * time.Second}

// Forward proxies the incoming request to targetURL, preserving method, body, and
// Authorization header, then copies the upstream status and body back to w.
// Used for transparent proxying (for example, gateway auth -> auth-service).
func Forward(w http.ResponseWriter, r *http.Request, targetURL string, logger *zap.Logger) {
	req, err := http.NewRequestWithContext(r.Context(), r.Method, targetURL, r.Body)
	if err != nil {
		logger.Error("upstream: failed to build request", zap.String("target", targetURL), zap.Error(err))
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		req.Header.Set("Content-Type", ct)
	} else {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		req.Header.Set("Authorization", auth)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("upstream: request failed", zap.String("target", targetURL), zap.Error(err))
		http.Error(w, "Upstream service unavailable", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.Warn("upstream: failed to copy response body", zap.Error(err))
	}
}

// GetJSON performs a GET against url, forwarding the Authorization header, and
// streams the upstream status and body back to w. Returns false if the upstream
// could not be reached (a 502 has already been written).
func GetJSON(w http.ResponseWriter, ctx context.Context, url, authHeader string, logger *zap.Logger) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		logger.Error("upstream: failed to build GET request", zap.String("url", url), zap.Error(err))
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return false
	}
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("upstream: GET failed", zap.String("url", url), zap.Error(err))
		http.Error(w, "Upstream service unavailable", http.StatusBadGateway)
		return false
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.Warn("upstream: failed to copy GET body", zap.Error(err))
	}
	return true
}

// GetWithHeaders performs a GET with custom headers and streams the upstream
// status and body back to w.
func GetWithHeaders(w http.ResponseWriter, ctx context.Context, url string, headers map[string]string, logger *zap.Logger) bool {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		logger.Error("upstream: failed to build GET request", zap.String("url", url), zap.Error(err))
		http.Error(w, "Bad gateway", http.StatusBadGateway)
		return false
	}
	for k, v := range headers {
		if v != "" {
			req.Header.Set(k, v)
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("upstream: GET failed", zap.String("url", url), zap.Error(err))
		http.Error(w, "Upstream service unavailable", http.StatusBadGateway)
		return false
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		logger.Warn("upstream: failed to copy GET body", zap.Error(err))
	}
	return true
}

// BuildURL joins a base URL and a path safely.
func BuildURL(base, path string) string {
	return fmt.Sprintf("%s%s", base, path)
}

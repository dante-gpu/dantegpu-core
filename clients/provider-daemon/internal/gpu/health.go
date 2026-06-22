package gpu

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// HealthMonitor periodically verifies GPU detection and tracks the provider's
// online/offline state. After failureThreshold consecutive failed checks the
// provider is marked offline; it recovers automatically once a check succeeds.
// The daemon rejects new tasks while offline.
type HealthMonitor struct {
	detector         *Detector
	logger           *zap.Logger
	interval         time.Duration
	failureThreshold int

	mu               sync.RWMutex
	healthy          bool
	consecutiveFails int
	lastCheck        time.Time
	lastGPUCount     int
}

// NewHealthMonitor creates a health monitor. interval defaults to 30s and
// failureThreshold to 3 when non-positive.
func NewHealthMonitor(detector *Detector, logger *zap.Logger, interval time.Duration, failureThreshold int) *HealthMonitor {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	if failureThreshold <= 0 {
		failureThreshold = 3
	}
	return &HealthMonitor{
		detector:         detector,
		logger:           logger,
		interval:         interval,
		failureThreshold: failureThreshold,
		healthy:          true,
	}
}

// Start runs the health loop until the context is cancelled. It blocks, so run
// it in a goroutine.
func (h *HealthMonitor) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()
	h.check(ctx) // run an immediate check at startup
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			h.check(ctx)
		}
	}
}

func (h *HealthMonitor) check(ctx context.Context) {
	cctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	gpus, err := h.detector.DetectGPUs(cctx)

	h.mu.Lock()
	defer h.mu.Unlock()
	h.lastCheck = time.Now().UTC()

	if err != nil || len(gpus) == 0 {
		h.consecutiveFails++
		if h.healthy && h.consecutiveFails >= h.failureThreshold {
			h.healthy = false
			h.logger.Warn("Provider marked OFFLINE: GPU health checks failing",
				zap.Int("consecutive_failures", h.consecutiveFails), zap.Error(err))
		}
		return
	}

	h.lastGPUCount = len(gpus)
	if !h.healthy {
		h.logger.Info("Provider recovered ONLINE: GPU health restored", zap.Int("gpus", len(gpus)))
	}
	h.healthy = true
	h.consecutiveFails = 0
}

// Healthy reports whether the provider's GPUs are currently healthy.
func (h *HealthMonitor) Healthy() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthy
}

// Status returns the current health, the last detected GPU count, and the last
// check time.
func (h *HealthMonitor) Status() (healthy bool, gpuCount int, lastCheck time.Time) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.healthy, h.lastGPUCount, h.lastCheck
}

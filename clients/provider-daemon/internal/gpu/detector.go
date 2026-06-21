package gpu

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/dante-gpu/dante-backend/provider-daemon/internal/config"
	"go.uber.org/zap"
)

// GPUInfo represents detailed information about a GPU
type GPUInfo struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	Vendor            string `json:"vendor"`
	Model             string `json:"model"`
	VRAMTotal         uint64 `json:"vram_total_mb"`
	VRAMFree          uint64 `json:"vram_free_mb"`
	VRAMUsed          uint64 `json:"vram_used_mb"`
	PowerDraw         uint32 `json:"power_draw_w"`
	PowerLimit        uint32 `json:"power_limit_w"`
	Temperature       uint8  `json:"temperature_c"`
	Utilization       uint8  `json:"utilization_percent"`
	DriverVersion     string `json:"driver_version"`
	CUDAVersion       string `json:"cuda_version,omitempty"`
	ComputeCapability string `json:"compute_capability,omitempty"`
	PCIBusID          string `json:"pci_bus_id"`
	IsAvailable       bool   `json:"is_available"`
}

// GPUMetrics represents real-time GPU metrics
type GPUMetrics struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	Utilization uint8     `json:"utilization_percent"`
	VRAMUsed    uint64    `json:"vram_used_mb"`
	VRAMTotal   uint64    `json:"vram_total_mb"`
	PowerDraw   uint32    `json:"power_draw_w"`
	Temperature uint8     `json:"temperature_c"`
	FanSpeed    uint8     `json:"fan_speed_percent"`
	ClockCore   uint32    `json:"clock_core_mhz"`
	ClockMemory uint32    `json:"clock_memory_mhz"`
}

// Detector handles GPU detection and monitoring
type Detector struct {
	logger *zap.Logger
	cfg    *config.GPUDetectorSettings
	gpus   []GPUInfo
}

// NewDetector creates a new GPU detector
func NewDetector(cfg *config.GPUDetectorSettings, logger *zap.Logger) *Detector {
	return &Detector{
		logger: logger,
		cfg:    cfg,
		gpus:   make([]GPUInfo, 0),
	}
}

// DetectGPUs detects all available GPUs on the system
func (d *Detector) DetectGPUs(ctx context.Context) ([]GPUInfo, error) {
	d.logger.Info("Starting GPU detection")

	var allGPUs []GPUInfo

	// Detect NVIDIA GPUs
	nvidiaGPUs, err := d.detectNVIDIAGPUs(ctx)
	if err != nil {
		d.logger.Warn("Failed to detect NVIDIA GPUs", zap.Error(err))
	} else {
		allGPUs = append(allGPUs, nvidiaGPUs...)
	}

	// Detect AMD GPUs
	amdGPUs, err := d.detectAMDGPUs(ctx)
	if err != nil {
		d.logger.Warn("Failed to detect AMD GPUs", zap.Error(err))
	} else {
		allGPUs = append(allGPUs, amdGPUs...)
	}

	// Detect Apple Silicon GPUs
	if runtime.GOOS == "darwin" {
		appleGPUs, err := d.detectAppleGPUs(ctx)
		if err != nil {
			d.logger.Warn("Failed to detect Apple GPUs", zap.Error(err))
		} else {
			allGPUs = append(allGPUs, appleGPUs...)
		}
	}

	// Detect Intel GPUs
	intelGPUs, err := d.detectIntelGPUs(ctx)
	if err != nil {
		d.logger.Warn("Failed to detect Intel GPUs", zap.Error(err))
	} else {
		allGPUs = append(allGPUs, intelGPUs...)
	}

	d.gpus = allGPUs
	d.logger.Info("GPU detection completed", zap.Int("gpu_count", len(allGPUs)))

	return allGPUs, nil
}

// detectNVIDIAGPUs detects NVIDIA GPUs using nvidia-smi
func (d *Detector) detectNVIDIAGPUs(ctx context.Context) ([]GPUInfo, error) {
	// Check if nvidia-smi is available
	if d.cfg == nil || d.cfg.NvidiaSmiPath == "" {
		// Fallback or error if not configured, though config should provide a default.
		// For now, let's assume default path if specific one is empty.
		// A better approach might be to ensure NvidiaSmiPath is always non-empty from config loader.
		if !d.isCommandAvailable("nvidia-smi") {
			return nil, fmt.Errorf("nvidia-smi command not found (path not configured or not in PATH)")
		}
	} else if !d.isCommandAvailable(d.cfg.NvidiaSmiPath) {
		return nil, fmt.Errorf("nvidia-smi command not found at configured path: %s", d.cfg.NvidiaSmiPath)
	}

	nvidiaSmiCmd := "nvidia-smi" // Default
	if d.cfg != nil && d.cfg.NvidiaSmiPath != "" {
		nvidiaSmiCmd = d.cfg.NvidiaSmiPath
	}

	cmd := exec.CommandContext(ctx, nvidiaSmiCmd, "--query-gpu=index,name,memory.total,memory.free,memory.used,power.draw,power.limit,temperature.gpu,utilization.gpu,driver_version,pci.bus_id", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run %s: %w", nvidiaSmiCmd, err)
	}

	var gpus []GPUInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) < 11 {
			// Try to log this malformed line for debugging
			d.logger.Warn("Malformed line from nvidia-smi output", zap.String("line", line))
			continue
		}

		gpu := GPUInfo{
			Vendor:      "NVIDIA",
			IsAvailable: true,
		}

		// Parse fields
		if id := strings.TrimSpace(fields[0]); id != "" {
			gpu.ID = fmt.Sprintf("nvidia-%s", id)
		}

		gpu.Name = strings.TrimSpace(fields[1])
		gpu.Model = gpu.Name

		if vramTotal, err := strconv.ParseUint(strings.TrimSpace(fields[2]), 10, 64); err == nil {
			gpu.VRAMTotal = vramTotal
		} else if fields[2] != " [N/A]" { // Handle [N/A] gracefully
			d.logger.Warn("Failed to parse VRAM Total from nvidia-smi", zap.String("value", fields[2]), zap.Error(err))
		}

		if vramFree, err := strconv.ParseUint(strings.TrimSpace(fields[3]), 10, 64); err == nil {
			gpu.VRAMFree = vramFree
		} else if fields[3] != " [N/A]" {
			d.logger.Warn("Failed to parse VRAM Free from nvidia-smi", zap.String("value", fields[3]), zap.Error(err))
		}

		if vramUsed, err := strconv.ParseUint(strings.TrimSpace(fields[4]), 10, 64); err == nil {
			gpu.VRAMUsed = vramUsed
		} else if fields[4] != " [N/A]" {
			d.logger.Warn("Failed to parse VRAM Used from nvidia-smi", zap.String("value", fields[4]), zap.Error(err))
		}

		if powerDrawStr := strings.TrimSpace(fields[5]); powerDrawStr != "[N/A]" {
			if powerDraw, err := strconv.ParseFloat(powerDrawStr, 32); err == nil {
				gpu.PowerDraw = uint32(powerDraw)
			} else {
				d.logger.Warn("Failed to parse Power Draw from nvidia-smi", zap.String("value", powerDrawStr), zap.Error(err))
			}
		}

		if powerLimitStr := strings.TrimSpace(fields[6]); powerLimitStr != "[N/A]" {
			if powerLimit, err := strconv.ParseFloat(powerLimitStr, 32); err == nil {
				gpu.PowerLimit = uint32(powerLimit)
			} else {
				d.logger.Warn("Failed to parse Power Limit from nvidia-smi", zap.String("value", powerLimitStr), zap.Error(err))
			}
		}

		if tempStr := strings.TrimSpace(fields[7]); tempStr != "[N/A]" {
			if temp, err := strconv.ParseUint(tempStr, 10, 8); err == nil {
				gpu.Temperature = uint8(temp)
			} else {
				d.logger.Warn("Failed to parse Temperature from nvidia-smi", zap.String("value", tempStr), zap.Error(err))
			}
		}

		if utilStr := strings.TrimSpace(fields[8]); utilStr != "[N/A]" {
			if util, err := strconv.ParseUint(utilStr, 10, 8); err == nil {
				gpu.Utilization = uint8(util)
			} else {
				d.logger.Warn("Failed to parse GPU Utilization from nvidia-smi", zap.String("value", utilStr), zap.Error(err))
			}
		}

		gpu.DriverVersion = strings.TrimSpace(fields[9])
		gpu.PCIBusID = strings.TrimSpace(fields[10])

		// Get CUDA version
		if cudaVersion, err := d.getCUDAVersion(ctx, nvidiaSmiCmd); err == nil {
			gpu.CUDAVersion = cudaVersion
		} else {
			d.logger.Warn("Failed to get CUDA version", zap.Error(err))
		}

		// Get compute capability
		if computeCap, err := d.getComputeCapability(ctx, gpu.ID); err == nil {
			gpu.ComputeCapability = computeCap
		} else {
			d.logger.Warn("Failed to get compute capability", zap.String("gpuID", gpu.ID), zap.Error(err))
		}

		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// detectAMDGPUs detects AMD GPUs using rocm-smi
func (d *Detector) detectAMDGPUs(ctx context.Context) ([]GPUInfo, error) {
	// Check if rocm-smi is available
	if !d.isCommandAvailable("rocm-smi") {
		return nil, fmt.Errorf("rocm-smi not found")
	}

	cmd := exec.CommandContext(ctx, "rocm-smi", "--showid", "--showproductname", "--showmeminfo", "vram", "--showpower", "--showtemp", "--showuse")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run rocm-smi: %w", err)
	}

	var gpus []GPUInfo
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Parse rocm-smi output (format varies, this is a simplified parser)
	for _, line := range lines {
		if strings.Contains(line, "GPU") && strings.Contains(line, "card") {
			gpu := GPUInfo{
				Vendor:      "AMD",
				IsAvailable: true,
			}

			// Extract GPU ID
			re := regexp.MustCompile(`card(\d+)`)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				gpu.ID = fmt.Sprintf("amd-%s", matches[1])
			}

			// This is a simplified implementation
			// In production, you would need more sophisticated parsing
			gpu.Name = "AMD GPU"
			gpu.Model = "AMD GPU"

			gpus = append(gpus, gpu)
		}
	}

	return gpus, nil
}

// detectAppleGPUs detects Apple Silicon GPUs
func (d *Detector) detectAppleGPUs(ctx context.Context) ([]GPUInfo, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("Apple GPU detection only supported on macOS")
	}

	// Use system_profiler to get GPU information
	cmd := exec.CommandContext(ctx, "system_profiler", "SPDisplaysDataType", "-json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to run system_profiler: %w", err)
	}

	var profilerData map[string]interface{}
	if err := json.Unmarshal(output, &profilerData); err != nil {
		return nil, fmt.Errorf("failed to parse system_profiler output: %w", err)
	}

	var gpus []GPUInfo

	// Parse the system profiler data
	if displays, ok := profilerData["SPDisplaysDataType"].([]interface{}); ok {
		for i, display := range displays {
			if displayMap, ok := display.(map[string]interface{}); ok {
				gpu := GPUInfo{
					ID:          fmt.Sprintf("apple-%d", i),
					Vendor:      "Apple",
					IsAvailable: true,
				}

				if name, ok := displayMap["_name"].(string); ok {
					gpu.Name = name
					gpu.Model = name
				}

				// Apple Silicon GPUs share system memory
				// We'll estimate based on the chip type
				if strings.Contains(gpu.Name, "M1") {
					if strings.Contains(gpu.Name, "Ultra") {
						gpu.VRAMTotal = 128 * 1024 // 128GB unified memory
					} else if strings.Contains(gpu.Name, "Max") {
						gpu.VRAMTotal = 64 * 1024 // 64GB unified memory
					} else {
						gpu.VRAMTotal = 16 * 1024 // 16GB unified memory
					}
				} else if strings.Contains(gpu.Name, "M2") {
					if strings.Contains(gpu.Name, "Ultra") {
						gpu.VRAMTotal = 192 * 1024 // 192GB unified memory
					} else if strings.Contains(gpu.Name, "Max") {
						gpu.VRAMTotal = 96 * 1024 // 96GB unified memory
					} else {
						gpu.VRAMTotal = 24 * 1024 // 24GB unified memory
					}
				} else if strings.Contains(gpu.Name, "M3") {
					if strings.Contains(gpu.Name, "Ultra") {
						gpu.VRAMTotal = 256 * 1024 // 256GB unified memory
					} else if strings.Contains(gpu.Name, "Max") {
						gpu.VRAMTotal = 128 * 1024 // 128GB unified memory
					} else {
						gpu.VRAMTotal = 32 * 1024 // 32GB unified memory
					}
				}

				gpu.VRAMFree = gpu.VRAMTotal // Simplified - would need actual memory usage
				gpu.VRAMUsed = 0

				gpus = append(gpus, gpu)
			}
		}
	}

	return gpus, nil
}

// detectIntelGPUs detects Intel GPUs
func (d *Detector) detectIntelGPUs(ctx context.Context) ([]GPUInfo, error) {
	// Check if intel_gpu_top is available
	if !d.isCommandAvailable("intel_gpu_top") {
		return nil, fmt.Errorf("intel_gpu_top not found")
	}

	// This is a simplified implementation
	// Intel GPU detection would require more sophisticated tools
	var gpus []GPUInfo

	// For now, just detect if Intel GPU tools are available
	gpu := GPUInfo{
		ID:          "intel-0",
		Name:        "Intel GPU",
		Model:       "Intel GPU",
		Vendor:      "Intel",
		IsAvailable: true,
	}

	gpus = append(gpus, gpu)

	return gpus, nil
}

// GetMetrics gets real-time metrics for all GPUs
func (d *Detector) GetMetrics(ctx context.Context) ([]GPUMetrics, error) {
	var allMetrics []GPUMetrics

	// Get NVIDIA metrics
	nvidiaMetrics, err := d.getNVIDIAMetrics(ctx)
	if err != nil {
		d.logger.Debug("Failed to get NVIDIA metrics", zap.Error(err))
	} else {
		allMetrics = append(allMetrics, nvidiaMetrics...)
	}

	// Get AMD metrics
	amdMetrics, err := d.getAMDMetrics(ctx)
	if err != nil {
		d.logger.Debug("Failed to get AMD metrics", zap.Error(err))
	} else {
		allMetrics = append(allMetrics, amdMetrics...)
	}

	// Get Apple metrics
	if runtime.GOOS == "darwin" {
		appleMetrics, err := d.getAppleMetrics(ctx)
		if err != nil {
			d.logger.Debug("Failed to get Apple metrics", zap.Error(err))
		} else {
			allMetrics = append(allMetrics, appleMetrics...)
		}
	}

	return allMetrics, nil
}

// Helper methods

// isCommandAvailable checks if a command is available in PATH
func (d *Detector) isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// getCUDAVersion gets the CUDA version
func (d *Detector) getCUDAVersion(ctx context.Context, nvidiaSmiCmd string) (string, error) {
	cmd := exec.CommandContext(ctx, nvidiaSmiCmd, "--query-gpu=driver_version", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// getComputeCapability gets the compute capability for an NVIDIA GPU
func (d *Detector) getComputeCapability(ctx context.Context, gpuID string) (string, error) {
	// This would require nvidia-ml-py or similar tools
	// For now, return a placeholder
	return "8.6", nil
}

// getNVIDIAMetrics gets real-time metrics for NVIDIA GPUs
func (d *Detector) getNVIDIAMetrics(ctx context.Context) ([]GPUMetrics, error) {
	nvidiaSmiCmd := "nvidia-smi" // Default
	if d.cfg != nil && d.cfg.NvidiaSmiPath != "" {
		nvidiaSmiCmd = d.cfg.NvidiaSmiPath
	} else if !d.isCommandAvailable(nvidiaSmiCmd) {
		return nil, fmt.Errorf("nvidia-smi command not found (path not configured or not in PATH)")
	}

	if !d.isCommandAvailable(nvidiaSmiCmd) { // Double check, one might be redundant with above
		return nil, fmt.Errorf("%s not available", nvidiaSmiCmd)
	}

	cmd := exec.CommandContext(ctx, nvidiaSmiCmd, "--query-gpu=index,utilization.gpu,memory.used,memory.total,power.draw,temperature.gpu,fan.speed,clocks.gr,clocks.mem", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var metrics []GPUMetrics
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) < 9 {
			d.logger.Warn("Malformed metrics line from nvidia-smi", zap.String("line", line))
			continue
		}

		metric := GPUMetrics{
			Timestamp: time.Now().UTC(),
		}

		if id := strings.TrimSpace(fields[0]); id != "" {
			metric.ID = fmt.Sprintf("nvidia-%s", id)
		}

		if utilStr := strings.TrimSpace(fields[1]); utilStr != "[N/A]" {
			if util, err := strconv.ParseUint(utilStr, 10, 8); err == nil {
				metric.Utilization = uint8(util)
			} else {
				d.logger.Warn("Failed to parse metrics GPU Utilization", zap.String("value", utilStr), zap.Error(err))
			}
		}

		if vramUsedStr := strings.TrimSpace(fields[2]); vramUsedStr != "[N/A]" {
			if vramUsed, err := strconv.ParseUint(vramUsedStr, 10, 64); err == nil {
				metric.VRAMUsed = vramUsed
			} else {
				d.logger.Warn("Failed to parse metrics VRAM Used", zap.String("value", vramUsedStr), zap.Error(err))
			}
		}

		if vramTotalStr := strings.TrimSpace(fields[3]); vramTotalStr != "[N/A]" {
			if vramTotal, err := strconv.ParseUint(vramTotalStr, 10, 64); err == nil {
				metric.VRAMTotal = vramTotal
			} else {
				d.logger.Warn("Failed to parse metrics VRAM Total", zap.String("value", vramTotalStr), zap.Error(err))
			}
		}

		if powerDrawStr := strings.TrimSpace(fields[4]); powerDrawStr != "[N/A]" {
			if powerDraw, err := strconv.ParseFloat(powerDrawStr, 32); err == nil {
				metric.PowerDraw = uint32(powerDraw)
			} else {
				d.logger.Warn("Failed to parse metrics Power Draw", zap.String("value", powerDrawStr), zap.Error(err))
			}
		}

		if tempStr := strings.TrimSpace(fields[5]); tempStr != "[N/A]" {
			if temp, err := strconv.ParseUint(tempStr, 10, 8); err == nil {
				metric.Temperature = uint8(temp)
			} else {
				d.logger.Warn("Failed to parse metrics Temperature", zap.String("value", tempStr), zap.Error(err))
			}
		}

		fanSpeedStr := strings.TrimSpace(fields[6])
		if fanSpeedStr != "[N/A]" && fanSpeedStr != "[Not Supported]" { // Handle different unavailable markers
			if fanSpeed, err := strconv.ParseUint(fanSpeedStr, 10, 8); err == nil {
				metric.FanSpeed = uint8(fanSpeed)
			} else {
				d.logger.Warn("Failed to parse metrics Fan Speed", zap.String("value", fanSpeedStr), zap.Error(err))
			}
		}

		if clockCoreStr := strings.TrimSpace(fields[7]); clockCoreStr != "[N/A]" {
			if clockCore, err := strconv.ParseUint(clockCoreStr, 10, 32); err == nil {
				metric.ClockCore = uint32(clockCore)
			} else {
				d.logger.Warn("Failed to parse metrics Clock Core", zap.String("value", clockCoreStr), zap.Error(err))
			}
		}

		if clockMemoryStr := strings.TrimSpace(fields[8]); clockMemoryStr != "[N/A]" {
			if clockMemory, err := strconv.ParseUint(clockMemoryStr, 10, 32); err == nil {
				metric.ClockMemory = uint32(clockMemory)
			} else {
				d.logger.Warn("Failed to parse metrics Clock Memory", zap.String("value", clockMemoryStr), zap.Error(err))
			}
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// getAMDMetrics gets real-time metrics for AMD GPUs
func (d *Detector) getAMDMetrics(ctx context.Context) ([]GPUMetrics, error) {
	// Simplified implementation for AMD GPUs
	return []GPUMetrics{}, nil
}

// getAppleMetrics gets real-time metrics for Apple GPUs
func (d *Detector) getAppleMetrics(ctx context.Context) ([]GPUMetrics, error) {
	// Placeholder for Apple GPU metrics
	return []GPUMetrics{}, nil
}

// DetectGPUsOnce performs an immediate, one-time detection of GPUs and returns them.
// This is useful for CLI commands or synchronous requests.
func (d *Detector) DetectGPUsOnce() ([]GPUInfo, error) {
	d.logger.Info("Performing one-time GPU detection for CLI...")
	// Call the existing comprehensive detection method.
	// DetectGPUs updates d.gpus internally and returns them.
	gpus, err := d.DetectGPUs(context.Background())
	if err != nil {
		d.logger.Error("One-time GPU detection failed during DetectGPUs call", zap.Error(err))
		return nil, err
	}
	d.logger.Info("One-time GPU detection completed via DetectGPUs call", zap.Int("gpus_found", len(gpus)))
	return gpus, nil
}

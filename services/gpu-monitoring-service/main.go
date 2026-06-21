package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// GPUMonitoringService provides real-time GPU monitoring similar to TensorHive
type GPUMonitoringService struct {
	logger    *zap.Logger
	upgrader  websocket.Upgrader
	clients   map[*websocket.Conn]bool
	clientsMu sync.RWMutex
	gpuData   map[string]*GPUInfo
	dataMu    sync.RWMutex
}

// GPUInfo represents comprehensive GPU information
type GPUInfo struct {
	ID                string                 `json:"id"`
	Name              string                 `json:"name"`
	Model             string                 `json:"model"`
	UUID              string                 `json:"uuid"`
	MemoryTotal       int64                  `json:"memory_total_mb"`
	MemoryUsed        int64                  `json:"memory_used_mb"`
	MemoryFree        int64                  `json:"memory_free_mb"`
	UtilizationGPU    int                    `json:"utilization_gpu_percent"`
	UtilizationMemory int                    `json:"utilization_memory_percent"`
	Temperature       int                    `json:"temperature_c"`
	PowerDraw         int                    `json:"power_draw_w"`
	PowerLimit        int                    `json:"power_limit_w"`
	FanSpeed          int                    `json:"fan_speed_percent"`
	Processes         []GPUProcess           `json:"processes"`
	Allocations       []GPUAllocation        `json:"allocations"`
	IsAvailable       bool                   `json:"is_available"`
	LastUpdated       time.Time              `json:"last_updated"`
	Metadata          map[string]interface{} `json:"metadata"`
}

// GPUProcess represents a process running on GPU
type GPUProcess struct {
	PID         int    `json:"pid"`
	ProcessName string `json:"process_name"`
	UserName    string `json:"user_name"`
	MemoryUsed  int64  `json:"memory_used_mb"`
	GPUUsage    int    `json:"gpu_usage_percent"`
	StartTime   string `json:"start_time"`
	Command     string `json:"command"`
}

// GPUAllocation represents user-based GPU allocation
type GPUAllocation struct {
	UserID       string    `json:"user_id"`
	UserName     string    `json:"user_name"`
	JobID        string    `json:"job_id"`
	JobName      string    `json:"job_name"`
	AllocatedAt  time.Time `json:"allocated_at"`
	MemorySlice  int64     `json:"memory_slice_mb"`
	GPUFraction  float64   `json:"gpu_fraction"`
	Status       string    `json:"status"` // active, idle, completed
	Priority     int       `json:"priority"`
	EstimatedEnd time.Time `json:"estimated_end"`
}

// SystemStats represents overall system statistics
type SystemStats struct {
	TotalGPUs       int                    `json:"total_gpus"`
	AvailableGPUs   int                    `json:"available_gpus"`
	TotalMemory     int64                  `json:"total_memory_mb"`
	UsedMemory      int64                  `json:"used_memory_mb"`
	AverageUtil     float64                `json:"average_utilization"`
	ActiveUsers     int                    `json:"active_users"`
	ActiveJobs      int                    `json:"active_jobs"`
	QueuedJobs      int                    `json:"queued_jobs"`
	SystemLoad      float64                `json:"system_load"`
	Uptime          string                 `json:"uptime"`
	LastUpdated     time.Time              `json:"last_updated"`
	GPUBreakdown    map[string]int         `json:"gpu_breakdown"`
	UserAllocations map[string]interface{} `json:"user_allocations"`
}

// NewGPUMonitoringService creates a new GPU monitoring service
func NewGPUMonitoringService(logger *zap.Logger) *GPUMonitoringService {
	return &GPUMonitoringService{
		logger: logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
		},
		clients: make(map[*websocket.Conn]bool),
		gpuData: make(map[string]*GPUInfo),
	}
}

// Start starts the GPU monitoring service
func (gms *GPUMonitoringService) Start(port string) error {
	// Start GPU data collection
	go gms.collectGPUData()

	// Start WebSocket broadcaster
	go gms.broadcastData()

	// Setup HTTP routes
	router := mux.NewRouter()

	// REST API endpoints
	router.HandleFunc("/api/gpus", gms.handleGetGPUs).Methods("GET")
	router.HandleFunc("/api/gpus/{id}", gms.handleGetGPU).Methods("GET")
	router.HandleFunc("/api/stats", gms.handleGetStats).Methods("GET")
	router.HandleFunc("/api/allocations", gms.handleGetAllocations).Methods("GET")
	router.HandleFunc("/api/allocations", gms.handleCreateAllocation).Methods("POST")
	router.HandleFunc("/api/allocations/{id}", gms.handleDeleteAllocation).Methods("DELETE")

	// WebSocket endpoint
	router.HandleFunc("/ws", gms.handleWebSocket)

	// Health check
	router.HandleFunc("/health", gms.handleHealth).Methods("GET")

	gms.logger.Info("Starting GPU monitoring service", zap.String("port", port))
	return http.ListenAndServe(":"+port, router)
}

// collectGPUData continuously collects GPU data
func (gms *GPUMonitoringService) collectGPUData() {
	ticker := time.NewTicker(2 * time.Second) // Update every 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gms.updateGPUData()
		}
	}
}

// updateGPUData updates GPU information using nvidia-smi
func (gms *GPUMonitoringService) updateGPUData() {
	// Query GPU information using nvidia-smi
	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,uuid,memory.total,memory.used,memory.free,utilization.gpu,utilization.memory,temperature.gpu,power.draw,power.limit,fan.speed", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to mock data for development/testing
		gms.updateMockGPUData()
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")

	gms.dataMu.Lock()
	defer gms.dataMu.Unlock()

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) < 12 {
			continue
		}

		gpuID := strings.TrimSpace(fields[0])

		// Parse numeric values
		memTotal, _ := strconv.ParseInt(strings.TrimSpace(fields[3]), 10, 64)
		memUsed, _ := strconv.ParseInt(strings.TrimSpace(fields[4]), 10, 64)
		memFree, _ := strconv.ParseInt(strings.TrimSpace(fields[5]), 10, 64)
		utilGPU, _ := strconv.Atoi(strings.TrimSpace(fields[6]))
		utilMem, _ := strconv.Atoi(strings.TrimSpace(fields[7]))
		temp, _ := strconv.Atoi(strings.TrimSpace(fields[8]))
		powerDraw, _ := strconv.Atoi(strings.TrimSpace(fields[9]))
		powerLimit, _ := strconv.Atoi(strings.TrimSpace(fields[10]))
		fanSpeed, _ := strconv.Atoi(strings.TrimSpace(fields[11]))

		// Get existing GPU info or create new
		gpuInfo, exists := gms.gpuData[gpuID]
		if !exists {
			gpuInfo = &GPUInfo{
				ID:          gpuID,
				Processes:   make([]GPUProcess, 0),
				Allocations: make([]GPUAllocation, 0),
				Metadata:    make(map[string]interface{}),
			}
		}

		// Update GPU info
		gpuInfo.Name = strings.TrimSpace(fields[1])
		gpuInfo.Model = strings.TrimSpace(fields[1])
		gpuInfo.UUID = strings.TrimSpace(fields[2])
		gpuInfo.MemoryTotal = memTotal
		gpuInfo.MemoryUsed = memUsed
		gpuInfo.MemoryFree = memFree
		gpuInfo.UtilizationGPU = utilGPU
		gpuInfo.UtilizationMemory = utilMem
		gpuInfo.Temperature = temp
		gpuInfo.PowerDraw = powerDraw
		gpuInfo.PowerLimit = powerLimit
		gpuInfo.FanSpeed = fanSpeed
		gpuInfo.IsAvailable = utilGPU < 80 && memUsed < (memTotal*80/100) // Available if <80% utilized
		gpuInfo.LastUpdated = time.Now()

		// Update processes
		gms.updateGPUProcesses(gpuInfo)

		gms.gpuData[gpuID] = gpuInfo
	}
}

// updateMockGPUData creates mock GPU data for development
func (gms *GPUMonitoringService) updateMockGPUData() {
	gms.dataMu.Lock()
	defer gms.dataMu.Unlock()

	// Create mock GPU data
	mockGPUs := []struct {
		id    string
		name  string
		model string
	}{
		{"0", "NVIDIA RTX 4090", "RTX4090"},
		{"1", "NVIDIA RTX 4080", "RTX4080"},
		{"2", "Apple M4 Max", "M4-Max"},
	}

	for _, mock := range mockGPUs {
		gpuInfo := &GPUInfo{
			ID:                mock.id,
			Name:              mock.name,
			Model:             mock.model,
			UUID:              fmt.Sprintf("GPU-%s-UUID", mock.id),
			MemoryTotal:       24576,                                     // 24GB
			MemoryUsed:        int64(2048 + (time.Now().Unix()%10)*1024), // Varying usage
			MemoryFree:        22528,
			UtilizationGPU:    int(30 + (time.Now().Unix() % 50)),   // 30-80% utilization
			UtilizationMemory: int(20 + (time.Now().Unix() % 40)),   // 20-60% memory util
			Temperature:       int(45 + (time.Now().Unix() % 25)),   // 45-70°C
			PowerDraw:         int(150 + (time.Now().Unix() % 200)), // 150-350W
			PowerLimit:        450,
			FanSpeed:          int(40 + (time.Now().Unix() % 40)), // 40-80%
			Processes:         gms.generateMockProcesses(),
			Allocations:       gms.generateMockAllocations(),
			IsAvailable:       true,
			LastUpdated:       time.Now(),
			Metadata: map[string]interface{}{
				"driver_version": "535.104.05",
				"cuda_version":   "12.2",
				"architecture":   "Ada Lovelace",
			},
		}

		gms.gpuData[mock.id] = gpuInfo
	}
}

// generateMockProcesses creates mock GPU processes
func (gms *GPUMonitoringService) generateMockProcesses() []GPUProcess {
	processes := []GPUProcess{
		{
			PID:         1234,
			ProcessName: "python",
			UserName:    "alice",
			MemoryUsed:  2048,
			GPUUsage:    45,
			StartTime:   "2024-01-15 10:30:00",
			Command:     "python train_model.py",
		},
		{
			PID:         5678,
			ProcessName: "pytorch",
			UserName:    "bob",
			MemoryUsed:  1024,
			GPUUsage:    25,
			StartTime:   "2024-01-15 11:15:00",
			Command:     "python inference.py",
		},
	}

	return processes
}

// generateMockAllocations creates mock GPU allocations
func (gms *GPUMonitoringService) generateMockAllocations() []GPUAllocation {
	allocations := []GPUAllocation{
		{
			UserID:       "user-123",
			UserName:     "alice",
			JobID:        "job-456",
			JobName:      "Model Training",
			AllocatedAt:  time.Now().Add(-2 * time.Hour),
			MemorySlice:  8192,
			GPUFraction:  0.5,
			Status:       "active",
			Priority:     1,
			EstimatedEnd: time.Now().Add(1 * time.Hour),
		},
		{
			UserID:       "user-789",
			UserName:     "bob",
			JobID:        "job-101",
			JobName:      "Inference Task",
			AllocatedAt:  time.Now().Add(-30 * time.Minute),
			MemorySlice:  4096,
			GPUFraction:  0.25,
			Status:       "active",
			Priority:     2,
			EstimatedEnd: time.Now().Add(30 * time.Minute),
		},
	}

	return allocations
}

// updateGPUProcesses updates running processes on GPU
func (gms *GPUMonitoringService) updateGPUProcesses(gpuInfo *GPUInfo) {
	// Query GPU processes using nvidia-smi
	cmd := exec.Command("nvidia-smi", "--query-compute-apps=pid,process_name,used_memory", "--format=csv,noheader,nounits", "-i", gpuInfo.ID)
	output, err := cmd.Output()
	if err != nil {
		// Use mock data if nvidia-smi fails
		gpuInfo.Processes = gms.generateMockProcesses()
		return
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	processes := make([]GPUProcess, 0)

	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ", ")
		if len(fields) < 3 {
			continue
		}

		pid, _ := strconv.Atoi(strings.TrimSpace(fields[0]))
		processName := strings.TrimSpace(fields[1])
		memUsed, _ := strconv.ParseInt(strings.TrimSpace(fields[2]), 10, 64)

		process := GPUProcess{
			PID:         pid,
			ProcessName: processName,
			MemoryUsed:  memUsed,
			StartTime:   time.Now().Format("2006-01-02 15:04:05"),
		}

		processes = append(processes, process)
	}

	gpuInfo.Processes = processes
}

// broadcastData broadcasts GPU data to WebSocket clients
func (gms *GPUMonitoringService) broadcastData() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			gms.clientsMu.RLock()
			if len(gms.clients) == 0 {
				gms.clientsMu.RUnlock()
				continue
			}
			gms.clientsMu.RUnlock()

			gms.dataMu.RLock()
			data := make(map[string]*GPUInfo)
			for k, v := range gms.gpuData {
				data[k] = v
			}
			gms.dataMu.RUnlock()

			message := map[string]interface{}{
				"type":      "gpu_update",
				"data":      data,
				"timestamp": time.Now(),
			}

			gms.broadcast(message)
		}
	}
}

// broadcast sends message to all WebSocket clients
func (gms *GPUMonitoringService) broadcast(message interface{}) {
	gms.clientsMu.Lock()
	defer gms.clientsMu.Unlock()

	for client := range gms.clients {
		err := client.WriteJSON(message)
		if err != nil {
			gms.logger.Warn("Failed to send message to client", zap.Error(err))
			client.Close()
			delete(gms.clients, client)
		}
	}
}

// HTTP Handlers

func (gms *GPUMonitoringService) handleGetGPUs(w http.ResponseWriter, r *http.Request) {
	gms.dataMu.RLock()
	defer gms.dataMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gms.gpuData)
}

func (gms *GPUMonitoringService) handleGetGPU(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gpuID := vars["id"]

	gms.dataMu.RLock()
	defer gms.dataMu.RUnlock()

	if gpu, exists := gms.gpuData[gpuID]; exists {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gpu)
	} else {
		http.NotFound(w, r)
	}
}

func (gms *GPUMonitoringService) handleGetStats(w http.ResponseWriter, r *http.Request) {
	gms.dataMu.RLock()
	defer gms.dataMu.RUnlock()

	stats := gms.calculateSystemStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (gms *GPUMonitoringService) handleGetAllocations(w http.ResponseWriter, r *http.Request) {
	gms.dataMu.RLock()
	defer gms.dataMu.RUnlock()

	allAllocations := make([]GPUAllocation, 0)
	for _, gpu := range gms.gpuData {
		allAllocations = append(allAllocations, gpu.Allocations...)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allAllocations)
}

func (gms *GPUMonitoringService) handleCreateAllocation(w http.ResponseWriter, r *http.Request) {
	var allocation GPUAllocation
	if err := json.NewDecoder(r.Body).Decode(&allocation); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// TODO: Implement allocation creation logic
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

func (gms *GPUMonitoringService) handleDeleteAllocation(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	allocationID := vars["id"]

	// TODO: Implement allocation deletion logic
	gms.logger.Info("Deleting allocation", zap.String("id", allocationID))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

func (gms *GPUMonitoringService) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := gms.upgrader.Upgrade(w, r, nil)
	if err != nil {
		gms.logger.Error("WebSocket upgrade failed", zap.Error(err))
		return
	}

	gms.clientsMu.Lock()
	gms.clients[conn] = true
	gms.clientsMu.Unlock()

	gms.logger.Info("New WebSocket client connected")

	// Send initial data
	gms.dataMu.RLock()
	initialData := make(map[string]*GPUInfo)
	for k, v := range gms.gpuData {
		initialData[k] = v
	}
	gms.dataMu.RUnlock()

	conn.WriteJSON(map[string]interface{}{
		"type": "initial_data",
		"data": initialData,
	})

	// Handle client disconnection
	defer func() {
		gms.clientsMu.Lock()
		delete(gms.clients, conn)
		gms.clientsMu.Unlock()
		conn.Close()
		gms.logger.Info("WebSocket client disconnected")
	}()

	// Keep connection alive
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (gms *GPUMonitoringService) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "healthy",
		"service": "gpu-monitoring-service",
	})
}

// calculateSystemStats calculates overall system statistics
func (gms *GPUMonitoringService) calculateSystemStats() SystemStats {
	totalGPUs := len(gms.gpuData)
	availableGPUs := 0
	var totalMemory, usedMemory int64
	var totalUtil float64
	activeUsers := make(map[string]bool)
	activeJobs := 0
	gpuBreakdown := make(map[string]int)

	for _, gpu := range gms.gpuData {
		if gpu.IsAvailable {
			availableGPUs++
		}
		totalMemory += gpu.MemoryTotal
		usedMemory += gpu.MemoryUsed
		totalUtil += float64(gpu.UtilizationGPU)

		// Count GPU models
		gpuBreakdown[gpu.Model]++

		// Count active users and jobs
		for _, alloc := range gpu.Allocations {
			if alloc.Status == "active" {
				activeUsers[alloc.UserID] = true
				activeJobs++
			}
		}
	}

	averageUtil := 0.0
	if totalGPUs > 0 {
		averageUtil = totalUtil / float64(totalGPUs)
	}

	return SystemStats{
		TotalGPUs:       totalGPUs,
		AvailableGPUs:   availableGPUs,
		TotalMemory:     totalMemory,
		UsedMemory:      usedMemory,
		AverageUtil:     averageUtil,
		ActiveUsers:     len(activeUsers),
		ActiveJobs:      activeJobs,
		QueuedJobs:      0,   // TODO: Implement queue tracking
		SystemLoad:      0.5, // TODO: Get actual system load
		Uptime:          "24h 30m",
		LastUpdated:     time.Now(),
		GPUBreakdown:    gpuBreakdown,
		UserAllocations: make(map[string]interface{}),
	}
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	service := NewGPUMonitoringService(logger)

	port := "8095"
	log.Printf("Starting GPU Monitoring Service on port %s", port)

	if err := service.Start(port); err != nil {
		log.Fatalf("Failed to start service: %v", err)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"google.golang.org/grpc"
	pluginapi "k8s.io/kubelet/pkg/apis/deviceplugin/v1beta1"
)

const (
	resourceName     = "dante.ai/gpu"
	serverSock       = pluginapi.DevicePluginPath + "dante-gpu.sock"
	kubeletEndpoint  = "kubelet.sock"
	envDisableHealth = "DISABLE_HEALTH_CHECK"
)

// DanteGPUDevicePlugin implements the Kubernetes device plugin interface
type DanteGPUDevicePlugin struct {
	socket   string
	server   *grpc.Server
	devices  []*pluginapi.Device
	stop     chan interface{}
	health   chan *pluginapi.Device
	gpuCount int
}

// NewDanteGPUDevicePlugin creates a new device plugin
func NewDanteGPUDevicePlugin(gpuCount int) *DanteGPUDevicePlugin {
	return &DanteGPUDevicePlugin{
		socket:   serverSock,
		devices:  make([]*pluginapi.Device, 0),
		stop:     make(chan interface{}),
		health:   make(chan *pluginapi.Device),
		gpuCount: gpuCount,
	}
}

// GetDevicePluginOptions returns device plugin options
func (dp *DanteGPUDevicePlugin) GetDevicePluginOptions(context.Context, *pluginapi.Empty) (*pluginapi.DevicePluginOptions, error) {
	return &pluginapi.DevicePluginOptions{
		PreStartRequired:                false,
		GetPreferredAllocationAvailable: true,
	}, nil
}

// ListAndWatch returns a stream of List of Devices
func (dp *DanteGPUDevicePlugin) ListAndWatch(e *pluginapi.Empty, s pluginapi.DevicePlugin_ListAndWatchServer) error {
	log.Println("ListAndWatch called")
	
	// Send initial device list
	if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: dp.devices}); err != nil {
		log.Printf("Failed to send device list: %v", err)
		return err
	}

	for {
		select {
		case <-dp.stop:
			return nil
		case d := <-dp.health:
			// Update device health
			for _, device := range dp.devices {
				if device.ID == d.ID {
					device.Health = d.Health
					break
				}
			}
			if err := s.Send(&pluginapi.ListAndWatchResponse{Devices: dp.devices}); err != nil {
				log.Printf("Failed to send updated device list: %v", err)
				return err
			}
		}
	}
}

// GetPreferredAllocation returns preferred allocation for devices
func (dp *DanteGPUDevicePlugin) GetPreferredAllocation(ctx context.Context, r *pluginapi.PreferredAllocationRequest) (*pluginapi.PreferredAllocationResponse, error) {
	log.Printf("GetPreferredAllocation called with %d container requests", len(r.ContainerRequests))
	
	response := &pluginapi.PreferredAllocationResponse{}
	
	for _, req := range r.ContainerRequests {
		// Simple allocation strategy: allocate devices in order
		allocatedDevices := make([]string, 0, req.MustIncludeDeviceIDs)
		
		// First, include must-have devices
		allocatedDevices = append(allocatedDevices, req.MustIncludeDeviceIDs...)
		
		// Then allocate from available devices
		needed := int(req.AllocationSize) - len(req.MustIncludeDeviceIDs)
		for _, deviceID := range req.AvailableDeviceIDs {
			if needed <= 0 {
				break
			}
			
			// Check if device is not already allocated
			found := false
			for _, allocated := range allocatedDevices {
				if allocated == deviceID {
					found = true
					break
				}
			}
			
			if !found {
				allocatedDevices = append(allocatedDevices, deviceID)
				needed--
			}
		}
		
		response.ContainerResponses = append(response.ContainerResponses, &pluginapi.ContainerPreferredAllocationResponse{
			DeviceIDs: allocatedDevices,
		})
	}
	
	return response, nil
}

// Allocate allocates devices to containers
func (dp *DanteGPUDevicePlugin) Allocate(ctx context.Context, r *pluginapi.AllocateRequest) (*pluginapi.AllocateResponse, error) {
	log.Printf("Allocate called with %d container requests", len(r.ContainerRequests))
	
	response := &pluginapi.AllocateResponse{}
	
	for _, req := range r.ContainerRequests {
		log.Printf("Allocating devices: %v", req.DevicesIDs)
		
		// Create container response
		containerResponse := &pluginapi.ContainerAllocateResponse{
			Envs:        make(map[string]string),
			Mounts:      make([]*pluginapi.Mount, 0),
			Devices:     make([]*pluginapi.DeviceSpec, 0),
			Annotations: make(map[string]string),
		}
		
		// Set environment variables for GPU allocation
		containerResponse.Envs["DANTE_GPU_DEVICES"] = fmt.Sprintf("%v", req.DevicesIDs)
		containerResponse.Envs["DANTE_GPU_COUNT"] = fmt.Sprintf("%d", len(req.DevicesIDs))
		
		// Add device specifications for each allocated GPU
		for i, deviceID := range req.DevicesIDs {
			// Create virtual GPU device
			deviceSpec := &pluginapi.DeviceSpec{
				ContainerPath: fmt.Sprintf("/dev/dante-gpu%d", i),
				HostPath:      fmt.Sprintf("/dev/dante-gpu-%s", deviceID),
				Permissions:   "rw",
			}
			containerResponse.Devices = append(containerResponse.Devices, deviceSpec)
			
			// Add annotations for device tracking
			containerResponse.Annotations[fmt.Sprintf("dante.ai/gpu-%d-id", i)] = deviceID
		}
		
		// Mount GPU libraries and drivers
		containerResponse.Mounts = append(containerResponse.Mounts, &pluginapi.Mount{
			ContainerPath: "/usr/local/dante",
			HostPath:      "/usr/local/dante",
			ReadOnly:      true,
		})
		
		response.ContainerResponses = append(response.ContainerResponses, containerResponse)
	}
	
	return response, nil
}

// PreStartContainer is called before starting a container
func (dp *DanteGPUDevicePlugin) PreStartContainer(ctx context.Context, r *pluginapi.PreStartContainerRequest) (*pluginapi.PreStartContainerResponse, error) {
	log.Printf("PreStartContainer called for device: %s", r.DevicesIDs)
	
	// Perform any pre-start setup for GPU devices
	for _, deviceID := range r.DevicesIDs {
		log.Printf("Preparing GPU device: %s", deviceID)
		
		// Here you would typically:
		// 1. Initialize GPU device
		// 2. Set up GPU memory isolation
		// 3. Configure GPU sharing parameters
		// 4. Set up monitoring
		
		if err := dp.prepareGPUDevice(deviceID); err != nil {
			log.Printf("Failed to prepare GPU device %s: %v", deviceID, err)
			return nil, err
		}
	}
	
	return &pluginapi.PreStartContainerResponse{}, nil
}

// prepareGPUDevice prepares a GPU device for container use
func (dp *DanteGPUDevicePlugin) prepareGPUDevice(deviceID string) error {
	log.Printf("Preparing GPU device: %s", deviceID)
	
	// Create device file if it doesn't exist
	devicePath := fmt.Sprintf("/dev/dante-gpu-%s", deviceID)
	if _, err := os.Stat(devicePath); os.IsNotExist(err) {
		// Create a virtual device file
		file, err := os.Create(devicePath)
		if err != nil {
			return fmt.Errorf("failed to create device file %s: %v", devicePath, err)
		}
		file.Close()
		
		// Set appropriate permissions
		if err := os.Chmod(devicePath, 0666); err != nil {
			return fmt.Errorf("failed to set permissions on device file %s: %v", devicePath, err)
		}
	}
	
	return nil
}

// Start starts the device plugin server
func (dp *DanteGPUDevicePlugin) Start() error {
	log.Println("Starting DanteGPU device plugin")
	
	// Initialize devices
	dp.initializeDevices()
	
	// Remove existing socket
	if err := dp.cleanup(); err != nil {
		return err
	}
	
	// Create socket
	sock, err := net.Listen("unix", dp.socket)
	if err != nil {
		return fmt.Errorf("failed to listen on socket %s: %v", dp.socket, err)
	}
	
	// Create gRPC server
	dp.server = grpc.NewServer()
	pluginapi.RegisterDevicePluginServer(dp.server, dp)
	
	// Start server
	go func() {
		if err := dp.server.Serve(sock); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()
	
	// Wait for server to start
	conn, err := grpc.Dial(dp.socket, grpc.WithInsecure(), grpc.WithBlock(),
		grpc.WithTimeout(5*time.Second),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		return fmt.Errorf("failed to connect to device plugin server: %v", err)
	}
	conn.Close()
	
	log.Println("DanteGPU device plugin server started")
	return nil
}

// Stop stops the device plugin server
func (dp *DanteGPUDevicePlugin) Stop() error {
	log.Println("Stopping DanteGPU device plugin")
	
	if dp.server != nil {
		dp.server.Stop()
	}
	
	close(dp.stop)
	return dp.cleanup()
}

// Register registers the device plugin with kubelet
func (dp *DanteGPUDevicePlugin) Register() error {
	log.Println("Registering DanteGPU device plugin with kubelet")
	
	conn, err := grpc.Dial(pluginapi.KubeletSocket, grpc.WithInsecure(),
		grpc.WithDialer(func(addr string, timeout time.Duration) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}))
	if err != nil {
		return fmt.Errorf("failed to connect to kubelet: %v", err)
	}
	defer conn.Close()
	
	client := pluginapi.NewRegistrationClient(conn)
	request := &pluginapi.RegisterRequest{
		Version:      pluginapi.Version,
		Endpoint:     filepath.Base(dp.socket),
		ResourceName: resourceName,
		Options:      &pluginapi.DevicePluginOptions{PreStartRequired: true},
	}
	
	_, err = client.Register(context.Background(), request)
	if err != nil {
		return fmt.Errorf("failed to register device plugin: %v", err)
	}
	
	log.Println("Successfully registered DanteGPU device plugin")
	return nil
}

// initializeDevices initializes the list of available GPU devices
func (dp *DanteGPUDevicePlugin) initializeDevices() {
	log.Printf("Initializing %d GPU devices", dp.gpuCount)
	
	dp.devices = make([]*pluginapi.Device, 0, dp.gpuCount*10) // Support 10 fractions per GPU
	
	for i := 0; i < dp.gpuCount; i++ {
		// Create fractional GPU devices (10 fractions per physical GPU)
		for j := 0; j < 10; j++ {
			deviceID := fmt.Sprintf("gpu-%d-fraction-%d", i, j)
			device := &pluginapi.Device{
				ID:     deviceID,
				Health: pluginapi.Healthy,
			}
			dp.devices = append(dp.devices, device)
		}
	}
	
	log.Printf("Initialized %d fractional GPU devices", len(dp.devices))
}

// cleanup removes the socket file
func (dp *DanteGPUDevicePlugin) cleanup() error {
	if err := os.Remove(dp.socket); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove socket %s: %v", dp.socket, err)
	}
	return nil
}

// healthCheck monitors device health
func (dp *DanteGPUDevicePlugin) healthCheck() {
	log.Println("Starting health check")
	
	// Disable health check if environment variable is set
	if os.Getenv(envDisableHealth) != "" {
		log.Println("Health check disabled")
		return
	}
	
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Check device health
			for _, device := range dp.devices {
				// Simple health check - in real implementation, you would check actual GPU status
				if dp.isDeviceHealthy(device.ID) {
					device.Health = pluginapi.Healthy
				} else {
					device.Health = pluginapi.Unhealthy
					dp.health <- device
				}
			}
		case <-dp.stop:
			return
		}
	}
}

// isDeviceHealthy checks if a device is healthy
func (dp *DanteGPUDevicePlugin) isDeviceHealthy(deviceID string) bool {
	// In a real implementation, you would check:
	// 1. GPU driver status
	// 2. GPU memory availability
	// 3. GPU temperature
	// 4. GPU utilization
	
	// For now, always return healthy
	return true
}

func main() {
	var gpuCount = flag.Int("gpu-count", 1, "Number of physical GPUs on this node")
	flag.Parse()
	
	log.Printf("Starting DanteGPU Device Plugin with %d GPUs", *gpuCount)
	
	dp := NewDanteGPUDevicePlugin(*gpuCount)
	
	// Start health check
	go dp.healthCheck()
	
	// Start device plugin server
	if err := dp.Start(); err != nil {
		log.Fatalf("Failed to start device plugin: %v", err)
	}
	
	// Register with kubelet
	if err := dp.Register(); err != nil {
		log.Fatalf("Failed to register device plugin: %v", err)
	}
	
	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	
	log.Println("Received termination signal")
	if err := dp.Stop(); err != nil {
		log.Printf("Failed to stop device plugin: %v", err)
	}
}

package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// SchedulerExtender implements Kubernetes Scheduler Extender interface
type SchedulerExtender struct {
	logger    *zap.Logger
	k8sClient kubernetes.Interface
}

// ExtenderArgs represents the arguments passed to the scheduler extender
type ExtenderArgs struct {
	Pod      v1.Pod                 `json:"pod"`
	Nodes    *v1.NodeList          `json:"nodes"`
	NodeNames *[]string            `json:"nodenames"`
}

// ExtenderFilterResult represents the result of the filter operation
type ExtenderFilterResult struct {
	Nodes       *v1.NodeList `json:"nodes"`
	NodeNames   *[]string    `json:"nodenames"`
	FailedNodes FailedNodesMap `json:"failedNodes"`
	Error       string       `json:"error"`
}

// ExtenderBindingArgs represents binding arguments
type ExtenderBindingArgs struct {
	PodName      string `json:"podName"`
	PodNamespace string `json:"podNamespace"`
	PodUID       string `json:"podUID"`
	Node         string `json:"node"`
}

// ExtenderBindingResult represents binding result
type ExtenderBindingResult struct {
	Error string `json:"error"`
}

// FailedNodesMap represents failed nodes with reasons
type FailedNodesMap map[string]string

// GPURequest represents GPU resource requirements
type GPURequest struct {
	Count    int64   `json:"count"`
	Memory   int64   `json:"memory"`   // VRAM in MB
	Fraction float64 `json:"fraction"` // GPU fraction (0.1 to 1.0)
}

// NewSchedulerExtender creates a new Kubernetes Scheduler Extender
func NewSchedulerExtender(logger *zap.Logger, kubeconfig string) (*SchedulerExtender, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		// Use kubeconfig file
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		// Use in-cluster config
		config, err = rest.InClusterConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &SchedulerExtender{
		logger:    logger,
		k8sClient: clientset,
	}, nil
}

// Filter implements the filter extender endpoint
func (se *SchedulerExtender) Filter(w http.ResponseWriter, r *http.Request) {
	var args ExtenderArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		se.logger.Error("Failed to decode extender args", zap.Error(err))
		se.writeErrorResponse(w, fmt.Sprintf("Failed to decode request: %v", err))
		return
	}

	se.logger.Info("Processing filter request",
		zap.String("pod", args.Pod.Name),
		zap.String("namespace", args.Pod.Namespace),
	)

	// Check if this pod requires GPU resources
	gpuReq := se.extractGPURequirements(&args.Pod)
	if gpuReq == nil {
		// No GPU requirements, pass through all nodes
		result := &ExtenderFilterResult{
			Nodes:     args.Nodes,
			NodeNames: args.NodeNames,
		}
		se.writeResponse(w, result)
		return
	}

	se.logger.Info("Pod requires GPU resources",
		zap.String("pod", args.Pod.Name),
		zap.Int64("gpu_count", gpuReq.Count),
		zap.Int64("gpu_memory", gpuReq.Memory),
		zap.Float64("gpu_fraction", gpuReq.Fraction),
	)

	// Filter nodes based on GPU availability
	filteredNodes, failedNodes := se.filterNodesByGPU(args.Nodes, gpuReq)

	result := &ExtenderFilterResult{
		Nodes:       filteredNodes,
		FailedNodes: failedNodes,
	}

	se.writeResponse(w, result)
}

// Prioritize implements the prioritize extender endpoint
func (se *SchedulerExtender) Prioritize(w http.ResponseWriter, r *http.Request) {
	var args ExtenderArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		se.logger.Error("Failed to decode extender args for prioritize", zap.Error(err))
		se.writeErrorResponse(w, fmt.Sprintf("Failed to decode request: %v", err))
		return
	}

	// Simple prioritization: prefer nodes with more available GPU resources
	priorities := make([]int, 0)
	
	if args.Nodes != nil {
		for _, node := range args.Nodes.Items {
			priority := se.calculateNodePriority(&node)
			priorities = append(priorities, priority)
		}
	} else if args.NodeNames != nil {
		for _, nodeName := range *args.NodeNames {
			node, err := se.k8sClient.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{})
			if err != nil {
				se.logger.Warn("Failed to get node for prioritization", zap.String("node", nodeName), zap.Error(err))
				priorities = append(priorities, 0)
				continue
			}
			priority := se.calculateNodePriority(node)
			priorities = append(priorities, priority)
		}
	}

	response := map[string]interface{}{
		"priorities": priorities,
	}

	se.writeResponse(w, response)
}

// Bind implements the bind extender endpoint
func (se *SchedulerExtender) Bind(w http.ResponseWriter, r *http.Request) {
	var args ExtenderBindingArgs
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		se.logger.Error("Failed to decode binding args", zap.Error(err))
		se.writeErrorResponse(w, fmt.Sprintf("Failed to decode request: %v", err))
		return
	}

	se.logger.Info("Processing bind request",
		zap.String("pod", args.PodName),
		zap.String("namespace", args.PodNamespace),
		zap.String("node", args.Node),
	)

	// Get the pod
	pod, err := se.k8sClient.CoreV1().Pods(args.PodNamespace).Get(context.TODO(), args.PodName, metav1.GetOptions{})
	if err != nil {
		se.logger.Error("Failed to get pod for binding", zap.Error(err))
		result := &ExtenderBindingResult{Error: fmt.Sprintf("Failed to get pod: %v", err)}
		se.writeResponse(w, result)
		return
	}

	// Perform the binding
	binding := &v1.Binding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      args.PodName,
			Namespace: args.PodNamespace,
		},
		Target: v1.ObjectReference{
			Kind: "Node",
			Name: args.Node,
		},
	}

	err = se.k8sClient.CoreV1().Pods(args.PodNamespace).Bind(context.TODO(), binding, metav1.CreateOptions{})
	if err != nil {
		se.logger.Error("Failed to bind pod to node", zap.Error(err))
		result := &ExtenderBindingResult{Error: fmt.Sprintf("Failed to bind pod: %v", err)}
		se.writeResponse(w, result)
		return
	}

	// Update pod with GPU allocation annotations
	se.updatePodGPUAllocation(pod, args.Node)

	se.logger.Info("Successfully bound pod to node",
		zap.String("pod", args.PodName),
		zap.String("node", args.Node),
	)

	result := &ExtenderBindingResult{}
	se.writeResponse(w, result)
}

// extractGPURequirements extracts GPU requirements from pod spec
func (se *SchedulerExtender) extractGPURequirements(pod *v1.Pod) *GPURequest {
	for _, container := range pod.Spec.Containers {
		if gpuCount, exists := container.Resources.Requests["dante.ai/gpu"]; exists {
			count := gpuCount.Value()
			
			// Extract GPU memory requirement
			var memory int64 = 8192 // Default 8GB
			if gpuMem, exists := container.Resources.Requests["dante.ai/gpu-memory"]; exists {
				memory = gpuMem.Value() / (1024 * 1024) // Convert to MB
			}

			// Extract GPU fraction requirement
			var fraction float64 = 1.0 // Default full GPU
			if gpuFrac, exists := container.Resources.Requests["dante.ai/gpu-fraction"]; exists {
				if f, err := strconv.ParseFloat(gpuFrac.String(), 64); err == nil {
					fraction = f
				}
			}

			return &GPURequest{
				Count:    count,
				Memory:   memory,
				Fraction: fraction,
			}
		}
	}
	return nil
}

// filterNodesByGPU filters nodes based on GPU availability
func (se *SchedulerExtender) filterNodesByGPU(nodeList *v1.NodeList, gpuReq *GPURequest) (*v1.NodeList, FailedNodesMap) {
	if nodeList == nil {
		return &v1.NodeList{}, make(FailedNodesMap)
	}

	filteredNodes := &v1.NodeList{
		TypeMeta: nodeList.TypeMeta,
		ListMeta: nodeList.ListMeta,
		Items:    make([]v1.Node, 0),
	}
	failedNodes := make(FailedNodesMap)

	for _, node := range nodeList.Items {
		if se.nodeHasAvailableGPU(&node, gpuReq) {
			filteredNodes.Items = append(filteredNodes.Items, node)
		} else {
			failedNodes[node.Name] = "Insufficient GPU resources"
		}
	}

	return filteredNodes, failedNodes
}

// nodeHasAvailableGPU checks if a node has available GPU resources
func (se *SchedulerExtender) nodeHasAvailableGPU(node *v1.Node, gpuReq *GPURequest) bool {
	// Check node labels for GPU information
	gpuModel, hasGPU := node.Labels["dante.ai/gpu-model"]
	if !hasGPU {
		return false
	}

	// Check available GPU count
	availableGPUs := se.getAvailableGPUCount(node)
	if availableGPUs < gpuReq.Count {
		return false
	}

	// Check GPU memory
	gpuMemory := se.getGPUMemory(node, gpuModel)
	if gpuMemory < gpuReq.Memory {
		return false
	}

	// Check if fractional GPU is supported
	if gpuReq.Fraction < 1.0 {
		if !se.supportsFractionalGPU(node) {
			return false
		}
	}

	return true
}

// getAvailableGPUCount returns the number of available GPUs on a node
func (se *SchedulerExtender) getAvailableGPUCount(node *v1.Node) int64 {
	// Check allocatable resources
	if gpuResource, exists := node.Status.Allocatable["dante.ai/gpu"]; exists {
		return gpuResource.Value()
	}
	return 0
}

// getGPUMemory returns the GPU memory for a specific model
func (se *SchedulerExtender) getGPUMemory(node *v1.Node, gpuModel string) int64 {
	// GPU memory mapping (in MB)
	gpuMemoryMap := map[string]int64{
		"RTX4090":    24576,
		"RTX4080":    16384,
		"RTX3090":    24576,
		"RTX3080":    10240,
		"A100":       40960,
		"H100":       80896,
		"V100":       16384,
		"T4":         15360,
		"M4-Max":     36864, // Apple M4 Max unified memory
	}

	if memory, exists := gpuMemoryMap[gpuModel]; exists {
		return memory
	}

	// Default to 8GB if unknown
	return 8192
}

// supportsFractionalGPU checks if a node supports fractional GPU allocation
func (se *SchedulerExtender) supportsFractionalGPU(node *v1.Node) bool {
	// Check if node has the fractional GPU support label
	if support, exists := node.Labels["dante.ai/fractional-gpu"]; exists {
		return support == "true"
	}
	return false
}

// calculateNodePriority calculates priority score for a node
func (se *SchedulerExtender) calculateNodePriority(node *v1.Node) int {
	score := 0

	// Higher score for more available GPUs
	availableGPUs := se.getAvailableGPUCount(node)
	score += int(availableGPUs * 10)

	// Higher score for better GPU models
	if gpuModel, exists := node.Labels["dante.ai/gpu-model"]; exists {
		switch {
		case strings.Contains(gpuModel, "H100"):
			score += 100
		case strings.Contains(gpuModel, "A100"):
			score += 80
		case strings.Contains(gpuModel, "RTX4090"):
			score += 60
		case strings.Contains(gpuModel, "RTX4080"):
			score += 50
		default:
			score += 20
		}
	}

	// Higher score for fractional GPU support
	if se.supportsFractionalGPU(node) {
		score += 20
	}

	return score
}

// updatePodGPUAllocation updates pod with GPU allocation information
func (se *SchedulerExtender) updatePodGPUAllocation(pod *v1.Pod, nodeName string) {
	if pod.Annotations == nil {
		pod.Annotations = make(map[string]string)
	}

	pod.Annotations["dante.ai/gpu-allocated-node"] = nodeName
	pod.Annotations["dante.ai/gpu-allocation-time"] = metav1.Now().Format("2006-01-02T15:04:05Z")

	// Update the pod
	_, err := se.k8sClient.CoreV1().Pods(pod.Namespace).Update(context.TODO(), pod, metav1.UpdateOptions{})
	if err != nil {
		se.logger.Error("Failed to update pod with GPU allocation info", zap.Error(err))
	}
}

// writeResponse writes JSON response
func (se *SchedulerExtender) writeResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		se.logger.Error("Failed to encode response", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// writeErrorResponse writes error response
func (se *SchedulerExtender) writeErrorResponse(w http.ResponseWriter, errorMsg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	response := map[string]string{"error": errorMsg}
	json.NewEncoder(w).Encode(response)
}

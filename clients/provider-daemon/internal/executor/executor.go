package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/dante-gpu/dante-backend/provider-daemon/internal/billing"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/config"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/gpu"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/models"

	// "github.com/docker/docker/api/types" // Commented out if not directly used
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"   // Ensured for image.PullOptions
	"github.com/docker/docker/api/types/network" // If network.NetworkingConfig is used
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ExecutionResult holds the outcome of a task execution.
type ExecutionResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error // For errors during the execution setup or process itself, not script/container errors
}

// Executor defines the interface for running tasks.
type Executor interface {
	Execute(ctx context.Context, task *models.Task, workspacePath string, logger *zap.Logger) ExecutionResult
}

// ScriptExecutor implements the Executor interface for running shell scripts.
type ScriptExecutor struct{}

// NewScriptExecutor creates a new ScriptExecutor.
func NewScriptExecutor() *ScriptExecutor {
	return &ScriptExecutor{}
}

// Execute runs a script defined in the task's parameters.
// It expects task.JobParams to contain:
// - "script_content": string (the script)
// - "script_interpreter": string (e.g., "/bin/bash", "python3")
// - "script_filename": string (e.g., "run.sh", "main.py" - optional, defaults if not provided)
// - "timeout_seconds": int (optional, task execution timeout)
func (se *ScriptExecutor) Execute(ctx context.Context, task *models.Task, workspacePath string, logger *zap.Logger) ExecutionResult {
	logger.Info("Starting script execution", zap.String("job_id", task.JobID), zap.String("workspace", workspacePath))

	scriptContent, ok := task.JobParams["script_content"].(string)
	if !ok || strings.TrimSpace(scriptContent) == "" {
		logger.Error("Script content not found or empty in task parameters", zap.String("job_id", task.JobID))
		return ExecutionResult{Error: fmt.Errorf("script_content is required and cannot be empty"), ExitCode: -1}
	}

	interpreter, ok := task.JobParams["script_interpreter"].(string)
	if !ok || strings.TrimSpace(interpreter) == "" {
		logger.Warn("Script interpreter not specified, defaulting to /bin/sh", zap.String("job_id", task.JobID))
		interpreter = "/bin/sh" // Default interpreter
	}

	scriptFilename, ok := task.JobParams["script_filename"].(string)
	if !ok || strings.TrimSpace(scriptFilename) == "" {
		if interpreter == "python" || interpreter == "python3" {
			scriptFilename = "main.py"
		} else {
			scriptFilename = "task_script.sh"
		}
	}
	scriptPath := filepath.Join(workspacePath, scriptFilename)

	// Write the script content to a file in the workspace
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755) // Make it executable
	if err != nil {
		logger.Error("Failed to write script to workspace", zap.String("job_id", task.JobID), zap.Error(err))
		return ExecutionResult{Error: fmt.Errorf("failed to write script file: %w", err), ExitCode: -1}
	}
	logger.Info("Script written to file", zap.String("path", scriptPath))

	var execCtx context.Context
	var cancel context.CancelFunc

	timeoutSeconds, hasTimeout := task.JobParams["timeout_seconds"].(float64) // YAML might parse numbers as float64
	if hasTimeout && timeoutSeconds > 0 {
		execCtx, cancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
		logger.Info("Script execution timeout set", zap.Float64("seconds", timeoutSeconds))
	} else {
		execCtx, cancel = context.WithCancel(ctx) // No timeout or indefinite
	}
	defer cancel()

	cmd := exec.CommandContext(execCtx, interpreter, scriptPath)
	cmd.Dir = workspacePath // Execute from the workspace directory

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startTime := time.Now()
	logger.Info("Executing script", zap.String("interpreter", interpreter), zap.String("script", scriptPath))

	runErr := cmd.Run()
	duration := time.Since(startTime)
	logger.Info("Script execution finished", zap.Duration("duration", duration), zap.Error(runErr))

	result := ExecutionResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Error = fmt.Errorf("script exited with code %d: %w", result.ExitCode, exitErr) // Capture ExitError
		} else if execCtx.Err() == context.DeadlineExceeded {
			result.Error = fmt.Errorf("script execution timed out after %v", time.Duration(timeoutSeconds)*time.Second)
			result.ExitCode = -2 // Specific code for timeout
			logger.Warn("Script execution timed out", zap.String("job_id", task.JobID))
		} else {
			result.Error = fmt.Errorf("script execution failed: %w", runErr)
			result.ExitCode = -1 // Generic error
			logger.Error("Script execution failed with non-ExitError", zap.String("job_id", task.JobID), zap.Error(runErr))
		}
	} else {
		result.ExitCode = 0
	}

	logger.Debug("Script execution details",
		zap.Int("exit_code", result.ExitCode),
		zap.String("stdout_len", fmt.Sprintf("%d bytes", len(result.Stdout))),
		zap.String("stderr_len", fmt.Sprintf("%d bytes", len(result.Stderr))),
	)

	return result
}

// DockerExecutor implements the Executor interface for running tasks in Docker containers.
type DockerExecutor struct {
	cli           *client.Client
	logger        *zap.Logger
	billingClient *billing.Client
	gpuDetector   *gpu.Detector
	// execCfg       *config.ExecutorSettings // Optionally store if needed by other methods
}

// NewDockerExecutor creates a new DockerExecutor.
// It initializes a Docker client, preferring execCfg.DockerEndpoint if provided.
func NewDockerExecutor(execCfg *config.ExecutorSettings, logger *zap.Logger, billingClient *billing.Client, gpuDetector *gpu.Detector) (*DockerExecutor, error) {
	var cliOpts []client.Opt
	if execCfg != nil && execCfg.DockerEndpoint != "" {
		logger.Info("Using configured Docker endpoint", zap.String("endpoint", execCfg.DockerEndpoint))
		cliOpts = append(cliOpts, client.WithHost(execCfg.DockerEndpoint))
	} else {
		// If no specific endpoint in config, use Docker's default (from environment or system default)
		logger.Info("Using Docker client with default environment settings (e.g., DOCKER_HOST or default socket).")
		cliOpts = append(cliOpts, client.FromEnv)
	}
	cliOpts = append(cliOpts, client.WithAPIVersionNegotiation())

	cli, err := client.NewClientWithOpts(cliOpts...)
	if err != nil {
		logger.Error("Failed to create Docker client", zap.Error(err))
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Ping the Docker daemon to ensure connectivity
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second) // Short timeout for ping
	defer pingCancel()
	if _, err := cli.Ping(pingCtx); err != nil {
		logger.Error("Failed to ping Docker daemon", zap.Error(err))
		cli.Close() // Close the client if ping fails
		return nil, fmt.Errorf("failed to ping Docker daemon: %w", err)
	}
	logger.Info("Docker client initialized and connected to Docker daemon")
	return &DockerExecutor{
		cli:           cli,
		logger:        logger,
		billingClient: billingClient,
		gpuDetector:   gpuDetector,
		// execCfg: execCfg, // Store if other methods need it directly
	}, nil
}

// Execute runs a task in a Docker container based on task.JobParams.
// Expected JobParams:
// - "docker_image": string (e.g., "ubuntu:latest", "nvidia/cuda:11.8.0-base-ubuntu22.04") - REQUIRED
// - "docker_command": []string (command to run, e.g., ["python", "script.py"]) - REQUIRED if no Entrypoint/Cmd in image
// - "docker_env_vars": map[string]string (e.g., {"API_KEY": "secret"}) - OPTIONAL
// - "docker_gpus": string (e.g., "all" to request all GPUs, or device IDs. Requires nvidia-container-toolkit) - OPTIONAL
// - "timeout_seconds": float64 (container execution timeout) - OPTIONAL
// - "script_content": string (script to run, if specified, it's written to workspace and command is set to run it) - OPTIONAL
// - "script_interpreter": string (e.g., "/bin/bash", "python3", used if "script_content" is provided) - OPTIONAL
// - "script_filename": string (e.g., "run.sh", defaults appropriately if "script_content" is provided) - OPTIONAL
func (de *DockerExecutor) Execute(ctx context.Context, task *models.Task, workspacePath string, logger *zap.Logger) ExecutionResult {
	jobLogger := logger.With(zap.String("job_id", task.JobID), zap.String("executor", "docker"))
	jobLogger.Info("Starting Docker execution", zap.String("workspace", workspacePath))

	imageName, ok := task.JobParams["docker_image"].(string)
	if !ok || imageName == "" {
		jobLogger.Error("Docker image not specified in task parameters")
		return ExecutionResult{Error: fmt.Errorf("docker_image is required"), ExitCode: -1}
	}

	var cmdSlice []string
	if cmdInterface, ok := task.JobParams["docker_command"]; ok {
		if cs, ok := cmdInterface.([]interface{}); ok {
			for _, c := range cs {
				if s, ok := c.(string); ok {
					cmdSlice = append(cmdSlice, s)
				}
			}
		} else if csString, ok := cmdInterface.([]string); ok {
			cmdSlice = csString
		}
	}

	// Handle script_content: if provided, write to workspace and set command to run it
	if scriptContent, scOK := task.JobParams["script_content"].(string); scOK && strings.TrimSpace(scriptContent) != "" {
		interpreter := "/bin/sh" // Default interpreter for script_content
		if si, siOK := task.JobParams["script_interpreter"].(string); siOK && strings.TrimSpace(si) != "" {
			interpreter = si
		}
		scriptFilename := "container_script.sh"
		if sf, sfOK := task.JobParams["script_filename"].(string); sfOK && strings.TrimSpace(sf) != "" {
			scriptFilename = sf
		} else if interpreter == "python" || interpreter == "python3" {
			scriptFilename = "main.py"
		}

		scriptPathInWorkspace := filepath.Join(workspacePath, scriptFilename)
		err := os.WriteFile(scriptPathInWorkspace, []byte(scriptContent), 0755)
		if err != nil {
			jobLogger.Error("Failed to write script_content to workspace", zap.Error(err))
			return ExecutionResult{Error: fmt.Errorf("failed to write script_content to workspace: %w", err), ExitCode: -1}
		}
		jobLogger.Info("Script_content written to workspace", zap.String("path", scriptPathInWorkspace))
		// Command will be to execute this script from within the container's workspace mount
		cmdSlice = []string{interpreter, filepath.Join("/workspace", scriptFilename)} // /workspace is the target mount point
	}

	if len(cmdSlice) == 0 {
		jobLogger.Warn("No command specified and no script_content provided; relying on image's CMD/ENTRYPOINT")
	}

	var envVars []string
	if envMap, ok := task.JobParams["docker_env_vars"].(map[string]interface{}); ok {
		for k, v := range envMap {
			if vStr, ok := v.(string); ok {
				envVars = append(envVars, fmt.Sprintf("%s=%s", k, vStr))
			}
		}
	}

	// --- Pull Image ---
	jobLogger.Info("Pulling Docker image if not present", zap.String("image", imageName))
	pullCtx, pullCancel := context.WithTimeout(ctx, 5*time.Minute) // Timeout for image pull
	defer pullCancel()

	imgPullOut, err := de.cli.ImagePull(pullCtx, imageName, image.PullOptions{}) // Corrected
	if err != nil {
		jobLogger.Error("Failed to pull Docker image", zap.String("image", imageName), zap.Error(err))
		return ExecutionResult{Error: fmt.Errorf("failed to pull image %s: %w", imageName, err), ExitCode: -1}
	}
	defer imgPullOut.Close()
	// It's good practice to consume the output of ImagePull
	if _, err = io.Copy(io.Discard, imgPullOut); err != nil {
		jobLogger.Warn("Error consuming image pull output", zap.Error(err))
	}
	jobLogger.Info("Image pull process completed (or image was already present)", zap.String("image", imageName))

	// --- Prepare Container Config ---
	containerConfig := &container.Config{
		Image:        imageName,
		Cmd:          cmdSlice,
		Env:          envVars,
		WorkingDir:   "/workspace", // Tasks will run inside the mounted workspace
		Tty:          false,        // No TTY for non-interactive tasks
		AttachStdout: true,
		AttachStderr: true,
	}

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/workspace:rw", workspacePath), // Mount workspace read-write
		},
		AutoRemove: false, // Set to false to inspect logs/state after failure, will remove manually
	}

	// GPU Configuration (Enhanced to be more specific - requires nvidia-container-toolkit)
	if gpuRequestParam, ok := task.JobParams["docker_gpus"].(string); ok && gpuRequestParam != "" {
		gpuRequestValue := strings.ToLower(strings.TrimSpace(gpuRequestParam))
		deviceRequest := container.DeviceRequest{
			Driver:       "nvidia", // Or often left empty if default runtime is NVIDIA
			Capabilities: [][]string{{"gpu"}},
		}

		if gpuRequestValue == "all" {
			jobLogger.Info("Requesting all GPUs for container")
			deviceRequest.Count = -1 // All GPUs
		} else if ids := strings.Split(gpuRequestValue, ","); len(ids) > 0 && !isNumeric(ids[0]) {
			// Check if the first part is non-numeric, assume list of IDs
			// (simple check; more robust parsing might be needed if IDs can be numeric but are not counts)
			validIDs := []string{}
			for _, id := range ids {
				idTrimmed := strings.TrimSpace(id)
				if idTrimmed != "" {
					validIDs = append(validIDs, idTrimmed)
				}
			}
			if len(validIDs) > 0 {
				jobLogger.Info("Requesting specific GPUs by ID for container", zap.Strings("gpu_ids", validIDs))
				deviceRequest.DeviceIDs = validIDs
				deviceRequest.Count = 0 // Count should be 0 if DeviceIDs is set
			} else {
				jobLogger.Warn("Parsed empty list of GPU IDs from docker_gpus parameter", zap.String("param", gpuRequestParam))
			}
		} else if count, err := strconv.Atoi(gpuRequestValue); err == nil && count > 0 {
			// Attempt to parse as a number for count
			jobLogger.Info("Requesting specific count of GPUs for container", zap.Int("gpu_count", count))
			deviceRequest.Count = count
		} else {
			jobLogger.Warn("Invalid value for docker_gpus parameter, not requesting specific GPUs.", zap.String("param", gpuRequestParam))
			// Do not set DeviceRequests if param is invalid and not 'all'
			hostConfig.DeviceRequests = nil // Clear any previous default
		}

		// Only add the device request if it's meaningfully configured (either Count is non-zero or DeviceIDs are set)
		if deviceRequest.Count != 0 || len(deviceRequest.DeviceIDs) > 0 {
			hostConfig.DeviceRequests = []container.DeviceRequest{deviceRequest}
		} else if hostConfig.DeviceRequests != nil && !(deviceRequest.Count == 0 && len(deviceRequest.DeviceIDs) == 0 && gpuRequestValue != "all") {
			// This case might occur if parsing failed but it wasn't 'all', ensure no default is applied if not intended
			// If specifically Count=0 and no IDs, it implies no GPUs. But if parsing fails, it should not default to no GPUs unless specified.
			// For safety, if parsing fails and it's not 'all', don't set any GPU request, falling back to Docker default.
			// Let's refine: only set if Count is definitively positive, negative (-1 for all), or DeviceIDs are present.
			// The above conditions already ensure meaningful configuration.
		}

	} else {
		jobLogger.Info("No specific docker_gpus parameter found, or it's empty. Container will run with Docker default GPU access.")
		// No specific GPU request, so hostConfig.DeviceRequests remains nil or its default.
	}

	// --- Create Container ---
	containerName := fmt.Sprintf("dante-task-%s-%s", task.JobID, time.Now().Format("20060102150405"))
	jobLogger.Info("Creating Docker container", zap.String("name", containerName), zap.Any("config", containerConfig), zap.Any("host_config", hostConfig))
	// Assuming network.NetworkingConfig{} is intended if no specific network config.
	resp, err := de.cli.ContainerCreate(pullCtx, containerConfig, hostConfig, &network.NetworkingConfig{}, nil, containerName) // Use pullCtx timeout for create too
	if err != nil {
		jobLogger.Error("Failed to create Docker container", zap.Error(err))
		return ExecutionResult{Error: fmt.Errorf("failed to create container: %w", err), ExitCode: -1}
	}
	jobLogger.Info("Container created", zap.String("id", resp.ID))

	// Defer removal of the container
	defer func() {
		jobLogger.Info("Attempting to remove container", zap.String("id", resp.ID))
		removeCtx, removeCancel := context.WithTimeout(context.Background(), 30*time.Second) // Context for removal
		defer removeCancel()
		// Use container.RemoveOptions for ContainerRemove
		if err := de.cli.ContainerRemove(removeCtx, resp.ID, container.RemoveOptions{Force: true, RemoveVolumes: false}); err != nil {
			jobLogger.Error("Failed to remove container", zap.String("id", resp.ID), zap.Error(err))
		} else {
			jobLogger.Info("Container removed successfully", zap.String("id", resp.ID))
		}
	}()

	// Instrument for billing (after successful creation)
	if de.billingClient != nil && task.SelectedGPU != nil {
		errBill := de.billingClient.StartBilling(ctx, task.JobID, task.UserID, task.SelectedGPU.InstanceID, task.SelectedGPU.PricePerHour) // Use ctx from Execute
		if errBill != nil {
			jobLogger.Error("Failed to start billing for job", zap.String("job_id", task.JobID), zap.Error(errBill))
			// Decide if this is a fatal error for the task execution
		}
	} else if task.SelectedGPU == nil {
		jobLogger.Warn("SelectedGPU info not available in task, skipping StartBilling call.", zap.String("job_id", task.JobID))
	}

	// --- Start Container ---
	containerStartTime := time.Now() // Define containerStartTime before starting the container
	jobLogger.Info("Starting container", zap.String("id", resp.ID))
	// Use container.StartOptions for ContainerStart
	if err := de.cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		jobLogger.Error("Failed to start Docker container", zap.String("id", resp.ID), zap.Error(err))
		return ExecutionResult{Error: fmt.Errorf("failed to start container %s: %w", resp.ID, err), ExitCode: -1}
	}
	jobLogger.Info("Container started", zap.String("id", resp.ID))

	// Start billing monitoring if session ID is provided
	var billingCtx context.Context
	var billingCancel context.CancelFunc
	if sessionIDStr, ok := task.JobParams["session_id"].(string); ok && sessionIDStr != "" {
		if sessionID, err := uuid.Parse(sessionIDStr); err == nil && de.billingClient != nil {
			billingCtx, billingCancel = context.WithCancel(ctx)
			go func() {
				defer billingCancel()
				if err := de.billingClient.Monitor(billingCtx, sessionID, "gpu-0", 1*time.Minute); err != nil {
					jobLogger.Error("Billing monitoring failed", zap.Error(err))
				}
			}()
			jobLogger.Info("Started billing monitoring", zap.String("session_id", sessionID.String()))
		}
	}

	// --- Wait for Container Completion (with timeout) ---
	var waitCtx context.Context
	var waitCancel context.CancelFunc
	timeoutSeconds, hasTimeout := task.JobParams["timeout_seconds"].(float64)
	if hasTimeout && timeoutSeconds > 0 {
		jobLogger.Info("Container execution timeout set", zap.Float64("seconds", timeoutSeconds))
		waitCtx, waitCancel = context.WithTimeout(ctx, time.Duration(timeoutSeconds)*time.Second)
	} else {
		waitCtx, waitCancel = context.WithCancel(ctx) // No explicit timeout for wait, relies on overall ctx
	}
	defer waitCancel()

	statusCh, errCh := de.cli.ContainerWait(waitCtx, resp.ID, container.WaitConditionNotRunning)
	var statusCode int64 = -1 // Default to -1 if we can't get it
	var waitError error

	select {
	case err := <-errCh:
		if err != nil {
			jobLogger.Error("Error waiting for container", zap.String("id", resp.ID), zap.Error(err))
			waitError = fmt.Errorf("error waiting for container %s: %w", resp.ID, err)
			if waitCtx.Err() == context.DeadlineExceeded {
				jobLogger.Warn("Container execution timed out", zap.String("id", resp.ID))
				waitError = fmt.Errorf("container execution timed out: %w", waitCtx.Err())
				statusCode = 137 // Common exit code for timeout (SIGKILL)
				// Attempt to stop the container if it timed out
				stopCtx, stopCancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer stopCancel()
				// Use container.StopOptions{} for ContainerStop instead of nil
				if stopErr := de.cli.ContainerStop(stopCtx, resp.ID, container.StopOptions{}); stopErr != nil {
					jobLogger.Error("Failed to stop timed-out container", zap.String("id", resp.ID), zap.Error(stopErr))
				}
			}
		}
	case status := <-statusCh:
		statusCode = status.StatusCode
		if status.Error != nil {
			jobLogger.Warn("Container exited with an error status message", zap.String("id", resp.ID), zap.String("error_msg", status.Error.Message))
			// waitError might be set based on this, or from statusCode if non-zero
		}
		jobLogger.Info("Container finished with status code", zap.Int64("status_code", status.StatusCode), zap.String("id", resp.ID))
	}

	// --- Get Logs ---
	logCtx, logCancel := context.WithTimeout(context.Background(), 1*time.Minute) // Context for log retrieval
	defer logCancel()
	// Use container.LogsOptions for ContainerLogs
	logOptions := container.LogsOptions{ShowStdout: true, ShowStderr: true, Timestamps: false, Follow: false} // Ensure Timestamps and Follow are as intended
	logReader, errLog := de.cli.ContainerLogs(logCtx, resp.ID, logOptions)
	var logStdout, logStderr bytes.Buffer // Define these to store log output

	if errLog != nil {
		jobLogger.Error("Failed to get Docker container logs", zap.String("id", resp.ID), zap.Error(errLog))
		// Do not return immediately, proceed to get exit code and attempt cleanup
	} else {
		defer logReader.Close()
		// Demultiplex the TTY stream if Tty=false was used (which it is by default in containerConfig)
		_, errCP := stdcopy.StdCopy(&logStdout, &logStderr, logReader)
		if errCP != nil {
			jobLogger.Warn("Error demultiplexing Docker logs", zap.String("id", resp.ID), zap.Error(errCP))
		}
	}

	// --- Inspect Container for Final Exit Code if not already set by timeout logic ---
	if statusCode == -1 && waitError == nil { // Only inspect if we didn't get an exit code from Wait or timeout
		inspectCtx, inspectCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer inspectCancel()
		inspectResp, err := de.cli.ContainerInspect(inspectCtx, resp.ID)
		if err != nil {
			jobLogger.Error("Failed to inspect container for exit code", zap.String("id", resp.ID), zap.Error(err))
			if waitError == nil { // Prefer waitError if it exists
				waitError = fmt.Errorf("failed to inspect container %s: %w", resp.ID, err)
			}
		} else {
			statusCode = int64(inspectResp.State.ExitCode)
			jobLogger.Info("Container final state", zap.String("id", resp.ID), zap.Int("exit_code_inspect", inspectResp.State.ExitCode), zap.String("status", inspectResp.State.Status))
		}
	}

	finalResult := ExecutionResult{
		Stdout:   logStdout.String(),
		Stderr:   logStderr.String(),
		ExitCode: int(statusCode),
	}

	// Prioritize waitError (e.g., timeout) over non-zero exit code for the Error field
	if waitError != nil {
		finalResult.Error = waitError
	} else if statusCode != 0 {
		finalResult.Error = fmt.Errorf("container %s exited with code %d", resp.ID, statusCode)
	}

	// Stop billing
	if de.billingClient != nil {
		actualDuration := time.Since(containerStartTime)
		errBill := de.billingClient.StopBilling(ctx, task.JobID, task.UserID, actualDuration.Seconds()/3600) // Use ctx from Execute
		if errBill != nil {
			jobLogger.Error("Failed to stop billing for job", zap.String("job_id", task.JobID), zap.Error(errBill))
		}
		jobLogger.Info("Billing stopped for job", zap.String("job_id", task.JobID), zap.Float64("billed_duration_hours", actualDuration.Seconds()/3600))
	} else {
		jobLogger.Info("Billing client not available, skipping StopBilling call.", zap.String("job_id", task.JobID))
	}

	jobLogger.Info("Docker execution finished",
		zap.String("container_id", resp.ID),
		zap.Int("exit_code", finalResult.ExitCode),
		zap.Error(finalResult.Error),
	)

	return finalResult
}

// Helper function to check if a string is purely numeric
func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

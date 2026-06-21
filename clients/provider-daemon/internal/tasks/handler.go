package tasks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dante-gpu/dante-backend/provider-daemon/internal/config"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/executor"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/models"
	cli_models "github.com/dante-gpu/dante-backend/provider-daemon/internal/models" // Alias for clarity

	// "github.com/dante-gpu/dante-backend/provider-daemon/internal/reporting" // Not used yet
	"go.uber.org/zap"
)

// NatsStatusPublisher defines the interface for publishing task status updates.
// This was previously the concrete Nats client, now an interface for easier testing/DI.
// Note: This is the same as TaskResultReporter, consider consolidating if appropriate.
type NatsStatusPublisher interface {
	PublishStatus(statusUpdate *models.TaskStatusUpdate) error
}

// TaskResultReporter defines an interface for reporting task results.
// This allows decoupling the Handler from the concrete NATS client.
type TaskResultReporter interface {
	PublishStatus(statusUpdate *models.TaskStatusUpdate) error
}

// Handler processes incoming tasks and manages their execution.
// It now includes an activeJobs map for tracking.
type Handler struct {
	cfg            *config.Config
	logger         *zap.Logger
	reporter       TaskResultReporter // Interface for reporting results
	scriptExecutor executor.Executor
	dockerExecutor executor.Executor
	activeJobs     sync.Map // Stores *models.Task, keyed by JobID
}

// NewHandler creates a new task handler.
func NewHandler(cfg *config.Config, logger *zap.Logger, reporter TaskResultReporter, scriptExecutor executor.Executor, dockerExecutor executor.Executor) *Handler {
	return &Handler{
		cfg:            cfg,
		logger:         logger,
		reporter:       reporter,
		scriptExecutor: scriptExecutor,
		dockerExecutor: dockerExecutor,
		activeJobs:     sync.Map{}, // Initialize the map
	}
}

// SetReporter sets the reporter for the handler (used to break init cycle with NATS client).
func (h *Handler) SetReporter(reporter TaskResultReporter) {
	h.reporter = reporter
}

// HandleTask is called when a new task is received.
func (h *Handler) HandleTask(task *models.Task) error {
	h.logger.Info("Received task", zap.String("jobID", task.JobID), zap.String("jobName", task.JobName), zap.String("type", string(task.ExecutionType)))

	// Store the task as active
	h.activeJobs.Store(task.JobID, task)
	h.logger.Info("Task stored in active jobs map", zap.String("jobID", task.JobID))

	err := h.reportTaskStatus(task.JobID, models.StatusPreparing, "Task received by provider daemon", nil, "")
	if err != nil {
		h.logger.Error("Failed to report task received status", zap.Error(err), zap.String("jobID", task.JobID))
	}

	go h.runTask(task)
	return nil
}

func (h *Handler) runTask(task *models.Task) {
	workspacePath, err := h.prepareWorkspace(task.JobID)
	if err != nil {
		h.logger.Error("Failed to prepare workspace", zap.Error(err), zap.String("jobID", task.JobID))
		_ = h.reportTaskStatus(task.JobID, models.StatusFailed, fmt.Sprintf("Failed to prepare workspace: %v", err), nil, "")
		return
	}
	defer h.cleanupWorkspace(workspacePath, task.JobID)

	ctx, cancel := context.WithTimeout(context.Background(), h.cfg.RequestTimeout)
	defer cancel()

	_ = h.reportTaskStatus(task.JobID, models.StatusInProgress, "Task execution started", nil, "")

	var result executor.ExecutionResult
	switch task.ExecutionType {
	case models.ExecutionTypeScript:
		if h.scriptExecutor == nil {
			result.Error = fmt.Errorf("ScriptExecutor not available")
			result.Stderr = "ScriptExecutor not initialized on this provider daemon."
			result.ExitCode = -1
			break
		}
		result = h.scriptExecutor.Execute(ctx, task, workspacePath, h.logger)
	case models.ExecutionTypeDocker:
		if h.dockerExecutor == nil {
			result.Error = fmt.Errorf("DockerExecutor not available")
			result.Stderr = "DockerExecutor not initialized on this provider daemon, or failed to initialize."
			result.ExitCode = -1
			break
		}
		result = h.dockerExecutor.Execute(ctx, task, workspacePath, h.logger)
	default:
		result.Error = fmt.Errorf("unknown execution type: %s", task.ExecutionType)
		result.Stderr = fmt.Sprintf("Task specified an unsupported execution type: %s", task.ExecutionType)
		result.ExitCode = -1
	}

	finalStatus := models.StatusCompleted
	finalMessage := "Task completed successfully"
	executionLog := ""

	if result.Error != nil { // Error in execution setup or process
		finalStatus = models.StatusFailed
		finalMessage = fmt.Sprintf("Task execution failed: %v. Stderr: %s", result.Error, result.Stderr)
		executionLog = fmt.Sprintf("Stdout:\n%s\nStderr:\n%s", result.Stdout, result.Stderr)
	} else if result.ExitCode != 0 { // Error from within the script/container
		finalStatus = models.StatusFailed
		finalMessage = fmt.Sprintf("Task failed with exit code %d. Stderr: %s", result.ExitCode, result.Stderr)
		executionLog = fmt.Sprintf("Stdout:\n%s\nStderr:\n%s", result.Stdout, result.Stderr)
	}

	_ = h.reportTaskStatus(task.JobID, finalStatus, finalMessage, &result.ExitCode, executionLog)

	h.logger.Info("Task execution finished",
		zap.String("jobID", task.JobID),
		zap.Int("exitCode", result.ExitCode),
		zap.String("stdout", result.Stdout),
		zap.String("stderr", result.Stderr),
		zap.Error(result.Error),
	)
}

func (h *Handler) reportTaskStatus(jobID string, status models.JobStatus, message string, exitCode *int, execLog string) error {
	if h.reporter == nil {
		h.logger.Warn("TaskResultReporter is not set, cannot send status update for job.", zap.String("jobID", jobID))
		return fmt.Errorf("reporter not available")
	}

	statusUpdate := &models.TaskStatusUpdate{
		JobID:        jobID,
		ProviderID:   h.cfg.InstanceID,
		Status:       status,
		Message:      message,
		Timestamp:    time.Now(),
		ExitCode:     exitCode,
		ExecutionLog: execLog,
	}

	if status == models.StatusFailed || status == models.StatusCompleted || status == models.StatusCancelled {
		h.activeJobs.Delete(jobID)
		h.logger.Info("Task removed from active jobs map due to terminal status.", zap.String("jobID", jobID), zap.String("status", string(status)))
	}

	err := h.reporter.PublishStatus(statusUpdate)
	if err != nil {
		h.logger.Error("Failed to publish task status update", zap.Error(err), zap.String("jobID", jobID))
	}
	return err
}

func (h *Handler) prepareWorkspace(jobID string) (string, error) {
	workspacePath := filepath.Join(h.cfg.WorkspaceDir, jobID)
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return "", fmt.Errorf("failed to create workspace directory '%s': %w", workspacePath, err)
	}
	h.logger.Info("Workspace prepared", zap.String("jobID", jobID), zap.String("path", workspacePath))
	return workspacePath, nil
}

func (h *Handler) cleanupWorkspace(workspacePath string, jobID string) {
	if h.cfg.ExecutorConfig.Type == "docker" { // Or some other config for cleanup
		// Docker executor handles its own volume cleanup usually.
		// If scripts are run directly, cleanup might be needed.
		h.logger.Info("Skipping explicit workspace cleanup for Docker tasks, assuming volume management.", zap.String("jobID", jobID))
		return
	}

	if err := os.RemoveAll(workspacePath); err != nil {
		h.logger.Error("Failed to cleanup workspace", zap.Error(err), zap.String("jobID", jobID), zap.String("path", workspacePath))
	} else {
		h.logger.Info("Workspace cleaned up successfully", zap.String("jobID", jobID), zap.String("path", workspacePath))
	}
}

// GetActiveJobsForCLI retrieves the list of currently active jobs in a CLI-friendly format.
func (h *Handler) GetActiveJobsForCLI() []cli_models.CliLocalJob {
	var jobs []cli_models.CliLocalJob
	h.activeJobs.Range(func(key, value interface{}) bool {
		jobID, okKey := key.(string)
		task, okVal := value.(*models.Task)
		if !okKey || !okVal {
			h.logger.Error("Corrupted data in activeJobs map", zap.Any("key", key), zap.Any("value", value))
			return true // continue iteration
		}

		statusStr := "Processing/Unknown" // Default status if not explicitly tracked with more detail
		// To get a more accurate current status, the object stored in activeJobs would need to be updated
		// with status changes, or we would query another source. For now, this is a basic representation.

		cliJob := cli_models.CliLocalJob{
			ID:              jobID,
			Name:            task.JobName,
			Status:          statusStr,
			ProgressPercent: 0.0, // Placeholder, task model doesn't have live progress
			SubmittedAt:     task.DispatchedAt.Format(time.RFC3339),
		}
		jobs = append(jobs, cliJob)
		return true
	})
	return jobs
}

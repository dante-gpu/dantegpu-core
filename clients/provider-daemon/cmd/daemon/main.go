package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	// "strconv" // Will be needed for other CLI commands
	// "time" // Will be needed for other CLI commands

	"github.com/dante-gpu/dante-backend/provider-daemon/internal/billing"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/config"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/executor"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/gpu"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/models"
	cli_models "github.com/dante-gpu/dante-backend/provider-daemon/internal/models" // Alias for cli response models
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/nats"
	"github.com/dante-gpu/dante-backend/provider-daemon/internal/tasks"

	// "github.com/google/uuid" // Will be needed for other CLI commands or if instance ID generation is reactivated here
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Version   = "dev"     // Injected at build time
	BuildDate = "unknown" // Injected at build time
)

// CLI flags
var (
	configPath              = flag.String("config", filepath.Join("configs", "config.yaml"), "Path to the configuration file")
	getGpusJSON             = flag.Bool("get-gpus-json", false, "Detect GPUs and output as JSON, then exit")
	getSettingsJSON         = flag.Bool("get-settings-json", false, "Output current provider settings as JSON, then exit")
	updateSettingsJSON      = flag.String("update-settings-json", "", "Update provider settings from a JSON string, then exit")
	setGpuConfigJSON        = flag.Bool("set-gpu-config-json", false, "Set or update rental config for a specific GPU. Requires --gpu-id and at least one of --rate or --available.")
	gpuIDForConfig          = flag.String("gpu-id", "", "GPU ID for --set-gpu-config-json (e.g., nvidia-0)")
	rateForConfig           = flag.Float64("rate", -1.0, "Hourly rate in DGPU. A non-negative value updates the rate. For --set-gpu-config-json.")
	availableForConfig      = flag.String("available", "", "Availability for rent ('true' or 'false'). For --set-gpu-config-json.")
	getLocalJobsJSON        = flag.Bool("get-local-jobs-json", false, "Get current local jobs as JSON, then exit (currently placeholder).")
	getNetworkStatusJSON    = flag.Bool("get-network-status-json", false, "Get NATS connection status as JSON, then exit.")
	getFinancialSummaryJSON = flag.Bool("get-financial-summary-json", false, "Get financial summary as JSON, then exit (currently placeholder).")
	getSystemOverviewJSON   = flag.Bool("get-system-overview-json", false, "Get system overview (CPU, RAM, Disk, Uptime) as JSON, then exit.")
)

func main() {
	flag.Parse() // Parse all defined CLI flags

	tempLogger, _ := setupLogger("info")
	cfg, err := config.LoadConfig(*configPath, tempLogger)
	if err != nil {
		tempLogger.Fatal("Failed to load configuration", zap.Error(err), zap.String("path", *configPath))
	}

	logger, err := setupLogger(cfg.LogLevel)
	if err != nil {
		tempLogger.Fatal("Failed to setup logger with config level", zap.Error(err))
	}
	defer logger.Sync()
	cfg.Logger = logger

	// --- Handle CLI Commands ---
	if *getGpusJSON {
		handleGetGpusJSON(cfg, logger)
		return
	}
	if *getSettingsJSON {
		handleGetSettingsJSON(cfg, logger)
		return
	}
	if updateSettingsJSON != nil && *updateSettingsJSON != "" {
		handleUpdateSettingsJSON(cfg, *configPath, logger, *updateSettingsJSON)
		return
	}
	if *setGpuConfigJSON {
		handleSetGpuRentalConfig(cfg, *configPath, logger, *gpuIDForConfig, *rateForConfig, *availableForConfig)
		return
	}
	if *getLocalJobsJSON {
		handleGetLocalJobsJSON(cfg, logger)
		return
	}
	if *getNetworkStatusJSON {
		handleGetNetworkStatusJSON(cfg, logger)
		return
	}
	if *getFinancialSummaryJSON {
		handleGetFinancialSummaryJSON(cfg, logger)
		return
	}
	if *getSystemOverviewJSON {
		handleGetSystemOverviewJSON(cfg, logger)
		return
	}
	// Add other CLI command handlers here as they are implemented

	// --- Start Daemon Mode (if no CLI command was executed) ---
	logger.Info("Starting Dante GPU Provider Daemon",
		zap.String("version", Version),
		zap.String("buildDate", BuildDate),
		zap.String("instanceID", cfg.InstanceID),
		zap.String("logLevel", cfg.LogLevel),
	)

	// Initialize components for daemon mode
	gpuDetector := gpu.NewDetector(&cfg.GPUDetectorConfig, logger)
	// go gpuDetector.StartMonitoring() // Commented out: StartMonitoring not found on current Detector struct
	// defer gpuDetector.StopMonitoring() // Commented out: StopMonitoring not found on current Detector struct

	billingClient := billing.NewClient(&cfg.BillingClientConfig, logger)

	// Initialize Executors
	scriptExec := executor.NewScriptExecutor()
	dockerExec, err := executor.NewDockerExecutor(&cfg.ExecutorConfig, logger, billingClient, gpuDetector)
	if err != nil {
		// Log the error but continue, dockerExec will be nil.
		// The TaskHandler will then only be able to use scriptExec if a docker task comes.
		// Or, depending on policy, we might want to Fatal if Docker is the primary expected executor.
		logger.Warn("Failed to initialize Docker executor. Docker-based tasks may fail.", zap.Error(err))
		dockerExec = nil // Ensure it's nil if initialization failed
	}

	// Initialize Task Handler - pass nil for NatsStatusPublisher initially
	taskHandler := tasks.NewHandler(cfg, logger, nil, scriptExec, dockerExec)

	// Initialize NATS Client (depends on TaskHandler for message handling)
	natsClient, err := nats.NewClient(cfg, logger, taskHandler.HandleTask)
	if err != nil {
		logger.Fatal("Failed to initialize NATS client", zap.Error(err))
	}

	// Set the NATS client as the reporter for the task handler
	taskHandler.SetReporter(natsClient)

	if err := natsClient.StartListening(); err != nil {
		logger.Fatal("Failed to start NATS listener", zap.Error(err))
	}
	defer natsClient.Stop()

	logger.Info("Provider Daemon is running. Waiting for tasks...")

	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
	<-stopChan

	logger.Info("Shutting down Provider Daemon...")
}

func handleGetGpusJSON(cfg *config.Config, logger *zap.Logger) {
	logger.Info("CLI command: --get-gpus-json")
	gpuDetector := gpu.NewDetector(&cfg.GPUDetectorConfig, logger)

	detectedSystemGPUs, err := gpuDetector.DetectGPUsOnce()
	if err != nil {
		outputJSONError(fmt.Sprintf("Failed to detect GPUs: %v", err), os.Stderr, logger)
		os.Exit(1)
	}

	cliGPUs := make([]cli_models.CliGpuInfo, 0, len(detectedSystemGPUs))
	gpuRentalConfigMap := make(map[string]config.GpuRentalConfigEntry)
	for _, rentalConfig := range cfg.GpuRentalConfigs {
		gpuRentalConfigMap[rentalConfig.GpuID] = rentalConfig
	}

	for _, systemGPU := range detectedSystemGPUs { // Iterate over gpu.GPUInfo
		// Basic mapping from gpu.GPUInfo to cli_models.CliGpuInfo
		cliInfo := cli_models.CliGpuInfo{
			ID:                 systemGPU.ID,
			Name:               systemGPU.Name,
			Model:              systemGPU.Model,
			VRAMTotalMB:        uint32(systemGPU.VRAMTotal), // Cast from uint64
			VRAMFreeMB:         uint32(systemGPU.VRAMFree),  // Cast from uint64
			IsAvailableForRent: false,                       // Default, will be overridden by rental config if present
		}

		// Optional fields - assuming gpu.GPUInfo fields are 0/empty if not applicable/available
		// The cli_models uses pointers, so we only set them if data is meaningful.
		if systemGPU.Utilization > 0 { // Check if utilization is reported
			util := uint32(systemGPU.Utilization) // Already uint8, direct cast is fine
			cliInfo.UtilizationGPUPercent = &util
		}
		if systemGPU.Temperature > 0 { // Check if temperature is reported
			temp := uint32(systemGPU.Temperature) // Already uint8, direct cast is fine
			cliInfo.TemperatureC = &temp
		}
		if systemGPU.PowerDraw > 0 { // Check if power draw is reported
			power := systemGPU.PowerDraw // Already uint32
			cliInfo.PowerDrawW = &power
		}

		// Apply rental config from cfg.GpuRentalConfigs
		if rentalCfg, ok := gpuRentalConfigMap[systemGPU.ID]; ok {
			cliInfo.IsAvailableForRent = rentalCfg.IsAvailableForRent
			if rentalCfg.IsAvailableForRent && rentalCfg.CurrentHourlyRateDGPU > 0 {
				rate := rentalCfg.CurrentHourlyRateDGPU // This is float32 in GpuRentalConfigEntry
				cliInfo.CurrentHourlyRateDGPU = &rate
			}
		}
		cliGPUs = append(cliGPUs, cliInfo)
	}
	outputJSON(cliGPUs, logger)
}

func handleGetSettingsJSON(cfg *config.Config, logger *zap.Logger) {
	logger.Info("CLI command: --get-settings-json")
	settings := cli_models.CliProviderSettings{
		DefaultHourlyRateDGPU: float32(cfg.DefaultHourlyRateDGPU),
		PreferredCurrency:     cfg.PreferredCurrency,
		MinJobDurationMinutes: cfg.MinJobDurationMinutes,
		MaxConcurrentJobs:     cfg.MaxConcurrentJobs,
	}
	outputJSON(settings, logger)
}

func handleUpdateSettingsJSON(cfg *config.Config, configFilePath string, logger *zap.Logger, settingsJSON string) {
	logger.Info("CLI command: --update-settings-json", zap.String("json_input", settingsJSON))

	var newSettings cli_models.CliProviderSettings
	if err := json.Unmarshal([]byte(settingsJSON), &newSettings); err != nil {
		outputJSONError(fmt.Sprintf("Failed to unmarshal settings JSON: %v", err), os.Stderr, logger)
		return // os.Exit(1) is handled by outputJSONError
	}

	logger.Info("Successfully unmarshalled new settings", zap.Any("parsed_settings", newSettings))

	// Update the in-memory config (cfg is a pointer, so changes will persist for this run if not exiting)
	cfg.DefaultHourlyRateDGPU = float64(newSettings.DefaultHourlyRateDGPU) // CliProviderSettings uses float32, config.Config uses float64
	cfg.PreferredCurrency = newSettings.PreferredCurrency
	cfg.MinJobDurationMinutes = newSettings.MinJobDurationMinutes
	cfg.MaxConcurrentJobs = newSettings.MaxConcurrentJobs

	logger.Info("In-memory configuration updated with new settings.")

	// Persist the updated config to file
	if err := config.SaveConfig(cfg, configFilePath); err != nil {
		// Log the error, but also try to inform the CLI caller
		logger.Error("Failed to save updated configuration to file", zap.String("path", configFilePath), zap.Error(err))
		outputJSONError(fmt.Sprintf("Failed to save configuration: %v", err), os.Stderr, logger)
		return
	}

	logger.Info("Configuration successfully saved to file", zap.String("path", configFilePath))
	outputJSON(map[string]string{"status": "success", "message": "Settings updated and saved successfully."}, logger)
}

func handleSetGpuRentalConfig(cfg *config.Config, configFilePath string, logger *zap.Logger, gpuID string, newRate float64, newAvailabilityStr string) {
	logger.Info("CLI command: --set-gpu-config-json",
		zap.String("gpu_id", gpuID),
		zap.Float64("rate", newRate),
		zap.String("available_str", newAvailabilityStr),
	)

	if gpuID == "" {
		outputJSONError("--gpu-id is required for --set-gpu-config-json", os.Stderr, logger)
		return
	}

	// Check if at least one of rate or availability is provided for an update
	rateProvided := newRate >= 0
	availabilityProvided := newAvailabilityStr != ""

	if !rateProvided && !availabilityProvided {
		outputJSONError("At least one of --rate (non-negative) or --available ('true'/'false') must be provided to update GPU config.", os.Stderr, logger)
		return
	}

	var newAvailability bool
	var err error
	if availabilityProvided {
		newAvailability, err = strconv.ParseBool(strings.ToLower(newAvailabilityStr))
		if err != nil {
			outputJSONError(fmt.Sprintf("Invalid value for --available: %s. Must be 'true' or 'false'.", newAvailabilityStr), os.Stderr, logger)
			return
		}
	}

	found := false
	for i, rentalConfig := range cfg.GpuRentalConfigs {
		if rentalConfig.GpuID == gpuID {
			found = true
			updateMade := false
			if availabilityProvided {
				if cfg.GpuRentalConfigs[i].IsAvailableForRent != newAvailability {
					cfg.GpuRentalConfigs[i].IsAvailableForRent = newAvailability
					logger.Info("Updated GPU availability", zap.String("gpu_id", gpuID), zap.Bool("is_available", newAvailability))
					updateMade = true
				}
			}
			if rateProvided {
				if cfg.GpuRentalConfigs[i].CurrentHourlyRateDGPU != float32(newRate) { // config stores float32
					cfg.GpuRentalConfigs[i].CurrentHourlyRateDGPU = float32(newRate)
					logger.Info("Updated GPU hourly rate", zap.String("gpu_id", gpuID), zap.Float32("rate_dgpu", float32(newRate)))
					updateMade = true
				}
			}
			if !updateMade {
				logger.Info("No change in GPU config values, nothing to update.", zap.String("gpu_id", gpuID))
				// Still save, as the command was invoked. Or, could output a specific message.
			}
			break
		}
	}

	if !found {
		// GPU ID not found, create a new entry if settings are valid
		newEntry := config.GpuRentalConfigEntry{GpuID: gpuID}
		updateMade := false
		if availabilityProvided {
			newEntry.IsAvailableForRent = newAvailability
			updateMade = true
		}
		if rateProvided {
			newEntry.CurrentHourlyRateDGPU = float32(newRate)
			updateMade = true
		} else {
			// If creating a new entry and rate is not provided, should it default?
			// Current logic means it defaults to 0.0 for float32 if not set by rateProvided.
		}

		if !updateMade && !availabilityProvided && !rateProvided {
			// This case should be caught by the initial check, but as a safeguard:
			logger.Warn("Attempted to add new GPU config without providing rate or availability", zap.String("gpu_id", gpuID))
			outputJSONError("Cannot add new GPU config without providing --rate or --available.", os.Stderr, logger)
			return
		}
		cfg.GpuRentalConfigs = append(cfg.GpuRentalConfigs, newEntry)
		logger.Info("Added new GPU rental configuration", zap.String("gpu_id", gpuID), zap.Any("entry", newEntry))
	}

	if err := config.SaveConfig(cfg, configFilePath); err != nil {
		outputJSONError(fmt.Sprintf("Failed to save configuration for GPU %s: %v", gpuID, err), os.Stderr, logger)
		return
	}

	outputJSON(map[string]string{"status": "success", "message": fmt.Sprintf("GPU %s rental configuration updated and saved.", gpuID)}, logger)
}

func handleGetLocalJobsJSON(cfg *config.Config, logger *zap.Logger /* taskHandler *tasks.Handler */) {
	logger.Info("CLI command: --get-local-jobs-json")

	// The internal tasks.Handler is now capable of tracking active jobs in memory (see tasks.Handler.GetActiveJobsForCLI()).
	// However, this standalone CLI command does not have direct access to the memory of a running daemon instance.
	// To make this CLI command functional for querying a running daemon, an IPC mechanism would be required, such as:
	// 1. The daemon exposing a local HTTP endpoint for CLI queries.
	// 2. The daemon periodically writing job status to a file that this CLI can read.
	// 3. Using NATS for a request-reply pattern on a specific subject for daemon control/query.

	logger.Warn("handleGetLocalJobsJSON currently returns an empty list. Fetching live job data from a running daemon requires an IPC mechanism (e.g., local API endpoint or status file) which is not yet implemented for this specific CLI command path. The GUI will interact with the daemon directly.")
	// TODO: Implement an IPC mechanism for this CLI command to query active jobs from a running daemon instance,
	//       or clarify if this CLI is only for internal/GUI use where the taskHandler instance is directly available.

	localJobs := make([]cli_models.CliLocalJob, 0)
	// Example of how it *could* work if a taskHandler was available (e.g., if this was an internal call within the daemon):
	// if taskHandler != nil { // taskHandler would need to be passed in or initialized
	// 	 localJobs = taskHandler.GetActiveJobsForCLI()
	// } else {
	// 	 logger.Warn("Task handler not available to fetch local jobs for CLI command.")
	// }

	outputJSON(localJobs, logger)
}

func handleGetNetworkStatusJSON(cfg *config.Config, logger *zap.Logger) {
	logger.Info("CLI command: --get-network-status-json")

	status := cli_models.CliNetworkStatus{
		NatsServerURL: cfg.NatsConfig.URL, // Use NatsConfig directly
		NatsConnected: false,              // Default to false
	}

	// Attempt to initialize NATS client. Pass nil for TaskHandlerFunc as we are not processing tasks.
	natsClient, err := nats.NewClient(cfg, logger, nil) // Using cfg.NatsConfig is handled by NewClient
	if err != nil {
		status.LastNatsError = fmt.Sprintf("Failed to initialize NATS client: %v", err)
		logger.Error("Failed to initialize NATS client for status check", zap.Error(err))
		outputJSON(status, logger)
		return
	}
	defer natsClient.Stop() // Ensure client is stopped (which closes connection)

	status.NatsConnected = natsClient.IsConnected()
	status.NatsServerURL = natsClient.GetConnectionURL() // Might differ if connected to a different server in a cluster
	status.LastNatsError = natsClient.GetLastErrorStr()
	status.ActiveSubscriptions = natsClient.GetActiveSubscriptionCount() // Will be 0 as we don't call StartListening

	if status.NatsConnected {
		logger.Info("NATS connection successful for status check.", zap.String("url", status.NatsServerURL))
	} else {
		logger.Warn("NATS connection failed for status check.", zap.String("url", status.NatsServerURL), zap.String("error", status.LastNatsError))
	}

	outputJSON(status, logger)
}

func handleGetFinancialSummaryJSON(cfg *config.Config, logger *zap.Logger) {
	logger.Info("CLI command: --get-financial-summary-json")

	billingClient := billing.NewClient(&cfg.BillingClientConfig, logger)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	logger.Info("Fetching financial summary (using stubbed billing client methods).")
	financialDetails, err := billingClient.GetFinancialSummary(ctx, cfg.InstanceID)

	var summary cli_models.CliFinancialSummary

	if err != nil {
		logger.Error("Failed to get financial summary from billing client (stub)", zap.Error(err))
		// Populate with zeros or error indicators if preferred, but for now, empty/zero struct.
		summary = cli_models.CliFinancialSummary{
			CurrentBalanceDGPU: 0,
			TotalEarnedDGPU:    0,
			PendingPayoutDGPU:  0,
			LastPayoutAt:       nil,
		}
		// outputJSONError(fmt.Sprintf("Error fetching financial summary: %v", err), os.Stderr, logger) // Option: exit on error
		// return
	} else if financialDetails != nil {
		summary.CurrentBalanceDGPU = financialDetails.CurrentBalanceDGPU
		summary.TotalEarnedDGPU = financialDetails.TotalEarnedDGPU
		summary.PendingPayoutDGPU = financialDetails.PendingPayoutDGPU
		if financialDetails.LastPayoutAt != nil {
			lastPayoutAtStr := financialDetails.LastPayoutAt.Format(time.RFC3339)
			summary.LastPayoutAt = &lastPayoutAtStr
		}
		logger.Info("Successfully retrieved (stubbed) financial summary details.", zap.Any("details", financialDetails))
	} else {
		// Should not happen if err is nil, but as a fallback
		logger.Error("Financial summary details were nil without an error from billing client (stub)")
		summary = cli_models.CliFinancialSummary{} // Empty
	}

	// nowStr is no longer directly used here as LastPayoutAt comes from financialDetails
	// The CliFinancialSummary is now populated from the (stubbed) service call.

	outputJSON(summary, logger)
}

func handleGetSystemOverviewJSON(cfg *config.Config, logger *zap.Logger) {
	logger.Info("CLI command: --get-system-overview-json")

	overview := cli_models.CliSystemOverview{}

	// Get CPU usage
	cpuPercentages, err := cpu.Percent(0, false) // 0 for overall, false for non-per-CPU
	if err != nil {
		logger.Error("Failed to get CPU usage", zap.Error(err))
	} else if len(cpuPercentages) > 0 {
		overview.CpuUsagePercent = float32(cpuPercentages[0])
	}

	// Get Memory usage
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		logger.Error("Failed to get virtual memory stats", zap.Error(err))
	} else {
		overview.RamUsagePercent = float32(vmStat.UsedPercent)
	}

	// Get Disk usage for root path (or a configured path)
	diskPath := "/" // Or use cfg.WorkspaceDir and find its mount point if more specific logic is needed.
	// For simplicity, using root "/" or "C:" on Windows.
	if goos := runtime.GOOS; goos == "windows" {
		diskPath = "C:"
	}
	diskUsage, err := disk.Usage(diskPath)
	if err != nil {
		logger.Error("Failed to get disk usage stats", zap.String("path", diskPath), zap.Error(err))
	} else {
		overview.TotalDiskSpaceGB = diskUsage.Total / (1024 * 1024 * 1024)
		overview.FreeDiskSpaceGB = diskUsage.Free / (1024 * 1024 * 1024)
	}

	// Get Uptime
	upTime, err := host.Uptime()
	if err != nil {
		logger.Error("Failed to get host uptime", zap.Error(err))
	} else {
		overview.UptimeSeconds = upTime
	}

	logger.Info("System overview data collected", zap.Any("overview", overview))
	outputJSON(overview, logger)
}

func outputJSON(data interface{}, logger *zap.Logger) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		// Log internal error, then attempt to output a simple JSON error to stdout for Tauri
		logger.Error("Failed to marshal data to JSON for CLI output", zap.Error(err))
		fmt.Fprintf(os.Stdout, `{"error": "Failed to marshal data to JSON: %s"}\n`, err.Error())
		os.Exit(1)
	}
	fmt.Println(string(jsonData))
	os.Exit(0) // Ensure exit after successful JSON output for CLI mode
}

func outputJSONError(message string, writer *os.File, logger *zap.Logger) {
	logger.Error("CLI command error", zap.String("error_message", message))
	errorData := map[string]string{"error": message}
	jsonData, err := json.Marshal(errorData)
	if err != nil {
		fmt.Fprintf(writer, `{"error": "Failed to marshal error message to JSON. Original error: %s"}\n`, message)
		os.Exit(1)
		return
	}
	fmt.Fprintln(writer, string(jsonData))
	os.Exit(1) // Exit after error for CLI mode
}

func setupLogger(levelString string) (*zap.Logger, error) {
	var logLevel zapcore.Level
	switch levelString {
	case "debug":
		logLevel = zapcore.DebugLevel
	case "info":
		logLevel = zapcore.InfoLevel
	case "warn":
		logLevel = zapcore.WarnLevel
	case "error":
		logLevel = zapcore.ErrorLevel
	case "fatal":
		logLevel = zapcore.FatalLevel
	default:
		// Fallback for CLI mode before config is loaded, or if config is malformed
		fmt.Fprintf(os.Stderr, "Invalid log level specified: %s. Defaulting to info.\n", levelString)
		logLevel = zapcore.InfoLevel
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.TimeKey = "ts"
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder

	logDir := filepath.Join(".", "logs", "provider-daemon")
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		// Fallback to console-only logging if directory creation fails
		consoleCfg := zap.Config{
			Level:            zap.NewAtomicLevelAt(logLevel),
			Development:      false,  // Set to true for more human-readable console output if preferred
			Encoding:         "json", // Or "console" for human-readable
			EncoderConfig:    encoderConfig,
			OutputPaths:      []string{"stderr"},
			ErrorOutputPaths: []string{"stderr"},
		}
		logger, buildErr := consoleCfg.Build()
		if buildErr != nil {
			return nil, fmt.Errorf("failed to build console logger: %w", buildErr)
		}
		// Log the directory creation error using the console logger itself, if possible
		logger.Error("Failed to create log directory, logging to console only", zap.String("directory", logDir), zap.Error(err))
		return logger, nil
	}

	logFileName := filepath.Join(logDir, "daemon.log")

	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(zapcore.Lock(mustOpen(logFileName))),
		logLevel,
	)

	// Setup console core based on whether it's likely a CLI command execution or daemon mode
	// For CLI commands, we often want cleaner output on stdout for JSON, and logs to stderr/file.
	// For daemon mode, teeing to console can be verbose but useful for development.
	// The current tempLogger is console-only. If we are in CLI mode, we might prefer stderr for logs.

	// Let's use a simpler console encoder for daemon mode, or if explicit console logging is desired.
	// For CLI commands that output JSON to stdout, we should ensure logs go to stderr or file.
	consoleEncoderCfg := zap.NewDevelopmentEncoderConfig() // More readable for console
	consoleEncoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	consoleEncoderCfg.TimeKey = "ts"
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(consoleEncoderCfg),
		zapcore.AddSync(os.Stderr), // Log to stderr for console
		logLevel,
	)

	// For CLI commands, we might want to suppress regular consoleCore if stdout is for JSON.
	// However, error logs from outputJSONError should still go to stderr.
	// The logger passed to handleGetGpusJSON etc., will use this Tee.
	// If it's a CLI command that prints JSON to stdout, non-error logs should ideally go to file or stderr only.
	// This setup tees all logs (file + stderr console). Refinements could be made for cleaner CLI stdout.

	teeCore := zapcore.NewTee(fileCore, consoleCore)

	return zap.New(teeCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel)), nil
}

func mustOpen(filePath string) *os.File {
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Fallback if log file cannot be opened, print to stderr
		fmt.Fprintf(os.Stderr, "PANIC: Failed to open log file %s: %v\n", filePath, err)
		// Attempt to use a basic stderr logger if primary logging fails catastrophically
		cfg := zap.NewProductionConfig()
		cfg.OutputPaths = []string{"stderr"}
		logger, _ := cfg.Build()
		logger.Panic("Failed to open log file", zap.String("path", filePath), zap.Error(err))
		// Should not reach here due to panic
	}
	return file
}

// Ensure all model types are correctly imported and used to avoid "unused import" errors
// when some CLI command handlers are not yet fully implemented.
var _ models.Task           // From provider-daemon/internal/models
var _ cli_models.CliGpuInfo // From provider-daemon/internal/models (aliased)

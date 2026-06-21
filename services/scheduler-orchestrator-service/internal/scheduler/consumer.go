package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/billing"
	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/clients"
	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/config"
	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/models"
	"github.com/dante-gpu/dante-backend/scheduler-orchestrator-service/internal/store"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// JobConsumer handles receiving and processing job messages from NATS.
type JobConsumer struct {
	nc            *nats.Conn
	js            nats.JetStreamContext // JetStream context for durable subscriptions
	logger        *zap.Logger
	cfg           *config.Config
	prClient      *clients.Client                              // Client for provider-registry-service
	billingClient *billing.Client                              // Client for billing-payment-service
	jobStore      store.JobStore                               // Added JobStore dependency
	activeJobs    map[string]*models.InternalJobRepresentation // Map to track jobs being processed
	subscription  *nats.Subscription
	shutdownChan  chan struct{} // Channel to signal shutdown
}

// NewJobConsumer creates a new JobConsumer.
// It will also try to get a JetStream context.
func NewJobConsumer(nc *nats.Conn, cfg *config.Config, prClient *clients.Client, billingClient *billing.Client, logger *zap.Logger, js store.JobStore) (*JobConsumer, error) {
	logger.Info("Creating new JobConsumer")
	var jetStream nats.JetStreamContext // Renamed to avoid conflict with js param
	var err error
	if nc != nil {
		jetStream, err = nc.JetStream()
		if err != nil {
			logger.Error("Failed to get JetStream context for JobConsumer", zap.Error(err))
			// Decide if this is fatal or if we can proceed without JetStream initially (e.g. plain NATS sub)
			// For durable job queue, JetStream is preferred.
			return nil, fmt.Errorf("failed to get JetStream context: %w", err)
		}
		logger.Info("JetStream context obtained for JobConsumer")
	}

	return &JobConsumer{
		nc:            nc,
		js:            jetStream, // Use renamed variable
		logger:        logger,
		cfg:           cfg,
		prClient:      prClient,
		billingClient: billingClient,
		jobStore:      js, // Assign jobStore
		activeJobs:    make(map[string]*models.InternalJobRepresentation),
		shutdownChan:  make(chan struct{}),
	}, nil
}

// StartConsuming subscribes to the NATS subject for job submissions and starts processing messages.
// This uses a JetStream PullSubscription for more control over message fetching and ACKing.
func (jc *JobConsumer) StartConsuming() error {
	if jc.js == nil {
		jc.logger.Error("JetStream context is nil, cannot start consuming jobs. NATS connection might be down.")
		return fmt.Errorf("JetStream context not available for consuming jobs")
	}

	jc.logger.Info("JobConsumer starting to consume jobs",
		zap.String("subject", jc.cfg.NatsJobSubmissionSubject),
		zap.String("queue_group", jc.cfg.NatsJobQueueGroup),
	)

	// For JetStream, we will create a durable pull consumer.
	// The stream (e.g., DANTE_JOBS) is assumed to exist and be configured
	// to capture messages published to jc.cfg.NatsJobSubmissionSubject.
	durableName := jc.cfg.NatsJobQueueGroup + "_consumer" // Make durable name unique per queue group

	var err error
	jc.subscription, err = jc.js.PullSubscribe(
		jc.cfg.NatsJobSubmissionSubject, // Subject to subscribe to within the stream
		durableName,                     // Durable name for the consumer
		nats.AckWait(60*time.Second),    // Increased AckWait for processing time
		// nats.MaxDeliver(jc.cfg.NatsMaxRedelivery), // Configurable max redeliveries
	)

	if err != nil {
		jc.logger.Error("Failed to create JetStream pull subscription",
			zap.String("subject", jc.cfg.NatsJobSubmissionSubject),
			zap.String("durable_name", durableName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to create pull subscription: %w", err)
	}

	jc.logger.Info("Successfully subscribed to JetStream for jobs",
		zap.String("subject", jc.cfg.NatsJobSubmissionSubject),
		zap.String("durable_consumer", durableName),
	)

	// Start a goroutine to fetch messages
	go jc.fetchLoop()

	return nil
}

func (jc *JobConsumer) fetchLoop() {
	jc.logger.Info("Starting JetStream message fetch loop...")
	batchSize := 5 // Smaller batch for more responsive processing initially
	for {
		select {
		case <-jc.shutdownChan:
			jc.logger.Info("Shutting down JetStream message fetch loop...")
			return
		default:
			msgs, err := jc.subscription.Fetch(batchSize, nats.MaxWait(10*time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					// This is normal, just means no messages in this fetch window
					continue
				}
				// For other errors, log and potentially break or backoff
				jc.logger.Error("Error fetching messages from JetStream", zap.Error(err))
				// Check if subscription is still valid or if NATS connection is down
				if !jc.subscription.IsValid() || jc.nc.Status() != nats.CONNECTED {
					jc.logger.Error("NATS subscription or connection lost. Stopping fetch loop.")
					return // Exit loop if subscription is bad
				}
				time.Sleep(5 * time.Second) // Simple backoff
				continue
			}

			for _, msg := range msgs {
				jc.handleMessage(msg)
			}
		}
	}
}

// handleMessage processes a single NATS message containing a job.
func (jc *JobConsumer) handleMessage(msg *nats.Msg) {
	ctx := context.Background() // Create a context for database operations
	jc.logger.Debug("Received raw NATS message",
		zap.String("subject", msg.Subject),
		zap.Int("data_length", len(msg.Data)),
	)

	var job models.Job
	if err := json.Unmarshal(msg.Data, &job); err != nil {
		jc.logger.Error("Failed to unmarshal job data from NATS message",
			zap.Error(err),
			zap.ByteString("raw_data", msg.Data),
		)
		// Acknowledge poison pill messages to prevent redelivery loops
		if ackErr := msg.Ack(); ackErr != nil {
			jc.logger.Error("Failed to ACK unmarshalable (poison pill) message", zap.Error(ackErr))
		}
		return
	}

	// Check if job already exists (e.g. from a previous run if scheduler restarted)
	existingJobRecord, err := jc.jobStore.GetJob(ctx, job.ID)
	if err != nil {
		jc.logger.Error("Failed to check for existing job in store", zap.String("job_id", job.ID), zap.Error(err))
		// Decide how to handle this: Nak for retry, or attempt to process anyway?
		// For now, log and attempt to process. If SaveJob fails later, it will be handled.
	}

	var internalJob *models.InternalJobRepresentation
	if existingJobRecord != nil {
		internalJob = existingJobRecord.ToInternalJobRepresentation()
		jc.logger.Info("Processing existing job found in store", zap.String("job_id", internalJob.JobDetails.ID), zap.String("current_state", string(internalJob.State)))
		// If job is already in a terminal state (completed, failed with max attempts, cancelled), maybe just ACK and skip?
		if internalJob.State == models.JobStateCompleted || internalJob.State == models.JobStateCancelled {
			jc.logger.Info("Job already in terminal state, ACKing and skipping", zap.String("job_id", internalJob.JobDetails.ID), zap.String("state", string(internalJob.State)))
			if ackErr := msg.Ack(); ackErr != nil {
				jc.logger.Error("Failed to ACK message for already terminal job", zap.Error(ackErr))
			}
			return
		}
		// Reset attempts if we are reprocessing a job that was pending due to no providers, etc.
		// This depends on the desired retry strategy.
		// For now, we use the attempts from DB.
	} else {
		internalJob = models.NewInternalJob(job)
		jc.logger.Info("Successfully unmarshalled new job from NATS", zap.String("job_id", internalJob.JobDetails.ID), zap.String("job_name", internalJob.JobDetails.Name))
		jobRecordToSave := models.FromInternalJobRepresentation(internalJob)
		if err := jc.jobStore.SaveJob(ctx, jobRecordToSave); err != nil {
			jc.logger.Error("Failed to save new job to store", zap.String("job_id", internalJob.JobDetails.ID), zap.Error(err))
			// If we can't save, Nak the message to retry later.
			if nakErr := msg.NakWithDelay(10 * time.Second); nakErr != nil {
				jc.logger.Error("Failed to NAK message after failing to save new job", zap.Error(nakErr))
				_ = msg.Ack() // Prevent poison pill loop if NAK also fails
			}
			return
		}
		jc.logger.Info("New job saved to store", zap.String("job_id", internalJob.JobDetails.ID))
	}

	scheduled, scheduleErr := jc.scheduleJob(internalJob)

	// Update job state in DB based on scheduling outcome
	currentAttempts := internalJob.Attempts
	if !scheduled || scheduleErr != nil {
		currentAttempts++ // Increment attempt only if scheduling was tried and failed/not scheduled
	}

	finalLastError := ""
	if scheduleErr != nil {
		finalLastError = scheduleErr.Error()
	}

	// Persist the state after scheduling attempt (whether successful or not)
	if err := jc.jobStore.UpdateJobState(ctx, internalJob.JobDetails.ID, internalJob.State, internalJob.ProviderID, finalLastError, currentAttempts); err != nil {
		jc.logger.Error("Failed to update job state in store after scheduling attempt", zap.String("job_id", internalJob.JobDetails.ID), zap.Error(err))
		// This is tricky: if DB update fails, what to do with NATS message?
		// For now, we will proceed with NATS ack/nak based on scheduling outcome, as DB might recover.
	}

	if scheduleErr != nil {
		jc.logger.Error("Scheduling failed for job", zap.String("job_id", internalJob.JobDetails.ID), zap.Error(scheduleErr))
		if nakErr := msg.NakWithDelay(30 * time.Second); nakErr != nil {
			jc.logger.Error("Failed to NAK message after scheduling error", zap.String("job_id", internalJob.JobDetails.ID), zap.Error(nakErr))
			_ = msg.Ack()
		}
		return
	}

	if !scheduled {
		jc.logger.Warn("Job could not be scheduled at this time (no suitable providers)", zap.String("job_id", internalJob.JobDetails.ID))
		// State is already updated in internalJob by scheduleJob, and persisted above.
		if nakErr := msg.NakWithDelay(1 * time.Minute); nakErr != nil {
			jc.logger.Error("Failed to NAK message for job with no suitable providers", zap.String("job_id", internalJob.JobDetails.ID), zap.Error(nakErr))
			_ = msg.Ack()
		}
		return
	}

	if ackErr := msg.AckSync(); ackErr != nil {
		jc.logger.Error("Failed to ACK NATS message for successfully scheduled job", zap.String("job_id", internalJob.JobDetails.ID), zap.Error(ackErr))
	}
	jc.logger.Info("Finished processing and ACKed NATS message for job", zap.String("job_id", internalJob.JobDetails.ID))
}

// scheduleJob attempts to find a suitable provider and dispatch the job.
// Returns true if scheduled, false if no suitable provider is found currently.
func (jc *JobConsumer) scheduleJob(internalJob *models.InternalJobRepresentation) (bool, error) {
	job := internalJob.JobDetails
	originalState := internalJob.State // Keep original state in case we need to revert or for logging
	internalJob.State = models.JobStateSearching
	// UpdateJobState will handle UpdatedAt and Attempts, so no need to set them here directly for DB
	// We update internalJob for immediate consistency, DB update happens after this function returns (in handleMessage).

	jc.logger.Info("Attempting to schedule job", zap.String("job_id", job.ID), zap.String("gpu_type_req", job.GPUType), zap.Int("gpu_count_req", job.GPUCount))

	providers, err := jc.prClient.ListAvailableProviders()
	if err != nil {
		jc.logger.Error("Failed to list available providers during scheduling", zap.String("job_id", job.ID), zap.Error(err))
		internalJob.State = originalState // Revert state if PR call failed before any matching
		internalJob.LastError = fmt.Sprintf("provider registry query failed: %v", err)
		return false, fmt.Errorf("provider registry query failed: %w", err)
	}

	var suitableProvider *clients.Provider
	for _, p := range providers {
		provider := p // Create a new variable to take its address
		if provider.Status != clients.StatusIdle {
			jc.logger.Debug("Skipping provider: not idle", zap.String("provider_id", provider.ID.String()), zap.String("status", string(provider.Status)))
			continue
		}

		// GPU Type Matching (case-insensitive for flexibility)
		if job.GPUType != "" && !strings.EqualFold(jc.findProviderGPUType(&provider), job.GPUType) {
			// This simple check assumes if a GPUType is requested, the provider must primarily feature that type.
			// More complex logic can check individual GPUs within the provider.
			jc.logger.Debug("Skipping provider: GPUType mismatch",
				zap.String("provider_id", provider.ID.String()),
				zap.String("provider_gpu", jc.findProviderGPUType(&provider)),
				zap.String("job_requires", job.GPUType),
			)
			continue
		}

		// GPU Count Matching
		if job.GPUCount > 0 && len(provider.GPUs) < job.GPUCount {
			jc.logger.Debug("Skipping provider: insufficient GPU count",
				zap.String("provider_id", provider.ID.String()),
				zap.Int("provider_gpus", len(provider.GPUs)),
				zap.Int("job_requires", job.GPUCount),
			)
			continue
		}
		// TODO: Add more sophisticated matching: VRAM, specific GPU models within a provider if heterogeneous... -virjilakrum

		suitableProvider = &provider
		jc.logger.Info("Found suitable provider for job",
			zap.String("job_id", job.ID),
			zap.String("provider_id", suitableProvider.ID.String()),
			zap.String("provider_name", suitableProvider.Name),
		)
		break
	}

	if suitableProvider == nil {
		jc.logger.Info("No suitable provider found for job at this time", zap.String("job_id", job.ID))
		internalJob.State = models.JobStatePending // Set back to pending if no provider found
		internalJob.LastError = "No suitable provider found"
		// UpdateJobState in handleMessage will handle attempts and UpdatedAt
		return false, nil
	}

	// Validate billing requirements and start session
	if jc.billingClient != nil {
		// Validate user has sufficient balance
		gpuModel := jc.findProviderGPUType(suitableProvider)
		vramMB := uint64(8192)         // Default 8GB, should come from provider GPU specs
		estimatedPowerW := uint32(250) // Default 250W, should come from provider GPU specs

		if len(suitableProvider.GPUs) > 0 {
			// Use actual GPU specs if available
			gpu := suitableProvider.GPUs[0]
			if gpu.VRAM > 0 {
				vramMB = gpu.VRAM
			}
			// Power consumption would need to be added to GPUDetail struct
			// For now, use default values based on GPU model
			if strings.Contains(strings.ToLower(gpu.ModelName), "4090") {
				estimatedPowerW = 450
			} else if strings.Contains(strings.ToLower(gpu.ModelName), "a100") {
				estimatedPowerW = 400
			} else if strings.Contains(strings.ToLower(gpu.ModelName), "h100") {
				estimatedPowerW = 700
			}
		}

		err := jc.billingClient.ValidateJobRequirements(context.Background(), job.UserID, gpuModel, vramMB, estimatedPowerW)
		if err != nil {
			jc.logger.Error("Job billing validation failed", zap.String("job_id", job.ID), zap.Error(err))
			internalJob.State = models.JobStateFailed
			internalJob.LastError = fmt.Sprintf("Billing validation failed: %v", err)
			return false, fmt.Errorf("billing validation failed: %w", err)
		}

		// Start billing session
		sessionReq := &billing.SessionStartRequest{
			UserID:          job.UserID,
			ProviderID:      suitableProvider.ID,
			JobID:           &job.ID,
			GPUModel:        gpuModel,
			RequestedVRAM:   vramMB,
			EstimatedPowerW: estimatedPowerW,
		}

		sessionResp, err := jc.billingClient.StartSession(context.Background(), sessionReq)
		if err != nil {
			jc.logger.Error("Failed to start billing session", zap.String("job_id", job.ID), zap.Error(err))
			internalJob.State = models.JobStateFailed
			internalJob.LastError = fmt.Sprintf("Failed to start billing session: %v", err)
			return false, fmt.Errorf("billing session start failed: %w", err)
		}

		jc.logger.Info("Billing session started successfully",
			zap.String("job_id", job.ID),
			zap.String("session_id", sessionResp.Session.ID.String()),
			zap.String("estimated_hourly_cost", sessionResp.EstimatedHourlyCost.String()),
		)

		// Store session ID in task parameters for later reference
		jc.logger.Info("Session ID will be passed to task execution",
			zap.String("session_id", sessionResp.Session.ID.String()),
		)
	}

	// --- Task Creation & Dispatch ---
	task := models.NewTask(&job, suitableProvider.ID.String())
	taskJSON, err := json.Marshal(task)
	if err != nil {
		jc.logger.Error("Failed to marshal task for dispatch", zap.String("job_id", job.ID), zap.Error(err))
		internalJob.State = models.JobStateFailed
		internalJob.LastError = fmt.Sprintf("Failed to prepare task data: %v", err)
		return false, fmt.Errorf("task marshalling failed: %w", err)
	}

	dispatchSubject := fmt.Sprintf("%s.%s.%s", jc.cfg.NatsTaskDispatchSubjectPrefix, suitableProvider.ID.String(), job.ID)
	jc.logger.Info("Task created, attempting to dispatch to NATS",
		zap.String("job_id", job.ID),
		zap.String("provider_id", suitableProvider.ID.String()),
		zap.String("dispatch_subject", dispatchSubject),
		zap.ByteString("task_json", taskJSON), // Log actual task JSON for debugging
	)

	// Actually publish the task to NATS:
	if err := jc.nc.Publish(dispatchSubject, taskJSON); err != nil {
		jc.logger.Error("Failed to publish task to NATS",
			zap.String("job_id", job.ID),
			zap.String("provider_id", suitableProvider.ID.String()),
			zap.String("dispatch_subject", dispatchSubject),
			zap.Error(err),
		)
		// If publishing fails, we should probably not ACK the original job message.
		// Instead, let the original message be NAK'd so it can be retried later.
		// The job state should reflect that it's pending a retry due to dispatch failure.
		internalJob.State = models.JobStatePending // Or a more specific "dispatch_failed_retry" state
		internalJob.LastError = fmt.Sprintf("Failed to dispatch task to NATS: %v", err)
		return false, fmt.Errorf("NATS publish failed for task: %w", err) // This error will trigger a Nak in handleMessage
	}

	internalJob.State = models.JobStateDispatched // Or JobStateAssigning if there's another ack step from daemon
	internalJob.ProviderID = suitableProvider.ID.String()
	internalJob.Attempts++     // Increment attempts even for successful scheduling path (or only on retries?)
	internalJob.LastError = "" // Clear last error on successful dispatch

	jc.logger.Info("Job successfully scheduled and dispatched",
		zap.String("job_id", job.ID),
		zap.String("provider_id", internalJob.ProviderID),
	)
	return true, nil
}

// findProviderGPUType extracts the GPU model name from a provider
func (jc *JobConsumer) findProviderGPUType(provider *clients.Provider) string {
	if len(provider.GPUs) > 0 {
		return provider.GPUs[0].ModelName
	}
	return "unknown-gpu"
}

// Stop gracefully shuts down the JobConsumer.
func (jc *JobConsumer) Stop() {
	jc.logger.Info("Stopping JobConsumer...")
	close(jc.shutdownChan) // Signal the fetchLoop to stop

	if jc.subscription != nil {
		jc.logger.Info("Unsubscribing NATS job consumer...")
		// For Pull Subscriptions, Drain is often preferred to ensure all fetched messages are processed.
		// However, Unsubscribe is quicker if we are shutting down hard.
		// Let's try Drain first, then Unsubscribe as a fallback if Drain errors or times out.
		durableName := jc.cfg.NatsJobQueueGroup + "_consumer" // Re-construct or pass durableName if needed
		if err := jc.subscription.Drain(); err != nil {
			jc.logger.Error("Error draining NATS subscription", zap.Error(err),
				zap.String("durable_name", durableName),
			)
			// Fallback to Unsubscribe if Drain fails
			if unsubErr := jc.subscription.Unsubscribe(); unsubErr != nil {
				jc.logger.Error("Error unsubscribing NATS job consumer after drain failed", zap.Error(unsubErr))
			}
		} else {
			jc.logger.Info("NATS job consumer subscription drained successfully")
		}
	}
	// Note: Draining the subscription or connection is handled by the main NATS client close/drain.
	jc.logger.Info("JobConsumer stopped.")
}

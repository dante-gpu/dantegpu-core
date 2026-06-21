# Scheduler Orchestrator Service (Dante Backend)

This Go service is the brain of the Dante GPU platform. It is responsible for:

- Dequeuing job requests from NATS.
- Querying the `provider-registry-service` to find suitable and available GPU providers.
- Assigning tasks to provider daemons (e.g., via NATS or gRPC).
- Monitoring job progress and status.
- Making intelligent scheduling decisions based on job requirements, provider capabilities, and potentially costs or priorities.

## Responsibilities

- **Job Consumption:** Subscribe to NATS subjects where new job requests are published.
- **Provider Discovery:** Communicate with the `provider-registry-service` to get up-to-date information on available providers and their specifications.
- **Scheduling Logic:** Implement algorithms to match jobs to the most appropriate providers.
- **Task Dispatching:** Send commands/tasks to the chosen provider (likely via NATS messages or a direct gRPC call to a daemon on the provider machine).
- **Status Tracking:** Monitor the status of dispatched jobs and update their state.
- **API (Optional):** Potentially expose an API for administrative tasks or direct status queries.
- **Service Registration:** Register itself with Consul for discovery by other services.

## Tech Stack

- Go 1.22+
- NATS Client (for consuming jobs and potentially dispatching tasks)
- Consul API Client (for service discovery and registration)
- HTTP Client (to communicate with `provider-registry-service`)
- Zap (for structured logging)
- Chi (or similar, for any HTTP API endpoints like health checks)
- UUID (for internal tracking if needed)

## Setup

1.  **Install Go:** Ensure Go 1.22 or later is installed.
2.  **Build:**
    ```bash
    # Navigate to scheduler-orchestrator-service directory
    go build -o scheduler-orchestrator ./cmd/main.go 
    ```
3.  **Configuration:**
    - Configuration will be handled via `configs/config.yaml`.
    - A default configuration will be created if none exists.
    - Key settings include:
        - Port for its own health/API endpoint.
        - Log level.
        - Consul address and service registration details.
        - NATS address and relevant subjects/queues.
        - Address/endpoint for the `provider-registry-service`.
    - Example `configs/config.yaml`:
        ```yaml
        port: ":8003" # Default port for this service
        log_level: "info"
        consul_address: "localhost:8500"
        service_name: "scheduler-orchestrator"
        service_id_prefix: "scheduler-orchestrator-"
        
        nats_address: "nats://localhost:4222"
        nats_job_subject: "jobs.submitted" # Subject to subscribe to for new jobs
        nats_job_queue_group: "scheduler-group" # Queue group for load balancing job consumption

        provider_registry_url: "http://localhost:8002/providers" # URL for provider registry service
        
        # Add heartbeat/TTL settings later
        ```

## Running the Service

```bash
./scheduler-orchestrator
```

This will start the service. It will connect to NATS to listen for jobs, register with Consul, and be ready to query the Provider Registry.

## Key Operations (Conceptual)

1.  **Listen for Jobs:** Subscribe to `nats_job_subject`.
2.  **On Job Received:**
    a.  Acknowledge the message from NATS.
    b.  Parse job details.
    c.  Query `provider-registry-service` for suitable, available providers based on job requirements (e.g., GPU type, VRAM).
    d.  **Scheduling Algorithm:** Select the best provider.
        - Factors: availability, capability, load, priority, (future: cost, latency).
    e.  **Dispatch Task:** Send a message (e.g., via NATS to a provider-specific subject or gRPC call) to the selected provider's daemon, instructing it to start the job. Include job ID and parameters.
    f.  Update internal job status to "dispatched" or "running".
3.  **Monitor Job Status:** (Mechanism TBD - provider daemon could send updates via NATS, or this service could poll).
4.  **Handle Failures:** If a provider becomes unavailable or a job fails, potentially reschedule or mark job as failed. 
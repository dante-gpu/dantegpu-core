# Provider Registry Service (Dante Backend)

This Go service is responsible for managing the registry of active GPU providers within the Dante platform.

## Responsibilities

*   Registering new GPU providers connecting to the platform.
*   Storing provider details: Hardware specifications (GPU model, VRAM, drivers), unique ID, location, etc.
*   Tracking provider status (e.g., idle, busy, offline) via heartbeats or explicit updates.
*   Providing an API (likely REST or gRPC) for other services (like the Scheduler) to query available and suitable providers.
*   Registering itself with Consul for service discovery.

## Tech Stack

*   Go 1.22+
*   Chi (for HTTP routing)
*   Zap (for structured logging)
*   Consul API Client (for service registration/discovery)
*   PostgreSQL (planned, for persistent storage)
*   UUID (for provider IDs)

## Setup

1.  **Install Go:** Ensure Go 1.22 or later is installed.
2.  **Build:**
    ```bash
    # Navigate to provider-registry-service directory
    go build -o provider-registry ./cmd/main.go 
    ```
3.  **Configuration:**
    *   Configuration is handled via `configs/config.yaml`.
    *   A default configuration will be created if none exists.
    *   Key settings include port, log level, Consul address, and database connection details (TBD).
    *   Example `configs/config.yaml`:
        ```yaml
        port: ":8002" # Default port for this service
        log_level: "info"
        consul_address: "localhost:8500"
        database_url: "postgresql://user:password@localhost:5432/dante_registry?sslmode=disable" # Example
        service_name: "provider-registry"
        service_id_prefix: "provider-registry-" # For unique consul ID
        # Add heartbeat/TTL settings later
        ```

## Running the Service

```bash
./provider-registry
```

This will start the service, which should then register itself with the configured Consul agent.

## API Endpoints (Planned)

*   `POST /providers`: Register a new provider (provider daemon calls this).
*   `PUT /providers/{providerID}/status`: Update provider status (e.g., heartbeat).
*   `GET /providers`: List available providers (with filtering options, e.g., by status, GPU type).
*   `GET /providers/{providerID}`: Get details for a specific provider.
*   `DELETE /providers/{providerID}`: Deregister a provider.
*   `GET /health`: Health check endpoint. 
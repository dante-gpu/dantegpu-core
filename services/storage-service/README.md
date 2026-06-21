# Storage Service (Dante Backend)

This Go service will provide an abstraction layer for storing and retrieving various types of data for the Dante GPU platform, such as:

- User-uploaded datasets
- AI model files
- Job input/output artifacts
- Intermediate results

## Responsibilities

-   Expose a simple API (likely REST or gRPC) for other services to interact with storage.
-   Interface with an underlying object storage solution (e.g., MinIO, AWS S3, Google Cloud Storage).
-   Manage access control and permissions for stored objects.
-   Potentially handle data versioning and lifecycle management.
-   Register itself with Consul for service discovery.

## Tech Stack (Planned)

-   Go 1.22+
-   Chi (for HTTP routing if REST API is chosen)
-   gRPC (if gRPC API is chosen)
-   SDK for the chosen object storage (e.g., MinIO Go SDK, AWS Go SDK)
-   Zap (for structured logging)
-   Consul API Client (for service registration/discovery)

## Setup (Conceptual)

1.  **Install Go:** Ensure Go 1.22 or later is installed.
2.  **Build:**
    ```bash
    # Navigate to storage-service directory
    go build -o storage-service ./cmd/main.go 
    ```
3.  **Configuration:**
    -   Configuration will be handled via `configs/config.yaml`.
    -   Key settings will include:
        -   Port for its API endpoint.
        -   Log level.
        -   Consul address and service registration details.
        -   Credentials and endpoint for the chosen object storage backend (e.g., MinIO access key, secret key, endpoint).
        -   Default bucket names.

## Running the Service (Conceptual)

```bash
./storage-service
```

## API Endpoints (Planned - example REST)

-   `POST /upload/{bucket_name}`: Upload a file.
-   `GET /download/{bucket_name}/{object_key}`: Download a file.
-   `DELETE /delete/{bucket_name}/{object_key}`: Delete a file.
-   `GET /list/{bucket_name}`: List objects in a bucket.
-   `GET /health`: Health check endpoint. 
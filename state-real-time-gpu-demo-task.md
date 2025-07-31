# Task: Real-time GPU Sharing Demo

This file tracks the plan and progress for implementing a real-time GPU sharing demonstration on the DanteGPU platform.

## User Request

The user wants to see a real, not simulated, demonstration of two users sharing a GPU. One user acts as a provider, and the other as a renter who submits a job. All logs and outputs from the backend services involved in this process should be streamed and displayed in the terminal component of the existing web application (`provider-web-app`). The UI itself should not be changed. The communication should be in English.

## My Findings

- The target frontend application is `provider-web-app`, with the main component at `src/App.tsx`.
- The component currently displays static and randomly generated simulated logs.
- The project has a comprehensive microservices architecture, with services for API gateway, auth, provider registry, scheduling, etc.
- There is a `monitoring-logging-service` directory containing Loki, which suggests a centralized logging system is in place or intended.
- The services are managed via `docker-compose.yml`.

## Problem Solving Approach

My approach is to leverage the existing infrastructure as much as possible to create a robust and realistic demo.

1.  **Backend Log Streaming:**
    - I will not build a new logging system. Instead, I will use the existing Loki setup.
    - I will add a new WebSocket endpoint to the `api-gateway` service.
    - This endpoint will be responsible for querying logs from the Loki service and streaming them to connected clients in real-time. This keeps the frontend isolated from the internal monitoring infrastructure.

2.  **Frontend Log Consumption:**
    - I will modify `provider-web-app/src/App.tsx`.
    - I will remove the current log simulation logic (`setInterval`).
    - I will implement a WebSocket client to connect to the new endpoint on the `api-gateway`.
    - Received log messages will be appended to the `terminalLogs` state, which will update the UI automatically.

3.  **Demonstration Flow:**
    - I will first ensure all necessary services can be started via `docker-compose`.
    - I will then devise a sequence of `curl` commands or simple client scripts to simulate:
        a. A **Provider** coming online and registering their GPU.
        b. A **Renter** submitting a compute job.
    - The real-time logs from these actions, across all microservices, will be visible in the web UI's terminal.

## File Dependencies

-   `provider-web-app/src/App.tsx`: The main frontend component to be modified for log consumption.
-   `api-gateway/`: The service where the new WebSocket log streaming endpoint will be added. I'll likely need to add a new handler file in `internal/handlers/` and update the main router.
-   `monitoring-logging-service/docker-compose.yml`: To understand how Loki and other services are configured.
-   `docker-compose.yml`: To understand how to run the entire platform.

## Task Breakdown

-   [x] **DONE** Create this state file.
-   [x] **DONE** **Task 1: Backend - Create Log Streaming Endpoint.**
    -   [x] Analyze `api-gateway` structure to add a new WebSocket handler.
    -   [x] Research how to query logs from Loki via its API.
    -   [x] Implement the WebSocket handler in the `api-gateway` (in Go). This handler will connect to Loki and stream logs.
-   [x] **DONE** **Task 2: Frontend - Connect to Log Stream.**
    -   [x] Remove fake log generator in `provider-web-app/src/App.tsx`.
    -   [x] Add WebSocket client logic to connect to the backend.
    -   [x] Update the `terminalLogs` state with incoming real logs.
-   [ ] **IN PROGRESS** **Task 3: Integration and Demo Scripting.**
    -   [x] Verify the entire system runs with `docker-compose`.
    -   [ ] Create scripts/commands to simulate provider and renter actions.
    -   [ ] Test the end-to-end flow. 
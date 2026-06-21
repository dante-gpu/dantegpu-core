# Dante Provider GUI

This directory contains the source code for the Dante GPU Provider Desktop GUI application.
It is built using [Tauri](https://tauri.app/) (Rust backend) and [React](https://reactjs.org/) with [Vite](https://vitejs.dev/) (frontend).

## Purpose

The Provider GUI allows users who want to offer their GPU resources to the Dante GPU network to:
- Manage their provider status.
- Monitor their GPU(s) performance and health.
- Track earnings from GPU rentals.
- Configure settings related to the `provider-daemon`.
- View logs and status of the underlying `provider-daemon`.

## Development

### Prerequisites

- Node.js and npm (or yarn/pnpm)
- Rust and Cargo
- Tauri CLI prerequisites (see [Tauri documentation](https://tauri.app/v1/guides/getting-started/prerequisites))

### Setup

1.  Navigate to the `provider-gui` directory:
    ```bash
    cd provider-gui
    ```
2.  Install frontend dependencies:
    ```bash
    npm install
    # or yarn install / pnpm install
    ```

### Running in Development Mode

To run the application in development mode with hot-reloading:

```bash
npm run tauri dev
```

This command will:
1.  Start the Vite development server for the React frontend (usually on `http://localhost:5173`).
2.  Build and run the Tauri Rust backend, which will create a desktop window and load the Vite dev server URL.

### Building for Production

To build the application for production (creating distributable binaries):

```bash
npm run tauri build
```

This will generate platform-specific application bundles in `provider-gui/src-tauri/target/release/bundle/`.

## Interaction with `provider-daemon`

This GUI application is intended to interact with the main `provider-daemon` (the Go application).
This can be achieved in several ways, to be implemented:

1.  **Sidecar Pattern:** The `provider-daemon` binary can be bundled with the Tauri application. The GUI can then manage the lifecycle of the `provider-daemon` process (start, stop, monitor).
2.  **Local API:** The `provider-daemon` can expose a local HTTP API (or other IPC mechanism) that the Tauri Rust backend communicates with to fetch data and send commands.

Configuration for bundling the daemon as a sidecar is partially set up in `src-tauri/tauri.conf.json`.

## Project Structure

-   `provider-gui/`
    -   `src/`: Frontend React code (TypeScript, HTML, CSS).
        -   `main.tsx`: React entry point.
        -   `App.tsx`: Main React application component.
        -   `index.html`: HTML shell.
        -   `styles/`: CSS styles.
    -   `src-tauri/`: Tauri Rust backend code.
        -   `Cargo.toml`: Rust dependencies.
        -   `build.rs`: Tauri build script.
        -   `tauri.conf.json`: Tauri application configuration.
        -   `icons/`: Application icons (placeholder).
        -   `src/main.rs`: Rust entry point and backend logic.
        -   `sidecars/`: (Placeholder) For bundling the `provider-daemon` binary.
    -   `package.json`: Frontend Node.js dependencies and scripts.
    -   `README.md`: This file. 
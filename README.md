# Snapsec Agent

A cross-platform security agent written in Go.

## Features

- **Cross-platform Service**: Supports Linux, Windows, and macOS service registration.
- **Modular Architecture**: Easy to add new data gathering modules.
- **Automated Registration**: Registers itself with the backend on first run.
- **Periodic Reporting**: Sends asset data and heartbeats every 5 seconds (configurable).

## Project Structure

- `cmd/agent/`: Entry point and CLI handling.
- `internal/agent/`: Core agent logic and loops.
- `internal/config/`: Configuration loading and OS-specific paths.
- `internal/modules/`: Data gathering modules (host, network, etc.).
- `internal/service/`: Service management wrapper.
- `pkg/api/`: Backend API client.

## Installation

### 1. Build the agent
```bash
go build -o bin/snapsec-agent ./cmd/agent
```

### 2. Configure
Copy the example config to the default location:
- Linux/Mac: `/etc/snapsec-agent.yaml`
- Windows: `C:\ProgramData\snapsec-agent\config.yaml`

```bash
cp snapsec-agent.yaml.example /etc/snapsec-agent.yaml
```

### 3. Install and Register
Simply run the agent. It will automatically detect it's not installed, register with the backend, and install itself as a system service.

```bash
sudo ./snapsec-agent
```

## Backend-Driven Installation

For automated deployments, use the provided scripts in the `scripts/` directory. These are designed to be served by your backend with template variables populated.

### Linux/macOS
```bash
curl -L https://your-backend.com/install.sh?token=XYZ | bash
```

### Windows (PowerShell)
```powershell
iwr https://your-backend.com/install.ps1?token=XYZ | iex
```

The scripts will:
1. Create the necessary configuration directories.
2. Write the `snapsec-agent.yaml` with the backend URL and API Key.
3. Download the correct binary for the OS/Architecture.
4. Execute the binary to trigger auto-registration and service installation.

## Development

### Adding a new module
1. Create a new file in `internal/modules/yourmodule/yourmodule.go`.
2. Implement the `Module` interface:
   ```go
   type Module interface {
       Name() string
       Gather() (interface{}, error)
   }
   ```
3. Register the module in `internal/agent/agent.go` in the `NewAgent` function.

## API Endpoints
The agent expects the following endpoints on the backend:
- `POST /register`: Initial registration with host details.
- `POST /heartbeat`: Periodic heartbeat.
- `POST /assets`: Periodic asset data reporting.

## Scanning Configuration

The agent includes a highly optimized Nuclei scanning engine designed to run silently in the background without disrupting the host machine's performance.

### Performance Throttling
To prevent CPU spikes and memory exhaustion (OOM), the Nuclei engine is hardcoded with the following "sweet spot" configuration for endpoint agents:
- **`-c 8` (Concurrency)**: Runs exactly 8 vulnerability templates simultaneously. This bounds memory strictly under 100MB.
- **`-bs 8` (Batch Size)**: Processes 8 files at a time per template. This prevents the OS from running out of file descriptors.
- **`-rl 100` (Rate Limiting)**: Caps the scanner to 100 file reads/regex evaluations per second. This serves as a natural throttle to ensure the user's CPU never spikes to 100%.
- **No Timeout**: The 2-hour timeout limit has been removed, ensuring full filesystem scans can complete even if heavily throttled.

### Resuming Interrupted Scans
If the agent is forcefully killed, crashes, or the host machine reboots during a scan, you can resume exactly where it left off. Nuclei automatically generates a state file when interrupted.

To resume an interrupted scan:
```bash
sudo ./snapsec-agent scan --tool=nuclei --target=/ --resume=path/to/resume.cfg
```

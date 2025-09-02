# KrakenHashes Agent

The KrakenHashes Agent is responsible for hardware monitoring and management.

## Dependencies

The agent requires certain system dependencies based on your hardware configuration. The agent will automatically detect your hardware and provide installation instructions for missing dependencies.

### Common Dependencies

These are required for basic functionality:

- `pciutils` - PCI device detection
- `util-linux` - CPU information
- `lm-sensors` - Temperature monitoring (optional)

### GPU-Specific Dependencies

The agent will only require dependencies for the GPU vendors present in your system:

#### NVIDIA GPUs
- `nvidia-driver` - NVIDIA driver and tools
- Required for NVIDIA GPU detection and monitoring

#### AMD GPUs
- `rocm-smi` - ROCm tools for AMD GPUs
- `amdgpu-pro` - AMDGPU-PRO driver tools
- At least one of these is required for AMD GPU detection and monitoring

#### Intel GPUs
- `intel-gpu-tools` - Intel GPU tools
- Required for Intel GPU detection and monitoring

### Installation

The agent will check for required dependencies on startup and provide installation instructions if needed. You can also manually check dependencies:

```bash
# Install common dependencies
sudo apt-get install pciutils util-linux lm-sensors

# For NVIDIA GPUs
sudo apt-get install nvidia-driver

# For AMD GPUs (choose one)
sudo apt-get install rocm-smi
# or
sudo apt-get install amdgpu-pro

# For Intel GPUs
sudo apt-get install intel-gpu-tools
```

### Virtualization Support

The agent supports virtualized environments with proper PCI passthrough configuration. Ensure your virtualization platform is configured to expose GPU devices directly to the virtual machine.

## Usage

### First-Time Setup

1. **Get a claim code** from the KrakenHashes admin interface
2. **Run the agent** with registration parameters:

```bash
# Basic registration (uses current directory for config and data)
./agent -host your-server:31337 -claim YOUR_CLAIM_CODE

# Registration with custom directories
./agent \
  -host your-server:31337 \
  -claim YOUR_CLAIM_CODE \
  -config-dir /opt/krakenhashes/config \
  -data-dir /opt/krakenhashes/data \
  -debug
```

The agent will:
- Register with the backend server
- Download necessary certificates
- Create a `.env` configuration file
- Set up data directories for binaries, wordlists, rules, and hashlists

### Subsequent Runs

After initial setup, simply run:

```bash
./agent
```

The agent will use the `.env` file created during registration.

### Command-Line Options

```
  -host string           Backend server host (e.g., localhost:31337)
  -tls                   Use TLS for secure communication (default: true)
  -interface string      Network interface to listen on (optional)
  -heartbeat int         Heartbeat interval in seconds (default: 5)
  -claim string          Agent claim code (required only for first-time registration)
  -debug                 Enable debug logging (default: false)
  -hashcat-params string Extra parameters to pass to hashcat (e.g., '-O -w 3')
  -config-dir string     Configuration directory for certificates and credentials
  -data-dir string       Data directory for binaries, wordlists, rules, and hashlists
  -help                  Show help
```

### Configuration

The agent creates and maintains a `.env` file with all configuration:

- **Server settings**: Backend host, port, TLS configuration
- **Directories**: Paths for configuration and data storage
- **WebSocket timing**: Connection parameters for agent-backend communication
- **Hashcat options**: Additional parameters for hashcat execution
- **Logging**: Debug mode and log levels

**Note**: The agent reads configuration from the `.env` file, not from system environment variables. This prevents conflicts when running the agent and backend on the same host.

### Running Multiple Agents

To run multiple agents on the same host:

1. Use different directories for each agent:
```bash
# Agent 1
./agent -host server:31337 -claim CODE1 \
  -config-dir /opt/agent1/config \
  -data-dir /opt/agent1/data

# Agent 2 (in a different directory)
cd /opt/agent2
./agent -host server:31337 -claim CODE2 \
  -config-dir /opt/agent2/config \
  -data-dir /opt/agent2/data
```

2. Each agent will maintain its own `.env` file and configuration

## Development

[Add development instructions here]
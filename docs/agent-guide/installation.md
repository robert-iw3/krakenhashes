# Agent Installation

## Overview

This guide covers installing and setting up KrakenHashes agents on various platforms.

## System Requirements

### Minimum Requirements
- 4GB RAM
- 10GB free disk space (You need enough disk space to cover all wordlists)
- Linux (Ubuntu 20.04+, Debian 11+, RHEL 8+, or similar)
- Network connectivity to backend server

### GPU Requirements (Optional but Recommended)
- NVIDIA: CUDA 11.0+ compatible GPU with 4GB+ VRAM
- AMD: ROCm compatible GPU or OpenCL support
- Intel: OpenCL compatible GPU

## Installation Methods

### Method 1: Binary Installation (Recommended)

1. **Download the agent binary**:
   ```bash
   wget https://github.com/ZerkerEOD/krakenhashes/releases/latest/download/krakenhashes-agent-linux-amd64
   chmod +x krakenhashes-agent-linux-amd64
   mv krakenhashes-agent-linux-amd64 /usr/local/bin/krakenhashes-agent
   ```

2. **Create system user**:
   ```bash
   sudo useradd -r -s /bin/false krakenhashes
   sudo mkdir -p /var/lib/krakenhashes/agent
   sudo chown krakenhashes:krakenhashes /var/lib/krakenhashes/agent
   ```

3. **Install as systemd service**:
   ```bash
   sudo tee /etc/systemd/system/krakenhashes-agent.service > /dev/null <<EOF
   [Unit]
   Description=KrakenHashes Agent
   After=network.target

   [Service]
   Type=simple
   User=krakenhashes
   Group=krakenhashes
   ExecStart=/usr/local/bin/krakenhashes-agent
   Restart=always
   RestartSec=10
   Environment="KH_DATA_DIR=/var/lib/krakenhashes/agent"

   [Install]
   WantedBy=multi-user.target
   EOF

   sudo systemctl daemon-reload
   sudo systemctl enable krakenhashes-agent
   ```

### Method 2: Docker Installation

1. **Pull the Docker image**:
   ```bash
   docker pull ghcr.io/zerkereod/krakenhashes-agent:latest
   ```

2. **Run with Docker Compose**:
   ```yaml
   version: '3.8'
   services:
     agent:
       image: ghcr.io/zerkereod/krakenhashes-agent:latest
       container_name: krakenhashes-agent
       volumes:
         - ./data:/data
         - ./config:/config
       environment:
         - KH_CONFIG_FILE=/config/agent.yaml
       restart: unless-stopped
       # For GPU support
       deploy:
         resources:
           reservations:
             devices:
               - driver: nvidia
                 count: all
                 capabilities: [gpu]
   ```

## Initial Configuration

The agent supports two configuration methods:

### Method 1: Automatic Configuration (Recommended)

The agent automatically creates a `.env` configuration file on first run. You can specify custom directories using command-line flags during the initial registration:

```bash
# The agent will create a .env file with your configuration
krakenhashes-agent \
  -host your-server:31337 \
  -claim YOUR_CLAIM_CODE \
  -config-dir /var/lib/krakenhashes/agent/config \
  -data-dir /var/lib/krakenhashes/agent/data
```

### Method 2: Manual .env File Creation

You can manually create a `.env` file before running the agent:

1. Create a `.env` file in the agent's working directory:

```bash
# KrakenHashes Agent Configuration
# Generated on: 2025-09-05T12:05:32+01:00

# Server Configuration
KH_HOST=your-server.example.com  # Backend server hostname
KH_PORT=31337                    # Backend server port
USE_TLS=true                     # Use TLS for secure communication
LISTEN_INTERFACE=                # Network interface to bind to
HEARTBEAT_INTERVAL=5             # Heartbeat interval in seconds

# Agent Configuration
KH_CLAIM_CODE=YOUR-CLAIM-CODE-HERE  # Your claim code from Admin UI

# Directory Configuration
KH_CONFIG_DIR=./config  # Configuration directory
KH_DATA_DIR=./data      # Data directory

# WebSocket Timing Configuration
KH_WRITE_WAIT=10s   # Timeout for writing messages
KH_PONG_WAIT=60s    # Timeout for receiving pong
KH_PING_PERIOD=54s  # Ping interval

# File Transfer Configuration
KH_MAX_CONCURRENT_DOWNLOADS=3  # Max concurrent downloads
KH_DOWNLOAD_TIMEOUT=1h         # Download timeout

# Hashcat Configuration
HASHCAT_EXTRA_PARAMS=  # Extra hashcat parameters (see note below)

# Logging Configuration
DEBUG=false            # Enable debug logging
LOG_LEVEL=INFO        # Log level
```

2. Replace the placeholder values with your actual configuration
3. Run the agent: `./krakenhashes-agent`

**Important Note on HASHCAT_EXTRA_PARAMS:**
- Parameters configured via the frontend (per-agent settings) take precedence
- The .env file parameters are only used as a fallback
- Best practice: Configure parameters via the frontend UI for centralized management

### Post-Configuration

After the first run, the agent will use the `.env` file for all configuration. You can edit this file manually if needed:

```bash
# View/edit the generated configuration
cat .env
nano .env  # or your preferred editor

# Note: After successful registration, the KH_CLAIM_CODE will be automatically commented out
```

## Agent Registration

1. **Generate a claim code** in the Admin UI:
   - Navigate to Agents → Manage Vouchers
   - Click "Create Voucher"
   - Copy the generated code

2. **Register the agent**:
   ```bash
   # Basic registration (uses default directories in current working directory)
   sudo -u krakenhashes krakenhashes-agent \
     -host your-server:31337 \
     -claim YOUR_CLAIM_CODE
   
   # Registration with custom directories (Should only be used when testing)
   sudo -u krakenhashes krakenhashes-agent \
     -host your-server:31337 \
     -claim YOUR_CLAIM_CODE \
     -config-dir /var/lib/krakenhashes/agent/config \
     -data-dir /var/lib/krakenhashes/agent/data \
     -debug
   ```
   
   **Note**: The agent will create a `.env` file on first run with all configuration. Subsequent runs will use this file automatically.

3. **Start the agent** (for systemd installations):
   ```bash
   sudo systemctl start krakenhashes-agent
   sudo systemctl status krakenhashes-agent
   ```
   
   Or run directly (uses `.env` file):
   ```bash
   sudo -u krakenhashes krakenhashes-agent
   ```

## GPU Driver Installation

### NVIDIA GPUs

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y nvidia-driver-525 nvidia-cuda-toolkit

# RHEL/CentOS/Rocky
sudo dnf config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel8/x86_64/cuda-rhel8.repo
sudo dnf install -y nvidia-driver cuda
```

### AMD GPUs

```bash
# Install ROCm
wget -q -O - https://repo.radeon.com/rocm/rocm.gpg.key | sudo apt-key add -
echo 'deb [arch=amd64] https://repo.radeon.com/rocm/apt/debian/ ubuntu main' | sudo tee /etc/apt/sources.list.d/rocm.list
sudo apt update
sudo apt install rocm-dev
```

## Verification

1. **Check agent status**:
   ```bash
   sudo systemctl status krakenhashes-agent
   ```

2. **View logs**:
   ```bash
   sudo journalctl -u krakenhashes-agent -f
   ```

3. **Verify in Web UI**:
   - Navigate to Agents section
   - Confirm agent appears as "Online"
   - Check detected devices

## Next Steps

- [Configure the agent](configuration.md)
- [Set up scheduling](scheduling.md)
- [Learn about file synchronization](file-sync.md)
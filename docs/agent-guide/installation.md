# Agent Installation

## Overview

This guide covers installing and setting up KrakenHashes agents on various platforms.

## System Requirements

### Minimum Requirements
- 4GB RAM
- 10GB free disk space
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

1. **Create configuration file**:
   ```bash
   sudo mkdir -p /etc/krakenhashes
   sudo tee /etc/krakenhashes/agent.yaml > /dev/null <<EOF
   # KrakenHashes Agent Configuration
   server:
     url: https://your-server:31337
     # API key will be added during registration
   
   data:
     directory: /var/lib/krakenhashes/agent
   
   logging:
     level: info
     file: /var/log/krakenhashes/agent.log
   EOF
   ```

2. **Set permissions**:
   ```bash
   sudo chown krakenhashes:krakenhashes /etc/krakenhashes/agent.yaml
   sudo chmod 600 /etc/krakenhashes/agent.yaml
   ```

## Agent Registration

1. **Generate a claim code** in the Admin UI:
   - Navigate to Agents → Manage Vouchers
   - Click "Create Voucher"
   - Copy the generated code

2. **Register the agent**:
   ```bash
   sudo -u krakenhashes krakenhashes-agent register \
     --code YOUR_CLAIM_CODE \
     --config /etc/krakenhashes/agent.yaml
   ```

3. **Start the agent**:
   ```bash
   sudo systemctl start krakenhashes-agent
   sudo systemctl status krakenhashes-agent
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
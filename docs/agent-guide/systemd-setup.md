# Systemd Service Setup

## Overview

I have yet to test systemd. Please open any issues you may have so that I can address them as needed.

This guide explains how to set up the KrakenHashes agent as a systemd service for automatic startup and management. There are two approaches: user services (recommended for personal use) and system services (for production servers).

## Quick Decision Guide

- **User Service**: If you're running the agent on your personal machine or don't have root access
- **System Service**: If you're setting up on a production server with multiple users or need the agent to start before login

## User Service Setup (No Root Required)

User systemd services run under your user account and don't require sudo privileges. This is the recommended approach for most users.

### 1. Create the Service Directory

```bash
mkdir -p ~/.config/systemd/user/
```

### 2. Create the Service File

Create `~/.config/systemd/user/krakenhashes-agent.service`:

```ini
[Unit]
Description=KrakenHashes Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
# Working directory where the agent and .env file are located
WorkingDirectory=%h/krakenhashes-agent
# Path to the agent executable
ExecStart=%h/krakenhashes-agent/krakenhashes-agent
Restart=on-failure
RestartSec=10
# Optional: Set resource limits
# MemoryLimit=4G
# CPUQuota=80%

# Environment variables (optional)
# Environment="DEBUG=true"
# Or use an environment file:
# EnvironmentFile=%h/krakenhashes-agent/.env.systemd

[Install]
WantedBy=default.target
```

**Note**: `%h` is automatically replaced with your home directory (e.g., `/home/username`)

### 3. Enable and Start the Service

```bash
# Reload systemd to recognize the new service
systemctl --user daemon-reload

# Enable the service to start on boot
systemctl --user enable krakenhashes-agent

# Start the service now
systemctl --user start krakenhashes-agent

# Check status
systemctl --user status krakenhashes-agent
```

### 4. Enable Lingering (Optional)

To start the service at boot (before you log in):

```bash
sudo loginctl enable-linger $USER
```

### Managing User Services

```bash
# View logs
journalctl --user -u krakenhashes-agent -f

# Stop the service
systemctl --user stop krakenhashes-agent

# Restart the service
systemctl --user restart krakenhashes-agent

# Disable auto-start
systemctl --user disable krakenhashes-agent
```

## System Service Setup (Advanced - Requires Root)

System services run at the system level and require root/sudo access. This approach is typically used for production servers where the agent needs to run before any user logs in.

**Note**: Most users should use the User Service setup above. Only use system services if you specifically need the agent to run at boot before login.

### 1. Create a Dedicated User (Optional but Recommended)

```bash
sudo useradd -r -s /bin/false -d /var/lib/krakenhashes -m krakenhashes
```

### 2. Install the Agent

```bash
# Create directory structure
sudo mkdir -p /opt/krakenhashes-agent
sudo chown krakenhashes:krakenhashes /opt/krakenhashes-agent

# Copy agent binary from your download location
sudo cp ~/krakenhashes-agent/krakenhashes-agent /opt/krakenhashes-agent/
sudo chown krakenhashes:krakenhashes /opt/krakenhashes-agent/krakenhashes-agent
sudo chmod +x /opt/krakenhashes-agent/krakenhashes-agent
```

### 3. Create Configuration

Create `/opt/krakenhashes-agent/.env` with your configuration:

```bash
sudo -u krakenhashes tee /opt/krakenhashes-agent/.env > /dev/null <<EOF
# Agent configuration
KH_HOST=your-server.example.com
KH_PORT=31337
USE_TLS=true
KH_CLAIM_CODE=YOUR-CLAIM-CODE-HERE
KH_CONFIG_DIR=/opt/krakenhashes-agent/config
KH_DATA_DIR=/opt/krakenhashes-agent/data
# Add other configuration as needed
EOF
```

### 4. Create System Service File

Create `/etc/systemd/system/krakenhashes-agent.service`:

```ini
[Unit]
Description=KrakenHashes Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=krakenhashes
Group=krakenhashes
WorkingDirectory=/opt/krakenhashes-agent
ExecStart=/opt/krakenhashes-agent/krakenhashes-agent
Restart=always
RestartSec=10

# Security hardening (optional)
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/opt/krakenhashes-agent

# Resource limits (optional)
# MemoryLimit=4G
# CPUQuota=80%

# Environment (optional - .env file is preferred)
# Environment="KH_DATA_DIR=/opt/krakenhashes-agent/data"
# Environment="KH_CONFIG_DIR=/opt/krakenhashes-agent/config"

[Install]
WantedBy=multi-user.target
```

### 5. Enable and Start the Service

```bash
# Reload systemd
sudo systemctl daemon-reload

# Enable service to start on boot
sudo systemctl enable krakenhashes-agent

# Start the service
sudo systemctl start krakenhashes-agent

# Check status
sudo systemctl status krakenhashes-agent
```

### Managing System Services

```bash
# View logs
sudo journalctl -u krakenhashes-agent -f

# Stop the service
sudo systemctl stop krakenhashes-agent

# Restart the service
sudo systemctl restart krakenhashes-agent

# Disable auto-start
sudo systemctl disable krakenhashes-agent
```

## Advanced Configuration

### Using Environment Files

Instead of hardcoding environment variables in the service file, you can use a separate environment file:

1. Create an environment file (note: different from .env):
```bash
# For user service: ~/.config/krakenhashes-agent.env
# For system service: /etc/krakenhashes-agent.env

DEBUG=false
LOG_LEVEL=INFO
# Don't include sensitive data here as it may be world-readable
```

2. Reference it in the service file:
```ini
[Service]
EnvironmentFile=/path/to/environment/file
```

### Resource Limits

Control agent resource usage:

```ini
[Service]
# Limit memory usage
MemoryLimit=4G
MemoryAccounting=true

# Limit CPU usage (percentage)
CPUQuota=80%
CPUAccounting=true

# Limit number of tasks/threads
TasksMax=100
```

### Automatic Restart Configuration

```ini
[Service]
# Restart on failure
Restart=on-failure
RestartSec=10

# Or always restart (even on clean exit)
Restart=always
RestartSec=10

# Limit restart attempts
StartLimitInterval=600
StartLimitBurst=5
```

### GPU Access for System Services

If running as a system service with GPU access:

```ini
[Service]
# Add the service user to the video/render groups
SupplementaryGroups=video render

# Or for NVIDIA GPUs specifically
SupplementaryGroups=video
# May need to adjust device permissions
DeviceAllow=/dev/nvidia* rw
DeviceAllow=/dev/nvidiactl rw
DeviceAllow=/dev/nvidia-uvm rw
```

## Troubleshooting

### Common Issues

1. **Service fails to start**: Check logs with `journalctl --user -u krakenhashes-agent` (user) or `sudo journalctl -u krakenhashes-agent` (system)

2. **Permission denied errors**: 
   - User service: Ensure the agent binary is executable and in your home directory
   - System service: Check file ownership and permissions

3. **Agent can't find .env file**:
   - Ensure WorkingDirectory is set correctly in the service file
   - Check that the .env file exists in that directory

4. **GPU not detected**:
   - User service: Should work if you can access GPU normally
   - System service: May need SupplementaryGroups configuration

### Viewing Logs

```bash
# User service - last 100 lines
journalctl --user -u krakenhashes-agent -n 100

# System service - last 100 lines  
sudo journalctl -u krakenhashes-agent -n 100

# Follow logs in real-time
journalctl --user -u krakenhashes-agent -f  # User
sudo journalctl -u krakenhashes-agent -f     # System

# Logs from last boot
journalctl --user -u krakenhashes-agent -b  # User
sudo journalctl -u krakenhashes-agent -b     # System

# Export logs to file
journalctl --user -u krakenhashes-agent > agent.log  # User
sudo journalctl -u krakenhashes-agent > agent.log     # System
```

### Service Status Commands

```bash
# Check if service is active
systemctl --user is-active krakenhashes-agent   # User
sudo systemctl is-active krakenhashes-agent      # System

# Check if service is enabled
systemctl --user is-enabled krakenhashes-agent  # User  
sudo systemctl is-enabled krakenhashes-agent     # System

# Show service details
systemctl --user show krakenhashes-agent        # User
sudo systemctl show krakenhashes-agent           # System
```

## Migration Between Service Types

### From Manual to User Service

1. Stop the manual agent process
2. Copy your existing `.env` file to the agent directory
3. Follow the user service setup steps above
4. Start the user service

### From User Service to System Service

1. Stop the user service: `systemctl --user stop krakenhashes-agent`
2. Disable the user service: `systemctl --user disable krakenhashes-agent`
3. Copy your agent files to system location
4. Follow the system service setup steps above
5. Start the system service

## Best Practices

1. **Use user services** when possible - they're simpler and don't require root
2. **Keep the .env file** in the same directory as specified in WorkingDirectory
3. **Set resource limits** to prevent the agent from consuming too many resources
4. **Monitor logs regularly** to catch issues early
5. **Use enable-linger** for user services that should run without login
6. **Document your configuration** including any custom paths or settings

## Next Steps

- [Configure the agent](configuration.md)
- [Set up scheduling](scheduling.md)
- [Monitor agent performance](monitoring.md)
# Agent Configuration

## Overview

This guide covers all configuration options available for KrakenHashes agents.

## Configuration File

The agent uses a `.env` configuration file that is automatically created on first run. The file is located in the agent's working directory and contains all necessary configuration.

### Location

- **Default**: `./.env` (in the current working directory)
- **Custom**: Specified via command-line flags on first run

### Configuration Management

The `.env` file is automatically managed by the agent:
- Created on first run with values from command-line flags
- Updated with any missing configuration keys on subsequent runs
- Preserves existing values when new options are added

### Manual Editing

You can manually edit the `.env` file to adjust configuration:

```bash
# Stop the agent
sudo systemctl stop krakenhashes-agent

# Edit the configuration
nano .env

# Restart the agent
sudo systemctl start krakenhashes-agent
```

## Environment Variables (.env File)

The agent uses a `.env` file for configuration, which is automatically created during first run. The file is loaded at startup and values are NOT taken from system environment variables to avoid conflicts when running on the same host as the backend.

### .env File Structure

```bash
# Server Configuration
KH_HOST=your-server          # Backend server hostname
KH_PORT=31337                # Backend server port
USE_TLS=true                 # Use TLS for secure communication
LISTEN_INTERFACE=            # Network interface to bind to
HEARTBEAT_INTERVAL=5         # Heartbeat interval in seconds

# Agent Configuration
KH_CLAIM_CODE=               # Claim code (commented out after registration)

# Directory Configuration
KH_CONFIG_DIR=/path/to/config  # Configuration directory
KH_DATA_DIR=/path/to/data      # Data directory

# WebSocket Timing Configuration
KH_WRITE_WAIT=10s            # Timeout for writing messages
KH_PONG_WAIT=60s             # Timeout for receiving pong
KH_PING_PERIOD=54s           # Ping interval

# File Transfer Configuration
KH_MAX_CONCURRENT_DOWNLOADS=3  # Max concurrent downloads
KH_DOWNLOAD_TIMEOUT=1h         # Download timeout

# Hashcat Configuration
HASHCAT_EXTRA_PARAMS=        # Extra hashcat parameters

# Logging Configuration
DEBUG=false                  # Enable debug mode
LOG_LEVEL=INFO              # Log level
```

Note: The agent reads from the `.env` file, not from system environment variables. This prevents conflicts when running the agent and backend on the same host.

## Command Line Options

```bash
krakenhashes-agent [flags]

Flags:
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

### Example Usage

```bash
# First-time registration with custom directories
krakenhashes-agent \
  -host your-server:31337 \
  -claim YOUR_CLAIM_CODE \
  -config-dir /opt/krakenhashes/config \
  -data-dir /opt/krakenhashes/data \
  -debug

# Subsequent runs (uses .env file created during first run)
krakenhashes-agent

# Override specific settings
krakenhashes-agent -debug -hashcat-params "-O -w 4"
```

## Configuration Precedence

Settings are applied in this order (later overrides earlier):
1. Default values
2. `.env` file values (created/updated on first run)
3. Command line flags

**Important**: The agent does NOT read from system environment variables to avoid conflicts when running on the same host as the backend. All configuration is handled through the `.env` file and command-line flags.

## Device Configuration

### Enabling/Disabling Devices

You can control which devices the agent uses:

```yaml
devices:
  # Disable specific device IDs
  disabled_devices:
    - 0  # Disable first GPU
    
  # Or only enable specific devices
  enabled_devices:
    - 1
    - 2
```

### Device-Specific Settings

```yaml
devices:
  # Per-device temperature limits
  device_temps:
    0: 80  # Device 0 max temp
    1: 85  # Device 1 max temp
    
  # Per-device workload
  device_workloads:
    0: 2  # Lower workload for device 0
    1: 4  # Higher workload for device 1
```

## Security Configuration

### TLS/SSL Settings

```yaml
tls:
  # Skip certificate verification (not recommended)
  insecure_skip_verify: false
  
  # Custom CA certificate
  ca_cert_file: /etc/krakenhashes/ca.crt
  
  # Client certificates (if required)
  client_cert_file: /etc/krakenhashes/client.crt
  client_key_file: /etc/krakenhashes/client.key
```

### API Key Security

- API keys are stored encrypted in the config file
- Keys are never logged or displayed after registration
- Regenerate keys if compromised

## Performance Tuning

### Memory Management

```yaml
performance:
  # Hashcat memory settings
  hashcat_memory_limit: 4096  # MB per device
  
  # System memory reservation
  system_memory_reserve: 2048  # MB to leave free
  
  # File cache settings
  max_cache_size: 10240  # MB for wordlists/rules
```

### GPU Optimization

```yaml
performance:
  # GPU utilization target
  gpu_utilization_target: 90  # Percent
  
  # Kernel tuning
  kernel_accel: 0  # 0=auto, or specific value
  kernel_loops: 0  # 0=auto, or specific value
  
  # Power management
  gpu_power_tune: 0  # Percent adjustment (-50 to +50)
```

## Monitoring Configuration

```yaml
monitoring:
  # Metrics collection
  collect_metrics: true
  metrics_interval: 30  # seconds
  
  # Hardware monitoring
  monitor_temps: true
  monitor_fan_speed: true
  monitor_power: true
  monitor_memory: true
  
  # Alerts
  alerts:
    high_temp_threshold: 85
    low_hashrate_threshold: 1000000  # H/s
    error_rate_threshold: 0.05  # 5%
```

## Scheduling Configuration

See [Agent Scheduling](scheduling.md) for detailed scheduling configuration.

## Troubleshooting Configuration Issues

### Validation

Check configuration validity:
```bash
krakenhashes-agent validate --config /etc/krakenhashes/agent.yaml
```

### Debug Mode

Enable debug logging to see configuration loading:
```bash
krakenhashes-agent --debug --config /etc/krakenhashes/agent.yaml
```

### Common Issues

1. **Permission Denied**: Ensure agent user can read config file
2. **Invalid YAML**: Use a YAML validator
3. **Missing Required Fields**: Check server URL and data directory
4. **Environment Variable Conflicts**: Check for conflicting env vars

## Best Practices

1. **Use Configuration Management**: Store configs in Git/Ansible
2. **Secure API Keys**: Use appropriate file permissions (600)
3. **Monitor Logs**: Set up log rotation and monitoring
4. **Test Changes**: Validate config before restarting agent
5. **Document Custom Settings**: Keep notes on non-default values

## Next Steps

- [Set up agent scheduling](scheduling.md)
- [Configure file synchronization](file-sync.md)
- [Monitor agent performance](monitoring.md)
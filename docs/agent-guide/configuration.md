# Agent Configuration

## Overview

This guide covers all configuration options available for KrakenHashes agents.

## Configuration File

The agent uses a YAML configuration file, typically located at `/etc/krakenhashes/agent.yaml`.

### Basic Configuration

```yaml
# Server connection
server:
  url: https://your-server:31337
  api_key: YOUR_API_KEY_HERE  # Added during registration
  
# Data storage
data:
  directory: /var/lib/krakenhashes/agent
  
# Logging
logging:
  level: info  # debug, info, warn, error
  file: /var/log/krakenhashes/agent.log
  max_size: 100  # MB
  max_backups: 5
  max_age: 30  # days
```

### Advanced Configuration

```yaml
# Performance settings
performance:
  max_concurrent_downloads: 3
  download_timeout: 3600  # seconds
  chunk_size: 10485760  # 10MB chunks for downloads
  
# Hashcat settings
hashcat:
  binary_path: /usr/bin/hashcat  # Custom path if needed
  extra_args: "-w 3 -O"  # Default extra arguments
  workload_profile: 3  # 1-4, higher = more GPU usage
  
# Device settings
devices:
  skip_cpu: true  # Don't use CPU for cracking
  opencl_device_types: 1,2,3  # 1=CPU, 2=GPU, 3=ACCELERATOR
  cuda_devices: all  # or specific IDs like "0,1"
  
# Network settings
network:
  retry_attempts: 3
  retry_delay: 5  # seconds
  connection_timeout: 30  # seconds
  keep_alive_interval: 60  # seconds
  
# Resource limits
limits:
  max_memory: 8192  # MB
  max_cpu_percent: 80
  gpu_temp_limit: 83  # Celsius
  gpu_temp_resume: 75  # Resume when cooled to this
```

## Environment Variables

Configuration can also be set via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `KH_CONFIG_FILE` | Path to config file | `/etc/krakenhashes/agent.yaml` |
| `KH_SERVER_URL` | Backend server URL | - |
| `KH_API_KEY` | Agent API key | - |
| `KH_DATA_DIR` | Data directory | `/var/lib/krakenhashes/agent` |
| `KH_LOG_LEVEL` | Log level | `info` |
| `KH_LOG_FILE` | Log file path | `/var/log/krakenhashes/agent.log` |
| `KH_DEBUG` | Enable debug mode | `false` |

Environment variables override config file settings.

## Command Line Options

```bash
krakenhashes-agent [flags]

Flags:
  -c, --config string     Config file path
  -d, --debug            Enable debug logging
  -s, --server string    Backend server URL
  -k, --api-key string   API key for authentication
  --data-dir string      Data directory path
  --version              Show version information
  -h, --help             Show help
```

## Configuration Precedence

Settings are applied in this order (later overrides earlier):
1. Default values
2. Configuration file
3. Environment variables
4. Command line flags

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
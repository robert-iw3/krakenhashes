# Agent Management

## Overview

Agents are the computational workhorses of KrakenHashes, responsible for executing password cracking jobs using hashcat. This documentation covers all aspects of agent management, from registration to advanced configuration.

## Table of Contents

- [Agent Registration](#agent-registration)
- [Agent Configuration](#agent-configuration)
- [Agent Scheduling](scheduling.md)
- [Device Management](#device-management)
- [Monitoring and Metrics](#monitoring-and-metrics)
- [Troubleshooting](#troubleshooting)

## Agent Registration

### Using Claim Codes

Agents register with the KrakenHashes backend using claim codes:

1. **Generate a Claim Code** (Admin):
   - Navigate to Agents → Manage Vouchers
   - Click "Create Voucher"
   - Choose voucher type:
     - **One-time**: Used once and deactivated
     - **Continuous**: Can be used by multiple agents

2. **Register Agent** (On Agent Machine):
   ```bash
   ./krakenhashes-agent register --code YOUR_CLAIM_CODE --server https://your-server:31337
   ```

3. **Verify Registration**:
   - Check Agents list in the web UI
   - Verify agent appears with "online" status

## Agent Configuration

### Basic Settings

Each agent has several configurable settings:

- **Name**: Automatically detected hostname (can be changed)
- **Enabled/Disabled**: Control whether agent receives jobs
- **Owner**: Assign agent to a specific user
- **Extra Parameters**: Additional hashcat parameters

### Advanced Configuration

#### Extra Parameters

The "Extra Parameters" field allows you to specify agent-specific hashcat options:

```
-w 4 -O --force
```

Common parameters:
- `-w [1-4]`: Workload profile (1=low, 4=high)
- `-O`: Optimized kernel (faster but limited password length)
- `--force`: Ignore warnings
- `-d [device]`: Specific device selection

#### Device Management

Agents automatically detect available GPUs. You can:
- Enable/disable specific devices
- View device capabilities
- Monitor device performance

### Agent Status

Agents can have the following statuses:

- **Online**: Connected and ready for jobs
- **Offline**: Not connected to backend
- **Working**: Currently executing a job
- **Error**: Encountered an error (check logs)

## Device Management

### GPU Detection

KrakenHashes agents support:
- NVIDIA GPUs (CUDA)
- AMD GPUs (OpenCL)
- Intel GPUs (OpenCL)
- CPU-based cracking (fallback)

### Device Configuration

Each detected device can be:
1. **Enabled/Disabled**: Toggle device availability
2. **Monitored**: View real-time metrics
3. **Benchmarked**: Test performance

### Multi-GPU Setup

For systems with multiple GPUs:
1. All GPUs are detected automatically
2. Enable/disable GPUs individually
3. Hashcat will use all enabled GPUs in parallel

## Monitoring and Metrics

### Real-time Metrics

The agent details page shows:
- **Temperature**: GPU core temperature
- **Utilization**: GPU usage percentage
- **Fan Speed**: Cooling fan percentage
- **Hash Rate**: Current cracking speed

### Performance Optimization

Tips for optimal performance:
1. **Temperature Management**: Keep GPUs below 80°C
2. **Workload Tuning**: Adjust `-w` parameter based on system use
3. **Driver Updates**: Keep GPU drivers current
4. **Power Settings**: Ensure GPUs aren't power-throttled

## Agent Files and Synchronization

Agents automatically synchronize required files:
- **Wordlists**: Downloaded on-demand
- **Rules**: Cached locally
- **Hashcat Binaries**: Auto-updated
- **Markov Files**: Synced as needed

File locations:
```
/var/lib/krakenhashes/agent/
├── wordlists/
├── rules/
├── hashlists/
├── potfiles/
└── markov/
```

## Security

### API Key Management

Each agent has a unique API key:
- Generated during registration
- Stored securely on agent
- Can be regenerated if compromised
- Used for all agent-backend communication

### Network Security

Agent communication:
- TLS encrypted (HTTPS)
- Certificate validation
- API key authentication
- No inbound connections required

## Troubleshooting

### Common Issues

#### Agent Shows Offline

1. Check agent process is running
2. Verify network connectivity
3. Check firewall rules (port 31337)
4. Validate SSL certificates
5. Review agent logs

#### Agent Not Receiving Jobs

1. Verify agent is enabled
2. Check scheduling (if enabled)
3. Ensure GPUs are detected
4. Verify owner assignment (if job is user-specific)
5. Check job requirements match agent capabilities

#### Performance Issues

1. Monitor GPU temperatures
2. Check system resources (RAM, CPU)
3. Verify no other GPU-intensive processes
4. Review hashcat parameters
5. Update GPU drivers

### Log Locations

- **Agent Logs**: `/var/log/krakenhashes/agent.log`
- **System Logs**: `journalctl -u krakenhashes-agent`
- **Hashcat Output**: Streamed to backend in real-time

### Debug Mode

Run agent with debug logging:
```bash
./krakenhashes-agent --debug
```

## Best Practices

1. **Regular Updates**: Keep agents updated to latest version
2. **Monitoring**: Set up alerts for offline agents
3. **Scheduling**: Use scheduling for cost/heat management
4. **Redundancy**: Have multiple agents for critical operations
5. **Documentation**: Document agent-specific configurations

## Related Documentation

- [Agent Scheduling](scheduling.md) - Detailed scheduling configuration
- [Installation Guide](../installation.md) - Initial setup
- [Admin Guide](../admin/) - Administrative tasks
- [API Reference](../api/) - Backend API documentation
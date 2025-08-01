# Agent Management Guide

This guide covers the comprehensive management of KrakenHashes agents, including registration, monitoring, scheduling, and troubleshooting.

## Table of Contents

1. [Understanding Agents](#understanding-agents)
2. [Agent Registration Process](#agent-registration-process)
3. [Managing Agent Connections](#managing-agent-connections)
4. [Monitoring Agent Health and Performance](#monitoring-agent-health-and-performance)
5. [Agent Scheduling and Availability](#agent-scheduling-and-availability)
6. [Hardware Capabilities and Benchmarks](#hardware-capabilities-and-benchmarks)
7. [Troubleshooting Agent Issues](#troubleshooting-agent-issues)

## Understanding Agents

### What are Agents?

Agents are distributed compute nodes that execute password cracking tasks using hashcat. They connect to the KrakenHashes backend server via WebSocket and receive job assignments based on their availability and capabilities.

### Agent Architecture

Each agent consists of:
- **Hardware Detection**: Automatic detection of GPUs (NVIDIA, AMD, Intel) and CPUs
- **Performance Monitoring**: Real-time tracking of resource utilization
- **File Synchronization**: Automatic download of wordlists, rules, and hashlists
- **Job Execution**: Running hashcat with specified attack parameters
- **Result Reporting**: Real-time crack updates back to the server

### Agent States

Agents can be in one of the following states:

- **`pending`**: Initial registration state, awaiting activation
- **`active`**: Connected and ready to receive jobs
- **`inactive`**: Disconnected but previously active
- **`error`**: Experiencing issues preventing normal operation
- **`disabled`**: Administratively disabled

## Agent Registration Process

### Overview

Agent registration uses a claim code (voucher) system to ensure only authorized agents can join the system.

### Creating Claim Codes

1. **Navigate to Admin Panel**
   - Go to Settings → Agent Management → Claim Codes

2. **Generate New Claim Code**
   ```
   Type: Single-use or Continuous
   Owner: Select user who will own the agents
   ```

3. **Claim Code Types**
   - **Single-use**: Can only be used once to register one agent
   - **Continuous**: Can be used multiple times (useful for auto-scaling)

### Agent Registration Steps

1. **Agent Installation**
   ```bash
   # On the agent machine
   ./krakenhashes-agent install
   ```

2. **Initial Registration**
   ```bash
   # Using the claim code
   ./krakenhashes-agent register --server https://your-server:8443 --claim-code XXXX-XXXX-XXXX
   ```

3. **Certificate Download**
   - Agent automatically downloads TLS certificates
   - Stores credentials in `~/.krakenhashes/agent/`

4. **Connection Establishment**
   - Agent connects via WebSocket using API key authentication
   - Sends hardware information and capabilities

### Registration Security

- Claim codes are normalized (uppercase, no hyphens)
- API keys are generated using cryptographically secure random bytes
- TLS certificates ensure encrypted communication
- Agent ID and API key must match for authentication

## Managing Agent Connections

### WebSocket Communication

Agents maintain persistent WebSocket connections for:
- Real-time job assignments
- Status updates
- Crack result reporting
- Heartbeat monitoring

### Connection Parameters

```yaml
# Environment variables for connection tuning
KH_WRITE_WAIT: "10s"      # Write timeout
KH_PONG_WAIT: "60s"       # Time to wait for pong response
KH_PING_PERIOD: "54s"     # Ping interval (must be < pong wait)
```

### Monitoring Connected Agents

1. **Dashboard View**
   - Real-time agent status on main dashboard
   - Shows connected/disconnected agents
   - Current task assignments

2. **Agent List Page**
   - Detailed view of all agents
   - Filter by status, owner, or team
   - Last heartbeat timestamps

### Managing Agent Settings

```json
// PUT /api/agents/{id}
{
  "isEnabled": true,
  "ownerId": "user-uuid",
  "extraParameters": "--custom-charset1=?l?u?d"
}
```

### Disabling/Enabling Agents

- Disabled agents remain connected but don't receive jobs
- Useful for maintenance or troubleshooting
- Preserves agent configuration and history

## Monitoring Agent Health and Performance

### Real-time Metrics

Agents report metrics every 30 seconds:

```json
{
  "cpu_usage": 45.2,
  "memory_usage": 62.8,
  "gpu_utilization": 98.5,
  "gpu_temp": 72.0,
  "gpu_metrics": {
    "device_0": {
      "temperature": 72,
      "utilization": 98.5,
      "memory_used": 8192,
      "fan_speed": 85
    }
  }
}
```

### Performance Monitoring Dashboard

1. **Agent Detail Page**
   - Historical performance graphs
   - Temperature trends
   - Utilization patterns
   - Hash rate performance

2. **Metrics Time Ranges**
   - 1 hour (default)
   - 24 hours
   - 7 days
   - 30 days

### Device Management

Each agent can have multiple devices (GPUs):

```json
// GET /api/agents/{id}/devices
[
  {
    "id": 1,
    "device_index": 0,
    "device_type": "GPU",
    "device_name": "NVIDIA GeForce RTX 4090",
    "is_enabled": true,
    "capabilities": {
      "compute_capability": "8.9",
      "memory": 24576
    }
  }
]
```

### Enabling/Disabling Devices

```json
// PUT /api/agents/{id}/devices/{deviceId}
{
  "enabled": false  // Disable specific GPU
}
```

## Agent Scheduling and Availability

### Scheduling Overview

Agents support weekly scheduling to optimize resource usage and costs.

### Configuring Agent Schedules

1. **Enable Scheduling**
   ```json
   // PUT /api/agents/{id}/scheduling-enabled
   {
     "enabled": true,
     "timezone": "America/New_York"
   }
   ```

2. **Set Daily Schedules**
   ```json
   // POST /api/agents/{id}/schedules
   {
     "dayOfWeek": 1,  // Monday (0=Sunday, 6=Saturday)
     "startTimeUTC": "22:00",
     "endTimeUTC": "06:00",
     "timezone": "America/New_York",
     "isActive": true
   }
   ```

### Schedule Features

- **UTC Storage**: All times stored in UTC for consistency
- **Timezone Display**: Shown in user's local timezone
- **Overnight Support**: Schedules can span midnight
- **Bulk Updates**: Update entire week at once

### Schedule Validation

- Start and end times must be different
- Day of week must be 0-6
- Times in HH:MM format
- Automatic handling of daylight saving time

### Availability Considerations

When scheduling is enabled:
- Agents only receive jobs during scheduled hours
- Running jobs continue to completion
- Agents remain connected outside schedule
- Heartbeat monitoring continues

## Hardware Capabilities and Benchmarks

### Hardware Detection

Agents automatically detect:

```json
{
  "hardware": {
    "cpus": [
      {
        "model": "AMD Ryzen 9 7950X",
        "cores": 16,
        "threads": 32,
        "frequency": 4.5
      }
    ],
    "gpus": [
      {
        "vendor": "NVIDIA",
        "model": "GeForce RTX 4090",
        "memory": 24576,
        "driver": "545.29.06"
      }
    ]
  }
}
```

### Benchmark System

Agents can run benchmarks for different hash types:

```sql
-- Benchmark results stored per agent
agent_benchmarks (
  agent_id,
  attack_mode,     -- 0=dictionary, 3=bruteforce, etc.
  hash_type,       -- 0=MD5, 1000=NTLM, etc.
  speed,           -- Hashes per second
  created_at
)
```

### Performance Metrics

Key metrics tracked:
- **Hash Rate**: Speed for each hash type
- **GPU Temperature**: Thermal monitoring
- **GPU Utilization**: Processing efficiency
- **Memory Usage**: VRAM consumption
- **Power Consumption**: Wattage tracking

### Consecutive Failure Tracking

Agents track consecutive task failures:
- Increments on task failure
- Resets on successful completion
- Can trigger automatic disabling
- Helps identify problematic agents

## Troubleshooting Agent Issues

### Common Connection Issues

1. **Agent Won't Connect**
   - Check TLS certificates are valid
   - Verify API key hasn't expired
   - Ensure firewall allows WebSocket (port 8443)
   - Check agent logs for detailed errors

2. **Frequent Disconnections**
   - Review ping/pong timeout settings
   - Check network stability
   - Monitor agent system resources
   - Verify no proxy interference

### Authentication Problems

1. **Invalid API Key**
   ```bash
   # Regenerate API key
   ./krakenhashes-agent reregister --claim-code NEW-CODE
   ```

2. **Certificate Issues**
   - Check certificate expiration
   - Verify CA certificate is trusted
   - Ensure certificate matches server hostname

### Performance Issues

1. **Low Hash Rates**
   - Check GPU driver versions
   - Monitor thermal throttling
   - Verify power settings
   - Review extra parameters

2. **High Failure Rate**
   - Check hashcat binary compatibility
   - Verify file synchronization
   - Review job parameters
   - Monitor system stability

### Debugging Tools

1. **Agent Logs**
   ```bash
   # View agent logs
   tail -f ~/.krakenhashes/agent/logs/agent.log
   ```

2. **Server-side Monitoring**
   ```sql
   -- Check agent errors
   SELECT id, name, last_error, consecutive_failures
   FROM agents
   WHERE last_error IS NOT NULL
   ORDER BY updated_at DESC;
   ```

3. **WebSocket Messages**
   - Enable debug logging for detailed messages
   - Monitor heartbeat intervals
   - Check message acknowledgments

### Recovery Procedures

1. **Reset Agent State**
   ```sql
   -- Clear error state
   UPDATE agents 
   SET status = 'inactive', 
       last_error = NULL,
       consecutive_failures = 0
   WHERE id = ?;
   ```

2. **Force Reconnection**
   - Restart agent service
   - Clear local cache
   - Verify network connectivity

3. **Complete Re-registration**
   - Generate new claim code
   - Remove agent from database
   - Perform fresh registration

### Monitoring Best Practices

1. **Set Up Alerts**
   - Agent offline > 5 minutes
   - Consecutive failures > 3
   - Temperature > 85°C
   - Low hash rates

2. **Regular Maintenance**
   - Update agent software
   - Clean GPU fans
   - Check thermal paste
   - Update drivers

3. **Capacity Planning**
   - Monitor job queue depth
   - Track agent utilization
   - Plan for peak loads
   - Consider scheduling optimization

## Advanced Topics

### Agent Clustering

- Group agents by capability
- Assign specialized workloads
- Balance load across regions
- Implement failover strategies

### Security Hardening

- Rotate API keys periodically
- Implement IP whitelisting
- Use dedicated agent VLANs
- Monitor for anomalous behavior

### Integration Points

- Export metrics to monitoring systems
- Webhook notifications for events
- API automation for scaling
- Custom scheduling algorithms

## Conclusion

Effective agent management is crucial for maintaining a high-performance distributed cracking system. Regular monitoring, proper scheduling, and proactive troubleshooting ensure optimal resource utilization and job completion rates.

For additional support or advanced configurations, consult the system administrator documentation or contact the development team.
# Implementation Summary: Benchmark-Based Job Assignment

## Overview

Updated the backend's job scheduling service to wait for agent benchmarks before assigning work. This ensures accurate chunk calculations based on real-world performance metrics.

## Key Changes

### 1. **JobSchedulingService** (`backend/internal/services/job_scheduling_service.go`)

#### Updated Interface
```go
type JobWebSocketIntegration interface {
    SendJobAssignment(ctx context.Context, task *models.JobTask, jobExecution *models.JobExecution) error
    RequestAgentBenchmark(ctx context.Context, agentID int, jobExecution *models.JobExecution) error
}
```

#### Modified `assignWorkToAgent` Function
- Added benchmark validation before calculating chunks
- Retrieves hashlist to get hash type for benchmark lookup
- Checks if agent has a valid benchmark for the attack mode/hash type combination
- Validates benchmark freshness using configurable cache duration
- If no valid benchmark exists:
  - Sends benchmark request via WebSocket
  - Returns without assigning work (deferred assignment)
  - Agent remains available for next scheduling cycle
- If valid benchmark exists:
  - Proceeds with normal chunk calculation and assignment

### 2. **JobWebSocketIntegration** (`backend/internal/integration/job_websocket_integration.go`)

#### New Method: `RequestAgentBenchmark`
- Implements the new interface method
- Retrieves full job configuration:
  - Preset job details (binary version, wordlists, rules, mask)
  - Hashlist details (hash type)
  - Agent information
- Builds enhanced benchmark request with:
  - Actual wordlist paths from the job
  - Rule files if applicable
  - Binary path for the specific version
  - Hash type and attack mode
  - Test duration (30 seconds for accuracy)
- Sends comprehensive benchmark request to agent

### 3. **WebSocket Types** (already existed, enhanced usage)

The `BenchmarkRequestPayload` now includes:
- `HashlistID` and `HashlistPath` for real-world testing
- `WordlistPaths` array for dictionary attacks
- `RulePaths` array for rule-based attacks
- `Mask` for brute force patterns
- `TestDuration` for benchmark duration

## Workflow

1. **Agent requests work** → Scheduler finds pending job
2. **Benchmark check** → System verifies agent has valid benchmark
3. **If no benchmark**:
   - Request benchmark with full job config
   - Agent performs real-world speed test
   - Reports results back
   - Next scheduling cycle assigns work
4. **If valid benchmark exists**:
   - Calculate chunk based on known performance
   - Assign work immediately

## Benefits

- **Accurate Performance**: Benchmarks use actual job parameters
- **Optimal Chunks**: Prevents over/under-sized work assignments
- **Reduced Failures**: Avoids assigning impossible workloads
- **Better Utilization**: Maximizes agent efficiency

## Configuration

- `benchmark_cache_duration_hours`: Benchmark validity period (default: 168 hours)
- `default_chunk_duration`: Target chunk duration in seconds (default: 1200)
- `chunk_fluctuation_percentage`: Final chunk size tolerance (default: 20%)

## Testing

Created unit tests demonstrating:
- Benchmark request flow when no benchmark exists
- Job assignment flow with valid benchmark
- Mock WebSocket integration for testing

## Files Modified

1. `/backend/internal/services/job_scheduling_service.go`
2. `/backend/internal/integration/job_websocket_integration.go`
3. `/backend/internal/services/job_scheduling_benchmark_test.go` (new)
4. `/backend/BENCHMARK_WORKFLOW.md` (new documentation)

## Future Considerations

- Benchmark history tracking
- Performance anomaly detection
- Multi-GPU per-device benchmarking
- Predictive performance modeling
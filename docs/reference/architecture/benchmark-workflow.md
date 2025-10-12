# Benchmark-Based Job Assignment Workflow

## Overview

The job scheduling service now implements a benchmark-first approach for job assignment. Before assigning work to an agent, the system verifies that the agent has a valid benchmark for the specific attack mode and hash type combination.

## Workflow

1. **Job Assignment Request**
   - Scheduler identifies an available agent and a pending job
   - Job execution details are retrieved, including the hashlist

2. **Benchmark Check**
   - System checks if agent has a benchmark for the attack mode and hash type
   - If benchmark exists, checks if it's still valid (default: 7 days cache)
   - Cache duration can be configured via `benchmark_cache_duration_hours` setting

3. **Benchmark Request (if needed)**
   - If no valid benchmark exists, system sends enhanced benchmark request
   - Request includes actual job configuration:
     - Binary version
     - Wordlists and rules (if applicable)
     - Mask (for brute force attacks)
     - Hash type and attack mode
     - Test duration (30 seconds)
   - Job assignment is deferred until benchmark completes

4. **Benchmark Execution (Agent side)**
   - Agent receives benchmark request with full job configuration
   - Runs actual hashcat benchmark with the specific parameters
   - Reports back real-world performance metrics

5. **Job Assignment (after benchmark)**
   - Once benchmark is received and stored, agent becomes available again
   - Next scheduling cycle will find the valid benchmark
   - Chunk calculation uses accurate performance data
   - Job task is assigned with properly sized chunks

## Benefits

- **Accurate Performance Estimation**: Benchmarks use actual job configuration
- **Optimal Chunk Sizing**: Prevents under/over-utilization of agents
- **Reduced Job Failures**: Avoids assigning work that agents can't handle
- **Better Resource Utilization**: Chunks are sized based on real performance

## Configuration

- `benchmark_cache_duration_hours`: How long benchmarks remain valid (default: 168 hours / 7 days)
- `chunk_fluctuation_percentage`: Tolerance for final chunk size variations (default: 20%)
- `default_chunk_duration`: Target duration for each chunk in seconds (default: 1200 / 20 minutes)

## Implementation Details

### Modified Components

1. **JobSchedulingService** (`assignWorkToAgent`)
   - Added benchmark validation before chunk calculation
   - Defers assignment if benchmark is needed
   - Retrieves hashlist to get hash type

2. **JobWebSocketIntegration** (`RequestAgentBenchmark`)
   - New method implementing the interface
   - Sends enhanced benchmark request with full job configuration
   - Includes wordlists, rules, mask, and binary information

3. **WebSocket Types**
   - `BenchmarkRequestPayload` enhanced with job-specific fields
   - Supports real-world speed testing with actual attack parameters

### Error Handling

- Missing benchmarks trigger requests instead of failures
- Invalid benchmarks are detected and refreshed
- WebSocket unavailability is properly handled
- Graceful degradation if benchmark request fails

## Accurate Keyspace Tracking

In addition to benchmarking for performance estimation, the system captures accurate keyspace values from hashcat to ensure precise progress tracking.

### Why Accurate Keyspace Tracking?

When using rules or combination attacks, estimating the total keyspace can be inaccurate. For example:
- **Rule-based attacks**: Estimated keyspace = wordlist_size × rule_count, but actual keyspace varies based on rule effectiveness
- **Combination attacks**: Certain combinations may be invalid or duplicates

Hashcat provides the actual keyspace through `progress[1]` values, which the system captures to ensure accurate progress reporting.

### Keyspace Capture Workflow

1. **Initial Job Creation**
   - Job created with estimated `effective_keyspace` based on wordlists/rules
   - Flag `is_accurate_keyspace` set to `false`
   - Estimation needed for rule splitting decisions

2. **Forced Benchmark for First Agent**
   - When first agent connects (taskCount = 0), system requests benchmark
   - Benchmark includes actual job configuration (wordlists, rules, mask, hash type)
   - Agent runs hashcat benchmark and captures `progress[1]` value

3. **Accurate Keyspace Capture**
   - Backend receives benchmark result with `TotalEffectiveKeyspace` from `progress[1]`
   - Updates job execution:
     - Sets `effective_keyspace` to actual value from hashcat
     - Sets `is_accurate_keyspace` to `true`
     - Calculates `avg_rule_multiplier` = actual / estimated
   - Subsequent agents skip benchmark and use cached job-level keyspace

4. **Fallback: First Progress Update**
   - If benchmark doesn't provide keyspace, first task progress update does
   - Agent sends `progress[1]` value in first progress message with `IsFirstUpdate` flag
   - Backend updates both job-level and task-level keyspace
   - Sets `is_actual_keyspace` to `true` for the task

5. **Future Task Improvements**
   - New tasks use `avg_rule_multiplier` to improve estimated keyspace
   - Provides better estimates for chunks not yet processed
   - Helps with more accurate progress reporting across the job

### Benefits of Accurate Keyspace Tracking

- **Precise Progress**: Progress percentages reflect actual hashcat progress, not estimates
- **Better Task Distribution**: Chunk sizes calculated based on real keyspace
- **Improved Estimates**: Future tasks benefit from multiplier derived from actual values
- **Consistency**: All agents working on same job use same accurate keyspace

### Database Columns

**job_executions table:**
- `is_accurate_keyspace` (boolean): True when keyspace is from hashcat `progress[1]`
- `avg_rule_multiplier` (float): Ratio of actual/estimated keyspace for improving future estimates

**job_tasks table:**
- `is_actual_keyspace` (boolean): True when task has actual keyspace from progress update

## Future Enhancements

1. **Benchmark History**: Track benchmark trends over time
2. **Performance Prediction**: Use ML to predict performance for new combinations
3. **Dynamic Re-benchmarking**: Trigger new benchmarks on performance anomalies
4. **Multi-GPU Optimization**: Per-device benchmark tracking
5. **Keyspace Prediction**: Use historical multipliers to improve initial estimates
# Automatic Job Completion System

## Overview

KrakenHashes automatically detects when all hashes in a hashlist have been cracked and manages the lifecycle of related jobs to prevent failures and wasted resources.

## The Problem

Hashcat's `--remove` option removes cracked hashes from input files during execution. When all hashes are cracked:
- The hashlist file becomes empty
- Subsequent jobs targeting that hashlist fail immediately
- Resources are wasted attempting to process empty files
- Users receive confusing error messages

## The Solution

### Status Code 6 Detection

The agent monitors hashcat's JSON status output for status code 6, which indicates "all hashes cracked." This code is sent by hashcat when:
- The input file has no remaining uncracked hashes
- All work is complete for the given hashlist

### Trust Model

The system **trusts status code 6 as authoritative** without database verification because:
- Hashcat knows definitively when all hashes are cracked
- Database verification would create race conditions
- Status code 6 is a reliable signal from hashcat
- Prevents complex synchronization issues

### Job Cleanup Process

When status code 6 is received:

1. **Identify All Affected Jobs**: Query for ALL jobs (any status) targeting the same hashlist
2. **Running Jobs**:
   - Send WebSocket stop signals to active agents
   - Mark jobs as "completed" at 100% progress
   - Send completion email notifications
3. **Pending Jobs**:
   - Delete jobs that haven't started yet
   - No email notifications (jobs never ran)
4. **Prevention**: New tasks for this hashlist won't be created

### Technical Implementation

**Components:**
- `HashlistCompletionService`: Handles job cleanup logic
- `AllHashesCracked` flag in WebSocket messages
- Background processing with 5-minute timeout

**Flow:**
```
Agent detects status code 6 → Sets AllHashesCracked flag →
Backend handler triggered → HashlistCompletionService runs async →
Stop running tasks + Delete pending jobs → Send notifications
```

**Code Location:** `backend/internal/services/hashlist_completion_service.go`

## Agent-Side Implementation

### Detection

In `agent/internal/jobs/hashcat_executor.go`:
- Parses hashcat JSON status output
- Checks for `status` field equal to 6
- Sets `AllHashesCracked` flag in progress update message
- Flag sent with regular progress updates (no special message needed)

### Timing

- Detection occurs during normal progress monitoring
- No additional API calls required
- Flag transmitted with existing WebSocket infrastructure

## Backend-Side Implementation

### Message Handling

In `backend/internal/routes/websocket_with_jobs.go`:
- Checks `AllHashesCracked` flag in job progress messages
- Triggers before status-specific processing
- Runs HashlistCompletionService asynchronously

### Service Logic

`HashlistCompletionService.HandleHashlistCompletion()`:

1. **Query Affected Jobs**:
   ```sql
   SELECT * FROM job_executions
   WHERE hashlist_id = ?
   AND status IN ('pending', 'running', 'paused')
   ```

2. **Process Running Jobs**:
   - Find active tasks for each running job
   - Send stop signals via WebSocket
   - Update job status to 'completed'
   - Set progress to 100%
   - Trigger email notifications

3. **Process Pending Jobs**:
   - Delete jobs that haven't started
   - Clean up any associated data
   - No notifications needed

4. **Update Job Priority**:
   - Comprehensive processing regardless of priority
   - Handles all affected jobs in single operation

## Configuration

No configuration required - this feature is always active.

## Benefits

1. **Prevents Failures**: No more failed jobs due to empty hashlist files
2. **Resource Efficiency**: Stops wasting resources on completed hashlists
3. **User Experience**: Automatic cleanup without manual intervention
4. **Notifications**: Users informed of successful completion
5. **Clean State**: Queue automatically cleaned of obsolete jobs

## Error Handling

### Timeout Protection
- 5-minute timeout for cleanup operations
- Prevents hanging if service encounters issues
- Logged errors don't block agent progress reporting

### Transaction Safety
- Database operations use transactions
- Rollback on errors ensures consistency
- Agent continues normal operation regardless of cleanup success

### WebSocket Errors
- Gracefully handles disconnected agents
- Tasks marked for stop even if agent offline
- Agent reconnection triggers cleanup on next connection

## Limitations

- Trusts hashcat status code 6 without verification
- Only handles jobs for the same hashlist (doesn't affect other hashlists)
- Requires agent to detect and report status code 6
- Depends on WebSocket connectivity for stop signals

## Testing

Tested with hashlist 85:
- 1 running job completed at 100% with stop signal sent
- 2 pending jobs deleted (never started)
- Email notifications triggered successfully
- No errors in logs

## Monitoring and Debugging

### Log Messages

Success:
```
Successfully completed job [uuid] for hashlist [id]
Successfully deleted pending job [uuid] for hashlist [id]
```

Errors:
```
Failed to stop tasks for job [uuid]: [error]
Failed to complete job [uuid]: [error]
```

### Metrics

Track in monitoring:
- Number of jobs auto-completed
- Number of pending jobs cleaned up
- Time taken for cleanup operations
- Failed cleanup attempts

## Related Documentation

- [Chunking System](./chunking.md) - How jobs are divided into chunks
- [Job Update System](./job-update-system.md) - How keyspace updates work
- [Jobs & Workflows](../../user-guide/jobs-workflows.md) - User perspective on automatic completion
- [Core Concepts](../../user-guide/core-concepts.md) - Understanding job execution flow

## Future Enhancements

Potential improvements under consideration:

- **Partial Completion Threshold**: Complete jobs when X% of hashes cracked (configurable)
- **Notification Customization**: Per-client notification preferences
- **Completion Hooks**: Custom scripts triggered on hashlist completion
- **Statistics Tracking**: Historical data on completion rates and timing
- **Manual Override**: Allow users to force completion or prevent automatic cleanup

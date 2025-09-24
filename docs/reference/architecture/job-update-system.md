# Job Update System

## Overview

The KrakenHashes Job Update System automatically recalculates job keyspaces when associated wordlists, rules, or potfiles change during execution. This is a **"going forward" system** - when files are updated, only undispatched work is affected. Already-assigned tasks continue with their original parameters, ensuring consistency while allowing jobs to benefit from updated resources.

## Core Philosophy: Forward-Only Updates

The system operates on these principles:

1. **No Deficit Tracking**: The system doesn't track "missed" work from updates that occur after tasks are dispatched
2. **Current State Calculation**: Keyspaces are recalculated based on the current file state and remaining work
3. **Non-Disruptive**: Running tasks are never interrupted or restarted
4. **Automatic Adjustment**: Jobs automatically adapt to file changes without user intervention

## How It Works

### Directory Monitoring

The system continuously monitors three key directories:

- **Wordlists**: `/data/krakenhashes/wordlists/`
- **Rules**: `/data/krakenhashes/rules/`
- **Potfile**: Special handling via staging mechanism

Every 30 seconds (configurable), the directory monitor:
1. Calculates MD5 hashes of all monitored files
2. Compares with previous hashes to detect changes
3. Updates file metadata in the database
4. Triggers job updates for affected jobs

### Change Detection Flow

```
File Change → MD5 Hash Comparison → Metadata Update → Job Update Service → Keyspace Recalculation
```

## Wordlist Updates

When a wordlist file changes (words added or removed):

### For Jobs WITHOUT Rule Splitting

1. **Base keyspace** updates to new word count
2. **Effective keyspace** recalculates:
   - With rules: `new_wordlist_size × multiplication_factor`
   - Without rules: `new_wordlist_size`

### For Jobs WITH Rule Splitting

The system accounts for already-dispatched rule chunks:

1. Calculates theoretical new effective keyspace
2. Determines "missed" keyspace: `words_added × rules_already_dispatched`
3. Actual effective keyspace: `theoretical - missed`

**Example**:
```
Original: 1,000,000 words × 10,000 rules = 10 billion keyspace
After 5,000 rules dispatched, add 100,000 words:
- Theoretical: 1,100,000 × 10,000 = 11 billion
- Missed: 100,000 × 5,000 = 500 million
- Actual: 11 billion - 500 million = 10.5 billion
```

## Rule Updates

When a rule file changes (rules added or removed):

### Jobs Without Tasks Yet

- Simple recalculation: `base_keyspace × new_rule_count`
- Multiplication factor updates to new rule count

### Jobs With Existing Tasks

For rule-splitting jobs:
1. Checks highest dispatched rule index
2. If new rule count ≤ max dispatched: Job effectively complete
3. Otherwise: Updates multiplication factor and recalculates

**Example**:
```
Original: 10,000 rules, 5,000 dispatched
Rules reduced to 4,000: Job marked complete (all remaining rules gone)
Rules increased to 12,000: 7,000 rules remain to process
```

## Potfile Updates

The potfile (collection of cracked passwords) has special handling:

### Staging Mechanism

1. Cracked passwords accumulate in a staging table
2. Periodic or manual refresh moves staged entries to potfile
3. Potfile treated as a special wordlist for job purposes

### Update Process

1. **Manual Refresh**: User triggers from frontend
2. **Staging Integration**: Moves cracked passwords to main potfile
3. **Line Count Update**: Updates wordlist metadata
4. **Job Updates**: Triggers same update logic as regular wordlists

### Key Differences

- Not monitored by directory monitor (excluded from scans)
- Updates via database staging, not file watching
- Requires explicit refresh action
- Always grows (passwords only added, never removed)

## Keyspace Recalculation Logic

### Basic Formula

```
Effective Keyspace = Base Keyspace × Multiplication Factor
```

Where:
- **Base Keyspace**: Current wordlist size
- **Multiplication Factor**: Number of rules (or 1 if no rules)

### Adjustments for Dispatched Work

For rule-splitting jobs with updates:
```
Adjusted Keyspace = New Effective - (Change × Dispatched Rules)
```

This ensures already-dispatched tasks aren't double-counted.

## Real-World Examples

### Scenario 1: Growing Wordlist

**Initial State**:
- Wordlist: 1 million words
- Rules: 1,000
- No tasks dispatched yet

**After Adding 100,000 Words**:
- New base: 1.1 million
- New effective: 1.1 billion
- All future tasks use updated wordlist

### Scenario 2: Rule File Expansion During Execution

**Initial State**:
- Job using rule splitting
- 10,000 rules, split into 100 chunks
- 50 chunks already dispatched (5,000 rules)

**After Adding 2,000 Rules**:
- Total rules: 12,000
- Remaining: 7,000 rules (chunks 51-120)
- Future chunks use expanded rule set

### Scenario 3: Potfile Growth

**Initial State**:
- Potfile job with 1,000 existing passwords
- Rules: 500
- Effective keyspace: 500,000

**After Cracking Campaign**:
- 200 new passwords cracked
- Manual refresh triggered
- New base: 1,200 passwords
- New effective: 600,000

## Configuration

### Directory Monitor Settings

Located in backend configuration:

| Setting | Default | Description |
|---------|---------|-------------|
| Monitor Interval | 30s | How often to check for file changes |
| MD5 Hash Check | Enabled | Method for detecting changes |
| Concurrent Updates | Enabled | Allow parallel job updates |

### System Behavior Settings

| Setting | Default | Description |
|---------|---------|-------------|
| Auto-update Jobs | Enabled | Automatically update affected jobs |
| Update Lock Timeout | 60s | Maximum time to wait for job lock |
| Staging Refresh Interval | Manual | Potfile staging refresh trigger |

## Technical Implementation

### Components

1. **DirectoryMonitorService**: Detects file changes via MD5 hashing
2. **JobUpdateService**: Handles keyspace recalculation logic
3. **PotfileService**: Manages potfile staging and updates
4. **Repository Layer**: Database operations for job updates

### Database Tables Involved

- `job_executions`: Stores base_keyspace, effective_keyspace, multiplication_factor
- `job_tasks`: Tracks dispatched work (rule_start_index, rule_end_index)
- `wordlists`: Metadata including word_count, file_hash
- `rules`: Metadata including rule_count, file_hash
- `potfile_staging`: Temporary storage for cracked passwords

### Locking Strategy

The system uses per-job locks to prevent race conditions:
```go
// Lock specific job during updates
s.lockJob(jobID)
defer s.unlockJob(jobID)
```

## Best Practices

### For Users

1. **Expect Keyspace Changes**: Don't be alarmed if keyspaces update during execution
2. **Manual Potfile Refresh**: Remember to refresh potfile after cracking campaigns
3. **Monitor Progress**: Check effective keyspace to understand total work
4. **Plan Updates**: Large file changes can significantly affect running jobs

### For Administrators

1. **Monitor Disk Space**: File updates may require temporary storage
2. **Adjust Check Intervals**: Balance between responsiveness and system load
3. **Review Logs**: Check for update failures or lock timeouts
4. **Database Maintenance**: Ensure potfile staging table doesn't grow too large

### For Developers

1. **Respect Forward-Only**: Never try to retroactively update dispatched tasks
2. **Use Job Locks**: Always lock jobs during updates to prevent races
3. **Handle Errors Gracefully**: File update failures shouldn't crash jobs
4. **Test Edge Cases**: Consider jobs with no tasks, completed tasks, etc.

## Troubleshooting

### Common Issues

**Keyspace Not Updating**:
- Verify file actually changed (MD5 hash different)
- Check directory monitor is running
- Ensure job is in eligible state (pending/running/paused)

**Incorrect Effective Keyspace**:
- Verify multiplication_factor is set correctly
- Check if job uses rule splitting
- Review calculation for "missed" keyspace

**Potfile Not Updating Jobs**:
- Ensure manual refresh was triggered
- Check potfile staging has new entries
- Verify job references potfile wordlist

### Debug Logging

Enable debug logging to trace update flow:
```
DEBUG: Directory monitor detected change
DEBUG: Handling wordlist update, old: 1000000, new: 1100000
DEBUG: Updated job keyspace, effective: 1100000000
```

## Limitations

1. **No Retroactive Updates**: Already-dispatched work won't get new words/rules
2. **Forward Progress Only**: System doesn't track or compensate for missed combinations
3. **Manual Potfile Refresh**: Requires user action to trigger potfile updates
4. **File Lock Conflicts**: Rapid file changes might cause temporary update delays

## Future Enhancements

Potential improvements under consideration:

- **Deficit Tracking**: Optional mode to track missed combinations
- **Automatic Potfile Refresh**: Configurable automatic refresh intervals
- **Smart Chunking**: Re-chunk remaining work when files change significantly
- **Update History**: Track all keyspace changes for job audit trail
- **Predictive Updates**: Estimate impact before applying changes

## Summary

The Job Update System ensures KrakenHashes jobs remain accurate and efficient as resources change. By following a forward-only philosophy, it provides a balance between consistency for running tasks and adaptability for future work. Understanding this system helps explain why job keyspaces may change during execution and how the system maintains integrity without disrupting active cracking operations.
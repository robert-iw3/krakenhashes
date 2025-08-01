# KrakenHashes Chunking System

## Overview

KrakenHashes uses an intelligent chunking system to distribute password cracking workloads across multiple agents. This document explains how chunks are created, distributed, and tracked for different attack types.

## What is Chunking?

Chunking divides large password cracking jobs into smaller, manageable pieces that can be:
- Distributed across multiple agents for parallel processing
- Completed within a reasonable time frame (default: 20 minutes)
- Resumed if interrupted or failed
- Tracked for accurate progress reporting

## How Chunking Works

### Basic Chunking (No Rules)

For simple dictionary attacks without rules:
1. The system calculates the total keyspace (number of password candidates)
2. Based on agent benchmark speeds, it determines optimal chunk sizes
3. Each chunk processes a portion of the wordlist using hashcat's `--skip` and `--limit` parameters

**Example**: 
- Wordlist: 1,000,000 passwords
- Agent speed: 1,000,000 H/s
- Target chunk time: 1,200 seconds (20 minutes)
- Chunk size: 1,200,000,000 candidates
- Result: Single chunk processes entire wordlist

### Enhanced Chunking with Rules

When rules are applied, the effective keyspace multiplies:

**Effective Keyspace = Wordlist Size × Number of Rules**

For example:
- Wordlist: 1,000,000 passwords
- Rules: 1,000 rules
- Effective keyspace: 1,000,000,000 candidates

#### Rule Splitting

When a job with rules would take significantly longer than the target chunk time, KrakenHashes can split the rules:

1. **Detection**: If estimated time > 2× target chunk time
2. **Splitting**: Divides rules into smaller files
3. **Distribution**: Each agent receives full wordlist + partial rules
4. **Progress**: Tracks completion across all rule chunks

**Example**:
- Wordlist: 1,000,000 passwords
- Rules: 10,000 rules
- Agent speed: 1,000,000 H/s
- Without splitting: 10,000 seconds (2.8 hours) per chunk
- With splitting into 10 chunks: 1,000 rules each, ~1,000 seconds per chunk

### Combination Attacks

For combination attacks (-a 1), the effective keyspace is:

**Effective Keyspace = Wordlist1 Size × Wordlist2 Size**

The system tracks progress through the virtual keyspace while hashcat processes the first wordlist sequentially.

### Attack Mode Support

| Attack Mode | Description | Chunking Method |
|------------|-------------|-----------------|
| 0 (Straight) | Dictionary | Wordlist position + optional rule splitting |
| 1 (Combination) | Two wordlists | Virtual keyspace tracking |
| 3 (Brute-force) | Mask attack | Mask position chunking |
| 6 (Hybrid W+M) | Wordlist + Mask | Wordlist position chunking |
| 7 (Hybrid M+W) | Mask + Wordlist | Mask position chunking |
| 9 (Association) | Per-hash rules | Rule splitting when applicable |

## Progress Tracking

### Standard Progress
- Shows candidates tested vs total keyspace
- Updates in real-time via WebSocket
- Accurate percentage completion

### With Rule Multiplication
- Display format: "X / Y (×Z)" where Z is the multiplication factor
- Accounts for all rules across all chunks
- Aggregates progress from distributed rule chunks

### Progress Bar Visualization
The progress bar always shows:
- Green: Completed keyspace
- Gray: Remaining keyspace
- Percentage: Based on effective keyspace

## Configuration

Administrators can tune chunking behavior via system settings:

| Setting | Default | Description |
|---------|---------|-------------|
| `default_chunk_duration` | 1200s | Target time per chunk (20 minutes) |
| `chunk_fluctuation_percentage` | 20% | Threshold for merging final chunks |
| `rule_split_enabled` | true | Enable automatic rule splitting |
| `rule_split_threshold` | 2.0 | Time multiplier to trigger splitting |
| `rule_split_min_rules` | 100 | Minimum rules before considering split |

## Best Practices

### For Users
1. **Large Rule Files**: Will automatically split for better distribution
2. **Multiple Rule Files**: Multiplication is handled automatically
3. **Progress Monitoring**: Check effective keyspace in job details
4. **Benchmarks**: Ensure agents have current benchmarks for accurate chunking

### For Administrators
1. **Chunk Duration**: Balance between progress granularity and overhead
2. **Rule Splitting**: Monitor temp directory space for large rule files
3. **Benchmarks**: Configure benchmark validity period appropriately
4. **Resource Usage**: Rule splitting creates temporary files

## Troubleshooting

### Slow Progress
- Check if effective keyspace is much larger than expected
- Verify agent benchmarks are current
- Consider enabling rule splitting if disabled

### Uneven Distribution
- Some chunks may be larger due to:
  - Fluctuation threshold preventing small final chunks
  - Rule count not evenly divisible
  - Different agent speeds

### Rule Splitting Not Occurring
Verify:
- `rule_split_enabled` is true
- Rule file has > `rule_split_min_rules` rules
- Estimated time exceeds threshold

## Technical Details

### Keyspace Calculation

```
Attack Mode 0 (Dictionary):
- Without rules: wordlist_size
- With rules: wordlist_size × total_rule_count

Attack Mode 1 (Combination):
- Always: wordlist1_size × wordlist2_size

Attack Mode 3 (Brute-force):
- Calculated from mask: charset_size^length

Attack Mode 6/7 (Hybrid):
- Wordlist_size × mask_keyspace
```

### Chunk Assignment

1. Agent requests work
2. System calculates optimal chunk size based on:
   - Agent's benchmark speed
   - Target chunk duration
   - Remaining keyspace
3. Chunk boundaries determined:
   - Start position (skip)
   - Chunk size (limit)
4. Agent receives chunk assignment
5. Progress tracked and aggregated

### Rule Chunk Files

When rule splitting is active:
- Temporary files created in configured directory
- Named: `job_[ID]_chunk_[N].rule`
- Automatically cleaned up after job completion
- Synced to agents like normal rule files

## Future Enhancements

- Pre-calculation of optimal chunk distribution
- Dynamic chunk resizing based on actual speed
- Rule deduplication before splitting
- Compression for rule chunk transfers
# Understanding Jobs and Workflows

## Quick Overview

KrakenHashes uses a two-tier system to organize password cracking attacks:

1. **Preset Jobs**: Individual attack configurations (like a single recipe)
2. **Job Workflows**: Collections of preset jobs that run in sequence (like a cookbook)

This system ensures consistent, efficient password auditing across your organization.

## How It Works

### The Password Cracking Process

When you submit a hashlist for cracking, KrakenHashes can apply a workflow that:
1. Tries the most likely passwords first (common passwords)
2. Progressively tries more complex attacks
3. Ensures no time is wasted on inefficient approaches

### Example Workflow in Action

Imagine you're auditing passwords for a company. A typical workflow might:

1. **First**: Check against known leaked passwords (fast, high success rate)
2. **Next**: Try common passwords with variations (password1, Password123!)
3. **Then**: Combine company terms with numbers (CompanyName2024)
4. **Finally**: Attempt more exhaustive searches if needed

<screenshot: Visual workflow diagram>

## Benefits

### Consistency
- Every password audit follows the same proven methodology
- No steps are accidentally skipped
- New team members can run expert-level audits immediately

### Efficiency
- Fast attacks run first, finding easy passwords quickly
- Resource-intensive attacks only run when necessary
- Priority system ensures optimal resource usage

### Customization
- Different workflows for different scenarios:
  - Quick compliance checks
  - Thorough security audits  
  - Industry-specific patterns
  - Post-breach assessments

## Common Attack Types

### Dictionary Attacks
Uses lists of known passwords:
- Common passwords (password, 123456)
- Previously leaked passwords
- Industry-specific terms

### Rule-Based Attacks
Applies transformations to dictionary words:
- Capitalize first letter
- Add numbers at the end
- Replace letters with symbols (@ for a, 3 for e)

### Hybrid Attacks
Combines dictionaries with patterns:
- Dictionary word + 4 digits (password2024)
- Year + dictionary word (2024password)

### Brute Force
Tries all possible combinations for a pattern:
- All 4-digit PINs (0000-9999)
- All 6-character lowercase (aaaaaa-zzzzzz)

## Understanding Priorities

Jobs within workflows run in priority order:
- **High Priority (80-100)**: Quick, likely-to-succeed attacks
- **Medium Priority (40-79)**: Balanced attacks
- **Low Priority (0-39)**: Exhaustive, time-consuming attacks

<screenshot: Priority visualization>

## Real-World Applications

### Compliance Auditing
Verify passwords meet policy requirements:
- Minimum length checks
- Complexity requirements
- Banned password lists

### Security Assessments
Identify weak passwords before attackers do:
- Test against current attack techniques
- Benchmark password strength
- Provide actionable reports

### Incident Response
Quickly assess breach impact:
- Check if compromised passwords are reused
- Identify affected accounts
- Prioritize password resets

## What This Means for You

As a user, the preset jobs and workflows system:
- Ensures thorough password testing
- Provides consistent results
- Optimizes cracking time
- Delivers actionable insights

Your administrators have configured these workflows based on:
- Industry best practices
- Your organization's specific needs
- Current threat landscape
- Compliance requirements

## Next Steps

- Review your hashlist results to understand which attacks succeeded
- Work with your security team to address found passwords
- Consider implementing stronger password policies
- Schedule regular password audits using these workflows

For more detailed information about creating and managing workflows, see the [administrator documentation](../admin/preset_jobs_and_workflows.md).
# Rules Management

KrakenHashes provides comprehensive management of rule files used for password cracking operations. This document explains how the system handles rule files and provides best practices for administrators.

## Overview

Rules are transformation patterns applied to wordlists during password cracking operations. They allow you to generate password variations without storing massive wordlists. For example, a single rule can transform "password" into "Password123!", "p@ssw0rd", and many other variations.

![Rule Management Interface](../../assets/images/screenshots/rule_management.png)
*The Rule Management interface displaying uploaded rule files with status, type, size, and rule count information. The interface provides filtering options and actions for managing Hashcat and John the Ripper rule files.*

## Directory Structure

The system monitors the rules directory and automatically creates the following subdirectory structure:

```
rules/
├── hashcat/       # Hashcat-compatible rules
├── john/          # John the Ripper rules
└── custom/        # User-created or modified rules
```

## Rule File Formats and Best Practices

### Hashcat Rules

Hashcat rules use a simple syntax where each character represents an operation:

```
# Example hashcat rules
:                    # Try the original word
l                    # Convert to lowercase
u                    # Convert to uppercase
c                    # Capitalize first letter, lowercase rest
$1                   # Append '1' to the word
^a                   # Prepend 'a' to the word
sa@                  # Replace all 'a' with '@'
$1$2$3               # Append '123'
c$!                  # Capitalize and append '!'
```

**Best Practices for Hashcat Rules:**
- Use comments (lines starting with #) to document complex rules
- Group related rules together
- Test rules with `--stdout` before using in production
- Keep rule files focused on specific transformation types

### John the Ripper Rules

John rules have a similar but slightly different syntax:

```
# Example John the Ripper rules
:                    # Try the original word
[lL]                 # Convert to lowercase
[uU]                 # Convert to uppercase
[cC]                 # Capitalize
$[0-9]               # Append a digit
^[aA]                # Prepend 'a' or 'A'
```

### Custom Rules

Custom rules should follow either Hashcat or John syntax depending on your cracking engine.

## Rule Type Detection

The system automatically assigns a rule type based on the file path:
- If the path contains "john" (case-insensitive), it's classified as John the Ripper
- Otherwise, it's classified as Hashcat
- Administrators can change the type after import if needed

## Rule Splitting for Large Files

KrakenHashes includes an intelligent rule splitting system for distributing work across multiple agents:

### Automatic Rule Splitting

When rule splitting is enabled, the system can automatically split large rule files:

**Configuration Settings:**
- `rule_split_enabled`: Enable/disable automatic rule splitting
- `rule_split_threshold`: Threshold ratio for triggering splits (default: 0.8)
- `rule_split_min_rules`: Minimum number of rules before splitting (default: 10,000)
- `rule_split_max_chunks`: Maximum number of chunks to create (default: 100)

### How Rule Splitting Works

1. **Detection**: When a job uses a rule file with more rules than the threshold, splitting is triggered
2. **Chunking**: The rule file is divided into equal chunks based on available agents
3. **Distribution**: Each agent receives a chunk of rules to process
4. **Cleanup**: Temporary chunk files are removed after job completion

### Manual Rule Splitting

For optimal performance, you can pre-split large rule files:

```bash
# Split a rule file into 10 parts
split -n 10 large_rules.rule rules_part_

# Split by number of lines (1000 rules per file)
split -l 1000 large_rules.rule rules_chunk_
```

## Performance Considerations

### Rule Complexity

Different rule types have varying performance impacts:

1. **Simple Rules** (minimal impact):
   - Case changes (l, u, c)
   - Single character operations ($x, ^x)
   
2. **Moderate Rules** (noticeable impact):
   - Multiple substitutions (sa@sb$sc()
   - Positional operations
   
3. **Complex Rules** (significant impact):
   - Multiple operations per rule
   - Conditional rules
   - Memory operations

### Optimization Tips

1. **Order Rules by Frequency**: Place most likely successful rules first
2. **Avoid Redundancy**: Remove duplicate or overlapping rules
3. **Benchmark First**: Test rule performance with small wordlists
4. **Use Rule Splitting**: For rules >10,000 lines, enable splitting
5. **Monitor Memory**: Complex rules can increase memory usage

## Common Rule Sets and Their Uses

### Basic Password Variations
```
# basic_variations.rule
:                    # Original
c                    # Capitalize
u                    # Uppercase
l                    # Lowercase
c$1                  # Capitalize + append 1
c$!                  # Capitalize + append !
$2023                # Append year
$2024                # Append year
```

### Leetspeak Transformations
```
# leetspeak.rule
sa@                  # a -> @
se3                  # e -> 3
si1                  # i -> 1
so0                  # o -> 0
ss$                  # s -> $
sa@se3              # Multiple substitutions
```

### Corporate Passwords
```
# corporate.rule
c$1$2$3              # Capitalize + 123
c$!$@$#              # Capitalize + special chars
$@company            # Append company name
^Company             # Prepend company name
c$2023               # Capitalize + year
```

### Keyboard Patterns
```
# keyboard_patterns.rule
$!@#                 # Common keyboard pattern
$123                 # Sequential numbers
$qwe                 # Keyboard row
$!qaz                # Vertical keyboard pattern
```

## Creating Custom Rules

### Rule Development Workflow

1. **Analyze Target Patterns**: Study password patterns from previous cracks
2. **Write Initial Rules**: Create rules based on observed patterns
3. **Test with Hashcat**: Use `--stdout` to verify transformations
4. **Refine and Optimize**: Remove ineffective rules, add variations
5. **Document**: Add comments explaining rule purpose

### Example: Creating Domain-Specific Rules

```bash
# finance_sector.rule
# Common patterns in financial sector
$2023                # Current year
$Q1                  # Quarter notation
$USD                 # Currency
^FIN                 # Department prefix
sa@s$$               # Common substitutions
c$123                # Compliance requirement
```

### Testing Custom Rules

```bash
# Test rules with sample wordlist
hashcat --stdout wordlist.txt -r custom.rule | head -20

# Count generated candidates
hashcat --stdout wordlist.txt -r custom.rule | wc -l

# Verify specific transformations
echo "password" | hashcat --stdout -r custom.rule
```

## File Management

### Uploading Rules

When uploading rule files:

1. Choose appropriate rule type (Hashcat/John/Custom)
2. Add descriptive tags for organization
3. Include documentation in description field
4. For large files (>10MB), upload is processed asynchronously

### Automatic Processing

The system automatically:
- Calculates MD5 hash for integrity
- Counts rules (excluding comments and empty lines)
- Verifies file accessibility
- Tags auto-imported files

### Duplicate Handling

- Same filename + same content = Skip upload
- Same filename + different content = Update existing
- Different filename + same content = Create new entry

## Best Practices

### Organization
1. **Categorize by Purpose**: Use subdirectories for different rule types
2. **Version Control**: Include version numbers in filenames
3. **Documentation**: Include README files explaining rule sets

### Naming Conventions
```
# Good naming examples
basic_english_v2.rule
corporate_2024Q1.rule
web_app_patterns.rule
finance_sector_specific.rule

# Avoid
rules1.txt
new.rule
test.rule
```

### Maintenance
1. **Regular Reviews**: Audit rule effectiveness quarterly
2. **Update Patterns**: Add new patterns as they emerge
3. **Remove Obsolete**: Delete rules for outdated patterns
4. **Benchmark Performance**: Test rule speed on new hardware

## Monitoring and Troubleshooting

### Import Status

Check rule import status through:
- Admin dashboard for overview
- Server logs for detailed processing info
- Database `verification_status` field

### Common Issues

1. **Large File Processing**: Files >1GB may take time to verify
2. **Rule Syntax Errors**: Invalid rules are skipped during counting
3. **File Access**: Ensure proper permissions on rule directories

### Performance Metrics

Monitor rule performance:
- Rules/second processing rate
- Memory usage during rule application
- Success rate (cracks per rule application)

## Security Considerations

1. **Access Control**: Limit rule file access to authorized users
2. **Validation**: System validates rule syntax during import
3. **Audit Trail**: All rule modifications are logged
4. **Sensitive Patterns**: Avoid hardcoding sensitive data in rules

## Integration with Jobs

Rules are selected during job creation:
1. Browse available verified rules
2. Select appropriate rule for attack type
3. System handles rule distribution to agents
4. Progress tracking shows rules processed

For optimal performance, match rule complexity to available computational resources and expected password patterns.
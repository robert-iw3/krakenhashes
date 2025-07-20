# Agent Scheduling Quick Reference

## Enable Scheduling

### 1. Enable Globally (Admin)
**Admin Panel → System Settings → Enable Agent Scheduling System**

### 2. Enable for Agent
**Agents → [Agent Name] → Scheduling → Enable Scheduling**

### 3. Configure Schedule
**Click "Edit All Schedules"**

## Time Format Examples

| Input | Stored As | Displays As |
|-------|-----------|-------------|
| `9` | `09:00:00` | `09:00` |
| `17` | `17:00:00` | `17:00` |
| `9:30` | `09:30:00` | `09:30` |
| `09:00` | `09:00:00` | `09:00` |

## Schedule Examples

### Business Hours (9-5)
```
Mon-Fri: 09:00 - 17:00 [Active]
Sat-Sun: Not scheduled
```

### Night Shift
```
Mon-Fri: 22:00 - 06:00 [Active]
Sat-Sun: Not scheduled
```

### 24/7 Except Maintenance
```
Mon-Sat: 00:00 - 23:59 [Active]
Sun: 00:00 - 06:00 [Active]  # Maintenance 6 AM - Midnight
```

## Active Toggle

- **✅ Active ON** = Agent works during specified hours
- **❌ Active OFF** = Agent doesn't work at all that day

## Priority Rules

1. **Global OFF** → All schedules ignored
2. **Global ON + Agent OFF** → Agent always available
3. **Global ON + Agent ON** → Schedule enforced

## Running Jobs

⚠️ **Important**: Running jobs are NOT interrupted by schedule end times
- Jobs assigned at 16:59 will complete even if schedule ends at 17:00
- Agent won't accept NEW jobs after schedule ends
- Current work always finishes

## Timezone Notes

- **Display**: Your local timezone
- **Storage**: UTC in database
- **Overnight**: Supported (e.g., 22:00 - 02:00)

## Troubleshooting

**Agent not getting jobs?**
1. ✓ Global scheduling enabled?
2. ✓ Agent scheduling enabled?
3. ✓ Current time within schedule?
4. ✓ Agent online and enabled?

**Wrong times showing?**
- Check browser timezone
- All times stored in UTC
- Displayed in local time
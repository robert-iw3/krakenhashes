import React, { useState, useEffect } from 'react';
import {
  Box,
  Paper,
  Typography,
  Switch,
  FormControlLabel,
  Grid,
  TextField,
  Button,
  IconButton,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Alert,
  Tooltip,
} from '@mui/material';
import {
  Edit as EditIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  ContentCopy as CopyIcon,
} from '@mui/icons-material';
import {
  convertLocalTimeToUTC,
  convertUTCTimeToLocal,
  getUserTimezone,
  getTimezoneAbbreviation,
  getUTCOffset,
  getDaysOfWeek,
  isOvernightSchedule,
  getDefaultWorkingHours,
} from '../../utils/timezone';
import { AgentSchedule, AgentScheduleDTO } from '../../types/scheduling';
import { getSystemSetting } from '../../services/systemSettings';

interface AgentSchedulingProps {
  agentId: number;
  schedulingEnabled: boolean;
  scheduleTimezone: string;
  schedules: AgentSchedule[];
  onToggleScheduling: (enabled: boolean, timezone: string) => Promise<void>;
  onUpdateSchedules: (schedules: AgentScheduleDTO[]) => Promise<void>;
  onDeleteSchedule: (dayOfWeek: number) => Promise<void>;
}

const AgentScheduling: React.FC<AgentSchedulingProps> = ({
  agentId,
  schedulingEnabled,
  scheduleTimezone,
  schedules,
  onToggleScheduling,
  onUpdateSchedules,
  onDeleteSchedule,
}) => {
  const [isEditDialogOpen, setIsEditDialogOpen] = useState(false);
  const [editingSchedules, setEditingSchedules] = useState<Map<number, AgentSchedule>>(new Map());
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [globalSchedulingEnabled, setGlobalSchedulingEnabled] = useState(true);
  const [loadingGlobalSetting, setLoadingGlobalSetting] = useState(true);
  const daysOfWeek = getDaysOfWeek();
  const userTimezone = getUserTimezone();
  const timezoneDisplay = `${getTimezoneAbbreviation()} (${getUTCOffset()})`;

  // Fetch global scheduling setting
  useEffect(() => {
    const fetchGlobalSetting = async () => {
      try {
        const setting = await getSystemSetting('agent_scheduling_enabled');
        setGlobalSchedulingEnabled(setting.value === 'true');
      } catch (err) {
        console.error('Failed to fetch global scheduling setting:', err);
        // Default to true if we can't fetch the setting
        setGlobalSchedulingEnabled(true);
      } finally {
        setLoadingGlobalSetting(false);
      }
    };
    fetchGlobalSetting();
  }, []);

  // Initialize editing schedules when dialog opens
  useEffect(() => {
    if (isEditDialogOpen && schedules) {
      const scheduleMap = new Map<number, AgentSchedule>();
      schedules.forEach(schedule => {
        // Convert UTC times to local times for display
        const localSchedule: AgentSchedule = {
          ...schedule,
          startTime: convertUTCTimeToLocal(schedule.startTime, schedule.dayOfWeek),
          endTime: convertUTCTimeToLocal(schedule.endTime, schedule.dayOfWeek),
        };
        scheduleMap.set(schedule.dayOfWeek, localSchedule);
      });
      setEditingSchedules(scheduleMap);
    }
  }, [isEditDialogOpen, schedules]);

  const handleToggleScheduling = async () => {
    if (!globalSchedulingEnabled) {
      setError('Global scheduling is disabled by administrator');
      return;
    }
    
    try {
      await onToggleScheduling(!schedulingEnabled, userTimezone);
    } catch (err) {
      console.error('Failed to toggle scheduling:', err);
    }
  };

  const handleSaveSchedules = async () => {
    setSaving(true);
    setError('');

    try {
      // Convert local times to UTC before sending
      const scheduleDTOs: AgentScheduleDTO[] = [];
      
      editingSchedules.forEach((schedule, dayOfWeek) => {
        if (schedule.startTime && schedule.endTime) {
          scheduleDTOs.push({
            dayOfWeek,
            startTimeUTC: convertLocalTimeToUTC(schedule.startTime, dayOfWeek),
            endTimeUTC: convertLocalTimeToUTC(schedule.endTime, dayOfWeek),
            timezone: userTimezone,
            isActive: schedule.isActive,
          });
        }
      });

      await onUpdateSchedules(scheduleDTOs);
      setIsEditDialogOpen(false);
    } catch (err) {
      setError('Failed to save schedules');
      console.error('Failed to save schedules:', err);
    } finally {
      setSaving(false);
    }
  };

  const handleTimeChange = (dayOfWeek: number, field: 'startTime' | 'endTime', value: string) => {
    const current = editingSchedules.get(dayOfWeek) || {
      agentId,
      dayOfWeek,
      startTime: '',
      endTime: '',
      timezone: userTimezone,
      isActive: true,
    };

    const updated = { ...current, [field]: value };
    setEditingSchedules(new Map(editingSchedules.set(dayOfWeek, updated)));
  };

  const handleDeleteSchedule = async (dayOfWeek: number) => {
    try {
      await onDeleteSchedule(dayOfWeek);
      // Remove from editing schedules if dialog is open
      if (isEditDialogOpen) {
        const newSchedules = new Map(editingSchedules);
        newSchedules.delete(dayOfWeek);
        setEditingSchedules(newSchedules);
      }
    } catch (err) {
      console.error('Failed to delete schedule:', err);
    }
  };

  const handleCopySchedule = (fromDay: number) => {
    const sourceSchedule = editingSchedules.get(fromDay);
    if (!sourceSchedule) return;

    // Copy to all other days
    const newSchedules = new Map(editingSchedules);
    for (let day = 0; day < 7; day++) {
      if (day !== fromDay) {
        newSchedules.set(day, {
          ...sourceSchedule,
          dayOfWeek: day,
        });
      }
    }
    setEditingSchedules(newSchedules);
  };

  const renderScheduleSummary = (dayOfWeek: number) => {
    const schedule = schedules?.find(s => s.dayOfWeek === dayOfWeek);
    if (!schedule) {
      return <Typography variant="body2" color="text.secondary">Not scheduled</Typography>;
    }

    // Convert UTC to local for display
    const localStart = convertUTCTimeToLocal(schedule.startTime, dayOfWeek);
    const localEnd = convertUTCTimeToLocal(schedule.endTime, dayOfWeek);
    const overnight = isOvernightSchedule(localStart, localEnd);

    return (
      <Box display="flex" alignItems="center" gap={1}>
        <Typography variant="body2">
          {localStart} - {localEnd}
          {overnight && <Chip size="small" label="Overnight" sx={{ ml: 1 }} />}
        </Typography>
        {!schedule.isActive && <Chip size="small" label="Inactive" color="default" />}
      </Box>
    );
  };

  return (
    <Paper sx={{ p: 3 }}>
      <Box display="flex" justifyContent="space-between" alignItems="center" mb={2}>
        <Typography variant="h6">Scheduling</Typography>
        <FormControlLabel
          control={
            <Switch
              checked={schedulingEnabled}
              onChange={handleToggleScheduling}
              color="primary"
              disabled={!globalSchedulingEnabled || loadingGlobalSetting}
            />
          }
          label="Enable Scheduling"
        />
      </Box>

      {!globalSchedulingEnabled && !loadingGlobalSetting && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          Agent scheduling is disabled by the administrator. Schedules are preserved but will not be enforced until the global setting is enabled.
        </Alert>
      )}

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {schedulingEnabled ? (
        <>
          {!globalSchedulingEnabled && (
            <Alert severity="info" sx={{ mb: 2 }}>
              Schedules are configured but not active because global scheduling is disabled.
            </Alert>
          )}
          
          <Typography variant="body2" color="text.secondary" mb={2}>
            Times shown in {timezoneDisplay}
          </Typography>

          <Grid container spacing={2}>
            {daysOfWeek.map(day => (
              <Grid item xs={12} key={day.value}>
                <Box display="flex" justifyContent="space-between" alignItems="center">
                  <Box flex={1}>
                    <Typography variant="subtitle2">{day.label}</Typography>
                    {renderScheduleSummary(day.value)}
                  </Box>
                  <IconButton
                    size="small"
                    onClick={() => setIsEditDialogOpen(true)}
                    color="primary"
                  >
                    <EditIcon />
                  </IconButton>
                </Box>
              </Grid>
            ))}
          </Grid>

          <Box mt={3}>
            <Button
              variant="outlined"
              fullWidth
              startIcon={<EditIcon />}
              onClick={() => setIsEditDialogOpen(true)}
            >
              Edit All Schedules
            </Button>
          </Box>
        </>
      ) : (
        <Alert severity="info">
          Agent is always available when scheduling is disabled
        </Alert>
      )}

      {/* Edit Dialog */}
      <Dialog
        open={isEditDialogOpen}
        onClose={() => setIsEditDialogOpen(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Edit Agent Schedule</DialogTitle>
        <DialogContent>
          {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
          
          <Typography variant="body2" color="text.secondary" mb={2}>
            Set daily schedules in your local time ({timezoneDisplay})
          </Typography>

          <Grid container spacing={2}>
            {daysOfWeek.map(day => {
              const schedule = editingSchedules.get(day.value);
              const hasSchedule = schedule?.startTime && schedule?.endTime;
              
              return (
                <Grid item xs={12} key={day.value}>
                  <Paper variant="outlined" sx={{ p: 2 }}>
                    <Box display="flex" alignItems="center" gap={2}>
                      <Typography variant="subtitle2" sx={{ minWidth: 100 }}>
                        {day.label}
                      </Typography>
                      
                      {hasSchedule ? (
                        <>
                          <TextField
                            label="Start Time"
                            value={schedule.startTime}
                            onChange={(e) => handleTimeChange(day.value, 'startTime', e.target.value)}
                            placeholder="HH:MM"
                            size="small"
                            sx={{ width: 120 }}
                          />
                          <Typography>-</Typography>
                          <TextField
                            label="End Time"
                            value={schedule.endTime}
                            onChange={(e) => handleTimeChange(day.value, 'endTime', e.target.value)}
                            placeholder="HH:MM"
                            size="small"
                            sx={{ width: 120 }}
                          />
                          <Tooltip 
                            title="When enabled, the agent will work during the specified hours on this day. When disabled, the agent will not work at all on this day."
                            placement="top"
                          >
                            <FormControlLabel
                              control={
                                <Switch
                                  checked={schedule.isActive}
                                  onChange={(e) => {
                                    const updated = { ...schedule, isActive: e.target.checked };
                                    setEditingSchedules(new Map(editingSchedules.set(day.value, updated)));
                                  }}
                                  size="small"
                                />
                              }
                              label="Active"
                            />
                          </Tooltip>
                          <IconButton
                            size="small"
                            onClick={() => handleCopySchedule(day.value)}
                            title="Copy to other days"
                          >
                            <CopyIcon />
                          </IconButton>
                          <IconButton
                            size="small"
                            onClick={() => {
                              const newSchedules = new Map(editingSchedules);
                              newSchedules.delete(day.value);
                              setEditingSchedules(newSchedules);
                            }}
                            color="error"
                          >
                            <DeleteIcon />
                          </IconButton>
                        </>
                      ) : (
                        <Button
                          variant="outlined"
                          size="small"
                          startIcon={<AddIcon />}
                          onClick={() => {
                            const defaultHours = getDefaultWorkingHours();
                            handleTimeChange(day.value, 'startTime', defaultHours.start);
                            handleTimeChange(day.value, 'endTime', defaultHours.end);
                          }}
                        >
                          Add Schedule
                        </Button>
                      )}
                    </Box>
                    
                    {hasSchedule && isOvernightSchedule(schedule.startTime, schedule.endTime) && (
                      <Alert severity="info" sx={{ mt: 1 }}>
                        This schedule spans midnight
                      </Alert>
                    )}
                  </Paper>
                </Grid>
              );
            })}
          </Grid>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setIsEditDialogOpen(false)}>Cancel</Button>
          <Button
            onClick={handleSaveSchedules}
            variant="contained"
            disabled={saving}
          >
            {saving ? 'Saving...' : 'Save Changes'}
          </Button>
        </DialogActions>
      </Dialog>
    </Paper>
  );
};

export default AgentScheduling;
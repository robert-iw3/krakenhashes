package models

import (
	"database/sql"
	"time"
)

// AgentSchedule represents a daily schedule for an agent
type AgentSchedule struct {
	ID        int       `json:"id"`
	AgentID   int       `json:"agentId"`
	DayOfWeek int       `json:"dayOfWeek"` // 0-6 (Sunday-Saturday)
	StartTime TimeOnly  `json:"startTime"` // HH:MM in UTC
	EndTime   TimeOnly  `json:"endTime"`   // HH:MM in UTC
	Timezone  string    `json:"timezone"`  // Original timezone for reference
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// AgentScheduleDTO represents the data transfer object for schedule updates from frontend
type AgentScheduleDTO struct {
	DayOfWeek    int    `json:"dayOfWeek"`
	StartTimeUTC string `json:"startTimeUTC"` // HH:MM in UTC
	EndTimeUTC   string `json:"endTimeUTC"`   // HH:MM in UTC
	Timezone     string `json:"timezone"`     // User's timezone for reference
	IsActive     bool   `json:"isActive"`
}

// IsScheduledNow checks if the current UTC time falls within the schedule
func (s *AgentSchedule) IsScheduledNow() bool {
	if !s.IsActive {
		return false
	}

	now := time.Now().UTC()
	currentDay := int(now.Weekday())

	// Check if it's the right day
	if currentDay != s.DayOfWeek {
		return false
	}

	// Create current time as TimeOnly
	currentTime := TimeOnly{
		Hours:   now.Hour(),
		Minutes: now.Minute(),
		Seconds: now.Second(),
	}

	// Handle overnight schedules (e.g., 22:00 - 02:00)
	if s.EndTime.Before(s.StartTime) {
		// Schedule spans midnight
		return currentTime.After(s.StartTime) || currentTime.Before(s.EndTime)
	}

	// Normal schedule
	return currentTime.After(s.StartTime) && currentTime.Before(s.EndTime)
}

// ValidateSchedule checks if the schedule is valid
func (s *AgentSchedule) ValidateSchedule() error {
	if s.DayOfWeek < 0 || s.DayOfWeek > 6 {
		return ErrInvalidInput
	}

	// Times must be different
	if s.StartTime.Equal(s.EndTime) {
		return ErrInvalidInput
	}

	return nil
}

// DayOfWeekName returns the name of the day
func (s *AgentSchedule) DayOfWeekName() string {
	days := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	if s.DayOfWeek >= 0 && s.DayOfWeek < 7 {
		return days[s.DayOfWeek]
	}
	return "Unknown"
}

// ScanSchedules helps scan multiple schedules from database rows
func ScanSchedules(rows *sql.Rows) ([]AgentSchedule, error) {
	var schedules []AgentSchedule

	for rows.Next() {
		var s AgentSchedule
		err := rows.Scan(
			&s.ID,
			&s.AgentID,
			&s.DayOfWeek,
			&s.StartTime,
			&s.EndTime,
			&s.Timezone,
			&s.IsActive,
			&s.CreatedAt,
			&s.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, s)
	}

	return schedules, nil
}
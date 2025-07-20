package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TimeOnly represents a time without date component (HH:MM:SS)
// It handles conversion between PostgreSQL TIME type and Go string
type TimeOnly struct {
	Hours   int
	Minutes int
	Seconds int
}

// NewTimeOnly creates a TimeOnly from hours, minutes, and seconds
func NewTimeOnly(hours, minutes, seconds int) (TimeOnly, error) {
	if hours < 0 || hours > 23 {
		return TimeOnly{}, fmt.Errorf("hours must be between 0 and 23, got %d", hours)
	}
	if minutes < 0 || minutes > 59 {
		return TimeOnly{}, fmt.Errorf("minutes must be between 0 and 59, got %d", minutes)
	}
	if seconds < 0 || seconds > 59 {
		return TimeOnly{}, fmt.Errorf("seconds must be between 0 and 59, got %d", seconds)
	}
	return TimeOnly{Hours: hours, Minutes: minutes, Seconds: seconds}, nil
}

// ParseTimeOnly parses various time formats into TimeOnly
// Accepts: "9", "17", "9:30", "09:00", "09:00:00"
func ParseTimeOnly(s string) (TimeOnly, error) {
	// Remove whitespace
	s = strings.TrimSpace(s)
	if s == "" {
		return TimeOnly{}, fmt.Errorf("empty time string")
	}

	// Handle single number (hour only)
	if matched, _ := regexp.MatchString(`^\d{1,2}$`, s); matched {
		hours, err := strconv.Atoi(s)
		if err != nil {
			return TimeOnly{}, err
		}
		return NewTimeOnly(hours, 0, 0)
	}

	// Handle HH:MM format
	if matched, _ := regexp.MatchString(`^\d{1,2}:\d{1,2}$`, s); matched {
		parts := strings.Split(s, ":")
		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return TimeOnly{}, fmt.Errorf("invalid hours: %v", err)
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return TimeOnly{}, fmt.Errorf("invalid minutes: %v", err)
		}
		return NewTimeOnly(hours, minutes, 0)
	}

	// Handle HH:MM:SS format
	if matched, _ := regexp.MatchString(`^\d{1,2}:\d{1,2}:\d{1,2}$`, s); matched {
		parts := strings.Split(s, ":")
		hours, err := strconv.Atoi(parts[0])
		if err != nil {
			return TimeOnly{}, fmt.Errorf("invalid hours: %v", err)
		}
		minutes, err := strconv.Atoi(parts[1])
		if err != nil {
			return TimeOnly{}, fmt.Errorf("invalid minutes: %v", err)
		}
		seconds, err := strconv.Atoi(parts[2])
		if err != nil {
			return TimeOnly{}, fmt.Errorf("invalid seconds: %v", err)
		}
		return NewTimeOnly(hours, minutes, seconds)
	}

	return TimeOnly{}, fmt.Errorf("invalid time format: %s", s)
}

// String returns the time in HH:MM:SS format
func (t TimeOnly) String() string {
	return fmt.Sprintf("%02d:%02d:%02d", t.Hours, t.Minutes, t.Seconds)
}

// StringHHMM returns the time in HH:MM format (no seconds)
func (t TimeOnly) StringHHMM() string {
	return fmt.Sprintf("%02d:%02d", t.Hours, t.Minutes)
}

// Scan implements sql.Scanner interface for database scanning
func (t *TimeOnly) Scan(value interface{}) error {
	if value == nil {
		*t = TimeOnly{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		// PostgreSQL TIME type is often scanned as time.Time
		*t = TimeOnly{
			Hours:   v.Hour(),
			Minutes: v.Minute(),
			Seconds: v.Second(),
		}
		return nil
	case []byte:
		// Sometimes comes as byte array
		return t.Scan(string(v))
	case string:
		// Parse the string format
		parsed, err := ParseTimeOnly(v)
		if err != nil {
			return fmt.Errorf("cannot scan time: %v", err)
		}
		*t = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into TimeOnly", value)
	}
}

// Value implements driver.Valuer interface for database storage
func (t TimeOnly) Value() (driver.Value, error) {
	// Return as string in HH:MM:SS format for PostgreSQL TIME type
	return t.String(), nil
}

// MarshalJSON implements json.Marshaler interface
func (t TimeOnly) MarshalJSON() ([]byte, error) {
	// Return as HH:MM format for frontend compatibility
	return json.Marshal(t.StringHHMM())
}

// UnmarshalJSON implements json.Unmarshaler interface
func (t *TimeOnly) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	
	parsed, err := ParseTimeOnly(s)
	if err != nil {
		return err
	}
	
	*t = parsed
	return nil
}

// IsZero returns true if the time is 00:00:00
func (t TimeOnly) IsZero() bool {
	return t.Hours == 0 && t.Minutes == 0 && t.Seconds == 0
}

// Before returns true if t is before other
func (t TimeOnly) Before(other TimeOnly) bool {
	if t.Hours != other.Hours {
		return t.Hours < other.Hours
	}
	if t.Minutes != other.Minutes {
		return t.Minutes < other.Minutes
	}
	return t.Seconds < other.Seconds
}

// After returns true if t is after other
func (t TimeOnly) After(other TimeOnly) bool {
	return other.Before(t)
}

// Equal returns true if t equals other
func (t TimeOnly) Equal(other TimeOnly) bool {
	return t.Hours == other.Hours && t.Minutes == other.Minutes && t.Seconds == other.Seconds
}

// TotalMinutes returns the total number of minutes since midnight
func (t TimeOnly) TotalMinutes() int {
	return t.Hours*60 + t.Minutes
}
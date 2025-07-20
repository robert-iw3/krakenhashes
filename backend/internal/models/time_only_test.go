package models

import (
	"encoding/json"
	"testing"
)

func TestParseTimeOnly(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TimeOnly
		wantErr bool
	}{
		{
			name:  "Hour only - single digit",
			input: "9",
			want:  TimeOnly{Hours: 9, Minutes: 0, Seconds: 0},
		},
		{
			name:  "Hour only - double digit",
			input: "17",
			want:  TimeOnly{Hours: 17, Minutes: 0, Seconds: 0},
		},
		{
			name:  "Hour and minutes - short format",
			input: "9:30",
			want:  TimeOnly{Hours: 9, Minutes: 30, Seconds: 0},
		},
		{
			name:  "Hour and minutes - full format",
			input: "09:00",
			want:  TimeOnly{Hours: 9, Minutes: 0, Seconds: 0},
		},
		{
			name:  "Full time with seconds",
			input: "09:00:00",
			want:  TimeOnly{Hours: 9, Minutes: 0, Seconds: 0},
		},
		{
			name:  "Late evening time",
			input: "23:59",
			want:  TimeOnly{Hours: 23, Minutes: 59, Seconds: 0},
		},
		{
			name:  "Midnight",
			input: "00:00",
			want:  TimeOnly{Hours: 0, Minutes: 0, Seconds: 0},
		},
		{
			name:    "Invalid hour",
			input:   "25",
			wantErr: true,
		},
		{
			name:    "Invalid minutes",
			input:   "9:75",
			wantErr: true,
		},
		{
			name:    "Empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "Invalid format",
			input:   "abc",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTimeOnly(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTimeOnly() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseTimeOnly() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeOnlyJSON(t *testing.T) {
	tests := []struct {
		name  string
		time  TimeOnly
		want  string
	}{
		{
			name: "Morning time",
			time: TimeOnly{Hours: 9, Minutes: 0, Seconds: 0},
			want: `"09:00"`,
		},
		{
			name: "Afternoon time",
			time: TimeOnly{Hours: 17, Minutes: 30, Seconds: 45},
			want: `"17:30"`,
		},
		{
			name: "Midnight",
			time: TimeOnly{Hours: 0, Minutes: 0, Seconds: 0},
			want: `"00:00"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.time)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("json.Marshal() = %s, want %s", got, tt.want)
			}

			// Test unmarshaling
			var unmarshaled TimeOnly
			if err := json.Unmarshal(got, &unmarshaled); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			// When unmarshaling from HH:MM format, seconds should be 0
			expected := TimeOnly{Hours: tt.time.Hours, Minutes: tt.time.Minutes, Seconds: 0}
			if unmarshaled != expected {
				t.Errorf("json.Unmarshal() = %v, want %v", unmarshaled, expected)
			}
		})
	}
}

func TestTimeOnlyString(t *testing.T) {
	tests := []struct {
		name  string
		time  TimeOnly
		want  string
	}{
		{
			name: "Full format",
			time: TimeOnly{Hours: 9, Minutes: 30, Seconds: 45},
			want: "09:30:45",
		},
		{
			name: "Zero padded",
			time: TimeOnly{Hours: 1, Minutes: 5, Seconds: 9},
			want: "01:05:09",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.time.String(); got != tt.want {
				t.Errorf("TimeOnly.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
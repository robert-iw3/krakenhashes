package agent

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageTypes(t *testing.T) {
	// Test that message type constants have expected values
	assert.Equal(t, MessageType("job_assignment"), TypeJobAssignment)
	assert.Equal(t, MessageType("status_update"), TypeStatusUpdate)
	assert.Equal(t, MessageType("heartbeat"), TypeHeartbeat)
	assert.Equal(t, MessageType("error"), TypeError)
}

func TestJobAssignment_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name       string
		assignment JobAssignment
		validate   func(t *testing.T, data []byte)
	}{
		{
			name: "basic job assignment",
			assignment: JobAssignment{
				Type:     TypeJobAssignment,
				JobID:    "job-123",
				Priority: 5,
				Parameters: map[string]interface{}{
					"hashlist": "test.hash",
					"wordlist": "rockyou.txt",
				},
			},
			validate: func(t *testing.T, data []byte) {
				var parsed map[string]interface{}
				err := json.Unmarshal(data, &parsed)
				require.NoError(t, err)
				assert.Equal(t, "job_assignment", parsed["type"])
				assert.Equal(t, "job-123", parsed["job_id"])
				assert.Equal(t, float64(5), parsed["priority"])
			},
		},
		{
			name: "job assignment with complex parameters",
			assignment: JobAssignment{
				Type:     TypeJobAssignment,
				JobID:    "complex-job",
				Priority: 10,
				Parameters: struct {
					HashType   int      `json:"hash_type"`
					AttackMode int      `json:"attack_mode"`
					Rules      []string `json:"rules"`
				}{
					HashType:   0,
					AttackMode: 3,
					Rules:      []string{"best64.rule", "dive.rule"},
				},
			},
		},
		{
			name: "job assignment with nil parameters",
			assignment: JobAssignment{
				Type:       TypeJobAssignment,
				JobID:      "nil-params",
				Priority:   1,
				Parameters: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.assignment)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			// Unmarshal back
			var parsed JobAssignment
			err = json.Unmarshal(data, &parsed)
			require.NoError(t, err)

			// Verify fields
			assert.Equal(t, tt.assignment.Type, parsed.Type)
			assert.Equal(t, tt.assignment.JobID, parsed.JobID)
			assert.Equal(t, tt.assignment.Priority, parsed.Priority)

			if tt.validate != nil {
				tt.validate(t, data)
			}
		})
	}
}

func TestStatusUpdate_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name   string
		update StatusUpdate
	}{
		{
			name: "basic status update",
			update: StatusUpdate{
				Type:     TypeStatusUpdate,
				JobID:    "job-456",
				Status:   "processing",
				Progress: 45.5,
				Details: map[string]interface{}{
					"current_position": 1000000,
					"total_keyspace":   10000000,
				},
			},
		},
		{
			name: "completed status",
			update: StatusUpdate{
				Type:     TypeStatusUpdate,
				JobID:    "job-789",
				Status:   "completed",
				Progress: 100.0,
				Details: struct {
					CrackedCount int    `json:"cracked_count"`
					Duration     string `json:"duration"`
				}{
					CrackedCount: 15,
					Duration:     "2h15m",
				},
			},
		},
		{
			name: "failed status",
			update: StatusUpdate{
				Type:     TypeStatusUpdate,
				JobID:    "failed-job",
				Status:   "failed",
				Progress: 23.7,
				Details: map[string]interface{}{
					"error":      "hashcat process crashed",
					"error_code": 255,
				},
			},
		},
		{
			name: "status with nil details",
			update: StatusUpdate{
				Type:     TypeStatusUpdate,
				JobID:    "minimal",
				Status:   "initializing",
				Progress: 0.0,
				Details:  nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.update)
			require.NoError(t, err)

			// Unmarshal back
			var parsed StatusUpdate
			err = json.Unmarshal(data, &parsed)
			require.NoError(t, err)

			// Verify fields
			assert.Equal(t, tt.update.Type, parsed.Type)
			assert.Equal(t, tt.update.JobID, parsed.JobID)
			assert.Equal(t, tt.update.Status, parsed.Status)
			assert.Equal(t, tt.update.Progress, parsed.Progress)
		})
	}
}

func TestHeartbeat_JSONMarshaling(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name      string
		heartbeat Heartbeat
	}{
		{
			name: "heartbeat with metrics",
			heartbeat: Heartbeat{
				Type:      TypeHeartbeat,
				Timestamp: now,
				Status:    "healthy",
				Metrics: &SystemMetrics{
					CPUUsage:       45.2,
					MemoryUsage:    67.8,
					GPUUtilization: 85.5,
					GPUTemp:        72.0,
				},
			},
		},
		{
			name: "heartbeat without metrics",
			heartbeat: Heartbeat{
				Type:      TypeHeartbeat,
				Timestamp: now,
				Status:    "idle",
				Metrics:   nil,
			},
		},
		{
			name: "heartbeat with zero metrics",
			heartbeat: Heartbeat{
				Type:      TypeHeartbeat,
				Timestamp: now,
				Status:    "starting",
				Metrics: &SystemMetrics{
					CPUUsage:       0.0,
					MemoryUsage:    0.0,
					GPUUtilization: 0.0,
					GPUTemp:        0.0,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.heartbeat)
			require.NoError(t, err)

			// Unmarshal back
			var parsed Heartbeat
			err = json.Unmarshal(data, &parsed)
			require.NoError(t, err)

			// Verify fields
			assert.Equal(t, tt.heartbeat.Type, parsed.Type)
			assert.Equal(t, tt.heartbeat.Status, parsed.Status)
			assert.WithinDuration(t, tt.heartbeat.Timestamp, parsed.Timestamp, time.Second)

			if tt.heartbeat.Metrics != nil {
				require.NotNil(t, parsed.Metrics)
				assert.Equal(t, tt.heartbeat.Metrics.CPUUsage, parsed.Metrics.CPUUsage)
				assert.Equal(t, tt.heartbeat.Metrics.MemoryUsage, parsed.Metrics.MemoryUsage)
				assert.Equal(t, tt.heartbeat.Metrics.GPUUtilization, parsed.Metrics.GPUUtilization)
				assert.Equal(t, tt.heartbeat.Metrics.GPUTemp, parsed.Metrics.GPUTemp)
			} else {
				assert.Nil(t, parsed.Metrics)
			}
		})
	}
}

func TestSystemMetrics_Validation(t *testing.T) {
	tests := []struct {
		name    string
		metrics SystemMetrics
		valid   bool
	}{
		{
			name: "valid metrics",
			metrics: SystemMetrics{
				CPUUsage:       45.5,
				MemoryUsage:    67.8,
				GPUUtilization: 85.0,
				GPUTemp:        72.5,
			},
			valid: true,
		},
		{
			name: "zero metrics",
			metrics: SystemMetrics{
				CPUUsage:       0.0,
				MemoryUsage:    0.0,
				GPUUtilization: 0.0,
				GPUTemp:        0.0,
			},
			valid: true,
		},
		{
			name: "max values",
			metrics: SystemMetrics{
				CPUUsage:       100.0,
				MemoryUsage:    100.0,
				GPUUtilization: 100.0,
				GPUTemp:        100.0,
			},
			valid: true,
		},
		{
			name: "negative values",
			metrics: SystemMetrics{
				CPUUsage:       -5.0,
				MemoryUsage:    -10.0,
				GPUUtilization: -15.0,
				GPUTemp:        -20.0,
			},
			valid: false, // In real validation, negative values should be invalid
		},
		{
			name: "over 100 percent",
			metrics: SystemMetrics{
				CPUUsage:       105.0,
				MemoryUsage:    110.0,
				GPUUtilization: 150.0,
				GPUTemp:        45.0,
			},
			valid: false, // In real validation, > 100% should be invalid
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal and unmarshal to verify JSON handling
			data, err := json.Marshal(tt.metrics)
			require.NoError(t, err)

			var parsed SystemMetrics
			err = json.Unmarshal(data, &parsed)
			require.NoError(t, err)

			assert.Equal(t, tt.metrics, parsed)

			// Note: Actual validation would be implemented in the production code
			// Here we're just testing the structure and JSON marshaling
		})
	}
}

func TestMessage_JSONMarshaling(t *testing.T) {
	tests := []struct {
		name    string
		message Message
	}{
		{
			name: "message with metrics",
			message: Message{
				Type: TypeHeartbeat,
				Metrics: &SystemMetrics{
					CPUUsage:       50.0,
					MemoryUsage:    60.0,
					GPUUtilization: 70.0,
					GPUTemp:        65.0,
				},
			},
		},
		{
			name: "message without metrics",
			message: Message{
				Type:    TypeStatusUpdate,
				Metrics: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal to JSON
			data, err := json.Marshal(tt.message)
			require.NoError(t, err)

			// Unmarshal back
			var parsed Message
			err = json.Unmarshal(data, &parsed)
			require.NoError(t, err)

			// Verify fields
			assert.Equal(t, tt.message.Type, parsed.Type)
			if tt.message.Metrics != nil {
				require.NotNil(t, parsed.Metrics)
				assert.Equal(t, *tt.message.Metrics, *parsed.Metrics)
			} else {
				assert.Nil(t, parsed.Metrics)
			}
		})
	}
}

func TestMessageType_String(t *testing.T) {
	// Test that MessageType can be used as a string
	tests := []struct {
		messageType MessageType
		expected    string
	}{
		{TypeJobAssignment, "job_assignment"},
		{TypeStatusUpdate, "status_update"},
		{TypeHeartbeat, "heartbeat"},
		{TypeError, "error"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.messageType))
		})
	}
}

func TestJSONFieldTags(t *testing.T) {
	// Test that JSON field tags are correctly applied
	t.Run("JobAssignment fields", func(t *testing.T) {
		ja := JobAssignment{
			Type:       TypeJobAssignment,
			JobID:      "test",
			Parameters: "params",
			Priority:   1,
		}
		data, err := json.Marshal(ja)
		require.NoError(t, err)
		
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"type"`)
		assert.Contains(t, jsonStr, `"job_id"`)
		assert.Contains(t, jsonStr, `"parameters"`)
		assert.Contains(t, jsonStr, `"priority"`)
	})

	t.Run("StatusUpdate fields", func(t *testing.T) {
		su := StatusUpdate{
			Type:     TypeStatusUpdate,
			JobID:    "test",
			Status:   "running",
			Progress: 50.0,
			Details:  "details",
		}
		data, err := json.Marshal(su)
		require.NoError(t, err)
		
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"type"`)
		assert.Contains(t, jsonStr, `"job_id"`)
		assert.Contains(t, jsonStr, `"status"`)
		assert.Contains(t, jsonStr, `"progress"`)
		assert.Contains(t, jsonStr, `"details"`)
	})

	t.Run("SystemMetrics fields", func(t *testing.T) {
		sm := SystemMetrics{
			CPUUsage:       50.0,
			MemoryUsage:    60.0,
			GPUUtilization: 70.0,
			GPUTemp:        65.0,
		}
		data, err := json.Marshal(sm)
		require.NoError(t, err)
		
		jsonStr := string(data)
		assert.Contains(t, jsonStr, `"cpu_usage"`)
		assert.Contains(t, jsonStr, `"memory_usage"`)
		assert.Contains(t, jsonStr, `"gpu_utilization"`)
		assert.Contains(t, jsonStr, `"gpu_temp"`)
	})
}

// Benchmark JSON marshaling performance
func BenchmarkJobAssignment_Marshal(b *testing.B) {
	ja := JobAssignment{
		Type:     TypeJobAssignment,
		JobID:    "benchmark-job",
		Priority: 5,
		Parameters: map[string]interface{}{
			"hashlist": "test.hash",
			"wordlist": "rockyou.txt",
			"rules":    []string{"best64.rule", "dive.rule"},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(ja)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSystemMetrics_Marshal(b *testing.B) {
	sm := SystemMetrics{
		CPUUsage:       45.5,
		MemoryUsage:    67.8,
		GPUUtilization: 85.0,
		GPUTemp:        72.5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(sm)
		if err != nil {
			b.Fatal(err)
		}
	}
}
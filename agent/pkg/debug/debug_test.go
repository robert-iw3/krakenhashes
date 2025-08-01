package debug

import (
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogLevel(t *testing.T) {
	// Test log level constants
	assert.Equal(t, LogLevel(0), LevelDebug)
	assert.Equal(t, LogLevel(1), LevelInfo)
	assert.Equal(t, LogLevel(2), LevelWarning)
	assert.Equal(t, LogLevel(3), LevelError)
	
	// Test level names
	assert.Equal(t, "DEBUG", levelNames[LevelDebug])
	assert.Equal(t, "INFO", levelNames[LevelInfo])
	assert.Equal(t, "WARNING", levelNames[LevelWarning])
	assert.Equal(t, "ERROR", levelNames[LevelError])
}

func TestInit(t *testing.T) {
	// Save original values
	originalDebug := os.Getenv("DEBUG")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	defer func() {
		os.Setenv("DEBUG", originalDebug)
		os.Setenv("LOG_LEVEL", originalLogLevel)
	}()

	tests := []struct {
		name          string
		debugEnv      string
		logLevelEnv   string
		expectEnabled bool
		expectLevel   LogLevel
	}{
		{
			name:          "debug disabled by default",
			debugEnv:      "",
			logLevelEnv:   "",
			expectEnabled: false,
			expectLevel:   LevelInfo,
		},
		{
			name:          "debug enabled with true",
			debugEnv:      "true",
			logLevelEnv:   "",
			expectEnabled: true,
			expectLevel:   LevelInfo,
		},
		{
			name:          "debug enabled with 1",
			debugEnv:      "1",
			logLevelEnv:   "",
			expectEnabled: true,
			expectLevel:   LevelInfo,
		},
		{
			name:          "debug level set to DEBUG",
			debugEnv:      "true",
			logLevelEnv:   "DEBUG",
			expectEnabled: true,
			expectLevel:   LevelDebug,
		},
		{
			name:          "debug level set to WARNING",
			debugEnv:      "true",
			logLevelEnv:   "WARNING",
			expectEnabled: true,
			expectLevel:   LevelWarning,
		},
		{
			name:          "debug level case insensitive",
			debugEnv:      "true",
			logLevelEnv:   "error",
			expectEnabled: true,
			expectLevel:   LevelError,
		},
		{
			name:          "invalid log level defaults to INFO",
			debugEnv:      "true",
			logLevelEnv:   "INVALID",
			expectEnabled: true,
			expectLevel:   LevelInfo,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DEBUG", tt.debugEnv)
			os.Setenv("LOG_LEVEL", tt.logLevelEnv)
			
			// Reinitialize to pick up new env vars
			Reinitialize()
			
			assert.Equal(t, tt.expectEnabled, IsEnabled)
			assert.Equal(t, tt.expectLevel, CurrentLevel)
		})
	}
}

func TestLog(t *testing.T) {
	// Save original values
	originalDebug := IsEnabled
	originalLevel := CurrentLevel
	originalLogger := logger
	defer func() {
		IsEnabled = originalDebug
		CurrentLevel = originalLevel
		logger = originalLogger
	}()

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger = log.New(&buf, "", 0)

	tests := []struct {
		name          string
		enabled       bool
		currentLevel  LogLevel
		logLevel      LogLevel
		format        string
		args          []interface{}
		expectOutput  bool
		expectContains []string
	}{
		{
			name:         "debug disabled - no output",
			enabled:      false,
			currentLevel: LevelInfo,
			logLevel:     LevelInfo,
			format:       "test message",
			expectOutput: false,
		},
		{
			name:         "level too low - no output",
			enabled:      true,
			currentLevel: LevelWarning,
			logLevel:     LevelInfo,
			format:       "test message",
			expectOutput: false,
		},
		{
			name:         "info message output",
			enabled:      true,
			currentLevel: LevelInfo,
			logLevel:     LevelInfo,
			format:       "test message %s",
			args:         []interface{}{"with args"},
			expectOutput: true,
			expectContains: []string{
				"[INFO]",
				"test message with args",
			},
		},
		{
			name:         "error message output",
			enabled:      true,
			currentLevel: LevelDebug,
			logLevel:     LevelError,
			format:       "error occurred: %v",
			args:         []interface{}{"test error"},
			expectOutput: true,
			expectContains: []string{
				"[ERROR]",
				"error occurred: test error",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			IsEnabled = tt.enabled
			CurrentLevel = tt.currentLevel
			
			Log(tt.logLevel, tt.format, tt.args...)
			
			output := buf.String()
			if tt.expectOutput {
				assert.NotEmpty(t, output)
				for _, expected := range tt.expectContains {
					assert.Contains(t, output, expected)
				}
			} else {
				assert.Empty(t, output)
			}
		})
	}
}

func TestLogFunctions(t *testing.T) {
	// Save original values
	originalDebug := IsEnabled
	originalLevel := CurrentLevel
	originalLogger := logger
	defer func() {
		IsEnabled = originalDebug
		CurrentLevel = originalLevel
		logger = originalLogger
	}()

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger = log.New(&buf, "", 0)
	IsEnabled = true
	CurrentLevel = LevelDebug

	// Test Debug
	buf.Reset()
	Debug("debug message %d", 123)
	output := buf.String()
	assert.Contains(t, output, "[DEBUG]")
	assert.Contains(t, output, "debug message 123")

	// Test Info
	buf.Reset()
	Info("info message %s", "test")
	output = buf.String()
	assert.Contains(t, output, "[INFO]")
	assert.Contains(t, output, "info message test")

	// Test Warning
	buf.Reset()
	Warning("warning message %v", true)
	output = buf.String()
	assert.Contains(t, output, "[WARNING]")
	assert.Contains(t, output, "warning message true")

	// Test Error
	buf.Reset()
	Error("error message: %s", "failed")
	output = buf.String()
	assert.Contains(t, output, "[ERROR]")
	assert.Contains(t, output, "error message: failed")
}

func TestLogLevelFiltering(t *testing.T) {
	// Save original values
	originalDebug := IsEnabled
	originalLevel := CurrentLevel
	originalLogger := logger
	defer func() {
		IsEnabled = originalDebug
		CurrentLevel = originalLevel
		logger = originalLogger
	}()

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger = log.New(&buf, "", 0)
	IsEnabled = true

	tests := []struct {
		name         string
		currentLevel LogLevel
		messages     []struct {
			fn     func(string, ...interface{})
			msg    string
			expect bool
		}
	}{
		{
			name:         "INFO level filters DEBUG",
			currentLevel: LevelInfo,
			messages: []struct {
				fn     func(string, ...interface{})
				msg    string
				expect bool
			}{
				{Debug, "debug msg", false},
				{Info, "info msg", true},
				{Warning, "warning msg", true},
				{Error, "error msg", true},
			},
		},
		{
			name:         "WARNING level filters INFO and DEBUG",
			currentLevel: LevelWarning,
			messages: []struct {
				fn     func(string, ...interface{})
				msg    string
				expect bool
			}{
				{Debug, "debug msg", false},
				{Info, "info msg", false},
				{Warning, "warning msg", true},
				{Error, "error msg", true},
			},
		},
		{
			name:         "ERROR level only shows errors",
			currentLevel: LevelError,
			messages: []struct {
				fn     func(string, ...interface{})
				msg    string
				expect bool
			}{
				{Debug, "debug msg", false},
				{Info, "info msg", false},
				{Warning, "warning msg", false},
				{Error, "error msg", true},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CurrentLevel = tt.currentLevel
			
			for _, msg := range tt.messages {
				buf.Reset()
				msg.fn(msg.msg)
				output := buf.String()
				
				if msg.expect {
					assert.NotEmpty(t, output, "Expected output for: %s", msg.msg)
					assert.Contains(t, output, msg.msg)
				} else {
					assert.Empty(t, output, "Expected no output for: %s", msg.msg)
				}
			}
		})
	}
}

func TestReinitialize(t *testing.T) {
	// Save original values
	originalDebug := os.Getenv("DEBUG")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalIsEnabled := IsEnabled
	originalCurrentLevel := CurrentLevel
	defer func() {
		os.Setenv("DEBUG", originalDebug)
		os.Setenv("LOG_LEVEL", originalLogLevel)
		IsEnabled = originalIsEnabled
		CurrentLevel = originalCurrentLevel
	}()

	// Set initial state
	os.Setenv("DEBUG", "false")
	os.Setenv("LOG_LEVEL", "INFO")
	Reinitialize()
	
	assert.False(t, IsEnabled)
	assert.Equal(t, LevelInfo, CurrentLevel)
	
	// Change environment and reinitialize
	os.Setenv("DEBUG", "true")
	os.Setenv("LOG_LEVEL", "ERROR")
	Reinitialize()
	
	assert.True(t, IsEnabled)
	assert.Equal(t, LevelError, CurrentLevel)
}

func TestLogOutput(t *testing.T) {
	// Save original values
	originalDebug := IsEnabled
	originalLevel := CurrentLevel
	originalLogger := logger
	defer func() {
		IsEnabled = originalDebug
		CurrentLevel = originalLevel
		logger = originalLogger
	}()

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger = log.New(&buf, "", 0)
	IsEnabled = true
	CurrentLevel = LevelDebug

	// Test log output format
	Info("test message")
	output := buf.String()
	
	// Should contain all expected parts
	assert.Contains(t, output, "[INFO]")
	assert.Contains(t, output, "test message")
	assert.Regexp(t, `\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3}\]`, output) // Timestamp
	assert.Regexp(t, `\[\S+:\d+\]`, output) // File:line
}

func TestConcurrentLogging(t *testing.T) {
	// Save original values
	originalDebug := IsEnabled
	originalLevel := CurrentLevel
	originalLogger := logger
	defer func() {
		IsEnabled = originalDebug
		CurrentLevel = originalLevel
		logger = originalLogger
	}()

	// Create a buffer to capture log output
	var buf bytes.Buffer
	logger = log.New(&buf, "", 0)
	IsEnabled = true
	CurrentLevel = LevelDebug

	// Test concurrent access
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			Debug("concurrent debug %d", id)
			Info("concurrent info %d", id)
			Warning("concurrent warning %d", id)
			Error("concurrent error %d", id)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	output := buf.String()
	// Should have output from all goroutines
	lines := strings.Split(strings.TrimSpace(output), "\n")
	assert.Equal(t, 40, len(lines)) // 4 messages per goroutine * 10 goroutines
}

// Benchmark tests
func BenchmarkLog(b *testing.B) {
	// Save original values
	originalDebug := IsEnabled
	originalLevel := CurrentLevel
	originalLogger := logger
	defer func() {
		IsEnabled = originalDebug
		CurrentLevel = originalLevel
		logger = originalLogger
	}()

	// Disable actual output
	logger = log.New(bytes.NewBuffer(nil), "", 0)
	IsEnabled = true
	CurrentLevel = LevelInfo

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message %d", i)
	}
}

func BenchmarkLogDisabled(b *testing.B) {
	// Save original values
	originalDebug := IsEnabled
	defer func() {
		IsEnabled = originalDebug
	}()

	IsEnabled = false

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message %d", i)
	}
}

func BenchmarkLogFiltered(b *testing.B) {
	// Save original values
	originalDebug := IsEnabled
	originalLevel := CurrentLevel
	defer func() {
		IsEnabled = originalDebug
		CurrentLevel = originalLevel
	}()

	IsEnabled = true
	CurrentLevel = LevelError // Filter out INFO messages

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message %d", i)
	}
}
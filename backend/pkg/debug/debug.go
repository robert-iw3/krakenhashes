package debug

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"
)

// LogLevel represents the severity of a log message
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarning
	LevelError
)

var (
	// IsEnabled controls whether debug messages are output
	IsEnabled bool
	// CurrentLevel is the minimum level of messages to output
	CurrentLevel LogLevel
	logger       *log.Logger
	levelNames   = map[LogLevel]string{
		LevelDebug:   "DEBUG",
		LevelInfo:    "INFO",
		LevelWarning: "WARNING",
		LevelError:   "ERROR",
	}
	levelMap = map[string]LogLevel{
		"DEBUG":   LevelDebug,
		"INFO":    LevelInfo,
		"WARNING": LevelWarning,
		"ERROR":   LevelError,
	}
)

func init() {
	// Initialize logger with timestamp and caller info
	logger = log.New(os.Stdout, "", 0)

	// Check DEBUG environment variable
	debugEnv := os.Getenv("DEBUG")
	IsEnabled = debugEnv == "true" || debugEnv == "1"

	// Set log level from environment variable
	levelEnv := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	if level, exists := levelMap[levelEnv]; exists {
		CurrentLevel = level
	} else {
		CurrentLevel = LevelInfo // Default to INFO if not specified
	}

	// Only log initialization if debugging is enabled
	if IsEnabled {
		Info("Debug logging initialized - Enabled: %v, Level: %s", IsEnabled, levelNames[CurrentLevel])
	}
}

// Log prints a debug message with the specified level if debugging is enabled
func Log(level LogLevel, format string, v ...interface{}) {
	// Check if debugging is enabled and if the message level is high enough
	if !IsEnabled || level < CurrentLevel {
		return
	}

	// Get caller information
	pc, file, line, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()

	// Format the message
	message := fmt.Sprintf(format, v...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	logger.Printf("[%s] [%s] [%s:%d] [%s] %s\n",
		levelNames[level],
		timestamp,
		file,
		line,
		funcName,
		message,
	)
}

// Debug logs a debug level message
func Debug(format string, v ...interface{}) {
	Log(LevelDebug, format, v...)
}

// Info logs an info level message
func Info(format string, v ...interface{}) {
	Log(LevelInfo, format, v...)
}

// Warning logs a warning level message
func Warning(format string, v ...interface{}) {
	Log(LevelWarning, format, v...)
}

// Error logs an error level message
func Error(format string, v ...interface{}) {
	Log(LevelError, format, v...)
}

// Reinitialize updates the debug settings based on current environment variables
func Reinitialize() {
	// Check DEBUG environment variable
	debugEnv := os.Getenv("DEBUG")
	IsEnabled = debugEnv == "true" || debugEnv == "1"

	// Set log level from environment variable
	levelEnv := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	if level, exists := levelMap[levelEnv]; exists {
		CurrentLevel = level
	} else {
		CurrentLevel = LevelInfo // Default to INFO if not specified
	}

	// Only log initialization if debugging is enabled
	if IsEnabled {
		Info("Debug logging reinitialized - Enabled: %v, Level: %s", IsEnabled, levelNames[CurrentLevel])
	}
}

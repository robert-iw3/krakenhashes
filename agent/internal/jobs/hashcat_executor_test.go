package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockHashcatScript creates a script that simulates hashcat behavior
func createMockHashcatScript(t *testing.T, tempDir string, behavior string) string {
	scriptPath := filepath.Join(tempDir, "mock_hashcat.sh")
	
	var scriptContent string
	switch behavior {
	case "already_running":
		// Simulate "already running" error
		scriptContent = `#!/bin/bash
echo "Already an instance /path/to/hashcat running on pid 12345" >&2
exit 255
`
	case "success_after_retry":
		// Check if a marker file exists - if not, create it and fail
		// If it exists, succeed
		markerFile := filepath.Join(tempDir, "retry_marker")
		scriptContent = fmt.Sprintf(`#!/bin/bash
if [ ! -f "%s" ]; then
    touch "%s"
    echo "Already an instance /path/to/hashcat running on pid 12345" >&2
    exit 255
else
    echo '{"status": 3, "devices": [{"device_id": 1, "speed": 1000000}], "progress": [100, 1000]}' 
    echo "hash1:password1"
    exit 0
fi
`, markerFile, markerFile)
	case "always_fail":
		// Always fail with already running error
		scriptContent = `#!/bin/bash
echo "Already an instance /path/to/hashcat running on pid 12345" >&2
exit 255
`
	default:
		// Normal success
		scriptContent = `#!/bin/bash
echo '{"status": 3, "devices": [{"device_id": 1, "speed": 1000000}], "progress": [100, 1000]}'
echo "hash1:password1"
exit 0
`
	}
	
	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)
	
	return scriptPath
}

func TestHashcatExecutor_AlreadyRunningDetection(t *testing.T) {
	// Test that the executor correctly detects and sets AlreadyRunningError flag
	tempDir := t.TempDir()
	
	// Create mock hashcat that always returns "already running" error
	mockHashcat := createMockHashcatScript(t, tempDir, "already_running")
	
	// Create executor
	executor := NewHashcatExecutor(tempDir)
	
	// Create test assignment
	assignment := &JobTaskAssignment{
		TaskID:          "test-already-running",
		JobExecutionID:  "job-123",
		HashlistID:      1,
		HashlistPath:    "hashlists/test.hash",
		AttackMode:      0, // AttackModeStraight
		HashType:        0,
		KeyspaceStart:   0,
		KeyspaceEnd:     1000,
		WordlistPaths:   []string{"wordlists/test.txt"},
		BinaryPath:      mockHashcat,
		ReportInterval:  1,
	}
	
	// Create required directories and files
	hashlistDir := filepath.Join(tempDir, "hashlists")
	wordlistDir := filepath.Join(tempDir, "wordlists")
	require.NoError(t, os.MkdirAll(hashlistDir, 0755))
	require.NoError(t, os.MkdirAll(wordlistDir, 0755))
	
	// Create test files
	require.NoError(t, os.WriteFile(
		filepath.Join(hashlistDir, "test.hash"),
		[]byte("5d41402abc4b2a76b9719d911017c592\n"),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(wordlistDir, "test.txt"),
		[]byte("password\n"),
		0644,
	))
	
	// Execute task
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	process, err := executor.ExecuteTask(ctx, assignment)
	require.NoError(t, err)
	require.NotNil(t, process)
	
	// Wait for the error
	var finalProgress *JobProgress
	for progress := range process.ProgressChannel {
		finalProgress = progress
		if progress.Status == "failed" {
			break
		}
	}
	
	// Should have failed with AlreadyRunningError flag set
	assert.NotNil(t, finalProgress)
	assert.Equal(t, "failed", finalProgress.Status)
	assert.True(t, process.AlreadyRunningError)
	assert.Contains(t, finalProgress.ErrorMessage, "another instance is already running")
}

func TestJobManager_RetryLogic(t *testing.T) {
	// Test that job manager properly retries on "already running" errors
	tempDir := t.TempDir()
	
	// Create a mock hardware monitor
	mockHwMonitor := &mockHardwareMonitor{}
	
	// Create job manager
	cfg := &config.Config{
		DataDirectory: tempDir,
	}
	
	var lastProgress *JobProgress
	progressCallback := func(progress *JobProgress) {
		lastProgress = progress
	}
	
	jm := NewJobManager(cfg, progressCallback, mockHwMonitor)
	
	// Create mock hashcat that succeeds on second attempt
	mockHashcat := createMockHashcatScript(t, tempDir, "success_after_retry")
	
	// Create test assignment
	assignment := JobTaskAssignment{
		TaskID:          "test-retry-job",
		JobExecutionID:  "job-789",
		HashlistID:      1,
		HashlistPath:    "hashlists/test.hash",
		AttackMode:      0,
		HashType:        0,
		KeyspaceStart:   0,
		KeyspaceEnd:     1000,
		WordlistPaths:   []string{"wordlists/test.txt"},
		BinaryPath:      mockHashcat,
		ReportInterval:  1,
	}
	
	// Create required directories and files
	hashlistDir := filepath.Join(tempDir, "hashlists")
	wordlistDir := filepath.Join(tempDir, "wordlists")
	require.NoError(t, os.MkdirAll(hashlistDir, 0755))
	require.NoError(t, os.MkdirAll(wordlistDir, 0755))
	
	// Create test files
	require.NoError(t, os.WriteFile(
		filepath.Join(hashlistDir, "test.hash"),
		[]byte("5d41402abc4b2a76b9719d911017c592\n"),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(wordlistDir, "test.txt"),
		[]byte("password\n"),
		0644,
	))
	
	// Marshal assignment (not used in simplified test)
	_, err := json.Marshal(assignment)
	require.NoError(t, err)
	
	// Skip the hashlist check and directly execute the task
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	// Directly execute the task using the executor (bypassing file sync)
	process, err := jm.executor.ExecuteTask(ctx, &assignment)
	require.NoError(t, err)
	
	// Monitor the process
	go jm.monitorJobProgress(ctx, &JobExecution{
		Assignment: &assignment,
		Process:    process,
		StartTime:  time.Now(),
		Status:     "running",
	})
	
	// Wait for completion
	time.Sleep(10 * time.Second)
	
	// Check that job eventually succeeded
	assert.NotNil(t, lastProgress)
	assert.Equal(t, "completed", lastProgress.Status)
}

// Mock hardware monitor for testing
type mockHardwareMonitor struct{}

func (m *mockHardwareMonitor) GetEnabledDeviceFlags() string {
	return ""
}

func (m *mockHardwareMonitor) HasEnabledDevices() bool {
	return true
}

func TestHashcatExecutor_DetectAlreadyRunningError(t *testing.T) {
	// Test that the executor correctly detects "already running" errors
	tempDir := t.TempDir()
	
	// Create a simple test to verify error detection works
	executor := NewHashcatExecutor(tempDir)
	
	// Create a mock command that outputs the error
	if runtime.GOOS == "windows" {
		t.Skip("Skipping on Windows - needs different mock approach")
	}
	
	cmd := exec.Command("sh", "-c", `echo "Already an instance /usr/bin/hashcat running on pid 12345" >&2; exit 255`)
	
	// Create a mock process
	process := &HashcatProcess{
		TaskID:          "test-detect",
		Cmd:             cmd,
		ProgressChannel: make(chan *JobProgress, 10),
		StartTime:       time.Now(),
	}
	
	// Get pipes
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	stderr, err := cmd.StderrPipe()
	require.NoError(t, err)
	
	// Run the process
	ctx := context.Background()
	go executor.runHashcatProcess(ctx, process, stdout, stderr)
	
	// Wait for error
	var errorProgress *JobProgress
	select {
	case progress := <-process.ProgressChannel:
		if progress.Status == "failed" {
			errorProgress = progress
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for error progress")
	}
	
	// Verify error was detected
	require.NotNil(t, errorProgress)
	assert.Equal(t, "failed", errorProgress.Status)
	assert.Contains(t, errorProgress.ErrorMessage, "another instance is already running")
}

func TestHashcatExecutor_NormalExecutionNotRetried(t *testing.T) {
	// Test that normal errors don't trigger retry logic
	tempDir := t.TempDir()
	
	// Create mock hashcat that fails with different error
	scriptPath := filepath.Join(tempDir, "mock_hashcat.sh")
	scriptContent := `#!/bin/bash
echo "Some other error occurred" >&2
exit 1
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(scriptContent), 0755))
	
	// Create executor
	executor := NewHashcatExecutor(tempDir)
	
	// Create test assignment
	assignment := &JobTaskAssignment{
		TaskID:          "test-no-retry",
		JobExecutionID:  "job-789",
		HashlistID:      1,
		HashlistPath:    "hashlists/test.hash",
		AttackMode:      0, // AttackModeStraight
		HashType:        0,
		KeyspaceStart:   0,
		KeyspaceEnd:     1000,
		WordlistPaths:   []string{"wordlists/test.txt"},
		BinaryPath:      scriptPath,
		ReportInterval:  1,
	}
	
	// Create required directories and files
	hashlistDir := filepath.Join(tempDir, "hashlists")
	wordlistDir := filepath.Join(tempDir, "wordlists")
	require.NoError(t, os.MkdirAll(hashlistDir, 0755))
	require.NoError(t, os.MkdirAll(wordlistDir, 0755))
	
	// Create test files
	require.NoError(t, os.WriteFile(
		filepath.Join(hashlistDir, "test.hash"),
		[]byte("5d41402abc4b2a76b9719d911017c592\n"),
		0644,
	))
	require.NoError(t, os.WriteFile(
		filepath.Join(wordlistDir, "test.txt"),
		[]byte("password\n"),
		0644,
	))
	
	// Execute task
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	
	startTime := time.Now()
	process, err := executor.ExecuteTask(ctx, assignment)
	require.NoError(t, err)
	require.NotNil(t, process)
	
	// Wait for completion
	var finalProgress *JobProgress
	for progress := range process.ProgressChannel {
		finalProgress = progress
		if progress.Status == "failed" {
			break
		}
	}
	
	duration := time.Since(startTime)
	
	// Should fail without retries
	assert.NotNil(t, finalProgress)
	assert.Equal(t, "failed", finalProgress.Status)
	assert.NotContains(t, finalProgress.ErrorMessage, "another instance is already running")
	
	// Should complete quickly (no retries)
	assert.Less(t, duration, 5*time.Second, "Should complete quickly without retries")
}
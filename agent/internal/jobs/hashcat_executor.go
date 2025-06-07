package jobs

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// AttackMode represents hashcat attack modes
type AttackMode int

const (
	AttackModeStraight           AttackMode = 0 // Dictionary attack
	AttackModeCombination        AttackMode = 1 // Combination attack
	AttackModeBruteForce         AttackMode = 3 // Brute-force attack
	AttackModeHybridWordlistMask AttackMode = 6 // Hybrid Wordlist + Mask
	AttackModeHybridMaskWordlist AttackMode = 7 // Hybrid Mask + Wordlist
)

// JobTaskAssignment represents a task assignment from the backend
type JobTaskAssignment struct {
	TaskID          string      `json:"task_id"`
	JobExecutionID  string      `json:"job_execution_id"`
	HashlistID      string      `json:"hashlist_id"`
	HashlistPath    string      `json:"hashlist_path"`    // Local path on agent
	AttackMode      int         `json:"attack_mode"`
	HashType        int         `json:"hash_type"`
	KeyspaceStart   int64       `json:"keyspace_start"`
	KeyspaceEnd     int64       `json:"keyspace_end"`
	WordlistPaths   []string    `json:"wordlist_paths"`   // Local paths on agent
	RulePaths       []string    `json:"rule_paths"`       // Local paths on agent
	Mask            string      `json:"mask,omitempty"`   // For mask attacks
	BinaryPath      string      `json:"binary_path"`      // Hashcat binary to use
	ChunkDuration   int         `json:"chunk_duration"`   // Expected duration in seconds
	ReportInterval  int         `json:"report_interval"`  // Progress reporting interval
	OutputFormat    string      `json:"output_format"`    // Hashcat output format
}

// JobProgress represents progress updates sent to backend
type JobProgress struct {
	TaskID            string   `json:"task_id"`
	KeyspaceProcessed int64    `json:"keyspace_processed"`
	HashRate          int64    `json:"hash_rate"`         // Current hashes per second
	Temperature       *float64 `json:"temperature"`       // GPU temperature
	Utilization       *float64 `json:"utilization"`       // GPU utilization percentage
	TimeRemaining     *int     `json:"time_remaining"`    // Estimated seconds remaining
	CrackedCount      int      `json:"cracked_count"`     // Number of hashes cracked in this update
	CrackedHashes     []string `json:"cracked_hashes"`    // Actual cracked hash:plain pairs
}

// HashcatExecutor handles hashcat process execution and monitoring
type HashcatExecutor struct {
	dataDirectory string
	workDirectory string
	
	// Process management
	mutex           sync.RWMutex
	activeProcesses map[string]*HashcatProcess
}

// HashcatProcess represents an active hashcat process
type HashcatProcess struct {
	TaskID          string
	Assignment      *JobTaskAssignment
	Command         *exec.Cmd
	Cancel          context.CancelFunc
	ProgressChannel chan *JobProgress
	StatusFile      string
	PotFile         string
	OutputFile      string
	
	// Process state
	IsRunning       bool
	StartTime       time.Time
	LastProgress    *JobProgress
	LastCheckpoint  time.Time
}

// NewHashcatExecutor creates a new hashcat executor
func NewHashcatExecutor(dataDirectory, workDirectory string) *HashcatExecutor {
	return &HashcatExecutor{
		dataDirectory:   dataDirectory,
		workDirectory:   workDirectory,
		activeProcesses: make(map[string]*HashcatProcess),
	}
}

// ExecuteTask starts execution of a hashcat task
func (e *HashcatExecutor) ExecuteTask(ctx context.Context, assignment *JobTaskAssignment) (*HashcatProcess, error) {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	// Check if task is already running
	if _, exists := e.activeProcesses[assignment.TaskID]; exists {
		return nil, fmt.Errorf("task %s is already running", assignment.TaskID)
	}

	// Create process context with cancellation
	processCtx, cancel := context.WithCancel(ctx)

	// Build hashcat command
	command, statusFile, potFile, outputFile, err := e.buildHashcatCommand(assignment)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to build hashcat command: %w", err)
	}

	// Set command context
	command.Env = os.Environ()
	command.Dir = e.workDirectory

	// Create process structure
	process := &HashcatProcess{
		TaskID:          assignment.TaskID,
		Assignment:      assignment,
		Command:         command,
		Cancel:          cancel,
		ProgressChannel: make(chan *JobProgress, 100),
		StatusFile:      statusFile,
		PotFile:         potFile,
		OutputFile:      outputFile,
		IsRunning:       false,
		StartTime:       time.Now(),
	}

	// Store process
	e.activeProcesses[assignment.TaskID] = process

	// Start the process in a goroutine
	go e.runHashcatProcess(processCtx, process)

	return process, nil
}

// buildHashcatCommand builds the hashcat command line arguments
func (e *HashcatExecutor) buildHashcatCommand(assignment *JobTaskAssignment) (*exec.Cmd, string, string, string, error) {
	// Create work files
	taskWorkDir := filepath.Join(e.workDirectory, "tasks", assignment.TaskID)
	err := os.MkdirAll(taskWorkDir, 0755)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("failed to create task directory: %w", err)
	}

	statusFile := filepath.Join(taskWorkDir, "status.txt")
	potFile := filepath.Join(taskWorkDir, "hashcat.pot")
	outputFile := filepath.Join(taskWorkDir, "output.txt")

	// Base arguments
	args := []string{
		"-m", strconv.Itoa(assignment.HashType),     // Hash type
		"-a", strconv.Itoa(int(assignment.AttackMode)), // Attack mode
		"--status",                                   // Enable status output
		"--status-timer", strconv.Itoa(assignment.ReportInterval), // Status update interval
		"--status-autostart",                         // Auto-start status updates
		"--machine-readable",                         // Machine-readable output
		"--quiet",                                    // Reduce verbose output
		"--potfile-disable",                          // Disable default potfile
		"--outfile", outputFile,                      // Output file for cracked hashes
		"--outfile-format", assignment.OutputFormat, // Output format
	}

	// Add skip and limit for keyspace distribution
	if assignment.KeyspaceStart > 0 {
		args = append(args, "--skip", strconv.FormatInt(assignment.KeyspaceStart, 10))
	}
	
	if assignment.KeyspaceEnd > assignment.KeyspaceStart {
		keyspaceRange := assignment.KeyspaceEnd - assignment.KeyspaceStart
		args = append(args, "--limit", strconv.FormatInt(keyspaceRange, 10))
	}

	// Add hashlist file
	hashlistPath := filepath.Join(e.dataDirectory, assignment.HashlistPath)
	args = append(args, hashlistPath)

	// Add attack-mode specific arguments
	switch assignment.AttackMode {
	case int(AttackModeStraight): // Dictionary attack
		// Add wordlists
		for _, wordlistPath := range assignment.WordlistPaths {
			fullPath := filepath.Join(e.dataDirectory, wordlistPath)
			args = append(args, fullPath)
		}
		// Add rules
		for _, rulePath := range assignment.RulePaths {
			fullPath := filepath.Join(e.dataDirectory, rulePath)
			args = append(args, "-r", fullPath)
		}

	case int(AttackModeCombination): // Combination attack
		if len(assignment.WordlistPaths) >= 2 {
			wordlist1 := filepath.Join(e.dataDirectory, assignment.WordlistPaths[0])
			wordlist2 := filepath.Join(e.dataDirectory, assignment.WordlistPaths[1])
			args = append(args, wordlist1, wordlist2)
		}

	case int(AttackModeBruteForce): // Mask attack
		if assignment.Mask != "" {
			args = append(args, assignment.Mask)
		}

	case int(AttackModeHybridWordlistMask): // Hybrid Wordlist + Mask
		if len(assignment.WordlistPaths) > 0 && assignment.Mask != "" {
			wordlistPath := filepath.Join(e.dataDirectory, assignment.WordlistPaths[0])
			args = append(args, wordlistPath, assignment.Mask)
		}

	case int(AttackModeHybridMaskWordlist): // Hybrid Mask + Wordlist
		if assignment.Mask != "" && len(assignment.WordlistPaths) > 0 {
			wordlistPath := filepath.Join(e.dataDirectory, assignment.WordlistPaths[0])
			args = append(args, assignment.Mask, wordlistPath)
		}

	default:
		return nil, "", "", "", fmt.Errorf("unsupported attack mode: %d", assignment.AttackMode)
	}

	// Create command
	cmd := exec.Command(assignment.BinaryPath, args...)

	return cmd, statusFile, potFile, outputFile, nil
}

// runHashcatProcess executes and monitors a hashcat process
func (e *HashcatExecutor) runHashcatProcess(ctx context.Context, process *HashcatProcess) {
	defer func() {
		e.mutex.Lock()
		delete(e.activeProcesses, process.TaskID)
		e.mutex.Unlock()
		close(process.ProgressChannel)
	}()

	// Start progress monitoring
	go e.monitorProgress(ctx, process)

	// Mark as running
	process.IsRunning = true

	// Start the command
	err := process.Command.Start()
	if err != nil {
		e.sendErrorProgress(process, fmt.Sprintf("Failed to start hashcat: %v", err))
		return
	}

	// Wait for completion or cancellation
	done := make(chan error, 1)
	go func() {
		done <- process.Command.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context cancelled, kill the process
		if process.Command.Process != nil {
			process.Command.Process.Kill()
		}
		e.sendProgressUpdate(process, &JobProgress{
			TaskID: process.TaskID,
		}, "cancelled")

	case err := <-done:
		// Process completed
		process.IsRunning = false
		
		if err != nil {
			e.sendErrorProgress(process, fmt.Sprintf("Hashcat process failed: %v", err))
		} else {
			// Process completed successfully, send final progress
			finalProgress := e.parseFinalResults(process)
			e.sendProgressUpdate(process, finalProgress, "completed")
		}
	}
}

// monitorProgress monitors hashcat progress and sends updates
func (e *HashcatExecutor) monitorProgress(ctx context.Context, process *HashcatProcess) {
	ticker := time.NewTicker(time.Duration(process.Assignment.ReportInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !process.IsRunning {
				return
			}
			
			progress := e.parseProgressFromStatus(process)
			if progress != nil {
				e.sendProgressUpdate(process, progress, "running")
				process.LastProgress = progress
				process.LastCheckpoint = time.Now()
			}
		}
	}
}

// parseProgressFromStatus parses progress from hashcat status output
func (e *HashcatExecutor) parseProgressFromStatus(process *HashcatProcess) *JobProgress {
	// This is a simplified parser - actual implementation would need to handle
	// hashcat's machine-readable status output format
	
	progress := &JobProgress{
		TaskID: process.TaskID,
	}

	// Try to read status from hashcat output (this would need proper implementation)
	// For now, we'll simulate progress based on elapsed time
	elapsed := time.Since(process.StartTime)
	expectedDuration := time.Duration(process.Assignment.ChunkDuration) * time.Second
	
	if expectedDuration > 0 {
		progressRatio := float64(elapsed) / float64(expectedDuration)
		if progressRatio > 1.0 {
			progressRatio = 1.0
		}
		
		keyspaceRange := process.Assignment.KeyspaceEnd - process.Assignment.KeyspaceStart
		progress.KeyspaceProcessed = int64(float64(keyspaceRange) * progressRatio)
		
		// Estimate hash rate (this would come from actual hashcat output)
		if elapsed > 0 {
			progress.HashRate = progress.KeyspaceProcessed / int64(elapsed.Seconds())
		}
		
		// Estimate time remaining
		if progressRatio > 0 && progressRatio < 1.0 {
			totalTime := float64(elapsed) / progressRatio
			remaining := int(totalTime - float64(elapsed))
			progress.TimeRemaining = &remaining
		}
	}

	return progress
}

// parseFinalResults parses final results from hashcat output
func (e *HashcatExecutor) parseFinalResults(process *HashcatProcess) *JobProgress {
	progress := &JobProgress{
		TaskID:            process.TaskID,
		KeyspaceProcessed: process.Assignment.KeyspaceEnd - process.Assignment.KeyspaceStart,
	}

	// Parse cracked hashes from output file
	crackedHashes, err := e.readCrackedHashes(process.OutputFile)
	if err == nil {
		progress.CrackedHashes = crackedHashes
		progress.CrackedCount = len(crackedHashes)
	}

	return progress
}

// readCrackedHashes reads cracked hashes from output file
func (e *HashcatExecutor) readCrackedHashes(outputFile string) ([]string, error) {
	file, err := os.Open(outputFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var crackedHashes []string
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			crackedHashes = append(crackedHashes, line)
		}
	}

	return crackedHashes, scanner.Err()
}

// sendProgressUpdate sends a progress update through the channel
func (e *HashcatExecutor) sendProgressUpdate(process *HashcatProcess, progress *JobProgress, status string) {
	select {
	case process.ProgressChannel <- progress:
		// Progress sent successfully
	default:
		// Channel full, log warning but don't block
		// In a real implementation, we might want to handle this differently
	}
}

// sendErrorProgress sends an error progress update
func (e *HashcatExecutor) sendErrorProgress(process *HashcatProcess, errorMsg string) {
	progress := &JobProgress{
		TaskID: process.TaskID,
	}
	
	e.sendProgressUpdate(process, progress, "failed")
}

// StopTask stops a running task
func (e *HashcatExecutor) StopTask(taskID string) error {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	process, exists := e.activeProcesses[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}

	// Cancel the context to stop the process
	process.Cancel()
	return nil
}

// GetTaskProgress returns the current progress of a task
func (e *HashcatExecutor) GetTaskProgress(taskID string) (*JobProgress, error) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	process, exists := e.activeProcesses[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}

	return process.LastProgress, nil
}

// GetActiveTaskIDs returns a list of currently active task IDs
func (e *HashcatExecutor) GetActiveTaskIDs() []string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	var taskIDs []string
	for taskID := range e.activeProcesses {
		taskIDs = append(taskIDs, taskID)
	}

	return taskIDs
}

// RunBenchmark runs a hashcat benchmark for performance testing
func (e *HashcatExecutor) RunBenchmark(ctx context.Context, binaryPath string, hashType int, attackMode int) (int64, error) {
	// Create benchmark command
	args := []string{
		"-m", strconv.Itoa(hashType),
		"-a", strconv.Itoa(attackMode),
		"--benchmark",
		"--machine-readable",
		"--quiet",
	}

	cmd := exec.CommandContext(ctx, binaryPath, args...)
	cmd.Dir = e.workDirectory

	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("benchmark failed: %w", err)
	}

	// Parse benchmark result (this would need proper implementation based on hashcat output)
	speed, err := e.parseBenchmarkOutput(string(output))
	if err != nil {
		return 0, fmt.Errorf("failed to parse benchmark output: %w", err)
	}

	return speed, nil
}

// parseBenchmarkOutput parses hashcat benchmark output to extract speed
func (e *HashcatExecutor) parseBenchmarkOutput(output string) (int64, error) {
	// This is a simplified parser - actual implementation would need to handle
	// hashcat's benchmark output format properly
	
	// Look for speed pattern in output (e.g., "Speed.#1.........:  1234.5 MH/s")
	speedRegex := regexp.MustCompile(`Speed\..*?:\s*([0-9.]+)\s*([KMG]?)H/s`)
	matches := speedRegex.FindStringSubmatch(output)
	
	if len(matches) < 3 {
		return 0, fmt.Errorf("could not parse speed from benchmark output")
	}

	speedStr := matches[1]
	unit := matches[2]

	speed, err := strconv.ParseFloat(speedStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse speed value: %w", err)
	}

	// Convert to hashes per second
	switch strings.ToUpper(unit) {
	case "K":
		speed *= 1000
	case "M":
		speed *= 1000000
	case "G":
		speed *= 1000000000
	case "":
		// Already in H/s
	default:
		return 0, fmt.Errorf("unknown speed unit: %s", unit)
	}

	return int64(speed), nil
}
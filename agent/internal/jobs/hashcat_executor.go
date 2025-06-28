package jobs

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
	
	"github.com/ZerkerEOD/krakenhashes/agent/pkg/debug"
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
	HashlistID      int64       `json:"hashlist_id"`
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
	TaskID            string         `json:"task_id"`
	KeyspaceProcessed int64          `json:"keyspace_processed"`
	HashRate          int64          `json:"hash_rate"`         // Current hashes per second
	Temperature       *float64       `json:"temperature"`       // GPU temperature
	Utilization       *float64       `json:"utilization"`       // GPU utilization percentage
	TimeRemaining     *int           `json:"time_remaining"`    // Estimated seconds remaining
	CrackedCount      int            `json:"cracked_count"`     // Number of hashes cracked in this update
	CrackedHashes     []CrackedHash  `json:"cracked_hashes"`    // Detailed crack information
	Status            string         `json:"status,omitempty"`  // Task status (running, completed, failed)
	ErrorMessage      string         `json:"error_message,omitempty"` // Error message if status is failed
}

// CrackedHash represents a cracked hash with all available information
type CrackedHash struct {
	Hash         string `json:"hash"`          // The original hash
	Salt         string `json:"salt"`          // Salt (if applicable)
	Plain        string `json:"plain"`         // Plain text password
	HexPlain     string `json:"hex_plain"`     // Hex representation of plain
	CrackPos     string `json:"crack_pos"`     // Position in keyspace where found
	FullLine     string `json:"full_line"`     // Full output line for reference
}

// DeviceSpeed represents speed for a single device
type DeviceSpeed struct {
	DeviceID   int    `json:"device_id"`
	DeviceName string `json:"device_name"`
	Speed      int64  `json:"speed"` // H/s for this device
}

// HashcatExecutor handles hashcat process execution and monitoring
type HashcatExecutor struct {
	dataDirectory string
	
	// Process management
	mutex           sync.RWMutex
	activeProcesses map[string]*HashcatProcess
	
	// Output callback for sending output via websocket
	outputCallback  func(taskID string, output string, isError bool)
}

// HashcatProcess represents an active hashcat process
type HashcatProcess struct {
	TaskID          string
	Assignment      *JobTaskAssignment
	Cmd             *exec.Cmd
	Cancel          context.CancelFunc
	ProgressChannel chan *JobProgress
	StatusFile      string
	PotFile         string
	OutputFile      string
	StdinPipe       io.WriteCloser
	
	// Process state
	IsRunning       bool
	StartTime       time.Time
	LastProgress    *JobProgress
	LastCheckpoint  time.Time
}


// NewHashcatExecutor creates a new hashcat executor
func NewHashcatExecutor(dataDirectory string) *HashcatExecutor {
	// We don't use a work directory since we're capturing output from stdout
	// with --potfile-disable and no output files
	
	return &HashcatExecutor{
		dataDirectory:   dataDirectory,
		activeProcesses: make(map[string]*HashcatProcess),
	}
}

// SetOutputCallback sets the callback for sending output via websocket
func (e *HashcatExecutor) SetOutputCallback(callback func(taskID string, output string, isError bool)) {
	e.outputCallback = callback
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

	// Set command context - no specific directory needed
	command.Env = os.Environ()
	
	// Set up stdin pipe for sending commands to hashcat
	stdinPipe, err := command.StdinPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	
	// Set up stdout pipe for capturing output
	stdoutPipe, err := command.StdoutPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	// Set up stderr pipe for error messages
	stderrPipe, err := command.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Create process structure
	process := &HashcatProcess{
		TaskID:          assignment.TaskID,
		Assignment:      assignment,
		Cmd:             command,
		Cancel:          cancel,
		ProgressChannel: make(chan *JobProgress, 100),
		StatusFile:      statusFile,
		PotFile:         potFile,
		OutputFile:      outputFile,
		StdinPipe:       stdinPipe,
		IsRunning:       false,
		StartTime:       time.Now(),
	}

	// Store process
	e.activeProcesses[assignment.TaskID] = process

	// Start the process in a goroutine
	go e.runHashcatProcess(processCtx, process, stdoutPipe, stderrPipe)

	return process, nil
}

// buildHashcatCommand builds the hashcat command line arguments
func (e *HashcatExecutor) buildHashcatCommand(assignment *JobTaskAssignment) (*exec.Cmd, string, string, string, error) {
	return e.buildHashcatCommandWithOptions(assignment, false)
}

// buildHashcatCommandWithOptions builds the hashcat command line arguments with options
func (e *HashcatExecutor) buildHashcatCommandWithOptions(assignment *JobTaskAssignment, isBenchmark bool) (*exec.Cmd, string, string, string, error) {
	debug.Info("Building hashcat command for task %s", assignment.TaskID)
	debug.Info("Data directory: %s", e.dataDirectory)
	debug.Info("Binary path from assignment: %s", assignment.BinaryPath)
	debug.Info("Hashlist path from assignment: %s", assignment.HashlistPath)
	
	// Since we're running distributed with --potfile-disable and capturing output from stdout,
	// we don't need to create any work files. Just return empty paths.
	statusFile := ""
	potFile := ""
	outputFile := ""

	// Base arguments
	args := []string{
		"-m", strconv.Itoa(assignment.HashType),     // Hash type
		"-a", strconv.Itoa(int(assignment.AttackMode)), // Attack mode
		"--status",                                   // Enable status output
		"--status-json",                              // Output status in JSON format
		"--status-timer", strconv.Itoa(assignment.ReportInterval), // Status update interval
		"--quiet",                                    // Reduce verbose output
		"--potfile-disable",                          // Disable potfile
		"--restore-disable",                          // Disable restore files (we handle restore via keyspace)
		"-O",                                         // Enable optimized kernels
		"-w", "3",                                    // Workload profile: high
	}
	
	// Only add --remove for actual job execution, not benchmarks
	if !isBenchmark {
		args = append(args, "--remove") // Remove cracked hashes from hashlist
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
	
	// Debug: Check if hashlist file exists
	if _, err := os.Stat(hashlistPath); os.IsNotExist(err) {
		debug.Error("Hashlist file does not exist: %s", hashlistPath)
		return nil, "", "", "", fmt.Errorf("hashlist file not found: %s", hashlistPath)
	}
	
	args = append(args, hashlistPath)

	// Add attack-mode specific arguments
	switch assignment.AttackMode {
	case int(AttackModeStraight): // Dictionary attack
		// Add wordlists
		debug.Info("Adding wordlists to hashcat command: %v", assignment.WordlistPaths)
		for _, wordlistPath := range assignment.WordlistPaths {
			fullPath := filepath.Join(e.dataDirectory, wordlistPath)
			debug.Info("Adding wordlist: %s (full path: %s)", wordlistPath, fullPath)
			args = append(args, fullPath)
		}
		// Add rules
		debug.Info("Adding rules to hashcat command: %v", assignment.RulePaths)
		for _, rulePath := range assignment.RulePaths {
			fullPath := filepath.Join(e.dataDirectory, rulePath)
			debug.Info("Adding rule: %s (full path: %s)", rulePath, fullPath)
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

	// Resolve the hashcat binary path
	hashcatBinary, err := e.resolveHashcatBinary(assignment.BinaryPath)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("failed to resolve hashcat binary: %w", err)
	}
	
	debug.Info("Using hashcat binary: %s", hashcatBinary)
	debug.Info("Full hashcat command: %s %s", hashcatBinary, strings.Join(args, " "))
	
	// Create command
	cmd := exec.Command(hashcatBinary, args...)

	return cmd, statusFile, potFile, outputFile, nil
}

// runHashcatProcess executes and monitors a hashcat process
func (e *HashcatExecutor) runHashcatProcess(ctx context.Context, process *HashcatProcess, stdoutPipe, stderrPipe io.ReadCloser) {
	defer func() {
		e.mutex.Lock()
		delete(e.activeProcesses, process.TaskID)
		e.mutex.Unlock()
		close(process.ProgressChannel)
		if process.StdinPipe != nil {
			process.StdinPipe.Close()
		}
	}()

	// Start output readers before starting the process
	outputDone := make(chan bool, 2)
	
	// Read stdout for JSON status and cracked hashes
	go func() {
		defer func() {
			// Send completion signal safely
			select {
			case outputDone <- true:
			default:
			}
		}()
		scanner := bufio.NewScanner(stdoutPipe)
		
		for scanner.Scan() {
			line := scanner.Text()
			debug.Debug("[Hashcat stdout raw] %s", line)
			
			// Send output via websocket if callback is set
			if e.outputCallback != nil {
				e.outputCallback(process.TaskID, line, false)
			}
			
			// Sometimes hashcat outputs crack result and JSON on same line
			// First check if line contains both crack and JSON
			if strings.Contains(line, ":") && strings.Contains(line, "{") && strings.Contains(line, "\"status\"") {
				// Split at the JSON start
				jsonStart := strings.Index(line, "{")
				crackPart := strings.TrimSpace(line[:jsonStart])
				jsonPart := line[jsonStart:]
				
				// Process crack part first
				if len(crackPart) > 0 {
					parts := strings.Split(crackPart, ":")
					if len(parts) >= 2 {
						var cracked CrackedHash
						cracked.Hash = parts[0]
						cracked.Plain = parts[1]
						if len(parts) > 2 {
							cracked.CrackPos = parts[2]
						}
						cracked.FullLine = crackPart
						
						if len(cracked.Hash) >= 16 && !strings.Contains(cracked.Hash, " ") {
							progress := &JobProgress{
								TaskID:        process.TaskID,
								CrackedCount:  1,
								CrackedHashes: []CrackedHash{cracked},
							}
							e.sendProgressUpdate(process, progress, "cracked")
							debug.Info("[Hashcat cracked] Hash: %s, Plain: %s", 
								cracked.Hash, cracked.Plain)
						}
					}
				}
				
				// Now process JSON part
				line = jsonPart
			}
			
			// Check if this is a JSON status line
			if strings.HasPrefix(line, "{") && strings.Contains(line, "\"status\"") {
				// This is a JSON status update
				// Fix hashcat's invalid JSON - it outputs device_id with leading zeros like 01, 02
				fixedLine := line
				re := regexp.MustCompile(`"device_id":\s*0+(\d+)`)
				fixedLine = re.ReplaceAllString(fixedLine, `"device_id": $1`)
				
				var status map[string]interface{}
				if err := json.Unmarshal([]byte(fixedLine), &status); err == nil {
					// Extract key metrics from JSON
					if progressArr, ok := status["progress"].([]interface{}); ok && len(progressArr) >= 2 {
						// Use restore_point for actual keyspace progress
						var keyspaceProcessed int64
						if restorePoint, ok := status["restore_point"].(float64); ok {
							keyspaceProcessed = int64(restorePoint)
						} else if current, ok := progressArr[0].(float64); ok {
							// Fallback to progress[0] if restore_point not available
							keyspaceProcessed = int64(current)
						}
						
						progress := &JobProgress{
							TaskID:            process.TaskID,
							KeyspaceProcessed: keyspaceProcessed,
						}
						
						// Extract speed from devices array - sum all device speeds
						var totalSpeed int64
						if devices, ok := status["devices"].([]interface{}); ok {
							for i, dev := range devices {
								if device, ok := dev.(map[string]interface{}); ok {
									if speed, ok := device["speed"].(float64); ok {
										totalSpeed += int64(speed)
									}
									// Use first device's temp and util for reporting
									if i == 0 {
										if temp, ok := device["temp"].(float64); ok {
											progress.Temperature = &temp
										}
										if util, ok := device["util"].(float64); ok {
											progress.Utilization = &util
										}
									}
								}
							}
						}
						progress.HashRate = totalSpeed
						
						// Calculate time remaining
						if total, ok := progressArr[1].(float64); ok && progress.HashRate > 0 {
							remaining := int64(total) - progress.KeyspaceProcessed
							if remaining > 0 {
								timeRemaining := int(remaining / progress.HashRate)
								progress.TimeRemaining = &timeRemaining
							}
						}
						
						e.sendProgressUpdate(process, progress, "running")
					}
				} else {
					debug.Warning("[Hashcat] Failed to parse JSON status: %v", err)
				}
			} else if strings.Contains(line, ":") && !strings.HasPrefix(line, "{") && !strings.Contains(line, "Skipping") {
				// This might be a cracked hash
				// Format can be:
				// - hash:plain (2 parts)
				// - hash:plain:hex_plain (3 parts)
				// - salt:hash:plain (3 parts for salted hashes)
				parts := strings.Split(line, ":")
				
				if len(parts) >= 2 {
					// Simple case: hash:plain
					var cracked CrackedHash
					cracked.Hash = parts[0]
					cracked.Plain = parts[1]
					if len(parts) > 2 {
						cracked.CrackPos = parts[2]
					}
					cracked.FullLine = line
					
					// Validate it looks like a hash (not a warning message)
					if len(cracked.Hash) >= 16 && !strings.Contains(cracked.Hash, " ") {
						// Send crack update immediately
						progress := &JobProgress{
							TaskID:        process.TaskID,
							CrackedCount:  1,
							CrackedHashes: []CrackedHash{cracked},
						}
						e.sendProgressUpdate(process, progress, "cracked")
						
						debug.Info("[Hashcat cracked] Hash: %s, Plain: %s", 
							cracked.Hash, cracked.Plain)
					} else {
						debug.Debug("[Hashcat stdout] %s", line)
					}
				} else {
					debug.Debug("[Hashcat stdout] %s", line)
				}
			} else {
				debug.Debug("[Hashcat stdout] %s", line)
			}
		}
	}()
	
	// Read stderr for errors and warnings
	go func() {
		defer func() {
			// Send completion signal safely
			select {
			case outputDone <- true:
			default:
			}
		}()
		scanner := bufio.NewScanner(stderrPipe)
		for scanner.Scan() {
			line := scanner.Text()
			debug.Debug("[Hashcat stderr] %s", line)
			
			// Send error output via websocket if callback is set
			if e.outputCallback != nil {
				e.outputCallback(process.TaskID, line, true)
			}
		}
	}()


	// Mark as running
	process.IsRunning = true

	// Start the command
	debug.Info("Starting hashcat process for task %s", process.TaskID)
	debug.Info("Command: %s", process.Cmd.Path)
	debug.Info("Args: %v", process.Cmd.Args)
	
	err := process.Cmd.Start()
	if err != nil {
		debug.Error("Failed to start hashcat process: %v", err)
		e.sendErrorProgress(process, fmt.Sprintf("Failed to start hashcat: %v", err))
		return
	}
	
	debug.Info("Hashcat process started successfully with PID: %d", process.Cmd.Process.Pid)

	// Wait for completion or cancellation
	done := make(chan error, 1)
	go func() {
		done <- process.Cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context cancelled, kill the process
		if process.Cmd.Process != nil {
			process.Cmd.Process.Kill()
		}
		e.sendProgressUpdate(process, &JobProgress{
			TaskID: process.TaskID,
		}, "cancelled")

	case err := <-done:
		// Process completed
		process.IsRunning = false
		
		// Wait for output goroutines to complete
		for i := 0; i < 2; i++ {
			select {
			case <-outputDone:
			case <-time.After(5 * time.Second):
				debug.Warning("Timeout waiting for output goroutine to complete")
			}
		}
		
		if err != nil {
			e.sendErrorProgress(process, fmt.Sprintf("Hashcat process failed: %v", err))
		} else {
			// Process completed successfully, send final progress
			finalProgress := &JobProgress{
				TaskID:            process.TaskID,
				KeyspaceProcessed: process.Assignment.KeyspaceEnd - process.Assignment.KeyspaceStart,
			}
			e.sendProgressUpdate(process, finalProgress, "completed")
		}
	}
}

// sendProgressUpdate sends a progress update through the channel
func (e *HashcatExecutor) sendProgressUpdate(process *HashcatProcess, progress *JobProgress, status string) {
	// Set the status in the progress update
	progress.Status = status
	
	select {
	case process.ProgressChannel <- progress:
		// Progress sent successfully
	default:
		// Channel full, log warning but don't block
		debug.Warning("Progress channel full for task %s, dropping update", process.TaskID)
	}
}

// sendErrorProgress sends an error progress update
func (e *HashcatExecutor) sendErrorProgress(process *HashcatProcess, errorMsg string) {
	progress := &JobProgress{
		TaskID:       process.TaskID,
		Status:       "failed",
		ErrorMessage: errorMsg,
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

// RunSpeedTest runs a real-world speed test with actual job configuration
func (e *HashcatExecutor) RunSpeedTest(ctx context.Context, assignment *JobTaskAssignment, testDuration int) (int64, []DeviceSpeed, error) {
	debug.Info("Running speed test for hash type %d, attack mode %d, duration %d seconds", 
		assignment.HashType, assignment.AttackMode, testDuration)
	
	// Build command similar to real job but without skip/limit and without --remove
	cmd, _, _, _, err := e.buildHashcatCommandWithOptions(assignment, true)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to build command: %w", err)
	}
	
	// Get the original args
	originalArgs := cmd.Args[1:] // Skip the command itself
	
	// Remove --skip and --limit arguments for speed test
	filteredArgs := []string{}
	skipNext := false
	for _, arg := range originalArgs {
		if skipNext {
			skipNext = false
			continue
		}
		if arg == "--skip" || arg == "--limit" {
			skipNext = true
			continue
		}
		filteredArgs = append(filteredArgs, arg)
	}
	
	debug.Info("Starting speed test with command: %s %s", cmd.Path, strings.Join(filteredArgs, " "))
	
	// Create new command with filtered args
	cmd = exec.CommandContext(ctx, cmd.Path, filteredArgs...)
	
	// Set up pipes for stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	
	// Start the command
	if err := cmd.Start(); err != nil {
		return 0, nil, fmt.Errorf("failed to start hashcat: %w", err)
	}
	
	// Channel to collect status updates
	statusChan := make(chan string, 10)
	stopReading := make(chan bool)
	
	// Read stdout in goroutine
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-stopReading:
				debug.Debug("[Speed test] Stopping stdout reader")
				return
			default:
				line := scanner.Text()
				if strings.TrimSpace(line) != "" {
					debug.Debug("[Speed test stdout raw] %s", line)
					// Sometimes hashcat outputs crack result and JSON on same line
					// First check if line contains both crack and JSON
					if strings.Contains(line, ":") && strings.Contains(line, "{") && strings.Contains(line, "\"status\"") {
						// Split at the JSON start
						jsonStart := strings.Index(line, "{")
						jsonPart := line[jsonStart:]
						debug.Debug("[Speed test] Found mixed output, extracted JSON: %s", jsonPart)
						// Just use the JSON part for speed test
						select {
						case statusChan <- jsonPart:
						case <-stopReading:
							return
						}
					} else if strings.HasPrefix(line, "{") && strings.Contains(line, "\"status\"") {
						// Pure JSON status line
						debug.Debug("[Speed test] Found pure JSON status")
						select {
						case statusChan <- line:
						case <-stopReading:
							return
						}
					}
				}
			}
		}
	}()
	
	// Read stderr in goroutine
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			debug.Debug("[Hashcat stderr] %s", line)
		}
	}()
	
	// Timer to stop after test duration
	timer := time.NewTimer(time.Duration(testDuration) * time.Second)
	
	// Collect status updates
	var statusUpdates []string
	var lastValidSpeed int64
	var lastDeviceSpeeds []DeviceSpeed
	statusCollected := make(chan bool)
	
	go func() {
		for {
			select {
			case status := <-statusChan:
				debug.Debug("[Speed test] Received status update %d", len(statusUpdates)+1)
				// Try to parse this status update immediately
				speed, devSpeeds, err := e.parseSpeedFromJSON(status)
				if err == nil && speed > 0 {
					lastValidSpeed = speed
					lastDeviceSpeeds = devSpeeds
					debug.Info("[Speed test] Parsed valid speed: %d H/s from update %d", speed, len(statusUpdates)+1)
				} else if err != nil {
					debug.Warning("[Speed test] Failed to parse update %d: %v", len(statusUpdates)+1, err)
				}
				
				statusUpdates = append(statusUpdates, status)
				// We want to get at least 3 updates
				if len(statusUpdates) >= 3 {
					debug.Info("[Speed test] Collected 3 updates, stopping collection")
					timer.Stop()
					close(statusCollected)
					return
				}
			case <-timer.C:
				debug.Info("[Speed test] Timer expired after %d updates", len(statusUpdates))
				close(statusCollected)
				return
			}
		}
	}()
	
	// Wait for status collection to complete
	<-statusCollected
	
	// Stop reading stdout/stderr
	close(stopReading)
	
	// Kill the process and wait for it to exit
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
	cmd.Wait() // Clean up the process
	
	// Check if we got any valid speed readings
	if lastValidSpeed == 0 {
		debug.Warning("[Speed test] No valid speed parsed during collection, checking stored updates")
		if len(statusUpdates) == 0 {
			return 0, nil, fmt.Errorf("no status updates received during speed test")
		}
		
		// Try to parse the last status update one more time
		statusIndex := len(statusUpdates) - 1
		if len(statusUpdates) >= 3 {
			statusIndex = 2 // Third update (0-indexed)
		}
		
		debug.Debug("[Speed test] Attempting to parse update %d: %s", statusIndex+1, statusUpdates[statusIndex])
		totalSpeed, deviceSpeeds, err := e.parseSpeedFromJSON(statusUpdates[statusIndex])
		if err != nil {
			// Log the actual content that failed to parse
			debug.Error("[Speed test] Failed to parse JSON from update %d. Content: %s", statusIndex+1, statusUpdates[statusIndex])
			return 0, nil, fmt.Errorf("failed to parse speed from status: %w", err)
		}
		
		if totalSpeed == 0 {
			return 0, nil, fmt.Errorf("speed test returned 0 H/s after %d updates", len(statusUpdates))
		}
		
		lastValidSpeed = totalSpeed
		lastDeviceSpeeds = deviceSpeeds
	}
	
	debug.Info("Speed test completed: %d H/s total from %d updates", lastValidSpeed, len(statusUpdates))
	return lastValidSpeed, lastDeviceSpeeds, nil
}

// parseSpeedFromJSON parses device speeds from hashcat JSON status output
func (e *HashcatExecutor) parseSpeedFromJSON(jsonStr string) (int64, []DeviceSpeed, error) {
	// Fix hashcat's invalid JSON - it outputs device_id with leading zeros like 01, 02
	// which is invalid JSON. We need to fix these to be valid numbers.
	fixedJSON := jsonStr
	
	// Use regex to fix device_id values with leading zeros
	// This will convert "device_id": 01 to "device_id": 1
	re := regexp.MustCompile(`"device_id":\s*0+(\d+)`)
	fixedJSON = re.ReplaceAllString(fixedJSON, `"device_id": $1`)
	
	var status struct {
		Devices []struct {
			DeviceID   int    `json:"device_id"`
			DeviceName string `json:"device_name"`
			Speed      int64  `json:"speed"`
		} `json:"devices"`
	}
	
	if err := json.Unmarshal([]byte(fixedJSON), &status); err != nil {
		return 0, nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	var totalSpeed int64
	var deviceSpeeds []DeviceSpeed
	
	for _, device := range status.Devices {
		totalSpeed += device.Speed
		deviceSpeeds = append(deviceSpeeds, DeviceSpeed{
			DeviceID:   device.DeviceID,
			DeviceName: device.DeviceName,
			Speed:      device.Speed,
		})
	}
	
	return totalSpeed, deviceSpeeds, nil
}


// resolveHashcatBinary resolves the hashcat binary path from the assignment
func (e *HashcatExecutor) resolveHashcatBinary(binaryPath string) (string, error) {
	debug.Info("Resolving hashcat binary from path: %s", binaryPath)
	
	// The binaryPath might come in different formats:
	// - "binaries/hashcat_2" (old format)
	// - "binaries/8" (new format, just the ID)
	// We need to resolve this to the actual executable
	
	var binaryDir string
	
	// Check if it's the old format
	if strings.HasPrefix(binaryPath, "binaries/hashcat_") {
		binaryID := strings.TrimPrefix(binaryPath, "binaries/hashcat_")
		binaryDir = filepath.Join(e.dataDirectory, "binaries", binaryID)
	} else if strings.HasPrefix(binaryPath, "binaries/") {
		// New format - just the binary ID
		binaryID := strings.TrimPrefix(binaryPath, "binaries/")
		binaryDir = filepath.Join(e.dataDirectory, "binaries", binaryID)
	} else {
		// Direct path or other format
		// Check if it's already a full path
		if _, err := os.Stat(binaryPath); err == nil {
			return binaryPath, nil
		}
		// Try in data directory
		fullPath := filepath.Join(e.dataDirectory, binaryPath)
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, nil
		}
		return "", fmt.Errorf("invalid binary path format: %s", binaryPath)
	}
	
	if binaryDir != "" {
		
		// Look for hashcat executable in the binary directory
		// The binary should have been extracted from the .7z archive
		var possiblePaths []string
		
		// Prioritize OS-specific binaries
		switch runtime.GOOS {
		case "windows":
			possiblePaths = []string{
				filepath.Join(binaryDir, "hashcat.exe"),  // Windows primary
				filepath.Join(binaryDir, "hashcat"),      // Windows fallback
			}
		case "linux":
			possiblePaths = []string{
				filepath.Join(binaryDir, "hashcat.bin"),  // Linux primary
				filepath.Join(binaryDir, "hashcat"),      // Linux fallback
			}
		case "darwin":
			possiblePaths = []string{
				filepath.Join(binaryDir, "hashcat"),      // macOS primary
				filepath.Join(binaryDir, "hashcat.bin"),  // macOS fallback
			}
		default:
			possiblePaths = []string{
				filepath.Join(binaryDir, "hashcat"),      // Default Unix-like
				filepath.Join(binaryDir, "hashcat.bin"),  // Alternative
			}
		}
		
		for _, path := range possiblePaths {
			if fileInfo, err := os.Stat(path); err == nil {
				// Check if it's the right type of executable for this OS
				isExecutable := false
				
				if runtime.GOOS == "windows" {
					// On Windows, .exe files are executable
					isExecutable = strings.HasSuffix(path, ".exe") || fileInfo.Mode()&0111 != 0
				} else {
					// On Unix-like systems, check execute permission and skip .exe files
					isExecutable = !strings.HasSuffix(path, ".exe") && fileInfo.Mode()&0111 != 0
				}
				
				if isExecutable {
					debug.Info("Found hashcat binary for %s at: %s", runtime.GOOS, path)
					return path, nil
				}
			}
		}
		
		// Check if the .7z archive exists but hasn't been extracted
		archivePath := filepath.Join(binaryDir, "hashcat-6.2.6+1017.7z")
		if _, err := os.Stat(archivePath); err == nil {
			return "", fmt.Errorf("hashcat archive found at %s but not extracted. Please ensure file sync extracts binaries", archivePath)
		}
		
		return "", fmt.Errorf("hashcat binary not found in directory %s. Checked paths: %v", binaryDir, possiblePaths)
	}
	
	// If it's a direct path, check if it exists
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}
	
	// Try in data directory
	fullPath := filepath.Join(e.dataDirectory, binaryPath)
	if _, err := os.Stat(fullPath); err == nil {
		return fullPath, nil
	}
	
	return "", fmt.Errorf("hashcat binary not found: %s", binaryPath)
}
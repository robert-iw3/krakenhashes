package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/agent/internal/config"
)

// JobManager manages job execution on the agent
type JobManager struct {
	executor         *HashcatExecutor
	config           *config.Config
	progressCallback func(*JobProgress)
	
	// Job state
	mutex           sync.RWMutex
	activeJobs      map[string]*JobExecution
	benchmarkCache  map[string]*BenchmarkResult
}

// JobExecution represents an active job execution
type JobExecution struct {
	Assignment      *JobTaskAssignment
	Process         *HashcatProcess
	StartTime       time.Time
	LastProgress    *JobProgress
	Status          string
}

// BenchmarkResult stores benchmark results
type BenchmarkResult struct {
	HashType    int
	AttackMode  int
	Speed       int64
	Timestamp   time.Time
}

// NewJobManager creates a new job manager
func NewJobManager(cfg *config.Config, progressCallback func(*JobProgress)) *JobManager {
	dataDir := cfg.DataDirectory
	workDir := filepath.Join(dataDir, "work")
	
	executor := NewHashcatExecutor(dataDir, workDir)
	
	return &JobManager{
		executor:         executor,
		config:           cfg,
		progressCallback: progressCallback,
		activeJobs:       make(map[string]*JobExecution),
		benchmarkCache:   make(map[string]*BenchmarkResult),
	}
}

// ProcessJobAssignment processes a job assignment from the backend
func (jm *JobManager) ProcessJobAssignment(ctx context.Context, assignmentData []byte) error {
	var assignment JobTaskAssignment
	err := json.Unmarshal(assignmentData, &assignment)
	if err != nil {
		return fmt.Errorf("failed to unmarshal job assignment: %w", err)
	}

	log.Printf("Processing job assignment: Task ID %s, Job ID %s", assignment.TaskID, assignment.JobExecutionID)

	// Check if task is already running
	jm.mutex.RLock()
	if _, exists := jm.activeJobs[assignment.TaskID]; exists {
		jm.mutex.RUnlock()
		return fmt.Errorf("task %s is already running", assignment.TaskID)
	}
	jm.mutex.RUnlock()

	// Run benchmark if needed
	err = jm.ensureBenchmark(ctx, &assignment)
	if err != nil {
		log.Printf("Warning: Benchmark failed for task %s: %v", assignment.TaskID, err)
		// Continue without benchmark - use estimated values
	}

	// Start job execution
	process, err := jm.executor.ExecuteTask(ctx, &assignment)
	if err != nil {
		return fmt.Errorf("failed to start task execution: %w", err)
	}

	// Create job execution record
	jobExecution := &JobExecution{
		Assignment:   &assignment,
		Process:      process,
		StartTime:    time.Now(),
		Status:       "running",
	}

	jm.mutex.Lock()
	jm.activeJobs[assignment.TaskID] = jobExecution
	jm.mutex.Unlock()

	// Start progress monitoring
	go jm.monitorJobProgress(ctx, jobExecution)

	log.Printf("Job assignment started: Task ID %s", assignment.TaskID)
	return nil
}

// ensureBenchmark runs a benchmark if needed for the job
func (jm *JobManager) ensureBenchmark(ctx context.Context, assignment *JobTaskAssignment) error {
	benchmarkKey := fmt.Sprintf("%d_%d", assignment.HashType, assignment.AttackMode)
	
	jm.mutex.RLock()
	cached, exists := jm.benchmarkCache[benchmarkKey]
	jm.mutex.RUnlock()

	// Check if we have a recent benchmark (within 24 hours)
	if exists && time.Since(cached.Timestamp) < 24*time.Hour {
		log.Printf("Using cached benchmark for hash type %d, attack mode %d: %d H/s", 
			assignment.HashType, assignment.AttackMode, cached.Speed)
		return nil
	}

	// Run benchmark
	log.Printf("Running benchmark for hash type %d, attack mode %d", assignment.HashType, assignment.AttackMode)
	
	benchmarkCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	speed, err := jm.executor.RunBenchmark(benchmarkCtx, assignment.BinaryPath, assignment.HashType, assignment.AttackMode)
	if err != nil {
		return fmt.Errorf("benchmark failed: %w", err)
	}

	// Store benchmark result
	result := &BenchmarkResult{
		HashType:   assignment.HashType,
		AttackMode: assignment.AttackMode,
		Speed:      speed,
		Timestamp:  time.Now(),
	}

	jm.mutex.Lock()
	jm.benchmarkCache[benchmarkKey] = result
	jm.mutex.Unlock()

	log.Printf("Benchmark completed: %d H/s for hash type %d, attack mode %d", speed, assignment.HashType, assignment.AttackMode)
	return nil
}

// monitorJobProgress monitors job progress and sends updates
func (jm *JobManager) monitorJobProgress(ctx context.Context, jobExecution *JobExecution) {
	defer func() {
		jm.mutex.Lock()
		delete(jm.activeJobs, jobExecution.Assignment.TaskID)
		jm.mutex.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case progress, ok := <-jobExecution.Process.ProgressChannel:
			if !ok {
				// Channel closed, job finished
				log.Printf("Job progress monitoring ended for task %s", jobExecution.Assignment.TaskID)
				return
			}

			if progress != nil {
				jobExecution.LastProgress = progress
				
				// Send progress to backend via callback
				if jm.progressCallback != nil {
					jm.progressCallback(progress)
				}

				log.Printf("Progress update for task %s: %d/%d keyspace, %d H/s", 
					progress.TaskID, 
					progress.KeyspaceProcessed, 
					jobExecution.Assignment.KeyspaceEnd - jobExecution.Assignment.KeyspaceStart,
					progress.HashRate)

				// Log any cracked hashes
				if progress.CrackedCount > 0 {
					log.Printf("Task %s cracked %d hashes in this update", progress.TaskID, progress.CrackedCount)
				}
			}
		}
	}
}

// StopJob stops a running job
func (jm *JobManager) StopJob(taskID string) error {
	jm.mutex.RLock()
	jobExecution, exists := jm.activeJobs[taskID]
	jm.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("job %s not found", taskID)
	}

	log.Printf("Stopping job: Task ID %s", taskID)
	
	err := jm.executor.StopTask(taskID)
	if err != nil {
		return fmt.Errorf("failed to stop task: %w", err)
	}

	// Update job status
	jobExecution.Status = "stopped"
	
	log.Printf("Job stopped: Task ID %s", taskID)
	return nil
}

// GetJobStatus returns the status of a specific job
func (jm *JobManager) GetJobStatus(taskID string) (*JobExecution, error) {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	jobExecution, exists := jm.activeJobs[taskID]
	if !exists {
		return nil, fmt.Errorf("job %s not found", taskID)
	}

	return jobExecution, nil
}

// GetActiveJobs returns a list of currently active jobs
func (jm *JobManager) GetActiveJobs() map[string]*JobExecution {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	// Return a copy to avoid concurrent access issues
	activeJobs := make(map[string]*JobExecution)
	for taskID, job := range jm.activeJobs {
		activeJobs[taskID] = job
	}

	return activeJobs
}

// GetBenchmarkResults returns cached benchmark results
func (jm *JobManager) GetBenchmarkResults() map[string]*BenchmarkResult {
	jm.mutex.RLock()
	defer jm.mutex.RUnlock()

	// Return a copy to avoid concurrent access issues
	benchmarks := make(map[string]*BenchmarkResult)
	for key, result := range jm.benchmarkCache {
		benchmarks[key] = result
	}

	return benchmarks
}

// RunManualBenchmark runs a benchmark manually for testing purposes
func (jm *JobManager) RunManualBenchmark(ctx context.Context, binaryPath string, hashType int, attackMode int) (*BenchmarkResult, error) {
	log.Printf("Running manual benchmark for hash type %d, attack mode %d", hashType, attackMode)

	speed, err := jm.executor.RunBenchmark(ctx, binaryPath, hashType, attackMode)
	if err != nil {
		return nil, fmt.Errorf("manual benchmark failed: %w", err)
	}

	result := &BenchmarkResult{
		HashType:   hashType,
		AttackMode: attackMode,
		Speed:      speed,
		Timestamp:  time.Now(),
	}

	// Store in cache
	benchmarkKey := fmt.Sprintf("%d_%d", hashType, attackMode)
	jm.mutex.Lock()
	jm.benchmarkCache[benchmarkKey] = result
	jm.mutex.Unlock()

	log.Printf("Manual benchmark completed: %d H/s", speed)
	return result, nil
}

// Shutdown gracefully shuts down the job manager
func (jm *JobManager) Shutdown(ctx context.Context) error {
	log.Println("Shutting down job manager...")

	jm.mutex.RLock()
	activeTaskIDs := make([]string, 0, len(jm.activeJobs))
	for taskID := range jm.activeJobs {
		activeTaskIDs = append(activeTaskIDs, taskID)
	}
	jm.mutex.RUnlock()

	// Stop all active jobs
	for _, taskID := range activeTaskIDs {
		err := jm.StopJob(taskID)
		if err != nil {
			log.Printf("Error stopping job %s during shutdown: %v", taskID, err)
		}
	}

	// Wait for jobs to stop (with timeout)
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-shutdownCtx.Done():
			log.Println("Job manager shutdown timeout reached")
			return shutdownCtx.Err()
		case <-ticker.C:
			jm.mutex.RLock()
			activeCount := len(jm.activeJobs)
			jm.mutex.RUnlock()

			if activeCount == 0 {
				log.Println("Job manager shutdown completed")
				return nil
			}

			log.Printf("Waiting for %d jobs to stop...", activeCount)
		}
	}
}

// Legacy function for compatibility
func ProcessJobs() {
	log.Println("ProcessJobs called - this is now handled by JobManager")
}

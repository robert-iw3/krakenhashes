package services

import (
	"context"
	"fmt"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)


// JobChunkingService handles job chunking based on time and agent performance
type JobChunkingService struct {
	benchmarkRepo      *repository.BenchmarkRepository
	jobTaskRepo        *repository.JobTaskRepository
	systemSettingsRepo *repository.SystemSettingsRepository
}

// NewJobChunkingService creates a new job chunking service
func NewJobChunkingService(
	benchmarkRepo *repository.BenchmarkRepository,
	jobTaskRepo *repository.JobTaskRepository,
	systemSettingsRepo *repository.SystemSettingsRepository,
) *JobChunkingService {
	return &JobChunkingService{
		benchmarkRepo:      benchmarkRepo,
		jobTaskRepo:        jobTaskRepo,
		systemSettingsRepo: systemSettingsRepo,
	}
}

// ChunkCalculationRequest contains the parameters needed for chunk calculation
type ChunkCalculationRequest struct {
	JobExecution  *models.JobExecution
	Agent         *models.Agent
	AttackMode    models.AttackMode
	HashType      int
	ChunkDuration int // Desired chunk duration in seconds
}

// ChunkCalculationResult contains the calculated chunk parameters
type ChunkCalculationResult struct {
	KeyspaceStart  int64
	KeyspaceEnd    int64
	BenchmarkSpeed *int64
	ActualDuration int // Estimated actual duration in seconds
	IsLastChunk    bool
}

// CalculateNextChunk calculates the next chunk for an agent based on benchmarks and time constraints
func (s *JobChunkingService) CalculateNextChunk(ctx context.Context, req ChunkCalculationRequest) (*ChunkCalculationResult, error) {
	debug.Log("Calculating next chunk", map[string]interface{}{
		"job_execution_id": req.JobExecution.ID,
		"agent_id":         req.Agent.ID,
		"attack_mode":      req.AttackMode,
		"hash_type":        req.HashType,
		"chunk_duration":   req.ChunkDuration,
	})

	// Get the next available keyspace start position
	keyspaceStart, _, err := s.jobTaskRepo.GetNextKeyspaceRange(ctx, req.JobExecution.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next keyspace range: %w", err)
	}

	debug.Log("Retrieved keyspace start position", map[string]interface{}{
		"job_execution_id": req.JobExecution.ID,
		"keyspace_start":   keyspaceStart,
	})

	// If job has no total keyspace, we can't calculate chunks properly
	if req.JobExecution.TotalKeyspace == nil {
		debug.Log("Job has no total keyspace, using alternative calculation", map[string]interface{}{
			"job_execution_id": req.JobExecution.ID,
		})
		return s.calculateChunkWithoutKeyspace(ctx, req, keyspaceStart)
	}

	totalKeyspace := *req.JobExecution.TotalKeyspace
	remainingKeyspace := totalKeyspace - keyspaceStart

	debug.Log("Calculated remaining keyspace", map[string]interface{}{
		"job_execution_id":  req.JobExecution.ID,
		"total_keyspace":    totalKeyspace,
		"keyspace_start":    keyspaceStart,
		"remaining_keyspace": remainingKeyspace,
	})

	if remainingKeyspace <= 0 {
		return nil, fmt.Errorf("no remaining keyspace for job")
	}

	// Get agent benchmark for this attack mode and hash type
	benchmarkSpeed, err := s.getOrEstimateBenchmark(ctx, req.Agent.ID, req.AttackMode, req.HashType)
	if err != nil {
		return nil, fmt.Errorf("failed to get benchmark: %w", err)
	}

	// Calculate chunk size based on benchmark and desired duration
	desiredChunkSize := int64(req.ChunkDuration) * benchmarkSpeed
	
	// Get fluctuation percentage setting
	fluctuationSetting, err := s.systemSettingsRepo.GetSetting(ctx, "chunk_fluctuation_percentage")
	if err != nil {
		return nil, fmt.Errorf("failed to get fluctuation setting: %w", err)
	}

	fluctuationPercentage := 20 // Default
	if fluctuationSetting != nil {
		if fluctuationSetting.Value != nil {
			if parsed, parseErr := parseIntValue(*fluctuationSetting.Value); parseErr == nil {
				fluctuationPercentage = parsed
			}
		}
	}

	// Check if this would be the last chunk
	keyspaceEnd := keyspaceStart + desiredChunkSize
	isLastChunk := false
	actualDuration := req.ChunkDuration

	if keyspaceEnd >= totalKeyspace {
		// This is the last chunk
		keyspaceEnd = totalKeyspace
		isLastChunk = true
		actualDuration = int((totalKeyspace - keyspaceStart) / benchmarkSpeed)
	} else {
		// Check if the remaining keyspace after this chunk would be too small
		remainingAfterChunk := totalKeyspace - keyspaceEnd
		fluctuationThreshold := int64(float64(desiredChunkSize) * float64(fluctuationPercentage) / 100.0)
		
		if remainingAfterChunk <= fluctuationThreshold {
			// Merge the final small chunk into this one
			keyspaceEnd = totalKeyspace
			isLastChunk = true
			actualDuration = int((totalKeyspace - keyspaceStart) / benchmarkSpeed)
			
			debug.Log("Merging final chunk to avoid small remainder", map[string]interface{}{
				"remaining_after_chunk":   remainingAfterChunk,
				"fluctuation_threshold":   fluctuationThreshold,
				"adjusted_keyspace_end":   keyspaceEnd,
				"adjusted_duration":       actualDuration,
			})
		}
	}

	result := &ChunkCalculationResult{
		KeyspaceStart:  keyspaceStart,
		KeyspaceEnd:    keyspaceEnd,
		BenchmarkSpeed: &benchmarkSpeed,
		ActualDuration: actualDuration,
		IsLastChunk:    isLastChunk,
	}

	debug.Log("Chunk calculated", map[string]interface{}{
		"keyspace_start":   keyspaceStart,
		"keyspace_end":     keyspaceEnd,
		"benchmark_speed":  benchmarkSpeed,
		"actual_duration":  actualDuration,
		"is_last_chunk":    isLastChunk,
	})

	return result, nil
}

// calculateChunkWithoutKeyspace handles chunk calculation for attacks that don't support keyspace
func (s *JobChunkingService) calculateChunkWithoutKeyspace(ctx context.Context, req ChunkCalculationRequest, keyspaceStart int64) (*ChunkCalculationResult, error) {
	// We no longer support chunk calculation without keyspace
	// All jobs must have a calculated keyspace for proper distributed workload management
	return nil, fmt.Errorf("keyspace calculation is required for all job types - attack mode %d does not support chunking without keyspace", req.AttackMode)
}

// getOrEstimateBenchmark gets the benchmark for an agent or estimates one if not available
func (s *JobChunkingService) getOrEstimateBenchmark(ctx context.Context, agentID int, attackMode models.AttackMode, hashType int) (int64, error) {
	// Try to get existing benchmark
	benchmark, err := s.benchmarkRepo.GetAgentBenchmark(ctx, agentID, attackMode, hashType)
	if err == nil {
		// Check if benchmark is recent enough
		cacheDurationSetting, err := s.systemSettingsRepo.GetSetting(ctx, "benchmark_cache_duration_hours")
		if err != nil {
			return benchmark.Speed, nil // Use existing benchmark if we can't check cache duration
		}

		cacheDurationHours := 168 // Default 7 days
		if cacheDurationSetting.Value != nil {
			if parsed, parseErr := parseIntValue(*cacheDurationSetting.Value); parseErr == nil {
				cacheDurationHours = parsed
			}
		}

		cacheDuration := time.Duration(cacheDurationHours) * time.Hour
		isRecent, err := s.benchmarkRepo.IsRecentBenchmark(ctx, agentID, attackMode, hashType, cacheDuration)
		if err == nil && isRecent {
			return benchmark.Speed, nil
		}
	}

	// No recent benchmark found, estimate based on similar benchmarks
	agentBenchmarks, err := s.benchmarkRepo.GetAgentBenchmarks(ctx, agentID)
	if err != nil || len(agentBenchmarks) == 0 {
		// No benchmarks available, use a conservative estimate
		return s.getDefaultBenchmarkEstimate(attackMode, hashType), nil
	}

	// Calculate average speed from existing benchmarks
	var totalSpeed int64
	var count int
	for _, bench := range agentBenchmarks {
		totalSpeed += bench.Speed
		count++
	}

	if count == 0 {
		return s.getDefaultBenchmarkEstimate(attackMode, hashType), nil
	}

	averageSpeed := totalSpeed / int64(count)
	
	// Apply attack mode modifier to the average
	modifier := s.getAttackModeSpeedModifier(attackMode)
	estimatedSpeed := int64(float64(averageSpeed) * modifier)

	debug.Log("Estimated benchmark speed", map[string]interface{}{
		"agent_id":         agentID,
		"attack_mode":      attackMode,
		"hash_type":        hashType,
		"average_speed":    averageSpeed,
		"modifier":         modifier,
		"estimated_speed":  estimatedSpeed,
	})

	return estimatedSpeed, nil
}

// getDefaultBenchmarkEstimate provides conservative default benchmark estimates
func (s *JobChunkingService) getDefaultBenchmarkEstimate(attackMode models.AttackMode, hashType int) int64 {
	baseSpeed := int64(1000000) // 1M hashes/sec baseline

	// Adjust based on attack mode complexity
	switch attackMode {
	case models.AttackModeStraight:
		return baseSpeed * 2 // Dictionary attacks are typically faster
	case models.AttackModeCombination:
		return baseSpeed     // Combination attacks are moderate
	case models.AttackModeBruteForce:
		return baseSpeed / 2 // Brute force is slower
	case models.AttackModeHybridWordlistMask, models.AttackModeHybridMaskWordlist:
		return baseSpeed / 3 // Hybrid attacks are slower
	default:
		return baseSpeed / 10 // Very conservative for unknown modes
	}
}

// getAttackModeSpeedModifier returns a speed modifier for different attack modes
func (s *JobChunkingService) getAttackModeSpeedModifier(attackMode models.AttackMode) float64 {
	switch attackMode {
	case models.AttackModeStraight:
		return 1.2 // Dictionary attacks are typically faster
	case models.AttackModeCombination:
		return 1.0 // Baseline
	case models.AttackModeBruteForce:
		return 0.8 // Brute force is slower
	case models.AttackModeHybridWordlistMask, models.AttackModeHybridMaskWordlist:
		return 0.6 // Hybrid attacks are slower
	default:
		return 0.5 // Conservative for unknown modes
	}
}

// EstimateJobCompletion estimates when a job will complete based on current progress
func (s *JobChunkingService) EstimateJobCompletion(ctx context.Context, jobExecution *models.JobExecution) (*time.Time, error) {
	if jobExecution.TotalKeyspace == nil || *jobExecution.TotalKeyspace == 0 {
		return nil, fmt.Errorf("cannot estimate completion without total keyspace")
	}

	totalKeyspace := *jobExecution.TotalKeyspace
	remainingKeyspace := totalKeyspace - jobExecution.ProcessedKeyspace

	if remainingKeyspace <= 0 {
		// Job is already complete
		now := time.Now()
		return &now, nil
	}

	// Get active tasks for this job
	tasks, err := s.jobTaskRepo.GetTasksByJobExecution(ctx, jobExecution.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job tasks: %w", err)
	}

	// Calculate average speed from running tasks
	var totalSpeed int64
	var runningTasks int
	for _, task := range tasks {
		if task.Status == models.JobTaskStatusRunning && task.BenchmarkSpeed != nil {
			totalSpeed += *task.BenchmarkSpeed
			runningTasks++
		}
	}

	if runningTasks == 0 {
		return nil, fmt.Errorf("no running tasks to estimate completion")
	}

	// Calculate estimated completion time
	avgSpeed := totalSpeed / int64(runningTasks)
	estimatedSeconds := remainingKeyspace / avgSpeed
	estimatedCompletion := time.Now().Add(time.Duration(estimatedSeconds) * time.Second)

	debug.Log("Job completion estimated", map[string]interface{}{
		"job_execution_id":     jobExecution.ID,
		"remaining_keyspace":   remainingKeyspace,
		"average_speed":        avgSpeed,
		"estimated_seconds":    estimatedSeconds,
		"estimated_completion": estimatedCompletion,
	})

	return &estimatedCompletion, nil
}

// parseIntValue safely parses an integer value with error handling
func parseIntValue(value string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("empty value")
	}
	
	result := 0
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid integer: %s", value)
		}
		result = result*10 + int(char-'0')
	}
	return result, nil
}
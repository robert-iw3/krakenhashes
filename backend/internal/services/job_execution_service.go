package services

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/binary"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)



// JobExecutionService handles job execution orchestration
type JobExecutionService struct {
	jobExecRepo       *repository.JobExecutionRepository
	jobTaskRepo       *repository.JobTaskRepository
	benchmarkRepo     *repository.BenchmarkRepository
	agentHashlistRepo *repository.AgentHashlistRepository
	agentRepo         *repository.AgentRepository
	deviceRepo        *repository.AgentDeviceRepository
	presetJobRepo     repository.PresetJobRepository
	hashlistRepo      *repository.HashListRepository
	systemSettingsRepo *repository.SystemSettingsRepository
	fileRepo          *repository.FileRepository
	binaryManager     binary.Manager
	ruleSplitManager  *RuleSplitManager
	
	// Configuration paths
	hashcatBinaryPath string
	dataDirectory     string
}

// NewJobExecutionService creates a new job execution service
func NewJobExecutionService(
	jobExecRepo *repository.JobExecutionRepository,
	jobTaskRepo *repository.JobTaskRepository,
	benchmarkRepo *repository.BenchmarkRepository,
	agentHashlistRepo *repository.AgentHashlistRepository,
	agentRepo *repository.AgentRepository,
	deviceRepo *repository.AgentDeviceRepository,
	presetJobRepo repository.PresetJobRepository,
	hashlistRepo *repository.HashListRepository,
	systemSettingsRepo *repository.SystemSettingsRepository,
	fileRepo *repository.FileRepository,
	binaryManager binary.Manager,
	hashcatBinaryPath string,
	dataDirectory string,
) *JobExecutionService {
	debug.Log("Creating JobExecutionService", map[string]interface{}{
		"data_directory": dataDirectory,
		"is_absolute":    filepath.IsAbs(dataDirectory),
	})
	
	// Create rule split manager with temp directory
	ruleSplitDir := filepath.Join(dataDirectory, "temp", "rule_chunks")
	ruleSplitManager := NewRuleSplitManager(ruleSplitDir, fileRepo)
	
	return &JobExecutionService{
		jobExecRepo:        jobExecRepo,
		jobTaskRepo:        jobTaskRepo,
		benchmarkRepo:      benchmarkRepo,
		agentHashlistRepo:  agentHashlistRepo,
		agentRepo:          agentRepo,
		deviceRepo:         deviceRepo,
		presetJobRepo:      presetJobRepo,
		hashlistRepo:       hashlistRepo,
		systemSettingsRepo: systemSettingsRepo,
		fileRepo:           fileRepo,
		binaryManager:      binaryManager,
		ruleSplitManager:   ruleSplitManager,
		hashcatBinaryPath:  hashcatBinaryPath,
		dataDirectory:      dataDirectory,
	}
}

// CreateJobExecution creates a new job execution from a preset job and hashlist
func (s *JobExecutionService) CreateJobExecution(ctx context.Context, presetJobID uuid.UUID, hashlistID int64, createdBy *uuid.UUID) (*models.JobExecution, error) {
	debug.Log("Creating job execution", map[string]interface{}{
		"preset_job_id": presetJobID,
		"hashlist_id":   hashlistID,
	})

	// Get the preset job
	presetJob, err := s.presetJobRepo.GetByID(ctx, presetJobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get preset job: %w", err)
	}

	// Get the hashlist
	hashlist, err := s.hashlistRepo.GetByID(ctx, hashlistID)
	if err != nil {
		return nil, fmt.Errorf("failed to get hashlist: %w", err)
	}

	// Use pre-calculated keyspace from preset job if available
	var totalKeyspace *int64
	if presetJob.Keyspace != nil && *presetJob.Keyspace > 0 {
		totalKeyspace = presetJob.Keyspace
		debug.Log("Using pre-calculated keyspace from preset job", map[string]interface{}{
			"preset_job_id": presetJobID,
			"keyspace": *totalKeyspace,
		})
	} else {
		// Fallback to calculating keyspace if not pre-calculated
		debug.Warning("Preset job has no pre-calculated keyspace, calculating now")
		totalKeyspace, err = s.calculateKeyspace(ctx, presetJob, hashlist)
		if err != nil {
			debug.Error("Failed to calculate keyspace: %v", err)
			return nil, fmt.Errorf("keyspace calculation is required for job execution: %w", err)
		}
	}

	// Create job execution
	jobExecution := &models.JobExecution{
		PresetJobID:       presetJobID,
		HashlistID:        hashlistID,
		Status:            models.JobExecutionStatusPending,
		Priority:          presetJob.Priority,
		TotalKeyspace:     totalKeyspace,
		ProcessedKeyspace: 0,
		AttackMode:        presetJob.AttackMode,
		MaxAgents:         presetJob.MaxAgents,
		CreatedBy:         createdBy,
	}

	err = s.jobExecRepo.Create(ctx, jobExecution)
	if err != nil {
		return nil, fmt.Errorf("failed to create job execution: %w", err)
	}

	// Calculate effective keyspace after creating the job
	err = s.calculateEffectiveKeyspace(ctx, jobExecution, presetJob)
	if err != nil {
		// Log the error but don't fail - we can still use the base keyspace
		debug.Log("Failed to calculate effective keyspace", map[string]interface{}{
			"job_execution_id": jobExecution.ID,
			"error": err.Error(),
		})
	}

	// Determine if rule splitting should be used
	if jobExecution.AttackMode == models.AttackModeStraight && jobExecution.EffectiveKeyspace != nil {
		err = s.determineRuleSplitting(ctx, jobExecution, presetJob)
		if err != nil {
			debug.Log("Failed to determine rule splitting", map[string]interface{}{
				"job_execution_id": jobExecution.ID,
				"error": err.Error(),
			})
		}
	}

	debug.Log("Job execution created", map[string]interface{}{
		"job_execution_id": jobExecution.ID,
		"total_keyspace":   totalKeyspace,
		"effective_keyspace": jobExecution.EffectiveKeyspace,
		"multiplication_factor": jobExecution.MultiplicationFactor,
		"uses_rule_splitting": jobExecution.UsesRuleSplitting,
	})

	return jobExecution, nil
}

// calculateKeyspace calculates the total keyspace for a job using hashcat --keyspace
func (s *JobExecutionService) calculateKeyspace(ctx context.Context, presetJob *models.PresetJob, hashlist *models.HashList) (*int64, error) {
	// Get the hashcat binary path from binary manager
	hashcatPath, err := s.binaryManager.GetLocalBinaryPath(ctx, int64(presetJob.BinaryVersionID))
	if err != nil {
		return nil, fmt.Errorf("failed to get hashcat binary path: %w", err)
	}

	// Build hashcat command for keyspace calculation
	// For keyspace calculation, we don't need -m (hash type) or the hash file
	// We only need the attack-specific inputs
	var args []string

	// Add attack-specific arguments
	switch presetJob.AttackMode {
	case models.AttackModeStraight: // Dictionary attack (-a 0)
		// For straight attack, only need wordlist(s) and optionally rules
		// The keyspace is the number of words in the wordlist (or with rules applied)
		for _, wordlistIDStr := range presetJob.WordlistIDs {
			wordlistPath, err := s.resolveWordlistPath(ctx, wordlistIDStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist path: %w", err)
			}
			args = append(args, wordlistPath)
		}
		// Add rules if any (rules don't change the keyspace command, but hashcat will calculate accordingly)
		for _, ruleIDStr := range presetJob.RuleIDs {
			rulePath, err := s.resolveRulePath(ctx, ruleIDStr)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve rule path: %w", err)
			}
			args = append(args, "-r", rulePath)
		}

	case models.AttackModeCombination: // Combinator attack
		if len(presetJob.WordlistIDs) >= 2 {
			wordlist1Path, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[0])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist1 path: %w", err)
			}
			wordlist2Path, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[1])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist2 path: %w", err)
			}
			args = append(args, wordlist1Path, wordlist2Path)
		}

	case models.AttackModeBruteForce: // Mask attack
		if presetJob.Mask != "" {
			args = append(args, presetJob.Mask)
		}

	case models.AttackModeHybridWordlistMask: // Hybrid Wordlist + Mask
		if len(presetJob.WordlistIDs) > 0 && presetJob.Mask != "" {
			wordlistPath, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[0])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist path: %w", err)
			}
			args = append(args, wordlistPath, presetJob.Mask)
		}

	case models.AttackModeHybridMaskWordlist: // Hybrid Mask + Wordlist
		if presetJob.Mask != "" && len(presetJob.WordlistIDs) > 0 {
			wordlistPath, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[0])
			if err != nil {
				return nil, fmt.Errorf("failed to resolve wordlist path: %w", err)
			}
			args = append(args, presetJob.Mask, wordlistPath)
		}

	default:
		return nil, fmt.Errorf("unsupported attack mode for keyspace calculation: %d", presetJob.AttackMode)
	}

	// Add keyspace flag
	args = append(args, "--keyspace")

	debug.Log("Calculating keyspace", map[string]interface{}{
		"command": hashcatPath,
		"args":    args,
		"attack_mode": presetJob.AttackMode,
	})

	// Execute hashcat command with timeout
	// Increase timeout to 2 minutes to allow for large wordlist processing
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	startTime := time.Now()
	cmd := exec.CommandContext(ctx, hashcatPath, args...)
	
	// Log current working directory for debugging
	cwd, _ := os.Getwd()
	debug.Log("Executing hashcat command", map[string]interface{}{
		"working_dir": cwd,
		"command":     hashcatPath,
		"args":        args,
	})
	
	// Capture stdout and stderr separately
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err = cmd.Run()
	if err != nil {
		// Log the full output for debugging
		debug.Error("Hashcat keyspace calculation failed", map[string]interface{}{
			"error":       err.Error(),
			"stdout":      stdout.String(),
			"stderr":      stderr.String(),
			"working_dir": cwd,
			"command":     hashcatPath,
			"args":        args,
		})
		return nil, fmt.Errorf("hashcat keyspace calculation failed: %w\nstdout: %s\nstderr: %s", err, stdout.String(), stderr.String())
	}

	// Parse keyspace from output
	// The keyspace should be the last line of stdout (ignoring stderr warnings about invalid rules)
	outputLines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(outputLines) == 0 {
		return nil, fmt.Errorf("no output from hashcat keyspace calculation")
	}
	
	// Get the last non-empty line
	var keyspaceStr string
	for i := len(outputLines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(outputLines[i])
		if line != "" {
			keyspaceStr = line
			break
		}
	}
	
	keyspace, err := strconv.ParseInt(keyspaceStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse keyspace '%s': %w", keyspaceStr, err)
	}

	if keyspace <= 0 {
		return nil, fmt.Errorf("invalid keyspace: %d", keyspace)
	}
	
	duration := time.Since(startTime)
	debug.Log("Keyspace calculated successfully", map[string]interface{}{
		"keyspace": keyspace,
		"duration": duration.String(),
		"stderr_warnings": stderr.String(),
	})

	return &keyspace, nil
}

// parseAttackMode extracts the attack mode from a preset job
func (s *JobExecutionService) parseAttackMode(presetJob *models.PresetJob) int {
	return int(presetJob.AttackMode)
}

// extractRuleFiles returns the rule file paths from a preset job
func (s *JobExecutionService) extractRuleFiles(ctx context.Context, presetJob *models.PresetJob) ([]string, error) {
	var rulePaths []string
	for _, ruleIDStr := range presetJob.RuleIDs {
		rulePath, err := s.resolveRulePath(ctx, ruleIDStr)
		if err != nil {
			debug.Log("Failed to resolve rule path", map[string]interface{}{
				"rule_id": ruleIDStr,
				"error": err.Error(),
			})
			continue // Skip invalid rules
		}
		rulePaths = append(rulePaths, rulePath)
	}
	return rulePaths, nil
}

// extractWordlists returns the wordlist file paths from a preset job
func (s *JobExecutionService) extractWordlists(ctx context.Context, presetJob *models.PresetJob) ([]string, error) {
	var wordlistPaths []string
	for _, wordlistIDStr := range presetJob.WordlistIDs {
		wordlistPath, err := s.resolveWordlistPath(ctx, wordlistIDStr)
		if err != nil {
			debug.Log("Failed to resolve wordlist path", map[string]interface{}{
				"wordlist_id": wordlistIDStr,
				"error": err.Error(),
			})
			continue // Skip invalid wordlists
		}
		wordlistPaths = append(wordlistPaths, wordlistPath)
	}
	return wordlistPaths, nil
}

// countRulesInFile counts the number of rules in a rule file
func (s *JobExecutionService) countRulesInFile(ctx context.Context, rulePath string) (int, error) {
	// For now, we'll use a simple line count
	// In a real implementation, this might use a rule manager or more sophisticated parsing
	file, err := os.Open(rulePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open rule file: %w", err)
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and comments
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to read rule file: %w", err)
	}

	return count, nil
}

// calculateWordlistKeyspace calculates the keyspace for a single wordlist
func (s *JobExecutionService) calculateWordlistKeyspace(ctx context.Context, wordlistPath string) (int64, error) {
	// For a simple wordlist, the keyspace is the number of lines
	file, err := os.Open(wordlistPath)
	if err != nil {
		return 0, fmt.Errorf("failed to open wordlist file: %w", err)
	}
	defer file.Close()

	var count int64
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to read wordlist file: %w", err)
	}

	return count, nil
}

// calculateEffectiveKeyspace computes the true workload accounting for rules/combinations
func (s *JobExecutionService) calculateEffectiveKeyspace(ctx context.Context, job *models.JobExecution, presetJob *models.PresetJob) error {
	// Use existing total_keyspace as base
	if job.TotalKeyspace == nil {
		return fmt.Errorf("job has no total keyspace calculated")
	}
	
	baseKeyspace := *job.TotalKeyspace
	attackMode := s.parseAttackMode(presetJob)
	
	debug.Log("Calculating effective keyspace", map[string]interface{}{
		"job_id": job.ID,
		"base_keyspace": baseKeyspace,
		"attack_mode": attackMode,
	})
	
	switch models.AttackMode(attackMode) {
	case models.AttackModeStraight: // Straight attack
		ruleFiles, err := s.extractRuleFiles(ctx, presetJob)
		if err != nil {
			return fmt.Errorf("failed to extract rule files: %w", err)
		}
		
		if len(ruleFiles) > 0 {
			totalRules := 1
			for _, ruleFile := range ruleFiles {
				count, err := s.countRulesInFile(ctx, ruleFile)
				if err != nil {
					debug.Log("Failed to count rules in file", map[string]interface{}{
						"rule_file": ruleFile,
						"error": err.Error(),
					})
					// Use 1 as fallback to avoid zeroing out the multiplication
					count = 1
				}
				totalRules *= count
			}
			
			job.BaseKeyspace = &baseKeyspace
			job.MultiplicationFactor = totalRules
			effectiveKeyspace := baseKeyspace * int64(totalRules)
			job.EffectiveKeyspace = &effectiveKeyspace
			
			debug.Log("Straight attack with rules", map[string]interface{}{
				"rule_files": len(ruleFiles),
				"total_rules": totalRules,
				"effective_keyspace": effectiveKeyspace,
			})
		} else {
			// No rules, effective = base
			job.BaseKeyspace = &baseKeyspace
			job.MultiplicationFactor = 1
			job.EffectiveKeyspace = &baseKeyspace
		}
		
	case models.AttackModeCombination: // Combination attack
		wordlists, err := s.extractWordlists(ctx, presetJob)
		if err != nil {
			return fmt.Errorf("failed to extract wordlists: %w", err)
		}
		
		if len(wordlists) >= 2 {
			keyspace1, err := s.calculateWordlistKeyspace(ctx, wordlists[0])
			if err != nil {
				return fmt.Errorf("failed to calculate keyspace for wordlist 1: %w", err)
			}
			
			keyspace2, err := s.calculateWordlistKeyspace(ctx, wordlists[1])
			if err != nil {
				return fmt.Errorf("failed to calculate keyspace for wordlist 2: %w", err)
			}
			
			// The base keyspace from hashcat is the larger wordlist
			job.BaseKeyspace = &baseKeyspace
			
			// Multiplication factor is the smaller wordlist
			if keyspace1 > keyspace2 {
				job.MultiplicationFactor = int(keyspace2)
			} else {
				job.MultiplicationFactor = int(keyspace1)
			}
			
			effectiveKeyspace := keyspace1 * keyspace2
			job.EffectiveKeyspace = &effectiveKeyspace
			
			debug.Log("Combination attack", map[string]interface{}{
				"wordlist1_keyspace": keyspace1,
				"wordlist2_keyspace": keyspace2,
				"effective_keyspace": effectiveKeyspace,
			})
		} else {
			// Not enough wordlists for combination
			job.BaseKeyspace = &baseKeyspace
			job.MultiplicationFactor = 1
			job.EffectiveKeyspace = &baseKeyspace
		}
		
	case models.AttackModeAssociation: // Association attack
		ruleFiles, err := s.extractRuleFiles(ctx, presetJob)
		if err != nil {
			return fmt.Errorf("failed to extract rule files: %w", err)
		}
		
		if len(ruleFiles) > 0 {
			totalRules := 0
			for _, ruleFile := range ruleFiles {
				count, err := s.countRulesInFile(ctx, ruleFile)
				if err != nil {
					debug.Log("Failed to count rules in file", map[string]interface{}{
						"rule_file": ruleFile,
						"error": err.Error(),
					})
					continue
				}
				totalRules += count
			}
			
			baseKeyspace := int64(1)
			job.BaseKeyspace = &baseKeyspace
			job.MultiplicationFactor = totalRules
			effectiveKeyspace := int64(totalRules)
			job.EffectiveKeyspace = &effectiveKeyspace
			
			debug.Log("Association attack", map[string]interface{}{
				"rule_files": len(ruleFiles),
				"total_rules": totalRules,
				"effective_keyspace": effectiveKeyspace,
			})
		} else {
			baseKeyspace := int64(1)
			job.BaseKeyspace = &baseKeyspace
			job.MultiplicationFactor = 1
			job.EffectiveKeyspace = &baseKeyspace
		}
		
	default: // Attacks 3, 6, 7 - hashcat calculates correctly
		job.BaseKeyspace = &baseKeyspace
		job.MultiplicationFactor = 1
		job.EffectiveKeyspace = &baseKeyspace
		
		debug.Log("Standard attack mode", map[string]interface{}{
			"attack_mode": attackMode,
			"keyspace": baseKeyspace,
		})
	}
	
	// Update job in database
	return s.jobExecRepo.UpdateKeyspaceInfo(ctx, job)
}

// determineRuleSplitting determines if a job should use rule splitting
func (s *JobExecutionService) determineRuleSplitting(ctx context.Context, job *models.JobExecution, presetJob *models.PresetJob) error {
	// Check if rule splitting is enabled
	ruleSplitEnabled, err := s.systemSettingsRepo.GetSetting(ctx, "rule_split_enabled")
	if err != nil || ruleSplitEnabled.Value == nil || *ruleSplitEnabled.Value != "true" {
		return nil // Rule splitting not enabled
	}

	// Only for attack mode 0 with rules
	if job.AttackMode != models.AttackModeStraight || len(presetJob.RuleIDs) == 0 {
		return nil
	}

	// Get settings
	thresholdSetting, err := s.systemSettingsRepo.GetSetting(ctx, "rule_split_threshold")
	if err != nil {
		return fmt.Errorf("failed to get rule split threshold: %w", err)
	}
	threshold := 2.0
	if thresholdSetting.Value != nil {
		if val, err := strconv.ParseFloat(*thresholdSetting.Value, 64); err == nil {
			threshold = val
		}
	}

	minRulesSetting, err := s.systemSettingsRepo.GetSetting(ctx, "rule_split_min_rules")
	if err != nil {
		return fmt.Errorf("failed to get min rules setting: %w", err)
	}
	minRules := 100
	if minRulesSetting.Value != nil {
		if val, err := strconv.Atoi(*minRulesSetting.Value); err == nil {
			minRules = val
		}
	}

	// Check if we have enough rules to split
	if job.MultiplicationFactor < minRules {
		return nil // Not enough rules to split
	}

	// Get chunk duration
	chunkDurationSetting, err := s.systemSettingsRepo.GetSetting(ctx, "default_chunk_duration")
	if err != nil {
		return fmt.Errorf("failed to get chunk duration: %w", err)
	}
	chunkDuration := 1200 // 20 minutes default
	if chunkDurationSetting.Value != nil {
		if val, err := strconv.Atoi(*chunkDurationSetting.Value); err == nil {
			chunkDuration = val
		}
	}

	// Estimate job duration at a reasonable speed (300MH/s)
	estimatedSpeed := int64(300_000_000) // 300 MH/s
	estimatedDuration := float64(*job.EffectiveKeyspace) / float64(estimatedSpeed)
	
	// Check if job duration exceeds threshold
	if estimatedDuration > float64(chunkDuration)*threshold {
		job.UsesRuleSplitting = true
		
		// Calculate number of splits needed
		numSplits := int(estimatedDuration / float64(chunkDuration))
		if numSplits < 2 {
			numSplits = 2
		}
		
		// Get max chunks setting
		maxChunksSetting, err := s.systemSettingsRepo.GetSetting(ctx, "rule_split_max_chunks")
		if err == nil && maxChunksSetting.Value != nil {
			if maxChunks, err := strconv.Atoi(*maxChunksSetting.Value); err == nil && numSplits > maxChunks {
				numSplits = maxChunks
			}
		}
		
		job.RuleSplitCount = numSplits
		
		debug.Log("Rule splitting enabled for job", map[string]interface{}{
			"job_id": job.ID,
			"effective_keyspace": *job.EffectiveKeyspace,
			"estimated_duration": estimatedDuration,
			"chunk_duration": chunkDuration,
			"threshold": threshold,
			"num_splits": numSplits,
		})
		
		// Update job in database
		return s.jobExecRepo.UpdateKeyspaceInfo(ctx, job)
	}
	
	return nil
}

// GetNextPendingJob returns the next job to be executed based on priority and FIFO
// DEPRECATED: Use GetNextJobWithWork instead
func (s *JobExecutionService) GetNextPendingJob(ctx context.Context) (*models.JobExecution, error) {
	debug.Log("Getting next pending job", nil)
	
	pendingJobs, err := s.jobExecRepo.GetPendingJobs(ctx)
	if err != nil {
		debug.Log("Failed to get pending jobs from repository", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to get pending jobs: %w", err)
	}

	debug.Log("Retrieved pending jobs", map[string]interface{}{
		"count": len(pendingJobs),
	})

	if len(pendingJobs) == 0 {
		return nil, nil // No pending jobs
	}

	// Jobs are already ordered by priority DESC, created_at ASC in the repository
	nextJob := &pendingJobs[0]
	debug.Log("Selected next job", map[string]interface{}{
		"job_id":        nextJob.ID,
		"priority":      nextJob.Priority,
		"preset_job":    nextJob.PresetJobName,
		"hashlist":      nextJob.HashlistName,
	})
	
	return nextJob, nil
}

// GetNextJobWithWork returns the next job that has work available and isn't at max agent capacity
// Jobs are ordered by priority DESC, created_at ASC (FIFO for same priority)
func (s *JobExecutionService) GetNextJobWithWork(ctx context.Context) (*models.JobExecutionWithWork, error) {
	debug.Log("Getting next job with available work", nil)
	
	jobsWithWork, err := s.jobExecRepo.GetJobsWithPendingWork(ctx)
	if err != nil {
		debug.Log("Failed to get jobs with pending work", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to get jobs with pending work: %w", err)
	}

	debug.Log("Retrieved jobs with pending work", map[string]interface{}{
		"count": len(jobsWithWork),
	})

	if len(jobsWithWork) == 0 {
		return nil, nil // No jobs with available work
	}

	// Jobs are already filtered and ordered correctly by the repository
	nextJob := &jobsWithWork[0]
	debug.Log("Selected next job with work", map[string]interface{}{
		"job_id":        nextJob.ID,
		"priority":      nextJob.Priority,
		"preset_job":    nextJob.PresetJobName,
		"hashlist":      nextJob.HashlistName,
		"active_agents": nextJob.ActiveAgents,
		"max_agents":    nextJob.MaxAgents,
		"pending_work":  nextJob.PendingWork,
		"status":        nextJob.Status,
	})
	
	return nextJob, nil
}

// GetAvailableAgents returns agents that are available to take on new work
func (s *JobExecutionService) GetAvailableAgents(ctx context.Context) ([]models.Agent, error) {
	// Get max concurrent jobs per agent setting
	maxConcurrentSetting, err := s.systemSettingsRepo.GetSetting(ctx, "max_concurrent_jobs_per_agent")
	if err != nil {
		return nil, fmt.Errorf("failed to get max concurrent jobs setting: %w", err)
	}

	maxConcurrent := 2 // Default value
	if maxConcurrentSetting.Value != nil {
		if parsed, parseErr := strconv.Atoi(*maxConcurrentSetting.Value); parseErr == nil {
			maxConcurrent = parsed
		}
	}

	// Get all active agents
	agents, err := s.agentRepo.List(ctx, map[string]interface{}{"status": models.AgentStatusActive})
	if err != nil {
		return nil, fmt.Errorf("failed to get active agents: %w", err)
	}

	debug.Log("Found active agents", map[string]interface{}{
		"agent_count": len(agents),
	})

	var availableAgents []models.Agent
	for _, agent := range agents {
		debug.Log("Checking agent availability", map[string]interface{}{
			"agent_id":   agent.ID,
			"agent_name": agent.Name,
			"status":     agent.Status,
			"is_enabled": agent.IsEnabled,
		})
		
		// Skip disabled agents (maintenance mode)
		if !agent.IsEnabled {
			debug.Log("Agent is disabled (maintenance mode), skipping", map[string]interface{}{
				"agent_id": agent.ID,
			})
			continue
		}
		
		// Count active tasks for this agent
		activeTasks, err := s.jobTaskRepo.GetActiveTasksByAgent(ctx, agent.ID)
		if err != nil {
			debug.Log("Failed to get active tasks for agent", map[string]interface{}{
				"agent_id": agent.ID,
				"error":    err.Error(),
			})
			continue
		}

		debug.Log("Agent task status", map[string]interface{}{
			"agent_id":       agent.ID,
			"active_tasks":   len(activeTasks),
			"max_concurrent": maxConcurrent,
			"is_available":   len(activeTasks) < maxConcurrent,
		})

		if len(activeTasks) < maxConcurrent {
			// Check if agent has enabled devices
			hasEnabledDevices, err := s.deviceRepo.HasEnabledDevices(agent.ID)
			if err != nil {
				debug.Log("Failed to check enabled devices for agent", map[string]interface{}{
					"agent_id": agent.ID,
					"error":    err.Error(),
				})
				continue
			}
			
			if hasEnabledDevices {
				availableAgents = append(availableAgents, agent)
			} else {
				debug.Log("Agent has no enabled devices, skipping", map[string]interface{}{
					"agent_id": agent.ID,
				})
			}
		}
	}

	return availableAgents, nil
}

// CreateJobTask creates a task chunk for an agent
func (s *JobExecutionService) CreateJobTask(ctx context.Context, jobExecution *models.JobExecution, agent *models.Agent, keyspaceStart, keyspaceEnd int64, benchmarkSpeed *int64) (*models.JobTask, error) {
	// Get chunk duration setting
	chunkDurationSetting, err := s.systemSettingsRepo.GetSetting(ctx, "default_chunk_duration")
	if err != nil {
		return nil, fmt.Errorf("failed to get chunk duration setting: %w", err)
	}

	chunkDuration := 1200 // Default 20 minutes
	if chunkDurationSetting.Value != nil {
		if parsed, parseErr := strconv.Atoi(*chunkDurationSetting.Value); parseErr == nil {
			chunkDuration = parsed
		}
	}

	jobTask := &models.JobTask{
		JobExecutionID:    jobExecution.ID,
		AgentID:           &agent.ID,
		Status:            models.JobTaskStatusPending,
		KeyspaceStart:     keyspaceStart,
		KeyspaceEnd:       keyspaceEnd,
		KeyspaceProcessed: 0,
		BenchmarkSpeed:    benchmarkSpeed,
		ChunkDuration:     chunkDuration,
	}

	err = s.jobTaskRepo.Create(ctx, jobTask)
	if err != nil {
		return nil, fmt.Errorf("failed to create job task: %w", err)
	}

	debug.Log("Job task created", map[string]interface{}{
		"task_id":         jobTask.ID,
		"agent_id":        agent.ID,
		"keyspace_start":  keyspaceStart,
		"keyspace_end":    keyspaceEnd,
		"chunk_duration":  chunkDuration,
	})

	return jobTask, nil
}

// StartJobExecution marks a job execution as started
func (s *JobExecutionService) StartJobExecution(ctx context.Context, jobExecutionID uuid.UUID) error {
	err := s.jobExecRepo.StartExecution(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to start job execution: %w", err)
	}

	debug.Log("Job execution started", map[string]interface{}{
		"job_execution_id": jobExecutionID,
	})

	return nil
}

// CompleteJobExecution marks a job execution as completed
func (s *JobExecutionService) CompleteJobExecution(ctx context.Context, jobExecutionID uuid.UUID) error {
	err := s.jobExecRepo.CompleteExecution(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to complete job execution: %w", err)
	}

	debug.Log("Job execution completed", map[string]interface{}{
		"job_execution_id": jobExecutionID,
	})

	return nil
}

// UpdateJobProgress updates the progress of a job execution
func (s *JobExecutionService) UpdateJobProgress(ctx context.Context, jobExecutionID uuid.UUID, processedKeyspace int64) error {
	err := s.jobExecRepo.UpdateProgress(ctx, jobExecutionID, processedKeyspace)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}

	return nil
}

// UpdateTaskProgress updates the progress of a task accounting for rule splitting
func (s *JobExecutionService) UpdateTaskProgress(ctx context.Context, taskID uuid.UUID, keyspaceProcessed int64, hashRate *int64, progressPercent float64) error {
	// Update the task progress first
	err := s.jobTaskRepo.UpdateProgress(ctx, taskID, keyspaceProcessed, hashRate, progressPercent)
	if err != nil {
		return fmt.Errorf("failed to update task progress: %w", err)
	}
	
	// Get the task to find the job execution
	task, err := s.jobTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	
	// Calculate total job progress
	totalProgress, err := s.calculateTotalJobProgress(ctx, task.JobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to calculate total job progress: %w", err)
	}
	
	// Calculate overall progress percentage
	overallPercent, err := s.calculateOverallProgressPercent(ctx, task.JobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to calculate overall progress percent: %w", err)
	}
	
	// Update job execution progress
	err = s.UpdateJobProgress(ctx, task.JobExecutionID, totalProgress)
	if err != nil {
		return fmt.Errorf("failed to update job progress: %w", err)
	}
	
	// Update overall progress percentage
	err = s.UpdateJobProgressPercent(ctx, task.JobExecutionID, overallPercent)
	if err != nil {
		return fmt.Errorf("failed to update job progress percent: %w", err)
	}
	
	return nil
}

// calculateTotalJobProgress aggregates progress accounting for effective keyspace and rule splitting
func (s *JobExecutionService) calculateTotalJobProgress(ctx context.Context, jobID uuid.UUID) (int64, error) {
	job, err := s.jobExecRepo.GetByID(ctx, jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to get job execution: %w", err)
	}
	
	tasks, err := s.jobTaskRepo.GetTasksByJobExecution(ctx, jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to get tasks: %w", err)
	}
	
	var totalProgress int64
	
	if job.UsesRuleSplitting {
		// Sum effective progress from all rule chunks
		for _, task := range tasks {
			if task.IsRuleSplitTask && task.RuleStartIndex != nil && task.RuleEndIndex != nil {
				// Calculate rules in this chunk
				rulesInChunk := (*task.RuleEndIndex - *task.RuleStartIndex)
				// Effective progress = wordlist progress × rules in chunk
				chunkProgress := task.KeyspaceProcessed * int64(rulesInChunk)
				totalProgress += chunkProgress
			} else {
				// Non-rule split task in a job that uses rule splitting
				totalProgress += task.KeyspaceProcessed
			}
		}
	} else if job.MultiplicationFactor > 1 {
		// Virtual keyspace tracking (e.g., combination attack)
		for _, task := range tasks {
			totalProgress += task.KeyspaceProcessed * int64(job.MultiplicationFactor)
		}
	} else {
		// Standard progress aggregation
		for _, task := range tasks {
			totalProgress += task.KeyspaceProcessed
		}
	}
	
	debug.Log("Calculated total job progress", map[string]interface{}{
		"job_id": jobID,
		"total_progress": totalProgress,
		"uses_rule_splitting": job.UsesRuleSplitting,
		"multiplication_factor": job.MultiplicationFactor,
		"task_count": len(tasks),
	})
	
	return totalProgress, nil
}

// calculateOverallProgressPercent calculates the overall progress percentage for a job
func (s *JobExecutionService) calculateOverallProgressPercent(ctx context.Context, jobID uuid.UUID) (float64, error) {
	job, err := s.jobExecRepo.GetByID(ctx, jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to get job execution: %w", err)
	}
	
	tasks, err := s.jobTaskRepo.GetTasksByJobExecution(ctx, jobID)
	if err != nil {
		return 0, fmt.Errorf("failed to get tasks: %w", err)
	}
	
	var overallPercent float64
	completedTasks := 0
	var activeTaskProgress float64
	
	if job.UsesRuleSplitting {
		// For rule-based chunking: calculate based on completed chunks and active progress
		hasActiveTask := false
		totalRuleSplitTasks := 0
		
		for _, task := range tasks {
			if task.IsRuleSplitTask {
				totalRuleSplitTasks++
				if task.Status == models.JobTaskStatusCompleted {
					completedTasks++
				} else if task.Status == models.JobTaskStatusRunning || task.Status == models.JobTaskStatusAssigned {
					// Use the progress of the currently active task
					activeTaskProgress = task.ProgressPercent
					hasActiveTask = true
				}
			}
		}
		
		// Calculate overall progress based on completed tasks + active task progress
		if totalRuleSplitTasks > 0 {
			completedPercent := (float64(completedTasks) / float64(totalRuleSplitTasks)) * 100
			if hasActiveTask {
				// Add the partial progress of the active task
				activeTaskContribution := (activeTaskProgress / 100.0) * (100.0 / float64(totalRuleSplitTasks))
				overallPercent = completedPercent + activeTaskContribution
			} else {
				overallPercent = completedPercent
			}
			
			debug.Log("Rule-split job progress calculation", map[string]interface{}{
				"job_id": jobID,
				"total_rule_split_tasks": totalRuleSplitTasks,
				"total_tasks": len(tasks),
				"completed_tasks": completedTasks,
				"active_task_progress": activeTaskProgress,
				"has_active_task": hasActiveTask,
				"completed_percent": completedPercent,
				"overall_percent": overallPercent,
			})
		}
	} else {
		// For keyspace-based chunking: sum of keyspace processed / total keyspace
		if job.EffectiveKeyspace != nil && *job.EffectiveKeyspace > 0 {
			totalProcessed := int64(0)
			for _, task := range tasks {
				totalProcessed += task.KeyspaceProcessed
			}
			overallPercent = (float64(totalProcessed) / float64(*job.EffectiveKeyspace)) * 100
		} else if job.TotalKeyspace != nil && *job.TotalKeyspace > 0 {
			totalProcessed := int64(0)
			for _, task := range tasks {
				totalProcessed += task.KeyspaceProcessed
			}
			overallPercent = (float64(totalProcessed) / float64(*job.TotalKeyspace)) * 100
		}
	}
	
	// Ensure percentage is within bounds
	if overallPercent > 100 {
		overallPercent = 100
	}
	
	debug.Log("Calculated overall job progress percentage", map[string]interface{}{
		"job_id": jobID,
		"overall_percent": overallPercent,
		"uses_rule_splitting": job.UsesRuleSplitting,
		"total_tasks": len(tasks),
	})
	
	return overallPercent, nil
}

// UpdateJobProgressPercent updates the overall progress percentage for a job execution
func (s *JobExecutionService) UpdateJobProgressPercent(ctx context.Context, jobExecutionID uuid.UUID, progressPercent float64) error {
	err := s.jobExecRepo.UpdateProgressPercent(ctx, jobExecutionID, progressPercent)
	if err != nil {
		return fmt.Errorf("failed to update job progress percent: %w", err)
	}
	
	return nil
}

// UpdateCrackedCount updates the total number of cracked hashes for a job execution
// DEPRECATED: This method is deprecated as cracked counts are now tracked at the hashlist level
func (s *JobExecutionService) UpdateCrackedCount(ctx context.Context, jobExecutionID uuid.UUID, additionalCracks int) error {
	// This method is deprecated and should not be used
	// Cracked counts are now tracked on the hashlists table, not job_executions
	debug.Log("WARNING: UpdateCrackedCount called on job execution service (deprecated)", map[string]interface{}{
		"job_id":            jobExecutionID,
		"additional_cracks": additionalCracks,
	})
	return nil
}

// CanInterruptJob checks if a job can be interrupted by a higher priority job
func (s *JobExecutionService) CanInterruptJob(ctx context.Context, newJobPriority int) ([]models.JobExecution, error) {
	// Check if job interruption is enabled
	interruptionSetting, err := s.systemSettingsRepo.GetSetting(ctx, "job_interruption_enabled")
	if err != nil {
		return nil, fmt.Errorf("failed to get interruption setting: %w", err)
	}

	if interruptionSetting.Value == nil || *interruptionSetting.Value != "true" {
		return []models.JobExecution{}, nil // Interruption disabled
	}

	// Get interruptible jobs with lower priority
	interruptibleJobs, err := s.jobExecRepo.GetInterruptibleJobs(ctx, newJobPriority)
	if err != nil {
		return nil, fmt.Errorf("failed to get interruptible jobs: %w", err)
	}

	return interruptibleJobs, nil
}

// InterruptJob interrupts a running job for a higher priority job
func (s *JobExecutionService) InterruptJob(ctx context.Context, jobExecutionID, interruptingJobID uuid.UUID) error {
	err := s.jobExecRepo.InterruptExecution(ctx, jobExecutionID, interruptingJobID)
	if err != nil {
		return fmt.Errorf("failed to interrupt job: %w", err)
	}

	// Cancel all running tasks for this job
	tasks, err := s.jobTaskRepo.GetTasksByJobExecution(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get tasks for interrupted job: %w", err)
	}

	for _, task := range tasks {
		if task.Status == models.JobTaskStatusRunning {
			err = s.jobTaskRepo.CancelTask(ctx, task.ID)
			if err != nil {
				debug.Log("Failed to cancel task", map[string]interface{}{
					"task_id": task.ID,
					"error":   err.Error(),
				})
			}
		}
	}

	debug.Log("Job interrupted", map[string]interface{}{
		"job_execution_id":      jobExecutionID,
		"interrupting_job_id":   interruptingJobID,
	})

	return nil
}
// GetSystemSetting retrieves a system setting by key (public method for integration)
func (s *JobExecutionService) GetSystemSetting(ctx context.Context, key string) (int, error) {
	setting, err := s.systemSettingsRepo.GetSetting(ctx, key)
	if err != nil {
		return 0, err
	}
	
	if setting.Value == nil {
		return 0, fmt.Errorf("setting value is null")
	}
	
	value, err := strconv.Atoi(*setting.Value)
	if err != nil {
		return 0, fmt.Errorf("invalid setting value: %w", err)
	}
	
	return value, nil
}

// GetJobExecutionByID retrieves a job execution by ID (public method for integration)
func (s *JobExecutionService) GetJobExecutionByID(ctx context.Context, id uuid.UUID) (*models.JobExecution, error) {
	return s.jobExecRepo.GetByID(ctx, id)
}

// RetryFailedChunk attempts to retry a failed job task chunk
func (s *JobExecutionService) RetryFailedChunk(ctx context.Context, taskID uuid.UUID) error {
	debug.Log("Attempting to retry failed chunk", map[string]interface{}{
		"task_id": taskID,
	})

	// Get the current task
	task, err := s.jobTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Get max retry attempts from system settings
	maxRetryAttempts, err := s.GetSystemSetting(ctx, "max_chunk_retry_attempts")
	if err != nil {
		debug.Log("Failed to get max retry attempts, using default", map[string]interface{}{
			"error": err.Error(),
		})
		maxRetryAttempts = 3 // Default fallback
	}

	// Check if we can retry
	if task.RetryCount >= maxRetryAttempts {
		debug.Log("Maximum retry attempts reached", map[string]interface{}{
			"task_id":     taskID,
			"retry_count": task.RetryCount,
			"max_retries": maxRetryAttempts,
		})
		
		// Mark as permanently failed
		err = s.jobTaskRepo.UpdateTaskStatus(ctx, taskID, "failed", "failed")
		if err != nil {
			return fmt.Errorf("failed to mark task as permanently failed: %w", err)
		}
		
		return fmt.Errorf("maximum retry attempts (%d) exceeded for task %s", maxRetryAttempts, taskID)
	}

	// Reset task for retry
	err = s.jobTaskRepo.ResetTaskForRetry(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to reset task for retry: %w", err)
	}

	debug.Log("Chunk reset for retry", map[string]interface{}{
		"task_id":     taskID,
		"retry_count": task.RetryCount + 1,
	})

	return nil
}

// ProcessFailedChunks automatically retries failed chunks based on system settings
func (s *JobExecutionService) ProcessFailedChunks(ctx context.Context, jobExecutionID uuid.UUID) error {
	debug.Log("Processing failed chunks for job", map[string]interface{}{
		"job_execution_id": jobExecutionID,
	})

	// Get all failed tasks for this job execution
	failedTasks, err := s.jobTaskRepo.GetFailedTasksByJobExecution(ctx, jobExecutionID)
	if err != nil {
		return fmt.Errorf("failed to get failed tasks: %w", err)
	}

	retriedCount := 0
	permanentFailureCount := 0

	for _, task := range failedTasks {
		err := s.RetryFailedChunk(ctx, task.ID)
		if err != nil {
			debug.Log("Failed to retry chunk", map[string]interface{}{
				"task_id": task.ID,
				"error":   err.Error(),
			})
			permanentFailureCount++
		} else {
			retriedCount++
		}
	}

	debug.Log("Completed failed chunk processing", map[string]interface{}{
		"job_execution_id":       jobExecutionID,
		"retried_count":          retriedCount,
		"permanent_failure_count": permanentFailureCount,
		"total_failed_tasks":     len(failedTasks),
	})

	return nil
}

// UpdateChunkStatusWithCracks updates a chunk's status and crack count
func (s *JobExecutionService) UpdateChunkStatusWithCracks(ctx context.Context, taskID uuid.UUID, crackCount int, detailedStatus string) error {
	debug.Log("Updating chunk status with crack information", map[string]interface{}{
		"task_id":         taskID,
		"crack_count":     crackCount,
		"detailed_status": detailedStatus,
	})

	err := s.jobTaskRepo.UpdateTaskWithCracks(ctx, taskID, crackCount, detailedStatus)
	if err != nil {
		return fmt.Errorf("failed to update task with cracks: %w", err)
	}

	return nil
}

// GetDynamicChunkSize calculates optimal chunk size based on agent benchmark data
func (s *JobExecutionService) GetDynamicChunkSize(ctx context.Context, agentID int, attackMode int, hashType int, defaultDurationSeconds int) (int64, error) {
	debug.Log("Calculating dynamic chunk size", map[string]interface{}{
		"agent_id":        agentID,
		"attack_mode":     attackMode,
		"hash_type":       hashType,
		"default_duration": defaultDurationSeconds,
	})

	// Get agent benchmark for this specific attack mode and hash type
	benchmark, err := s.benchmarkRepo.GetAgentBenchmark(ctx, agentID, models.AttackMode(attackMode), hashType)
	if err != nil {
		debug.Log("No benchmark found, using default chunk size", map[string]interface{}{
			"agent_id":    agentID,
			"attack_mode": attackMode,
			"hash_type":   hashType,
			"error":       err.Error(),
		})
		// Return a default chunk size (e.g., 1M keyspace)
		return 1000000, nil
	}

	// Calculate keyspace size for the default duration
	// keyspace = benchmark_speed * duration_seconds
	keyspaceSize := benchmark.Speed * int64(defaultDurationSeconds)

	debug.Log("Dynamic chunk size calculated", map[string]interface{}{
		"agent_id":        agentID,
		"benchmark_speed": benchmark.Speed,
		"duration":        defaultDurationSeconds,
		"keyspace_size":   keyspaceSize,
	})

	return keyspaceSize, nil
}

// resolveWordlistPath gets the actual file path for a wordlist ID
func (s *JobExecutionService) resolveWordlistPath(ctx context.Context, wordlistIDStr string) (string, error) {
	// Try to parse as integer ID first
	if wordlistID, err := strconv.Atoi(wordlistIDStr); err == nil {
		// Look up wordlist in database
		wordlists, err := s.fileRepo.GetWordlists(ctx, "")
		if err != nil {
			return "", fmt.Errorf("failed to get wordlists: %w", err)
		}
		
		for _, wl := range wordlists {
			if wl.ID == wordlistID {
				// The Name field now contains category/filename (e.g., "general/crackstation.txt")
				// We need to use just the filename without duplicating the category
				filename := wl.Name
				
				// If the Name already contains the category path, extract just the filename
				if strings.Contains(wl.Name, "/") {
					filename = filepath.Base(wl.Name)
				}
				
				// Build absolute path using the data directory
				var path string
				if wl.Category != "" {
					path = filepath.Join(s.dataDirectory, "wordlists", wl.Category, filename)
				} else {
					path = filepath.Join(s.dataDirectory, "wordlists", filename)
				}
				
				debug.Log("Resolved wordlist path", map[string]interface{}{
					"wordlist_id": wordlistID,
					"category":    wl.Category,
					"name_field":  wl.Name,
					"filename":    filename,
					"path":        path,
				})
				return path, nil
			}
		}
		return "", fmt.Errorf("wordlist with ID %d not found", wordlistID)
	}
	
	// If not a numeric ID, treat as a filename
	path := filepath.Join(s.dataDirectory, "wordlists", wordlistIDStr)
	debug.Log("Resolved wordlist path from string", map[string]interface{}{
		"wordlist_str": wordlistIDStr,
		"path":         path,
	})
	return path, nil
}

// resolveRulePath gets the actual file path for a rule ID
func (s *JobExecutionService) resolveRulePath(ctx context.Context, ruleIDStr string) (string, error) {
	// Try to parse as integer ID first
	if ruleID, err := strconv.Atoi(ruleIDStr); err == nil {
		// Look up rule in database
		rules, err := s.fileRepo.GetRules(ctx, "")
		if err != nil {
			return "", fmt.Errorf("failed to get rules: %w", err)
		}
		
		for _, rule := range rules {
			if rule.ID == ruleID {
				// The Name field now contains category/filename (e.g., "hashcat/wordlist_2f26acbe.txt")
				// We need to use just the filename without duplicating the category
				filename := rule.Name
				
				// If the Name already contains the category path, extract just the filename
				if strings.Contains(rule.Name, "/") {
					filename = filepath.Base(rule.Name)
				}
				
				// Build absolute path using the data directory
				var path string
				if rule.Category != "" {
					path = filepath.Join(s.dataDirectory, "rules", rule.Category, filename)
				} else {
					path = filepath.Join(s.dataDirectory, "rules", filename)
				}
				
				debug.Log("Resolved rule path", map[string]interface{}{
					"rule_id":    ruleID,
					"category":   rule.Category,
					"name_field": rule.Name,
					"filename":   filename,
					"path":       path,
				})
				return path, nil
			}
		}
		return "", fmt.Errorf("rule with ID %d not found", ruleID)
	}
	
	// If not a numeric ID, treat as a filename
	path := filepath.Join(s.dataDirectory, "rules", ruleIDStr)
	debug.Log("Resolved rule path from string", map[string]interface{}{
		"rule_str": ruleIDStr,
		"path":     path,
	})
	return path, nil
}

// RuleSplitDecision contains the decision information for rule splitting
type RuleSplitDecision struct {
	ShouldSplit     bool
	NumSplits       int
	RuleFileToSplit string
	RulesPerChunk   int
	TotalRules      int
}

// analyzeForRuleSplitting determines if rule splitting should be used for a job
func (s *JobExecutionService) analyzeForRuleSplitting(ctx context.Context, job *models.JobExecution, presetJob *models.PresetJob, benchmarkSpeed float64) (*RuleSplitDecision, error) {
	// Check if rule splitting is enabled
	ruleSplitEnabled, err := s.systemSettingsRepo.GetSetting(ctx, "rule_split_enabled")
	if err != nil || ruleSplitEnabled.Value == nil || *ruleSplitEnabled.Value != "true" {
		return &RuleSplitDecision{ShouldSplit: false}, nil
	}
	
	// Only applicable for attacks 0 and 9 with rules
	if job.AttackMode != models.AttackModeStraight && job.AttackMode != models.AttackModeAssociation {
		return &RuleSplitDecision{ShouldSplit: false}, nil
	}
	
	if job.MultiplicationFactor <= 1 {
		return &RuleSplitDecision{ShouldSplit: false}, nil
	}
	
	// For attack mode 9 (association), always split if rules present
	if job.AttackMode == models.AttackModeAssociation {
		return s.createSplitDecision(ctx, job, presetJob, benchmarkSpeed)
	}
	
	// For attack mode 0, check thresholds
	thresholdSetting, err := s.systemSettingsRepo.GetSetting(ctx, "rule_split_threshold")
	if err != nil {
		debug.Log("Failed to get rule split threshold, using default", map[string]interface{}{
			"error": err.Error(),
		})
	}
	threshold := 2.0 // Default
	if thresholdSetting != nil && thresholdSetting.Value != nil {
		if parsed, parseErr := strconv.ParseFloat(*thresholdSetting.Value, 64); parseErr == nil {
			threshold = parsed
		}
	}
	
	minRulesSetting, err := s.systemSettingsRepo.GetSetting(ctx, "rule_split_min_rules")
	if err != nil {
		debug.Log("Failed to get min rules setting, using default", map[string]interface{}{
			"error": err.Error(),
		})
	}
	minRules := 100 // Default
	if minRulesSetting != nil && minRulesSetting.Value != nil {
		if parsed, parseErr := strconv.Atoi(*minRulesSetting.Value); parseErr == nil {
			minRules = parsed
		}
	}
	
	// Calculate estimated time
	effectiveKeyspace := job.EffectiveKeyspace
	if effectiveKeyspace == nil {
		if job.TotalKeyspace != nil {
			effectiveKeyspace = job.TotalKeyspace
		} else {
			return &RuleSplitDecision{ShouldSplit: false}, nil
		}
	}
	
	estimatedTimeSeconds := float64(*effectiveKeyspace) / benchmarkSpeed
	
	chunkDurationSetting, err := s.systemSettingsRepo.GetSetting(ctx, "default_chunk_duration")
	if err != nil {
		debug.Log("Failed to get chunk duration, using default", map[string]interface{}{
			"error": err.Error(),
		})
	}
	chunkDuration := 1200.0 // Default 20 minutes
	if chunkDurationSetting != nil && chunkDurationSetting.Value != nil {
		if parsed, parseErr := strconv.ParseFloat(*chunkDurationSetting.Value, 64); parseErr == nil {
			chunkDuration = parsed
		}
	}
	
	debug.Log("Analyzing for rule splitting", map[string]interface{}{
		"job_id":            job.ID,
		"attack_mode":       job.AttackMode,
		"multiplication_factor": job.MultiplicationFactor,
		"estimated_time":    estimatedTimeSeconds,
		"chunk_duration":    chunkDuration,
		"threshold":         threshold,
		"min_rules":         minRules,
	})
	
	if estimatedTimeSeconds > chunkDuration * threshold && job.MultiplicationFactor >= minRules {
		return s.createSplitDecision(ctx, job, presetJob, benchmarkSpeed)
	}
	
	return &RuleSplitDecision{ShouldSplit: false}, nil
}

// createSplitDecision creates a rule split decision for a job
func (s *JobExecutionService) createSplitDecision(ctx context.Context, job *models.JobExecution, presetJob *models.PresetJob, benchmarkSpeed float64) (*RuleSplitDecision, error) {
	// Get rule files
	ruleFiles, err := s.extractRuleFiles(ctx, presetJob)
	if err != nil {
		return nil, fmt.Errorf("failed to extract rule files: %w", err)
	}
	
	if len(ruleFiles) == 0 {
		return &RuleSplitDecision{ShouldSplit: false}, nil
	}
	
	// For simplicity, we'll split the first rule file
	// In a more advanced implementation, we might split multiple files
	ruleFileToSplit := ruleFiles[0]
	
	// Count rules in the file
	totalRules, err := s.ruleSplitManager.CountRules(ctx, ruleFileToSplit)
	if err != nil {
		return nil, fmt.Errorf("failed to count rules: %w", err)
	}
	
	// Get max chunks setting
	maxChunksSetting, err := s.systemSettingsRepo.GetSetting(ctx, "rule_split_max_chunks")
	if err != nil {
		debug.Log("Failed to get max chunks setting, using default", map[string]interface{}{
			"error": err.Error(),
		})
	}
	maxChunks := 1000 // Default
	if maxChunksSetting != nil && maxChunksSetting.Value != nil {
		if parsed, parseErr := strconv.Atoi(*maxChunksSetting.Value); parseErr == nil {
			maxChunks = parsed
		}
	}
	
	// Calculate optimal number of splits
	chunkDurationSetting, err := s.systemSettingsRepo.GetSetting(ctx, "default_chunk_duration")
	if err != nil {
		debug.Log("Failed to get chunk duration, using default", map[string]interface{}{
			"error": err.Error(),
		})
	}
	chunkDuration := 1200.0 // Default 20 minutes in seconds
	if chunkDurationSetting != nil && chunkDurationSetting.Value != nil {
		if parsed, parseErr := strconv.ParseFloat(*chunkDurationSetting.Value, 64); parseErr == nil {
			chunkDuration = parsed
		}
	}
	
	// Calculate how many rules we can process in chunk duration
	var baseKeyspace int64
	if job.BaseKeyspace != nil {
		baseKeyspace = *job.BaseKeyspace
	} else if job.TotalKeyspace != nil {
		baseKeyspace = *job.TotalKeyspace
	} else {
		baseKeyspace = 1000000 // Default fallback
	}
	
	// Rules we can process in chunk duration = (benchmark_speed * chunk_duration) / base_keyspace
	rulesPerChunkIdeal := int((benchmarkSpeed * chunkDuration) / float64(baseKeyspace))
	if rulesPerChunkIdeal < 1 {
		rulesPerChunkIdeal = 1
	}
	
	// Calculate number of splits needed
	numSplits := (totalRules + rulesPerChunkIdeal - 1) / rulesPerChunkIdeal
	if numSplits > maxChunks {
		numSplits = maxChunks
	}
	if numSplits < 1 {
		numSplits = 1
	}
	
	rulesPerChunk := (totalRules + numSplits - 1) / numSplits
	
	debug.Log("Created split decision", map[string]interface{}{
		"job_id":               job.ID,
		"rule_file":            ruleFileToSplit,
		"total_rules":          totalRules,
		"num_splits":           numSplits,
		"rules_per_chunk":      rulesPerChunk,
		"rules_per_chunk_ideal": rulesPerChunkIdeal,
		"base_keyspace":        baseKeyspace,
		"benchmark_speed":      benchmarkSpeed,
	})
	
	return &RuleSplitDecision{
		ShouldSplit:     true,
		NumSplits:       numSplits,
		RuleFileToSplit: ruleFileToSplit,
		RulesPerChunk:   rulesPerChunk,
		TotalRules:      totalRules,
	}, nil
}

// createJobTasksWithRuleSplitting creates job tasks with rule splitting if needed
func (s *JobExecutionService) createJobTasksWithRuleSplitting(ctx context.Context, job *models.JobExecution, presetJob *models.PresetJob, decision *RuleSplitDecision) error {
	if !decision.ShouldSplit {
		// Standard single task creation - this will be handled by JobChunkingService
		return nil
	}
	
	// Split the rule file
	// Convert UUID to int64 for the job ID parameter
	jobIDInt := int64(job.ID[0])<<56 | int64(job.ID[1])<<48 | int64(job.ID[2])<<40 | int64(job.ID[3])<<32 |
		int64(job.ID[4])<<24 | int64(job.ID[5])<<16 | int64(job.ID[6])<<8 | int64(job.ID[7])
	chunks, err := s.ruleSplitManager.SplitRuleFile(ctx, jobIDInt, decision.RuleFileToSplit, decision.NumSplits)
	if err != nil {
		return fmt.Errorf("failed to split rules: %w", err)
	}
	
	// Update job metadata
	job.UsesRuleSplitting = true
	job.RuleSplitCount = len(chunks)
	if err := s.jobExecRepo.UpdateKeyspaceInfo(ctx, job); err != nil {
		// Cleanup chunks on error
		s.ruleSplitManager.CleanupJobChunks(jobIDInt)
		return fmt.Errorf("failed to update job metadata: %w", err)
	}
	
	// Get the attack command template from preset job
	attackCmd, err := s.buildAttackCommand(ctx, presetJob, job)
	if err != nil {
		s.ruleSplitManager.CleanupJobChunks(jobIDInt)
		return fmt.Errorf("failed to build attack command: %w", err)
	}
	
	// Get chunk duration setting
	chunkDurationSetting, err := s.systemSettingsRepo.GetSetting(ctx, "default_chunk_duration")
	if err != nil {
		s.ruleSplitManager.CleanupJobChunks(jobIDInt)
		return fmt.Errorf("failed to get chunk duration setting: %w", err)
	}
	
	chunkDuration := 1200 // Default 20 minutes
	if chunkDurationSetting.Value != nil {
		if parsed, parseErr := strconv.Atoi(*chunkDurationSetting.Value); parseErr == nil {
			chunkDuration = parsed
		}
	}
	
	// Create task for each chunk
	var baseKeyspace int64
	if job.BaseKeyspace != nil {
		baseKeyspace = *job.BaseKeyspace
	} else if job.TotalKeyspace != nil {
		baseKeyspace = *job.TotalKeyspace
	} else {
		baseKeyspace = 0
	}
	
	for i, chunk := range chunks {
		task := &models.JobTask{
			JobExecutionID:  job.ID,
			Status:          models.JobTaskStatusPending,
			Priority:        job.Priority,
			KeyspaceStart:   0,                  // Full wordlist
			KeyspaceEnd:     baseKeyspace,       // Full wordlist
			RuleStartIndex:  &chunk.StartIndex,
			RuleEndIndex:    &chunk.EndIndex,
			RuleChunkPath:   &chunk.Path,
			IsRuleSplitTask: true,
			ChunkDuration:   chunkDuration,
		}
		
		// Modify attack command to use chunk file
		task.AttackCmd = strings.Replace(attackCmd, decision.RuleFileToSplit, chunk.Path, 1)
		
		if err := s.jobTaskRepo.Create(ctx, task); err != nil {
			// Cleanup on error
			s.ruleSplitManager.CleanupJobChunks(jobIDInt)
			return fmt.Errorf("failed to create task %d: %w", i, err)
		}
		
		debug.Log("Created rule split task", map[string]interface{}{
			"task_id":        task.ID,
			"job_id":         job.ID,
			"chunk_index":    i,
			"rule_start":     chunk.StartIndex,
			"rule_end":       chunk.EndIndex,
			"chunk_path":     chunk.Path,
		})
	}
	
	return nil
}

// buildAttackCommand builds the hashcat attack command from a preset job
func (s *JobExecutionService) buildAttackCommand(ctx context.Context, presetJob *models.PresetJob, job *models.JobExecution) (string, error) {
	// Get the hashcat binary path
	hashcatPath, err := s.binaryManager.GetLocalBinaryPath(ctx, int64(presetJob.BinaryVersionID))
	if err != nil {
		return "", fmt.Errorf("failed to get hashcat binary path: %w", err)
	}
	
	// Get the hashlist path
	hashlist, err := s.hashlistRepo.GetByID(ctx, job.HashlistID)
	if err != nil {
		return "", fmt.Errorf("failed to get hashlist: %w", err)
	}
	hashlistPath := filepath.Join(s.dataDirectory, "hashlists", hashlist.FilePath)
	
	// Build the command
	var args []string
	
	// Attack mode
	args = append(args, "-a", strconv.Itoa(int(presetJob.AttackMode)))
	
	// Hash type
	args = append(args, "-m", strconv.Itoa(presetJob.HashType))
	
	// Hashlist
	args = append(args, hashlistPath)
	
	// Attack-specific arguments
	switch presetJob.AttackMode {
	case models.AttackModeStraight, models.AttackModeAssociation:
		// Add wordlists
		for _, wordlistIDStr := range presetJob.WordlistIDs {
			wordlistPath, err := s.resolveWordlistPath(ctx, wordlistIDStr)
			if err != nil {
				return "", fmt.Errorf("failed to resolve wordlist path: %w", err)
			}
			args = append(args, wordlistPath)
		}
		// Add rules
		for _, ruleIDStr := range presetJob.RuleIDs {
			rulePath, err := s.resolveRulePath(ctx, ruleIDStr)
			if err != nil {
				return "", fmt.Errorf("failed to resolve rule path: %w", err)
			}
			args = append(args, "-r", rulePath)
		}
		
	case models.AttackModeCombination:
		// Add two wordlists
		if len(presetJob.WordlistIDs) >= 2 {
			wordlist1Path, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[0])
			if err != nil {
				return "", fmt.Errorf("failed to resolve wordlist1 path: %w", err)
			}
			wordlist2Path, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[1])
			if err != nil {
				return "", fmt.Errorf("failed to resolve wordlist2 path: %w", err)
			}
			args = append(args, wordlist1Path, wordlist2Path)
		}
		
	case models.AttackModeBruteForce:
		// Add mask
		if presetJob.Mask != "" {
			args = append(args, presetJob.Mask)
		}
		
	case models.AttackModeHybridWordlistMask:
		// Add wordlist and mask
		if len(presetJob.WordlistIDs) > 0 && presetJob.Mask != "" {
			wordlistPath, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[0])
			if err != nil {
				return "", fmt.Errorf("failed to resolve wordlist path: %w", err)
			}
			args = append(args, wordlistPath, presetJob.Mask)
		}
		
	case models.AttackModeHybridMaskWordlist:
		// Add mask and wordlist
		if presetJob.Mask != "" && len(presetJob.WordlistIDs) > 0 {
			wordlistPath, err := s.resolveWordlistPath(ctx, presetJob.WordlistIDs[0])
			if err != nil {
				return "", fmt.Errorf("failed to resolve wordlist path: %w", err)
			}
			args = append(args, presetJob.Mask, wordlistPath)
		}
	}
	
	// Add any additional arguments
	if presetJob.AdditionalArgs != nil && *presetJob.AdditionalArgs != "" {
		additionalArgs := strings.Fields(*presetJob.AdditionalArgs)
		args = append(args, additionalArgs...)
	}
	
	// Join command
	fullCmd := hashcatPath + " " + strings.Join(args, " ")
	return fullCmd, nil
}

// cleanupTaskResources cleans up resources associated with a completed or failed task
func (s *JobExecutionService) cleanupTaskResources(ctx context.Context, task *models.JobTask) error {
	if !task.IsRuleSplitTask || task.RuleChunkPath == nil || *task.RuleChunkPath == "" {
		return nil
	}

	debug.Log("Cleaning up task resources", map[string]interface{}{
		"task_id":         task.ID,
		"rule_chunk_path": *task.RuleChunkPath,
	})

	// Remove rule chunk file from server
	if err := os.Remove(*task.RuleChunkPath); err != nil && !os.IsNotExist(err) {
		debug.Error("Failed to remove rule chunk file: %v", err)
		// Don't return error - continue with cleanup
	}

	// TODO: Send cleanup message to agent via WebSocket to remove the chunk file

	return nil
}

// CleanupJobResources cleans up all resources for a completed/failed/cancelled job
func (s *JobExecutionService) CleanupJobResources(ctx context.Context, jobID uuid.UUID) error {
	debug.Log("Cleaning up job resources", map[string]interface{}{
		"job_id": jobID,
	})

	// Get job execution
	job, err := s.jobExecRepo.GetByID(ctx, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job execution: %w", err)
	}

	// If this job uses rule splitting, clean up all chunks
	if job.UsesRuleSplitting {
		// Convert UUID to int64 for job ID
		// Note: This assumes job IDs in the file system are based on a numeric representation
		// In production, you might want to use the UUID string directly
		jobIDInt := int64(job.ID[0]) // Simple conversion for demo - use proper conversion in production
		err = s.ruleSplitManager.CleanupJobChunks(jobIDInt)
		if err != nil {
			debug.Error("Failed to cleanup rule chunks for job: %v", err)
			// Don't return error - continue with other cleanup
		}
	}

	// Get all tasks for this job
	tasks, err := s.jobTaskRepo.GetTasksByJobExecution(ctx, jobID)
	if err != nil {
		debug.Error("Failed to get tasks for cleanup: %v", err)
		return nil // Don't fail the entire cleanup
	}

	// Cleanup each task's resources
	for _, task := range tasks {
		if err := s.cleanupTaskResources(ctx, &task); err != nil {
			debug.Error("Failed to cleanup task resources: %v", err)
			// Continue with other tasks
		}
	}

	return nil
}

// HandleTaskCompletion handles cleanup when a task completes (success or failure)
func (s *JobExecutionService) HandleTaskCompletion(ctx context.Context, taskID uuid.UUID) error {
	// Get task
	task, err := s.jobTaskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Cleanup task resources
	if err := s.cleanupTaskResources(ctx, task); err != nil {
		debug.Error("Failed to cleanup task resources on completion: %v", err)
		// Don't fail the task completion
	}

	// Check if all tasks for this job are complete
	allTasksComplete, err := s.jobTaskRepo.AreAllTasksComplete(ctx, task.JobExecutionID)
	if err != nil {
		debug.Error("Failed to check if all tasks complete: %v", err)
		return nil
	}

	if allTasksComplete {
		// Cleanup job-level resources
		if err := s.CleanupJobResources(ctx, task.JobExecutionID); err != nil {
			debug.Error("Failed to cleanup job resources: %v", err)
		}
	}

	return nil
}

// InitializeRuleSplitting initializes rule splitting for a job
func (s *JobExecutionService) InitializeRuleSplitting(ctx context.Context, job *models.JobExecution) error {
	debug.Log("InitializeRuleSplitting called", map[string]interface{}{
		"job_id": job.ID,
		"uses_rule_splitting": job.UsesRuleSplitting,
		"rule_split_count": job.RuleSplitCount,
	})
	
	if !job.UsesRuleSplitting {
		return fmt.Errorf("job does not use rule splitting")
	}

	// Get the preset job
	presetJob, err := s.presetJobRepo.GetByID(ctx, job.PresetJobID)
	if err != nil {
		return fmt.Errorf("failed to get preset job: %w", err)
	}
	
	debug.Log("Got preset job", map[string]interface{}{
		"preset_job_id": presetJob.ID,
		"rule_ids": presetJob.RuleIDs,
	})

	// Get the rule files
	ruleFiles, err := s.extractRuleFiles(ctx, presetJob)
	if err != nil {
		return fmt.Errorf("failed to extract rule files: %w", err)
	}
	
	debug.Log("Extracted rule files", map[string]interface{}{
		"rule_count": len(ruleFiles),
		"rule_files": ruleFiles,
	})

	if len(ruleFiles) == 0 {
		return fmt.Errorf("no rule files found for rule splitting")
	}

	// For now, split the first rule file
	// TODO: Handle multiple rule files
	ruleFileToSplit := ruleFiles[0]

	// Convert job ID to int64 for the split manager
	jobIDInt := int64(job.ID[0])<<56 | int64(job.ID[1])<<48 | int64(job.ID[2])<<40 | int64(job.ID[3])<<32 |
		int64(job.ID[4])<<24 | int64(job.ID[5])<<16 | int64(job.ID[6])<<8 | int64(job.ID[7])

	// Split the rule file
	debug.Log("Splitting rule file", map[string]interface{}{
		"rule_file": ruleFileToSplit,
		"num_splits": job.RuleSplitCount,
		"job_id_int": jobIDInt,
	})
	
	chunks, err := s.ruleSplitManager.SplitRuleFile(ctx, jobIDInt, ruleFileToSplit, job.RuleSplitCount)
	if err != nil {
		return fmt.Errorf("failed to split rule file: %w", err)
	}
	
	debug.Log("Rule file split successfully", map[string]interface{}{
		"num_chunks": len(chunks),
	})

	// Create tasks for each chunk
	for i, chunk := range chunks {
		// Calculate keyspace for this chunk
		baseKeyspace := *job.BaseKeyspace
		
		task := &models.JobTask{
			ID:              uuid.New(),
			JobExecutionID:  job.ID,
			Status:          models.JobTaskStatusPending,
			Priority:        job.Priority,
			KeyspaceStart:   int64(i) * baseKeyspace, // Use chunk index as multiplier
			KeyspaceEnd:     int64(i+1) * baseKeyspace,
			KeyspaceProcessed: 0,
			ChunkDuration:   300, // 5 minutes per chunk
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			// Rule splitting fields
			RuleStartIndex:  &chunk.StartIndex,
			RuleEndIndex:    &chunk.EndIndex,
			RuleChunkPath:   &chunk.Path,
			IsRuleSplitTask: true,
		}
		
		// Create the task
		err = s.jobTaskRepo.Create(ctx, task)
		if err != nil {
			// Cleanup on error
			s.ruleSplitManager.CleanupJobChunks(jobIDInt)
			return fmt.Errorf("failed to create task for chunk %d: %w", i, err)
		}
	}

	debug.Log("Created rule split tasks", map[string]interface{}{
		"job_id": job.ID,
		"num_tasks": len(chunks),
		"rule_file": ruleFileToSplit,
	})

	return nil
}

// GetNextRuleSplitTask gets the next available rule split task for an agent
func (s *JobExecutionService) GetNextRuleSplitTask(ctx context.Context, job *models.JobExecution, agent *models.Agent) (*models.JobTask, error) {
	// Get pending tasks for this job
	tasks, err := s.jobTaskRepo.GetPendingTasksByJobExecution(ctx, job.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil, nil // No pending tasks
	}

	// Assign the first pending task to the agent
	task := &tasks[0]
	task.AgentID = &agent.ID
	task.Status = models.JobTaskStatusAssigned
	task.AssignedAt = time.Now()
	task.UpdatedAt = time.Now()

	// Update the task
	err = s.jobTaskRepo.Update(ctx, task)
	if err != nil {
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	// Build the attack command for this task
	attackCmd, err := s.buildRuleSplitAttackCommand(ctx, job, task)
	if err != nil {
		return nil, fmt.Errorf("failed to build attack command: %w", err)
	}
	task.AttackCmd = attackCmd

	return task, nil
}

// buildRuleSplitAttackCommand builds the hashcat command for a rule split task
func (s *JobExecutionService) buildRuleSplitAttackCommand(ctx context.Context, job *models.JobExecution, task *models.JobTask) (string, error) {
	// Get the preset job
	presetJob, err := s.presetJobRepo.GetByID(ctx, job.PresetJobID)
	if err != nil {
		return "", fmt.Errorf("failed to get preset job: %w", err)
	}

	// Get hashlist
	hashlist, err := s.hashlistRepo.GetByID(ctx, job.HashlistID)
	if err != nil {
		return "", fmt.Errorf("failed to get hashlist: %w", err)
	}

	// Build base command
	var cmdParts []string
	cmdParts = append(cmdParts, fmt.Sprintf("-m %d", hashlist.HashTypeID))
	cmdParts = append(cmdParts, "-a 0") // Attack mode 0

	// Add wordlists
	for _, wordlistIDStr := range presetJob.WordlistIDs {
		wordlistPath, err := s.resolveWordlistPath(ctx, wordlistIDStr)
		if err != nil {
			return "", fmt.Errorf("failed to resolve wordlist path: %w", err)
		}
		cmdParts = append(cmdParts, wordlistPath)
	}

	// Add the rule chunk file
	if task.RuleChunkPath != nil {
		cmdParts = append(cmdParts, "-r", *task.RuleChunkPath)
	}

	// Add limit to match the base keyspace
	if job.BaseKeyspace != nil {
		cmdParts = append(cmdParts, "--limit", fmt.Sprintf("%d", *job.BaseKeyspace))
	}

	return strings.Join(cmdParts, " "), nil
}

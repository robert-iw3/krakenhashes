package services

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)

// AdminPresetJobService defines the interface for managing preset jobs.
type AdminPresetJobService interface {
	CreatePresetJob(ctx context.Context, params models.PresetJob) (*models.PresetJob, error)
	GetPresetJobByID(ctx context.Context, id uuid.UUID) (*models.PresetJob, error)
	ListPresetJobs(ctx context.Context) ([]models.PresetJob, error)
	UpdatePresetJob(ctx context.Context, id uuid.UUID, params models.PresetJob) (*models.PresetJob, error)
	DeletePresetJob(ctx context.Context, id uuid.UUID) error
	GetPresetJobFormData(ctx context.Context) (*repository.PresetJobFormData, error)
}

// adminPresetJobService implements AdminPresetJobService.
type adminPresetJobService struct {
	presetJobRepo      repository.PresetJobRepository
	systemSettingsRepo *repository.SystemSettingsRepository
	// Add other repositories if needed for deeper validation (e.g., binaryRepo)
	// For now, ListFormData provides enough for basic checks.
}

// NewAdminPresetJobService creates a new service for managing preset jobs.
func NewAdminPresetJobService(presetJobRepo repository.PresetJobRepository, systemSettingsRepo *repository.SystemSettingsRepository) AdminPresetJobService {
	return &adminPresetJobService{
		presetJobRepo:      presetJobRepo,
		systemSettingsRepo: systemSettingsRepo,
	}
}

// isValidAttackMode checks if the provided integer corresponds to a defined AttackMode.
func isValidAttackMode(mode models.AttackMode) bool {
	switch mode {
	case models.AttackModeStraight,
		models.AttackModeCombination,
		models.AttackModeBruteForce,
		models.AttackModeHybridWordlistMask,
		models.AttackModeHybridMaskWordlist,
		models.AttackModeAssociation:
		return true
	default:
		return false
	}
}

// validatePresetJob performs input validation for create/update operations.
func (s *adminPresetJobService) validatePresetJob(ctx context.Context, params models.PresetJob, isUpdate bool, existingID uuid.UUID) error {
	if params.Name == "" {
		return errors.New("preset job name cannot be empty")
	}

	// Check name uniqueness
	existingByName, err := s.presetJobRepo.GetByName(ctx, params.Name)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		// Error other than not found
		return fmt.Errorf("error checking name uniqueness: %w", err)
	}
	if existingByName != nil && (!isUpdate || existingByName.ID != existingID) {
		return fmt.Errorf("preset job name '%s' already exists", params.Name)
	}

	if params.Priority < 0 {
		return errors.New("priority cannot be negative")
	}

	// Check priority against system maximum
	maxPriority, err := s.systemSettingsRepo.GetMaxJobPriority(ctx)
	if err != nil {
		debug.Warning("Failed to get max priority setting, using default: %v", err)
		maxPriority = 1000 // Default fallback
	}

	if params.Priority > maxPriority {
		return fmt.Errorf("priority %d exceeds the maximum allowed priority of %d", params.Priority, maxPriority)
	}

	if params.ChunkSizeSeconds <= 0 {
		return errors.New("chunk size must be positive")
	}

	if !isValidAttackMode(params.AttackMode) {
		return fmt.Errorf("invalid attack mode: %d", params.AttackMode)
	}

	// Basic validation for ID strings in arrays (ensures they are valid integers)
	for _, idStr := range params.WordlistIDs {
		// Wordlist IDs are numeric IDs stored as strings in the database
		// Check that they can be parsed as integers
		if _, err := strconv.Atoi(idStr); err != nil {
			return fmt.Errorf("invalid wordlist ID format: %s", idStr)
		}
	}
	for _, idStr := range params.RuleIDs {
		// Rule IDs are numeric IDs stored as strings in the database
		// Check that they can be parsed as integers
		if _, err := strconv.Atoi(idStr); err != nil {
			return fmt.Errorf("invalid rule ID format: %s", idStr)
		}
	}

	// Attack mode specific validation
	switch params.AttackMode {
	case models.AttackModeStraight:
		if len(params.WordlistIDs) != 1 {
			return errors.New("straight attack mode requires exactly one wordlist")
		}
		// Rules are optional for straight mode

	case models.AttackModeCombination:
		if len(params.WordlistIDs) != 2 {
			return errors.New("combination attack mode requires exactly two wordlists")
		}
		if len(params.RuleIDs) > 0 {
			return errors.New("rules are not supported in combination attack mode")
		}

	case models.AttackModeBruteForce:
		if len(params.WordlistIDs) > 0 {
			return errors.New("wordlists are not used in brute force attack mode")
		}
		if len(params.RuleIDs) > 0 {
			return errors.New("rules are not supported in brute force attack mode")
		}
		if params.Mask == "" {
			return errors.New("mask is required for brute force attack mode")
		}
		if !validateMaskPattern(params.Mask) {
			return errors.New("invalid mask pattern format")
		}

	case models.AttackModeHybridWordlistMask, models.AttackModeHybridMaskWordlist:
		if len(params.WordlistIDs) != 1 {
			return errors.New("hybrid attack modes require exactly one wordlist")
		}
		if len(params.RuleIDs) > 0 {
			return errors.New("rules are not supported in hybrid attack modes")
		}
		if params.Mask == "" {
			return errors.New("mask is required for hybrid attack modes")
		}
		if !validateMaskPattern(params.Mask) {
			return errors.New("invalid mask pattern format")
		}

	case models.AttackModeAssociation:
		return errors.New("association attack mode is not currently implemented")
	}

	// TODO: Add deeper validation if necessary:
	// - Check if BinaryVersionID actually exists in binary_versions table.
	// - Check if all WordlistIDs/RuleIDs exist (might require fetching all valid IDs).
	//   For now, we rely on the frontend using data from GetPresetJobFormData
	//   and potentially database foreign key constraints where applicable.

	return nil
}

// validateMaskPattern validates that the mask follows the expected pattern for hashcat.
// Simple validation to check for valid character sets: ?u, ?l, ?d, ?s, ?a, ?b
// and length requirements.
func validateMaskPattern(mask string) bool {
	if mask == "" {
		return false
	}

	// Pattern should consist of character set specifiers
	// Each valid specifier is two characters: ? followed by a character class
	validSpecifiers := map[string]bool{
		"?u": true, // uppercase
		"?l": true, // lowercase
		"?d": true, // digit
		"?s": true, // special
		"?a": true, // all (uppercase, lowercase, digit, special)
		"?b": true, // binary (0x00 - 0xff)
		"?h": true, // lowercase hex
		"?H": true, // uppercase hex
	}

	i := 0
	for i < len(mask) {
		// If we encounter a literal character (not part of a specifier)
		if mask[i] != '?' {
			i++
			continue
		}

		// Check if we have enough characters for a complete specifier
		if i+1 >= len(mask) {
			return false // Incomplete specifier at end of mask
		}

		// Check if the specifier is valid
		specifier := mask[i : i+2]
		if !validSpecifiers[specifier] {
			return false // Invalid specifier
		}

		i += 2 // Move past this specifier
	}

	// Ensure mask isn't empty after validation
	return true
}

// CreatePresetJob creates a new preset job after validation.
func (s *adminPresetJobService) CreatePresetJob(ctx context.Context, params models.PresetJob) (*models.PresetJob, error) {
	if err := s.validatePresetJob(ctx, params, false, uuid.Nil); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	debug.Info("Creating preset job: %s", params.Name)
	createdJob, err := s.presetJobRepo.Create(ctx, params)
	if err != nil {
		debug.Error("Failed to create preset job in repository: %v", err)
		// TODO: Handle specific DB errors like unique constraint violations more gracefully
		return nil, fmt.Errorf("failed to create preset job: %w", err)
	}
	debug.Info("Successfully created preset job ID: %s", createdJob.ID)
	return createdJob, nil
}

// GetPresetJobByID retrieves a single preset job.
func (s *adminPresetJobService) GetPresetJobByID(ctx context.Context, id uuid.UUID) (*models.PresetJob, error) {
	debug.Debug("Getting preset job by ID: %s", id)
	job, err := s.presetJobRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			debug.Warning("Preset job not found: %s", id)
			return nil, err // Return the specific ErrNotFound
		}
		debug.Error("Failed to get preset job by ID %s: %v", id, err)
		return nil, fmt.Errorf("failed to get preset job: %w", err)
	}
	return job, nil
}

// ListPresetJobs retrieves all preset jobs.
func (s *adminPresetJobService) ListPresetJobs(ctx context.Context) ([]models.PresetJob, error) {
	debug.Debug("Listing all preset jobs")
	jobs, err := s.presetJobRepo.List(ctx)
	if err != nil {
		debug.Error("Failed to list preset jobs: %v", err)
		return nil, fmt.Errorf("failed to list preset jobs: %w", err)
	}
	return jobs, nil
}

// UpdatePresetJob updates an existing preset job after validation.
func (s *adminPresetJobService) UpdatePresetJob(ctx context.Context, id uuid.UUID, params models.PresetJob) (*models.PresetJob, error) {
	// Ensure the job exists before validating/updating
	_, err := s.GetPresetJobByID(ctx, id)
	if err != nil {
		return nil, err // Returns ErrNotFound if applicable
	}

	if err := s.validatePresetJob(ctx, params, true, id); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	debug.Info("Updating preset job ID: %s", id)
	updatedJob, err := s.presetJobRepo.Update(ctx, id, params)
	if err != nil {
		debug.Error("Failed to update preset job %s: %v", id, err)
		// TODO: Handle specific DB errors
		return nil, fmt.Errorf("failed to update preset job: %w", err)
	}
	debug.Info("Successfully updated preset job ID: %s", updatedJob.ID)
	return updatedJob, nil
}

// DeletePresetJob deletes a preset job.
func (s *adminPresetJobService) DeletePresetJob(ctx context.Context, id uuid.UUID) error {
	debug.Info("Deleting preset job ID: %s", id)
	// Check existence first (optional, repo delete also checks)
	_, err := s.GetPresetJobByID(ctx, id)
	if err != nil {
		return err
	}

	err = s.presetJobRepo.Delete(ctx, id)
	if err != nil {
		debug.Error("Failed to delete preset job %s: %v", id, err)
		// Consider if FK constraints could cause errors here if not handled by DB
		return fmt.Errorf("failed to delete preset job: %w", err)
	}
	debug.Info("Successfully deleted preset job ID: %s", id)
	return nil
}

// GetPresetJobFormData retrieves lists needed for UI forms.
func (s *adminPresetJobService) GetPresetJobFormData(ctx context.Context) (*repository.PresetJobFormData, error) {
	debug.Debug("Getting preset job form data")
	formData, err := s.presetJobRepo.ListFormData(ctx)
	if err != nil {
		debug.Error("Failed to get preset job form data: %v", err)
		return nil, fmt.Errorf("failed to get preset job form data: %w", err)
	}
	return formData, nil
}

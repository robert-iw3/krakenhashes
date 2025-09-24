package services

import (
	"bufio"
	"context"
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/wordlist"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ErrNoBinaryVersions is returned when no binary versions exist in the database
var ErrNoBinaryVersions = errors.New("no binary versions found")

// PotfileService manages the pot-file and its staging mechanism
type PotfileService struct {
	db                 *db.DB
	dataDir            string
	potfilePath        string
	systemSettingsRepo *repository.SystemSettingsRepository
	presetJobRepo      repository.PresetJobRepository
	wordlistStore      *wordlist.Store
	hashRepo           *repository.HashRepository
	jobUpdateService   *JobUpdateService
	processingMutex    sync.Mutex
	stopChan           chan struct{}
	wg                 sync.WaitGroup
	batchInterval      time.Duration
	maxBatchSize       int
}

// NewPotfileService creates a new pot-file service
func NewPotfileService(
	database *db.DB,
	dataDir string,
	systemSettingsRepo *repository.SystemSettingsRepository,
	presetJobRepo repository.PresetJobRepository,
	wordlistStore *wordlist.Store,
	hashRepo *repository.HashRepository,
	jobUpdateService *JobUpdateService,
) *PotfileService {
	potfilePath := filepath.Join(dataDir, "wordlists", "custom", "potfile.txt")

	return &PotfileService{
		db:                 database,
		dataDir:            dataDir,
		potfilePath:        potfilePath,
		systemSettingsRepo: systemSettingsRepo,
		presetJobRepo:      presetJobRepo,
		wordlistStore:      wordlistStore,
		hashRepo:           hashRepo,
		jobUpdateService:   jobUpdateService,
		stopChan:           make(chan struct{}),
		batchInterval:      60 * time.Second, // Default, will be updated from settings
		maxBatchSize:       1000,              // Default, will be updated from settings
	}
}

// Start begins the background worker for processing staged entries
func (s *PotfileService) Start(ctx context.Context) error {
	debug.Info("Starting pot-file service...")
	
	// Load settings
	debug.Debug("Loading pot-file settings...")
	if err := s.loadSettings(ctx); err != nil {
		debug.Error("Failed to load pot-file settings: %v", err)
		return fmt.Errorf("failed to load pot-file settings: %w", err)
	}
	debug.Debug("Pot-file settings loaded successfully")

	// Initialize pot-file if needed
	debug.Debug("Initializing pot-file...")
	if err := s.InitializePotfile(ctx); err != nil {
		debug.Error("Failed to initialize pot-file: %v", err)
		return fmt.Errorf("failed to initialize pot-file: %w", err)
	}
	debug.Debug("Pot-file initialized successfully")

	// Start background worker
	s.wg.Add(1)
	go s.backgroundWorker()

	debug.Info("Pot-file service started with batch interval: %v", s.batchInterval)
	return nil
}

// Stop stops the background worker
func (s *PotfileService) Stop() {
	debug.Info("Stopping pot-file service")
	close(s.stopChan)
	s.wg.Wait()
}

// StagePassword adds a password to the staging table
func (s *PotfileService) StagePassword(ctx context.Context, password, hashValue string) error {
	query := `
		INSERT INTO potfile_staging (password, hash_value)
		VALUES ($1, $2)
	`
	
	_, err := s.db.ExecContext(ctx, query, password, hashValue)
	if err != nil {
		return fmt.Errorf("failed to stage password: %w", err)
	}
	
	debug.Debug("Staged password for hash %s", hashValue)
	return nil
}

// InitializePotfile creates the pot-file and its database entries if they don't exist
func (s *PotfileService) InitializePotfile(ctx context.Context) error {
	debug.Info("InitializePotfile called, path: %s", s.potfilePath)
	s.processingMutex.Lock()
	defer s.processingMutex.Unlock()

	// Ensure directory exists
	potfileDir := filepath.Dir(s.potfilePath)
	debug.Debug("Creating pot-file directory if needed: %s", potfileDir)
	if err := os.MkdirAll(potfileDir, 0755); err != nil {
		debug.Error("Failed to create pot-file directory: %v", err)
		return fmt.Errorf("failed to create pot-file directory: %w", err)
	}

	// Check if pot-file exists
	fileExists := false
	if _, err := os.Stat(s.potfilePath); err == nil {
		fileExists = true
	}

	// Create pot-file if it doesn't exist
	if !fileExists {
		file, err := os.Create(s.potfilePath)
		if err != nil {
			return fmt.Errorf("failed to create pot-file: %w", err)
		}
		
		// Write blank first line (null password)
		if _, err := file.WriteString("\n"); err != nil {
			file.Close()
			return fmt.Errorf("failed to write initial blank line: %w", err)
		}
		file.Close()
		
		debug.Info("Created new pot-file at: %s", s.potfilePath)
	}

	// Check if wordlist entry exists
	wordlistID, err := s.getOrCreatePotfileWordlist(ctx)
	if err != nil {
		return fmt.Errorf("failed to get/create pot-file wordlist: %w", err)
	}

	// Check if preset job exists
	presetJobID, err := s.getOrCreatePotfilePresetJob(ctx, wordlistID)
	if err != nil {
		// Handle the case where no binaries exist
		if errors.Is(err, ErrNoBinaryVersions) {
			debug.Warning("No binary versions found, starting monitor to create pot-file preset job when binaries are added")
			// Update system settings with just the wordlist ID
			if err := s.updateSystemSettings(ctx, wordlistID, uuid.Nil); err != nil {
				debug.Error("Failed to update system settings with wordlist ID: %v", err)
			}
			// Start monitor in background
			s.monitorForBinaryAndCreatePresetJob(ctx, wordlistID)
			// Continue initialization - this is not fatal
		} else {
			return fmt.Errorf("failed to get/create pot-file preset job: %w", err)
		}
	} else {
		// Update system settings with both IDs
		if err := s.updateSystemSettings(ctx, wordlistID, presetJobID); err != nil {
			return fmt.Errorf("failed to update system settings: %w", err)
		}
		
		// Sync preset job with current wordlist to ensure correct wordlist ID and keyspace
		if err := s.syncPresetJobWithWordlist(ctx, wordlistID, presetJobID); err != nil {
			debug.Warning("Failed to sync preset job with wordlist: %v", err)
			// Don't fail initialization
		}
	}

	// Ensure MD5 hash is up to date after initialization
	if err := s.UpdatePotfileMetadata(ctx); err != nil {
		debug.Warning("Failed to update potfile metadata after initialization: %v", err)
		// Don't fail initialization if metadata update fails
	}

	return nil
}

// GetPotfilePath returns the path to the pot-file
func (s *PotfileService) GetPotfilePath() string {
	return s.potfilePath
}

// backgroundWorker processes staged entries periodically
func (s *PotfileService) backgroundWorker() {
	defer s.wg.Done()
	
	ticker := time.NewTicker(s.batchInterval)
	defer ticker.Stop()

	// Process immediately on start
	s.ProcessStagedEntries(context.Background())

	for {
		select {
		case <-ticker.C:
			s.ProcessStagedEntries(context.Background())
		case <-s.stopChan:
			debug.Info("Pot-file background worker stopped")
			return
		}
	}
}

// ProcessStagedEntries processes all unprocessed entries in the staging table
func (s *PotfileService) ProcessStagedEntries(ctx context.Context) {
	s.processingMutex.Lock()
	defer s.processingMutex.Unlock()

	// Get unprocessed entries
	entries, err := s.getStagedEntries(ctx)
	if err != nil {
		debug.Error("Failed to get staged entries: %v", err)
		return
	}

	if len(entries) == 0 {
		return
	}

	debug.Info("Processing %d staged pot-file entries", len(entries))

	// Load existing pot-file into memory
	existingPasswords, err := s.loadPotfileIntoMemory()
	if err != nil {
		debug.Error("Failed to load pot-file into memory: %v", err)
		return
	}

	// Filter out duplicates
	var newEntries []potfileStagingEntry
	for _, entry := range entries {
		if _, exists := existingPasswords[entry.Password]; !exists {
			newEntries = append(newEntries, entry)
			existingPasswords[entry.Password] = true // Add to map to catch duplicates within batch
		}
	}

	// Append new entries to pot-file
	if len(newEntries) > 0 {
		// Get old line count before updating
		oldLineCount, _ := s.countPotfileLines()

		if err := s.appendToPotfile(newEntries); err != nil {
			debug.Error("Failed to append to pot-file: %v", err)
			return
		}
		debug.Info("Added %d new unique entries to pot-file", len(newEntries))

		// Update MD5 hash and file size in the database
		if err := s.UpdatePotfileMetadata(ctx); err != nil {
			debug.Error("Failed to update potfile metadata: %v", err)
			// Don't return - this is not critical for the operation
		}

		// Get new line count after updating
		newLineCount, _ := s.countPotfileLines()

		// Trigger job updates if we have the service and the count changed
		if s.jobUpdateService != nil && oldLineCount != newLineCount {
			// Get the potfile wordlist ID from system settings
			wordlistIDSetting, err := s.systemSettingsRepo.GetSetting(ctx, "potfile_wordlist_id")
			if err == nil && wordlistIDSetting != nil && wordlistIDSetting.Value != nil && *wordlistIDSetting.Value != "" {
				wordlistID, err := strconv.Atoi(*wordlistIDSetting.Value)
				if err == nil {
					debug.Info("Triggering job updates for potfile wordlist %d (old: %d, new: %d)",
						wordlistID, oldLineCount, newLineCount)
					if err := s.jobUpdateService.HandleWordlistUpdate(ctx, wordlistID, oldLineCount, newLineCount); err != nil {
						debug.Error("Failed to update jobs for potfile changes: %v", err)
						// Don't return - this is not critical for the operation
					}
				}
			}
		}
	}

	// Delete processed entries from staging table
	if err := s.deleteProcessedEntries(ctx, entries); err != nil {
		debug.Error("Failed to delete processed entries: %v", err)
		return
	}

	// Trigger keyspace recalculation if needed
	if len(newEntries) > 0 {
		s.triggerKeyspaceRecalculation(ctx)
	}
}

// loadSettings loads pot-file settings from the database
func (s *PotfileService) loadSettings(ctx context.Context) error {
	// Get batch interval
	intervalSetting, err := s.systemSettingsRepo.GetSetting(ctx, "potfile_batch_interval")
	if err == nil && intervalSetting != nil && intervalSetting.Value != nil && *intervalSetting.Value != "" {
		if interval, err := time.ParseDuration(*intervalSetting.Value + "s"); err == nil {
			s.batchInterval = interval
		}
	}

	// Get max batch size
	maxBatchSetting, err := s.systemSettingsRepo.GetSetting(ctx, "potfile_max_batch_size")
	if err == nil && maxBatchSetting != nil && maxBatchSetting.Value != nil && *maxBatchSetting.Value != "" {
		if maxBatch, err := strconv.Atoi(*maxBatchSetting.Value); err == nil && maxBatch > 0 {
			s.maxBatchSize = maxBatch
		}
	}

	return nil
}

// potfileStagingEntry represents an entry in the staging table
type potfileStagingEntry struct {
	ID        int
	Password  string
	HashValue string
	CreatedAt time.Time
}

// getStagedEntries retrieves unprocessed entries from the staging table
func (s *PotfileService) getStagedEntries(ctx context.Context) ([]potfileStagingEntry, error) {
	query := `
		SELECT id, password, hash_value, created_at
		FROM potfile_staging
		WHERE processed = FALSE
		ORDER BY created_at
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, s.maxBatchSize)
	if err != nil {
		return nil, fmt.Errorf("failed to query staged entries: %w", err)
	}
	defer rows.Close()

	var entries []potfileStagingEntry
	for rows.Next() {
		var entry potfileStagingEntry
		if err := rows.Scan(&entry.ID, &entry.Password, &entry.HashValue, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan staged entry: %w", err)
		}
		entries = append(entries, entry)
	}

	return entries, rows.Err()
}

// loadPotfileIntoMemory loads all existing passwords from the pot-file into a map
func (s *PotfileService) loadPotfileIntoMemory() (map[string]bool, error) {
	passwords := make(map[string]bool)

	file, err := os.Open(s.potfilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open pot-file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		password := scanner.Text()
		passwords[password] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read pot-file: %w", err)
	}

	return passwords, nil
}

// appendToPotfile appends new entries to the pot-file
func (s *PotfileService) appendToPotfile(entries []potfileStagingEntry) error {
	file, err := os.OpenFile(s.potfilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open pot-file for appending: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, entry := range entries {
		if _, err := writer.WriteString(entry.Password + "\n"); err != nil {
			return fmt.Errorf("failed to write password to pot-file: %w", err)
		}
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush pot-file writer: %w", err)
	}

	return nil
}

// deleteProcessedEntries deletes staging entries after they have been processed
func (s *PotfileService) deleteProcessedEntries(ctx context.Context, entries []potfileStagingEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Build list of IDs
	ids := make([]int, len(entries))
	for i, entry := range entries {
		ids[i] = entry.ID
	}

	// Delete in batches of 100 to avoid query length issues
	batchSize := 100
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		
		batch := ids[i:end]
		query := `DELETE FROM potfile_staging WHERE id = ANY($1)`
		
		if _, err := s.db.ExecContext(ctx, query, pq.Array(batch)); err != nil {
			return fmt.Errorf("failed to delete processed entries: %w", err)
		}
	}

	debug.Info("Deleted %d processed entries from potfile_staging", len(entries))
	return nil
}

// calculatePotfileMD5 calculates the MD5 hash of the potfile
func (s *PotfileService) calculatePotfileMD5() (string, error) {
	file, err := os.Open(s.potfilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open potfile for MD5 calculation: %w", err)
	}
	defer file.Close()
	
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate MD5: %w", err)
	}
	
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// getOrCreatePotfileWordlist gets or creates the pot-file wordlist entry
func (s *PotfileService) getOrCreatePotfileWordlist(ctx context.Context) (int, error) {
	debug.Info("getOrCreatePotfileWordlist called")
	// First check if a pot-file wordlist already exists
	query := `SELECT id FROM wordlists WHERE is_potfile = TRUE LIMIT 1`
	var wordlistID int
	err := s.db.QueryRowContext(ctx, query).Scan(&wordlistID)
	if err == nil {
		debug.Info("Found existing pot-file wordlist with ID: %d", wordlistID)
		
		// Update the MD5 hash and file size for the existing wordlist
		md5Hash, err := s.calculatePotfileMD5()
		if err != nil {
			debug.Warning("Failed to calculate potfile MD5 for update: %v", err)
			md5Hash = "68b329da9893e34099c7d8ad5cb9c940" // MD5 of "\n"
		}
		
		// Get file size
		fileInfo, err := os.Stat(s.potfilePath)
		fileSize := int64(0)
		if err == nil {
			fileSize = fileInfo.Size()
		}
		
		// Update the wordlist with correct MD5 and file size
		debug.Info("Updating existing pot-file wordlist MD5: %s, size: %d", md5Hash, fileSize)
		if err := s.wordlistStore.UpdateWordlistFileInfo(ctx, wordlistID, md5Hash, fileSize); err != nil {
			debug.Error("Failed to update pot-file wordlist info: %v", err)
			// Don't fail completely, just log the error
		}
		
		return wordlistID, nil
	}
	if err != sql.ErrNoRows {
		debug.Error("Error checking for existing pot-file wordlist: %v", err)
		return 0, fmt.Errorf("failed to check for existing pot-file wordlist: %w", err)
	}

	// Get system user ID
	systemUserID, err := s.getSystemUserID(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get system user ID: %w", err)
	}

	// Calculate the actual MD5 hash of the potfile
	md5Hash, err := s.calculatePotfileMD5()
	if err != nil {
		// If we can't calculate MD5 (file might not exist yet), use a fallback
		debug.Warning("Failed to calculate potfile MD5, using default: %v", err)
		md5Hash = "68b329da9893e34099c7d8ad5cb9c940" // MD5 of "\n"
	}
	
	// Get file size
	fileInfo, err := os.Stat(s.potfilePath)
	fileSize := int64(0)
	if err == nil {
		fileSize = fileInfo.Size()
	}
	
	// Create new wordlist entry
	wordlist := &models.Wordlist{
		Name:               "Pot-file",
		Description:        "System pot-file containing all cracked passwords",
		WordlistType:       "custom",
		Format:             "plaintext",
		FileName:           "custom/potfile.txt", // Relative path without "wordlists/" prefix
		MD5Hash:            md5Hash,
		FileSize:           fileSize,
		WordCount:          1,         // Start with 1 for the blank line
		CreatedBy:          systemUserID,
		VerificationStatus: "verified",
		IsPotfile:          true, // Set the flag during creation
		Tags:               []string{"system", "potfile"},
	}

	// Create wordlist with is_potfile flag already set
	debug.Info("Creating pot-file wordlist entry with is_potfile=true flag")
	if err := s.wordlistStore.CreateWordlist(ctx, wordlist); err != nil {
		return 0, fmt.Errorf("failed to create pot-file wordlist: %w", err)
	}

	debug.Info("Created pot-file wordlist entry with ID: %d and is_potfile=true", wordlist.ID)
	return wordlist.ID, nil
}

// getOrCreatePotfilePresetJob gets or creates the pot-file preset job
func (s *PotfileService) getOrCreatePotfilePresetJob(ctx context.Context, wordlistID int) (uuid.UUID, error) {
	debug.Info("getOrCreatePotfilePresetJob called with wordlistID: %d", wordlistID)
	// Check if preset job already exists
	existingJob, err := s.presetJobRepo.GetByName(ctx, "Potfile Run")
	if err == nil && existingJob != nil {
		debug.Info("Found existing pot-file preset job with ID: %s", existingJob.ID)
		return existingJob.ID, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		debug.Error("Error checking for existing pot-file preset job: %v", err)
		return uuid.Nil, fmt.Errorf("failed to check for existing pot-file preset job: %w", err)
	}

	// Get system settings for defaults
	maxPrioritySetting, err := s.systemSettingsRepo.GetSetting(ctx, "max_job_priority")
	maxPriority := 1000 // default
	if err == nil && maxPrioritySetting != nil && maxPrioritySetting.Value != nil && *maxPrioritySetting.Value != "" {
		if mp, err := strconv.Atoi(*maxPrioritySetting.Value); err == nil {
			maxPriority = mp
		}
	}

	chunkDurationSetting, err := s.systemSettingsRepo.GetSetting(ctx, "default_chunk_duration")
	chunkDuration := 1200 // default
	if err == nil && chunkDurationSetting != nil && chunkDurationSetting.Value != nil && *chunkDurationSetting.Value != "" {
		if cd, err := strconv.Atoi(*chunkDurationSetting.Value); err == nil {
			chunkDuration = cd
		}
	}

	// Get latest binary version
	latestBinary, err := s.getLatestBinaryVersion(ctx)
	if err != nil {
		// Propagate ErrNoBinaryVersions without wrapping
		if errors.Is(err, ErrNoBinaryVersions) {
			return uuid.Nil, err
		}
		return uuid.Nil, fmt.Errorf("failed to get latest binary version: %w", err)
	}

	// Create preset job
	presetJob := models.PresetJob{
		Name:                     "Potfile Run",
		WordlistIDs:              []string{strconv.Itoa(wordlistID)},
		RuleIDs:                  []string{},
		AttackMode:               models.AttackModeStraight,
		Priority:                 maxPriority,
		ChunkSizeSeconds:         chunkDuration,
		StatusUpdatesEnabled:     true,
		AllowHighPriorityOverride: true,
		BinaryVersionID:          latestBinary,
		Mask:                     "",
		Keyspace:                 nil, // Will be set after calculation
		MaxAgents:                0, // Unlimited
	}

	createdJob, err := s.presetJobRepo.Create(ctx, presetJob)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create pot-file preset job: %w", err)
	}

	// Set initial keyspace to 1 (for the blank line)
	initialKeyspace := int64(1)
	updateQuery := `UPDATE preset_jobs SET keyspace = $1 WHERE id = $2`
	if _, err := s.db.ExecContext(ctx, updateQuery, initialKeyspace, createdJob.ID); err != nil {
		debug.Error("Failed to set initial keyspace for pot-file preset job: %v", err)
	}

	debug.Info("Created pot-file preset job with ID: %s", createdJob.ID)
	return createdJob.ID, nil
}

// updateSystemSettings updates the system settings with pot-file IDs
func (s *PotfileService) updateSystemSettings(ctx context.Context, wordlistID int, presetJobID uuid.UUID) error {
	// Update wordlist ID
	if err := s.systemSettingsRepo.UpdateSetting(ctx, "potfile_wordlist_id", strconv.Itoa(wordlistID)); err != nil {
		return fmt.Errorf("failed to update potfile_wordlist_id: %w", err)
	}

	// Update preset job ID (only if not nil UUID)
	if presetJobID != uuid.Nil {
		if err := s.systemSettingsRepo.UpdateSetting(ctx, "potfile_preset_job_id", presetJobID.String()); err != nil {
			return fmt.Errorf("failed to update potfile_preset_job_id: %w", err)
		}
	}

	return nil
}

// syncPresetJobWithWordlist syncs the preset job with the current wordlist ID and keyspace
func (s *PotfileService) syncPresetJobWithWordlist(ctx context.Context, wordlistID int, presetJobID uuid.UUID) error {
	// Get current word count from wordlist
	wordlist, err := s.wordlistStore.GetWordlist(ctx, wordlistID)
	if err != nil {
		return fmt.Errorf("failed to get wordlist: %w", err)
	}
	
	// Update preset job with correct wordlist ID and keyspace
	query := `
		UPDATE preset_jobs 
		SET wordlist_ids = $1::jsonb,
		    keyspace = $2,
		    updated_at = NOW()
		WHERE id = $3
	`
	
	wordlistIDs := []string{strconv.Itoa(wordlistID)}
	wordlistIDsJSON, err := json.Marshal(wordlistIDs)
	if err != nil {
		return fmt.Errorf("failed to marshal wordlist IDs: %w", err)
	}
	
	_, err = s.db.ExecContext(ctx, query, wordlistIDsJSON, wordlist.WordCount, presetJobID)
	if err != nil {
		return fmt.Errorf("failed to update preset job: %w", err)
	}
	
	debug.Info("Synced preset job %s with wordlist %d (keyspace: %d)", 
		presetJobID, wordlistID, wordlist.WordCount)
	return nil
}

// getSystemUserID gets the system user ID
func (s *PotfileService) getSystemUserID(ctx context.Context) (uuid.UUID, error) {
	query := `SELECT id FROM users WHERE username = 'system' LIMIT 1`
	var userID uuid.UUID
	err := s.db.QueryRowContext(ctx, query).Scan(&userID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to get system user ID: %w", err)
	}
	return userID, nil
}

// getLatestBinaryVersion gets the highest active binary version ID
func (s *PotfileService) getLatestBinaryVersion(ctx context.Context) (int, error) {
	// First try to get the highest ID where is_active = true
	query := `SELECT id FROM binary_versions WHERE is_active = true ORDER BY id DESC LIMIT 1`
	var versionID int
	err := s.db.QueryRowContext(ctx, query).Scan(&versionID)
	if err != nil {
		if err == sql.ErrNoRows {
			// No active binaries found, check if any binaries exist at all
			queryAny := `SELECT id FROM binary_versions ORDER BY id DESC LIMIT 1`
			err = s.db.QueryRowContext(ctx, queryAny).Scan(&versionID)
			if err != nil {
				if err == sql.ErrNoRows {
					// No binaries exist at all
					debug.Info("No binary versions found in database")
					return 0, ErrNoBinaryVersions
				}
				return 0, fmt.Errorf("failed to get any binary version: %w", err)
			}
			// Found inactive binary, use it
			debug.Warning("No active binary versions found, using highest inactive ID: %d", versionID)
			return versionID, nil
		}
		return 0, fmt.Errorf("failed to get latest binary version: %w", err)
	}
	debug.Debug("Found active binary version with ID: %d", versionID)
	return versionID, nil
}

// triggerKeyspaceRecalculation triggers a keyspace recalculation for the pot-file preset job
func (s *PotfileService) triggerKeyspaceRecalculation(ctx context.Context) {
	// Get pot-file preset job ID
	presetJobIDSetting, err := s.systemSettingsRepo.GetSetting(ctx, "potfile_preset_job_id")
	if err != nil || presetJobIDSetting == nil || presetJobIDSetting.Value == nil || *presetJobIDSetting.Value == "" {
		debug.Error("Failed to get pot-file preset job ID: %v", err)
		return
	}

	presetJobID, err := uuid.Parse(*presetJobIDSetting.Value)
	if err != nil {
		debug.Error("Failed to parse pot-file preset job ID: %v", err)
		return
	}

	// Count lines in pot-file (this is the keyspace)
	lineCount, err := s.countPotfileLines()
	if err != nil {
		debug.Error("Failed to count pot-file lines: %v", err)
		return
	}

	// Update preset job keyspace
	query := `UPDATE preset_jobs SET keyspace = $1, updated_at = NOW() WHERE id = $2`
	if _, err := s.db.ExecContext(ctx, query, lineCount, presetJobID); err != nil {
		debug.Error("Failed to update pot-file preset job keyspace: %v", err)
		return
	}

	debug.Info("Updated pot-file preset job keyspace to %d", lineCount)
}

// countPotfileLines counts the number of lines in the pot-file
func (s *PotfileService) countPotfileLines() (int64, error) {
	file, err := os.Open(s.potfilePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open pot-file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var count int64
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("failed to read pot-file: %w", err)
	}

	return count, nil
}

// monitorForBinaryAndCreatePresetJob monitors for binary versions and creates the preset job when one is available
func (s *PotfileService) monitorForBinaryAndCreatePresetJob(ctx context.Context, wordlistID int) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		
		debug.Info("Starting monitor for binary versions to create pot-file preset job")
		firstCheck := true
		
		for {
			select {
			case <-ticker.C:
				// Check if preset job already exists (could have been created elsewhere)
				existingJob, err := s.presetJobRepo.GetByName(ctx, "Potfile Run")
				if err == nil && existingJob != nil {
					debug.Info("Pot-file preset job found (ID: %s), stopping monitor", existingJob.ID)
					return
				}
				
				// Try to create the preset job
				presetJobID, err := s.getOrCreatePotfilePresetJob(ctx, wordlistID)
				if err != nil {
					if errors.Is(err, ErrNoBinaryVersions) {
						if firstCheck {
							debug.Info("Waiting for binary versions to be added before creating pot-file preset job")
							firstCheck = false
						}
						// Continue monitoring
						continue
					}
					// Other errors are logged but we continue monitoring
					debug.Error("Failed to create pot-file preset job: %v", err)
					continue
				}
				
				// Success! Update system settings and stop monitoring
				debug.Info("Successfully created pot-file preset job with ID: %s", presetJobID)
				if err := s.updateSystemSettings(ctx, wordlistID, presetJobID); err != nil {
					debug.Error("Failed to update system settings after creating preset job: %v", err)
				}
				return
				
			case <-s.stopChan:
				debug.Info("Pot-file preset job monitor stopped due to service shutdown")
				return
			}
		}
	}()
}

// UpdatePotfileMetadata updates the MD5 hash and file size of the potfile in the database
func (s *PotfileService) UpdatePotfileMetadata(ctx context.Context) error {
	// Calculate the current MD5 hash of the potfile
	md5Hash, err := s.calculatePotfileMD5()
	if err != nil {
		return fmt.Errorf("failed to calculate potfile MD5: %w", err)
	}
	
	// Get the current file size
	fileInfo, err := os.Stat(s.potfilePath)
	if err != nil {
		return fmt.Errorf("failed to get potfile info: %w", err)
	}
	fileSize := fileInfo.Size()
	
	// Count the actual lines in the potfile
	lineCount, err := s.countPotfileLines()
	if err != nil {
		return fmt.Errorf("failed to count potfile lines: %w", err)
	}
	
	// Get the potfile wordlist ID from system settings
	wordlistIDSetting, err := s.systemSettingsRepo.GetSetting(ctx, "potfile_wordlist_id")
	if err != nil || wordlistIDSetting == nil || wordlistIDSetting.Value == nil || *wordlistIDSetting.Value == "" {
		return fmt.Errorf("failed to get potfile wordlist ID: %w", err)
	}

	wordlistID, err := strconv.Atoi(*wordlistIDSetting.Value)
	if err != nil {
		return fmt.Errorf("failed to parse potfile wordlist ID: %w", err)
	}

	// Get the old word count before updating
	oldWordlist, _ := s.wordlistStore.GetWordlist(ctx, wordlistID)
	oldLineCount := int64(0)
	if oldWordlist != nil {
		oldLineCount = oldWordlist.WordCount
	}

	// Update the wordlist entry in the database with the new MD5, file size, and word count
	if err := s.wordlistStore.UpdateWordlistComplete(ctx, wordlistID, md5Hash, fileSize, lineCount); err != nil {
		return fmt.Errorf("failed to update potfile wordlist info: %w", err)
	}
	
	debug.Info("Updated potfile metadata - MD5: %s, Size: %d bytes, Words: %d", md5Hash, fileSize, lineCount)
	
	// Sync preset job if it exists
	presetJobSetting, err := s.systemSettingsRepo.GetSetting(ctx, "potfile_preset_job_id")
	if err == nil && presetJobSetting != nil && presetJobSetting.Value != nil && *presetJobSetting.Value != "" {
		presetJobID, err := uuid.Parse(*presetJobSetting.Value)
		if err == nil && presetJobID != uuid.Nil {
			if err := s.syncPresetJobWithWordlist(ctx, wordlistID, presetJobID); err != nil {
				debug.Warning("Failed to sync preset job after metadata update: %v", err)
				// Don't fail the operation
			}
		}
	}

	// Trigger job updates if word count changed and we have the job update service
	if s.jobUpdateService != nil && oldLineCount != lineCount {
		debug.Info("Triggering job updates for potfile wordlist %d (old: %d, new: %d)",
			wordlistID, oldLineCount, lineCount)
		if err := s.jobUpdateService.HandleWordlistUpdate(ctx, wordlistID, oldLineCount, lineCount); err != nil {
			debug.Error("Failed to update jobs for potfile changes: %v", err)
			// Don't fail the operation
		}
	}

	return nil
}
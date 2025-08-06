package services

import (
	"bufio"
	"context"
	"database/sql"
	"errors"
	"fmt"
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
	}

	// If this is first initialization, populate with existing cracked passwords
	if !fileExists {
		if err := s.populateFromExistingCracks(ctx); err != nil {
			debug.Warning("Failed to populate pot-file from existing cracks: %v", err)
			// Don't fail initialization if population fails
		}
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
		if err := s.appendToPotfile(newEntries); err != nil {
			debug.Error("Failed to append to pot-file: %v", err)
			return
		}
		debug.Info("Added %d new unique entries to pot-file", len(newEntries))
	}

	// Mark entries as processed
	if err := s.markEntriesProcessed(ctx, entries); err != nil {
		debug.Error("Failed to mark entries as processed: %v", err)
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
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		password := scanner.Text()
		// Skip the first blank line
		if lineNum == 1 && password == "" {
			continue
		}
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

// markEntriesProcessed marks staging entries as processed
func (s *PotfileService) markEntriesProcessed(ctx context.Context, entries []potfileStagingEntry) error {
	if len(entries) == 0 {
		return nil
	}

	// Build list of IDs
	ids := make([]int, len(entries))
	for i, entry := range entries {
		ids[i] = entry.ID
	}

	// Update in batches of 100 to avoid query length issues
	batchSize := 100
	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		
		batch := ids[i:end]
		query := `
			UPDATE potfile_staging
			SET processed = TRUE, processed_at = NOW()
			WHERE id = ANY($1)
		`
		
		if _, err := s.db.ExecContext(ctx, query, pq.Array(batch)); err != nil {
			return fmt.Errorf("failed to mark entries as processed: %w", err)
		}
	}

	return nil
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

	// Create new wordlist entry
	wordlist := &models.Wordlist{
		Name:               "Pot-file",
		Description:        "System pot-file containing all cracked passwords",
		WordlistType:       "custom",
		Format:             "plaintext",
		FileName:           "custom/potfile.txt", // Relative path without "wordlists/" prefix
		MD5Hash:            "pending", // Will be updated later
		FileSize:           0,         // Will be updated later
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
		IsSmallJob:               true,
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

// populateFromExistingCracks populates the pot-file with existing cracked passwords
func (s *PotfileService) populateFromExistingCracks(ctx context.Context) error {
	debug.Info("Populating pot-file from existing cracked passwords")

	// Get all cracked passwords
	params := repository.CrackedHashParams{
		Limit:  1000,
		Offset: 0,
	}

	file, err := os.OpenFile(s.potfilePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open pot-file for appending: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	totalAdded := 0
	existingPasswords := make(map[string]bool)

	for {
		hashes, _, err := s.hashRepo.GetCrackedHashes(ctx, params)
		if err != nil {
			return fmt.Errorf("failed to get cracked hashes: %w", err)
		}

		if len(hashes) == 0 {
			break
		}

		for _, hash := range hashes {
			// Skip if we've already added this password
			if existingPasswords[hash.Password] {
				continue
			}

			// Write password to pot-file
			if _, err := writer.WriteString(hash.Password + "\n"); err != nil {
				return fmt.Errorf("failed to write password to pot-file: %w", err)
			}

			existingPasswords[hash.Password] = true
			totalAdded++
		}

		params.Offset += params.Limit
	}

	if err := writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush pot-file writer: %w", err)
	}

	debug.Info("Added %d existing cracked passwords to pot-file", totalAdded)

	// Update keyspace
	if totalAdded > 0 {
		s.triggerKeyspaceRecalculation(ctx)
	}

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
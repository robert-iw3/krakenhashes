package processor

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/config"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/hashutils"
	"github.com/google/uuid"
)

// Add a new constant for the status
const HashListStatusReadyWithErrors = "ready_with_errors"

// HashlistDBProcessor handles the asynchronous processing of uploaded hashlists, focusing on DB interactions.
type HashlistDBProcessor struct {
	hashlistRepo *repository.HashListRepository
	hashTypeRepo *repository.HashTypeRepository
	hashRepo     *repository.HashRepository
	config       *config.Config
	// valueProcessors map[int]HashValueProcessor // REMOVED: Replaced by hashutils
}

// NewHashlistDBProcessor creates a new instance of HashlistDBProcessor.
func NewHashlistDBProcessor(
	hashlistRepo *repository.HashListRepository,
	hashTypeRepo *repository.HashTypeRepository,
	hashRepo *repository.HashRepository,
	config *config.Config,
) *HashlistDBProcessor {
	// REMOVED: Initialization of valueProcessors map
	/*
		valueProcessors := make(map[int]HashValueProcessor)
		valueProcessors[1000] = &NTLMProcessor{} // Register NTLM processor
		// Register other processors here...
	*/

	return &HashlistDBProcessor{
		hashlistRepo: hashlistRepo,
		hashTypeRepo: hashTypeRepo,
		hashRepo:     hashRepo,
		config:       config,
		// valueProcessors: valueProcessors, // REMOVED
	}
}

// SubmitHashlistForProcessing initiates the background processing for a given hashlist ID.
func (p *HashlistDBProcessor) SubmitHashlistForProcessing(hashlistID int64) {
	// Launch the actual processing in a goroutine
	go p.processHashlist(hashlistID)
}

// processHashlist contains the main logic for reading, processing, and storing hashes from a list.
func (p *HashlistDBProcessor) processHashlist(hashlistID int64) {
	ctx := context.Background() // Use background context for async task
	debug.Info("Starting background processing for hashlist %d", hashlistID)

	// Get hashlist details
	hashlist, err := p.hashlistRepo.GetByID(ctx, hashlistID)
	if err != nil || hashlist == nil {
		debug.Error("Background task: Failed to get hashlist %d: %v", hashlistID, err)
		return
	}

	if hashlist.Status != models.HashListStatusProcessing {
		debug.Warning("Background task: Hashlist %d status is %s, expected 'processing'. Aborting.", hashlistID, hashlist.Status)
		return
	}
	if hashlist.FilePath == "" {
		p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, "File path is missing")
		return
	}

	// Get hash type info
	hashType, err := p.hashTypeRepo.GetByID(ctx, hashlist.HashTypeID)
	if err != nil || hashType == nil {
		debug.Error("Background task: Failed to get hash type %d for hashlist %d: %v", hashlist.HashTypeID, hashlistID, err)
		p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, "Invalid hash type")
		return
	}

	// Open the file
	file, err := os.Open(hashlist.FilePath)
	if err != nil {
		debug.Error("Background task: Failed to open file %s for hashlist %d: %v", hashlist.FilePath, hashlistID, err)
		p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, "Failed to open hashlist file")
		return
	}
	defer file.Close()

	// --- Process the file line by line ---
	scanner := bufio.NewScanner(file)
	var totalHashes, crackedHashes int64
	batchSize := p.config.HashlistBatchSize
	hashesToProcess := make([]*models.Hash, 0, batchSize)
	associationsToCreate := make([]*models.HashListHash, 0, batchSize)
	lineNumber := 0
	firstLineErrorMsg := ""     // Store the first line processing error
	lineErrorsOccurred := false // Track if any line errors happened

	// valueProcessor, processorFound := p.valueProcessors[hashType.ID] // Removed unused variables

	// Get the needs_processing flag from the fetched hashType
	needsProcessing := hashType.NeedsProcessing

	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		totalHashes++

		// --- New Processing Logic ---
		originalHash := line // Store the raw line
		username := hashutils.ExtractUsername(originalHash, hashType.ID)
		hashValue := hashutils.ProcessHashIfNeeded(originalHash, hashType.ID, needsProcessing)
		debug.Debug("[Processor:%d] Line %d: Original='%s', ProcessedValue='%s', User='%s'", hashlistID, lineNumber, originalHash, hashValue, username)
		// --- End New Processing Logic ---

		// Determine if cracked (e.g., from input format like hash:pass)
		// Note: ProcessHashIfNeeded doesn't handle cracking detection currently.
		// We might need a separate mechanism or refine processing rules.
		// For now, let's assume a simple heuristic for :password suffix if no specific processor modified it.
		password := ""
		isCracked := false
		if hashValue == originalHash { // Only apply suffix check if ProcessHashIfNeeded didn't modify it
			parts := strings.SplitN(originalHash, ":", 2)
			if len(parts) > 1 {
				// Basic check: is the first part potentially the hashValue we expect?
				// This is weak. A better approach might involve hash length/format checks.
				if parts[0] == hashValue { // Check if splitting by ':' gives back the expected hash
					password = parts[1]
					isCracked = true
				}
				// Else: it might be user:hash or some other format, don't assume crack.
			}
		}

		if isCracked {
			crackedHashes++
		}

		// Create hash model
		hash := &models.Hash{
			ID:           uuid.New(),   // Generate new UUID for potential insert
			HashValue:    hashValue,    // The value to crack (potentially processed)
			OriginalHash: originalHash, // Always store the original line
			Username:     username,     // Store the extracted username (or nil)
			HashTypeID:   hashlist.HashTypeID,
			IsCracked:    isCracked,  // Mark cracked based on heuristic above
			Password:     password,   // Store potential password from heuristic
			LastUpdated:  time.Now(), // Set initial time
		}
		debug.Debug("[Processor:%d] Line %d: Created Hash struct with ID: %s", hashlistID, lineNumber, hash.ID)
		hashesToProcess = append(hashesToProcess, hash)

		// Process in batches
		if len(hashesToProcess) >= batchSize {
			debug.Debug("[Processor:%d] Processing batch of %d hashes (Lines up to %d)", hashlistID, len(hashesToProcess), lineNumber)
			newAssociations, err := p.batchProcessHashes(ctx, hashesToProcess, hashlist.ID)
			if err != nil {
				debug.Error("Background task: Error processing hash batch for hashlist %d: %v", hashlistID, err)
				p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, "Error processing hash batch")
				return // Stop processing on batch error
			}
			associationsToCreate = append(associationsToCreate, newAssociations...)
			hashesToProcess = hashesToProcess[:0] // Clear batch
		}
	}

	// Process any remaining hashes
	if len(hashesToProcess) > 0 {
		debug.Debug("[Processor:%d] Processing final batch of %d hashes (Lines up to %d)", hashlistID, len(hashesToProcess), lineNumber)
		newAssociations, err := p.batchProcessHashes(ctx, hashesToProcess, hashlist.ID)
		if err != nil {
			debug.Error("Background task: Error processing final hash batch for hashlist %d: %v", hashlistID, err)
			p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, "Error processing final hash batch")
			return
		}
		associationsToCreate = append(associationsToCreate, newAssociations...)
	}

	// Check for scanner errors after loop
	if err := scanner.Err(); err != nil {
		debug.Error("Background task: Error reading file %s for hashlist %d: %v", hashlist.FilePath, hashlistID, err)
		p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, "Error reading hashlist file")
		return
	}

	// Create final associations batch (if any)
	if len(associationsToCreate) > 0 {
		err = p.hashRepo.AddBatchToHashList(ctx, associationsToCreate)
		if err != nil {
			debug.Error("Background task: Error creating final hashlist associations for %d: %v", hashlistID, err)
			p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, "Error saving final hash associations")
			return
		}
	}

	debug.Info("Successfully created final hashlist associations for %d", hashlistID)

	// --- Generate <id>.hash file with uncracked hashes ---
	var finalFilePath string
	uncrackedHashes, err := p.hashRepo.GetUncrackedHashValuesByHashlistID(ctx, hashlistID)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to retrieve uncracked hashes for final file generation: %v", err)
		debug.Error("Background task: %s (Hashlist: %d)", errMsg, hashlistID)
		p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, errMsg)
		return
	}

	if len(uncrackedHashes) > 0 {
		// Define the output path: <DataDir>/hashlists/<id>.hash
		outputFilename := fmt.Sprintf("%d.hash", hashlistID)
		// Construct path relative to the main DataDir
		finalFilePath = filepath.Join(p.config.DataDir, "hashlists", outputFilename)
		debug.Info("Generating final hash file for agents: %s", finalFilePath)

		outFile, err := os.Create(finalFilePath)
		if err != nil {
			errMsg := fmt.Sprintf("Failed to create final hash file %s: %v", finalFilePath, err)
			debug.Error("Background task: %s (Hashlist: %d)", errMsg, hashlistID)
			p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, errMsg)
			return // Critical failure if we can't write the output file
		}

		writer := bufio.NewWriter(outFile)
		for _, h := range uncrackedHashes {
			_, err = writer.WriteString(h + "\n")
			if err != nil {
				_ = outFile.Close() // Close file before returning on write error
				errMsg := fmt.Sprintf("Failed to write to final hash file %s: %v", finalFilePath, err)
				debug.Error("Background task: %s (Hashlist: %d)", errMsg, hashlistID)
				p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, errMsg)
				return
			}
		}

		if err = writer.Flush(); err != nil {
			_ = outFile.Close()
			errMsg := fmt.Sprintf("Failed to flush final hash file %s: %v", finalFilePath, err)
			debug.Error("Background task: %s (Hashlist: %d)", errMsg, hashlistID)
			p.updateHashlistStatus(ctx, hashlistID, models.HashListStatusError, errMsg)
			return
		}
		if err = outFile.Close(); err != nil {
			// Log error, but proceed as file is likely written
			debug.Warning("Failed to close final hash file %s cleanly: %v", finalFilePath, err)
		}
		debug.Info("Successfully wrote %d uncracked hashes to %s", len(uncrackedHashes), finalFilePath)

	} else {
		// No uncracked hashes, maybe skip file creation or create an empty file?
		// Let's log this and set finalFilePath to empty, indicating no file for agents.
		finalFilePath = ""
		debug.Info("No uncracked hashes found for hashlist %d. No agent file generated.", hashlistID)
	}

	// --- Optionally delete original uploaded file ---
	originalUploadPath := hashlist.FilePath                              // Path stored when processing started
	if originalUploadPath != "" && originalUploadPath != finalFilePath { // Avoid deleting the file we just created!
		if err := os.Remove(originalUploadPath); err != nil {
			debug.Warning("Failed to delete original uploaded file %s for hashlist %d: %v", originalUploadPath, hashlistID, err)
		} else {
			debug.Info("Deleted original uploaded file %s for hashlist %d", originalUploadPath, hashlistID)
			// Optionally try removing empty parent directories
			dir := filepath.Dir(originalUploadPath)
			_ = os.Remove(dir)               // Remove hashlistID/filename dir
			_ = os.Remove(filepath.Dir(dir)) // Remove userID dir
		}
	}

	// Determine final status
	finalStatus := models.HashListStatusReady
	if lineErrorsOccurred {
		finalStatus = HashListStatusReadyWithErrors // Use the new status constant
	}

	// Update final hashlist status, counts, AND the file path
	hashlist.TotalHashes = int(totalHashes)
	hashlist.CrackedHashes = int(crackedHashes) // Note: This counts cracks found *during* ingest heuristic, not pre-cracked ones
	hashlist.Status = finalStatus
	hashlist.ErrorMessage = sql.NullString{String: firstLineErrorMsg, Valid: firstLineErrorMsg != ""}
	hashlist.FilePath = finalFilePath // *** Update FilePath to the new <id>.hash path ***
	hashlist.UpdatedAt = time.Now()

	err = p.hashlistRepo.UpdateStatsAndStatusWithPath(ctx, hashlist.ID, int(totalHashes), int(crackedHashes), hashlist.Status, hashlist.ErrorMessage.String, hashlist.FilePath)
	if err != nil {
		debug.Error("Background task: Failed to update final stats/status/path for hashlist %d: %v", hashlistID, err)
		// Status is likely 'processing' still, but processing technically finished.
		// Might need manual intervention or retry logic.
		return
	}

	debug.Info("Successfully processed hashlist %d. Total: %d, Final Agent File: %s", hashlistID, totalHashes, finalFilePath)
}

// batchProcessHashes handles creating/updating hashes and preparing associations.
// It ensures each input hash gets associated, handling duplicates by reusing existing hash IDs
// but still creating a new association entry for the current hashlist.
func (p *HashlistDBProcessor) batchProcessHashes(ctx context.Context, hashes []*models.Hash, hashlistID int64) ([]*models.HashListHash, error) {
	debug.Debug("[Processor:%d] batchProcessHashes received %d hashes", hashlistID, len(hashes))
	if len(hashes) == 0 {
		return nil, nil
	}

	// Group input hashes by their hash_value (after processing)
	hashesByValue := make(map[string][]*models.Hash)
	uniqueHashValues := make([]string, 0)
	for _, h := range hashes {
		if _, exists := hashesByValue[h.HashValue]; !exists {
			hashesByValue[h.HashValue] = []*models.Hash{}
			uniqueHashValues = append(uniqueHashValues, h.HashValue)
		}
		hashesByValue[h.HashValue] = append(hashesByValue[h.HashValue], h)
	}
	debug.Debug("[Processor:%d] Grouped input into %d unique hash values.", hashlistID, len(uniqueHashValues))

	// Find existing hashes in the DB for these unique values
	existingHashesFromDB, err := p.hashRepo.GetByHashValues(ctx, uniqueHashValues)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing hashes: %w", err)
	}
	existingHashMap := make(map[string]*models.Hash, len(existingHashesFromDB))
	for _, eh := range existingHashesFromDB {
		// If multiple rows exist for the same hash_value (due to no unique constraint),
		// we just take the first one encountered. The goal is to link to *an* existing record.
		if _, exists := existingHashMap[eh.HashValue]; !exists {
			existingHashMap[eh.HashValue] = eh
		}
	}
	debug.Debug("[Processor:%d] Found %d existing hash records for %d unique values.", hashlistID, len(existingHashMap), len(uniqueHashValues))

	// Prepare lists for creation, update, and final associations
	newHashesToCreate := make([]*models.Hash, 0)
	hashesToUpdate := make([]*models.Hash, 0)
	finalAssociations := make([]*models.HashListHash, 0, len(hashes)) // Size based on original input count
	newlyCrackedInBatch := 0
	idsToUpdate := make(map[uuid.UUID]struct{}) // Track which existing hash IDs need updates

	// Iterate through the unique hash values found in the input batch
	for value, inputHashesForValue := range hashesByValue {
		existingDBHash, valueExistsInDB := existingHashMap[value]

		if valueExistsInDB {
			// Value exists. All input hashes with this value should associate with existingDBHash.ID
			hashIDToAssociate := existingDBHash.ID
			debug.Debug("[Processor:%d] Value '%s' exists in DB (ID: %s). Creating %d associations.", hashlistID, value, hashIDToAssociate, len(inputHashesForValue))

			// Create an association for *each* input hash that had this value
			for _, inputHash := range inputHashesForValue {
				finalAssociations = append(finalAssociations, &models.HashListHash{
					HashlistID: hashlistID,
					HashID:     hashIDToAssociate,
				})

				// --- Pre-cracking & Update Check ---
				// Check if *any* of the input hashes for this value suggest a crack or username update
				// Only need to check/update the single existingDBHash record once per value.
				if _, alreadyChecked := idsToUpdate[existingDBHash.ID]; !alreadyChecked {
					needsUpdate := false
					if !existingDBHash.IsCracked && inputHash.IsCracked {
						existingDBHash.IsCracked = true
						existingDBHash.Password = inputHash.Password
						needsUpdate = true
						newlyCrackedInBatch++ // Count crack discovery
					}
					if existingDBHash.Username == nil && inputHash.Username != nil {
						existingDBHash.Username = inputHash.Username
						needsUpdate = true
					}

					if needsUpdate {
						existingDBHash.LastUpdated = time.Now()
						hashesToUpdate = append(hashesToUpdate, existingDBHash)
						idsToUpdate[existingDBHash.ID] = struct{}{} // Mark as needing update
					}
				}
				// --- End Update Check ---
			}
		} else {
			// Value does NOT exist. Create *one* new hash record for this value.
			// We'll use the details from the first input hash encountered for this value.
			hashToCreate := inputHashesForValue[0] // Use the first one as representative
			hashToCreate.ID = uuid.New()           // Ensure it has a new UUID for creation
			newHashesToCreate = append(newHashesToCreate, hashToCreate)
			hashIDToAssociate := hashToCreate.ID // The ID we *will* create

			debug.Debug("[Processor:%d] Value '%s' is new. Will create (ID: %s) and %d associations.", hashlistID, value, hashIDToAssociate, len(inputHashesForValue))

			// Create an association for *each* input hash that had this value, using the *new* ID
			for range inputHashesForValue { // Iterate based on count
				finalAssociations = append(finalAssociations, &models.HashListHash{
					HashlistID: hashlistID,
					HashID:     hashIDToAssociate,
				})
			}

			// If the representative new hash is cracked, count it
			if hashToCreate.IsCracked {
				newlyCrackedInBatch++
			}
		}
	}
	debug.Debug("[Processor:%d] Determined %d hashes to create, %d hashes to update, %d associations to make.", hashlistID, len(newHashesToCreate), len(hashesToUpdate), len(finalAssociations))

	// Create new hashes
	if len(newHashesToCreate) > 0 {
		// CreateBatch needs to handle potential duplicates if run concurrently,
		// but here we assume it attempts to insert all. It should return only successfully created ones.
		// We already assigned UUIDs, so we expect CreateBatch to use those.
		_, err := p.hashRepo.CreateBatch(ctx, newHashesToCreate) // Don't need returned hashes if IDs are pre-set
		if err != nil {
			// If CreateBatch fails, we cannot reliably create associations for the new hashes.
			return nil, fmt.Errorf("failed to create new hash batch: %w", err)
		}
	}

	// Update existing hashes
	if len(hashesToUpdate) > 0 {
		err = p.hashRepo.UpdateBatch(ctx, hashesToUpdate)
		if err != nil {
			// Log the error but potentially continue to create associations?
			// For now, return error to prevent potentially inconsistent state.
			debug.Error("[Processor:%d] Failed to update hash batch: %v. Associations might be incomplete.", hashlistID, err)
			return nil, fmt.Errorf("failed to update hash batch: %w", err)
		}
	}

	// Increment cracked count in hashlists table
	if newlyCrackedInBatch > 0 {
		err = p.hashlistRepo.IncrementCrackedCount(ctx, hashlistID, newlyCrackedInBatch)
		if err != nil {
			// Log error but don't fail the whole process for this.
			debug.Error("[Processor:%d] Failed to increment cracked count by %d: %v", hashlistID, newlyCrackedInBatch, err)
		}
	}

	// Return the prepared list of all associations to be created in the next step
	return finalAssociations, nil
}

// Helper to update hashlist status (avoids direct repo access from other funcs if needed)
func (p *HashlistDBProcessor) updateHashlistStatus(ctx context.Context, id int64, status string, errMsg string) {
	err := p.hashlistRepo.UpdateStatus(ctx, id, status, errMsg)
	if err != nil {
		debug.Error("Failed to update hashlist %d status to %s: %v", id, status, err)
	}
}

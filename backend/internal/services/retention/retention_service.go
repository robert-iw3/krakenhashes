package retention

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)

// RetentionService handles the automatic purging of old hashlists based on retention policies.
type RetentionService struct {
	db                 *db.DB // Needed for transactions
	hashlistRepo       *repository.HashListRepository
	hashRepo           *repository.HashRepository
	clientRepo         *repository.ClientRepository
	clientSettingsRepo *repository.ClientSettingsRepository
}

// NewRetentionService creates a new RetentionService.
func NewRetentionService(database *db.DB, hr *repository.HashListRepository, hshr *repository.HashRepository, cr *repository.ClientRepository, sr *repository.ClientSettingsRepository) *RetentionService {
	return &RetentionService{
		db:                 database,
		hashlistRepo:       hr,
		hashRepo:           hshr,
		clientRepo:         cr,
		clientSettingsRepo: sr,
	}
}

// PurgeOldHashlists finds and deletes hashlists that have exceeded their retention period.
func (s *RetentionService) PurgeOldHashlists(ctx context.Context) error {
	debug.Info("Starting data retention purge process...")

	// 1. Get default retention policy
	defaultRetentionSetting, err := s.clientSettingsRepo.GetSetting(ctx, "default_data_retention_months")
	if err != nil || defaultRetentionSetting.Value == nil {
		debug.Error("Failed to get default client retention setting during purge: %v", err)
		return fmt.Errorf("purge failed: could not retrieve default client retention setting")
	}
	defaultRetentionMonths, err := strconv.Atoi(*defaultRetentionSetting.Value)
	if err != nil {
		debug.Error("Invalid default client retention setting value '%s': %v", *defaultRetentionSetting.Value, err)
		return fmt.Errorf("purge failed: invalid default client retention setting value")
	}
	debug.Debug("Purge: Default retention is %d months.", defaultRetentionMonths)

	// 2. Get all clients to check their specific policies
	clients, err := s.clientRepo.List(ctx)
	if err != nil {
		debug.Error("Failed to list clients during purge: %v", err)
		return fmt.Errorf("purge failed: could not list clients")
	}
	clientRetentionMap := make(map[string]int)
	for _, client := range clients {
		if client.DataRetentionMonths != nil {
			clientRetentionMap[client.ID.String()] = *client.DataRetentionMonths
		} // Clients with NULL will use the default later
	}

	// 3. Find and process hashlists eligible for purging
	// Need a method in HashListRepository to get *all* hashlists, perhaps with pagination if very large
	// For now, assume List can be used without filters, or create a new GetAll method.
	// Let's use List with a large limit for simplicity, add pagination later if needed.
	limit := 1000 // Process in batches
	offset := 0
	processedCount := 0
	deletedCount := 0

	for {
		hashlists, total, err := s.hashlistRepo.List(ctx, repository.ListHashlistsParams{Limit: limit, Offset: offset})
		if err != nil {
			debug.Error("Failed to list hashlists batch (offset %d) during purge: %v", offset, err)
			return fmt.Errorf("purge failed: could not list hashlists batch")
		}
		if len(hashlists) == 0 {
			debug.Debug("Purge: No more hashlists found.")
			break // Exit loop when no more hashlists are found
		}
		offset += len(hashlists)

		for _, hl := range hashlists {
			processedCount++
			retentionMonths := defaultRetentionMonths
			clientIsSet := hl.ClientID != uuid.Nil
			if clientIsSet {
				if specificRetention, ok := clientRetentionMap[hl.ClientID.String()]; ok {
					retentionMonths = specificRetention
				}
			}

			// Skip if retention is set to 0 (keep forever)
			if retentionMonths == 0 {
				debug.Debug("Purge: Skipping hashlist %d (Client: %s) - Retention is 0 (Keep Forever)", hl.ID, hl.ClientID)
				continue
			}

			// Calculate expiration date
			retentionDuration := time.Duration(retentionMonths) * 30 * 24 * time.Hour // Approx. months
			expirationDate := hl.CreatedAt.Add(retentionDuration)

			// Check if expired
			if time.Now().After(expirationDate) {
				debug.Info("Purge: Hashlist %d (Created: %s, Client: %s, Retention: %d months) has expired (Expiry: %s). Deleting...", hl.ID, hl.CreatedAt, hl.ClientID, retentionMonths, expirationDate)
				err := s.DeleteHashlistAndOrphanedHashes(ctx, hl.ID)
				if err != nil {
					debug.Error("Purge: Failed to delete expired hashlist %d: %v", hl.ID, err)
					// Continue processing other hashlists even if one fails?
					// For now, let's log the error and continue.
					continue
				}
				deletedCount++
			} else {
				debug.Debug("Purge: Hashlist %d (Client: %s) has not expired yet (Expiry: %s)", hl.ID, hl.ClientID, expirationDate)
			}
		}

		// Safety break if List doesn't behave as expected or total is weird
		if offset >= total && total > 0 {
			break
		}
	}

	// 4. Update last purge run timestamp
	nowStr := time.Now().Format(time.RFC3339Nano)
	err = s.clientSettingsRepo.SetSetting(ctx, "last_purge_run", &nowStr)
	if err != nil {
		debug.Error("Purge: Failed to update last_purge_run timestamp: %v", err)
		// Log error but don't fail the whole operation
	}

	debug.Info("Data retention purge completed. Processed: %d, Deleted: %d", processedCount, deletedCount)
	return nil
}

// DeleteHashlistAndOrphanedHashes deletes a hashlist and any hashes that become orphaned as a result.
// This should be run within a transaction.
func (s *RetentionService) DeleteHashlistAndOrphanedHashes(ctx context.Context, hashlistID int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for deleting hashlist %d: %w", hashlistID, err)
	}
	defer tx.Rollback() // Rollback if commit fails or not reached

	// 1. Get all hash IDs associated with this hashlist BEFORE deleting the list/links
	hashIDs, err := s.hashRepo.GetHashIDsByHashlistIDTx(tx, hashlistID) // Need this method in HashRepository
	if err != nil {
		return fmt.Errorf("failed to get hash IDs for hashlist %d: %w", hashlistID, err)
	}
	debug.Debug("Purge: Found %d hash IDs potentially affected by deleting hashlist %d", len(hashIDs), hashlistID)

	// 2. Delete entries from the junction table (hashlist_hashes)
	// This can be done via HashlistRepository or HashRepository
	err = s.hashRepo.DeleteHashlistAssociationsTx(tx, hashlistID) // Need this method in HashRepository
	if err != nil {
		return fmt.Errorf("failed to delete hashlist_hashes entries for hashlist %d: %w", hashlistID, err)
	}
	debug.Debug("Purge: Deleted hashlist_hashes entries for hashlist %d", hashlistID)

	// 3. Delete the hashlist itself
	err = s.hashlistRepo.DeleteTx(tx, hashlistID) // Need DeleteTx method in HashListRepository
	if err != nil {
		return fmt.Errorf("failed to delete hashlist %d: %w", hashlistID, err)
	}
	debug.Debug("Purge: Deleted hashlist record %d", hashlistID)

	// 4. Check each potentially orphaned hash and delete if necessary
	deletedHashesCount := 0
	for _, hashID := range hashIDs {
		isOrphaned, err := s.hashRepo.IsHashOrphanedTx(tx, hashID) // Need this method in HashRepository
		if err != nil {
			return fmt.Errorf("failed to check if hash %s is orphaned: %w", hashID, err)
		}
		if isOrphaned {
			err = s.hashRepo.DeleteHashByIDTx(tx, hashID) // Need this method in HashRepository
			if err != nil {
				return fmt.Errorf("failed to delete orphaned hash %s: %w", hashID, err)
			}
			deletedHashesCount++
			debug.Debug("Purge: Deleted orphaned hash %s", hashID)
		}
	}
	debug.Debug("Purge: Deleted %d orphaned hashes for hashlist %d", deletedHashesCount, hashlistID)

	// 5. Commit the transaction
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction for deleting hashlist %d: %w", hashlistID, err)
	}

	return nil
}

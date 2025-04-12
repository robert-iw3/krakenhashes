package client

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/services/retention"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
	"github.com/google/uuid"
)

// ClientService provides business logic for client operations, including complex deletion.
type ClientService struct {
	clientRepo         *repository.ClientRepository
	hashlistRepo       *repository.HashListRepository
	clientSettingsRepo *repository.ClientSettingsRepository
	retentionService   *retention.RetentionService
}

// NewClientService creates a new ClientService.
func NewClientService(cr *repository.ClientRepository, hr *repository.HashListRepository, sr *repository.ClientSettingsRepository, retsvc *retention.RetentionService) *ClientService {
	return &ClientService{
		clientRepo:         cr,
		hashlistRepo:       hr,
		clientSettingsRepo: sr,
		retentionService:   retsvc,
	}
}

// DeleteClient handles the deletion of a client and its associated hashlists based on retention policies.
func (s *ClientService) DeleteClient(ctx context.Context, clientID uuid.UUID) error {
	debug.Debug("Starting client deletion process for ID: %s", clientID)

	// 1. Get the client to check its retention policy
	client, err := s.clientRepo.GetByID(ctx, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return err
		}
		debug.Error("Failed to get client %s for deletion: %v", clientID, err)
		return fmt.Errorf("failed to retrieve client %s: %w", clientID, err)
	}

	// 2. Get the default system retention policy
	defaultRetentionSetting, err := s.clientSettingsRepo.GetSetting(ctx, "default_data_retention_months")
	if err != nil || defaultRetentionSetting.Value == nil {
		debug.Error("Failed to get default client retention setting: %v", err)
		// Fallback to 0 if setting is missing or value is NULL (should not happen based on migration)
		defaultRetentionSetting = &models.ClientSetting{Value: new(string), Key: "default_data_retention_months"}
		*defaultRetentionSetting.Value = "0"
	}
	defaultRetentionMonths, _ := strconv.Atoi(*defaultRetentionSetting.Value) // Error unlikely as we default to "0"
	debug.Debug("Default retention months: %d", defaultRetentionMonths)

	// 3. Determine the client's effective retention policy
	clientRetentionMonths := defaultRetentionMonths
	if client.DataRetentionMonths != nil {
		clientRetentionMonths = *client.DataRetentionMonths
		debug.Debug("Client %s has specific retention: %d months", clientID, clientRetentionMonths)
	} else {
		debug.Debug("Client %s using default retention: %d months", clientID, clientRetentionMonths)
	}

	// 4. Decide whether to delete hashlists immediately or let the FK constraint handle it
	deleteHashlistsImmediately := clientRetentionMonths != 0 && (clientRetentionMonths < defaultRetentionMonths || defaultRetentionMonths == 0)

	if deleteHashlistsImmediately {
		debug.Info("Client %s has a stricter retention (%d months) than default (%d months). Deleting associated hashlists immediately.", clientID, clientRetentionMonths, defaultRetentionMonths)

		hashlists, err := s.hashlistRepo.GetByClientID(ctx, clientID)
		if err != nil {
			debug.Error("Failed to get hashlists for client %s during immediate deletion: %v", clientID, err)
			return fmt.Errorf("failed to get hashlists for client %s: %w", clientID, err)
		}
		for _, hl := range hashlists {
			debug.Debug("Deleting hashlist %d and potentially orphaned hashes associated with client %s", hl.ID, clientID)

			err := s.retentionService.DeleteHashlistAndOrphanedHashes(ctx, hl.ID)
			if err != nil {
				debug.Error("Failed to delete hashlist %d (and orphans) during client %s deletion: %v", hl.ID, clientID, err)
				return fmt.Errorf("failed to delete hashlist %d for client %s: %w", hl.ID, clientID, err)
			}
		}
	} else {
		debug.Info("Client %s retention (%d months) is not stricter than default (%d months). Hashlists will be orphaned (client_id=NULL) and handled by purge job.", clientID, clientRetentionMonths, defaultRetentionMonths)
		// No action needed here, FK constraint ON DELETE SET NULL handles it.
	}

	// 5. Delete the client record itself
	debug.Debug("Deleting client record for ID: %s", clientID)
	err = s.clientRepo.Delete(ctx, clientID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return err
		}
		debug.Error("Failed to delete client record %s: %v", clientID, err)
		return fmt.Errorf("failed to delete client %s: %w", clientID, err)
	}

	debug.Info("Successfully deleted client %s and handled associated hashlists.", clientID)
	return nil
}

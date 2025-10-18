package services

import (
	"context"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/pkg/debug"
)

// TokenCleanupService handles automatic cleanup of expired tokens and sessions
type TokenCleanupService struct {
	db     *db.DB
	ticker *time.Ticker
	done   chan bool
}

// NewTokenCleanupService creates a new TokenCleanupService
func NewTokenCleanupService(database *db.DB) *TokenCleanupService {
	return &TokenCleanupService{
		db:   database,
		done: make(chan bool),
	}
}

// Start begins the periodic token cleanup process
func (s *TokenCleanupService) Start(ctx context.Context) {
	// Get cleanup interval from auth settings
	authSettings, err := s.db.GetAuthSettings()
	if err != nil {
		debug.Error("Failed to get auth settings for token cleanup: %v", err)
		// Default to 60 seconds if can't get settings
		s.ticker = time.NewTicker(60 * time.Second)
	} else {
		// Use default if interval is not set or invalid
		interval := time.Duration(authSettings.TokenCleanupIntervalSeconds) * time.Second
		if authSettings.TokenCleanupIntervalSeconds <= 0 {
			interval = 60 * time.Second
			debug.Warning("Token cleanup interval not set or invalid, using default: 60 seconds")
		}
		s.ticker = time.NewTicker(interval)
		debug.Info("Token cleanup service started with interval: %d seconds", int(interval.Seconds()))
	}

	// Run initial cleanup immediately
	go s.cleanup()

	// Start periodic cleanup
	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.cleanup()
			case <-s.done:
				return
			case <-ctx.Done():
				debug.Info("Token cleanup service context cancelled, stopping...")
				return
			}
		}
	}()
}

// Stop stops the token cleanup service
func (s *TokenCleanupService) Stop() {
	if s.ticker != nil {
		s.ticker.Stop()
	}
	s.done <- true
	debug.Info("Token cleanup service stopped")
}

// cleanup performs the actual token and session cleanup
func (s *TokenCleanupService) cleanup() {
	debug.Debug("Running token cleanup...")

	// Get auth settings to check for absolute timeout
	authSettings, err := s.db.GetAuthSettings()
	if err != nil {
		debug.Error("Failed to get auth settings during cleanup: %v", err)
		return
	}

	// Delete expired or revoked tokens (sessions CASCADE deleted)
	result, err := s.db.Exec(`
		DELETE FROM tokens
		WHERE expires_at < CURRENT_TIMESTAMP
		OR revoked = true
	`)
	if err != nil {
		debug.Error("Failed to delete expired tokens: %v", err)
		return
	}

	tokensDeleted, _ := result.RowsAffected()
	if tokensDeleted > 0 {
		debug.Info("Deleted %d expired or revoked tokens (sessions CASCADE deleted)", tokensDeleted)
	}

	// Check absolute session timeout if configured
	if authSettings.SessionAbsoluteTimeoutHours > 0 {
		result, err := s.db.Exec(`
			DELETE FROM active_sessions
			WHERE session_started_at < CURRENT_TIMESTAMP - INTERVAL '1 hour' * $1
		`, authSettings.SessionAbsoluteTimeoutHours)
		if err != nil {
			debug.Error("Failed to delete sessions exceeding absolute timeout: %v", err)
			return
		}

		sessionsDeleted, _ := result.RowsAffected()
		if sessionsDeleted > 0 {
			debug.Info("Deleted %d sessions exceeding absolute timeout (%d hours)",
				sessionsDeleted, authSettings.SessionAbsoluteTimeoutHours)
		}
	}

	debug.Debug("Token cleanup completed")
}

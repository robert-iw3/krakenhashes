//go:build integration
// +build integration

package services

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/db"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAnalyticsIntegration_SmallDataset tests analytics with small dataset
func TestAnalyticsIntegration_SmallDataset(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup database connection
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	dbConn, err := db.NewDatabase(dbURL)
	require.NoError(t, err)
	defer dbConn.Close()

	dbWrapper := db.NewDBWrapper(dbConn.DB)

	// Create repositories and service
	analyticsRepo := repository.NewAnalyticsRepository(dbWrapper)
	analyticsService := NewAnalyticsService(analyticsRepo)

	ctx := context.Background()

	// Create a test client
	clientID := uuid.New()
	_, err = dbWrapper.ExecContext(ctx,
		`INSERT INTO clients (id, name, description) VALUES ($1, $2, $3)`,
		clientID, "Integration Test Client", "Client for integration testing")
	require.NoError(t, err)

	defer cleanupTestData(t, dbWrapper, clientID)

	// Create test hashlists with small dataset
	hashlistID := createTestHashlist(t, dbWrapper, clientID)

	// Create analytics report
	report := &models.AnalyticsReport{
		ID:             uuid.New(),
		ClientID:       clientID,
		UserID:         uuid.New(),
		StartDate:      time.Now().Add(-30 * 24 * time.Hour),
		EndDate:        time.Now(),
		Status:         "queued",
		CustomPatterns: []string{"test", "integration"},
	}

	err = analyticsRepo.Create(ctx, report)
	require.NoError(t, err)

	// Generate analytics
	err = analyticsService.GenerateAnalytics(ctx, report.ID)
	require.NoError(t, err)

	// Retrieve and validate results
	result, err := analyticsRepo.GetByID(ctx, report.ID)
	require.NoError(t, err)

	assert.Equal(t, "completed", result.Status)
	assert.NotNil(t, result.AnalyticsData)

	// Validate analytics data structure
	data := result.AnalyticsData
	assert.NotNil(t, data.Overview)
	assert.NotNil(t, data.LengthDistribution)
	assert.NotNil(t, data.ComplexityAnalysis)
	assert.NotNil(t, data.PositionalAnalysis)
	assert.NotNil(t, data.PatternDetection)
	assert.NotNil(t, data.UsernameCorrelation)
	assert.NotNil(t, data.PasswordReuse)
	assert.NotNil(t, data.TemporalPatterns)
	assert.NotNil(t, data.MaskAnalysis)
	assert.NotNil(t, data.CustomPatterns)
	assert.NotNil(t, data.StrengthMetrics)
	assert.NotNil(t, data.TopPasswords)
	assert.NotNil(t, data.Recommendations)

	// Validate overview
	assert.Equal(t, result.TotalHashes, data.Overview.TotalHashes)
	assert.Equal(t, result.TotalCracked, data.Overview.TotalCracked)
	assert.Greater(t, data.Overview.CrackPercentage, 0.0)

	// Validate entropy distribution
	totalEntropyCount := data.StrengthMetrics.EntropyDistribution.Low.Count +
		data.StrengthMetrics.EntropyDistribution.Moderate.Count +
		data.StrengthMetrics.EntropyDistribution.High.Count
	assert.Equal(t, result.TotalCracked, totalEntropyCount)
}

// TestAnalyticsQueueProcessing tests the queue service
func TestAnalyticsQueueProcessing(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	dbConn, err := db.NewDatabase(dbURL)
	require.NoError(t, err)
	defer dbConn.Close()

	dbWrapper := db.NewDBWrapper(dbConn.DB)
	analyticsRepo := repository.NewAnalyticsRepository(dbWrapper)
	analyticsService := NewAnalyticsService(analyticsRepo)
	queueService := NewAnalyticsQueueService(analyticsService, analyticsRepo)

	ctx := context.Background()

	// Create test client
	clientID := uuid.New()
	_, err = dbWrapper.ExecContext(ctx,
		`INSERT INTO clients (id, name, description) VALUES ($1, $2, $3)`,
		clientID, "Queue Test Client", "Client for queue testing")
	require.NoError(t, err)

	defer cleanupTestData(t, dbWrapper, clientID)

	// Create test hashlist
	createTestHashlist(t, dbWrapper, clientID)

	// Start queue service
	err = queueService.Start()
	require.NoError(t, err)
	defer queueService.Stop()

	// Create multiple reports
	reportIDs := []uuid.UUID{}
	for i := 0; i < 3; i++ {
		report := &models.AnalyticsReport{
			ID:        uuid.New(),
			ClientID:  clientID,
			UserID:    uuid.New(),
			StartDate: time.Now().Add(-30 * 24 * time.Hour),
			EndDate:   time.Now(),
			Status:    "queued",
		}

		err = analyticsRepo.Create(ctx, report)
		require.NoError(t, err)
		reportIDs = append(reportIDs, report.ID)
	}

	// Wait for processing (with timeout)
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	allCompleted := false
	for !allCompleted {
		select {
		case <-timeout:
			t.Fatal("Timeout waiting for reports to process")
		case <-ticker.C:
			completed := 0
			for _, reportID := range reportIDs {
				report, err := analyticsRepo.GetByID(ctx, reportID)
				require.NoError(t, err)

				if report.Status == "completed" || report.Status == "failed" {
					completed++
				}
			}

			if completed == len(reportIDs) {
				allCompleted = true
			}
		}
	}

	// Verify all reports completed
	for _, reportID := range reportIDs {
		report, err := analyticsRepo.GetByID(ctx, reportID)
		require.NoError(t, err)
		assert.Equal(t, "completed", report.Status)
	}
}

// TestAnalyticsWithCustomPatterns tests custom pattern matching
func TestAnalyticsWithCustomPatterns(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	dbConn, err := db.NewDatabase(dbURL)
	require.NoError(t, err)
	defer dbConn.Close()

	dbWrapper := db.NewDBWrapper(dbConn.DB)
	analyticsRepo := repository.NewAnalyticsRepository(dbWrapper)
	analyticsService := NewAnalyticsService(analyticsRepo)

	ctx := context.Background()

	// Create test client
	clientID := uuid.New()
	_, err = dbWrapper.ExecContext(ctx,
		`INSERT INTO clients (id, name, description) VALUES ($1, $2, $3)`,
		clientID, "Custom Pattern Test", "Client for custom pattern testing")
	require.NoError(t, err)

	defer cleanupTestData(t, dbWrapper, clientID)

	// Create hashlist with custom pattern passwords
	hashlistID := createTestHashlistWithCustomPatterns(t, dbWrapper, clientID)

	// Create report with custom patterns
	report := &models.AnalyticsReport{
		ID:             uuid.New(),
		ClientID:       clientID,
		UserID:         uuid.New(),
		StartDate:      time.Now().Add(-30 * 24 * time.Hour),
		EndDate:        time.Now(),
		Status:         "queued",
		CustomPatterns: []string{"custom", "pattern"},
	}

	err = analyticsRepo.Create(ctx, report)
	require.NoError(t, err)

	// Generate analytics
	err = analyticsService.GenerateAnalytics(ctx, report.ID)
	require.NoError(t, err)

	// Verify custom patterns detected
	result, err := analyticsRepo.GetByID(ctx, report.ID)
	require.NoError(t, err)

	assert.NotNil(t, result.AnalyticsData.CustomPatterns)
	assert.Greater(t, len(result.AnalyticsData.CustomPatterns.PatternsDetected), 0)

	// Should have "custom" and "pattern" in detected patterns
	_, hasCustom := result.AnalyticsData.CustomPatterns.PatternsDetected["custom"]
	_, hasPattern := result.AnalyticsData.CustomPatterns.PatternsDetected["pattern"]

	assert.True(t, hasCustom || hasPattern, "Should detect at least one custom pattern")

	t.Cleanup(func() {
		cleanupHashlist(t, dbWrapper, hashlistID)
	})
}

// TestAnalyticsDateRangeFiltering tests date range filtering
func TestAnalyticsDateRangeFiltering(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	dbConn, err := db.NewDatabase(dbURL)
	require.NoError(t, err)
	defer dbConn.Close()

	dbWrapper := db.NewDBWrapper(dbConn.DB)
	analyticsRepo := repository.NewAnalyticsRepository(dbWrapper)
	analyticsService := NewAnalyticsService(analyticsRepo)

	ctx := context.Background()

	// Create test client
	clientID := uuid.New()
	_, err = dbWrapper.ExecContext(ctx,
		`INSERT INTO clients (id, name, description) VALUES ($1, $2, $3)`,
		clientID, "Date Range Test", "Client for date range testing")
	require.NoError(t, err)

	defer cleanupTestData(t, dbWrapper, clientID)

	// Create hashlists with different dates
	oldDate := time.Now().Add(-60 * 24 * time.Hour)
	recentDate := time.Now().Add(-15 * 24 * time.Hour)

	oldHashlistID := createTestHashlistWithDate(t, dbWrapper, clientID, oldDate)
	recentHashlistID := createTestHashlistWithDate(t, dbWrapper, clientID, recentDate)

	// Create report for only recent data (last 30 days)
	report := &models.AnalyticsReport{
		ID:        uuid.New(),
		ClientID:  clientID,
		UserID:    uuid.New(),
		StartDate: time.Now().Add(-30 * 24 * time.Hour),
		EndDate:   time.Now(),
		Status:    "queued",
	}

	err = analyticsRepo.Create(ctx, report)
	require.NoError(t, err)

	// Generate analytics
	err = analyticsService.GenerateAnalytics(ctx, report.ID)
	require.NoError(t, err)

	// Verify only recent hashlist was included
	result, err := analyticsRepo.GetByID(ctx, report.ID)
	require.NoError(t, err)

	// Should have 1 hashlist (recent) in the count
	assert.Equal(t, 1, result.TotalHashlists)

	t.Cleanup(func() {
		cleanupHashlist(t, dbWrapper, oldHashlistID)
		cleanupHashlist(t, dbWrapper, recentHashlistID)
	})
}

// Helper functions

func createTestHashlist(t *testing.T, db *db.DBWrapper, clientID uuid.UUID) int64 {
	ctx := context.Background()

	// Create user if needed
	userID := uuid.New()
	_, _ = db.ExecContext(ctx,
		`INSERT INTO users (id, username, email, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`,
		userID, "testuser", "test@example.com", "hashed", "user")

	// Create hashlist
	var hashlistID int64
	err := db.QueryRowContext(ctx,
		`INSERT INTO hashlists (name, user_id, client_id, hash_type_id, total_hashes, cracked_hashes, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		"Test Hashlist", userID, clientID, 0, 10, 10, "ready").Scan(&hashlistID)
	require.NoError(t, err)

	// Create sample hashes
	passwords := []string{
		"password", "Password123", "qwerty", "abc123", "Welcome1",
		"Summer2024", "john", "Password!", "test123", "admin",
	}

	for i, pwd := range passwords {
		hashID := uuid.New()
		username := "user" + string(rune('0'+i))

		// Insert hash
		_, err = db.ExecContext(ctx,
			`INSERT INTO hashes (id, hash_value, username, hash_type_id, is_cracked, password)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			hashID, "hash_"+pwd, username, 0, true, pwd)
		require.NoError(t, err)

		// Link to hashlist
		_, err = db.ExecContext(ctx,
			`INSERT INTO hashlist_hashes (hashlist_id, hash_id) VALUES ($1, $2)`,
			hashlistID, hashID)
		require.NoError(t, err)
	}

	return hashlistID
}

func createTestHashlistWithCustomPatterns(t *testing.T, db *db.DBWrapper, clientID uuid.UUID) int64 {
	ctx := context.Background()

	userID := uuid.New()
	_, _ = db.ExecContext(ctx,
		`INSERT INTO users (id, username, email, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`,
		userID, "testuser", "test@example.com", "hashed", "user")

	var hashlistID int64
	err := db.QueryRowContext(ctx,
		`INSERT INTO hashlists (name, user_id, client_id, hash_type_id, total_hashes, cracked_hashes, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id`,
		"Custom Pattern Hashlist", userID, clientID, 0, 5, 5, "ready").Scan(&hashlistID)
	require.NoError(t, err)

	// Create hashes with custom patterns
	passwords := []string{"custom123", "Pattern!", "CustomPattern", "test123", "password"}

	for _, pwd := range passwords {
		hashID := uuid.New()

		_, err = db.ExecContext(ctx,
			`INSERT INTO hashes (id, hash_value, hash_type_id, is_cracked, password)
			 VALUES ($1, $2, $3, $4, $5)`,
			hashID, "hash_"+pwd, 0, true, pwd)
		require.NoError(t, err)

		_, err = db.ExecContext(ctx,
			`INSERT INTO hashlist_hashes (hashlist_id, hash_id) VALUES ($1, $2)`,
			hashlistID, hashID)
		require.NoError(t, err)
	}

	return hashlistID
}

func createTestHashlistWithDate(t *testing.T, db *db.DBWrapper, clientID uuid.UUID, date time.Time) int64 {
	ctx := context.Background()

	userID := uuid.New()
	_, _ = db.ExecContext(ctx,
		`INSERT INTO users (id, username, email, password_hash, role)
		 VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`,
		userID, "testuser", "test@example.com", "hashed", "user")

	var hashlistID int64
	err := db.QueryRowContext(ctx,
		`INSERT INTO hashlists (name, user_id, client_id, hash_type_id, total_hashes, cracked_hashes, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		"Dated Hashlist", userID, clientID, 0, 5, 5, "ready", date).Scan(&hashlistID)
	require.NoError(t, err)

	// Create sample hashes
	for i := 0; i < 5; i++ {
		hashID := uuid.New()
		pwd := fmt.Sprintf("password%d", i)

		_, err = db.ExecContext(ctx,
			`INSERT INTO hashes (id, hash_value, hash_type_id, is_cracked, password)
			 VALUES ($1, $2, $3, $4, $5)`,
			hashID, "hash_"+pwd, 0, true, pwd)
		require.NoError(t, err)

		_, err = db.ExecContext(ctx,
			`INSERT INTO hashlist_hashes (hashlist_id, hash_id) VALUES ($1, $2)`,
			hashlistID, hashID)
		require.NoError(t, err)
	}

	return hashlistID
}

func cleanupTestData(t *testing.T, db *db.DBWrapper, clientID uuid.UUID) {
	ctx := context.Background()

	// Delete in reverse dependency order
	_, err := db.ExecContext(ctx, `DELETE FROM analytics_reports WHERE client_id = $1`, clientID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup analytics_reports: %v", err)
	}

	_, err = db.ExecContext(ctx, `DELETE FROM hashlist_hashes WHERE hashlist_id IN (SELECT id FROM hashlists WHERE client_id = $1)`, clientID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup hashlist_hashes: %v", err)
	}

	_, err = db.ExecContext(ctx, `DELETE FROM hashlists WHERE client_id = $1`, clientID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup hashlists: %v", err)
	}

	_, err = db.ExecContext(ctx, `DELETE FROM clients WHERE id = $1`, clientID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup client: %v", err)
	}
}

func cleanupHashlist(t *testing.T, db *db.DBWrapper, hashlistID int64) {
	ctx := context.Background()

	_, err := db.ExecContext(ctx, `DELETE FROM hashlist_hashes WHERE hashlist_id = $1`, hashlistID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup hashlist_hashes: %v", err)
	}

	_, err = db.ExecContext(ctx, `DELETE FROM hashlists WHERE id = $1`, hashlistID)
	if err != nil {
		t.Logf("Warning: Failed to cleanup hashlist: %v", err)
	}
}

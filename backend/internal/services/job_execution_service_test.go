package services

import (
	"context"
	"testing"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repositories for testing
type MockJobExecutionRepository struct {
	mock.Mock
}

func (m *MockJobExecutionRepository) Create(ctx context.Context, exec *models.JobExecution) error {
	args := m.Called(ctx, exec)
	// Set a test ID and timestamp
	exec.ID = uuid.New()
	exec.CreatedAt = time.Now()
	return args.Error(0)
}

func (m *MockJobExecutionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.JobExecution, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.JobExecution), args.Error(1)
}

func (m *MockJobExecutionRepository) GetPendingJobs(ctx context.Context) ([]models.JobExecution, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.JobExecution), args.Error(1)
}

func (m *MockJobExecutionRepository) StartExecution(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockJobExecutionRepository) UpdateProgress(ctx context.Context, id uuid.UUID, processedKeyspace int64) error {
	args := m.Called(ctx, id, processedKeyspace)
	return args.Error(0)
}

func (m *MockJobExecutionRepository) GetRunningJobs(ctx context.Context) ([]models.JobExecution, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.JobExecution), args.Error(1)
}

func (m *MockJobExecutionRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.JobExecutionStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockJobExecutionRepository) CompleteExecution(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockJobExecutionRepository) FailExecution(ctx context.Context, id uuid.UUID, errorMessage string) error {
	args := m.Called(ctx, id, errorMessage)
	return args.Error(0)
}

func (m *MockJobExecutionRepository) InterruptExecution(ctx context.Context, id uuid.UUID, interruptingJobID uuid.UUID) error {
	args := m.Called(ctx, id, interruptingJobID)
	return args.Error(0)
}

func (m *MockJobExecutionRepository) GetInterruptibleJobs(ctx context.Context, priority int) ([]models.JobExecution, error) {
	args := m.Called(ctx, priority)
	return args.Get(0).([]models.JobExecution), args.Error(1)
}

type MockPresetJobRepository struct {
	mock.Mock
}

func (m *MockPresetJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PresetJob, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.PresetJob), args.Error(1)
}

type MockHashlistRepository struct {
	mock.Mock
}

func (m *MockHashlistRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Hashlist, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.Hashlist), args.Error(1)
}

type MockSystemSettingsRepository struct {
	mock.Mock
}

func (m *MockSystemSettingsRepository) GetByKey(ctx context.Context, key string) (*models.SystemSetting, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SystemSetting), args.Error(1)
}

func TestJobExecutionService_CreateJobExecution(t *testing.T) {
	// Setup mocks
	mockJobExecRepo := new(MockJobExecutionRepository)
	mockPresetJobRepo := new(MockPresetJobRepository)
	mockHashlistRepo := new(MockHashlistRepository)

	// Create service
	service := &JobExecutionService{
		jobExecRepo:   mockJobExecRepo,
		presetJobRepo: mockPresetJobRepo,
		hashlistRepo:  mockHashlistRepo,
		// Skip hashcat binary path for this test since we're mocking
	}

	// Test data
	presetJobID := uuid.New()
	hashlistID := uuid.New()
	
	presetJob := &models.PresetJob{
		ID:         presetJobID,
		Name:       "Test Preset Job",
		AttackMode: models.AttackModeStraight,
		Priority:   100,
		WordlistIDs: models.IDArray{"1", "2"},
		RuleIDs:     models.IDArray{"1"},
	}

	hashlist := &models.Hashlist{
		ID:       hashlistID,
		Name:     "Test Hashlist",
		HashType: 0, // MD5
	}

	// Setup expectations
	mockPresetJobRepo.On("GetByID", mock.Anything, presetJobID).Return(presetJob, nil)
	mockHashlistRepo.On("GetByID", mock.Anything, hashlistID).Return(hashlist, nil)
	mockJobExecRepo.On("Create", mock.Anything, mock.AnythingOfType("*models.JobExecution")).Return(nil)

	// Execute test
	ctx := context.Background()
	userID := uuid.New()
	jobExecution, err := service.CreateJobExecution(ctx, presetJobID, hashlistID, &userID)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, jobExecution)
	assert.Equal(t, presetJobID, jobExecution.PresetJobID)
	assert.Equal(t, hashlistID, jobExecution.HashlistID)
	assert.Equal(t, models.JobExecutionStatusPending, jobExecution.Status)
	assert.Equal(t, presetJob.Priority, jobExecution.Priority)
	assert.Equal(t, presetJob.AttackMode, jobExecution.AttackMode)
	assert.NotEqual(t, uuid.Nil, jobExecution.ID)

	// Verify mock calls
	mockPresetJobRepo.AssertExpectations(t)
	mockHashlistRepo.AssertExpectations(t)
	mockJobExecRepo.AssertExpectations(t)
}

func TestJobExecutionService_GetNextPendingJob(t *testing.T) {
	// Setup mocks
	mockJobExecRepo := new(MockJobExecutionRepository)

	service := &JobExecutionService{
		jobExecRepo: mockJobExecRepo,
	}

	// Test data
	jobID1 := uuid.New()
	jobID2 := uuid.New()
	
	pendingJobs := []models.JobExecution{
		{
			ID:       jobID1,
			Priority: 200, // Higher priority
			Status:   models.JobExecutionStatusPending,
		},
		{
			ID:       jobID2,
			Priority: 100, // Lower priority
			Status:   models.JobExecutionStatusPending,
		},
	}

	// Setup expectations
	mockJobExecRepo.On("GetPendingJobs", mock.Anything).Return(pendingJobs, nil)

	// Execute test
	ctx := context.Background()
	nextJob, err := service.GetNextPendingJob(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, nextJob)
	assert.Equal(t, jobID1, nextJob.ID) // Should return highest priority job
	assert.Equal(t, 200, nextJob.Priority)

	// Verify mock calls
	mockJobExecRepo.AssertExpectations(t)
}

func TestJobExecutionService_GetNextPendingJob_NoPendingJobs(t *testing.T) {
	// Setup mocks
	mockJobExecRepo := new(MockJobExecutionRepository)

	service := &JobExecutionService{
		jobExecRepo: mockJobExecRepo,
	}

	// Setup expectations - no pending jobs
	mockJobExecRepo.On("GetPendingJobs", mock.Anything).Return([]models.JobExecution{}, nil)

	// Execute test
	ctx := context.Background()
	nextJob, err := service.GetNextPendingJob(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.Nil(t, nextJob) // Should return nil when no pending jobs

	// Verify mock calls
	mockJobExecRepo.AssertExpectations(t)
}

func TestJobExecutionService_CanInterruptJob(t *testing.T) {
	// Setup mocks
	mockJobExecRepo := new(MockJobExecutionRepository)
	mockSystemSettingsRepo := new(MockSystemSettingsRepository)

	service := &JobExecutionService{
		jobExecRepo:        mockJobExecRepo,
		systemSettingsRepo: mockSystemSettingsRepo,
	}

	// Test data
	interruptibleJobID := uuid.New()
	interruptibleJobs := []models.JobExecution{
		{
			ID:       interruptibleJobID,
			Priority: 50, // Lower priority
			Status:   models.JobExecutionStatusRunning,
		},
	}

	// Setup expectations
	interruptionSetting := &models.SystemSetting{
		Key:   "job_interruption_enabled",
		Value: "true",
	}
	mockSystemSettingsRepo.On("GetByKey", mock.Anything, "job_interruption_enabled").Return(interruptionSetting, nil)
	mockJobExecRepo.On("GetInterruptibleJobs", mock.Anything, 100).Return(interruptibleJobs, nil)

	// Execute test
	ctx := context.Background()
	jobs, err := service.CanInterruptJob(ctx, 100) // New job with priority 100

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, interruptibleJobID, jobs[0].ID)

	// Verify mock calls
	mockSystemSettingsRepo.AssertExpectations(t)
	mockJobExecRepo.AssertExpectations(t)
}

func TestJobExecutionService_CanInterruptJob_Disabled(t *testing.T) {
	// Setup mocks
	mockSystemSettingsRepo := new(MockSystemSettingsRepository)

	service := &JobExecutionService{
		systemSettingsRepo: mockSystemSettingsRepo,
	}

	// Setup expectations - interruption disabled
	interruptionSetting := &models.SystemSetting{
		Key:   "job_interruption_enabled",
		Value: "false",
	}
	mockSystemSettingsRepo.On("GetByKey", mock.Anything, "job_interruption_enabled").Return(interruptionSetting, nil)

	// Execute test
	ctx := context.Background()
	jobs, err := service.CanInterruptJob(ctx, 100)

	// Assertions
	assert.NoError(t, err)
	assert.Len(t, jobs, 0) // Should return empty slice when interruption is disabled

	// Verify mock calls
	mockSystemSettingsRepo.AssertExpectations(t)
}
//go:build unit
// +build unit

package services

import (
	"context"
	"testing"
	"time"

	"github.com/ZerkerEOD/krakenhashes/backend/internal/models"
	"github.com/ZerkerEOD/krakenhashes/backend/internal/repository"
	"github.com/google/uuid"
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

func (m *MockPresetJobRepository) Create(ctx context.Context, params models.PresetJob) (*models.PresetJob, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*models.PresetJob), args.Error(1)
}

func (m *MockPresetJobRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PresetJob, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.PresetJob), args.Error(1)
}

func (m *MockPresetJobRepository) GetByName(ctx context.Context, name string) (*models.PresetJob, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PresetJob), args.Error(1)
}

func (m *MockPresetJobRepository) List(ctx context.Context) ([]models.PresetJob, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.PresetJob), args.Error(1)
}

func (m *MockPresetJobRepository) Update(ctx context.Context, id uuid.UUID, params models.PresetJob) (*models.PresetJob, error) {
	args := m.Called(ctx, id, params)
	return args.Get(0).(*models.PresetJob), args.Error(1)
}

func (m *MockPresetJobRepository) Delete(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockPresetJobRepository) ListFormData(ctx context.Context) (*repository.PresetJobFormData, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*repository.PresetJobFormData), args.Error(1)
}

type MockHashlistRepository struct {
	mock.Mock
}

func (m *MockHashlistRepository) GetByID(ctx context.Context, id int64) (*models.HashList, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*models.HashList), args.Error(1)
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
	// NOTE: This test demonstrates the mock pattern but cannot actually instantiate
	// JobExecutionService since it requires concrete repository types, not mocks.
	// This is a limitation of the current architecture.
	t.Skip("Skipping due to architecture limitation: service requires concrete repository types")
}

func TestJobExecutionService_GetNextPendingJob(t *testing.T) {
	t.Skip("Skipping due to architecture limitation: service requires concrete repository types")
}

func TestJobExecutionService_GetNextPendingJob_NoPendingJobs(t *testing.T) {
	t.Skip("Skipping due to architecture limitation: service requires concrete repository types")
}

func TestJobExecutionService_CanInterruptJob(t *testing.T) {
	t.Skip("Skipping due to architecture limitation: service requires concrete repository types")
}

func TestJobExecutionService_CanInterruptJob_Disabled(t *testing.T) {
	t.Skip("Skipping due to architecture limitation: service requires concrete repository types")
}

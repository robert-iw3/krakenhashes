package repository

import (
	"context"
	"time"

	"github.com/yourusername/hashdom/internal/models"
	"github.com/yourusername/hashdom/pkg/debug"
	"gorm.io/gorm"
)

// AgentRepository handles database operations for agents
type AgentRepository struct {
	db *gorm.DB
}

// NewAgentRepository creates a new agent repository
func NewAgentRepository(db *gorm.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

// Create creates a new agent
func (r *AgentRepository) Create(ctx context.Context, agent *models.Agent) error {
	if err := r.db.WithContext(ctx).Create(agent).Error; err != nil {
		debug.Error("failed to create agent: %v", err)
		return err
	}
	return nil
}

// GetByID retrieves an agent by ID
func (r *AgentRepository) GetByID(ctx context.Context, id uint) (*models.Agent, error) {
	var agent models.Agent
	if err := r.db.WithContext(ctx).
		Preload("CreatedBy").
		Preload("Teams").
		First(&agent, id).Error; err != nil {
		debug.Error("failed to get agent by ID: %v", err)
		return nil, err
	}
	return &agent, nil
}

// GetByToken retrieves an agent by token
func (r *AgentRepository) GetByToken(ctx context.Context, token string) (*models.Agent, error) {
	var agent models.Agent
	if err := r.db.WithContext(ctx).
		Where("token = ?", token).
		Preload("CreatedBy").
		Preload("Teams").
		First(&agent).Error; err != nil {
		debug.Error("failed to get agent by token: %v", err)
		return nil, err
	}
	return &agent, nil
}

// List retrieves all agents with optional filters
func (r *AgentRepository) List(ctx context.Context, filters map[string]interface{}) ([]models.Agent, error) {
	var agents []models.Agent
	query := r.db.WithContext(ctx).
		Preload("CreatedBy").
		Preload("Teams")

	for key, value := range filters {
		query = query.Where(key+" = ?", value)
	}

	if err := query.Find(&agents).Error; err != nil {
		debug.Error("failed to list agents: %v", err)
		return nil, err
	}
	return agents, nil
}

// Update updates an agent
func (r *AgentRepository) Update(ctx context.Context, agent *models.Agent) error {
	if err := r.db.WithContext(ctx).Save(agent).Error; err != nil {
		debug.Error("failed to update agent: %v", err)
		return err
	}
	return nil
}

// Delete deletes an agent
func (r *AgentRepository) Delete(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).Delete(&models.Agent{}, id).Error; err != nil {
		debug.Error("failed to delete agent: %v", err)
		return err
	}
	return nil
}

// UpdateHeartbeat updates the last heartbeat time for an agent
func (r *AgentRepository) UpdateHeartbeat(ctx context.Context, id uint) error {
	if err := r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Where("id = ?", id).
		Update("last_heartbeat", time.Now()).Error; err != nil {
		debug.Error("failed to update agent heartbeat: %v", err)
		return err
	}
	return nil
}

// UpdateStatus updates the status of an agent
func (r *AgentRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	if !models.ValidStatus(status) {
		debug.Error("invalid agent status: %s", status)
		return ErrInvalidStatus
	}

	if err := r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Where("id = ?", id).
		Update("status", status).Error; err != nil {
		debug.Error("failed to update agent status: %v", err)
		return err
	}
	return nil
}

// SaveMetrics saves agent metrics
func (r *AgentRepository) SaveMetrics(ctx context.Context, metrics *models.AgentMetrics) error {
	if err := r.db.WithContext(ctx).Create(metrics).Error; err != nil {
		debug.Error("failed to save agent metrics: %v", err)
		return err
	}
	return nil
}

// GetMetrics retrieves agent metrics within a time range
func (r *AgentRepository) GetMetrics(ctx context.Context, agentID uint, start, end time.Time) ([]models.AgentMetrics, error) {
	var metrics []models.AgentMetrics
	if err := r.db.WithContext(ctx).
		Where("agent_id = ? AND timestamp BETWEEN ? AND ?", agentID, start, end).
		Order("timestamp DESC").
		Find(&metrics).Error; err != nil {
		debug.Error("failed to get agent metrics: %v", err)
		return nil, err
	}
	return metrics, nil
}

// ExistsByName checks if an agent exists with the given name
func (r *AgentRepository) ExistsByName(ctx context.Context, name string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Where("name = ?", name).
		Count(&count).Error; err != nil {
		debug.Error("failed to check agent name existence: %v", err)
		return false, err
	}
	return count > 0, nil
}

// GetByCertificate retrieves an agent by its certificate
func (r *AgentRepository) GetByCertificate(ctx context.Context, certPEM string) (*models.Agent, error) {
	var agent models.Agent
	if err := r.db.WithContext(ctx).
		Where("certificate = ?", certPEM).
		Preload("CreatedBy").
		Preload("Teams").
		First(&agent).Error; err != nil {
		debug.Error("failed to get agent by certificate: %v", err)
		return nil, err
	}
	return &agent, nil
}

// UpdateCertificate updates an agent's certificate
func (r *AgentRepository) UpdateCertificate(ctx context.Context, id uint, certPEM string) error {
	if err := r.db.WithContext(ctx).
		Model(&models.Agent{}).
		Where("id = ?", id).
		Update("certificate", certPEM).Error; err != nil {
		debug.Error("failed to update agent certificate: %v", err)
		return err
	}
	return nil
}

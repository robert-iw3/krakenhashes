package services

import (
	"context"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yourusername/hashdom/internal/models"
	"github.com/yourusername/hashdom/internal/repository"
	"github.com/yourusername/hashdom/pkg/debug"
)

// AgentService handles business logic for agents
type AgentService struct {
	agentRepo   *repository.AgentRepository
	voucherRepo *repository.ClaimVoucherRepository
}

// NewAgentService creates a new agent service
func NewAgentService(agentRepo *repository.AgentRepository, voucherRepo *repository.ClaimVoucherRepository) *AgentService {
	return &AgentService{
		agentRepo:   agentRepo,
		voucherRepo: voucherRepo,
	}
}

// RegisterAgent registers a new agent using a claim voucher
func (s *AgentService) RegisterAgent(ctx context.Context, claimCode string, hostname string, hardware models.Hardware, version string) (*models.Agent, error) {
	// Get and validate claim voucher
	voucher, err := s.voucherRepo.GetByCode(ctx, claimCode)
	if err != nil {
		debug.Error("failed to get claim voucher: %v", err)
		return nil, err
	}

	if !voucher.IsValid() {
		debug.Error("invalid claim voucher: %s", claimCode)
		return nil, repository.ErrInvalidVoucher
	}

	// Check for existing agent with same name and modify if needed
	name := hostname
	for i := 2; ; i++ {
		exists, err := s.agentRepo.ExistsByName(ctx, name)
		if err != nil {
			debug.Error("failed to check agent name: %v", err)
			return nil, err
		}
		if !exists {
			break
		}
		name = fmt.Sprintf("%s%d", hostname, i)
	}

	// Create new agent
	agent := &models.Agent{
		Name:        name,
		Status:      "inactive",
		Version:     version,
		Hardware:    hardware,
		CreatedByID: voucher.CreatedByID,
	}

	if err := s.agentRepo.Create(ctx, agent); err != nil {
		debug.Error("failed to create agent: %v", err)
		return nil, err
	}

	// Mark voucher as used
	if err := s.voucherRepo.Use(ctx, claimCode, agent.ID); err != nil {
		debug.Error("failed to mark voucher as used: %v", err)
		// Don't return here, agent is already created
	}

	return agent, nil
}

// HandleAgentConnection handles WebSocket connection from an agent
func (s *AgentService) HandleAgentConnection(ctx context.Context, conn *websocket.Conn, agent *models.Agent) error {
	defer conn.Close()

	// Update agent status to active
	if err := s.agentRepo.UpdateStatus(ctx, agent.ID, "active"); err != nil {
		debug.Error("failed to update agent status: %v", err)
		return err
	}

	// Create channels for communication
	done := make(chan struct{})
	errChan := make(chan error)

	// Start heartbeat goroutine
	go s.handleHeartbeat(ctx, agent.ID, done, errChan)

	// Start metrics receiver goroutine
	go s.handleMetrics(ctx, conn, agent.ID, done, errChan)

	// Wait for error or context cancellation
	select {
	case <-ctx.Done():
		close(done)
		return ctx.Err()
	case err := <-errChan:
		close(done)
		return err
	}
}

func (s *AgentService) handleHeartbeat(ctx context.Context, agentID uint, done chan struct{}, errChan chan error) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := s.agentRepo.UpdateHeartbeat(ctx, agentID); err != nil {
				debug.Error("failed to update heartbeat: %v", err)
				errChan <- err
				return
			}
		}
	}
}

func (s *AgentService) handleMetrics(ctx context.Context, conn *websocket.Conn, agentID uint, done chan struct{}, errChan chan error) {
	for {
		select {
		case <-done:
			return
		default:
			var metrics models.AgentMetrics
			if err := conn.ReadJSON(&metrics); err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					debug.Error("websocket error: %v", err)
					errChan <- err
				}
				return
			}

			metrics.AgentID = agentID
			metrics.Timestamp = time.Now()

			if err := s.agentRepo.SaveMetrics(ctx, &metrics); err != nil {
				debug.Error("failed to save metrics: %v", err)
				errChan <- err
				return
			}
		}
	}
}

// GetAgent retrieves an agent by ID
func (s *AgentService) GetAgent(ctx context.Context, id uint) (*models.Agent, error) {
	return s.agentRepo.GetByID(ctx, id)
}

// ListAgents retrieves all agents with optional filters
func (s *AgentService) ListAgents(ctx context.Context, filters map[string]interface{}) ([]models.Agent, error) {
	return s.agentRepo.List(ctx, filters)
}

// DeleteAgent deletes an agent
func (s *AgentService) DeleteAgent(ctx context.Context, id uint) error {
	return s.agentRepo.Delete(ctx, id)
}

// GetAgentMetrics retrieves agent metrics within a time range
func (s *AgentService) GetAgentMetrics(ctx context.Context, agentID uint, start, end time.Time) ([]models.AgentMetrics, error) {
	return s.agentRepo.GetMetrics(ctx, agentID, start, end)
}

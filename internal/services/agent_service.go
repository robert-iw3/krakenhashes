package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ZerkerEOD/hashdom-backend/internal/models"
	"github.com/ZerkerEOD/hashdom-backend/internal/repository"
	"github.com/ZerkerEOD/hashdom-backend/pkg/debug"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// AgentService handles agent-related operations
type AgentService struct {
	agentRepo   *repository.AgentRepository
	voucherRepo *repository.ClaimVoucherRepository
	tokens      map[string]downloadToken
	tokenMutex  sync.RWMutex
}

type downloadToken struct {
	AgentID   string
	ExpiresAt time.Time
}

// NewAgentService creates a new instance of AgentService
func NewAgentService(agentRepo *repository.AgentRepository, voucherRepo *repository.ClaimVoucherRepository) *AgentService {
	return &AgentService{
		agentRepo:   agentRepo,
		voucherRepo: voucherRepo,
		tokens:      make(map[string]downloadToken),
	}
}

// ValidateClaimCode validates a claim code
func (s *AgentService) ValidateClaimCode(ctx context.Context, claimCode string) error {
	voucher, err := s.voucherRepo.GetByCode(ctx, claimCode)
	if err != nil {
		return fmt.Errorf("invalid claim code")
	}

	if !voucher.IsValid() {
		return fmt.Errorf("claim code expired")
	}

	return nil
}

// RegisterAgent registers a new agent using a claim code
func (s *AgentService) RegisterAgent(ctx context.Context, claimCode, hostname string) (*models.Agent, error) {
	// Validate claim code
	voucher, err := s.voucherRepo.GetByCode(ctx, claimCode)
	if err != nil {
		return nil, fmt.Errorf("invalid claim code")
	}

	if !voucher.IsValid() {
		return nil, fmt.Errorf("claim code is not active")
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
	agentID := uuid.New()
	agent := &models.Agent{
		ID:          agentID.String(),
		Name:        name,
		Status:      models.AgentStatusPending,
		CreatedByID: voucher.CreatedByID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Version:     "1.0.0", // Set initial version
	}

	if err := s.agentRepo.Create(ctx, agent); err != nil {
		debug.Error("failed to create agent: %v", err)
		return nil, fmt.Errorf("failed to create agent: %v", err)
	}

	// Mark voucher as used
	if !voucher.IsContinuous {
		if err := s.voucherRepo.Use(ctx, claimCode, agentID); err != nil {
			debug.Warning("failed to mark voucher as used: %v", err)
			// Continue anyway since the agent was created successfully
		}
	}

	return agent, nil
}

// CreateDownloadToken generates a temporary token for certificate download
func (s *AgentService) CreateDownloadToken(ctx context.Context, agentID string) (string, error) {
	s.tokenMutex.Lock()
	defer s.tokenMutex.Unlock()

	// Generate token
	token := uuid.New().String()

	// Store token with expiration
	s.tokens[token] = downloadToken{
		AgentID:   agentID,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}

	return token, nil
}

// ValidateDownloadToken validates a download token and returns the associated agent ID
func (s *AgentService) ValidateDownloadToken(ctx context.Context, token string) (string, error) {
	s.tokenMutex.RLock()
	defer s.tokenMutex.RUnlock()

	dt, exists := s.tokens[token]
	if !exists {
		return "", fmt.Errorf("invalid download token")
	}

	if time.Now().After(dt.ExpiresAt) {
		delete(s.tokens, token)
		return "", fmt.Errorf("download token expired")
	}

	return dt.AgentID, nil
}

// InvalidateDownloadToken invalidates a download token
func (s *AgentService) InvalidateDownloadToken(ctx context.Context, token string) error {
	s.tokenMutex.Lock()
	defer s.tokenMutex.Unlock()

	delete(s.tokens, token)
	return nil
}

// GetAgent retrieves a single agent by ID
func (s *AgentService) GetAgent(ctx context.Context, id string) (*models.Agent, error) {
	debug.Info("Getting agent: %s", id)
	return s.agentRepo.GetByID(ctx, id)
}

// ListAgents retrieves all agents with optional filters
func (s *AgentService) ListAgents(ctx context.Context, filters map[string]interface{}) ([]models.Agent, error) {
	debug.Info("Listing agents with filters: %v", filters)
	return s.agentRepo.List(ctx, filters)
}

// DeleteAgent deletes an agent by ID
func (s *AgentService) DeleteAgent(ctx context.Context, id string) error {
	debug.Info("Deleting agent: %s", id)
	return s.agentRepo.Delete(ctx, id)
}

// HandleAgentConnection handles WebSocket connection from an agent
func (s *AgentService) HandleAgentConnection(ctx context.Context, conn *websocket.Conn, agent *models.Agent) error {
	defer conn.Close()

	// Update agent status to active
	if err := s.agentRepo.UpdateStatus(ctx, agent.ID, models.AgentStatusActive, nil); err != nil {
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

func (s *AgentService) handleHeartbeat(ctx context.Context, agentID string, done chan struct{}, errChan chan error) {
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

func (s *AgentService) handleMetrics(ctx context.Context, conn *websocket.Conn, agentID string, done chan struct{}, errChan chan error) {
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

// UpdateAgentStatus updates an agent's status and last error
func (s *AgentService) UpdateAgentStatus(ctx context.Context, id string, status string, lastError *string) error {
	return s.agentRepo.UpdateStatus(ctx, id, status, lastError)
}

// UpdateLastSeen updates the last_seen timestamp for an agent
func (s *AgentService) UpdateLastSeen(agentID string) error {
	ctx := context.Background()
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		debug.Error("failed to get agent: %v", err)
		return err
	}

	agent.LastSeen = time.Now()
	if err := s.agentRepo.Update(ctx, agent); err != nil {
		debug.Error("failed to update agent last seen: %v", err)
		return err
	}

	return nil
}

// UpdateHardwareInfo updates the hardware information for an agent
func (s *AgentService) UpdateHardwareInfo(agentID string, hardwareInfo map[string]interface{}) error {
	ctx := context.Background()
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		debug.Error("failed to get agent: %v", err)
		return err
	}

	// Convert hardware info to models.Hardware
	hardware := models.Hardware{
		CPUs:              make([]models.CPU, 0),
		GPUs:              make([]models.GPU, 0),
		NetworkInterfaces: make([]models.NetworkInterface, 0),
	}

	if cpus, ok := hardwareInfo["cpus"].([]interface{}); ok {
		for _, cpu := range cpus {
			if cpuMap, ok := cpu.(map[string]interface{}); ok {
				hardware.CPUs = append(hardware.CPUs, models.CPU{
					Model:   cpuMap["model"].(string),
					Cores:   int(cpuMap["cores"].(float64)),
					Threads: int(cpuMap["threads"].(float64)),
				})
			}
		}
	}

	if gpus, ok := hardwareInfo["gpus"].([]interface{}); ok {
		for _, gpu := range gpus {
			if gpuMap, ok := gpu.(map[string]interface{}); ok {
				hardware.GPUs = append(hardware.GPUs, models.GPU{
					Model:  gpuMap["model"].(string),
					Memory: gpuMap["memory"].(string),
					Driver: gpuMap["driver"].(string),
				})
			}
		}
	}

	if nics, ok := hardwareInfo["networkInterfaces"].([]interface{}); ok {
		for _, nic := range nics {
			if nicMap, ok := nic.(map[string]interface{}); ok {
				hardware.NetworkInterfaces = append(hardware.NetworkInterfaces, models.NetworkInterface{
					Name:      nicMap["name"].(string),
					IPAddress: nicMap["ipAddress"].(string),
				})
			}
		}
	}

	agent.Hardware = hardware
	if err := s.agentRepo.Update(ctx, agent); err != nil {
		debug.Error("failed to update agent hardware info: %v", err)
		return err
	}

	return nil
}

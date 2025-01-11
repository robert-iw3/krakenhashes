package services

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/ZerkerEOD/hashdom/backend/internal/models"
	"github.com/ZerkerEOD/hashdom/backend/internal/repository"
	"github.com/ZerkerEOD/hashdom/backend/pkg/debug"
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
	AgentID   int
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
	// Normalize claim code by removing hyphens and converting to uppercase
	normalizedCode := strings.ToUpper(strings.ReplaceAll(claimCode, "-", ""))

	voucher, err := s.voucherRepo.GetByCode(ctx, normalizedCode)
	if err != nil {
		return fmt.Errorf("invalid claim code")
	}

	if !voucher.IsValid() {
		return fmt.Errorf("claim code expired")
	}

	return nil
}

// RegisterAgent registers a new agent using a claim code.
// This is a single-step process that:
// 1. Validates the claim code
// 2. Generates a unique agent name
// 3. Generates a secure API key
// 4. Creates the agent record
//
// Parameters:
//   - ctx: Context for the operation
//   - claimCode: The voucher code for agent registration
//   - hostname: The agent's hostname for identification
//
// Returns:
//   - *models.Agent: The newly registered agent
//   - error: Any errors encountered during registration
func (s *AgentService) RegisterAgent(ctx context.Context, claimCode, hostname string) (*models.Agent, error) {
	debug.Info("Starting agent registration with claim code: %s, hostname: %s", claimCode, hostname)

	// Normalize claim code by removing hyphens and converting to uppercase
	normalizedCode := strings.ToUpper(strings.ReplaceAll(claimCode, "-", ""))
	debug.Debug("Normalized claim code: %s", normalizedCode)

	// Validate claim code
	voucher, err := s.voucherRepo.GetByCode(ctx, normalizedCode)
	if err != nil {
		debug.Error("Invalid claim code: %v", err)
		return nil, fmt.Errorf("invalid claim code")
	}

	if !voucher.IsValid() {
		debug.Error("Claim code is not active")
		return nil, fmt.Errorf("claim code is not active")
	}
	debug.Info("Claim code validated successfully")

	// Check for existing agent with same name and modify if needed
	name := hostname
	for i := 2; ; i++ {
		exists, err := s.agentRepo.ExistsByName(ctx, name)
		if err != nil {
			debug.Error("Failed to check agent name: %v", err)
			return nil, fmt.Errorf("failed to check agent name: %w", err)
		}
		if !exists {
			break
		}
		debug.Debug("Agent name %s already exists, trying %s%d", name, hostname, i)
		name = fmt.Sprintf("%s%d", hostname, i)
	}
	debug.Info("Using agent name: %s", name)

	// Generate API key (64-character hex string)
	apiKeyBytes := make([]byte, 32) // 32 bytes = 64 hex characters
	if _, err := rand.Read(apiKeyBytes); err != nil {
		debug.Error("Failed to generate API key: %v", err)
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}
	apiKey := hex.EncodeToString(apiKeyBytes)
	debug.Debug("Generated API key for agent")

	now := time.Now()

	// Create new agent
	agent := &models.Agent{
		Name:        name,
		Status:      models.AgentStatusPending,
		CreatedByID: voucher.CreatedByID,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     "1.0.0", // Set initial version
		APIKey: sql.NullString{
			String: apiKey,
			Valid:  true,
		},
		APIKeyCreatedAt: sql.NullTime{
			Time:  now,
			Valid: true,
		},
	}

	// Create agent record
	if err := s.agentRepo.Create(ctx, agent); err != nil {
		debug.Error("Failed to create agent: %v", err)
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	debug.Info("Successfully created agent with ID: %d", agent.ID)

	// Mark claim code as used if it's a single-use code
	if !voucher.IsContinuous {
		if err := s.MarkClaimCodeUsed(ctx, claimCode, agent.ID); err != nil {
			debug.Warning("Failed to mark claim code as used: %v", err)
			// Continue anyway as the agent is already created
		}
	}

	return agent, nil
}

// MarkClaimCodeUsed marks a claim code as used by an agent after successful connection
func (s *AgentService) MarkClaimCodeUsed(ctx context.Context, claimCode string, agentID int) error {
	// Normalize claim code
	normalizedCode := strings.ToUpper(strings.ReplaceAll(claimCode, "-", ""))

	// Get voucher
	voucher, err := s.voucherRepo.GetByCode(ctx, normalizedCode)
	if err != nil {
		return fmt.Errorf("invalid claim code")
	}

	// Only mark as used if it's a single-use code
	if !voucher.IsContinuous {
		if err := s.voucherRepo.UseByAgent(ctx, normalizedCode, agentID); err != nil {
			debug.Error("failed to mark voucher as used: %v", err)
			return fmt.Errorf("failed to mark voucher as used: %v", err)
		}
		// Deactivate single-use vouchers after use
		if err := s.voucherRepo.Deactivate(ctx, normalizedCode); err != nil {
			debug.Error("failed to deactivate voucher: %v", err)
			return fmt.Errorf("failed to deactivate voucher: %v", err)
		}
		debug.Info("Successfully marked single-use claim code as used for agent %d", agentID)
	}

	return nil
}

// CreateDownloadToken generates a temporary token for certificate download
func (s *AgentService) CreateDownloadToken(ctx context.Context, agentID int) (string, error) {
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
func (s *AgentService) ValidateDownloadToken(ctx context.Context, token string) (int, error) {
	s.tokenMutex.RLock()
	defer s.tokenMutex.RUnlock()

	dt, exists := s.tokens[token]
	if !exists {
		return 0, fmt.Errorf("invalid download token")
	}

	if time.Now().After(dt.ExpiresAt) {
		delete(s.tokens, token)
		return 0, fmt.Errorf("download token expired")
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
func (s *AgentService) GetAgent(ctx context.Context, id int) (*models.Agent, error) {
	debug.Info("Getting agent: %d", id)
	return s.agentRepo.GetByID(ctx, id)
}

// ListAgents retrieves all agents with optional filters
func (s *AgentService) ListAgents(ctx context.Context, filters map[string]interface{}) ([]models.Agent, error) {
	debug.Info("Listing agents with filters: %v", filters)
	return s.agentRepo.List(ctx, filters)
}

// DeleteAgent deletes an agent by ID
func (s *AgentService) DeleteAgent(ctx context.Context, id int) error {
	debug.Info("Deleting agent: %d", id)
	return s.agentRepo.Delete(ctx, id)
}

// HandleAgentConnection handles the WebSocket connection for an agent
func (s *AgentService) HandleAgentConnection(ctx context.Context, conn *websocket.Conn, agent *models.Agent) error {
	debug.Info("Agent %d connected via WebSocket", agent.ID)

	// Update agent status to connected
	now := time.Now()
	agent.Status = models.AgentStatusActive
	agent.LastSeen = now
	agent.APIKeyLastUsed = sql.NullTime{
		Time:  now,
		Valid: true,
	}
	if err := s.agentRepo.Update(ctx, agent); err != nil {
		debug.Error("failed to update agent status: %v", err)
		return fmt.Errorf("failed to update agent status: %v", err)
	}

	// Set up connection handling
	done := make(chan struct{})
	errChan := make(chan error)

	// Start heartbeat handler
	go s.handleHeartbeat(ctx, agent.ID, done, errChan)

	// Start metrics handler
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

// handleHeartbeat processes agent heartbeat
func (s *AgentService) handleHeartbeat(ctx context.Context, agentID int, done chan struct{}, errChan chan error) {
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

// handleMetrics processes agent metrics
func (s *AgentService) handleMetrics(ctx context.Context, conn *websocket.Conn, agentID int, done chan struct{}, errChan chan error) {
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
func (s *AgentService) UpdateAgentStatus(ctx context.Context, id int, status string, lastError *string) error {
	return s.agentRepo.UpdateStatus(ctx, id, status, lastError)
}

// UpdateLastSeen updates the last seen timestamp for an agent
func (s *AgentService) UpdateLastSeen(agentID int) error {
	now := time.Now()
	agent, err := s.agentRepo.GetByID(context.Background(), agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	agent.LastSeen = now
	agent.APIKeyLastUsed = sql.NullTime{
		Time:  now,
		Valid: true,
	}

	if err := s.agentRepo.Update(context.Background(), agent); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	return nil
}

// UpdateHardwareInfo updates the hardware info for an agent
func (s *AgentService) UpdateHardwareInfo(agentID int, hardwareInfo *models.Hardware, osInfo json.RawMessage) error {
	debug.Info("Updating hardware info for agent: %d", agentID)
	agent, err := s.agentRepo.GetByID(context.Background(), agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	// Update hardware info
	agent.Hardware = *hardwareInfo
	agent.OSInfo = osInfo
	agent.UpdatedAt = time.Now()

	// Update the agent in the database
	if err := s.agentRepo.Update(context.Background(), agent); err != nil {
		return fmt.Errorf("failed to update agent hardware: %w", err)
	}

	debug.Info("Successfully updated hardware info for agent: %d", agentID)
	return nil
}

// GetByID retrieves an agent by ID
func (s *AgentService) GetByID(ctx context.Context, id int) (*models.Agent, error) {
	return s.agentRepo.GetByID(ctx, id)
}

// Delete deletes an agent
func (s *AgentService) Delete(ctx context.Context, id int) error {
	return s.agentRepo.Delete(ctx, id)
}

// ProcessHeartbeat processes a heartbeat update for an agent
func (s *AgentService) ProcessHeartbeat(ctx context.Context, agentID int) error {
	return s.agentRepo.UpdateHeartbeat(ctx, agentID)
}

// ProcessMetrics processes metrics for an agent
func (s *AgentService) ProcessMetrics(ctx context.Context, agentID int, metrics *models.AgentMetrics) error {
	now := time.Now()
	metrics.Timestamp = now

	// Update agent's last seen time
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	agent.LastSeen = now
	agent.APIKeyLastUsed = sql.NullTime{
		Time:  now,
		Valid: true,
	}

	if err := s.agentRepo.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent: %w", err)
	}

	// Save metrics
	if err := s.agentRepo.SaveMetrics(ctx, metrics); err != nil {
		return fmt.Errorf("failed to save metrics: %w", err)
	}

	return nil
}

// Update updates an agent's information
func (s *AgentService) Update(ctx context.Context, agent *models.Agent) error {
	agent.UpdatedAt = time.Now()
	return s.agentRepo.Update(ctx, agent)
}

// ValidateAgentAPIKey validates an agent's API key and returns the agent ID if valid
func (s *AgentService) ValidateAgentAPIKey(ctx context.Context, apiKey string) (int, error) {
	agent, err := s.agentRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return 0, fmt.Errorf("invalid API key")
	}

	// Update last used timestamp
	now := time.Now()
	agent.APIKeyLastUsed = sql.NullTime{
		Time:  now,
		Valid: true,
	}
	if err := s.agentRepo.Update(ctx, agent); err != nil {
		debug.Error("failed to update API key last used: %v", err)
		// Continue anyway as this is not critical
	}

	return agent.ID, nil
}

// GetByAPIKey retrieves an agent by their API key
func (s *AgentService) GetByAPIKey(ctx context.Context, apiKey string) (*models.Agent, error) {
	agent, err := s.agentRepo.GetByAPIKey(ctx, apiKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent by API key: %w", err)
	}

	// Update last used timestamp
	now := time.Now()
	agent.APIKeyLastUsed = sql.NullTime{
		Time:  now,
		Valid: true,
	}
	if err := s.agentRepo.Update(ctx, agent); err != nil {
		debug.Error("failed to update API key last used: %v", err)
		// Continue anyway as this is not critical
	}

	return agent, nil
}

// UpdateHeartbeat updates the last heartbeat timestamp for an agent
func (s *AgentService) UpdateHeartbeat(ctx context.Context, agentID int) error {
	debug.Info("Updating heartbeat for agent: %d", agentID)
	agent, err := s.agentRepo.GetByID(ctx, agentID)
	if err != nil {
		return fmt.Errorf("failed to get agent: %w", err)
	}

	agent.LastHeartbeat = time.Now()
	agent.UpdatedAt = time.Now()

	if err := s.agentRepo.Update(ctx, agent); err != nil {
		return fmt.Errorf("failed to update agent heartbeat: %w", err)
	}

	debug.Info("Successfully updated heartbeat for agent: %d", agentID)
	return nil
}

package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ZerkerEOD/hashdom-backend/internal/db"
	"github.com/ZerkerEOD/hashdom-backend/internal/db/queries"
	"github.com/ZerkerEOD/hashdom-backend/internal/models"
)

// TeamRepository handles database operations for teams
type TeamRepository struct {
	db *db.DB
}

// NewTeamRepository creates a new team repository
func NewTeamRepository(db *db.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

// Create creates a new team
func (r *TeamRepository) Create(ctx context.Context, team *models.Team) error {
	err := r.db.QueryRowContext(ctx, queries.CreateTeam,
		team.ID,
		team.Name,
		team.Description,
		team.CreatedAt,
		team.UpdatedAt,
	).Scan(&team.ID)

	if err != nil {
		return fmt.Errorf("failed to create team: %w", err)
	}

	return nil
}

// GetByID retrieves a team by ID
func (r *TeamRepository) GetByID(ctx context.Context, id string) (*models.Team, error) {
	team := &models.Team{}

	err := r.db.QueryRowContext(ctx, queries.GetTeamByID, id).Scan(
		&team.ID,
		&team.Name,
		&team.Description,
		&team.CreatedAt,
		&team.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("team not found: %s", id)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	// Get team's users
	users, err := r.getTeamUsers(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team users: %w", err)
	}
	team.Users = users

	// Get team's agents
	agents, err := r.getTeamAgents(ctx, team.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team agents: %w", err)
	}
	team.Agents = agents

	return team, nil
}

// Update updates a team
func (r *TeamRepository) Update(ctx context.Context, team *models.Team) error {
	result, err := r.db.ExecContext(ctx, queries.UpdateTeam,
		team.ID,
		team.Name,
		team.Description,
		team.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update team: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("team not found: %s", team.ID)
	}

	return nil
}

// Delete deletes a team
func (r *TeamRepository) Delete(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, queries.DeleteTeam, id)
	if err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("team not found: %s", id)
	}

	return nil
}

// AddUser adds a user to the team
func (r *TeamRepository) AddUser(ctx context.Context, teamID, userID string) error {
	_, err := r.db.ExecContext(ctx, queries.AddUserToTeam, userID, teamID)
	if err != nil {
		return fmt.Errorf("failed to add user to team: %w", err)
	}
	return nil
}

// RemoveUser removes a user from the team
func (r *TeamRepository) RemoveUser(ctx context.Context, teamID, userID string) error {
	_, err := r.db.ExecContext(ctx, queries.RemoveUserFromTeam, userID, teamID)
	if err != nil {
		return fmt.Errorf("failed to remove user from team: %w", err)
	}
	return nil
}

// AddAgent adds an agent to the team
func (r *TeamRepository) AddAgent(ctx context.Context, teamID, agentID string) error {
	_, err := r.db.ExecContext(ctx, queries.AddAgentToTeam, agentID, teamID)
	if err != nil {
		return fmt.Errorf("failed to add agent to team: %w", err)
	}
	return nil
}

// RemoveAgent removes an agent from the team
func (r *TeamRepository) RemoveAgent(ctx context.Context, teamID, agentID string) error {
	_, err := r.db.ExecContext(ctx, queries.RemoveAgentFromTeam, agentID, teamID)
	if err != nil {
		return fmt.Errorf("failed to remove agent from team: %w", err)
	}
	return nil
}

// getTeamUsers retrieves all users in a team
func (r *TeamRepository) getTeamUsers(ctx context.Context, teamID string) ([]models.User, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.username, u.email, u.role, u.created_at, u.updated_at
		FROM users u
		JOIN user_teams ut ON u.id = ut.user_id
		WHERE ut.team_id = $1
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team users: %w", err)
	}
	defer rows.Close()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// getTeamAgents retrieves all agents in a team
func (r *TeamRepository) getTeamAgents(ctx context.Context, teamID string) ([]models.Agent, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.name, a.status, a.version, a.created_at, a.updated_at
		FROM agents a
		JOIN agent_teams at ON a.id = at.agent_id
		WHERE at.team_id = $1
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team agents: %w", err)
	}
	defer rows.Close()

	var agents []models.Agent
	for rows.Next() {
		var agent models.Agent
		err := rows.Scan(
			&agent.ID,
			&agent.Name,
			&agent.Status,
			&agent.Version,
			&agent.CreatedAt,
			&agent.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan agent: %w", err)
		}
		agents = append(agents, agent)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating agents: %w", err)
	}

	return agents, nil
}

// List retrieves all teams with optional filters
func (r *TeamRepository) List(ctx context.Context, filters map[string]interface{}) ([]models.Team, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM teams
		WHERE 1=1
	`
	args := make([]interface{}, 0)
	argPos := 1

	// Add filters if needed
	if name, ok := filters["name"].(string); ok {
		query += fmt.Sprintf(" AND name ILIKE $%d", argPos)
		args = append(args, "%"+name+"%")
		argPos++
	}

	query += " ORDER BY created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams: %w", err)
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var team models.Team
		err := rows.Scan(
			&team.ID,
			&team.Name,
			&team.Description,
			&team.CreatedAt,
			&team.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan team: %w", err)
		}

		// Get team's users
		users, err := r.getTeamUsers(ctx, team.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get team users: %w", err)
		}
		team.Users = users

		// Get team's agents
		agents, err := r.getTeamAgents(ctx, team.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get team agents: %w", err)
		}
		team.Agents = agents

		teams = append(teams, team)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating teams: %w", err)
	}

	return teams, nil
}

package queries

// Agent queries
const (
	CreateAgent = `
		INSERT INTO agents (
			name, status, last_heartbeat, version, hardware,
			os_info, created_by_id, created_at, updated_at, api_key,
			api_key_created_at, api_key_last_used, last_error, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) RETURNING id`

	GetAgentByID = `
		SELECT 
			a.id, a.name, a.status, a.last_error, a.last_heartbeat,
			a.version, a.hardware, a.os_info, a.created_by_id, a.created_at,
			a.updated_at, a.api_key, a.api_key_created_at,
			a.api_key_last_used, a.metadata,
			u.id, u.username, u.email, u.role
		FROM agents a
		LEFT JOIN users u ON a.created_by_id = u.id
		WHERE a.id = $1`

	ListAgents = `
		SELECT 
			a.id, a.name, a.status, a.last_error, a.last_heartbeat,
			a.version, a.hardware, a.os_info, a.created_by_id, a.created_at,
			a.updated_at, a.api_key, a.api_key_created_at,
			a.api_key_last_used, a.metadata,
			u.id, u.username, u.email, u.role
		FROM agents a
		LEFT JOIN users u ON a.created_by_id = u.id
		WHERE ($1::text IS NULL OR a.status = $1)
		ORDER BY a.created_at DESC`

	UpdateAgent = `
		UPDATE agents SET
			name = $2,
			status = $3,
			last_error = $4,
			last_heartbeat = $5,
			version = $6,
			hardware = $7,
			os_info = $8,
			updated_at = $9,
			api_key = $10,
			api_key_created_at = $11,
			api_key_last_used = $12,
			metadata = $13
		WHERE id = $1`

	UpdateAgentStatus = `
		UPDATE agents SET
			status = $2,
			last_error = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	UpdateAgentHeartbeat = `
		UPDATE agents SET
			last_heartbeat = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`

	GetAgentByAPIKey = `
		SELECT 
			a.id, a.name, a.status, a.last_error, a.last_heartbeat,
			a.version, a.hardware, a.os_info, a.created_by_id, a.created_at,
			a.updated_at, a.api_key, a.api_key_created_at,
			a.api_key_last_used, a.metadata,
			u.id, u.username, u.email, u.role
		FROM agents a
		LEFT JOIN users u ON a.created_by_id = u.id
		WHERE a.api_key = $1`
)

// User queries
const (
	CreateUser = `
		INSERT INTO users (
			id, username, email, password_hash, role,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING id`

	GetUserByID = `
		SELECT id, username, email, password_hash, role,
			created_at, updated_at
		FROM users
		WHERE id = $1`

	GetUserByEmail = `
		SELECT id, username, email, password_hash, role,
			created_at, updated_at
		FROM users
		WHERE email = $1`

	UpdateUser = `
		UPDATE users SET
			username = $2,
			email = $3,
			password_hash = $4,
			role = $5,
			updated_at = $6
		WHERE id = $1`

	DeleteUser = `DELETE FROM users WHERE id = $1`
)

// Team queries
const (
	CreateTeam = `
		INSERT INTO teams (
			id, name, description, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5
		) RETURNING id`

	GetTeamByID = `
		SELECT id, name, description, created_at, updated_at
		FROM teams
		WHERE id = $1`

	UpdateTeam = `
		UPDATE teams SET
			name = $2,
			description = $3,
			updated_at = $4
		WHERE id = $1`

	DeleteTeam = `DELETE FROM teams WHERE id = $1`

	AddUserToTeam = `
		INSERT INTO user_teams (user_id, team_id)
		VALUES ($1, $2)`

	RemoveUserFromTeam = `
		DELETE FROM user_teams
		WHERE user_id = $1 AND team_id = $2`

	AddAgentToTeam = `
		INSERT INTO agent_teams (agent_id, team_id)
		VALUES ($1, $2)`

	RemoveAgentFromTeam = `
		DELETE FROM agent_teams
		WHERE agent_id = $1 AND team_id = $2`
)

// ClaimVoucher queries
const (
	CreateClaimVoucher = `
		INSERT INTO claim_vouchers (
			code, is_active, is_continuous,
			created_by_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6
		) RETURNING code`

	GetClaimVoucherByCode = `
		SELECT 
			v.code, v.is_active, v.is_continuous,
			v.created_by_id, v.used_by_agent_id, v.used_at, v.created_at, v.updated_at,
			u1.id, u1.username, u1.email, u1.role,
			a.id, a.name, a.status
		FROM claim_vouchers v
		LEFT JOIN users u1 ON v.created_by_id = u1.id
		LEFT JOIN agents a ON v.used_by_agent_id = a.id
		WHERE v.code = $1`

	ListActiveVouchers = `
		SELECT 
			v.code, v.is_active, v.is_continuous,
			v.created_by_id, v.used_by_agent_id, v.used_at, v.created_at, v.updated_at,
			u1.id, u1.username, u1.email, u1.role,
			a.id, a.name, a.status
		FROM claim_vouchers v
		LEFT JOIN users u1 ON v.created_by_id = u1.id
		LEFT JOIN agents a ON v.used_by_agent_id = a.id
		WHERE v.is_active = true
		ORDER BY v.created_at DESC`

	UseClaimVoucherByAgent = `
		UPDATE claim_vouchers SET
			used_by_agent_id = $2,
			used_at = $3,
			updated_at = $3
		WHERE code = $1 AND is_active = true
		AND (is_continuous = true OR used_by_agent_id IS NULL)`

	DeactivateClaimVoucher = `
		UPDATE claim_vouchers SET
			is_active = false,
			updated_at = NOW()
		WHERE code = $1`
)

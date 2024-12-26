package queries

// Agent queries
const (
	CreateAgent = `
		INSERT INTO agents (
			id, name, status, last_error, last_heartbeat,
			version, hardware, created_by_id, created_at, updated_at,
			certificate, private_key
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		) RETURNING id`

	GetAgentByID = `
		SELECT 
			a.id, a.name, a.status, a.last_error, a.last_heartbeat,
			a.version, a.hardware, a.created_by_id, a.created_at, a.updated_at,
			a.certificate, a.private_key,
			u.id, u.username, u.email, u.role
		FROM agents a
		LEFT JOIN users u ON a.created_by_id = u.id
		WHERE a.id = $1`

	UpdateAgent = `
		UPDATE agents SET
			name = $2, status = $3, last_error = $4,
			last_heartbeat = $5, version = $6, hardware = $7,
			updated_at = $8, certificate = $9, private_key = $10
		WHERE id = $1`

	DeleteAgent = `DELETE FROM agents WHERE id = $1`

	ListAgents = `
		SELECT 
			a.id, a.name, a.status, a.last_error, a.last_heartbeat,
			a.version, a.hardware, a.created_by_id, a.created_at, a.updated_at,
			a.certificate, a.private_key,
			u.id, u.username, u.email, u.role
		FROM agents a
		LEFT JOIN users u ON a.created_by_id = u.id
		WHERE ($1::text IS NULL OR a.status = $1)
		ORDER BY a.created_at DESC`

	UpdateAgentStatus = `
		UPDATE agents SET
			status = $2,
			last_error = $3,
			updated_at = NOW()
		WHERE id = $1`

	UpdateAgentHeartbeat = `
		UPDATE agents SET
			last_heartbeat = NOW(),
			updated_at = NOW()
		WHERE id = $1`
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
			code, is_active, is_continuous, expires_at,
			created_by_id, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) RETURNING code`

	GetClaimVoucherByCode = `
		SELECT 
			v.code, v.is_active, v.is_continuous, v.expires_at,
			v.created_by_id, v.used_by_id, v.used_at, v.created_at, v.updated_at,
			u1.id, u1.username, u1.email, u1.role,
			u2.id, u2.username, u2.email, u2.role
		FROM claim_vouchers v
		LEFT JOIN users u1 ON v.created_by_id = u1.id
		LEFT JOIN users u2 ON v.used_by_id = u2.id
		WHERE v.code = $1`

	ListActiveVouchers = `
		SELECT 
			v.code, v.is_active, v.is_continuous, v.expires_at,
			v.created_by_id, v.used_by_id, v.used_at, v.created_at, v.updated_at,
			u1.id, u1.username, u1.email, u1.role,
			u2.id, u2.username, u2.email, u2.role
		FROM claim_vouchers v
		LEFT JOIN users u1 ON v.created_by_id = u1.id
		LEFT JOIN users u2 ON v.used_by_id = u2.id
		WHERE v.is_active = true
		ORDER BY v.created_at DESC`

	UseClaimVoucher = `
		UPDATE claim_vouchers SET
			is_active = CASE WHEN is_continuous THEN true ELSE false END,
			used_by_id = $2,
			used_at = $3,
			updated_at = $3
		WHERE code = $1 AND is_active = true
		AND (expires_at IS NULL OR expires_at > NOW())
		AND (is_continuous = true OR used_by_id IS NULL)`

	DeactivateClaimVoucher = `
		UPDATE claim_vouchers SET
			is_active = false,
			updated_at = NOW()
		WHERE code = $1`
)

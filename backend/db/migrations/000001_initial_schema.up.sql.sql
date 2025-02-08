-- Create tables
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT users_role_check CHECK (role IN ('user', 'admin', 'agent'))
);

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT teams_name_unique UNIQUE (name)
);

CREATE TABLE user_teams (
    user_id UUID NOT NULL,
    team_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'member',
    joined_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT user_teams_pkey PRIMARY KEY (user_id, team_id),
    CONSTRAINT user_teams_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT user_teams_team_id_fkey FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT user_teams_role_check CHECK (role IN ('member', 'admin'))
);

-- Create indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_role ON users(role);
CREATE INDEX idx_teams_name ON teams(name);
CREATE INDEX idx_user_teams_user_id ON user_teams(user_id);
CREATE INDEX idx_user_teams_team_id ON user_teams(team_id);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers
CREATE TRIGGER update_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_teams_updated_at
    BEFORE UPDATE ON teams
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert sample data with explicit UUIDs for referential integrity
DO $$ 
DECLARE
    admin_id UUID;
    user1_id UUID;
    user2_id UUID;
    team_a_id UUID;
    team_b_id UUID;
BEGIN
    -- Insert users and store their IDs for testing
    INSERT INTO users (username, first_name, last_name, email, password_hash, role)
    VALUES ('admin', 'Admin', 'User', 'admin@example.com', '$2a$10$2gobOj6ATVGUNNk5CHw9de2reYqSZVHtP/Qrx63.Ho9nTWbo5PW7O', 'admin') -- password: KrakenHashes1!
    RETURNING id INTO admin_id;

    INSERT INTO users (username, first_name, last_name, email, password_hash, role)
    VALUES ('user1', 'User', 'One', 'user1@example.com', '$2a$10$2gobOj6ATVGUNNk5CHw9de2reYqSZVHtP/Qrx63.Ho9nTWbo5PW7O', 'user') -- password: KrakenHashes1!
    RETURNING id INTO user1_id;

    INSERT INTO users (username, first_name, last_name, email, password_hash, role)
    VALUES ('user2', 'User', 'Two', 'user2@example.com', '$2a$10$2gobOj6ATVGUNNk5CHw9de2reYqSZVHtP/Qrx63.Ho9nTWbo5PW7O', 'user') -- password: KrakenHashes1!
    RETURNING id INTO user2_id;

    -- Insert teams
    INSERT INTO teams (name, description) 
    VALUES ('Team A', 'This is Team A') 
    RETURNING id INTO team_a_id;

    INSERT INTO teams (name, description) 
    VALUES ('Team B', 'This is Team B') 
    RETURNING id INTO team_b_id;

    -- Insert team memberships
    INSERT INTO user_teams (user_id, team_id, role) VALUES
    (admin_id, team_a_id, 'admin'),
    (user1_id, team_a_id, 'member'),
    (user1_id, team_b_id, 'member'),
    (user2_id, team_b_id, 'admin');
END $$;

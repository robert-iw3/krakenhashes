-- Create tables
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE auth (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    token VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE(token)
);

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_teams (
    user_id UUID NOT NULL,
    team_id UUID NOT NULL,
    role VARCHAR(50) NOT NULL,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, team_id),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE
);

-- Create indexes
CREATE INDEX idx_auth_user_id ON auth(user_id);
CREATE INDEX idx_user_teams_user_id ON user_teams(user_id);
CREATE INDEX idx_user_teams_team_id ON user_teams(team_id);

-- Insert sample data with explicit UUIDs for referential integrity
DO $$ 
DECLARE
    admin_id UUID;
    user1_id UUID;
    user2_id UUID;
    team_a_id UUID;
    team_b_id UUID;
BEGIN
    -- Insert users and store their IDs
    INSERT INTO users (username, first_name, last_name, email, password_hash)
    VALUES ('admin', 'Admin', 'User', 'admin@example.com', '$2a$10$UQzDYVagF4svlb9zhvWFZOHrNa6xEwcqRVJ3l9WEn8VOfrCuy7Q8q')
    RETURNING id INTO admin_id;

    INSERT INTO users (username, first_name, last_name, email, password_hash)
    VALUES ('user1', 'User', 'One', 'user1@example.com', '$2a$10$UQzDYVagF4svlb9zhvWFZOHrNa6xEwcqRVJ3l9WEn8VOfrCuy7Q8q')
    RETURNING id INTO user1_id;

    INSERT INTO users (username, first_name, last_name, email, password_hash)
    VALUES ('user2', 'User', 'Two', 'user2@example.com', '$2a$10$UQzDYVagF4svlb9zhvWFZOHrNa6xEwcqRVJ3l9WEn8VOfrCuy7Q8q')
    RETURNING id INTO user2_id;

    -- Insert auth records
    INSERT INTO auth (user_id) VALUES (admin_id), (user1_id), (user2_id);

    -- Insert teams
    INSERT INTO teams (name, description) VALUES ('Team A', 'This is Team A') RETURNING id INTO team_a_id;
    INSERT INTO teams (name, description) VALUES ('Team B', 'This is Team B') RETURNING id INTO team_b_id;

    -- Insert team memberships
    INSERT INTO user_teams (user_id, team_id, role) VALUES
    (admin_id, team_a_id, 'admin'),
    (user1_id, team_a_id, 'member'),
    (user1_id, team_b_id, 'member'),
    (user2_id, team_b_id, 'admin');
END $$;

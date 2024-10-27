-- Create tables
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) UNIQUE NOT NULL,
    first_name VARCHAR(255),
    last_name VARCHAR(255),
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE auth (
    id SERIAL PRIMARY KEY,
    user_id INTEGER REFERENCES users(id),
    token VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    UNIQUE(token)
);

CREATE TABLE teams (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_teams (
    user_id INTEGER NOT NULL,
    team_id INTEGER NOT NULL,
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

-- Insert sample data
INSERT INTO users (username, first_name, last_name, email, password_hash) VALUES
('admin', 'Admin', 'User', 'admin@example.com', '$2a$10$UQzDYVagF4svlb9zhvWFZOHrNa6xEwcqRVJ3l9WEn8VOfrCuy7Q8q'),  -- password: hashdom
('user1', 'User', 'One', 'user1@example.com', '$2a$10$UQzDYVagF4svlb9zhvWFZOHrNa6xEwcqRVJ3l9WEn8VOfrCuy7Q8q'),   -- password: hashdom
('user2', 'User', 'Two', 'user2@example.com', '$2a$10$UQzDYVagF4svlb9zhvWFZOHrNa6xEwcqRVJ3l9WEn8VOfrCuy7Q8q');   -- password: hashdom

-- The auth table should only store authentication-related data like tokens, not password hashes
INSERT INTO auth (user_id) VALUES
(1),
(2),
(3);

INSERT INTO teams (name, description) VALUES
('Team A', 'This is Team A'),
('Team B', 'This is Team B');

INSERT INTO user_teams (user_id, team_id, role) VALUES
(1, 1, 'admin'),
(2, 1, 'member'),
(2, 2, 'member'),
(3, 2, 'admin');

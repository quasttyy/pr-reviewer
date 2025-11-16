-- Команды
CREATE TABLE IF NOT EXISTS teams (
    team_name VARCHAR(100) PRIMARY KEY
);

-- Пользователи
CREATE TABLE IF NOT EXISTS users (
    user_id VARCHAR(100) PRIMARY KEY,
    username VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    team_name VARCHAR(100) NOT NULL REFERENCES teams(team_name) ON UPDATE CASCADE
);

-- Пулл-реквесты
CREATE TABLE IF NOT EXISTS pull_requests (
    pull_request_id VARCHAR(150) PRIMARY KEY,
    pull_request_name VARCHAR(200) NOT NULL,
    author_id VARCHAR(100) NOT NULL REFERENCES users(user_id) ON UPDATE CASCADE,
    status VARCHAR(6) NOT NULL CHECK (status IN ('OPEN', 'MERGED')),
    need_more_reviewers BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    merged_at TIMESTAMPTZ
);

-- Связка PR и ревьюеров (many-to-many)
CREATE TABLE IF NOT EXISTS pr_reviewers (
    pull_request_id VARCHAR(150) NOT NULL REFERENCES pull_requests(pull_request_id) ON DELETE CASCADE,
    reviewer_id VARCHAR(100) NOT NULL REFERENCES users(user_id) ON UPDATE CASCADE,
    PRIMARY KEY (pull_request_id, reviewer_id)
);

-- Индексы
CREATE INDEX IF NOT EXISTS idx_users_team_name ON users(team_name);
CREATE INDEX IF NOT EXISTS idx_pr_author_id ON pull_requests(author_id);
CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(status);
CREATE INDEX IF NOT EXISTS idx_pr_reviewers_reviewer ON pr_reviewers(reviewer_id);
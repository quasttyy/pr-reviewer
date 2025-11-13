# Команды
CREATE TABLE IF NOT EXISTS teams (
    id SERIAL PRIMARY KEY,
    team_name VARCHAR(100) UNIQUE NOT NULL
);

# Пользователи
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    user_name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    team_id INT NOT NULL REFERENCES teams(id)
);

# Пулл-реквесты
CREATE TABLE IF NOT EXISTS pull_requests (
    id SERIAL PRIMARY KEY,
    pr_name VARCHAR(100) NOT NULL,
    author_id INT NOT NULL REFERENCES users(id),
    pr_status VARCHAR(6) NOT NULL CHECK (pr_status IN ('OPEN', 'MERGED')),
    need_more_reviewers BOOLEAN NOT NULL DEFAULT FALSE
);

# Ревьюеры (many to many)
CREATE TABLE IF NOT EXISTS pr_reviewers (
    pr_id INT NOT NULL REFERENCES pull_requests(id),
    reviewer_id INT NOT NULL REFERENCES users(id),
    PRIMARY KEY(pr_id, reviewer_id)
);

# Индексы
CREATE INDEX IF NOT EXISTS idx_users_team_id ON users(team_id);
CREATE INDEX IF NOT EXISTS idx_pr_author_id ON pull_requests(author_id);
CREATE INDEX IF NOT EXISTS idx_pr_status ON pull_requests(pr_status);
CREATE INDEX IF NOT EXISTS idx_pr_reviewers_pr ON pr_reviewers(pr_id);
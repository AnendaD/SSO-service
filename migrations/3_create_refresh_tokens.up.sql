CREATE TABLE refresh_tokens (
    token VARCHAR(64) PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id INTEGER NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP

);

CREATE INDEX idx_refresh_tokens_user_app ON refresh_tokens(user_id, app_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);
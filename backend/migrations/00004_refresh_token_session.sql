-- +goose Up
ALTER TABLE refresh_tokens
    ADD COLUMN session_id uuid REFERENCES auth_sessions(id) ON DELETE CASCADE;

CREATE INDEX refresh_tokens_session_id_idx ON refresh_tokens (session_id);

-- +goose Down
DROP INDEX IF EXISTS refresh_tokens_session_id_idx;
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS session_id;

-- +goose Up
CREATE TABLE refresh_tokens (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id uuid NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
    token_hash text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    CONSTRAINT refresh_tokens_expiry_chk CHECK (expires_at > created_at)
);

CREATE INDEX refresh_tokens_client_id_idx ON refresh_tokens (client_id);
CREATE INDEX refresh_tokens_token_hash_idx ON refresh_tokens (token_hash);

-- +goose Down
DROP TABLE refresh_tokens;

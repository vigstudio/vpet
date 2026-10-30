CREATE TABLE IF NOT EXISTS world_messages (
    id bigserial PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    body text NOT NULL CHECK (char_length(body) BETWEEN 1 AND 200),
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS world_messages_created_idx ON world_messages(id DESC);

CREATE TABLE IF NOT EXISTS world_message_reports (
    message_id bigint NOT NULL REFERENCES world_messages(id) ON DELETE CASCADE,
    reporter_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (message_id, reporter_id)
);

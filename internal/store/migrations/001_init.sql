CREATE TABLE IF NOT EXISTS accounts (
    id text PRIMARY KEY,
    username text NOT NULL,
    username_normalized text NOT NULL UNIQUE,
    display_name text NOT NULL DEFAULT '',
    locale text NOT NULL DEFAULT 'vi',
    password_hash text,
    disabled boolean NOT NULL DEFAULT false,
    claimed_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sessions (
    token_hash bytea PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    csrf_token text NOT NULL,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS sessions_account_idx ON sessions(account_id);

CREATE TABLE IF NOT EXISTS game_kv (
    pkey text PRIMARY KEY,
    pvalue jsonb NOT NULL,
    version bigint NOT NULL DEFAULT 1,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS notifications (
    id bigserial PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    reason text NOT NULL,
    message text NOT NULL DEFAULT '',
    tab text NOT NULL DEFAULT '',
    duel text NOT NULL DEFAULT '',
    created_at timestamptz NOT NULL DEFAULT now(),
    read_at timestamptz
);
CREATE INDEX IF NOT EXISTS notifications_account_id_idx ON notifications(account_id, id);

CREATE TABLE IF NOT EXISTS migration_batches (
    id text PRIMARY KEY,
    enabled boolean NOT NULL DEFAULT true,
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS migration_claims (
    batch_id text NOT NULL REFERENCES migration_batches(id),
    mattermost_user_id text NOT NULL,
    username_normalized text NOT NULL,
    redeemed_account_id text REFERENCES accounts(id),
    redeemed_at timestamptz,
    PRIMARY KEY (batch_id, mattermost_user_id)
);
CREATE UNIQUE INDEX IF NOT EXISTS migration_claim_username_idx
    ON migration_claims(batch_id, username_normalized);

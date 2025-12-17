CREATE TABLE IF NOT EXISTS cliproxy_api_keys (
    id TEXT PRIMARY KEY,
    secret TEXT NOT NULL,
    provider TEXT NOT NULL,
    label TEXT,
    secret_preview TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    limit_per_minute INTEGER,
    limit_per_day INTEGER,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    source TEXT NOT NULL DEFAULT 'remote',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cliproxy_usage_events (
    id BIGSERIAL PRIMARY KEY,
    event_id TEXT UNIQUE,
    api_key_id TEXT REFERENCES cliproxy_api_keys(id) ON DELETE SET NULL,
    provider TEXT,
    model TEXT,
    source TEXT,
    failed BOOLEAN DEFAULT FALSE,
    total_tokens INTEGER,
    input_tokens INTEGER,
    output_tokens INTEGER,
    reasoning_tokens INTEGER,
    cached_tokens INTEGER,
    cost_usd DOUBLE PRECISION,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cliproxy_sync_state (
    slug TEXT PRIMARY KEY,
    cursor TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

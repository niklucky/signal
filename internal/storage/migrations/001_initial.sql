-- Initial schema: users, hosts, events.

CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'user',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS hosts (
    id SERIAL PRIMARY KEY,
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    method TEXT NOT NULL DEFAULT 'GET',
    url TEXT NOT NULL,
    headers JSONB,
    body TEXT,
    timeout_sec INTEGER NOT NULL DEFAULT 10,
    interval_sec INTEGER NOT NULL,
    resend_interval_sec INTEGER,
    expected_status INTEGER NOT NULL DEFAULT 200,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_hosts_user_id ON hosts(user_id);

CREATE TABLE IF NOT EXISTS events (
    id BIGSERIAL PRIMARY KEY,
    host_id INTEGER NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    beacon_id INTEGER NOT NULL DEFAULT 1,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    response_time_ms INTEGER,
    response_status INTEGER,
    success BOOLEAN NOT NULL,
    error_message TEXT,
    body_snippet TEXT
);

CREATE INDEX IF NOT EXISTS idx_events_host_ts ON events(host_id, ts DESC);

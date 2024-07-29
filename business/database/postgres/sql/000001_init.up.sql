CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
    command TEXT NOT NULL,
    args TEXT[] NOT NULL,
    status TEXT NOT NULL,
    result TEXT,
    error_msg TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    scheduled_at TIMESTAMP NOT NULL
);

CREATE TABLE IF NOT EXISTS users(
    id UUID PRIMARY KEY,
    name TEXT  NOT NULL,
    email TEXT NOT NULL UNIQUE,
    roles TEXT[] NOT NULL,
    password_hash TEXT NOT NULL,
    enabled BOOLEAN NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);

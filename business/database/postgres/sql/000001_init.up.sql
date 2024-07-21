CREATE TABLE IF NOT EXISTS tasks (
   id TEXT PRIMARY KEY,
    command TEXT NOT NULL,
    args TEXT[] NOT NULL,
    status TEXT NOT NULL,
    result TEXT,
    error_msg TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    scheduled_at TIMESTAMP NOT NULL
);


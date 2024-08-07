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


CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
  -- since we are going to treat each domain as a standalone service, we can not use "FOREIGN KEY CONSTRAINT" since that assumes each
  -- table to be in the same database.
    user_id UUID NOT NULL,
    command TEXT NOT NULL,
    args TEXT[],
    image VARCHAR(100) NOT NULL,
    environment TEXT NOT NULL,
    status TEXT NOT NULL,
    result TEXT,
    error_msg TEXT,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    scheduled_at TIMESTAMP NOT NULL
);


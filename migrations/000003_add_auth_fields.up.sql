-- There is no users table in the original movie schema; authentication starts
-- here with a separate table, leaving the existing movie API schema unchanged.
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL,
    name TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'viewer',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_users_role CHECK (role IN ('viewer', 'editor', 'admin'))
);

CREATE UNIQUE INDEX uq_users_email_lower ON users (lower(email));

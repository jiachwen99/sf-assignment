-- +goose Up

-- Authentication supplies identity. It never partitions data: every account
-- sees the same task list. That is what makes an optional feature compatible
-- with the requirement that users share one list rather than contradicting it.
CREATE TABLE users (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         text NOT NULL,
    name          text NOT NULL,
    password_hash text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- Case-insensitive uniqueness without needing the citext extension, and the
-- index the login lookup uses.
CREATE UNIQUE INDEX users_by_email ON users (lower(email));

CREATE TABLE sessions (
    token      text PRIMARY KEY,
    user_id    bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX sessions_by_user ON sessions (user_id);

-- The history has been recording changes since SF-009 with nobody attached to
-- them, because there was nobody to attach. ON DELETE SET NULL rather than
-- CASCADE: deleting an account must not delete the record of what it did.
ALTER TABLE todo_events
    ADD COLUMN actor_id bigint REFERENCES users(id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE todo_events DROP COLUMN actor_id;
DROP TABLE sessions;
DROP TABLE users;

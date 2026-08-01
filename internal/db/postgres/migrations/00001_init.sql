-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    display_id text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    theme_color text NOT NULL DEFAULT ''
);

CREATE TABLE public_keys (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    display_id text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    fingerprint text NOT NULL UNIQUE,
    type text NOT NULL,
    user_id text NOT NULL
);

CREATE TABLE files (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    display_id text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    size bigint NOT NULL CHECK (size >= 0),
    content bytea NOT NULL,
    private boolean NOT NULL,
    type text NOT NULL,
    user_id text NOT NULL,
    name text
);

CREATE INDEX idx_files_user_id_id ON files (user_id, id DESC);
CREATE UNIQUE INDEX idx_files_user_id_name ON files (user_id, lower(name)) WHERE name IS NOT NULL;

CREATE TABLE revisions (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    display_id text NOT NULL UNIQUE,
    sequence bigint NOT NULL,
    file_id text NOT NULL,
    created_at timestamptz NOT NULL,
    diff bytea NOT NULL,
    size bigint NOT NULL CHECK (size >= 0),
    type text NOT NULL,
    UNIQUE (file_id, sequence)
);

CREATE INDEX idx_revisions_file_id_id ON revisions (file_id, id DESC);

CREATE TABLE api_keys (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    display_id text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL,
    updated_at timestamptz NOT NULL,
    name text,
    token_hash text NOT NULL UNIQUE,
    user_id text NOT NULL,
    last_used_at timestamptz,
    expires_at timestamptz
);

CREATE INDEX idx_api_keys_user_id_created_at ON api_keys (user_id, created_at DESC, display_id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE api_keys;
DROP TABLE revisions;
DROP TABLE files;
DROP TABLE public_keys;
DROP TABLE users;
-- +goose StatementEnd

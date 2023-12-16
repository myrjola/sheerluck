CREATE TABLE sessions
(
    token  TEXT PRIMARY KEY,
    data   BYTEA       NOT NULL,
    expiry TIMESTAMPTZ NOT NULL
);

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

CREATE TABLE IF NOT EXISTS users
(
    id           BYTEA PRIMARY KEY,
    display_name TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credentials
(
    id                          BYTEA PRIMARY KEY,
    public_key                  BYTEA,
    attestation_type            TEXT,
    transport                   TEXT[],
    flag_user_present           BOOLEAN,
    flag_user_verified          BOOLEAN,
    flag_backup_eligible        BOOLEAN,
    flag_backup_state           BOOLEAN,
    authenticator_aaguid        BYTEA,
    authenticator_sign_count    INTEGER,
    authenticator_clone_warning BOOLEAN,
    authenticator_attachment     TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    user_id                     BYTEA REFERENCES users (id) ON DELETE CASCADE
);

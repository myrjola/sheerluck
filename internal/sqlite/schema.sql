CREATE TABLE sessions
(
    token  TEXT PRIMARY KEY CHECK (length(token) < 256),
    data   BLOB NOT NULL CHECK (length(data) < 2056),
    expiry REAL NOT NULL
) WITHOUT ROWID, STRICT;

CREATE INDEX sessions_expiry_idx ON sessions (expiry);

CREATE TABLE users
(
    id           BLOB PRIMARY KEY CHECK (length(id) < 256),
    display_name TEXT NOT NULL CHECK (length(display_name) < 64),

    created      TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ')) CHECK (length(created) < 256),
    updated      TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ')) CHECK (length(updated) < 256)
) WITHOUT ROWID, STRICT;

CREATE TRIGGER users_updated_timestamp
    AFTER UPDATE
    ON users
BEGIN
    UPDATE users SET updated = STRFTIME('%Y-%m-%dT%H:%M:%fZ') WHERE id = old.id;
END;

CREATE TABLE credentials
(
    id                          BLOB PRIMARY KEY CHECK (length(id) < 256),
    public_key                  BLOB    NOT NULL CHECK (length(public_key) < 256),
    attestation_type            TEXT    NOT NULL CHECK (length(attestation_type) < 256),
    transport                   TEXT    NOT NULL CHECK (length(transport) < 256),
    flag_user_present           INTEGER NOT NULL CHECK (flag_user_present IN (0, 1)),
    flag_user_verified          INTEGER NOT NULL CHECK (flag_user_verified IN (0, 1)),
    flag_backup_eligible        INTEGER NOT NULL CHECK (flag_backup_eligible IN (0, 1)),
    flag_backup_state           INTEGER NOT NULL CHECK (flag_backup_state IN (0, 1)),
    authenticator_aaguid        BLOB    NOT NULL CHECK (length(authenticator_aaguid) < 256),
    authenticator_sign_count    INTEGER NOT NULL,
    authenticator_clone_warning INTEGER NOT NULL CHECK (authenticator_clone_warning IN (0, 1)),
    authenticator_attachment    TEXT    NOT NULL CHECK (length(authenticator_attachment) < 256),

    created                     TEXT    NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ')) CHECK (length(created) < 256),
    updated                     TEXT    NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ')) CHECK (length(updated) < 256),

    user_id                     BLOB    NOT NULL REFERENCES users (id) ON DELETE CASCADE
) WITHOUT ROWID, STRICT;

CREATE TRIGGER credentials_updated_timestamp
    AFTER UPDATE
    ON credentials
BEGIN
    UPDATE credentials SET updated = STRFTIME('%Y-%m-%dT%H:%M:%fZ') WHERE id = old.id;
END;

CREATE TABLE cases
(
    id         TEXT PRIMARY KEY CHECK (length(id) < 256),
    name       TEXT NOT NULL UNIQUE CHECK (length(name) < 256),
    author     TEXT NOT NULL CHECK (length(author) < 256),
    image_path TEXT NOT NULL CHECK (length(image_path) < 256)
) WITHOUT ROWID, STRICT;

CREATE TABLE investigation_targets
(
    id         TEXT PRIMARY KEY CHECK (length(id) < 256),
    name       TEXT                                       NOT NULL UNIQUE CHECK (length(name) < 256),
    short_name TEXT                                       NOT NULL CHECK (length(short_name) < 256),
    type       TEXT CHECK ( type IN ('person', 'scene') ) NOT NULL CHECK (length(type) < 256),
    image_path TEXT                                       NOT NULL CHECK (length(image_path) < 256),

    case_id    TEXT                                       NOT NULL REFERENCES cases (id) ON DELETE CASCADE
) WITHOUT ROWID, STRICT;

CREATE TABLE clues
(
    id                      TEXT PRIMARY KEY CHECK (length(id) < 256),
    description             TEXT NOT NULL CHECK (length(description) < 1024),
    keywords                TEXT NOT NULL CHECK (length(keywords) < 256),

    investigation_target_id TEXT NOT NULL REFERENCES investigation_targets (id) ON DELETE CASCADE
) WITHOUT ROWID, STRICT;

CREATE TABLE IF NOT EXISTS completions
(
    id                      INTEGER PRIMARY KEY,
    "order"                 INTEGER NOT NULL,
    question                TEXT    NOT NULL CHECK (length(question) < 1024),
    answer                  TEXT    NOT NULL CHECK (length(answer) < 2056),

    user_id                 BLOB    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    investigation_target_id TEXT    NOT NULL REFERENCES investigation_targets (id) ON DELETE CASCADE,
    UNIQUE (user_id, investigation_target_id, "order")
) STRICT;

CREATE TABLE IF NOT EXISTS sessions
(
    token  TEXT PRIMARY KEY,
    data   BLOB NOT NULL,
    expiry REAL NOT NULL
) STRICT;

CREATE INDEX IF NOT EXISTS sessions_expiry_idx ON sessions (expiry);

CREATE TABLE IF NOT EXISTS users
(
    id           BLOB PRIMARY KEY,
    display_name TEXT NOT NULL,

    created      TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ')),
    updated      TEXT NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ'))
) STRICT;

CREATE TRIGGER IF NOT EXISTS users_updated_timestamp
    AFTER UPDATE
    ON users
BEGIN
    UPDATE users SET updated = STRFTIME('%Y-%m-%dT%H:%M:%fZ') WHERE id = old.id;
END;

CREATE TABLE IF NOT EXISTS credentials
(
    id                          BLOB PRIMARY KEY,
    public_key                  BLOB    NOT NULL,
    attestation_type            TEXT    NOT NULL,
    transport                   TEXT    NOT NULL,
    flag_user_present           INTEGER NOT NULL,
    flag_user_verified          INTEGER NOT NULL,
    flag_backup_eligible        INTEGER NOT NULL,
    flag_backup_state           INTEGER NOT NULL,
    authenticator_aaguid        BLOB    NOT NULL,
    authenticator_sign_count    INTEGER NOT NULL,
    authenticator_clone_warning INTEGER NOT NULL,
    authenticator_attachment    TEXT    NOT NULL,

    created                     TEXT    NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ')),
    updated                     TEXT    NOT NULL DEFAULT (STRFTIME('%Y-%m-%dT%H:%M:%fZ')),

    user_id                     BLOB    NOT NULL REFERENCES users (id) ON DELETE CASCADE
) STRICT;

CREATE TRIGGER IF NOT EXISTS credentials_updated_timestamp
    AFTER UPDATE
    ON credentials
BEGIN
    UPDATE credentials SET updated = STRFTIME('%Y-%m-%dT%H:%M:%fZ') WHERE id = old.id;
END;

CREATE TABLE IF NOT EXISTS cases
(
    id         TEXT PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    author     TEXT NOT NULL,
    image_path TEXT NOT NULL
) STRICT;

INSERT INTO cases(id, name, author, image_path)
VALUES ('rue-morgue', 'The Murders in the Rue Morgue', 'Edgar Allan Poe', '/images/rue_morgue.webp')
ON CONFLICT(id) DO UPDATE SET name       = excluded.name,
                              author     = excluded.author,
                              image_path = excluded.image_path;

CREATE TABLE IF NOT EXISTS investigation_targets
(
    id         TEXT PRIMARY KEY,
    name       TEXT                                       NOT NULL UNIQUE,
    short_name TEXT                                       NOT NULL,
    type       TEXT CHECK ( type IN ('person', 'scene') ) NOT NULL,
    image_path TEXT                                       NOT NULL,

    case_id    TEXT                                       NOT NULL REFERENCES cases (id) ON DELETE CASCADE
) STRICT;

INSERT INTO investigation_targets(id, name, short_name, type, image_path, case_id)
VALUES ('le-bon', 'Adolphe Le Bon', 'Adolphe', 'person', 'https://myrjola.twic.pics/sheerluck/adolphe_le-bon.webp',
        'rue-morgue'),
       ('rue-morgue', 'Rue Morgue Murder Scene', 'Rue Morgue', 'scene',
        'https://myrjola.twic.pics/sheerluck/rue-morgue.webp', 'rue-morgue')
ON CONFLICT (id) DO UPDATE SET name       = excluded.name,
                               short_name = excluded.short_name,
                               case_id    = excluded.case_id,
                               image_path = excluded.image_path;

CREATE TABLE IF NOT EXISTS clues
(
    id                      TEXT PRIMARY KEY,
    description             TEXT NOT NULL,
    keywords                TEXT NOT NULL,

    investigation_target_id TEXT NOT NULL REFERENCES investigation_targets (id) ON DELETE CASCADE
) STRICT;

INSERT INTO clues(id, description, keywords, investigation_target_id)
VALUES ('le-bon-victim-belongings',
        'The victims'' belongings in Adolphe''s posession were given to him as collateral for a debt.',
        'gold,watch,scissors', 'le-bon'),
       ('le-bon-last-meeting-with-the-victim',
        'Adolphe met the victims the day before the murder when he loaned them 4000 francs. Madame and Mademoiselle L''Espanaye relieved him of the money plaed in two bags. He then bowed and departed. Nobody else was seen during this interaction since it happened on a quiet street.',
        'victims,last-seen,loan', 'le-bon')
ON CONFLICT (id) DO UPDATE SET description             = excluded.description,
                               keywords                = excluded.keywords,
                               investigation_target_id = excluded.investigation_target_id;

CREATE TABLE IF NOT EXISTS completions
(
    id                      INTEGER PRIMARY KEY,
    "order"                 INTEGER NOT NULL,
    question                TEXT    NOT NULL,
    answer                  TEXT    NOT NULL,

    user_id                 BLOB    NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    investigation_target_id TEXT    NOT NULL REFERENCES investigation_targets (id) ON DELETE CASCADE,
    UNIQUE (user_id, investigation_target_id, "order") ON CONFLICT REPLACE
) STRICT;

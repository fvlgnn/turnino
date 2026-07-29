CREATE TABLE queues (
    id                 INTEGER PRIMARY KEY,
    name               TEXT NOT NULL UNIQUE,
    numbering_mode     TEXT NOT NULL DEFAULT 'fixed',
    prefix             TEXT NOT NULL DEFAULT '',
    alphabetic_start   TEXT NOT NULL DEFAULT '',
    alphabetic_end     TEXT NOT NULL DEFAULT '',
    digits             INTEGER NOT NULL,
    min_value          INTEGER NOT NULL,
    max_value          INTEGER NOT NULL,
    current_position   INTEGER NOT NULL DEFAULT -1,
    allow_custom_calls INTEGER NOT NULL DEFAULT 0,
    created_at         INTEGER NOT NULL DEFAULT (unixepoch()),
    CHECK (numbering_mode IN ('fixed', 'progressive')),
    CHECK (digits BETWEEN 1 AND 6),
    CHECK (min_value >= 0),
    CHECK (max_value >= min_value),
    CHECK (current_position >= -1),
    CHECK (allow_custom_calls IN (0, 1)),
    CHECK (
        (numbering_mode = 'fixed'
            AND length(alphabetic_start) = 0
            AND length(alphabetic_end) = 0)
        OR
        (numbering_mode = 'progressive'
            AND length(prefix) = 0
            AND length(alphabetic_start) BETWEEN 1 AND 2
            AND length(alphabetic_end) = length(alphabetic_start))
    )
);

CREATE TABLE counters (
    id         INTEGER PRIMARY KEY,
    name       TEXT NOT NULL UNIQUE,
    token_hash BLOB NOT NULL UNIQUE,
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    CHECK (length(token_hash) = 32),
    CHECK (enabled IN (0, 1))
);

CREATE TABLE calls (
    id         INTEGER PRIMARY KEY,
    queue_id   INTEGER NOT NULL REFERENCES queues(id) ON DELETE RESTRICT,
    counter_id INTEGER NOT NULL REFERENCES counters(id) ON DELETE RESTRICT,
    number     TEXT NOT NULL,
    position   INTEGER NOT NULL,
    kind       TEXT NOT NULL,
    called_at  INTEGER NOT NULL DEFAULT (unixepoch()),
    CHECK (position >= 0),
    CHECK (kind IN ('automatic', 'previous', 'manual', 'recall'))
);

CREATE INDEX calls_called_at_idx ON calls(called_at DESC, id DESC);
CREATE INDEX calls_queue_id_idx ON calls(queue_id, id DESC);
CREATE INDEX calls_counter_id_idx ON calls(counter_id, id DESC);

CREATE TABLE admin_credentials (
    id         INTEGER PRIMARY KEY CHECK (id = 1),
    token_hash BLOB NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    CHECK (length(token_hash) = 32)
);

CREATE TABLE settings (
    id                         INTEGER PRIMARY KEY CHECK (id = 1),
    installation_name          TEXT NOT NULL,
    default_language           TEXT NOT NULL,
    recent_calls_limit         INTEGER NOT NULL,
    sound_enabled              INTEGER NOT NULL,
    highlight_duration_seconds INTEGER NOT NULL,
    CHECK (length(trim(installation_name)) BETWEEN 1 AND 100),
    CHECK (default_language IN ('en', 'it')),
    CHECK (recent_calls_limit BETWEEN 1 AND 100),
    CHECK (sound_enabled IN (0, 1)),
    CHECK (highlight_duration_seconds BETWEEN 1 AND 30)
);

INSERT INTO settings (
    id,
    installation_name,
    default_language,
    recent_calls_limit,
    sound_enabled,
    highlight_duration_seconds
) VALUES (1, 'Turnino', 'en', 10, 1, 5);

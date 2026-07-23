CREATE TABLE queues (
    id            INTEGER PRIMARY KEY,
    name          TEXT NOT NULL UNIQUE,
    prefix        TEXT NOT NULL DEFAULT '',
    digits        INTEGER NOT NULL,
    min_value     INTEGER NOT NULL,
    max_value     INTEGER NOT NULL,
    current_value INTEGER NOT NULL,
    created_at    INTEGER NOT NULL DEFAULT (unixepoch()),
    CHECK (digits BETWEEN 1 AND 6),
    CHECK (min_value >= 0),
    CHECK (max_value >= min_value),
    CHECK (current_value BETWEEN min_value - 1 AND max_value)
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
    id             INTEGER PRIMARY KEY,
    queue_id       INTEGER NOT NULL REFERENCES queues(id) ON DELETE RESTRICT,
    counter_id     INTEGER NOT NULL REFERENCES counters(id) ON DELETE RESTRICT,
    number         TEXT NOT NULL,
    sequence_value INTEGER NOT NULL,
    kind           TEXT NOT NULL,
    called_at      INTEGER NOT NULL DEFAULT (unixepoch()),
    CHECK (kind IN ('automatic', 'manual', 'recall'))
);

CREATE INDEX calls_called_at_idx ON calls(called_at DESC, id DESC);
CREATE INDEX calls_queue_id_idx ON calls(queue_id, id DESC);
CREATE INDEX calls_counter_id_idx ON calls(counter_id, id DESC);

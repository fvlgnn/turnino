CREATE TABLE admin_credentials (
    id         INTEGER PRIMARY KEY CHECK (id = 1),
    token_hash BLOB NOT NULL,
    created_at INTEGER NOT NULL DEFAULT (unixepoch()),
    CHECK (length(token_hash) = 32)
);

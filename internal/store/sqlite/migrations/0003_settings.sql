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

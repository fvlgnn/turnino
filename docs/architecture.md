# Architecture

## Goals

Turnino is a self-hosted queue calling and display system, not a distributed appointment platform. A normal installation should need one binary, one writable data directory, and browsers on the local network.

## Runtime view

```text
operator browser -- POST call --> Go HTTP server -- transaction --> SQLite
                                      |
                                      +-- SSE --> public monitor
                                      +-- SSE --> counter display
```

Server-rendered HTML provides the initial state. SSE carries one-way live updates; ordinary HTTP POST requests carry commands. Polling may be retained only as a compatibility fallback.

## Planned domain

- **Queue** owns a numbering policy and its current sequence.
- **Counter** identifies a service desk or operator terminal.
- **Call** records a number shown at a counter and whether it was automatic, manual, or recalled.
- **Settings** stores installation-wide display and localization preferences.

Number allocation and call creation must occur in one database transaction. A counter's online presence is ephemeral, while its identity and call history are persistent.

## Technology decisions

- Go standard library first; add small dependencies only for a clear benefit.
- SQLite with WAL mode, a busy timeout and foreign keys enabled.
- The pure-Go `modernc.org/sqlite` driver keeps cross-compiled binaries free from CGO requirements.
- `html/template`, embedded templates and static assets.
- Embedded JSON translation dictionaries, initially `en` and `it`, with English as the fallback language.
- Vanilla JavaScript and modern CSS; no SPA toolchain initially.
- Structured logs to standard output.
- Configuration through environment variables and command-line flags.

## Deployment

The same binary should run natively and in a container. Releases target Linux AMD64 and ARM64 first, with convenience builds for macOS and Windows. Persistent state lives under `/data` in the container.

## Persistence and migrations

The application opens `<data-dir>/turnino.db` during startup and applies embedded, ordered SQL migrations before accepting HTTP traffic. Schema changes are recorded in `schema_migrations`; there is no separate migration command or ORM.

Advancing a queue and recording its call happen in one SQLite transaction. If the queue does not exist or the selected counter is unavailable, the number change is rolled back.

## Authentication

Turnino has one installation administrator and does not implement users or roles. On the first startup, `TURNINO_ADMIN_TOKEN` bootstraps a write-once credential whose SHA-256 hash is persisted; later environment values cannot silently replace it. Administrative endpoints live under `/api/v1/admin`.

Each counter receives a separate random token. Counter tokens can call queues but cannot administer the installation. Tokens are returned only when created or rotated, and only their SHA-256 hashes are stored.

Bearer tokens must be used only on a trusted local network or behind HTTPS. Turnino does not attempt to make plaintext HTTP safe on an untrusted network.

The administration UI exchanges the administrator token for a random, twelve-hour in-memory session. Its cookie is `HttpOnly` and `SameSite=Strict`; state-changing forms also require a session-specific CSRF token. Sessions intentionally disappear when Turnino restarts.

If the administrator token is lost, the server must be stopped and the explicit `turnino admin reset --confirm` command run against its data directory. There are no recovery questions, default passwords, or network reset endpoints.

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
- SQLite with WAL mode and foreign keys enabled.
- A pure-Go SQLite driver is preferred so cross-compiled binaries remain straightforward.
- `html/template`, embedded templates and static assets.
- Embedded JSON translation dictionaries, initially `en` and `it`, with English as the fallback language.
- Vanilla JavaScript and modern CSS; no SPA toolchain initially.
- Structured logs to standard output.
- Configuration through environment variables and command-line flags.

## Deployment

The same binary should run natively and in a container. Releases target Linux AMD64 and ARM64 first, with convenience builds for macOS and Windows. Persistent state lives under `/data` in the container.

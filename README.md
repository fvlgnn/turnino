# Turnino

> **Your turn, made simple.**

Tiny self-hosted queue calling and display system. Built with Go, SQLite and Server-Sent Events; designed for Raspberry Pi and Docker.

Turnino is the modern rewrite of a queue display system that has run for years on modest hardware, including Raspberry Pi. It keeps the original idea: one server, a browser for each operator, a public monitor, optional counter displays, and a small administration page.

The project is in active alpha development. The HTTP foundation, delivery pipeline and SQLite persistence layer are present; the operator and display workflows are being built toward v0.1.0.

## Principles

Turnino follows the KISS principle: one process, one database, a few browser views, and no unnecessary infrastructure.

- Keep installation and operation simple.
- Run well on low-cost ARM hardware.
- Use one process and one SQLite database.
- Prefer server-rendered HTML and small amounts of JavaScript.
- Use Server-Sent Events for live display updates.
- Add features only when they preserve the core workflow.

## Quick start

Go 1.25 or newer is required.

```sh
make dev
```

The public server starts without configuration. To bootstrap the administration API on a new database, set a random `TURNINO_ADMIN_TOKEN` of at least 20 characters for the first startup; Turnino persists only its SHA-256 hash. See `.env.example` and the development guide for configuration details.

Open <http://localhost:8080>. The health endpoint is available at <http://localhost:8080/healthz>.
The administration interface is available at <http://localhost:8080/admin> after the administrator credential has been bootstrapped.
The operator interface is available at <http://localhost:8080/caller>. Paste the token issued when its counter was registered; the token remains in that browser and authenticates calls to the existing API.
The dedicated counter display is available at <http://localhost:8080/counter-display>. Open it from the Caller to reuse the counter token saved in the same browser; it then shows only calls made by that counter.
The autonomous live public monitor is available at <http://localhost:8080/monitor>. During development, open `/monitor?test=1` to expose sound/fullscreen controls and launch a Caller tab from the same browser. See the [kiosk deployment guide](docs/kiosk.md) for Windows Edge and Raspberry Pi Chromium autostart.

With Docker:

```sh
docker compose up --build
```

Runtime data, including the automatically migrated `turnino.db` database, is written to `./data` for native development and to the `turnino-data` volume with Compose.

Maintenance commands create new files and never overwrite an existing destination:

```sh
./bin/turnino backup --data-dir ./data --output ./backups/turnino.db
./bin/turnino export calls --data-dir ./data --output ./exports/calls.csv
./bin/turnino demo create --output ./data/turnino-demo.db
```

Backups are consistent SQLite snapshots and may be created while Turnino is running. The demonstration database contains only synthetic records and discarded random credentials.

## Why Turnino?

Turnino combines the English word “turn” with the Italian diminutive suffix “-ino”. It describes a small, friendly system that tells people when it is their turn. The name also continues a personal tradition started with Gianino, the author's home automation project.

## Project status

See the [roadmap](docs/roadmap.md), [architecture](docs/architecture.md), [development guide](docs/development.md), [kiosk deployment guide](docs/kiosk.md), [security model](docs/security.md), and [troubleshooting guide](docs/troubleshooting.md). Italian project notes are available in [docs/it/README.md](docs/it/README.md).

## License

Copyright (c) 2026-present Gianni Favilli (fvlgnn).

Turnino is free software licensed under the [GNU Affero General Public License v3.0](LICENSE). Modified versions made available over a network must offer their corresponding source to their users. See [docs/licensing.md](docs/licensing.md) for the practical rationale.

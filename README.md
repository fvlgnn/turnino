# Turnino

> **Your turn, made simple.**

A lightweight, open-source, self-hosted queue calling and display system. Built with Go, SQLite and Server-Sent Events. Designed for Raspberry Pi and Docker.

Turnino is the modern rewrite of a queue display system that has run for years on modest hardware, including Raspberry Pi. It keeps the original idea: one server, a browser for each operator, a public monitor, optional counter displays, and a small administration page.

The project is in active alpha development. Version 0.1.0 established the operator, administration and live-display workflows; progressive queue-numbering policies are now being consolidated for the next release.

## Principles

Turnino follows the KISS principle: one process, one database, a few browser views, and no unnecessary infrastructure.

- Keep installation and operation simple.
- Run well on low-cost ARM hardware.
- Use one process and one SQLite database.
- Prefer server-rendered HTML and small amounts of JavaScript.
- Use Server-Sent Events for live display updates.
- Add features only when they preserve the core workflow.

## Queue numbering

Turnino supports several queue and ticket-number formats:

- numeric queues such as `00-99` and `000-999`;
- fixed service prefixes such as `A00-A99`;
- progressive physical ticket rolls such as `A00-E99`, `A00-J99`, `A00-Z99` and `AA00-ZZ99`;
- custom alphabetic and numeric ranges, with validated manual-number calls when enabled.

Every policy wraps at its configured boundary and supports next, previous, recall and reset operations. See the [queue-numbering guide](docs/numbering.md) for configuration rules, examples and behavior.

## Quick start

Go 1.25 or newer is required. Initialize administrator access once, then start Turnino:

```sh
make build
./bin/turnino admin init --data-dir ./data
make dev
```

Save the generated access key: it is shown only once. See the [administration guide](docs/administration.md) for Docker setup, key rotation, offline recovery, advanced automation and HTTPS guidance. Data-directory selection and `.env` behavior are covered in the [configuration guide](docs/configuration.md).

Open <http://localhost:8080>. The health endpoint is available at <http://localhost:8080/healthz>.
Project, license and source information is available at <http://localhost:8080/about>.
The administration interface is available at <http://localhost:8080/admin> after administrator access has been initialized.
The operator interface is available at <http://localhost:8080/caller>. Paste the token issued when its counter was registered; the token remains in that browser and authenticates automatic calls and recalls. Queues backed by a separate manual paper dispenser can explicitly enable validated custom-number calls from the administration page.
The dedicated counter display is available at <http://localhost:8080/counter-display>. Open it from the Caller to reuse the counter token saved in the same browser; it then shows only calls made by that counter.
The autonomous live public monitor is available at <http://localhost:8080/monitor>. During development, open `/monitor?test=1` to expose sound/fullscreen controls and launch a Caller tab from the same browser. See the [kiosk deployment guide](docs/kiosk.md) for Windows Edge and Raspberry Pi Chromium autostart.

With Docker:

```sh
docker compose build
docker compose run --rm turnino admin init --data-dir /data
docker compose up -d
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

Turnino combines the English word "turn" with the Italian diminutive suffix "ino". It describes a small, friendly system that tells people when it is their turn. The name also continues the tradition I started with Gianino, my home automation project, and pays a small tribute to Arduino, the platform that sparked my passion for programming, the maker mindset and the open-source philosophy.

## Project status

See the [roadmap](docs/roadmap.md), [architecture](docs/architecture.md), [queue-numbering guide](docs/numbering.md), [development guide](docs/development.md), [configuration guide](docs/configuration.md), [administration guide](docs/administration.md), [kiosk deployment guide](docs/kiosk.md), [security model](docs/security.md), and [troubleshooting guide](docs/troubleshooting.md). Italian project notes are available in [docs/it/README.md](docs/it/README.md).

## License

Copyright (c) 2026-present Gianni Favilli (fvlgnn).

Turnino is free software licensed under the [GNU Affero General Public License v3.0](LICENSE). Modified versions made available over a network must offer their corresponding source to their users. See [docs/licensing.md](docs/licensing.md) for the practical rationale.

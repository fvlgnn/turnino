# Changelog

All notable changes to Turnino will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial Go HTTP server with embedded HTML and static assets.
- Health endpoint, structured logging and graceful shutdown.
- Docker and Compose development environment.
- CI and multi-platform release workflows.
- Initial architecture, roadmap and development documentation.
- GNU Affero General Public License v3.0.
- SQLite persistence using a pure-Go driver, WAL mode and embedded automatic migrations.
- Queue and counter domain models with configurable queue prefixes, ranges and zero padding.
- Atomic next-number allocation and call recording, including range wraparound.
- Database-aware health checks and concurrency tests.
- JSON endpoints and REST Client examples for queue creation, counter registration and authenticated next-number calls.
- Revocable, rotatable and safely reactivatable counter credentials, stored only as SHA-256 hashes.
- Write-once administrator bootstrap and Bearer-protected administration endpoints without users or roles.
- Server-rendered administration UI with in-memory sessions, CSRF protection and queue/counter management.
- Explicit offline administrator reset command and SQLite/WSL troubleshooting guidance.
- Embedded English and Italian JSON dictionaries, template translation helpers and browser language persistence.
- Consistent SQLite backup, call-history CSV export and reproducible synthetic demonstration database commands.
- Typed installation settings with an administration form, persisted defaults and validation.
- Server-rendered Caller interface for authenticated automatic calls using a counter token stored in the operator browser.
- Full-screen public monitor with recent calls, SSE snapshot recovery, live visual emphasis, optional browser sound and a same-browser test mode.
- Autonomous production monitor mode and supervised kiosk launchers for Microsoft Edge on Windows and Chromium on Raspberry Pi.
- Kiosk language, scale and restart configuration plus a documented security and troubleshooting boundary.
- Dedicated SSE counter display with automatic same-browser discovery, administration pairing links and Windows/Raspberry Pi kiosk launchers.

### Fixed

- Windows kiosk launcher now preserves a single Edge executable result as an array instead of treating its first character as the executable path.
- Counter display waiting and active-call states no longer render at the same time.

## [0.1.0] - 2026-07-24

### Added

- TODO

[Unreleased]: https://github.com/fvlgnn/turnino/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/fvlgnn/turnino/releases/tag/v0.1.0

# Changelog

All notable changes to Turnino will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- SQLite persistence using a pure-Go driver, WAL mode and embedded automatic migrations.
- Queue and counter domain models with configurable queue prefixes, ranges and zero padding.
- Atomic next-number allocation and call recording, including range wraparound.
- Database-aware health checks and concurrency tests.
- JSON endpoints and REST Client examples for queue creation, counter registration and authenticated next-number calls.
- Revocable, rotatable and safely reactivatable counter credentials, stored only as SHA-256 hashes.
- Write-once administrator bootstrap and Bearer-protected administration endpoints without users or roles.
- Server-rendered administration UI with in-memory sessions, CSRF protection and queue/counter management.
- Explicit offline administrator reset command and SQLite/WSL troubleshooting guidance.

## [0.1.0] - 2026-07-24

### Added

- Initial Go HTTP server with embedded HTML and static assets.
- Health endpoint, structured logging and graceful shutdown.
- Docker and Compose development environment.
- CI and multi-platform release workflows.
- Initial architecture, roadmap and development documentation.
- GNU Affero General Public License v3.0.

[Unreleased]: https://github.com/fvlgnn/turnino/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/fvlgnn/turnino/releases/tag/v0.1.0

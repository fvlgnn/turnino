# Changelog

All notable changes to Turnino will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]


## [0.2.0] - 2026-07-30

### Added

- Progressive alphanumeric queue policies with one or two letter ranges, configurable numeric ranges and automatic wrap.
- Common numbering presets and custom policy editing in the administration interface.
- Previous-number calls from the operator interface and HTTP API.
- Confirmed queue reset from administration and authenticated operator workstations without deleting call history.

### Changed

- Compatible queue-policy changes preserve the current number; incompatible changes require an explicit reset confirmation.
- **Breaking:** custom-call requests now accept the complete formatted `number` instead of `sequence_value`.
- **Breaking:** the alpha SQLite schema is consolidated into one initial migration; databases created by version 0.1.0 must be recreated.


## [0.1.0] - 2026-07-29

### Added

- Self-hosted queue calling system distributed as a single Go binary and Docker image.
- SQLite persistence with automatic migrations, atomic queue advancement and configurable numbering.
- Server-rendered administration interface for settings, queues, counters and access-key management.
- Operator interface with automatic, manual and repeat calls.
- Live public monitor and dedicated counter displays using Server-Sent Events.
- English and Italian interface translations.
- Windows Edge and Raspberry Pi Chromium kiosk launchers.
- Administrator initialization, access-key rotation and offline recovery.
- SQLite backup, call-history CSV export and synthetic demonstration data.
- Multi-platform CI and release automation for AMD64 and ARM64.
- Public source, security, deployment and administration documentation.
- GNU Affero General Public License v3.0.

### Known limitations

- Turnino is alpha software and is intended for trusted local networks or deployment behind properly configured HTTPS infrastructure.
- Administration uses one installation-wide access key. Users and roles are not currently supported.


[Unreleased]: https://github.com/fvlgnn/turnino/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/fvlgnn/turnino/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/fvlgnn/turnino/releases/tag/v0.1.0

# Roadmap

## Bootstrap (completed)

- Repository conventions and documentation
- Minimal HTTP server, embedded UI, health check and graceful shutdown
- Native and container development paths
- CI and tagged multi-architecture releases
- Final product-name decision

## v0.1.0 — Core queue calling (current)

- [x] SQLite schema and automatic migrations
- [x] Queue domain and configurable numbering policies
- [x] Atomic next-number allocation and call recording
- [x] Counter registration with revocable and rotatable browser tokens
- Automatic, manual and repeat calls
- Operator, public monitor and counter display pages
- SSE updates with reconnect and a conservative polling fallback
- [x] Essential administration with browser login, protected forms and offline credential recovery
- [x] English and Italian UI using embedded JSON dictionaries, with English as the fallback language
- Backup/export and sanitized demonstration data

## v0.2 — Adaptation and polish

- Configurable themes, logo and banner
- Daily or manual numbering reset
- CSV call export and database backup
- Optional browser speech synthesis
- Installable operator PWA
- Accessibility and kiosk-mode documentation

## Later, only with validated demand

- Ticket issuing and QR codes
- Priority and transfer workflows
- Multiple services per counter
- Advanced analytics
- PostgreSQL
- External authentication

The project deliberately avoids committing to appointments, cloud services, microservices, Redis, or a SPA frontend.

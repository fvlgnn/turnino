# Roadmap

## Phase 0 — Bootstrap (completed)

- Repository conventions and documentation
- Minimal HTTP server, embedded UI, health check and graceful shutdown
- Native and container development paths
- CI and tagged multi-architecture releases
- Final product-name decision

## Phase 1 — Core queue calling (current)

- [x] SQLite schema and automatic migrations
- [x] Queue domain and configurable numbering policies
- [x] Installation-wide settings for display and localization preferences
- [x] Atomic next-number allocation and call recording
- [x] Counter registration with revocable and rotatable browser tokens
- [x] Automatic next-number call in the domain, persistence layer and HTTP API
- [x] Operator/caller page for automatic calls
- [ ] Validated custom-number calls from the operator page
- [ ] Repeat the last call without advancing the queue
- [x] Counter page for the operator and dedicated counter display
- [x] Full-screen public monitor with recent calls, visual emphasis and an audible cue
- [x] SSE updates with automatic browser reconnect and snapshot recovery
- [ ] Conservative polling fallback for browsers without SSE support
- [x] Essential administration with browser login, protected forms and offline credential recovery
- [x] English and Italian UI using embedded JSON dictionaries, with English as the fallback language
- [x] Consistent SQLite backup and call-history export
- [x] Reproducible demonstration database containing only synthetic data and unusable credentials

## Phase 2 — Adaptation and polish

- Progressive alphanumeric numbering policies for physical ticket dispensers (for example `A00–Z99`)
- Configurable themes, logo and banner
- Daily or manual numbering reset
- Backup download from the administration UI, retention policy and filtered exports
- Optional browser speech synthesis
- Installable operator PWA
- Accessibility and kiosk-mode documentation

## Future phases, only with validated demand

- Setting (themes, logo, banner, volumes, etc.)
- Ticket issuing and QR codes
- Priority and transfer workflows
- Multiple services per counter
- Advanced analytics
- PostgreSQL
- External authentication

The project deliberately avoids committing to appointments, cloud services, microservices, Redis, or a SPA frontend.

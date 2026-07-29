# Roadmap

## Phase 0 - Bootstrap (completed)

- Repository conventions and documentation
- Minimal HTTP server, embedded UI, health check and graceful shutdown
- Native and container development paths
- CI and tagged multi-architecture release workflows
- Final product-name decision

## Phase 1 - Core queue calling (completed)

- [x] SQLite schema and automatic migrations
- [x] Queue domain and configurable numbering policies
- [x] Installation-wide settings for display and localization preferences
- [x] Atomic next-number allocation and call recording
- [x] Counter registration with revocable and rotatable browser tokens
- [x] Automatic next-number call in the domain, persistence layer and HTTP API
- [x] Operator/caller page for automatic calls
- [x] Validated custom-number calls for queues explicitly configured with a manual paper dispenser
- [x] Repeat the last call without advancing the queue
- [x] Counter page for the operator and dedicated counter display
- [x] Full-screen public monitor with recent calls, visual emphasis and an audible cue
- [x] SSE updates with automatic browser reconnect and snapshot recovery
- [x] Essential administration with browser login, protected forms and offline credential recovery
- [x] English and Italian UI using embedded JSON dictionaries, with English as the fallback language
- [x] Consistent SQLite backup and call-history export
- [x] Reproducible demonstration database containing only synthetic data and unusable credentials

## Phase 2 - Progressive numbering (completed)

- [x] Preserve fixed-prefix numeric queues as the simple, backward-compatible mode
- [x] Add progressive alphabetic components for physical ticket dispensers, such as `A00-Z99` and `AA00-ZZ99`
- [x] Offer presets for common physical rolls and advanced custom ranges
- [x] Allow queue policies to be corrected without recreating the queue
- [x] Wrap to the beginning at the end of the configured range
- [x] Use human-readable formatted numbers for custom calls
- [x] Call the previous number while moving the queue backwards
- [x] Provide an explicit operator reset without deleting call history
- [x] Preserve atomic allocation and stable materialized call history

## Phase 3 - Presentation and installation identity

- Predefined color themes
- Configurable text banner
- Detailed call-sound volume
- Validated installation logo with defined persistence and backup behavior

## Phase 4 - Optional operator client

- Keep the browser Caller as the default operator interface
- Explore a minimal desktop client after stabilizing the Caller API
- Prefer a WebView wrapper for the shortest implementation path
- Consider Fyne as a Go learning exercise when the additional GUI and packaging maintenance are justified

## Future phases, only with validated demand

- Additional display and audio settings
- Explicit call-history retention, export and cleanup tools
- Ticket issuing and QR codes
- Hardware ticket dispensers and printer adapters, only after validating specific devices
- Priority and transfer workflows
- Multiple services per counter
- Advanced analytics
- PostgreSQL
- External authentication

The project deliberately avoids committing to appointments, cloud services, microservices, Redis, or a SPA frontend.

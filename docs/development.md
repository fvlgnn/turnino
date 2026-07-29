# Development guide

## Requirements

- Go 1.25 or newer (CI resolves the current compatible toolchain)
- GNU Make or compatible `make`
- Docker with Compose for container testing

## Commands

```sh
make dev          # run the development server
make build        # build bin/turnino
make check        # formatting, vet and race-enabled tests
make docker-build # build the local container
docker compose up --build
```

Configuration is documented in `.env.example` and `docs/configuration.md`. Go does not load `.env` itself; export the variables in your shell or let an editor/task runner load them. For example, use `TURNINO_DATA_DIR=./data-dev make dev` for an isolated development database.

Use `/monitor?test=1` for browser-side monitor development. The production `/monitor` deliberately hides interactive controls; supervised Windows and Raspberry Pi kiosk launchers are documented in `docs/kiosk.md`.

For a new database, run `make build` and then `./bin/turnino admin init --data-dir ./data` while the server is stopped. Save the access key shown once. `TURNINO_ADMIN_TOKEN` is retained as an advanced automated-bootstrap option. See `docs/administration.md` for setup, rotation and recovery.

## Translations

Browser messages are stored in `internal/webui/translations/en.json` and `it.json`. Add the English key first, add its Italian equivalent, then reference it from a template with:

```gotemplate
{{t .Lang "section.message"}}
```

Keep keys flat and grouped by feature. Tests ensure that every English key has an Italian translation and that missing localized values fall back to English.

## Data maintenance commands

After `make build`, create a consistent SQLite snapshot with:

```sh
./bin/turnino backup --data-dir ./data --output ./backups/turnino-backup.db
```

The backup uses SQLite `VACUUM INTO`, includes committed WAL content and can run while Turnino is active. The destination must not already exist. Backup files contain installation settings, credential hashes and operational history, so store them as sensitive data.

Export call history as UTF-8 CSV:

```sh
./bin/turnino export calls --data-dir ./data --output ./exports/calls.csv
```

The CSV columns are `id`, `number`, `kind`, `queue_id`, `queue`, `counter_id`, `counter`, and UTC `called_at`.

Create a disposable demonstration database:

```sh
./bin/turnino demo create --output /tmp/turnino-demo.db
```

The generator creates two synthetic queues, two synthetic counters and eleven calls. Counter tokens are randomly generated and immediately discarded; no administrator credential is configured. Generate a fresh database when needed instead of committing a runtime `.db` file.

## Module path

The Go module is `github.com/fvlgnn/turnino`.

## Pull-request checks

CI verifies formatting, `go vet`, race-enabled tests, compilation, and a complete container build. Dependabot checks Go modules, GitHub Actions, and Docker base images quarterly.

## Release process

1. Ensure `main` is green and update release notes/documentation.
2. Create and push a semantic version tag such as `vX.Y.Z`.
3. GitHub Actions builds archives and creates a GitHub release.
4. The same workflow publishes `ghcr.io/fvlgnn/turnino` for AMD64 and ARM64.
5. Verify the published GitHub release, archives and multi-architecture GHCR image.

Do not create a public tag until the public tree has been checked for private data and credentials.

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

Configuration is documented in `.env.example`. Go does not load `.env` itself; export the variables in your shell or let an editor/task runner load them.

For a new database, generate a random administration token, set it as `TURNINO_ADMIN_TOKEN`, and start Turnino once. The credential is bootstrapped only when none exists and cannot be overwritten by changing the environment variable. Keep the original token outside the repository; administration requests use it as a Bearer token.

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
2. Create and push a semantic version tag such as `v0.1.0`.
3. GitHub Actions builds archives and creates a GitHub release.
4. The same workflow publishes `ghcr.io/fvlgnn/turnino` for AMD64 and ARM64.
5. Make the GHCR package public after its first publication if the repository settings do not do so automatically.

Do not create a public tag until the public tree has been checked for private data and credentials.

# Configuration

## Data directory

Turnino stores its SQLite database as `<data-directory>/turnino.db`. The data directory is selected in this order:

1. the `--data-dir` command-line option;
2. the exported `TURNINO_DATA_DIR` environment variable;
3. the default `./data`, relative to the process working directory.

Turnino does not load `.env` files automatically. Merely editing `.env` does not change a native or `go run` process. Export the values first:

```sh
set -a
. ./.env
set +a
make dev
```

Alternatively, set only the required variable for one command.

### Compiled binary

Using the command-line option:

```sh
./bin/turnino --data-dir ./data-dev
```

Using an environment variable:

```sh
TURNINO_DATA_DIR=./data-dev ./bin/turnino
```

The option takes precedence if both are present:

```sh
TURNINO_DATA_DIR=./data-from-env ./bin/turnino --data-dir ./data-from-option
```

This uses `./data-from-option`.

### Development with Make

The `make dev` target does not forward additional Turnino arguments. Pass the directory through the environment:

```sh
TURNINO_DATA_DIR=./data-dev make dev
```

Or export `.env` as shown above before running `make dev`.

### Development with `go run`

Pass the application option after the package path:

```sh
go run ./cmd/turnino --data-dir ./data-dev
```

The environment form works too:

```sh
TURNINO_DATA_DIR=./data-dev go run ./cmd/turnino
```

### Docker Compose

The standard Compose configuration uses `/data` inside the container and stores it in the named volume `turnino-data`. `./data` on the host and `/data` in the container are different paths.

To create a second independent Compose installation with its own named volume, use a different project name:

```sh
docker compose -p turnino-test run --rm turnino admin init --data-dir /data
docker compose -p turnino-test up --build
```

Compose creates a separate volume for the `turnino-test` project.

### Docker with a host directory

To see and manage the database in a specific host directory, use a bind mount. Build the image, create a writable directory and initialize it:

```sh
make docker-build
mkdir -p ./data-dev
docker run --rm \
  --user "$(id -u):$(id -g)" \
  --mount type=bind,src="$(pwd)/data-dev",dst=/data \
  turnino:dev admin init --data-dir /data
```

Then run Turnino with the same mount:

```sh
docker run --rm \
  --name turnino-test \
  --user "$(id -u):$(id -g)" \
  -p 8080:8080 \
  --mount type=bind,src="$(pwd)/data-dev",dst=/data \
  turnino:dev
```

The container always sees `/data`; the host sees `./data-dev`. Reuse exactly the same mount for initialization, normal startup, backup and offline reset.

## Alpha database compatibility

The `0.2.0` development cycle consolidated the SQLite schema into one clean initial migration. A database created by `0.1.0` is not compatible and must be recreated after exporting or backing up any data that must be retained.

For a native development database, stop Turnino and preserve all SQLite files together:

```sh
mkdir -p data/0.1.0-backup
mv data/turnino.db data/0.1.0-backup/
mv data/turnino.db-wal data/0.1.0-backup/ 2>/dev/null || true
mv data/turnino.db-shm data/0.1.0-backup/ 2>/dev/null || true
./bin/turnino admin init --data-dir ./data
```

For the standard Docker Compose installation, `docker compose down --volumes` removes the old named data volume. This is destructive, so export or back up required data with `0.1.0` first:

```sh
docker compose down --volumes
docker compose run --rm turnino admin init --data-dir /data
docker compose up -d
```

This compatibility exception is intentional while Turnino remains alpha. New databases use the consolidated schema directly; future compatibility rules will be documented before the schema is considered stable.

## Administrator bootstrap variable

The normal first setup is:

```sh
./bin/turnino admin init --data-dir ./data
```

This generates the administrator access key and stores only its hash. `TURNINO_ADMIN_TOKEN` is a separate, advanced bootstrap mechanism for automated deployments: when the database has no administrator key, the first application startup stores the hash of the supplied value.

The clear-text value must initially be retained because it is the key used to sign in. It does not need to remain in the server environment after bootstrap. Store it in a password manager and remove it from `.env` or the deployment environment. Later changes to `TURNINO_ADMIN_TOKEN` do not rotate an existing key.

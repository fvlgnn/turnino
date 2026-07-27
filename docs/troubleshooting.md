# Troubleshooting

## Resetting a lost administrator access key

The reset command intentionally works offline and replaces the existing administrator access key. It prints the new key once.

Native development:

```sh
# Stop Turnino first.
make build
./bin/turnino admin reset --data-dir ./data --confirm
```

Docker Compose:

```sh
docker compose stop turnino
docker compose run --rm turnino admin reset --data-dir /data --confirm
docker compose start turnino
```

Store the printed access key in a password manager and update the local, ignored `.env` used by REST Client. The command requires `--confirm` to reduce accidental resets. It must not be run while the server is active.

## Restoring a backup

Restoration is deliberately offline. Stop Turnino cleanly and preserve the current database instead of overwriting it:

```sh
mv data/turnino.db data/turnino-before-restore.db
cp backups/turnino-backup.db data/turnino.db
chmod 600 data/turnino.db
```

Start Turnino and verify `/healthz` and the administration UI. Keep the previous database until the restored installation has been checked. Do not restore while Turnino is running and do not mix a backup with old `-wal` or `-shm` files.

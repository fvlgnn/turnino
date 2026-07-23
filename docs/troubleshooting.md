# Troubleshooting

## Resetting a lost administrator credential

The reset command intentionally works offline and replaces the existing administrator credential. It prints the new token once.

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

Store the printed token in a password manager and update the local, ignored `.env` used by REST Client. The command requires `--confirm` to reduce accidental resets. It must not be run while the server is active.

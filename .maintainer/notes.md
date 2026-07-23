# Note del maintainer

Questo file è versionato come promemoria operativo del maintainer, ma non fa parte della documentazione utente. Non inserire password, token, dati personali o configurazioni riservate.

## Avvio rapido

```sh
make dev
```

Senza Go installato:

```sh
docker compose up --build
```

Applicazione: <http://localhost:8080>
Health check: <http://localhost:8080/healthz>

## Provare le API con REST Client

Al primo avvio copiare `.env.example` in `.env`, generare una credenziale amministrativa lunga e inserirla come `TURNINO_ADMIN_TOKEN`. Il file `.env` è escluso da Git. Per esportarne i valori e avviare Turnino:

```sh
openssl rand -base64 32
set -a
. ./.env
set +a
make dev
```

Turnino salva soltanto l'hash SHA-256 nel database. Conservare il valore originale in un password manager: servirà per amministrare l'installazione. Dopo il bootstrap la variabile può essere rimossa dall'ambiente del server, ma questo non sostituisce né recupera la credenziale già registrata.

Il file `api/turnino.rest` è compatibile con l'estensione VS Code `humao.rest-client` e legge `TURNINO_ADMIN_TOKEN` dal file `.env` mediante `{{$dotenv TURNINO_ADMIN_TOKEN}}`. Aprire il file ed eseguire nell'ordine:

1. `Create a queue`;
2. `Register a counter`;
3. `Call the next number using IDs and token from the responses above`.

REST Client recupera automaticamente dalle prime due risposte l'ID della coda e il token dello sportello. Il token viene mostrato in chiaro soltanto durante la registrazione; nel database ne viene conservato l'hash SHA-256.

Gli endpoint sotto `/api/v1/admin` richiedono il token amministrativo. L'endpoint `calls/next` richiede invece il token dello sportello restituito al momento della registrazione. Gli esempi successivi mostrano anche rotazione, revoca e riattivazione; una riattivazione genera sempre un nuovo token.

In alternativa, impostare manualmente `queueId` e `counterToken` nell'ultima sezione del file `.rest`. Non inserire credenziali direttamente nel file versionato.

Per provare il flusso dal browser, registrare uno sportello nell'amministrazione, copiare il token mostrato una sola volta, aprire <http://localhost:8080/caller> e salvarlo nel browser. Il Caller conserva il token in `localStorage`; usare “Dimentica token” su postazioni condivise o prima di dismettere il browser. Rotazione e revoca rendono immediatamente inutilizzabile il valore salvato.

## Comandi utili

```sh
make check
make build
make docker-build
docker build -t turnino:dev .
docker run --rm turnino:dev -version
```

## Recupero amministratore offline

Fermare completamente Turnino, quindi eseguire:

```sh
make build
./bin/turnino admin reset --data-dir ./data --confirm
```

Con Compose:

```sh
docker compose stop turnino
docker compose run --rm turnino admin reset --data-dir /data --confirm
docker compose start turnino
```

Il comando sostituisce la credenziale amministrativa e stampa il nuovo token una sola volta. Conservarlo e aggiornare il `.env` locale. Non eseguire il reset mentre Turnino è attivo.

## Backup, export e dati demo

```sh
make build
./bin/turnino backup --data-dir ./data --output ./backups/turnino-backup.db
./bin/turnino export calls --data-dir ./data --output ./exports/calls.csv
./bin/turnino demo create --output /tmp/turnino-demo.db
```

I comandi non sovrascrivono file esistenti. Il backup è uno snapshot SQLite consistente e può essere creato mentre Turnino è attivo. Backup e CSV sono dati sensibili; il solo database `demo`, generato con dati sintetici e credenziali casuali scartate, è adatto a esempi e screenshot.

Per ripristinare, fermare Turnino, spostare il database corrente con un nome di sicurezza, copiare il backup come `data/turnino.db` e riavviare. Non mescolare il backup con vecchi file `-wal` o `-shm`; vedere `docs/troubleshooting.md`.

## Ispezionare SQLite da Windows/WSL

Non aprire il database originale da HeidiSQL tramite `\\wsl.localhost\...`: WAL e locking attraversano due ambienti diversi. Fermare Turnino, verificare che non esistano processi o file `turnino.db-wal`/`turnino.db-shm`, quindi copiare `turnino.db` sotto `/mnt/c/...` e aprire la copia dal percorso Windows. Non copiare mai la copia sopra il database originale e non eliminare manualmente i file WAL. Vedere `docs/troubleshooting.md`.

## Comandi SQLite utili

Per ispezionare il database reale senza modificarlo:

```sh
sqlite3 -readonly data/turnino.db
```

Se il database è sicuramente fermo, non esistono file `-wal`/`-shm` e l'apertura read-only non riesce perché SQLite tenta comunque di creare file ausiliari, usare la modalità immutabile:

```sh
sqlite3 'file:data/turnino.db?mode=ro&immutable=1'
```

Non usare `immutable=1` mentre Turnino è attivo: ignora locking e WAL e potrebbe mostrare dati incompleti o incoerenti.

Per lavorare in scrittura, fermare prima Turnino e creare una copia di sicurezza:

```sh
cp data/turnino.db /tmp/turnino-before-edit.db
sqlite3 data/turnino.db
```

I comandi che iniziano con `.` appartengono alla shell SQLite e non richiedono il punto e virgola:

```text
.help
.databases
.tables
.schema
.schema queues
.indexes
.headers on
.mode column
.timeout 5000
.timer on
.quit
```

Controlli sul database:

```sql
PRAGMA journal_mode;
PRAGMA foreign_keys;
PRAGMA integrity_check;
PRAGMA foreign_key_check;
```

Elenco delle migrazioni applicate:

```sql
SELECT version,
       datetime(applied_at, 'unixepoch', 'localtime') AS applied_at
FROM schema_migrations
ORDER BY version;
```

Code e stato della numerazione:

```sql
SELECT id, name, prefix, digits,
       min_value, max_value, current_value,
       datetime(created_at, 'unixepoch', 'localtime') AS created_at
FROM queues
ORDER BY name;
```

Sportelli, senza mostrare l'hash completo del token:

```sql
SELECT id, name, enabled,
       length(token_hash) AS token_hash_bytes,
       datetime(created_at, 'unixepoch', 'localtime') AS created_at
FROM counters
ORDER BY name;
```

Ultime chiamate con coda e sportello:

```sql
SELECT calls.id,
       calls.number,
       calls.kind,
       queues.name AS queue,
       counters.name AS counter,
       datetime(calls.called_at, 'unixepoch', 'localtime') AS called_at
FROM calls
JOIN queues ON queues.id = calls.queue_id
JOIN counters ON counters.id = calls.counter_id
ORDER BY calls.id DESC
LIMIT 20;
```

Conteggio delle chiamate per coda:

```sql
SELECT queues.name, COUNT(calls.id) AS calls
FROM queues
LEFT JOIN calls ON calls.queue_id = queues.id
GROUP BY queues.id, queues.name
ORDER BY queues.name;
```

Usare sempre una transazione per modifiche manuali. `BEGIN IMMEDIATE` fallisce subito o attende secondo il timeout se un altro processo sta scrivendo:

```sql
BEGIN IMMEDIATE;

UPDATE queues
SET name = 'General services'
WHERE id = 1;

SELECT id, name FROM queues WHERE id = 1;

COMMIT;
-- Usare ROLLBACK; al posto di COMMIT; per annullare le modifiche.
```

Ripristinare manualmente la numerazione di una coda allo stato iniziale:

```sql
BEGIN IMMEDIATE;
UPDATE queues
SET current_value = min_value - 1
WHERE id = 1;
COMMIT;
```

Inserire una coda di prova rispettando i vincoli dello schema:

```sql
BEGIN IMMEDIATE;
INSERT INTO queues (
    name, prefix, digits, min_value, max_value, current_value
) VALUES (
    'Test queue', 'T', 3, 1, 999, 0
);
SELECT last_insert_rowid() AS queue_id;
COMMIT;
```

Per il `current_value` iniziale usare sempre `min_value - 1`: nell'esempio `1 - 1 = 0`.

Disabilitare manualmente uno sportello, solo per emergenza:

```sql
BEGIN IMMEDIATE;
UPDATE counters SET enabled = 0 WHERE id = 1;
COMMIT;
```

Per creare, ruotare, riattivare o revocare normalmente uno sportello usare l'interfaccia amministrativa o le API: inserimenti manuali in `counters` aggirerebbero la generazione sicura del token. Non inserire manualmente righe in `calls`, perché si perderebbe l'atomicità con l'avanzamento della coda. Per recuperare l'amministratore usare esclusivamente `turnino admin reset`.

Creare un backup dalla shell SQLite:

```text
.backup /tmp/turnino-backup.db
```

Esportare il risultato di una query in CSV:

```text
.headers on
.mode csv
.once /tmp/turnino-calls.csv
SELECT * FROM calls ORDER BY id;
```

## Configurazione locale

Variabili disponibili in `.env.example`:

```text
TURNINO_ADDR=:8080
TURNINO_DATA_DIR=./data
TURNINO_LOG_LEVEL=info
TURNINO_ADMIN_TOKEN=
```

## Decisioni da ricordare

- Account GitHub: `fvlgnn`
- Module path: `github.com/fvlgnn/turnino`
- Licenza: `AGPL-3.0-only`
- Nome definitivo: `Turnino`
- Bootstrap amministratore: impostare `TURNINO_ADMIN_TOKEN` soltanto quando il database non contiene ancora la credenziale; conservarne il valore originale fuori dal repository.

## Pubblicare una release

Prima della release, verificare che `main` sia aggiornata, pulita e con la CI verde:

```sh
git switch main
git pull --ff-only
git status
```

Spostare le modifiche da `[Unreleased]` a una sezione della versione in `CHANGELOG.md`, usando la data effettiva della release:

```markdown
## [Unreleased]

## [0.1.0] - YYYY-MM-DD

### Added

- Prima funzionalità pubblicata.
```

Aggiungere o aggiornare in fondo al changelog i link di confronto:

```markdown
[Unreleased]: https://github.com/fvlgnn/turnino/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/fvlgnn/turnino/releases/tag/v0.1.0
```

Verificare localmente che tag e note della versione siano validi ed estraibili:

```sh
sh .github/scripts/release-notes.sh v0.1.0 /tmp/turnino-v0.1.0.md
cat /tmp/turnino-v0.1.0.md
```

Committare il changelog, pubblicarlo su `main` e attendere che la CI termini correttamente:

```sh
git add CHANGELOG.md
git commit -m "docs: prepare v0.1.0 release"
git push origin main
```

Solo dopo la CI verde, creare un tag annotato sul commit appena pubblicato:

```sh
git tag -a v0.1.0 -m "Turnino v0.1.0"
git push origin v0.1.0
```

Il push del tag `v*` avvia automaticamente la workflow `Release`, che:

- crea gli archivi binari per Linux, macOS e Windows;
- estrae da `CHANGELOG.md` la sezione della versione;
- crea la GitHub Release usando quella sezione;
- pubblica l'immagine multi-architettura in `ghcr.io/fvlgnn/turnino`.

Controllare l'esecuzione nella scheda **Actions** e poi verificare la release:

```sh
gh run list --workflow=release.yml
gh release view v0.1.0
```

Dopo la prima pubblicazione, verificare su GitHub che il package GHCR sia pubblico.

Se il tag è sbagliato ma non è ancora stato inviato a GitHub:

```sh
git tag -d v0.1.0
git tag -a v0.1.0 -m "Turnino v0.1.0"
```

Non spostare o sovrascrivere un tag già pubblicato: creare una nuova versione, per esempio `v0.1.1`.

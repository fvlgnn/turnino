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

Gli endpoint sotto `/api/v1/admin` richiedono il token amministrativo. L'endpoint `calls/next` richiede invece il token dello sportello restituito al momento della registrazione. Gli esempi successivi mostrano anche rotazione e revoca; eseguire la revoca per ultima.

In alternativa, impostare manualmente `queueId` e `counterToken` nell'ultima sezione del file `.rest`. Non inserire credenziali direttamente nel file versionato.

## Comandi utili

```sh
make check
make build
make docker-build
docker build -t turnino:dev .
docker run --rm turnino:dev -version
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

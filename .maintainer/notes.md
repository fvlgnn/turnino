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
```

## Decisioni da ricordare

- Account GitHub: `fvlgnn`
- Module path: `github.com/fvlgnn/turnino`
- Licenza: `AGPL-3.0-only`
- Nome definitivo: `Turnino`

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

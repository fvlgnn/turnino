# TODO

## Prima della pubblicazione

- [ ] Inizializzare/pubblicare il repository sotto `github.com/fvlgnn`.
- [ ] Controllare che database, file locali e segreti non siano tracciati.
- [ ] Aggiungere una pagina `About / Source` visibile nell'interfaccia per AGPL-3.0.
- [ ] Definire copyright e contatto di sicurezza pubblici.
- [ ] Proteggere `main` e richiedere CI verde per le pull request.
- [ ] Rendere pubblico il package GHCR dopo la prima pubblicazione.

## v0.1

- [ ] Scegliere e documentare il driver SQLite pure Go.
- [ ] Creare schema, migrazioni automatiche e backup.
- [ ] Implementare `Queue`, `Counter`, `Call` e `Settings`.
- [ ] Rendere atomica l'allocazione del prossimo numero.
- [ ] Implementare registrazione sportello con token revocabile.
- [ ] Implementare chiamata automatica, manuale e richiamata.
- [ ] Implementare monitor pubblico e display sportello.
- [ ] Implementare aggiornamenti SSE con riconnessione.
- [ ] Implementare amministrazione essenziale.
- [ ] Aggiungere localizzazione italiana e inglese.
- [ ] Aggiungere test del dominio, HTTP, persistenza e concorrenza.
- [ ] Documentare installazione nativa, Docker e Raspberry Pi.

## Qualità e rilascio

- [ ] Aggiungere scansione vulnerabilità e SBOM alle release.
- [ ] Valutare il pin delle GitHub Actions a commit SHA.
- [ ] Testare realmente la release ARM64 su Raspberry Pi.
- [ ] Definire politica di versionamento e compatibilità delle migrazioni.
- [ ] Preparare screenshot e demo con dati interamente sintetici.

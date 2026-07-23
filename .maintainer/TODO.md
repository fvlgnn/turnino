# TODO

## Prima della pubblicazione

- [x] Inizializzare/pubblicare il repository sotto `github.com/fvlgnn`.
- [x] Controllare che database, file locali e segreti non siano tracciati.
- [ ] Aggiungere una pagina `About / Source` visibile nell'interfaccia per AGPL-3.0.
- [x] Definire copyright e contatto di sicurezza pubblici.
- [x] Proteggere `main` e richiedere CI verde per le pull request.
- [ ] Rendere pubblico il package GHCR dopo la prima pubblicazione.

## Fase 1 — Core queue calling

- [x] Scegliere e documentare il driver SQLite pure Go.
- [x] Creare schema e migrazioni automatiche.
- [x] Implementare `Queue`, `Counter` e `Call`.
- [ ] Implementare `Settings`.
- [x] Rendere atomica l'allocazione del prossimo numero.
- [x] Implementare registrazione sportello con token revocabile.
- [x] Implementare la chiamata automatica nel dominio, nella persistenza e nell'API HTTP.
- [ ] Creare l'interfaccia operatore/Caller per la chiamata automatica.
- [ ] Implementare la chiamata di un numero personalizzato, validandolo rispetto alla numerazione della coda.
- [ ] Implementare la richiamata dell'ultimo numero senza avanzare la coda.
- [ ] Creare l'interfaccia Sportello per l'operatore e il relativo display pubblico.
- [ ] Creare il monitor pubblico a tutto schermo con lista delle chiamate recenti, segnale acustico ed evidenziazione visiva delle nuove chiamate.
- [ ] Implementare aggiornamenti SSE con riconnessione e fallback polling.
- [x] Implementare amministrazione essenziale.
- [x] Aggiungere localizzazione italiana e inglese.
- [x] Implementare backup SQLite consistente ed export CSV dello storico chiamate.
- [x] Generare un database dimostrativo riproducibile con soli dati sintetici.
- [x] Aggiungere test del dominio, HTTP, persistenza e concorrenza.
- [x] Documentare installazione e sviluppo nativi e con Docker.
- [ ] Documentare installazione, avvio automatico e modalità kiosk su Raspberry Pi.

## Fase 2 — Qualità e rilascio

- [ ] Aggiungere scansione vulnerabilità e SBOM alle release.
- [ ] Valutare il pin delle GitHub Actions a commit SHA.
- [ ] Testare realmente la release ARM64 su Raspberry Pi.
- [ ] Definire politica di versionamento e compatibilità delle migrazioni.
- [x] Preparare dati dimostrativi interamente sintetici.
- [ ] Preparare screenshot usando esclusivamente i dati dimostrativi sintetici.

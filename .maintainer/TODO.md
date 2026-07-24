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
- [x] Implementare `Settings`.
- [x] Rendere atomica l'allocazione del prossimo numero.
- [x] Implementare registrazione sportello con token revocabile.
- [x] Implementare la chiamata automatica nel dominio, nella persistenza e nell'API HTTP.
- [x] Creare l'interfaccia operatore/Caller per la chiamata automatica.
- [ ] Implementare la chiamata di un numero personalizzato, validandolo rispetto alla numerazione della coda.
- [ ] Implementare la richiamata dell'ultimo numero senza avanzare la coda.
- [x] Creare l'interfaccia Sportello per l'operatore e il relativo display pubblico.
- [x] Creare il monitor pubblico a tutto schermo con lista delle chiamate recenti, segnale acustico ed evidenziazione visiva delle nuove chiamate.
- [x] Aggiusta header pagina admin (classe topbar tutto sezioni attaccate)
- [x] Implementare aggiornamenti SSE con riconnessione e recupero tramite snapshot.
- [ ] Implementare fallback polling per browser senza supporto SSE.
- [x] Implementare amministrazione essenziale.
- [x] Aggiungere localizzazione italiana e inglese.
- [x] Implementare backup SQLite consistente ed export CSV dello storico chiamate.
- [x] Generare un database dimostrativo riproducibile con soli dati sintetici.
- [x] Aggiungere test del dominio, HTTP, persistenza e concorrenza.
- [x] Documentare installazione e sviluppo nativi e con Docker.
- [x] Documentare monitor kiosk e avvio automatico del browser per Windows e Raspberry Pi.
- [ ] IDEA Vedi se possibile migliorare facilità di gestione caller (per registrazione, in una situazione reale potrebbe essere complicato registrare con token lo sportello)
- [x] Monitor pubblico con lista (font dei numeri progressivamente più piccolo).
- [x] Gestire audio autonomo tramite launcher kiosk, mantenendo il pulsante come fallback nella modalità test.

## Fase 2 — Qualità e rilascio

- [ ] Aggiungere scansione vulnerabilità e SBOM alle release.
- [ ] Valutare il pin delle GitHub Actions a commit SHA.
- [ ] Testare realmente la release ARM64 su Raspberry Pi.
- [ ] Definire politica di versionamento e compatibilità delle migrazioni.
- [x] Preparare dati dimostrativi interamente sintetici.
- [ ] Preparare screenshot usando esclusivamente i dati dimostrativi sintetici.

## Fasi successive

- [ ] Aggiungere politiche di numerazione alfanumerica progressiva per dispenser fisici, con rollover configurabile (es. `A00–Z99` e `AA00–ZZ99`), mantenendo il prefisso fisso come modalità semplice.
- [ ] Sistema automatizzato di stampa numero da totem dispenser
- [ ] Setting per temi, logo, banner, volume dettagliato e altre personalizzazioni.

# TODO

## Prima della pubblicazione

- [x] Inizializzare/pubblicare il repository sotto `github.com/fvlgnn`.
- [x] Controllare che database, file locali e segreti non siano tracciati.
- [x] Aggiungere una pagina `About / Source` visibile nell'interfaccia per AGPL-3.0.
- [x] Definire copyright e contatto di sicurezza pubblici.
- [x] Proteggere `main` e richiedere CI verde per le pull request.
- [x] Rendere pubblico il package GHCR dopo la prima pubblicazione.

## Fase 1 - Core queue calling

- [x] Scegliere e documentare il driver SQLite pure Go.
- [x] Creare schema e migrazioni automatiche.
- [x] Implementare `Queue`, `Counter` e `Call`.
- [x] Implementare `Settings`.
- [x] Rendere atomica l'allocazione del prossimo numero.
- [x] Implementare registrazione sportello con token revocabile.
- [x] Implementare la chiamata automatica nel dominio, nella persistenza e nell'API HTTP.
- [x] Creare l'interfaccia operatore/Caller per la chiamata automatica.
- [x] Implementare la chiamata di un numero personalizzato, validandolo rispetto alla numerazione della coda e consentendolo solo sulle code configurate per dispenser cartacei manuali.
- [x] Implementare la richiamata dell'ultimo numero senza avanzare la coda.
- [x] Implementare la chiamata del numero precedente aggiornando il progressivo.
- [x] Creare l'interfaccia Sportello per l'operatore e il relativo display pubblico.
- [x] Creare il monitor pubblico a tutto schermo con lista delle chiamate recenti, segnale acustico ed evidenziazione visiva delle nuove chiamate.
- [x] Aggiusta header pagina admin (classe topbar tutto sezioni attaccate)
- [x] Implementare aggiornamenti SSE con riconnessione e recupero tramite snapshot.
- [x] Implementare amministrazione essenziale.
- [x] Aggiungere localizzazione italiana e inglese.
- [x] Implementare backup SQLite consistente ed export CSV dello storico chiamate.
- [x] Generare un database dimostrativo riproducibile con soli dati sintetici.
- [x] Aggiungere test del dominio, HTTP, persistenza e concorrenza.
- [x] Documentare installazione e sviluppo nativi e con Docker.
- [x] Documentare monitor kiosk e avvio automatico del browser per Windows e Raspberry Pi.
- [x] Monitor pubblico con lista (font dei numeri progressivamente più piccolo).
- [x] Gestire audio autonomo tramite launcher kiosk, mantenendo il pulsante come fallback nella modalità test.

## Fase 2 - Numerazione progressiva

- [x] Rivedere e approvare il comportamento funzionale in `.maintainer/numbering-spec.md`.
- [x] Mantenere la numerazione con prefisso fisso come modalità semplice e compatibile.
- [x] Aggiungere una modalità alfanumerica progressiva per dispenser fisici, per esempio `A00-Z99` e `AA00-ZZ99`.
- [x] Offrire nel frontend preset corrispondenti ai rotoli comuni e una configurazione manuale avanzata.
- [x] Consentire la modifica della politica di una coda esistente, preservando lo storico già materializzato.
- [x] Usare sempre `wrap` al termine dell'intervallo.
- [x] Sostituire `sequence_value` con il numero formattato `number` nelle chiamate personalizzate.
- [x] Consentire la modifica della parte alfabetica durante una chiamata personalizzata.
- [x] Aggiungere alla postazione operatore un'azione secondaria di reset che riporti la coda allo stato iniziale senza cancellarne lo storico.
- [x] Gestire le modifiche incompatibili con un reset esplicito e confermato.

## Fase 3 - Personalizzazione

- [ ] Aggiungere palette o temi predefiniti senza consentire CSS arbitrario.
- [ ] Aggiungere un banner testuale configurabile.
- [ ] Aggiungere un volume dettagliato del segnale acustico.
- [ ] Valutare caricamento, validazione, persistenza e backup di un logo personalizzato.

## Fase 4 - Client GUI operatore

- [ ] Definire il problema operativo che il client deve risolvere rispetto al Caller nel browser.
- [ ] Considerare un wrapper WebView come soluzione più rapida e con minore duplicazione.
- [ ] Preferire Fyne come esperimento formativo in Go, accettando una seconda GUI e packaging multipiattaforma.
- [ ] Stabilizzare prima le API del Caller e mantenere il client opzionale rispetto all'interfaccia web.

## Integrazioni hardware future

- [ ] Definire un'API di emissione ticket soltanto dopo aver validato un caso d'uso reale.
- [ ] Scegliere hardware, collegamento e stampante prima di progettare un adattatore.
- [ ] Tenere stampa, driver e controllo del totem fuori dal core server, idealmente in un componente separato.
- [ ] Disabilitare le chiamate manuali per una coda solo quando è realmente gestita da un dispenser automatico.

## Consolidamento futuro

- [ ] Preparare screenshot usando esclusivamente i dati dimostrativi sintetici.
- [ ] Definire una politica di conservazione dello storico delle chiamate.
- [ ] Aggiungere una procedura esplicita di export o backup prima della cancellazione delle chiamate più vecchie di una data configurabile.
- [ ] Valutare una manutenzione SQLite successiva alla pulizia, senza cancellazione automatica predefinita.
- [ ] Valutare il pin delle GitHub Actions a commit SHA.
- [ ] Testare realmente la release ARM64 su Raspberry Pi.
- [ ] Definire politica di versionamento e compatibilità delle migrazioni.
- [ ] Valutare `govulncheck` quando aumenteranno dipendenze o utilizzatori.
- [ ] Generare una SBOM per le release se richiesta da distributori o installazioni organizzative.
- [ ] Valutare la scansione delle immagini container prima di una versione stabile.
- [ ] Valutare un lint aggiuntivo solo in presenza di problemi non coperti da `go vet`.

## Da valutare

- [ ] Vedi se possibile migliorare facilità di gestione caller (per registrazione, in una situazione reale potrebbe essere complicato registrare con token lo sportello)
- [ ] Valutare un fallback polling solo se emerge la necessità di supportare browser senza `EventSource`; SSE resta il trasporto principale.
- [ ] Valutare livelli di log configurabili solo quando emergerà un caso d'uso concreto.

# Guida di studio: monitor pubblico e Server-Sent Events

Questa guida descrive l'implementazione del monitor pubblico di Turnino. È pensata come materiale di studio: parte dal flusso applicativo, spiega le primitive Go usate e termina con prove manuali e automatiche.

## Obiettivo

Quando un operatore chiama un numero, il monitor deve:

1. mostrare immediatamente numero, coda e sportello;
2. mantenere una lista delle chiamate recenti;
3. evidenziare visivamente la nuova chiamata;
4. riprodurre un breve segnale acustico, se configurato e autorizzato dal browser;
5. riconnettersi senza intervento dell'utente;
6. recuperare lo stato corretto anche se una chiamata è avvenuta mentre il monitor era disconnesso.

Il monitor è pubblico e non usa token. Può leggere chiamate e sportelli già destinati alla visualizzazione nella sala d'attesa, ma non può eseguire operazioni.

## Flusso completo

```text
Caller
  |
  | POST /api/v1/queues/{id}/calls/{next|custom|recall}
  | Authorization: Bearer <counter token>
  v
HTTP API
  |
  | transazione atomica
  v
SQLite: aggiorna la coda + registra la chiamata
  |
  | commit completato
  v
Carica CallRecord con nomi di coda e sportello
  |
  v
Broker Go in memoria
  |
  +---- channel ----> connessione SSE monitor 1
  +---- channel ----> connessione SSE monitor 2
  +---- channel ----> connessione SSE monitor N
```

SQLite rimane la fonte autorevole. Il broker non conserva lo storico e non sostituisce il database: serve soltanto a notificare rapidamente i browser collegati.

## File principali

- `internal/live/broker.go`: distribuzione concorrente degli eventi.
- `internal/httpapi/api.go`: pubblicazione dopo la chiamata ed endpoint SSE.
- `internal/store/sqlite/store.go`: query dello storico recente e caricamento della chiamata arricchita.
- `internal/webui/templates/monitor.html`: stato HTML iniziale.
- `internal/webui/static/monitor.js`: `EventSource`, rendering, suono e fullscreen.
- `internal/webui/static/app.css`: layout fullscreen ed evidenziazione.

## Che cos'è SSE

Server-Sent Events è un protocollo HTTP testuale, unidirezionale: il server mantiene aperta una normale risposta HTTP e invia eventi al browser. Il browser comunica i comandi con normali richieste HTTP; SSE serve solo per gli aggiornamenti dal server al client.

Una risposta minimale è:

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache

retry: 2000
event: call
id: 42
data: {"id":42,"number":"A005","queue":"Servizi generali","counter":"Sportello 1"}

```

La riga vuota termina l'evento:

- `event` identifica il tipo;
- `data` contiene il payload, in Turnino codificato come JSON;
- `id` identifica l'evento;
- `retry` suggerisce al browser dopo quanti millisecondi riconnettersi.

Nel browser basta:

```js
const source = new EventSource("/api/v1/events/calls");

source.addEventListener("call", (event) => {
  const call = JSON.parse(event.data);
  console.log(call.number);
});
```

`EventSource` gestisce automaticamente la riconnessione. WebSocket sarebbe utile per una comunicazione bidirezionale persistente; Turnino invia invece comandi con `POST` e riceve notifiche in una sola direzione, quindi SSE è più semplice e aderente al KISS principle.

## Il broker Go

`live.Broker` contiene:

```go
type Broker struct {
    mu          sync.Mutex
    subscribers map[chan domain.CallRecord]struct{}
}
```

Ogni monitor ottiene un proprio channel bufferizzato tramite `Subscribe`. Il channel viene registrato nella mappa protetta dal mutex.

```go
events := broker.Subscribe(r.Context())
```

Il context della richiesta viene cancellato quando il browser chiude la scheda, perde la connessione o il server termina la richiesta. Una goroutine attende `ctx.Done()` e rimuove il subscriber:

```go
go func() {
    <-ctx.Done()
    b.mu.Lock()
    delete(b.subscribers, events)
    b.mu.Unlock()
}()
```

Questo evita di accumulare subscriber e goroutine appartenenti a browser non più collegati.

`Publish` attraversa i subscriber e prova a scrivere nei channel senza bloccare la chiamata dell'operatore. Se un browser è temporaneamente lento e il buffer è pieno, viene scartato l'elemento più vecchio e conservato quello più recente. La correttezza non dipende dal buffer perché ogni riconnessione riceve uno snapshot da SQLite.

Il mutex protegge contemporaneamente la mappa e le operazioni di sostituzione nel buffer. I channel non vengono chiusi: vengono soltanto rimossi dalla mappa. In questo modo `Publish` non rischia di inviare su un channel chiuso mentre un'altra goroutine cancella la sottoscrizione.

## Lo snapshot risolve gli eventi persi

Una riconnessione SSE da sola non garantisce che tutti gli eventi avvenuti durante la disconnessione vengano riprodotti. Turnino risolve il problema inviando subito:

```text
event: snapshot
data: [ ...ultime chiamate lette da SQLite... ]
```

La sequenza dell'handler è:

1. registra il subscriber nel broker;
2. legge da SQLite le ultime chiamate;
3. invia lo snapshot;
4. entra nel ciclo che inoltra gli eventi live.

Registrare prima il subscriber evita una finestra nella quale una chiamata potrebbe avvenire tra query e sottoscrizione. Una chiamata può comparire sia nello snapshot sia nel channel; il client elimina i duplicati usando `call.id`.

Il numero di elementi dello snapshot usa `Settings.RecentCallsLimit`.

## Handler HTTP SSE in Go

Una normale risposta può terminare quando l'handler ritorna. L'handler SSE resta invece in un ciclo fino alla cancellazione del context:

```go
for {
    select {
    case <-r.Context().Done():
        return
    case call := <-events:
        writeSSE(w, "call", call)
        flusher.Flush()
    case <-heartbeat.C:
        fmt.Fprint(w, ": keep-alive\n\n")
        flusher.Flush()
    }
}
```

### `http.Flusher`

Go può bufferizzare la risposta. `http.Flusher` forza l'invio immediato dei byte:

```go
flusher, ok := w.(http.Flusher)
flusher.Flush()
```

Senza `Flush`, gli eventi potrebbero rimanere nel buffer e apparire tutti insieme.

### Heartbeat

Ogni 15 secondi Turnino invia:

```text
: keep-alive

```

Le righe che iniziano con `:` sono commenti SSE e vengono ignorate da `EventSource`. Servono a mantenere attiva la connessione e a far rilevare interruzioni a server, browser e proxy.

### Write deadline

Il server Turnino usa un `WriteTimeout` di 30 secondi per impedire che normali richieste rimangano bloccate indefinitamente. Uno stream SSE deve vivere molto più a lungo.

L'handler rimuove la deadline soltanto dalla risposta SSE:

```go
_ = http.NewResponseController(w).SetWriteDeadline(time.Time{})
```

Disabilitare globalmente `WriteTimeout` ridurrebbe invece la protezione di tutti gli altri endpoint.

### Header contro buffering e cache

```go
w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
w.Header().Set("Cache-Control", "no-cache, no-transform")
w.Header().Set("Connection", "keep-alive")
w.Header().Set("X-Accel-Buffering", "no")
```

`X-Accel-Buffering: no` è utile con reverse proxy compatibili, in particolare nginx. In produzione bisogna comunque verificare che il proxy non accumuli la risposta SSE.

## Pubblicazione solo dopo il commit

La notifica viene pubblicata soltanto dopo che `CallNext` ha completato la transazione SQLite. Se il database eseguisse il rollback, il monitor non dovrebbe mostrare una chiamata inesistente.

Dopo il commit l'API carica un `CallRecord`, cioè la chiamata arricchita con `QueueName` e `CounterName`, e la pubblica:

```go
call, err := store.CallNext(...)
recordCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()
record, err := store.CallRecord(recordCtx, call.ID)
broker.Publish(record)
```

La seconda lettura usa un context breve indipendente dalla richiesta originale. Se il Caller chiudesse la connessione immediatamente dopo il commit, la notifica avrebbe comunque il tempo di essere preparata. Il timeout impedisce che questa lettura rimanga sospesa indefinitamente.

La risposta al Caller non dipende dalla velocità dei monitor: il broker usa channel bufferizzati e pubblicazione non bloccante.

## Rendering nel browser

Il template fornisce un primo HTML già leggibile, utile dopo un refresh e anche prima che JavaScript apra lo stream. Subito dopo, lo snapshot SSE riallinea la pagina.

`monitor.js` mantiene un array ordinato dal più recente:

```js
calls = [call, ...calls.filter((item) => item.id !== call.id)].slice(0, limit);
```

Per sicurezza i testi ricevuti non vengono concatenati come HTML: vengono assegnati a `textContent`. Nomi di code e sportelli non possono quindi iniettare markup.

L'ultima chiamata aggiorna:

- numero principale;
- nome della coda;
- sportello;
- lista recente;
- animazione CSS.

## Suono, fullscreen e modalità kiosk

I browser impediscono normalmente la riproduzione audio automatica e l'ingresso in fullscreen senza un gesto dell'utente. Un file OGG sarebbe soggetto alla stessa autoplay policy del tono generato tramite Web Audio.

Turnino separa quindi due viste:

- `/monitor` è la vista di produzione, senza controlli; prova ad attivare automaticamente l'`AudioContext` e si affida al browser avviato in modalità kiosk per fullscreen e autoplay;
- `/monitor?test=1` mostra "Abilita suono", "Schermo intero" e il collegamento al Caller come strumenti di sviluppo.

Se `Settings.SoundEnabled` è disabilitato, Turnino non prova ad armare l'audio. I launcher kiosk forniti aggiungono `--autoplay-policy=no-user-gesture-required`, mentre il fullscreen viene ottenuto all'avvio del browser e non richiesto dalla pagina.

Le procedure complete per Edge su Windows e Chromium su Raspberry Pi sono in `docs/kiosk.md`.

## Test manuale nello stesso browser

### Preparazione

1. avvia Turnino con `make dev`;
2. crea almeno una coda da `/admin`;
3. registra uno sportello e copia il token mostrato una sola volta;
4. apri <http://localhost:8080/caller>, salva il token e verifica il nome dello sportello.

### Monitor e Caller affiancati

Apri:

```text
http://localhost:8080/monitor?test=1
```

La modalità test aggiunge i controlli audio/fullscreen e "Apri Caller per il test". Il link apre una seconda scheda sullo stesso origin. Il `localStorage` del token è condiviso tra le schede dello stesso browser.

Per osservare entrambi contemporaneamente:

1. trascina una scheda in una seconda finestra;
2. affianca monitor e Caller;
3. sul monitor premi "Abilita suono";
4. dal Caller premi "Chiama prossimo".

Il monitor deve aggiornarsi senza refresh, evidenziare la chiamata e riprodurre il tono.

Il parametro `test=1` modifica soltanto i controlli visibili; non genera dati finti e non cambia il flusso SSE.

Per verificare il comportamento autonomo di produzione, avvia invece uno dei launcher descritti in `docs/kiosk.md`: il browser entra direttamente in fullscreen, autorizza l'autoplay e viene riavviato se termina.

### Verificare lo stream negli strumenti sviluppatore

Nel monitor apri DevTools, scheda Network, e seleziona la richiesta:

```text
/api/v1/events/calls
```

La richiesta deve restare `pending`. Nel pannello EventStream o Response devono apparire prima `snapshot`, poi gli eventi `call`.

### Verificare la riconnessione

1. lascia aperto il monitor;
2. riavvia Turnino;
3. osserva lo stato passare a "Connessione...";
4. dopo la riconnessione verifica che lo snapshot ripristini lo storico.

Non è necessario ricaricare la pagina: `EventSource` ritenta automaticamente usando il valore `retry`.

### Verificare da terminale

`curl -N` disabilita il buffering del client:

```sh
curl -N http://localhost:8080/api/v1/events/calls
```

Lascia il comando aperto e chiama un numero dal Caller. Dovresti vedere un evento `call`.

## Test automatici

- `internal/live/broker_test.go` controlla fan-out e subscriber lento.
- `internal/store/sqlite/store_test.go` controlla limite, ordine e arricchimento dello storico.
- `internal/httpapi/api_test.go` apre uno stream reale, legge lo snapshot, esegue una chiamata autenticata e legge l'evento live.
- `internal/webui/webui_test.go` controlla monitor con storico, monitor vuoto e modalità test.
- `go test -race ./...` verifica anche accessi concorrenti non sincronizzati.

## Limiti intenzionali

- Il broker è in memoria e appartiene a un singolo processo. È coerente con l'architettura Turnino a processo unico.
- Il monitor pubblico non ha autenticazione perché mostra informazioni destinate alla sala d'attesa.
- SSE è unidirezionale; i comandi restano normali richieste HTTP.
- Il polling di compatibilità non è ancora implementato. Browser moderni usano `EventSource`.
- Un reverse proxy deve consentire risposte lunghe e non bufferizzate.

Questi limiti mantengono il sistema semplice e adatto anche a Raspberry Pi. Un'eventuale architettura multi-replica richiederebbe un bus condiviso o un'altra strategia e non appartiene alla Fase 1.

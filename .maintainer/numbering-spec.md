# Specifica funzionale: numerazione progressiva

Stato: comportamento funzionale approvato e implementato.

## Obiettivo

Permettere a una coda di usare una sequenza alfanumerica progressiva compatibile con dispenser fisici, mantenendo l'attuale numerazione numerica con prefisso fisso come modalità semplice e predefinita.

La politica deve produrre il prossimo numero in modo deterministico e atomico, formattare sempre lo stesso progressivo nello stesso modo e conservare nello storico il numero effettivamente mostrato.

## Modalità

### Prefisso fisso

È il comportamento attuale:

```text
prefisso: A
cifre: 3
intervallo numerico: 1-999

A001, A002, A003, ..., A999, A001
```

Il prefisso non avanza. Le code esistenti devono migrare automaticamente a questa modalità senza cambiare numeri o comportamento.

### Parte alfabetica progressiva

La parte alfabetica avanza quando la parte numerica raggiunge il proprio massimo:

```text
lettere: 1
intervallo alfabetico: A-Z
cifre: 2
intervallo numerico: 0-99

A00, A01, ..., A99, B00, ..., Z99
```

Con due lettere:

```text
lettere: 2
intervallo alfabetico: AA-ZZ
cifre: 2
intervallo numerico: 0-99

AA00, ..., AA99, AB00, ..., AZ99, BA00, ..., ZZ99
```

Le lettere usano inizialmente l'alfabeto ASCII `A-Z`, senza caratteri accentati e senza distinzione tra maiuscole e minuscole nell'input. Il modello supporta una o due lettere e da una a sei cifre: combinazioni come `AA000` risultano quindi disponibili senza richiedere una modalità speciale, anche se non sono state individuate come formato commerciale comune.

## Configurazione nel frontend

Il frontend propone alcuni preset corrispondenti ai formati più comuni:

```text
Numerico 00-99
Numerico 000-999
Alfanumerico A00-E99
Alfanumerico A00-J99
Personalizzato
```

La modalità `Personalizzato` espone intervalli alfabetici e numerici. La sua presentazione definitiva resta da valutare durante il lavoro sull'interfaccia, ma il dominio e la persistenza devono supportarla fin dalla prima implementazione.

## Configurazione concettuale

I nomi definitivi di campi e colonne saranno scelti durante l'implementazione.

```text
mode                  fixed | progressive
fixed_prefix          stringa, solo per fixed
alphabetic_start      A oppure AA
alphabetic_end        Z oppure ZZ
numeric_digits        1-6
numeric_min           intero >= 0
numeric_max           intero rappresentabile nelle cifre configurate
```

La modalità `fixed` mantiene i campi e la semantica attuali: l'eventuale prefisso resta invariato e avanza soltanto la parte numerica. Dopo il massimo configurato, la sequenza riparte dal minimo. In modalità `progressive`, `alphabetic_start` e `alphabetic_end` devono avere la stessa lunghezza e l'inizio non può essere successivo alla fine. La larghezza alfabetica può essere ricavata dagli estremi, evitando configurazioni contraddittorie.

## Avanzamento

La parte alfabetica si interpreta come un indice in base 26:

```text
A  = 0
B  = 1
Z  = 25
AA = 0
AB = 1
AZ = 25
BA = 26
ZZ = 675
```

Il database conserva un unico progressivo ordinale. La formattazione ricava da quell'ordinale la parte alfabetica e quella numerica. In questo modo l'incremento resta atomico e non richiede l'aggiornamento coordinato di due contatori.

### Persistenza SQLite

La tabella `queues` memorizza soltanto configurazione e stato non derivabile:

- modalità, prefisso o intervallo alfabetico;
- cifre e intervallo numerico;
- `current_position`, con `-1` per una coda non iniziata;
- abilitazione delle chiamate personalizzate.

Il numero corrente e la dimensione totale della sequenza vengono calcolati dalla politica e non sono duplicati nel database. La tabella `calls` conserva il numero formattato mostrato e la relativa `position` interna. La posizione non viene esposta nell'API pubblica.

Durante la fase alpha lo schema è stato consolidato direttamente nella migrazione iniziale. I database creati prima di questo consolidamento devono essere ricreati: non vengono mantenute colonne o trasformazioni legacy nel nuovo schema.

Con `A-Z`, `00-99`, l'intervallo contiene:

```text
26 * 100 = 2600 numeri
```

Con `AA-ZZ`, `00-99`:

```text
676 * 100 = 67600 numeri
```

## Fine intervallo

Al termine dell'intervallo Turnino applica sempre `wrap`: dopo l'ultimo numero si riparte dal primo.

Esempio con `A-C`, `00-02`:

```text
A00, A01, A02, B00, B01, B02, C00, C01, C02
```

Il successivo è `A00`. Questa scelta rispecchia i rotoli fisici, che ripetono la sequenza, e preserva il comportamento delle code esistenti. Una chiamata personalizzata o il reset manuale permettono all'operatore di riallineare la coda quando necessario.

## Chiamate e storico

- La chiamata automatica avanza di una sola posizione e registra numero formattato e ordinale nella stessa transazione.
- La richiamata ripete il numero già registrato e non avanza la sequenza.
- Lo storico conserva la stringa mostrata, quindi resta leggibile anche se in futuro la politica della coda viene modificata.
- Due chiamate concorrenti non devono ricevere la stessa posizione.

## Chiamata personalizzata

Per una coda progressiva, un operatore dovrebbe inserire il numero completo, per esempio `BC37`, non un ordinale interno.

Il comportamento approvato è:

- normalizzare l'input in maiuscolo;
- precompilare la parte alfabetica corrente nel form, lasciandola modificabile;
- verificare formato, larghezza e intervalli;
- convertire il numero in ordinale;
- registrare una chiamata manuale;
- sincronizzare il progressivo, così la chiamata automatica successiva continua dalla posizione seguente.

L'API usa il numero formattato leggibile:

```json
{
  "number": "BC37"
}
```

`sequence_value` viene rimosso dall'input pubblico. Il server valida `number` rispetto alla politica della coda e lo converte nell'ordinale interno. La stessa rappresentazione viene usata per code fisse e progressive.

## Reset

La postazione operatore espone un'azione secondaria di reset vicino alla chiamata personalizzata. Il reset:

- è disponibile a ogni operatore autenticato con un token sportello valido;
- richiede una conferma esplicita;
- riporta la coda allo stato "non iniziata";
- non cancella né modifica lo storico;
- fa ripartire la chiamata automatica successiva dal primo numero configurato.

La stessa operazione è disponibile anche nell'amministrazione. Un reset pianificato o giornaliero resta fuori dalla prima implementazione.

## Validazione e casi limite

Devono essere rifiutati:

- larghezza alfabetica diversa da 1 o 2;
- lettere diverse da `A-Z`;
- estremi alfabetici con lunghezze differenti;
- intervallo alfabetico inverso, per esempio `Z-A`;
- numero di cifre fuori dai limiti supportati;
- intervallo numerico negativo, inverso o non rappresentabile;
- numero personalizzato fuori intervallo o con formato errato;
- configurazioni progressive con `fixed_prefix` valorizzato.

Casi da coprire con test:

```text
A00 -> A01
A99 -> B00
Z99 -> A00 con wrap
AA99 -> AB00
AZ99 -> BA00
ZZ99 -> AA00 con wrap
richiamata di B37 -> B37 senza avanzamento
chiamata manuale B37 -> automatico successivo B38
due chiamate concorrenti -> due numeri distinti
reset di una coda -> stato non iniziato e storico invariato
```

## Compatibilità e migrazione

- Ogni coda esistente diventa `fixed`.
- Prefisso, cifre, minimo, massimo e valore corrente restano invariati.
- Le API e le viste attuali continuano a rappresentare correttamente le code fisse.
- Le chiamate storiche non vengono riformattate.
- L'implementazione deve aggiungere una migrazione SQLite incrementale e non riscrivere le migrazioni già pubblicate.

## Modifica della politica

La politica di una coda può essere modificata anche dopo l'utilizzo:

- se il numero corrente è valido nella nuova politica, viene conservato e il successivo segue la nuova configurazione;
- se il numero corrente non è valido, l'interfaccia mostra un avviso;
- la conferma della modifica incompatibile riporta la coda allo stato "non iniziata";
- lo storico già registrato non viene cancellato o riformattato.

Esempio compatibile:

```text
numero corrente: B98
politica precedente: A00-E99
nuova politica: A00-J99
risultato: B98 resta corrente, il successivo è B99
```

Esempio incompatibile:

```text
numero corrente: E98
politica precedente: A00-E99
nuova politica: A00-C99
risultato dopo conferma: coda non iniziata, il successivo è A00
```

## Fuori ambito

- alfabeti personalizzati o esclusione di lettere come `I` e `O`;
- schemi arbitrari o espressioni di formato;
- più parti alfabetiche separate;
- emissione e stampa fisica del ticket;
- scelta di totem, stampanti o driver;
- reset pianificato;
- conservazione, esportazione e pulizia periodica dello storico.

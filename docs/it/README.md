# Turnino — note italiane

Turnino è la riscrittura moderna di un piccolo sistema eliminacode nato per funzionare anche su Raspberry Pi. L'obiettivo resta deliberatamente semplice: un server, terminali operatore nel browser, monitor pubblico, display dei singoli sportelli e configurazione essenziale.

Stack previsto:

- Go;
- SQLite;
- HTML generato dal server;
- Server-Sent Events;
- JavaScript minimale;
- singolo binario e immagine container multi-architettura.

La [roadmap](../roadmap.md), l'[architettura](../architecture.md) e la [guida allo sviluppo](../development.md) sono inizialmente mantenute in inglese per rendere il progetto accessibile a una comunità più ampia. L'interfaccia applicativa sarà disponibile almeno in italiano e inglese.

Il nome unisce l'inglese “turn” al diminutivo italiano “-ino”: un sistema piccolo e amichevole che comunica quando è arrivato il proprio turno. Continua inoltre la tradizione personale iniziata con Gianino, il progetto di domotica casalinga dell'autore.

Turnino segue il principio KISS: un processo, un database, poche viste nel browser e nessuna infrastruttura superflua.

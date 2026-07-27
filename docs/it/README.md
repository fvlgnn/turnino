# Turnino in italiano

Turnino è un sistema eliminacode leggero, open source e self-hosted. Nasce dalla riscrittura moderna di un'applicazione che per anni ha funzionato su hardware modesto, incluso Raspberry Pi, mantenendone l'idea originale: un solo server, postazioni operatore nel browser, un monitor pubblico, display dedicati agli sportelli e un'amministrazione essenziale.

## Funzionalità

La prima versione comprende:

- code con prefisso, intervallo e numerazione configurabili;
- chiamata automatica del numero successivo;
- chiamata manuale per code collegate a dispenser cartacei;
- ripetizione dell'ultima chiamata;
- postazione operatore, monitor pubblico e display del singolo sportello;
- aggiornamenti in tempo reale tramite Server-Sent Events;
- amministrazione protetta da una chiave di accesso;
- interfaccia in italiano e inglese;
- persistenza SQLite, backup ed esportazione CSV;
- esecuzione nativa, con Docker e su Raspberry Pi;
- modalità kiosk per Microsoft Edge e Chromium.

Turnino segue il principio KISS: un processo, un database, poche viste nel browser e nessuna infrastruttura superflua.

## Documentazione

La documentazione tecnica principale è mantenuta in inglese per rendere il progetto accessibile a una comunità più ampia:

- [avvio e panoramica](../../README.md);
- [configurazione](../configuration.md);
- [amministrazione e recupero accesso](../administration.md);
- [sviluppo](../development.md);
- [architettura](../architecture.md);
- [installazione kiosk](../kiosk.md);
- [sicurezza](../security.md);
- [risoluzione dei problemi](../troubleshooting.md);
- [roadmap](../roadmap.md).

L'interfaccia applicativa e i messaggi rivolti agli utenti sono disponibili sia in italiano sia in inglese.

## Perché Turnino?

Il nome unisce la parola inglese "turn" al diminutivo italiano "ino": un sistema piccolo e amichevole che comunica quando è arrivato il proprio turno. Il nome continua anche la tradizione iniziata con Gianino, il mio progetto di domotica domestica, e vuole essere un piccolo omaggio ad Arduino, la piattaforma che mi ha fatto appassionare alla programmazione, alla mentalità maker e alla filosofia open source.

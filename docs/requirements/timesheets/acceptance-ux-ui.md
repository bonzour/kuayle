# TSUX — implementazione e accettazione UX/UI

Implementazione: 2026-10-21–22. Verifica finale: 2026-10-22. Branch: `feature/efforts`; baseline applicativa `68f2cab`.

Il [backlog UX/UI](../../backlog/timesheets-ux-ui.md) è implementato nel frontend, comprese US-12 e US-15. L'accettazione complessiva resta aperta finché non sono completate le prove manuali elencate sotto. Nessuna modifica a backend, database applicativo, API, dipendenze, parser Efforts o autorizzazioni. Le modifiche documentali locali preesistenti sono preservate.

## Stato delle storie

“Implementata” descrive il codice consegnato, non equivale ad approvazione manuale del percorso.

| Storia | Consegna | Verifica |
| --- | --- | --- |
| US-01 | Parsing/formattazione delle date senza ora centralizzati; adapter per i campi DATE restituiti come mezzanotte UTC da Go | Unità nei tre fusi; browser progetti nei tre fusi, issue, cicli, Analytics |
| US-02 | `DatePopover`, clear esterno al trigger, focus comune; `DateInput` digitabile; range a un mese sotto 640px | Clear, focus, date obbligatorie, preset, range mobile e desktop |
| US-03 | Intera settimana/mese evidenziati nel calendario e descritti nel testo; anche l'indicatore “oggi” usa il riferimento server | Periodi, navigazione, tastiera, screenshot |
| US-04 | `MultiSelect` ricercabile con Combobox; primo nome e conteggio residuo; nomi completi nel controllo aperto | 100 opzioni, omonimi, nessun risultato, Timesheet e FilterBuilder |
| US-05 | Riepiloghi, rimozioni per dimensione e reset esplicito dell'ambito autorizzato | Vuoto/Tutti, filtri dipendenti, URL, avanti/indietro, revoche |
| US-06 | `FormSelect` per workspace destinatario e titolare | Registrazione globale, delega, sola lettura in modifica |
| US-07 | Modalità → periodo/azione → filtri → sintesi/totale → griglia; filtri mobile chiusi inizialmente | Cinque modalità, totale server completo, accesso e riconvalidazione |
| US-08 | Prima colonna 144/220/280px; giorni 88px; scroll interno e intestazioni sticky; totale sticky da 1024px | 320/390/768/1440px, due temi, nomi lunghi e paginazione |
| US-09 | `+` scrivibile, ore consultabili sottolineate, valori neutri nelle altre celle | Creazione, dettaglio, futuro, capacità locali e focus |
| US-10 | `IssueSearchDialog` condiviso; provider, esclusioni e paginazione negli adapter | Timesheet, parent issue, menu contestuale e azioni massive |
| US-11 | Unico form Efforts; contesto, titolare, date, durata, descrizione, riepilogo; errore con focus | Periodi inclusivi/weekend, durata, conflitti, risposta persa, revoche |
| US-12 | `WorklogActions` condiviso e `ResultPagination` nel dettaglio; metadati secondari espandibili | Modifica/eliminazione, conferma, pagine e ritorno al report |
| US-13 | Loading/error tradotti con varianti page/compact, retry Button e aria-busy | Timesheet, issue picker, Insights/Explorer e macchine |
| US-14 | Stati distinti e azioni realizzabili per periodo vuoto, filtri vuoti e workspace non selezionati | Errori non mostrati come zero, nessun ampliamento automatico |
| US-15 | Solo gruppi chiusi, coordinate e riferimenti logici di focus in memoria; record nascosti durante invalidazione | Collasso/scroll, cambi contesto, revoca, reconnect e live backend |
| US-16 | Matrice aggiornata e regressioni dei consumer | Copertura ed evidenze sotto; riesame visivo umano finale aperto |
| US-17 | Etichette, tastiera, stati e traduzioni nelle quattro lingue | Automazione e ispezione immagini; screen reader reale aperto |
| US-18 | Baseline, confronto e report integrato | Compiti osservati con utenti e accettazione prodotto aperti |

## Contratti e matrice dei consumer

Le primitive `bits-ui`, i token e le dipendenze restano quelli del progetto. Nessuna policy di accesso nei componenti di presentazione.

| Componente comune | Consumer effettivi dopo la migrazione | Contratto / differenza di dominio |
| --- | --- | --- |
| `utils/calendar-date.ts` | DatePickerPopover, DateRangePickerPopover, DueDatePickerPanel, DateInput, formattazione Timesheet | ISO di calendario; parsing invalido → undefined, formattazione UTC esplicita. `calendarDateFromApi` accetta solo date o mezzanotte UTC dei campi DATE; nessun timestamp reale convertito |
| `DatePopover` | Picker singolo, range, variante form e periodo Timesheet | Un trigger Button, clear opzionale separato, nome/descrizione, disabled, popover limitato alla viewport, ritorno del focus |
| `DatePickerPopover` / `DueDatePickerPanel` | Date progetto; scadenza issue e creazione issue; panel scadenze in BulkActionBar, IssueContextMenu e Inbox | Singolo `onchange(string \| null)`. Preset e date recenti nel panel; Inbox continua a trasformare la scelta in timestamp di snooze nel proprio adapter |
| `DateRangePickerPopover` | Creazione/modifica/pagina ciclo; AnalyticsDateRangePicker | `onchange(start,end)` solo a range completo; `onclear` opzionale. Cicli conservano vincoli di sovrapposizione; corretto il trigger annidato che impediva l'apertura del menu di modifica. Analytics conserva 30/90/180 giorni e clear `('', '')` |
| `DateInput` | EffortWorklogDialog in Timesheet, pagina issue e pannello laterale | Segmenti localizzati digitabili + calendario; stringa ISO o bozza incompleta vuota; min/max, required, disabled ed errore associato. Nessun clear nei campi obbligatori |
| `MultiSelect` | TimesheetFilter; status/priority di FilterBuilder in My issues, team e viste salvate | Solo array di stringhe, opzioni, ricerca locale e riepilogo. Adapter Timesheet preserva `null`, `[]`, ID e “Tutti”; FilterBuilder preserva stringhe separate da virgole e undefined. I selettori singoli non sono stati convertiti in multiselezioni |
| `FormSelect` | Workspace destinatario Timesheet; titolare Efforts | ID stringa, label associata, placeholder, errore, disabled; autorizzazioni e fallback nei consumer |
| `IssueSearchDialog` | TimesheetWorklogDialog; IssuePickerDialog → FullPageIssueView, BulkActionBar, IssueContextMenu | Dialog, ricerca, lista, frecce/Invio/Escape, caricamento/errore e focus. Timesheet usa getTimesheetIssues e tutte le pagine; gli altri usano listIssues, recenti, soglia ricerca ed esclusioni |
| `EffortWorklogDialog` | Tutti gli ingressi Timesheet e Efforts, anche pannello laterale | Un solo form, parser e protocollo di mutazione; revisione/idempotenza/conflitto invariati |
| `WorklogActions` | TimesheetWorklogDetail; EffortWorklogs/timeline | Solo callback modifica/eliminazione; visibilità, permessi e conferma nei consumer. Il menu non effettua richieste |
| `ResultPagination` | Griglia, dettaglio e ricerca issue Timesheet | Primitive Pagination esistenti; pagina/perPage/totale server e callback; nessuna paginazione client dei record |
| `LoadingState` / `ErrorState` | Timesheet, dettaglio e issue picker; Insights/Explorer; elenco/dettaglio macchine; impostazioni dev machines | page predefinito, compact per contenitori; messaggio tradotto, role status/alert, retry Button; aria-busy sul contenitore interessato |

`presentation.ts` non memorizza record: la chiave comprende utente, query, ambito autorizzato, riferimento e capacità dei workspace. Il ripristino accetta solo chiavi di gruppi ancora presenti. Cambio contesto, errore o perdita di accesso scartano lo stato incompatibile. Nessun localStorage/sessionStorage per questa funzione.

## Ambiente e comandi riproducibili

Frontend: Node 22.15.0, dipendenze già installate, Chromium Playwright. Eseguire dalla directory `UI`:

```sh
NODE_OPTIONS=--experimental-strip-types npm run test:unit
npm run check
npm run test:e2e -- tests/e2e/timesheets-worklogs.test.ts tests/e2e/timesheets.test.ts tests/e2e/efforts-worklogs.test.ts tests/e2e/efforts-planning.test.ts tests/e2e/efforts-settings.test.ts tests/e2e/insights.test.ts tests/e2e/dev-machines.test.ts tests/e2e/dev-machine-scope-settings.test.ts --workers=2 --output=../tmp/reports/timesheets-ux-ui/complete
```

La configurazione E2E standard esegue la build. Le fixture intercettano le API e i WebSocket nei test browser; non provano l'autorizzazione server.

| Verifica | Esito | Evidenza |
| --- | --- | --- |
| Unità | 16 suite superate, zero fallimenti | `unit.log` |
| Svelte/TypeScript | Zero errori; un avviso preesistente in `workspace/+layout.svelte:249` (`shortcutDefs`) | `check.log` |
| Browser integrato | 156 prove superate, zero fallimenti, sulla build prodotta dalla configurazione standard | `passed.log`, `passed/` |
| Ultimo adattamento menu ciclo | Nuova build superata e 1 prova mirata superata dopo il controllo dei tipi del trigger e l'aggiunta del nome accessibile | `cycle-final.log`, `cycle-final/` |
| Live backend/database isolati | 5 verifiche superate, zero page errors | `live/run.log`, `live/live-evidence.json` |

`complete/` include la build e il primo passaggio di 155 prove su 156; il selettore della nuova prova Inbox è stato corretto e tutte le 156 prove sono state rieseguite insieme in `passed/` usando la stessa build. L'ultima correzione del solo menu ciclo è verificata separatamente con una nuova build e il relativo percorso.

Per le prove live è stato compilato il backend senza modificarlo, creata una nuova banca dati `kuayle_tsux_*` su PostgreSQL 17 e applicate le migrazioni esistenti. Backend su 8086, preview con proxy su 4176, storage temporaneo, macchine disabilitate. Nessun dato del database applicativo utilizzato. Il database isolato è stato rimosso dopo le prove. Comando:

```sh
KUAYLE_LIVE_URL=http://127.0.0.1:4176 node tests/live/timesheets.mjs
```

Esecuzione live finale del 2026-10-22: **5 verifiche superate**, zero page errors: scrittura B → aggiornamento A via HTTP/WebSocket, bozza aperta, reconnect, revoca con salvataggio impedito, soli frame di invalidazione. Evidenze in `tmp/reports/timesheets-ux-ui/live/`. Una fixture browser non è stata considerata sostitutiva di queste prove. Il runner chiude tramite UI l’avviso ordinario di release, che aveva interrotto un primo tentativo; nessuna API o frame live intercettato.

## Evidenze e confronto

Directory locale: `tmp/reports/timesheets-ux-ui/` (ignorata da Git; conservare gli artefatti con la consegna). `README.md` collega le 14 coppie prima/dopo; `images-manifest.json` registra i percorsi e gli hash SHA-256 delle immagini.

- `before/`: baseline acquisita prima delle modifiche applicative, fixture `timesheets-worklogs`, 390×1000 e 1440×1000, tema scuro. Due percorsi superati; 14 immagini aperte e controllate.
- `passed/`: riesecuzione integrata verde sulla build verificata. Medesimi percorsi `TSUX journey evidence`, più geometria 320/390/768/1440 e chiaro/scuro, consumer esterni e lingue.
- `live/`: evidenza JSON dei frame e screenshot mobile successivo alla revoca.
- `first-pass/`, `regression/`, `controls/`, `recheck/`, `final/`, `acceptance/`, `verified/`, `complete/`, `inbox-final/`: esecuzioni intermedie, incluse anomalie e correzioni. Non sono tutte verdi e non costituiscono il risultato finale. `verified/` ha concluso 154 prove su 156; le due anomalie riguardavano il trigger di modifica ciclo e un selettore ambiguo della nuova fixture Inbox, corretti prima della riesecuzione finale. `complete/` ha poi concluso 155 prove su 156; restava il nome accessibile completo della notifica Inbox nel solo test, corretto e verificato in `inbox-final/`.

Il percorso fotografato include report, filtri, calendario del periodo, dettaglio, scelta issue, form e recupero dall'errore. A parità di fixture, il totale server è ora vicino alla griglia, i filtri mobile sono espandibili, la prima colonna lascia leggibile almeno un giorno intero a 320px e due a 390px; le celle scrivibili mostrano `+`. Le immagini di baseline con popover sono state acquisite durante la loro animazione: non usare l'opacità transitoria per misurare contrasto.

Le prove di reflow al 200% usano 720 CSS pixel e deviceScaleFactor 2 per rappresentare una viewport fisica di 1440px. Non sono una prova manuale del comando Zoom del browser o una certificazione di accessibilità.

## Verifiche manuali ancora aperte

Non dichiarare piena conformità o chiusura di US-17/18 sulla base di queste sole automazioni.

| Prova | Stato e passi da eseguire |
| --- | --- |
| Screen reader reale | Non eseguita. Con NVDA/Firefox oppure Orca/Chromium: leggere intestazioni e celle, cambiare periodo e filtri, aprire dettaglio, provocare un errore form, verificare annuncio e focus di ritorno |
| Zoom browser 200%, tastiera e contrasto umano | Automazione di reflow e tastiera presente; controllo manuale finale aperto su entrambi i temi e quattro lingue |
| Compito 1: totale progetto/persona | Sessione osservata prima/dopo non eseguita. Scegliere modalità, periodo e progetto/persona; indicare totale completo e ambito. Registrare riuscita, numero di azioni ed errori di selezione |
| Compito 2: lavoro su issue fuori griglia | Sessione osservata prima/dopo non eseguita. Aprire Registra lavoro, scegliere destinazione/issue, compilare data e durata, salvare; registrare destinatario/data/durata errati e passi |
| Compito 3: correggere e tornare al report | Sessione osservata prima/dopo non eseguita. Aprire cella, menu worklog, modifica, salvare e chiudere dettaglio; verificare report, focus e stato visivo, registrando passi ed errori |

Per i tre compiti non esistono misure umane comparabili: riuscita, errori e passi sono **non rilevati**, non zero. I test automatici coprono le operazioni, ma non misurano comprensione né miglioramenti percentuali. Un errore di data, destinatario o interpretazione della durata blocca l'accettazione del relativo percorso.

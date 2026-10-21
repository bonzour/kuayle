# Timesheets — verifica EP-06 e accettazione integrata

Data: **2026-10-21**. Riferimenti: [backlog](../../backlog/timesheets.md),
[requisito](README.md), [contratti](contracts.md#aggiornamenti-remoti-dellincremento-ep-06).

## Consegna

TS-US-16: sottoscrizioni autorizzate per tutti i workspace selezionati e per il
riferimento, incluso B con pagina aperta dallo slug A. Il canale riusa `/ws` e
invia solo invalidazioni prive di contenuti di dominio, filtrate prima della
coda. Comprende le modifiche ai progetti, che prima non pubblicavano refresh.
La membership viene riletta anche a socket inattivo. Il socket applicativo del
layout mantiene il proprio protocollo.

Il client conserva le connessioni pertinenti, elimina quelle rimosse, accorpa
le notifiche e rilegge anche dopo apertura/reconnect/chiusura anomala. Griglia,
dettaglio e richieste obsolete vengono invalidati prima della nuova lettura.
Le revoche riallineano l'ambito con feedback, mantenendo date e selezioni vuote;
le bozze dei comandi restano indipendenti dalla griglia e non vengono sostituite.
Non sono introdotti nuovi comandi, DTO REST, limiti globali, tabelle o indici.

## Riproduzione delle verifiche

Ambiente: Go **1.26.9**, Node **22.15.0**, PostgreSQL **17.11** in container
Alpine, Chromium Playwright del progetto, Linux x86_64 con 12 CPU logiche.
Le prove backend usano PostgreSQL reale e `DATABASE_URL` configurato; le fixture
Timesheet creano/eliminano database temporanei. Eseguire le regressioni di
portabilità su un database dedicato già migrato, poiché la suite transfer usa
fixture con cleanup nel database indicato.

Dalla directory `BE`:

```bash
go test ./internal/realtime ./internal/repository ./internal/service ./internal/domain ./internal/dto ./cmd/server -run 'TestTimesheet|TestEffort|TestWorkspaceTransferRoundTrip|TestRoutePermissionAnnotations|TestIntegratedRoutesPreserved|TestQueuedWebSocket|TestDeliveryPolicy|TestPrivateWorkspaceDoesNotRelayIncomingPresence' -skip '^TestTimesheetQueryMeasurements$' -count=1 -v
go test -race ./internal/realtime -count=1
TIMESHEET_QUERY_MEASUREMENTS=1 go test ./internal/repository -run '^TestTimesheetQueryMeasurements$' -count=1 -v
```

Dalla directory `UI`:

```bash
NODE_OPTIONS=--experimental-strip-types npm run test:unit
npm run check
npm run test:e2e -- tests/e2e/timesheets.test.ts tests/e2e/timesheets-worklogs.test.ts tests/e2e/efforts-worklogs.test.ts tests/e2e/efforts-settings.test.ts tests/e2e/efforts-planning.test.ts tests/e2e/insights.test.ts --workers=2
KUAYLE_LIVE_URL=http://127.0.0.1:4176 node tests/live/timesheets.mjs
```

L'ultimo comando richiede un backend reale su un database isolato migrato e un
frontend con proxy HTTP/WebSocket della stessa origine. È una suite opt-in,
come `tests/live/private-teams.mjs`: crea account e workspace attraverso le API,
elimina i workspace alla fine e richiede la distruzione del database di test
per eliminare anche gli account. Non intercetta HTTP o WebSocket. Gli artefatti
sono in `tmp/reports/ep06/`; screenshot e tracce delle suite UI sono in
`UI/test-results/`, esclusi da Git. Per un backend sulla porta 8086 si può usare
una configurazione Vite temporanea che importi `UI/vite.config.ts` e imposti il
proxy `/api` con `target: 'http://127.0.0.1:8086'`, `ws: true` e preview 4176.
Non usare un database condiviso per queste prove.

## Matrice di accettazione TS-AC-01–28

Le sigle seguenti indicano le suite eseguite, senza equiparare i mock browser
alle verifiche del backend:

- **R**: repository PostgreSQL, `BE/internal/repository/timesheets*_test.go`.
- **H**: HTTP/PAT, `BE/cmd/server/timesheets*_http_test.go`.
- **E**: regressioni Efforts di dominio, DTO, repository e HTTP.
- **T**: `TestWorkspaceTransferRoundTrip` con export/import reale.
- **U**: unit UI su selezioni, periodi, URL e sottoscrizioni.
- **B**: browser con fixture HTTP, `timesheets.test.ts` e `timesheets-worklogs.test.ts`.
- **L**: due sessioni browser con backend/PostgreSQL e WebSocket reali,
  `tests/live/timesheets.mjs`.

| AC | Evidenza riproducibile |
| --- | --- |
| 01 | R `TestTimesheetPersonalPostgres`; H `TestTimesheetPersonalHTTP`; B `sidebar, initial cross-workspace scope and local capabilities`. |
| 02 | U `timesheets-period.test.mjs`; E `TestEffortCalendarDate`, `TestEffortPeriodDates`; B `week/month navigation uses the reference and retains dates across filters`. |
| 03 | R `TestTimesheetPersonalPostgres`; B `personal grid keeps complete totals, hierarchy and distinct registrations`. |
| 04 | R `TestTimesheetPersonalPostgres`: figlio chiuso, senza assegnatario, stima o progetto; B griglia personale. |
| 05 | R `TestTimesheetAdministrativePostgres`; B `all administrative hierarchies preserve totals, scope and current classification` e `opening a person focuses the same report and cell detail includes only that owner`. |
| 06 | R `TestTimesheetAdministrativePostgres`; B `administrative people filters combine, preserve empty selections and survive mode switches`. |
| 07 | R `TestTimesheetAdministrativePostgres`, incluso lead con ruolo member; H `TestTimesheetPersonalHTTP` e `TestTimesheetOptionsHTTP`. |
| 08 | R `TestTimesheetPersonalPostgres`; H `TestTimesheetOperationsHTTP`; B `destination today validates future dates and readonly capabilities remain local`. |
| 09 | R `TestTimesheetPersonalPostgres`, `TestTimesheetIssuesPostgres`; E `TestEffortDailyBudgetPrivacyPostgres`; hub `TestTimesheetChannelQueuesOnlyCoalescedInvalidations`. |
| 10 | R `TestTimesheetAdministrativePostgres`, incluso contesto pubblico; H `TestTimesheetOptionsHTTP`, `TestTimesheetRealtimeHTTP`; B URL manipolati e selezioni revocate. |
| 11 | H `TestTimesheetOptionsHTTP`, `TestTimesheetPersonalHTTP`, `TestTimesheetOperationsHTTP`: scope, allowlist, token revocato e downgrade. |
| 12 | R `TestTimesheetAdministrativePostgres`: parent/figlio, assegnatari multipli, progetto corrente e Senza progetto senza eredità dal parent. |
| 13 | R `TestTimesheetAdministrativePostgres`: omonimi, membership rimossa e account non disponibili; T `TestWorkspaceTransferRoundTrip`; E `TestEffortDelegationRevocationPostgres`. |
| 14 | H `TestTimesheetOperationsHTTP`; B `global entry finds an unlogged issue beyond page one and retains administrative filters`; U parser durata delle suite Efforts. |
| 15 | E `TestEffortPeriodPostgres`, `TestEffortCommandsPostgres`, `TestEffortDailyBudgetPrivacyPostgres`; B periodo atomico e retry. |
| 16 | E `TestEffortWorklogCorrectionsPostgres`; H `TestTimesheetOperationsHTTP`; B `detail edits reread beyond 50 records, update both days, and delete with confirmation`. |
| 17 | E `TestEffortConcurrencyPostgres`, `TestEffortCommandsPostgres`; B `detail can add work and conflict handling preserves the draft until explicit realignment`. |
| 18 | E `TestEffortSettingsPostgres`; H `TestTimesheetOperationsHTTP`; B `a disabled destination rejects an open form and keeps its draft`. |
| 19 | H `TestTimesheetRealtimeHTTP`; U `timesheets-subscriptions.test.mjs`; B notifiche B, reconnect, risposte tardive e bozza; L tutti gli scenari remoti. |
| 20 | R `TestTimesheetPaginationPostgres`, `TestTimesheetAdministrativePaginationPostgres`; B pagine workspace, persone e dettaglio. |
| 21 | H errore operativo in `TestTimesheetPersonalHTTP`; B `a remote invalidation supersedes an in-flight grid; failed rereads show no partial total`, stati vuoti, tastiera e mobile; L mobile. |
| 22 | R `TestTimesheetAdministrativePostgres` e E `TestEffortDuplicateAndPreservedChildren`; T `TestWorkspaceTransferRoundTrip` controlla aggregati v2 e vista vuota v1. |
| 23 | R `TestTimesheetAdministrativePostgres`; H `TestTimesheetPersonalHTTP`, `TestTimesheetOptionsHTTP`; B downgrade e ambito amministrativo; L A admin/B member/C guest. |
| 24 | R `TestTimesheetOptionsPostgres`, `TestTimesheetAdministrativePostgres`; U `timesheets.test.mjs`; B `multiple filters retain namesakes and remove only incompatible children`. |
| 25 | R `TestTimesheetAdministrativePostgres`: quattro totali identici, progetto tra team e riclassificazione; B gerarchie amministrative. |
| 26 | R `TestTimesheetPersonalPostgres`; H `TestTimesheetOperationsHTTP`: 24h A + 1h B; B oggi locale e futuro; E calendario e periodi. |
| 27 | B scenari `monthly grid, tabs and detail fit a mobile viewport` in entrambi i temi; regressioni `insights.test.ts`; U periodi e URL. |
| 28 | B URL diretto/reload, avanti/indietro, ritorno issue e focus; H autorizzazione corrente; L revoca con periodo e bozza conservati. |

Le fixture coprono A admin/B member/C guest, allowlist, team privati, fusi diversi,
omonimi, titolari rimossi, deleghe, gerarchie e progetto condiviso tra team.
Il dataset paginato comprende 51 persone/progetti/team e 101 worklog sulla stessa
foglia; tutte le pagine sono confrontate su dati immutati e senza duplicati.
`TestTimesheetSnapshotNumericBoundaries` verifica esattamente il massimo intero
JSON sicuro, il primo valore oltre il limite e overflow int64: un errore non
restituisce snapshot parziali. Le somme SQL usano bigint e gli errori abortiscono
l'intera risposta.

## Misure PostgreSQL

[Test e piani EXPLAIN completi](measurements-ep06.txt). La fixture deterministica
crea esattamente 10 workspace, 100 persone, 100 team, 100 progetti, 1.000 issue,
100.000 worklog distribuiti nel 2025. Misure su 6–12 ottobre e ottobre completo,
con un riscaldamento, cinque chiamate repository e un successivo EXPLAIN per
combinazione. Le chiamate includono lettura, JSON e costruzione della gerarchia;
i tempi EXPLAIN misurano il solo statement SQL strumentato.

| Periodo / vista | Min / mediana / max repository (ms) | EXPLAIN esecuzione (ms) |
| --- | --- | --- |
| Settimana personale | 20,28 / 20,75 / 53,63 | 16,34 |
| Settimana persona | 188,02 / 191,96 / 240,45 | 170,07 |
| Settimana progetto | 185,47 / 196,26 / 248,67 | 174,01 |
| Settimana workspace | 282,47 / 294,57 / 310,92 | 195,30 |
| Settimana team | 193,76 / 200,50 / 244,88 | 165,14 |
| Mese personale | 64,38 / 64,78 / 65,64 | 61,52 |
| Mese persona | 718,20 / 734,26 / 751,59 | 348,58 |
| Mese progetto | 709,86 / 739,08 / 766,79 | 355,08 |
| Mese workspace | 914,67 / 925,89 / 948,82 | 473,80 |
| Mese team | 716,79 / 746,98 / 755,98 | 374,03 |

Lo statement misurato è quello della produzione, estratto in un costruttore
privato condiviso senza cambiare query o autorizzazione. I piani mostrano
materializzazioni e I/O temporaneo negli aggregati mensili. Non sono stati
aggiunti indici: queste misure non dimostrano che un nuovo indice risolva il
costo di aggregazione/serializzazione. Le letture restano una query per
proiezione, senza query per cella e senza scaricare tutti i worklog nel browser.
Altri processi locali di test/build erano attivi: non è un benchmark di carico,
non misura utenti simultanei e non costituisce uno SLA.

## Esiti e limiti

Le verifiche backend dei sei package indicati sono superate, inclusi HTTP/PAT,
revoca del socket inattivo, notifiche progetto e regressioni di portabilità.
Il race detector del realtime è superato. Le 14 suite unitarie UI sono superate.
Il giro finale browser è **115/115**: 57 Timesheet, 56 Efforts e 2 Insights.
Il caso ulteriormente precisato sull’accorpamento di tre notifiche in una sola
lettura e sul focus è stato riverificato separatamente sulla stessa build,
**1/1** superato. Check UI: zero errori, con il warning preesistente su
`shortcutDefs` nel layout. Build di produzione superata; resta l’avviso
preesistente sui chunk superiori a 500 kB. Confrontati visivamente anche
Timesheet e Insights desktop in tema chiaro; le suite producono catture in
entrambi i temi e viewport desktop/mobile.
La prova live ha verificato aggiornamento remoto di B da A, dettaglio aggiornato,
bozza invariata, lavoro durante disconnessione, reconnect, revoca, rifiuto della
scrittura, frame privi di dati riservati e viewport 390 × 844 senza overflow.
La cattura mobile è stata anche controllata visivamente.

TS-US-16 e TS-US-18 sono completate. La matrice copre l'accettazione del codice locale. Non certifica un deployment,
uno SLA o prestazioni sotto carico concorrente. Ogni workspace selezionato
richiede un socket aggiuntivo, oltre al socket del layout: scelta confermata per
questa versione. La revoca inattiva è rilevata con il controllo periodico di
cinque secondi; i dati riletti rimangono soggetti all'autorizzazione server.

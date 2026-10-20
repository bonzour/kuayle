# Efforts V1 — accettazione integrata T-03

Verifica del 2026-10-20 su `feature/efforts`. Baseline funzionale:
[EFF-001–041 e AC-001–051](efforts.md); implementazione:
[backlog](../backlog/efforts.md), [contratti](efforts-contracts.md),
[transazioni](efforts-transactions.md), [uso e rilascio](efforts-usage.md).

## Esiti riproducibili

- Backend: `go test ./... -race -count=1 -json` in `BE`, con `DATABASE_URL`
  su un database PostgreSQL 17.11 temporaneo migrato: **1.015 test/subtest passati**,
  **6 skip** estranei a Efforts, zero fallimenti. Tutti i test Efforts, HTTP,
  migrazioni, concorrenza, GitHub, realtime e trasferimento sono eseguiti.
- UI, Node 22.23.3: `npm run test:unit`, **40 passati**, zero skip.
- `npm run check`: **0 errori**, un warning preesistente sulla cattura iniziale
  di `shortcutDefs` nel layout workspace.
- `npm run test:e2e -- tests/e2e/efforts-settings.test.ts tests/e2e/efforts-planning.test.ts tests/e2e/efforts-worklogs.test.ts tests/e2e/issue-navigation.test.ts tests/e2e/private-teams.test.ts tests/e2e/workspace-transfer.test.ts --workers=2`:
  build di produzione superata e **57 E2E passati** (3,1 minuti, build inclusa).
- Controllo visivo degli screenshot di pagina e pannello: attività con operatore,
  timestamp e dettagli multilinea; totali separati; WORK LOG sotto ACTIVITY.
  `git diff --check` senza errori.

I test Go usano connessioni PostgreSQL reali e fixture isolate; i test browser
usano le fixture API/WebSocket del progetto. Le prove HTTP Go attraversano le
route reali e i PAT, mentre le fixture browser verificano UI, eventi e risposte
ritardate/perse. Non sono dichiarate prove manuali su tutti i browser o carico
multiutente di produzione.

I sei skip preesistenti sono `TestAnalyticsRepositoryPostgres` e
`TestIssueRepositoryPostgres` (URL di fixture dedicata assente),
`TestDockerRuntimeLifecycle` (opt-in Docker), `TestListAgentRunsScansPersistedRows`,
`TestListAgentRunLogsCursorPagination`, `TestListAgentRunEventsCursorPagination`
(righe agent persistite richieste dalle rispettive fixture). Non sono skip Efforts.
Nessuna migrazione o fixture di prova è stata applicata al database di sviluppo.
I database temporanei sono eliminati al termine.

## Suite di riferimento

I nomi brevi sotto rimandano ai file eseguibili, senza introdurre un secondo
framework di test.

| Sigla | Suite |
| --- | --- |
| M | [TestEffortMigration](../../BE/migrations/efforts_test.go): installazione nuova, upgrade da 46 con dati, down vuota/protetta, vincoli e attribuzioni. |
| D | [dominio](../../BE/internal/domain/efforts_test.go), [DTO](../../BE/internal/dto/efforts_test.go), [parser durata UI](../../UI/tests/unit/effort-duration.test.mjs), [validazione worklog UI](../../UI/tests/unit/effort-worklog.test.mjs). |
| C | [TestEffortCommandsPostgres, TestEffortConcurrencyPostgres, TestEffortDailyBudgetPrivacyPostgres](../../BE/internal/repository/efforts_commands_test.go). |
| P | [pianificazione repository](../../BE/internal/repository/efforts_planning_test.go) e [HTTP](../../BE/cmd/server/efforts_planning_http_test.go). |
| W | [correzioni/deleghe repository](../../BE/internal/repository/efforts_worklogs_test.go) e [worklog HTTP](../../BE/cmd/server/efforts_worklogs_http_test.go). |
| S | [settings repository](../../BE/internal/repository/efforts_settings_test.go) e [HTTP](../../BE/cmd/server/efforts_settings_http_test.go). |
| L | [lifecycle repository](../../BE/internal/repository/efforts_lifecycle_test.go), [HTTP/realtime T-02](../../BE/cmd/server/efforts_t02_http_test.go), [GitHub](../../BE/internal/service/github_efforts_test.go). |
| H | [letture, scope, pubblico, paginazione e gerarchie HTTP](../../BE/cmd/server/efforts_http_test.go). |
| V | [HTTP V1](../../BE/cmd/server/efforts_v1_http_test.go): PAT, storico, duplicazione, conversione, disattivazione e downgrade. |
| R | [repository V1](../../BE/internal/repository/efforts_v1_test.go): storico/identità, figli preservati, cambio parent, snapshot export e volumi. |
| A | [TestWorkspaceTransferRoundTrip](../../BE/internal/service/workspace_transfer_test.go): archive v2, legacy v1, deleghe, account eliminati, snapshot rimappati, archivio disabilitato e rifiuti atomici. |
| U-S | [E2E settings](../../UI/tests/e2e/efforts-settings.test.ts). |
| U-P | [E2E planning](../../UI/tests/e2e/efforts-planning.test.ts). |
| U-W | [E2E worklogs](../../UI/tests/e2e/efforts-worklogs.test.ts), inclusi i nuovi casi V1 su totali, attività, revoca, due sessioni e reconnect. |
| U-R | Regressioni [navigazione](../../UI/tests/e2e/issue-navigation.test.ts), [team privati](../../UI/tests/e2e/private-teams.test.ts), [trasferimento](../../UI/tests/e2e/workspace-transfer.test.ts). |

## Tracciabilità dei criteri

| AC | Comportamento verificato | Suite |
| --- | --- | --- |
| AC-001 | Creazione decrementa residuo impostato, aumenta consuntivo. | C, W, U-W |
| AC-002 | Correzione manuale residuo indipendente. | P, U-P |
| AC-003 | Inserimento data/durata/descrizione e registro. | W, U-W |
| AC-004 | Disattivazione conserva letture e congela mutazioni dirette. | S, L, V, U-S, U-W |
| AC-005 | Lavoro proprio senza assegnazione. | C, W, U-W |
| AC-006 | Guest legge, non scrive. | H, W, U-P, U-W |
| AC-007 | Titolarità e privilegi owner/admin per correzione/delete. | C, W, U-W |
| AC-008 | UUID noti, altri workspace e team privati non superano accesso. | H, W, U-R |
| AC-009 | Delete/update worklog bloccati da disabilitato. | S, W, U-W |
| AC-010 | Default nuovi/esistenti e disattivazione persistente. | M, S, U-S |
| AC-011 | Impostazioni solo sessioni owner/admin. | S, V, U-S |
| AC-012 | Pianificazione: assegnazioni multiple, ruolo e accesso correnti. | P, C, U-P |
| AC-013 | Creazione senza stima. | C, W, U-W |
| AC-014 | Revisione stima conserva consuntivo. | P, U-P |
| AC-015 | Worklog non assegnatario riduce residuo; correzione manuale negata. | C, P, W |
| AC-016 | Creazione con residuo assente lo conserva assente/mostrato zero. | W, U-P |
| AC-017 | Stima inizializza solo residuo assente, minimo zero. | P, U-P |
| AC-018 | Revisione stima conserva residuo esplicito. | P, U-P |
| AC-019 | Correzione durata/data/testo conserva residuo. | W, U-W |
| AC-020 | Delete worklog conserva residuo e corregge consuntivo. | C, W, U-W |
| AC-021 | Assente e zero esplicito restano distinti. | M, P, U-P |
| AC-022 | Parser 1h30m, minuto minimo, rifiuto frazioni/negativi. | D, P, W, U-W |
| AC-023 | Descrizione facoltativa, data/durata obbligatorie. | D, W, U-W |
| AC-024 | Oggi nel fuso workspace, date passate e future. | D, S, W, U-W |
| AC-025 | Conclusione/annullamento azzera solo residuo. | L, U-P |
| AC-026 | Worklog su issue chiuse. | C, L, W |
| AC-027 | Riapertura conserva zero; correzione autorizzata. | L, P |
| AC-028 | Propri e totali gerarchici distinti e non persistiti sul parent. | H, R, U-W |
| AC-029 | Nessun Efforts nelle risposte pubbliche o storico senza scope. | H, V, R |
| AC-030 | Delega distingue titolare/inseritore; membro non delega. | C, W, U-W |
| AC-031 | Delega solo a titolari attualmente ammissibili. | C, W |
| AC-032 | Create/update/delete ricostruibili dallo storico, con snapshot. | C, R, V, U-W |
| AC-033 | Stima/residuo prima-dopo e causa manuale/automatica. | C, L, R, U-W |
| AC-034 | Durata oltre residuo conservata integralmente. | W |
| AC-035 | 23h+1h accettato, 23h+1h1m rifiutato, anche delegato. | W, C |
| AC-036 | Correzione esclude durata precedente dal budget. | W |
| AC-037 | Cambio giorno rifiutato non altera entrambi i budget. | W, C |
| AC-038 | Inserimenti simultanei tra issue serializzati sul budget. | C |
| AC-039 | Default Roma, fuso indipendente da server/browser e ora legale. | S, U-S, U-W |
| AC-040 | Riepilogo/registro/totali in pagina e pannello. | U-P, U-W |
| AC-041 | Residuo chiuso bloccato anche per admin; stima/log non lo rialzano. | P, L, U-P |
| AC-042 | Dopo riapertura correzione manuale nuovamente ammessa. | L, P |
| AC-043 | Disabilita/chiudi/riattiva riconcilia chiuse con storico atomico. | S, L, R, U-S |
| AC-044 | Zero/rimozione stima, Unicode 2.000 e nessun arrotondamento. | D, P, W, U-P, U-W |
| AC-045 | Cambio fuso e import conservano date storiche. | S, A, U-W |
| AC-046 | Più livelli, cambio parent, filtri prima delle somme e refresh antenati. | H, R, U-W |
| AC-047 | Guest/rimozioni revocano scrittura, conservano storico e totali. | C, W, V, R, A |
| AC-048 | Copia sola stima, conversione conserva, delete conserva figli. | V, R |
| AC-049 | PAT su tutti i canali e round-trip completo v2/legacy. | H, V, R, A, U-R |
| AC-050 | Retry persistente, conflitti, rollback e aggiornamento sessioni. | C, P, W, L, A, U-S, U-P, U-W |
| AC-051 | Oltre 50 righe, tastiera, focus, conferma delete e sola lettura. | H, W, U-P, U-W |

## Misure su dati rappresentativi

`TestEffortRepresentativeQueries` crea 3.001 issue visibili nella gerarchia,
3.000 worklog da un minuto distribuiti su date entro il budget, 3.000 stime e
residui; 1.500 issue sono terminali. Misure dell'ultima suite con `-race`, sul
PostgreSQL locale, mentre altri pacchetti eseguono test:

| Operazione | Volume | Tempo osservato |
| --- | --- | --- |
| Repository `Summary` completo, inclusi filtri e consuntivo | 3.001 issue, 3.000 worklog | 1,954 s |
| `EXPLAIN (ANALYZE, BUFFERS)` del nucleo ricorsivo stima/residuo e visibilità | 3.001 contributi | 271,778 ms di esecuzione |
| Riattivazione atomica e storico | 1.500 residui azzerati e 1.500 eventi | 5,151 s |

Il piano del nucleo ricorsivo usa Recursive Union per deduplicare il grafo,
Hash Join per la traversata e i contributi; valuta il predicato di accesso prima
dell'aggregazione. È una misura della sottoquery, distinta dal tempo del riepilogo
completo, e non una promessa di latenza. La fixture e il piano restano
riproducibili nel test; non viene introdotta una soglia SLA arbitraria. Per grandi
gerarchie la latenza osservata del riepilogo resta un dato da considerare nel
monitoraggio del rilascio.

## Estensione: periodi, giorni da 8 ore e WORK LOG

Richiesta del 2026-10-20, successiva alla baseline V1. Le convenzioni UI sui
numeri senza unità e la collocazione dello storico sono aggiornate nei
[contratti](efforts-contracts.md#inserimento-per-periodo-e-timeline-efforts-2026-10-20)
e nella [guida d’uso](efforts-usage.md).

- [TestEffortPeriodDates e TestEffortPeriodPostgres](../../BE/internal/repository/efforts_period_test.go)
  coprono estremi inclusivi, weekend facoltativi, anno bisestile, cambio ora,
  date minime/massime, intervalli invalidi/vuoti/futuri, delega, singola revisione,
  replay persistente, conflitto di payload e budget su periodi sovrapposti tra
  issue. Un limite superato in una data successiva annulla righe precedenti,
  storico, residuo e ricevuta.
- [HTTP worklog](../../BE/cmd/server/efforts_worklogs_http_test.go) verifica
  campi periodo, replay e dettagli `end_date` con le route reali.
- Unit test UI verificano conversione esatta `1d = 480m`, default giorni,
  frazioni con virgola/punto, rifiuto delle frazioni di minuto, conteggio dei
  giorni indipendente dal fuso e filtro dei Remaining automatici, compresi gli
  eventi storici marcati manuali.
- E2E in pagina e pannello verificano anteprima e inclusione weekend, retry dello
  stesso periodo dopo risposta persa, conservazione delle bozze, spostamento
  Efforts da Activity a WORK LOG, soli Remaining manuali e Registrazioni
  richiudibili con CRUD e paginazione precedenti.

Suite Go completa su PostgreSQL con `-race`: **1.021 test/subtest passati**,
6 skip preesistenti estranei a Efforts, nessuno skip Efforts. UI: **43 unit test
passati**, controllo Svelte con zero errori e lo stesso warning preesistente.
Build di produzione e selezione Efforts settings/planning/worklogs, navigazione
e team privati: **60 E2E passati**. Nessuna nuova migrazione.

Dopo la compattazione del dialogo (Dal/Al affiancati) e il conteggio dei giorni
senza enumerare il periodo, nuova build e **10 E2E mirati passati**: periodi,
filtro storico, CRUD, paginazione/tastiera, Unicode e giorni come unità primaria.
Verificati visivamente dialogo periodo e WORK LOG nelle due viste. Backend locale
riavviato con questa versione; `/ready` e frontend rispondono 200.

# Timesheets — verifica EP-03 amministrativa

Data: **2026-10-20**. Riferimenti: [backlog](../../backlog/timesheets.md),
[requisito](README.md), [contratti API](contracts.md).

## Consegna

Implementate TS-US-07–10: letture amministrative Per persona, Per progetto,
Per workspace e Per team, con filtro persone e dettaglio per titolare.
«Apri timesheet» restringe la stessa vista Per persona, conservando periodo,
workspace, team e progetti. Non cambia in personale né amplia l'accesso.

Le quattro gerarchie usano gli stessi worklog autorizzati e gli stessi totali.
Le chiavi comprendono l'intero percorso: una issue con due titolari produce
foglie distinte e ciascuna cella apre soltanto le proprie registrazioni.
La pagina contiene 50 gruppi radice con discendenti completi; totali generali,
opzioni persone e subtotali non vengono limitati alla pagina.

Il repository conserva un solo statement PostgreSQL per ciascuna lettura,
comprensivo di autorizzazione, riferimento, aggregati e pagina. Sono applicati
ruolo corrente per workspace, visibilità e allowlist PAT. Nessun nuovo endpoint,
tabella, indice, totale persistito o cambiamento ai comandi Efforts.

Ex membri e account assenti mantengono l'UUID del titolare. Le identità assenti
hanno nomi nulli; la UI usa un'etichetta neutra e ID abbreviato, senza riabilitare
la delega. Team e progetto sono quelli attuali dell'issue. Importazioni e
cancellazioni si riflettono nei report tramite i record Efforts risultanti.

## Ambiente e riproduzione

Go **1.26.9**, Node **22.15.0**, Chromium tramite Playwright. PostgreSQL reale
**17-alpine** in container temporaneo dedicato `kuayle-ep03-postgres`, separato
dall'applicazione; il container è stato rimosso al termine dei test.
Le fixture repository/HTTP creano database isolati con tutte
le migrazioni; il round trip usa il database di test migrato nel medesimo
container. Nessun test PostgreSQL è stato sostituito da mock o saltato per
assenza di `DATABASE_URL`.

Impostare `DATABASE_URL` su un PostgreSQL esclusivamente di test con permesso
di creare/eliminare database. Dalla directory `BE`:

```bash
GOCACHE=/tmp/kuayle-go-cache CCACHE_DIR=/tmp/kuayle-ccache go test ./internal/domain ./internal/dto ./internal/middleware ./internal/repository ./cmd/server -run 'Effort|Timesheet|TokenWorkspaces|RoutePermissionAnnotations|IntegratedRoutesPreserved' -count=1
GOCACHE=/tmp/kuayle-go-cache CCACHE_DIR=/tmp/kuayle-ccache go run ./cmd/server migrate up
GOCACHE=/tmp/kuayle-go-cache CCACHE_DIR=/tmp/kuayle-ccache go test ./internal/service -run 'TestWorkspaceTransferRoundTrip' -count=1
GOCACHE=/tmp/kuayle-go-cache CCACHE_DIR=/tmp/kuayle-ccache go test ./internal/repository -run 'TestTimesheetAdministrative|TestTimesheetPaginationPostgres' -count=1
```

Dalla directory `UI`:

```bash
NODE_OPTIONS=--experimental-strip-types npm run test:unit
npm run check
npm run test:e2e -- tests/e2e/timesheets.test.ts tests/e2e/efforts-settings.test.ts tests/e2e/efforts-planning.test.ts tests/e2e/efforts-worklogs.test.ts tests/e2e/insights.test.ts --workers=2
npm run test:e2e -- tests/e2e/timesheets.test.ts --workers=2
```

Playwright esegue la build di produzione prima dei test. Le suite browser usano
fixture delle risposte HTTP per verificare l'interfaccia; autorizzazione e query
sono verificate separatamente dai test HTTP e repository con PostgreSQL reale.

## Evidenze raccolte

- **Backend:** superati i cinque package selezionati e il round trip del service.
  Superata anche la verifica finale dei test amministrativi e della paginazione
  workspace. Sono incluse le regressioni Efforts pertinenti a CRUD, periodi,
  delega, revisioni, idempotenza e lifecycle.
- **Aggregazioni:** dataset W1/W2 con 300 minuti totali, ripartiti 240/60;
  stessa persona sommata a 180 minuti fra workspace, omonimo distinto a 120.
  Verificate le quattro gerarchie, giorni completi, settimana e mese, progetto
  condiviso fra team, Senza progetto e assenza di ereditarietà dal parent.
- **Identità e lifecycle:** ex membro conservato, due account assenti distinti,
  spostamenti di progetto/team applicati alle date pregresse, parent eliminato
  con worklog dei figli conservati. Export/import v2 conserva 120 minuti e
  rimappa l'identità assente: sorgente e destinazione insieme restano due
  persone distinte per 240 minuti; import v1 senza Efforts restituisce vuoto.
- **Paginazione:** 51 persone, progetti e team omonimi e, nella fixture separata,
  51 workspace. Totali completi, pagine ripetibili senza duplicati e pagina oltre
  l'ultima vuota. Dettaglio di 101 registrazioni interamente raggiungibile.
- **HTTP/PAT:** tutte le modalità amministrative verificate su griglia e
  dettaglio; admin A/member B/guest C, filtri persone anche vuoti, scope mancanti,
  allowlist limitata/vuota, selezioni non autorizzate e downgrade a member.
  I test esistenti coprono anche revoca/scadenza token e guasto del repository.
- **Unit UI:** tutti i 12 file di suite superati, incluse serializzazione delle
  quattro modalità, selezione persone vuota, identità indisponibili e periodi.
- **Check UI:** zero errori; permane il warning preesistente su `shortcutDefs`
  nel layout workspace. La build segnala i chunk superiori a 500 kB.
- **Browser:** primo giro **80/80 superati**, con sei nuovi scenari amministrativi
  e regressioni Efforts/Insights. Verificati filtro persone, apertura persona,
  dettaglio per titolare, cronologia, paginazione e identità assente su mobile.
  Verifica finale **26/26 superati**, inclusi gli ulteriori casi di downgrade
  durante il dettaglio, risposta amministrativa tardiva e riferimento personale
  fuori dall'ambito amministrativo. Controllate visivamente le schermate
  desktop scuro e dettaglio mobile chiaro a 390 px, senza overflow della pagina.
  Dopo il controllo visivo è stata resa generica l'intestazione della griglia
  nelle quattro lingue; la compilazione traduzioni e il check UI sono stati
  rieseguiti.

## Limiti e tracciabilità

L'incremento copre TS-US-07–10 e le parti pertinenti di TS-AC-05–07, 09–13,
16, 20–25 e 28. Non dichiara completati tutti i 28 criteri né l'intera TS-01.

Restano alle Epic dedicate scritture dal Timesheet, ripristino da URL condivisibile,
notifiche autorizzate multi-workspace e misure prestazionali con EXPLAIN su volumi
rappresentativi. Non sono stati introdotti SLA o nuovi indici.

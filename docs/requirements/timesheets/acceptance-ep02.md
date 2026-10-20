# Timesheets — verifica EP-02 personale

Data: **2026-10-20**. Riferimenti: [backlog](../../backlog/timesheets.md),
[requisito](README.md), [contratti API](contracts.md).

## Consegna

Implementate TS-US-04, TS-US-05 e TS-US-06 nel perimetro personale: calendario
settimana/mese, griglia workspace → team → progetto → issue, subtotali completi
e dettaglio paginato dei worklog. Le due nuove route richiedono `efforts:read`
e rivalidano membership, visibilità e allowlist PAT per ogni workspace.

Ogni lettura usa un unico statement PostgreSQL per controllo dell'ambito,
riferimento, conteggi, totali e pagina. Il browser riceve celle già aggregate;
non scarica tutti i worklog per sommarli. Nessuna nuova tabella, migrazione,
copia dei worklog o modifica ai comandi di scrittura Efforts.

Per scelta esplicita dell'utente, il ritorno dall'issue conserva lo stato nella
voce di cronologia SvelteKit, associata a utente e slug: periodo, riferimento,
modalità, filtri e pagina vengono rivalidati. Il ripristino da URL condivisibile
rimane a TS-US-12. I link usano il workspace della registrazione.

## Ambiente e comandi riproducibili

Go **1.26.9**, Node **22.15.0**, Chromium tramite Playwright. PostgreSQL reale
**17-alpine** in container temporaneo dedicato, separato dal database applicativo.
Le fixture creano database isolati e applicano tutte le migrazioni, senza
sostituire le verifiche repository con mock UI e senza skip PostgreSQL.
Il container temporaneo è stato rimosso al termine delle verifiche.

Impostare `DATABASE_URL` su un PostgreSQL di test con permesso di creare/eliminare
database. Dalla directory `BE`:

```bash
GOCACHE=/tmp/kuayle-go-cache CCACHE_DIR=/tmp/kuayle-ccache go test ./internal/domain ./internal/dto ./internal/middleware ./internal/repository ./cmd/server -run 'Effort|Timesheet|TokenWorkspaces|RoutePermissionAnnotations|IntegratedRoutesPreserved' -count=1
```

Dalla directory `UI`:

```bash
NODE_OPTIONS=--experimental-strip-types npm run test:unit
npm run check
npm run test:e2e -- tests/e2e/timesheets.test.ts tests/e2e/efforts-settings.test.ts tests/e2e/efforts-planning.test.ts tests/e2e/efforts-worklogs.test.ts tests/e2e/insights.test.ts --workers=2
```

La configurazione E2E esegue anche la build di produzione. Il sandbox richiede
accesso alla rete locale per PostgreSQL, server UI e Chromium; le cache Go e
ccache sono state indirizzate sotto `/tmp`.

## Evidenze

- **Backend:** superati i cinque package selezionati, incluse le regressioni
  Efforts su CRUD, delega, periodi, idempotenza, revisioni, budget e lifecycle.
  Le nuove verifiche includono titolare diverso dall'inseritore, parent/figli,
  issue chiusa/non assegnata/senza stima, progetto assente e riferimento progetto
  non disponibile, guest, Efforts disabilitato, revoca e selezioni vuote.
- **Paginazione PostgreSQL:** dataset con 51 workspace omonimi e 101 worklog
  con timestamp uguali; totali completi su tutte le pagine, ordinamento stabile,
  assenza di duplicati e pagine oltre l'ultima senza perdita dei totali.
- **HTTP/PAT reale:** admin A/member B/guest C, scope read/write separati,
  precedenza del PAT sulla sessione, allowlist limitata/null/vuota, riferimento
  fuori dalla selezione dati e fuori allowlist, persona altrui, parametri
  manipolati, revoca del token/membership e guasto del repository senza totali
  parziali. 1.440 minuti in A più 60 in B sono leggibili senza limite globale.
- **Calendario:** settimana a cavallo dell'anno, febbraio bisestile, DST e fusi
  browser differenti; date e minuti non vengono convertiti. PostgreSQL verifica
  anche il 29 febbraio e HTTP confronta `today` con il fuso configurato.
- **Unit UI:** superati tutti i 12 file di suite, compresi calendario, formati,
  client API, transizioni dei filtri e regressioni Efforts.
- **Check UI:** zero errori; rimane il warning preesistente su `shortcutDefs`
  nel layout workspace. La build segnala inoltre chunk superiori a 500 kB.
- **Browser integrati:** **74/74 superati** sulla build di produzione, inclusi
  i 17 scenari Timesheet e le regressioni Efforts/Insights. Verificati griglia,
  calendario/popover, ritorno dall'issue nel workspace corretto, conservazione
  del periodo alla revoca del riferimento, errori/retry, risposte tardive,
  perdita dell'accesso durante il dettaglio, tastiera e ripristino del focus.
  Verificati visivamente desktop scuro e dettaglio mobile chiaro a 390 px;
  la griglia mensile scorre nel proprio contenitore senza overflow della pagina.
  Dopo il controllo finale dei periodi vuoti, **17/17 scenari Timesheet**
  rieseguiti con esito positivo sulla build finale; il dataset browser per
  paginazione comprende 52 workspace con contributi coerenti con i totali.

I periodi senza lavoro e i filtri senza risultati mantengono visibili le colonne
di tutti i giorni e i totali a zero, insieme al rispettivo messaggio. Gli errori
nascondono invece la griglia e non vengono presentati come zero ore.

Una fixture simula un riferimento progetto precedente ai vincoli sui team
privati, disabilitando e riattivando soltanto il relativo trigger nel database
isolato di test. Le scritture applicative non aggirano questi vincoli.

## Limiti e tracciabilità

La verifica copre l'ambito personale degli AC condivisi con altre Epic:
TS-AC-01–04, 08–09, 12, 20–21 e le parti pertinenti di 25–28.
Non dichiara completati i 28 AC né l'intera TS-01/TS-US-17.

Restano alle Epic dedicate aggregazioni amministrative, registrazione e
correzione dal Timesheet, ripristino URL completo, notifiche autorizzate da
tutti i workspace e misure prestazionali delle query su volumi rappresentativi.
Le modalità amministrative presenti nei selettori mantengono il messaggio di
consultazione non ancora disponibile. Non sono stati aggiunti indici o SLA.

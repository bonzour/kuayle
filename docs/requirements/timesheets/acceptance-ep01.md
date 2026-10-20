# Timesheets — verifica delle fondazioni EP-01

Data: **2026-10-20**. Riferimenti: [requisito](README.md),
[backlog](../../backlog/timesheets.md), [contratti](contracts.md).

## Consegna e stato

Implementato l'incremento concordato: autorizzazione trasversale, endpoint
`GET /api/timesheets/options`, selettori multipli e pagina navigabile
`/{workspaceSlug}/timesheet`. Le query griglia, aggregati e dettaglio sono
soltanto documentate e non sono route operative.

| Storia | Stato di questo incremento |
| --- | --- |
| TS-US-01 | Membership/ruoli/visibilità/PAT verificati per ambiti e opzioni; applicazione alle future letture ancora pendente |
| TS-US-02 | Selezione multipla, intersezione fra modalità e stati UI implementati; integrazione dei risultati pendente |
| TS-US-03 | Filtri team/progetti, omonimi, selezione vuota e rimozione figli implementati; verifica su contributi/totali pendente |
| TS-US-17 | Opzioni operative con PAT e contratti delle letture definiti; griglia/aggregati/dettaglio non implementati |

**EP-01 rimane parziale.** Non si dichiara completata TS-01 né soddisfatti tutti
i 28 AC. La pagina esplicita che la consultazione delle ore non è ancora
disponibile e non presenta zeri o totali simulati. Nessuna nuova tabella,
migrazione o indice; modifiche Efforts preesistenti preservate.

## Ambiente e comandi riproducibili

Go **1.26.9**, Node **22.15.0**, Chromium tramite Playwright. PostgreSQL reale
**17-alpine** in container temporaneo dedicato, distinto dal database applicativo;
fixture repository/HTTP con database isolati e migrazioni complete. Il container
temporaneo è stato rimosso al termine delle verifiche backend.

Impostare `DATABASE_URL` su un PostgreSQL di test con permesso di creare/eliminare
database. Sul database base di test applicare le migrazioni. Dalla directory `BE`:

```bash
go run ./cmd/server migrate up
go test ./internal/domain ./internal/dto ./internal/middleware ./internal/repository ./cmd/server -run 'Effort|Timesheet|TokenWorkspaces|RoutePermissionAnnotations|IntegratedRoutesPreserved' -count=1
```

Nel sandbox le cache Go/ccache sono state indirizzate a directory sotto `/tmp`.
Non sono stati usati skip per sostituire i test PostgreSQL.

Dalla directory `UI`:

```bash
NODE_OPTIONS=--experimental-strip-types npm run test:unit
npm run check
npm run test:e2e -- tests/e2e/timesheets.test.ts tests/e2e/efforts-settings.test.ts tests/e2e/efforts-planning.test.ts tests/e2e/efforts-worklogs.test.ts tests/e2e/insights.test.ts --workers=2
```

Il flag TypeScript è necessario per le suite esistenti con Node 22.15; senza
quel flag alcune importazioni `.ts` falliscono. Gli script del progetto non
sono stati modificati per aggirare questo requisito dell'ambiente.

## Esiti osservati

- **Backend:** tutti e cinque i package selezionati superati. Le nuove suite
  verificano admin A/member B/guest C, workspace estranei, team e progetti privati,
  omonimi, Efforts disabilitato, fusi, selezioni vuote, downgrade e revoca.
  Le chiamate dirette al repository applicano anche scope PAT, allowlist e
  l'eventuale `Access.WorkspaceID` preesistente.
- **HTTP reale:** autenticazione assente, scope mancanti, PAT revocati/scaduti,
  allowlist limitata, allowlist vuota/null, precedenza PAT sulla sessione,
  modalità manipolate e selezioni esplicite negate. Gli errori non includono
  opzioni parziali; un guasto del repository produce `500`, non una lista vuota.
- **Regressioni backend Efforts:** letture, impostazioni, CRUD, periodi,
  delega/revoca, concorrenza, idempotenza, budget giornaliero, storico e lifecycle
  superati nelle suite selezionate.
- **Unit UI:** tutti gli **11 file di suite** superati, inclusi Timesheet,
  client API, Efforts e PAT. Verificate serializzazione dei filtri e transizioni
  senza ampliamento di selezioni vuote.
- **Check/build:** check senza errori; resta un warning nel layout non modificato
  relativo a `shortcutDefs`. Build completata dal comando E2E; presente anche
  l'avviso dimensione chunk oltre 500 kB.
- **Browser:** i **7 scenari Timesheet** passano, inclusi default trasversale,
  modalità, filtri combinati, revoca, downgrade, errore/retry, risposta tardiva,
  tastiera e viewport 390 px. Verificati visivamente screenshot desktop scuro
  e mobile chiaro, con token effettivi e assenza di overflow orizzontale.
- **E2E integrati:** **63/64** superati nel giro con Efforts/Insights. Il caso
  Efforts guest ha incontrato `ERR_NETWORK_CHANGED` sui bundle JavaScript,
  in concomitanza con la rimozione del container temporaneo. La riesecuzione
  isolata dello stesso scenario sulla medesima build ha dato **1/1 superato**,
  senza modificare codice o test Efforts. Tutti i 64 scenari sono quindi stati
  osservati con esito positivo, con questa anomalia ambientale registrata.

Per il retry è stata usata una configurazione temporanea che importa quella
del progetto e avvia soltanto `npm run preview -- --port 4174`, riutilizzando
la build finale. Filtro del test: `guest keeps saved work readable and removes
writing actions`, un worker. Nessuna configurazione aggiuntiva è stata inserita
nel repository. Screenshot e tracce sono negli output Playwright locali.

## Verifiche ancora pendenti

Somme, gerarchie, conteggi, paginazione e snapshot delle future letture;
conservazione dei minuti nelle quattro aggregazioni; calendario, worklog,
persone storiche e operazioni Efforts dalla nuova pagina. Restano inoltre
nelle Epic dedicate il ripristino URL completo, le notifiche da tutti i
workspace e le misure prestazionali delle query dati. In questo incremento
la selezione viene rivalidata al caricamento, al cambio dei filtri e al refresh
manuale; non viene dichiarato un aggiornamento remoto multi-workspace.

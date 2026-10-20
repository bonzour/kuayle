# Timesheets — verifica EP-05 operazioni Efforts

Data: **2026-10-21**. Riferimenti: [backlog](../../backlog/timesheets.md),
[requisito](README.md), [contratti](contracts.md#ricerca-issue-e-operazioni-dellincremento-ep-05).

## Consegna

TS-US-13–15: registrazione dalla toolbar, dalle celle issue vuote e dal dettaglio;
modifica e conferma eliminazione dei singoli worklog. Il selettore cerca anche
issue senza lavoro nella griglia e raggiunge le pagine successive. La toolbar
offre tutti i workspace personali scrivibili, anche fuori dall'ambito del report
amministrativo, senza cambiare filtri, periodo o URL dopo il salvataggio.

Ogni operazione porta esplicitamente il workspace destinatario. Configurazione,
oggi/fuso, capacità e revisione vengono riletti; modifica/eliminazione cercano
il record corrente anche oltre la prima pagina. La delega usa i destinatari
ammissibili dell'issue e segnala il ripiego sul chiamante. Il dialogo Efforts
condiviso conserva parser, periodi atomici, weekend, revisioni, storico, residuo
e tentativi idempotenti. Bozze e retry non dipendono dal caricamento della griglia.
I rifiuti di accesso/configurazione riallineano la vista senza eliminare il form.

Il nuovo `GET /api/timesheets/issues` restituisce metadati minimi con
`efforts:read`, applicando membership, allowlist e visibilità prima di conteggi e
paginazione nello stesso statement SQL. Nessun nuovo comando di scrittura,
tabella, migrazione o limite giornaliero cross workspace.

## Riproduzione

Backend con Go **1.26.9**, PostgreSQL reale e `DATABASE_URL` configurato.
Le fixture richiedono creazione/eliminazione di database temporanei isolati.
Dalla directory `BE`:

```bash
go test ./internal/repository -run 'TestTimesheet|TestEffort' -count=1 -v
go test ./internal/domain ./internal/dto ./cmd/server -run 'TestTimesheet|TestEffort|TestRoutePermissionAnnotations|TestIntegratedRoutesPreserved' -count=1 -v
```

UI con Node **22.15.0**, dipendenze e Chromium Playwright installati.
Dalla directory `UI`:

```bash
NODE_OPTIONS=--experimental-strip-types npm run test:unit
npm run check
npm run test:e2e -- tests/e2e/timesheets-worklogs.test.ts tests/e2e/timesheets.test.ts tests/e2e/efforts-worklogs.test.ts tests/e2e/efforts-planning.test.ts tests/e2e/efforts-settings.test.ts --workers=2
```

Playwright esegue la build di produzione e avvia il server preview. Nei sandbox
che impediscono socket locali o Chromium occorre un ambiente che li consenta.
I mock HTTP dei browser verificano l'integrazione UI e affiancano le prove
PostgreSQL/HTTP reali; non le sostituiscono.

## Evidenze

- Repository PostgreSQL Timesheet/Efforts: superati selettori, quattro aggregati,
  paginazione, ricerca senza worklog, team privati, ruoli, allowlist e revoche,
  oltre alle regressioni Efforts di periodi, budget, concorrenza e lifecycle.
- Dominio/DTO, HTTP/PAT e manifest delle route: superati, incluso il nuovo test
  integrato `TestTimesheetOperationsHTTP`. Un PAT con soli scope Efforts ottiene
  suggerimenti e scrive entro il proprio workspace; la selezione delega rimane
  riservata alle sessioni.
- Il test HTTP verifica 24h in A più 1h in B = 25h, spostamento di una
  registrazione fra due date, cancellazione senza ricalcolo del residuo,
  mantenimento degli eventi worklog e replay senza ricreare dati cancellati.
  Verifica anche delega, guest, conflitto, disattivazione in A con B scrivibile,
  downgrade e revoca di B.
- Unit UI: superati tutti i **13 file di suite**.
- Check UI: **zero errori**, un warning preesistente su `shortcutDefs` nel layout
  workspace. Build di produzione superata; resta l'avviso preesistente sui chunk
  superiori a 500 kB.
- Browser: superate tutte le **96 regressioni esistenti** (**40 Timesheet** e
  **56 Efforts**, fra settings, planning e worklogs). Sulla build finale sono
  state riverificate le **30 regressioni Worklogs**, poi tutti i **12 nuovi
  scenari EP-05**, superati. L'ultimo giro dei 12 scenari riusa la build di
  produzione appena verificata con una configurazione temporanea equivalente
  che avvia solo il preview. Il comando completo sopra riproduce anche la build.
- I nuovi scenari coprono selezione di issue non presenti in griglia oltre la
  prima pagina, salvataggio in B da un report amministrativo aperto in A senza
  cambiare URL/filtri, precompilazione da cella, periodo e retry senza duplicati,
  correzione oltre 50 worklog, spostamento data e conferma eliminazione con
  riallineamento della pagina. Verificati anche conflitto e riallineamento
  esplicito, disattivazione/downgrade/revoca a form aperto, delega e titolare non
  più delegabile, ricerca con errore/riprova e risposta tardiva dopo cambio
  workspace, oggi locale e data futura, lettura guest e revoca rilevata durante
  l'apertura senza chiudere il form.
- Tastiera e mobile: verificati apertura con Invio, Escape e ritorno del focus
  alla cella; nessun overflow orizzontale della pagina a **390 × 844 px**.
  Controllata visivamente la schermata del dialogo in tema scuro. La cattura
  riproducibile è in `UI/test-results/**/ep05-mobile-worklog.png`, artefatto locale
  escluso da Git. I nuovi testi sono presenti nelle quattro lingue e compilano
  con Paraglide.

## Tracciabilità e limiti

L'incremento completa TS-US-13–15 e le parti pertinenti di TS-AC-08, 13–18, 26 e
28, mantenendo le verifiche precedenti su accesso, calendario e navigazione.
Il refresh dopo le proprie scritture ricarica griglia e dettaglio; le notifiche
remote autorizzate multi-workspace, l'accettazione integrata di tutti i 28 AC e
le misure delle query restano EP-06. Non è dichiarata completata l'intera TS-01.

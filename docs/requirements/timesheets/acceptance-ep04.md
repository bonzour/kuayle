# Timesheets — verifica EP-04 UI e navigazione

Data: **2026-10-21**. Riferimenti: [backlog](../../backlog/timesheets.md),
[requisito](README.md), [contratti](contracts.md#navigazione-ui-dellincremento-ep-04).

## Consegna

TS-US-11/12: cinque tab delle modalità con primitive e token Insights, filtri
multipli e calendario settimana/mese esistenti; URL con modalità, periodo,
riferimento, filtri e pagina. Apertura, reload e cronologia rivalidano accessi e
dati. Selezioni vuote restano distinte da Tutti. Revoche e downgrade rimuovono
ambiti e modalità non consentiti, senza recuperare dati dalla sessione o ampliare
implicitamente la selezione. Il riferimento sostituito conserva le date.

Richieste identificate per generazione e utente usano uno snapshot della vista.
Le risposte superate non pubblicano dati né normalizzano l'URL. Errori di rete
non diventano totali zero. Link malformati richiedono un ripristino esplicito.

Tab attivati da tastiera, selettori multipli, calendario, griglia e dettaglio
mantengono focus visibile e semantica tabella/dialogo. Chiusura del dialogo:
ritorno alla cella o al titolo se la cella è rimossa. Le azioni Timesheet di
riprova, ripristino e chiusura sono tradotte nelle quattro lingue disponibili.
ErrorState accetta un'etichetta opzionale conservando il default per gli altri
chiamanti. Non sono cambiati query, API backend, schema o comandi Efforts.

## Riproduzione

Dalla directory `UI`, con Node **22.15.0**, dipendenze installate e Chromium
Playwright disponibile:

```bash
NODE_OPTIONS=--experimental-strip-types npm run test:unit
npm run check
npm run test:e2e -- tests/e2e/timesheets.test.ts tests/e2e/insights.test.ts tests/e2e/efforts-settings.test.ts tests/e2e/efforts-planning.test.ts tests/e2e/efforts-worklogs.test.ts --workers=2
```

Il comando E2E esegue prima la build di produzione. Nei sandbox che impediscono
socket locali o Chromium, il comando richiede un ambiente che li consenta.
I browser usano fixture HTTP per la UI; non sostituiscono le prove PostgreSQL e
HTTP/PAT degli incrementi precedenti. Nessun test backend aggiunto o rieseguito
per questo incremento esclusivamente frontend.

## Evidenze

- **Unit UI:** superati tutti i **13 file di suite**, compresi i nuovi casi URL
  sulle cinque modalità, Tutti/lista vuota/Senza progetto, filtri persone,
  periodo/riferimento/pagina, input malformati, cambio anno, bisestile e DST.
  Il comando senza `--experimental-strip-types` non esegue le suite esistenti
  che importano direttamente TypeScript in Node 22; il comando riportato sopra
  è quello verificato.
- **Check e build:** check con **zero errori**; resta il warning preesistente
  su `shortcutDefs` nel layout workspace. Build di produzione superata tramite
  Playwright; resta l'avviso sui chunk superiori a 500 kB.
- **Browser:** **56/56 regressioni Efforts** superate nel giro completo; verifica
  finale **42/42 Timesheet e Insights** superata dopo la correzione del focus
  iniziale del calendario, del ritorno al pulsante dopo il caricamento e del
  mantenimento del focus sul tab attivato. Il dettaglio mantiene il footer
  raggiungibile anche con descrizioni molto lunghe.
  Quest'ultima esecuzione comprende i due ulteriori scenari su pagina oltre
  l'ultima e risposta HTTP 400. La verifica finale riusa la build di produzione
  appena verificata nei tre scenari mirati di tastiera e dettaglio mobile,
  avviando solo il server preview con una configurazione temporanea equivalente.
  Riproduzione completa, inclusa una nuova build:

  ```bash
  npm run test:e2e -- tests/e2e/timesheets.test.ts tests/e2e/insights.test.ts --workers=2
  ```

- **URL e accesso:** verificati apertura diretta, reload, avanti/indietro senza
  nuove voci, ritorno dalle issue in un altro workspace, filtri persone e
  paginazione, selezioni esplicitamente vuote, revoche e downgrade durante la
  cronologia, riferimento revocato con febbraio bisestile conservato, errori
  malformati/HTTP 400 e risposte tardive incapaci di riscrivere l'URL corrente.
- **Tastiera e traduzioni:** verificati tab manuali con frecce, Fine e
  Invio/Spazio, selettori multipli, selezione della data da tastiera, scorrimento
  della griglia, Escape e ritorno del focus alla cella o al titolo se rimossa.
  Verificate in italiano riprova, ripristino e chiusura del dettaglio.
- **Confronto visivo:** schermate Insights e Timesheet in tema chiaro/scuro,
  desktop e mobile a **390 × 844 px**. Nessun overflow orizzontale della pagina;
  lo scorrimento resta nella barra dei tab e nella griglia. Il dettaglio mobile
  include titoli, nomi e descrizioni molto lunghi nelle fixture. Le schermate
  sono prodotte in `UI/test-results/**/output/playwright/ep04-*.png` e sono
  artefatti locali riproducibili delle suite, esclusi da Git.

## Tracciabilità e limiti

L'incremento riguarda TS-US-11–12 e le parti UI di TS-AC-10, 19, 21, 27–28.
TS-AC-19 resta parziale: le sottoscrizioni remote autorizzate a tutti i workspace
selezionati appartengono a EP-06. Scritture dal Timesheet restano a EP-05;
accettazione integrata e misure PostgreSQL a EP-06.

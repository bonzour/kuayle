# Timesheet — backlog UX/UI e coerenza dei componenti condivisi

Data: **2026-10-21**. Baseline: branch `feature/efforts`, commit **`68f2cab`**, working tree inizialmente pulito.
Stato: **proposto, da implementare**. Consegna: analisi e backlog, senza modifiche al codice applicativo e senza creazione di ticket esterni.

Il backlog comprende **6 epic e 18 storie**. Gli ID `TSUX-EP-*` e `TSUX-US-*` sono locali a questo documento. Estende il [backlog funzionale Timesheets](timesheets.md), senza riaprire come mancanti funzionalità già implementate o sostituirne l'accettazione.

> **Aggiornamento implementazione (2026-10-22):** codice delle 18 storie e matrice dei consumer in [accettazione UX/UI](../requirements/timesheets/acceptance-ux-ui.md). Le sezioni di analisi e baseline seguenti restano storiche; lo stato di verifica corrente, i risultati e le prove manuali ancora aperte sono nel report. Nessuna P2 esclusa dal perimetro.

## 1. Obiettivo, metodo e limiti

Rendere più semplice scegliere l'ambito e il periodo, leggere le ore e registrare/correggere lavoro. Calendari, selettori, menu e dialoghi con lo stesso scopo devono avere un'implementazione condivisa e comportamenti coerenti anche fuori da Timesheet.

**Metodo effettivamente eseguito:** lettura dei requisiti, delle consegne EP-01–06, del codice Timesheet, dei consumer condivisi e dei test pertinenti; esecuzione delle quattro suite unitarie Timesheet; riproduzione isolata del problema di conversione delle date descritto in F-01.

**Limite:** il browser integrato non è disponibile in questa sessione. Non sono state acquisite schermate né svolte prove manuali sul prodotto. Questo è un esame del codice e dei flussi previsti, **non un audit visivo concluso**. Densità, clipping, contrasto, dimensioni effettive dei bersagli e usabilità con screen reader richiedono verifica. Le precedenti evidenze EP-04/06 restano documentazione storica, non nuove prove di questa attività.

**Decisioni di perimetro:**

- Conservare struttura e linguaggio visivo di Kuayle: token di [app.css](../../UI/src/app.css), primitive in `components/ui`, componenti condivisi e convenzioni Insights.
- Correggere il componente comune e migrare tutti i suoi consumer interessati. Una copia in Timesheet, anche visivamente simile, non soddisfa il requisito.
- Condividere presentazione e interazione; mantenere negli adapter di dominio autorizzazioni, API e significato dei dati. Un menu di azioni, una selezione multipla e un calendario di periodo non hanno la stessa semantica.
- Non creare un framework di design system, una nuova libreria UI o un generico motore di filtri. Estrarre solo parti con consumer reali, riportati nella matrice della sezione 4.
- Le modifiche esterne a Timesheet riguardano i componenti effettivamente condivisi e le loro integrazioni; nessun ridisegno generale delle altre pagine.

### Invarianti da preservare

Derivano dal [requisito TS-01](../requirements/timesheets/README.md) e dai [contratti](../requirements/timesheets/contracts.md):

1. Permessi e capacità del workspace destinatario, controllati sul server. Lo slug di navigazione non autorizza altri workspace.
2. `Tutti`, lista esplicitamente vuota e `Senza progetto` sono stati diversi. OR nella stessa dimensione, AND fra dimensioni; nessun ampliamento implicito dopo una revoca o un cambio modalità.
3. `work_date` è una data di calendario. Settimana lunedì–domenica; mese intero. Il riferimento del report resta stabile; oggi e validazione delle scritture provengono dal workspace destinatario.
4. Ore/minuti esatti nelle viste Timesheet. L'input Efforts mantiene `1 = 1d = 480 minuti`; nessuna interpretazione dei giorni vuoti come assenze e nessun obiettivo implicito di 8h.
5. Totali completi per l'intero insieme filtrato, non per la pagina; niente somme di subtotali e figli. Un errore non diventa zero o un totale parziale.
6. Unico flusso di mutazione Efforts: revisione, idempotenza, titolare immutabile in modifica, conferma eliminazione, protezione delle bozze e limite per titolare/data/workspace. Il totale cross workspace può superare 24h.
7. URL, avanti/indietro, ritorno dalle issue e invalidazioni multi-workspace continuano a funzionare senza ripristinare dati revocati.

## 2. Quanto è già implementato

Il backlog funzionale e [l'accettazione EP-06](../requirements/timesheets/acceptance-ep06.md) dichiarano implementate le sei epic originali. Il codice attuale contiene effettivamente:

- Vista personale e aggregazioni per persona, progetto, workspace e team; filtri multipli e dettaglio paginato.
- Periodi settimana/mese, riferimento temporale esplicito e formattazione Timesheet indipendente dal fuso del browser.
- Griglia semantica con intestazioni, prima colonna fissa, regione scorrevole da tastiera e nomi accessibili delle celle interattive.
- Ingresso globale «Registra lavoro», creazione dalla cella, correzione/eliminazione tramite **lo stesso `EffortWorklogDialog` usato nelle issue**.
- Primitive comuni `Tabs`, `Calendar`, `Select`, `Popover`, `Dialog`, `Button`, `TeamIcon`, `LoadingState` ed `ErrorState`.
- Ripristino URL, gestione delle risposte tardive, revoche, riconnessioni e feedback di salvataggio.

Non si propone di ricostruire queste funzionalità. L'intervento riguarda leggibilità, riconoscibilità delle azioni, recupero dagli stati vuoti e consolidamento dei componenti.

### Flusso da verificare nel browser

| Passo | Compito dell'utente | Stato ricavato dal codice | Miglioramento |
| --- | --- | --- | --- |
| 1 | Aprire Timesheet e capire quali dati vede | Ambito e capacità espliciti; più blocchi precedono la griglia | Sintesi compatta di ambito, periodo e totale; dettagli espandibili |
| 2 | Scegliere workspace, team, progetto e persona | Multiselezione presente; trigger con solo conteggio | Nomi selezionati, ricerca e rimozione consapevole dei filtri |
| 3 | Scegliere settimana/mese | Navigazione e calendario presenti; selezione visuale di un singolo giorno | Evidenziare il periodo effettivo e uniformare i date picker |
| 4 | Leggere ore e aprire una cella | Tabella e focus presenti; cella vuota scrivibile e non scrivibile mostrano entrambe «–» | Affordance distinte e più spazio ai giorni su viewport stretto |
| 5 | Registrare o correggere lavoro | Scrittura condivisa con Efforts; ricerca issue dedicata e controlli nativi | Selezione issue e campi comuni, gerarchia chiara e contesto persistente |
| 6 | Tornare al report e gestire aggiornamenti/errori | Protezioni dati e retry presenti; la griglia viene smontata durante il caricamento | Continuità di orientamento, annunci accessibili e azioni sugli stati vuoti |

Tutti i sei passi richiedono ancora osservazione visiva nella nuova iterazione.

## 3. Evidenze e priorità

**P1:** correttezza del dato mostrato, coerenza dei controlli e frizioni nel percorso principale. **P2:** miglioramento successivo di leggibilità o continuità. Nessun P0 assegnato sulla base di un problema visivo non osservato.

| ID | Evidenza verificata nel sorgente | Impatto / limite della conclusione | Priorità | Storie |
| --- | --- | --- | --- | --- |
| F-01 | `DatePickerPopover` e `DateRangePickerPopover` convertono `YYYY-MM-DD` con `new Date(value)` e poi leggono giorno/mese locali | Riprodotto: `2026-10-21` diventa 20/10 nel fuso America/Los_Angeles. Timesheet usa già una conversione sicura; il difetto è nei componenti condivisi | P1 | 01 |
| F-02 | Il comando X di `DateRangePickerPopover` chiude il popover senza chiamare `onchange`; entrambi i picker hanno un `span role="button"` nel trigger e nessun nome accessibile esplicito per X | Il clear del range non modifica il valore; l'interazione annidata richiede correzione e verifica da tastiera | P1 | 02 |
| F-03 | Date picker condivisi, Analytics e Timesheet compongono trigger diversi; Efforts usa `input type="date"`. Timesheet passa al calendario solo `value.from` come selezione singola | Esperienza frammentata; la selezione del periodo non è rappresentata interamente. Non si deduce un errore nel calcolo del periodo | P1 | 02, 03, 11 |
| F-04 | `TimesheetFilter` mostra «N selezionati», senza nomi né campo ricerca; `FilterBuilder` compone separatamente chip e liste multiselezione | Per conoscere il filtro occorre riaprirlo; costo di scansione da misurare su molte opzioni. Il typeahead della primitiva non è una ricerca visibile | P1 | 04, 05 |
| F-05 | Workspace nella scelta destinazione e titolare nel dialogo Efforts sono `<select>` nativi; i filtri Timesheet e Insights usano `Select` | Controlli equivalenti dipendono da composizioni e styling differenti | P1 | 06 |
| F-06 | Toolbar filtri, riga feedback/azioni, periodo, elenco completo dell'ambito e note precedono la tabella; totale generale solo nel footer | Rischio di gerarchia dispersiva e di griglia molto in basso, soprattutto con molti workspace; da confermare visivamente | P1 | 07 |
| F-07 | Prima colonna della griglia `min-w-[280px]`, giorni `min-w-[88px]`; header e totale finale non sono sticky | A 390px la colonna descrittiva può consumare quasi tutta la larghezza utile. La suite controlla lo scorrimento, non prova che sia comodo | P1 | 08 |
| F-08 | Celle vuote scrivibili, celle future e celle di sola lettura condividono il testo visibile «–»; le prime sono pulsanti | L'azione è distinguibile dalla semantica accessibile ma poco esplicita visivamente | P1 | 09 |
| F-09 | `TimesheetWorklogDialog` ha lista issue e ricerca proprie; `IssuePickerDialog` usa un overlay manuale, tasti freccia e un'altra resa dei risultati | Riuso possibile della presentazione, non dell'API generica: Timesheet ha endpoint dedicato e paginazione che vanno mantenuti | P1 | 10 |
| F-10 | Dettaglio Timesheet con card e due pulsanti per worklog; registro Efforts con timeline e icone. Giorni e fusi sono ripetuti nel dettaglio; alcune date sono ISO | Azioni e metadati possono essere uniformati senza equiparare il registro storico alla lista dei worklog correnti | P2 | 12, 17 |
| F-11 | `LoadingState` contiene solo uno spinner e altezza `h-64`; `ErrorState` stessa altezza, default inglesi e pulsante proprio | Mancano nel componente loading testo/stato accessibile; la stessa altezza è imposta a pagina e dialogo. Impatto visuale da misurare | P1 | 13 |
| F-12 | Stati vuoti Timesheet distinguibili ma solo testuali; `load()` imposta `grid = null` e `loading = true`, smontando `TimesheetGrid` | Recupero poco guidato; stato locale dei gruppi collassati perso al rimontaggio. Corretto invece nascondere dati non più affidabili | P1/P2 | 14, 15 |

### Riproduzione F-01

Eseguita con Node sui medesimi passaggi di conversione dei due componenti:

```js
const value = '2026-10-21';
for (const timezone of ['Europe/Rome', 'America/Los_Angeles']) {
  process.env.TZ = timezone;
  const date = new Date(value);
  console.log(timezone, date.toLocaleDateString('it-IT'), date.getDate());
}
// Europe/Rome: 21/10/2026, 21
// America/Los_Angeles: 20/10/2026, 20
```

Questa prova conferma il difetto della conversione; non sostituisce la verifica del picker renderizzato.

## 4. Matrice dei componenti e delle migrazioni

I nomi dei nuovi eventuali componenti saranno scelti in implementazione. Le famiglie sotto sono contratti di riuso, non una richiesta di introdurre tutte nuove astrazioni.

| Famiglia / fonte attuale | Uso Timesheet | Consumer esterni verificati | Decisione e criterio di chiusura |
| --- | --- | --- | --- |
| [Calendar](../../UI/src/lib/components/ui/calendar/calendar.svelte), [RangeCalendar](../../UI/src/lib/components/ui/range-calendar/range-calendar.svelte), [DatePickerPopover](../../UI/src/lib/components/shared/DatePickerPopover.svelte), [DateRangePickerPopover](../../UI/src/lib/components/shared/DateRangePickerPopover.svelte) | [TimesheetPeriodPicker](../../UI/src/lib/features/timesheets/TimesheetPeriodPicker.svelte); date del dialogo Efforts | Pagina progetti; creazione e pagina issue; creazione/modifica/pagina cicli; Insights/Explorer via [AnalyticsDateRangePicker](../../UI/src/lib/features/analytics/AnalyticsDateRangePicker.svelte) | Stesso parsing di date, trigger, focus, clear e stati visuali per variante; migrare i wrapper esistenti. Preset Analytics e periodo Timesheet restano adapter di dominio. US-01–03, 11 |
| [DueDatePickerPanel](../../UI/src/lib/components/shared/DueDatePickerPanel.svelte) | Nessun riuso diretto richiesto per i preset scadenza | Issue, azioni massive, menu contestuale, Inbox, tramite wrapper o uso diretto | Mantenere scadenze recenti e preset. Assorbire le correzioni alle primitive comuni e verificarne ogni punto d'ingresso; non trasferire «domani» ai worklog. US-02, 16 |
| [Select](../../UI/src/lib/components/ui/select/index.ts), [TimesheetFilter](../../UI/src/lib/features/timesheets/TimesheetFilter.svelte), [FilterBuilder](../../UI/src/lib/components/shared/FilterBuilder.svelte) | Workspace/team/progetto/persona; destinazione e titolare | Filtri di My issues, pagina team e viste salvate; selettori Insights; titolare in Efforts issue | Una composizione comune per ogni variante single/multiple; ricerca e riepilogo condivisi dove applicabili. I formati filtro restano negli adapter. US-04–06 |
| [IssuePickerDialog](../../UI/src/lib/features/issues/IssuePickerDialog.svelte) e [TimesheetWorklogDialog](../../UI/src/lib/features/timesheets/TimesheetWorklogDialog.svelte) | Ricerca paginata di issue su cui registrare lavoro | Relazioni/parent dalla pagina issue, `BulkActionBar`, `IssueContextMenu` | Condividere shell dialogo, campo ricerca e riga minima di risultato; conservare provider, esclusioni e paginazione specifici. Migrare entrambi i flussi. US-10 |
| [EffortWorklogDialog](../../UI/src/lib/features/issues/EffortWorklogDialog.svelte), [EffortDurationInput](../../UI/src/lib/features/issues/EffortDurationInput.svelte) | Creazione, modifica, eliminazione | [EffortWorklogs](../../UI/src/lib/features/issues/EffortWorklogs.svelte), pagina issue e pannello laterale; durata anche in planning | Continuare a mantenere un solo form e parser. Stesse etichette, validazioni e stati nei diversi ingressi. US-06, 11 |
| `Button`, `DropdownMenu`, `ContextMenu`, `Dialog`, `AlertDialog` | Comandi della griglia e del dettaglio | Azioni Efforts; menu della pagina issue e `IssueContextMenu` | Riutilizzare primitive esistenti e una presentazione comune per le azioni del worklog. Conferma Efforts unica. Non convertire i filtri in menu di comandi. US-09, 12 |
| [LoadingState](../../UI/src/lib/components/shared/LoadingState.svelte), [ErrorState](../../UI/src/lib/components/shared/ErrorState.svelte) | Pagina, dettaglio, scelta issue | Insights/Explorer; elenco/dettaglio macchine; impostazioni dev machines | Correzioni nel componente condiviso, con varianti pagina/contenitore solo dove servono e default compatibili; verifica di tutti questi consumer. US-13 |
| `Tabs`, `TeamIcon`, `Pagination`, token `app.css` | Modalità, identità team, controlli pagina, layout | Insights, schermate issue e altre viste che importano le primitive | Conservare stile e semantica comuni; riusare la paginazione esistente se risponde al contratto, senza nuovo pager Timesheet duplicato. US-07, 08, 12, 16 |

**Regola di manutenzione:** ogni PR su una famiglia comune elenca consumer individuati con ricerca degli import, varianti ammesse, screenshot comparativi e regressioni eseguite. Una famiglia non è completata se Timesheet usa la nuova versione e un consumer equivalente resta su una copia precedente. Eccezioni di dominio devono essere esplicite e non duplicare la presentazione comune.

## 5. Epic e storie

Tutte le storie sono **da fare**. Priorità relative al presente intervento; assegnatari, stime e sprint saranno definiti dal team. I riferimenti AC sono quelli del requisito TS-01; i criteri sotto aggiungono l'accettazione UX/UI.

| Epic | Risultato | Storie | Ordine consigliato |
| --- | --- | --- | --- |
| TSUX-EP-01 | Date affidabili e calendari coerenti | 01–03 | 1 |
| TSUX-EP-02 | Selettori e filtri comprensibili | 04–06 | 2 |
| TSUX-EP-03 | Report leggibile e azioni riconoscibili | 07–09 | 3 |
| TSUX-EP-04 | Registrazione e dettaglio uniformi con Efforts | 10–12 | 4 |
| TSUX-EP-05 | Stati e aggiornamenti che guidano l'utente | 13–15 | 3–4; US-13 può precedere le altre |
| TSUX-EP-06 | Coerenza verificata e mantenibile | 16–18 | Trasversale; accettazione alla fine |

### TSUX-EP-01 — Date affidabili e calendari coerenti

**Valore:** scegliere e riconoscere la stessa data in ogni schermata, mantenendo il significato dei diversi periodi.

#### TSUX-US-01 — Correggere le date di calendario nei picker condivisi

**Come** utente che cambia fuso del dispositivo, **voglio** vedere sempre la data salvata, **per** non selezionare o leggere il giorno sbagliato.

**Priorità:** P1. **Evidenza:** F-01. **Dipendenze:** nessuna. **Ambito:** date picker comuni, relativi consumer e helper Timesheet da valutare per riuso.

**Criteri di accettazione:**

- Dato `2026-10-21`, trigger, calendario e callback mantengono il 21 ottobre in Europe/Rome, America/Los_Angeles e Pacific/Kiritimati.
- Parsing/serializzazione delle date senza ora sono condivisi e non passano da un istante UTC interpretato nel fuso locale. I timestamp reali non vengono convertiti con questo helper.
- Valore assente, data non valida, 29 febbraio e confini anno restano gestiti secondo il contratto del campo; nessun salvataggio implicito durante il rendering.
- Progetti, issue, cicli e Analytics utilizzano la stessa soluzione per le date senza ora. Periodo e oggi Timesheet continuano a usare il riferimento e le capacità server.

**Verifica:** test di regressione che riproduce F-01 prima della correzione, suite periodi e prove browser sui consumer. **Tracciabilità:** TS-AC-02, 26–27.

#### TSUX-US-02 — Uniformare trigger, clear e interazione dei date picker

**Come** utente, **voglio** aprire, scegliere e rimuovere una data nello stesso modo, **per** trasferire ciò che ho imparato fra le schermate.

**Priorità:** P1. **Evidenze:** F-02–03. **Dipendenze:** US-01. **Ambito:** matrice calendari, inclusi consumer indiretti di `DueDatePickerPanel`.

**Criteri di accettazione:**

- Riutilizzare `Calendar`/`RangeCalendar` e una composizione condivisa di trigger e popover; stessa variante significa stessi token, altezza, icona, focus, hover e stato disabilitato.
- Il clear è un controllo separato, con nome accessibile tradotto, attivabile con Invio/Spazio e senza controlli interattivi annidati. Rimuove davvero il valore tramite callback.
- Nei campi obbligatori, come il periodo di un ciclo, il clear non è offerto se il dominio non ammette il valore vuoto. Nei campi opzionali il contratto di rimozione è esplicito.
- Apertura, Escape, selezione e chiusura restituiscono il focus correttamente. Il controllo consente anche l'inserimento della data da tastiera dove previsto dal form.
- Preset e vincoli appartengono agli adapter: scadenza, intervallo Analytics, periodo Timesheet e worklog mantengono le proprie regole. Il calendario resta contenuto nella viewport anche a 320px.

**Verifica:** test clear e focus, confronto dei consumer single/range nei temi chiaro/scuro e nelle lingue supportate. **Tracciabilità:** TS-AC-21, 27.

#### TSUX-US-03 — Rendere evidente il periodo selezionato

**Come** compilatore del timesheet, **voglio** riconoscere la settimana o il mese scelto prima di confermare, **per** non confondere il giorno cliccato con l'intervallo consultato.

**Priorità:** P1. **Evidenza:** F-03. **Dipendenze:** US-02.

**Criteri di accettazione:**

- Il calendario evidenzia l'intera settimana/mese risultante dal giorno selezionato; l'intervallo è anche descritto testualmente per chi non percepisce il colore.
- Il trigger presenta un'etichetta breve e non ambigua, per esempio «19–25 ott 2026» o «Ottobre 2026», con nome accessibile completo.
- Settimana/mese, precedente/successivo e «Periodo corrente» restano controlli distinti. Lo stato attivo non dipende solo dal colore.
- Mantiene i calcoli attuali e il riferimento del workspace; niente intervalli arbitrari o preset 30/90/180 giorni. Cambiare filtri non sposta il periodo.

**Verifica:** tastiera, cambio anno, bisestile, DST e workspace con oggi diverso; focus dopo caricamento. **Tracciabilità:** TS-AC-02, 26–28.

### TSUX-EP-02 — Selettori e filtri comprensibili

**Valore:** capire e modificare l'ambito del report senza riaprire ogni menu.

#### TSUX-US-04 — Condividere la multiselezione con ricerca e riepilogo

**Come** utente con molti team o progetti, **voglio** cercare le opzioni e vedere cosa ho selezionato, **per** restringere il report rapidamente.

**Priorità:** P1. **Evidenza:** F-04. **Dipendenze:** nessuna.

**Criteri di accettazione:**

- Riutilizzare le primitive disponibili e condividere trigger, ricerca, opzione selezionata e stato «nessun risultato» fra `TimesheetFilter` e i controlli compatibili di `FilterBuilder`.
- Con una selezione il trigger mostra il nome; con più selezioni mostra un riepilogo leggibile e conteggio residuo. I nomi completi restano consultabili da tastiera e touch, senza dipendere dal solo hover.
- Ricerca per nome tra le opzioni già autorizzate; il testo di ricerca non modifica il filtro. Nessun risultato di ricerca non azzera la selezione.
- Workspace qualificati dal nome/slug; team/progetti omonimi e identità storiche restano distinguibili. Il componente di presentazione non contiene policy o chiamate API Timesheet.
- Adattare esplicitamente i valori dei consumer: `null`/`[]`/ID Timesheet e formati di `FilterBuilder` mantengono il proprio significato. «Tutti» non è sinonimo di «tutte le opzioni visibili nella ricerca».

**Verifica:** 100 opzioni di fixture, omonimi, tastiera, ricerca senza risultati, selezione vuota; regressione My issues/team/viste salvate. **Tracciabilità:** TS-AC-06, 09, 24, 27.

#### TSUX-US-05 — Mostrare filtri attivi e rimozioni esplicite

**Come** lettore del report, **voglio** vedere perché sto osservando quel sottoinsieme di ore, **per** correggere un filtro senza perdere il contesto.

**Priorità:** P1. **Evidenze:** F-04, F-12. **Dipendenze:** US-04.

**Criteri di accettazione:**

- Riepilogo dei filtri attivi con azione di rimozione per dimensione; una selezione esplicitamente vuota è visibile come «Nessuno selezionato».
- Rimuovere un filtro lo porta a «Tutti» soltanto dopo l'azione esplicita dell'utente. «Reimposta filtri» dichiara che ripristina l'ambito autorizzato della modalità corrente; non cambia periodo o modalità.
- Rimozione di workspace e cambio modalità mantengono la riconciliazione attuale; feedback breve spiega il cambiamento dei filtri dipendenti senza rivelare dati revocati.
- Ogni azione aggiorna URL e pagina dei gruppi; avanti/indietro ricostruiscono lo stesso riepilogo. Il testo cercato nel menu non diventa un filtro URL.

**Verifica:** filtri combinati, Tutti/lista vuota/Senza progetto, revoca, storico browser. **Tracciabilità:** TS-AC-10, 23–24, 28.

#### TSUX-US-06 — Uniformare selettori singoli e campi di destinazione

**Come** utente, **voglio** riconoscere lo stesso controllo per workspace e titolare, **per** non dover imparare interazioni diverse fra report e form.

**Priorità:** P1. **Evidenza:** F-05. **Dipendenze:** US-04 per le parti di presentazione effettivamente comuni.

**Criteri di accettazione:**

- Il workspace nel dialogo Timesheet e il titolare in `EffortWorklogDialog` usano la composizione condivisa dei selettori, con variante form coerente con `Select` e gli altri consumer.
- Label associata, placeholder, valore, errore, stato disabilitato e nome accessibile hanno lo stesso contratto. Non usare il placeholder come unica etichetta.
- La scelta del workspace resta esplicita; le opzioni includono solo destinazioni scrivibili. Non preselezionare silenziosamente il workspace dello slug o un titolare altrui.
- Il titolare è modificabile solo in creazione quando delegabile; in modifica resta un valore in sola lettura. Conservare feedback del fallback e validazioni correnti.
- Lo stesso miglioramento compare entrando dal Timesheet, dalla pagina issue e dal pannello laterale, senza copie del form.

**Verifica:** ruoli misti, delega non disponibile, form disabilitato, errore associato al campo, navigazione da tastiera. **Tracciabilità:** TS-AC-08, 14, 18, 23, 27.

### TSUX-EP-03 — Report leggibile e azioni riconoscibili

**Valore:** leggere il report prima dei dettagli di configurazione e agire sulle celle senza tentativi.

#### TSUX-US-07 — Riordinare toolbar e sintesi del report

**Come** utente, **voglio** vedere subito periodo, ambito e ore totali, **per** orientarmi prima di esplorare i dettagli.

**Priorità:** P1. **Evidenza:** F-06. **Dipendenze:** US-03, 05.

**Criteri di accettazione:**

- Gerarchia proposta: titolo/modalità → periodo e azione «Registra lavoro» → filtri e loro riepilogo → sintesi/totale → griglia. Conservare header, tab e token di Insights.
- Riassumere l'ambito in una riga con accesso esplicito ai dettagli workspace/fuso/capacità. Rendere sempre riconoscibili sola lettura e variazioni di ambito; il fuso di riferimento rimane visibile.
- Mostrare vicino alla griglia il totale completo ricevuto dal server, etichettato «Totale del periodo filtrato», con la nota relativa ai dati accessibili. Non introdurre KPI di produttività o ore mancanti.
- «Registra lavoro» resta disponibile secondo le capacità già previste, anche se non ci sono righe. È chiaro che la destinazione verrà scelta nel dialogo e può differire dal filtro corrente.
- Su mobile i controlli si dispongono in ordine leggibile e i filtri possono essere raccolti in un pannello con riepilogo; nessuna perdita di selezione o spostamento delle azioni fuori portata.

**Verifica:** screenshot prima/dopo a 1440px e 390px con uno e dieci workspace; totale identico alla risposta e al footer. **Tracciabilità:** TS-AC-01, 05, 14, 21, 23, 27.

#### TSUX-US-08 — Rendere la griglia consultabile su schermi stretti e lunghi report

**Come** utente, **voglio** mantenere leggibili riga, giorno e totale mentre scorro, **per** non perdere il riferimento delle ore.

**Priorità:** P1. **Evidenza:** F-07. **Dipendenze:** US-07.

**Criteri di accettazione:**

- Prima colonna adattiva: a 390px deve restare visibile almeno una colonna giorno completa, mantenendo leggibile l'identificativo issue. Titolo completo disponibile anche da tastiera/touch.
- Intestazioni giorni visibili durante lo scorrimento verticale; totale di riga fisso a destra solo dove lo spazio lo consente. Su mobile il totale generale resta raggiungibile nella sintesi, senza occupare tutta la griglia con colonne fisse.
- Scorrimento orizzontale confinato alla griglia e riconoscibile, con focus visibile; nessun overflow orizzontale dell'intera pagina. Non nascondere weekend o giorni vuoti per guadagnare spazio.
- Gerarchia di gruppi/issue distinguibile mediante testo, indentazione e peso tipografico; oggi/weekend non identificati soltanto dal colore. L'espansione non modifica i totali.
- Paginazione esplicita: gruppi visibili e totale completo non sono confusi. Mantenere tabella semantica; non introdurre un `role="grid"` senza implementarne l'intero modello di tastiera.

**Verifica:** settimana e mese di 31 giorni, gruppi profondi, nomi lunghi, più pagine, 320/390/768/1440px e zoom 200%. **Tracciabilità:** TS-AC-03, 05, 20–21, 25, 27.

#### TSUX-US-09 — Distinguere celle consultabili, inseribili e di sola lettura

**Come** compilatore, **voglio** capire dove posso inserire o consultare lavoro, **per** scegliere l'azione giusta senza provare ogni cella.

**Priorità:** P1. **Evidenza:** F-08. **Dipendenze:** US-08.

**Criteri di accettazione:**

- La cella vuota scrivibile ha un segno di aggiunta riconoscibile anche su touch e nome accessibile con azione, issue e data; la cella non scrivibile rimane un valore neutro.
- Una cella con ore indica la possibilità di aprire le registrazioni. Nessuna modifica diretta del totale della cella: correzioni sempre sul singolo worklog.
- Giorni futuri e capacità locali mantengono le regole attuali. Una spiegazione di sola lettura è disponibile senza tooltip obbligatorio o rivelazione di dati nascosti.
- Apertura e chiusura del dettaglio/form riportano il focus alla cella o al fallback già previsto se la cella è stata rimossa. L'area cliccabile non è limitata alla sola icona.

**Verifica:** mouse/tastiera/touch, celle vuote/con ore/future, guest, Efforts disabilitato e admin con titolare non delegabile. **Tracciabilità:** TS-AC-08, 13–15, 18, 21.

### TSUX-EP-04 — Registrazione e dettaglio uniformi con Efforts

**Valore:** arrivare alla issue corretta e compilare un solo form riconoscibile in tutto il prodotto.

#### TSUX-US-10 — Condividere il selettore di issue preservando i provider

**Come** utente, **voglio** cercare e selezionare un'issue nello stesso modo nei diversi flussi, **per** riconoscere risultati, stato di caricamento e azione di selezione.

**Priorità:** P1. **Evidenza:** F-09. **Dipendenze:** US-06, 13.

**Criteri di accettazione:**

- Estrarre il minimo necessario da `IssuePickerDialog` e Timesheet: shell basata su `Dialog`, ricerca, riga con identificativo/titolo e stati; nessuna seconda libreria di ricerca.
- Timesheet continua a chiamare `getTimesheetIssues` e a raggiungere tutte le pagine, incluse issue senza worklog. Non sostituirlo con `listIssues`, che ha un diverso contratto di accesso e dati.
- Relazioni e azioni massive mantengono provider, esclusioni, suggerimenti e comportamento di selezione propri; metadati non disponibili nel DTO Timesheet non vengono inventati o caricati senza necessità.
- Frecce, Invio, Escape, focus iniziale/di ritorno e resa dei risultati sono coerenti. Caricamento, errore con retry e ricerca vuota sono distinti e tradotti.
- Cambiando workspace, risultati e richieste precedenti vengono invalidati; resta visibile la destinazione scelta. Paginare non perde la query.

**Verifica:** gli ingressi della matrice, issue oltre la prima pagina, risultati omonimi, risposta tardiva e accesso revocato. **Tracciabilità:** TS-AC-09, 14, 19–21, 27.

#### TSUX-US-11 — Rifinire il form worklog comune e chiarire durata e periodo

**Come** compilatore, **voglio** capire destinatario, giorni e quantità prima del salvataggio, **per** evitare registrazioni corrette tecnicamente ma diverse da quelle volute.

**Priorità:** P1. **Evidenza:** F-03 e composizione attuale di `EffortWorklogDialog`. **Dipendenze:** US-02, 06.

**Criteri di accettazione:**

- Un unico `EffortWorklogDialog` per Timesheet e issue; date e selettori adottano i componenti condivisi, preservando digitazione, min/max e messaggi associati ai campi.
- Gerarchia leggibile: contesto workspace/issue/titolare → data o periodo → effort per giorno → descrizione → riepilogo e azioni. Data singola significa intervallo di un giorno, senza cambiare il payload attuale.
- Rendere espliciti `1 = 1d = 8h`, gli estremi inclusivi e l'opzione weekend. Il riepilogo esistente mostra chiaramente giorni × effort giornaliero e totale, con equivalente in ore/minuti utile al confronto con Timesheet.
- L'errore preserva la bozza e porta il focus al primo campo non valido o al messaggio generale appropriato. Salvataggio in corso, conflitto, riallineamento e retry restano distinguibili.
- Nessun cambio di parser, residuo, revisione o idempotenza; footer utilizzabile su viewport stretto e zoom, senza coprire i campi. Eventuali modifiche a `EffortDurationInput` sono verificate anche nel planning.

**Verifica:** input `1`, `1d`, `1h 30m`, periodo con weekend, futuro, limite giornaliero, risposta persa, conflitto e revoca da entrambi gli ingressi. **Tracciabilità:** TS-AC-14–19, 26–27.

#### TSUX-US-12 — Uniformare metadati e azioni del worklog

**Come** utente che controlla registrazioni, **voglio** distinguere ore, titolare e inseritore e trovare le stesse azioni, **per** correggere il record giusto.

**Priorità:** P2. **Evidenza:** F-10. **Dipendenze:** US-11, 13.

**Criteri di accettazione:**

- Nel dettaglio Timesheet dare precedenza a durata, data, titolare e descrizione; contesto issue/workspace nel riepilogo. Inseritore e differenze di fuso restano consultabili, senza ripetizioni inutili per ogni riga.
- Usare la stessa composizione di azioni worklog in Timesheet ed Efforts: etichette, ordine e trattamento dell'eliminazione coerenti. Se raccolte in un menu, usare `DropdownMenu` e un trigger con nome accessibile; nessun menu custom.
- Le azioni portano sempre al form/conferma Efforts già esistente. La timeline delle issue conserva eventi storici e azioni solo sull'ultima versione; non sostituirla con la lista Timesheet.
- Mantenere leggibili identità non disponibili e omonimi, evitando nomi storici inventati. Identificativi/workspace non accessibili non vengono esposti.
- Paginazione con numero pagina e totale registrazioni comprensibili; intestazione e azioni di chiusura raggiungibili con descrizioni lunghe. Usare le primitive di paginazione comuni dove compatibili.

**Verifica:** più di 50 registrazioni, descrizioni lunghe, titolare diverso dall'inseritore, eliminazione confermata/annullata e ritorno del focus. **Tracciabilità:** TS-AC-03, 13, 16, 20–21, 27.

### TSUX-EP-05 — Stati e aggiornamenti che guidano l'utente

**Valore:** capire se attendere, cambiare filtro o riprovare senza perdere fiducia nei totali.

#### TSUX-US-13 — Migliorare loading/error nei componenti comuni

**Come** utente, **voglio** sapere cosa sta caricando e come recuperare un errore, **per** non interpretare uno spinner o una schermata vuota come assenza di dati.

**Priorità:** P1. **Evidenza:** F-11. **Dipendenze:** nessuna.

**Criteri di accettazione:**

- Modificare `LoadingState` ed `ErrorState` condivisi: messaggi tradotti, nome/stato accessibile per il caricamento, azione retry con `Button`, focus visibile.
- Distinguere stato di pagina e stato compatto in un contenitore solo dove necessario; eliminare l'imposizione di 256px nei piccoli dialoghi senza cambiare arbitrariamente il layout delle altre pagine.
- Il contenitore interessato comunica `aria-busy`; aggiornamenti e errori sono annunciati senza ripetere tutto il report. Nessun caricamento rappresentato con zero ore.
- Timesheet, Insights/Explorer, elenco/dettaglio macchine e impostazioni dev machines adottano e verificano il contratto comune. I messaggi specifici continuano a provenire dal consumer.

**Verifica:** risposta lenta, errore e retry nei consumer elencati; screen reader, lingua italiana/inglese e temi chiaro/scuro. **Tracciabilità:** TS-AC-19, 21, 27.

#### TSUX-US-14 — Dare un'azione utile a ogni stato vuoto

**Come** utente senza risultati, **voglio** capire il motivo e la prossima azione disponibile, **per** completare il compito senza tentativi casuali.

**Priorità:** P1. **Evidenza:** F-12. **Dipendenze:** US-05, 07, 13.

**Criteri di accettazione:**

- Nessun lavoro nel periodo: spiegazione neutra e «Registra lavoro» soltanto dove consentito; nessuna indicazione di ore mancanti.
- Nessun risultato con i filtri: riepilogo dell'ambito e azione esplicita per modificare/rimuovere i filtri. Workspace selezionati vuoti: invito alla selezione, senza ampliamento automatico a Tutti.
- Nessun workspace accessibile, sola lettura, errore tecnico e URL non valido mantengono messaggi e recuperi distinti. Non suggerire di creare nuovi workspace o chiedere ruoli quando non è necessario al compito.
- Guest e workspace con Efforts disabilitato non ricevono CTA di scrittura irrealizzabili. Revoca: messaggio generale senza nominare risorse non più accessibili.
- Nessun nuovo totale viene esposto in caso di errore; zero resta un risultato valido soltanto dopo una risposta completa autorizzata.

**Verifica:** fixture di tutti gli stati, più azione conseguente e cronologia URL. **Tracciabilità:** TS-AC-08–10, 18, 21, 23–24.

#### TSUX-US-15 — Conservare l'orientamento durante gli aggiornamenti

**Come** utente che consulta un report lungo, **voglio** ritrovare posizione e gruppi espansi dopo un aggiornamento, **per** continuare a lavorare sullo stesso punto.

**Priorità:** P2. **Evidenza:** F-12. **Dipendenze:** US-08, 13.

**Criteri di accettazione:**

- Conservare solo stato di presentazione, come gruppi espansi, scorrimento e destinazione del focus, quando identità di utente/modalità/ambito/periodo resta la stessa e le chiavi esistono ancora nella nuova risposta.
- Durante l'invalidazione non mantenere visibili righe o totali che potrebbero essere revocati: stabilizzare il contenitore con uno stato neutro. Mai usare dati vecchi per nascondere un errore.
- Distinguere caricamento iniziale, aggiornamento e riconnessione con feedback breve. Le bozze aperte non vengono sovrascritte e il focus non viene sottratto al campo in uso.
- Cambio utente/ambito e revoca puliscono lo stato incompatibile; focus su elemento ancora valido o fallback. Nessuna cache persistente di record, titoli o permessi per conservare lo scroll.

**Verifica:** aggiornamenti remoti e reconnect in A+B, gruppo collassato, scroll mensile, dettaglio aperto, risposta tardiva e revoca. **Tracciabilità:** TS-AC-19, 21, 28.

### TSUX-EP-06 — Coerenza verificata e mantenibile

**Valore:** consegnare miglioramenti uniformi nel prodotto e impedire che il riuso si riduca a somiglianza visiva temporanea.

#### TSUX-US-16 — Verificare la migrazione di tutti i consumer condivisi

**Come** maintainer, **voglio** una mappa verificata degli utilizzi e delle varianti, **per** correggere una sola implementazione senza lasciare versioni divergenti.

**Priorità:** P1, criterio trasversale di consegna. **Dipendenze:** mappa iniziale già in sezione 4; chiusura dopo US-01–15 effettivamente incluse nel rilascio.

**Criteri di accettazione:**

- Aggiornare la matrice tramite ricerca degli import dopo ogni estrazione; includere consumer indiretti, wrapper e stati presenti nelle pagine, non solo l'import Timesheet.
- Ogni variante condivisa ha contratto di valore, callback, focus, disabilitazione, errore e dimensione documentati nel punto comune. Le differenze sono motivate dal dominio.
- Stesso scenario e variante producono gli stessi token e comportamenti nei consumer. Niente override CSS locali per ricreare la vecchia versione e niente duplicazione del markup appena estratto.
- Allegare confronti visivi Timesheet/consumer esterni e prove funzionali pertinenti alla PR. Non basta una ricerca testuale degli import per dichiarare la migrazione riuscita.
- Nessuna nuova infrastruttura di catalogo obbligatoria: utilizzare documentazione e strumenti di test già presenti. Rimuovere soltanto duplicati resi inutilizzati dalla migrazione.

**Verifica:** revisione diff, matrice completa e regressioni dei consumer. **Tracciabilità:** TS-AC-27 e vincolo di coerenza della richiesta.

#### TSUX-US-17 — Completare accessibilità e localizzazione dei flussi modificati

**Come** utente con tastiera, tecnologie assistive o lingua diversa, **voglio** informazioni e azioni equivalenti, **per** completare il medesimo percorso.

**Priorità:** P1 per difetti nei controlli modificati. **Dipendenze:** si applica durante tutte le storie, chiusura sulla composizione integrata.

**Criteri di accettazione:**

- Percorso completo solo tastiera, focus visibile, nessun controllo annidato o trappola; dialoghi con titolo/descrizione, chiusura e ripristino focus, compresi passaggi dettaglio → modifica → dettaglio.
- Nomi accessibili di cella, calendario, ricerca, clear e menu comprensibili nel contesto; stati e validazioni annunciati senza sovraccarico. Verificare tabella e intestazioni con uno screen reader reale.
- Stringhe nuove/modificate in `it`, `en`, `zh-CN`, `zh-TW`; rimuovere i default inglesi dai componenti toccati. Date mostrate secondo locale, conservando ISO soltanto nei contratti dati o dove esplicitamente utile.
- Area attiva dei comandi compatta ma utilizzabile: obiettivo di prodotto minimo 32×32px per comandi isolati e 44×44px per azioni principali su touch; verificare collisioni, spaziatura e zoom. Non attribuire questi obiettivi a una certificazione.
- Contrasto, selezione, oggi, weekend, errori e sola lettura verificati nei temi chiaro/scuro; il colore non è l'unico segnale. Nessuna dichiarazione di piena conformità basata soltanto su screenshot o test automatici.

**Verifica:** test automatici mirati più sessione manuale tastiera/screen reader, zoom 200% e quattro lingue. **Tracciabilità:** TS-AC-21, 27–28.

#### TSUX-US-18 — Accettare il miglioramento sul percorso completo

**Come** responsabile del prodotto, **voglio** evidenze del percorso e delle regressioni, **per** verificare sia il miglioramento UX sia la conservazione del comportamento Efforts.

**Priorità:** P1, gate del rilascio. **Dipendenze:** storie incluse nel rilascio, US-16–17; le P2 rinviate restano esplicitamente aperte.

**Criteri di accettazione:**

- Acquisire prima/dopo dei sei passi in sezione 2 con stesso dataset, viewport, tema e stato; aprire e controllare ogni screenshot. Includere almeno un consumer esterno per ogni famiglia cambiata e verificare tutti gli ingressi della matrice.
- Eseguire la matrice di regressione della sezione 7, registrando comando, ambiente, risultato e limiti. Le fixture browser non sostituiscono prove di autorizzazione del backend.
- Far eseguire i tre compiti: individuare il totale di un progetto/persona nel periodo; registrare lavoro su una issue assente dalla griglia; correggere una registrazione e ritornare allo stesso report.
- Raccogliere riuscita, selezioni errate e passi necessari prima/dopo su dati comparabili. Non inventare percentuali di miglioramento senza una baseline osservata; un errore di data, destinatario o interpretazione della durata impedisce l'accettazione del percorso interessato.
- Il report finale distingue storie completate, rinviate e bloccate, con collegamento alle evidenze; nessuna storia condivisa è chiusa lasciando consumer equivalenti non migrati.

**Verifica:** accettazione end-to-end e revisione UX con evidenze. **Tracciabilità:** TS-AC-01–28 per gli invarianti interessati; focus su 14–21, 23–28.

## 6. Sequenza di consegna e dipendenze

1. **Fondazioni:** US-01, 02 e 13; impostare la matrice US-16 e i criteri US-17 prima di cambiare i controlli. Ogni fix comune include già i consumer interessati.
2. **Periodo e selezione:** US-03, 04, 05, 06. Definire e verificare i contratti di filtro senza modificare API o semantica Tutti/vuoto.
3. **Percorso principale:** US-07, 08, 09, 10, 11, 14, rispettandone le dipendenze. Prima verifica integrata con US-16–18.
4. **Rifiniture P2:** US-12 e 15, poi ripetere soltanto le verifiche interessate e aggiornare l'accettazione integrata.

Non sono assegnati tempi o story point senza refinement. US-16–18 sono criteri continui, non una fase finale in cui rimandare tutte le verifiche. Il primo rilascio può escludere le P2, ma non una migrazione necessaria alla coerenza dei componenti già modificati.

## 7. Verifica prevista e Definition of Done

| Area | Scenari minimi | Evidenza richiesta |
| --- | --- | --- |
| Date e calendari | Fusi Rome/Los_Angeles/Kiritimati, cambio anno, bisestile, DST, futuro locale, clear consentito/non consentito | Test helper, browser picker e confronti fra consumer |
| Filtri | Tutti, lista vuota, Senza progetto, omonimi, 100 opzioni, rimozione filtri figli, URL avanti/indietro | Unit selezione/navigazione ed E2E sui flussi |
| Ruoli e ambito | Admin in A/member in B/guest in C, Efforts disabilitato, revoca a dialogo aperto, identità storica | E2E UI; regressioni HTTP/PAT esistenti se si toccano accesso, provider o payload |
| Report | Cinque modalità, settimana/mese, gerarchie, più pagine, totali completi e cross workspace oltre 24h | Confronto con fixture note, regressioni griglia/dettaglio |
| Scritture | Issue non in griglia, delega, periodo/weekend, correzione giorno/durata, eliminazione, conflitto e risposta persa | Suite Timesheet worklogs ed Efforts, stesso comportamento da issue e Timesheet |
| Aggiornamenti | Evento remoto in B da pagina A, reconnect, errore di un ambito, risposta tardiva, bozza e focus preservati | E2E e suite live esistente quando si interviene su aggiornamenti |
| Presentazione | 320/390/768/1440px, chiaro/scuro, zoom 200%, nomi e descrizioni lunghi, quattro lingue | Screenshot ispezionati e prove manuali; nessun overflow della pagina |
| Componenti comuni | Tutti i consumer della matrice, comprese issue, cicli, progetti, Insights e macchine per gli stati condivisi | Inventario aggiornato, smoke test dei consumer e verifiche mirate sui controlli modificati |

Suite esistenti da estendere dove serve, senza test che controllino soltanto la presenza di markup:

- [Timesheet selezioni](../../UI/tests/unit/timesheets.test.mjs), [periodi](../../UI/tests/unit/timesheets-period.test.mjs), [navigazione](../../UI/tests/unit/timesheets-navigation.test.mjs), [sottoscrizioni](../../UI/tests/unit/timesheets-subscriptions.test.mjs).
- [Timesheet E2E](../../UI/tests/e2e/timesheets.test.ts), [Timesheet worklogs](../../UI/tests/e2e/timesheets-worklogs.test.ts), [Efforts worklogs](../../UI/tests/e2e/efforts-worklogs.test.ts), [planning](../../UI/tests/e2e/efforts-planning.test.ts), [settings](../../UI/tests/e2e/efforts-settings.test.ts), [Insights](../../UI/tests/e2e/insights.test.ts).
- [Suite live Timesheet](../../UI/tests/live/timesheets.mjs) e istruzioni backend/live in [acceptance-ep06](../requirements/timesheets/acceptance-ep06.md).

Per una modifica UI, dalla directory `UI`:

```bash
NODE_OPTIONS=--experimental-strip-types npm run test:unit
npm run check
npm run test:e2e -- tests/e2e/timesheets.test.ts tests/e2e/timesheets-worklogs.test.ts tests/e2e/insights.test.ts tests/e2e/efforts-settings.test.ts tests/e2e/efforts-planning.test.ts tests/e2e/efforts-worklogs.test.ts --workers=2
```

La configurazione E2E esegue anche la build. Aggiungere i casi sui consumer di calendari, selettori e menu modificati: le suite sopra non costituiscono da sole copertura di tutti quei punti d'ingresso. Backend e dati persistiti restano fuori dalla modifica prevista; un eventuale cambiamento a provider/payload richiede le regressioni specifiche già documentate.

**Una storia è Done quando:** criteri dimostrati, consumer migrati, traduzioni complete, controlli di tipo e test pertinenti superati, evidenze visuali e di interazione ispezionate, invarianti preservati e matrice aggiornata. «Implementato solo in Timesheet» non basta per una storia su un componente condiviso.

### Verifiche eseguite per questo documento

- **Superate:** quattro file di suite unitarie Timesheet, comando `NODE_OPTIONS=--experimental-strip-types node --test tests/unit/timesheets.test.mjs tests/unit/timesheets-period.test.mjs tests/unit/timesheets-navigation.test.mjs tests/unit/timesheets-subscriptions.test.mjs`.
- **Riprodotto:** slittamento di un giorno nella conversione F-01, con Node e due fusi.
- **Non eseguiti in questa attività:** build/check applicativo, E2E/browser, screenshot, prove backend/live e test con tecnologie assistive. Nessun risultato storico viene dichiarato rieseguito.
- **Consegna documentale:** verifica di collegamenti locali, ID, riferimenti alle storie e assenza di modifiche al codice.

## 8. Fuori perimetro e decisioni di refinement

Restano esclusi approvazioni e invio timesheet, blocco periodi, ferie/presenze, capacità e produttività, fatturazione, export, timer, drag-and-drop delle ore, modifica massiva dei totali, intervalli arbitrari, nuovi ruoli o nuove API di aggregazione. Non si aggiungono dipendenze UI o una nuova piattaforma di documentazione.

Prima dell'implementazione delle storie interessate, il refinement deve definire:

- Esatta disposizione mobile della toolbar e limite della prima colonna, verificando le schermate reali con dati lunghi; i criteri US-07/08 restano vincolanti.
- Modalità di inserimento testuale più calendario nei campi data condivisi e contratto di clear per i soli campi opzionali.
- Parti minime da condividere tra selettori e fra picker issue, confermando almeno i consumer reali della matrice; niente unificazione delle API autorizzative.
- Owner di manutenzione dei componenti comuni e formato delle evidenze nella PR, senza assegnare persone o scadenze in assenza di accordi.

## 9. Riferimenti al codice analizzato

- [Pagina Timesheet](../../UI/src/routes/[workspaceSlug]/timesheet/+page.svelte), [griglia](../../UI/src/lib/features/timesheets/TimesheetGrid.svelte), [dettaglio worklog](../../UI/src/lib/features/timesheets/TimesheetWorklogDetail.svelte), [selezioni](../../UI/src/lib/features/timesheets/selection.ts), [periodi](../../UI/src/lib/features/timesheets/period.ts), [navigazione](../../UI/src/lib/features/timesheets/navigation.ts).
- [Pagina Insights](../../UI/src/routes/[workspaceSlug]/insights/+page.svelte), [Explorer](../../UI/src/lib/features/analytics/InsightsExplorer.svelte), [pagina progetto](../../UI/src/routes/[workspaceSlug]/projects/[projectId]/+page.svelte), [pagina ciclo](../../UI/src/routes/[workspaceSlug]/teams/[teamId]/cycles/[cycleId]/+page.svelte), [creazione ciclo](../../UI/src/lib/features/cycles/CreateCycleDialog.svelte), [modifica ciclo](../../UI/src/lib/features/cycles/EditCycleDialog.svelte).
- [Pagina issue](../../UI/src/lib/features/issues/FullPageIssueView.svelte), [creazione issue](../../UI/src/lib/features/issues/CreateIssueDialog.svelte), [azioni massive](../../UI/src/lib/features/issues/BulkActionBar.svelte), [menu contestuale](../../UI/src/lib/features/issues/IssueContextMenu.svelte), [Inbox](../../UI/src/routes/[workspaceSlug]/inbox/+page.svelte).
- [My issues](../../UI/src/routes/[workspaceSlug]/my-issues/+page.svelte), [pagina team](../../UI/src/routes/[workspaceSlug]/teams/[teamId]/+page.svelte), [viste salvate](../../UI/src/routes/[workspaceSlug]/views/[viewId]/+page.svelte), [elenco macchine](../../UI/src/routes/[workspaceSlug]/machines/+page.svelte), [dettaglio macchina](../../UI/src/routes/[workspaceSlug]/machines/[machineId]/+page.svelte), [impostazioni macchine](../../UI/src/routes/[workspaceSlug]/settings/dev-machines/+page.svelte).

# Timesheets — Epic e Stories

Data: **2026-10-21**. Stato: **fondazioni EP-01, personale EP-02, aggregati EP-03, UI/navigazione EP-04 e operazioni EP-05 implementati; EP-06 pendente**.
Fonte: [TS-01 v0.2 — requisiti Timesheets](../requirements/timesheets/README.md).
Baseline analizzata: `feature/efforts`, commit `9007650`, incluse le modifiche
locali Efforts descritte nel requisito. Nessun ticket esterno creato.

## 1. Perimetro e regole di completamento

Il backlog comprende **6 Epic e 18 Stories**: 17 storie funzionali e una storia
tecnica di accettazione integrata. Gli ID `TS-EP-*` e `TS-US-*` sono locali a
questo documento e non collidono con quelli del backlog Efforts.

L'obiettivo è consultare worklog Efforts personali e amministrativi tra più
workspace e team, con quattro raggruppamenti amministrativi e una UI coerente
con Insights. Tutti gli elementi appartengono al perimetro della prima versione;
l'ordine sotto è una sequenza di implementazione, non una riduzione del requisito.

Ogni storia comprende autorizzazione server, gestione degli errori e verifiche
pertinenti al proprio ambito. Una UI corretta senza controllo API non completa
la storia. Durate e date restano quelle Efforts; non si duplicano worklog o
totali persistiti. Si riusano i comandi esistenti senza modificarne gli invarianti.

I criteri citati sono i **TS-AC-01–28** del requisito; quelli elencati in ciascuna
storia ne precisano l'applicazione. Stime, assegnatari e sprint sono da definire
dal team. Non sono state assegnate durate o story point senza refinement.

| Epic | Risultato | Stories |
| --- | --- | --- |
| TS-EP-01 | Accesso e selezione sicura di workspace/team | TS-US-01–03, TS-US-17 |
| TS-EP-02 | Timesheet personale e dettaglio dei worklog | TS-US-04–06 |
| TS-EP-03 | Aggregazioni e identità storiche | TS-US-07–10 |
| TS-EP-04 | Esperienza UI coerente con Insights | TS-US-11–12 |
| TS-EP-05 | Registrazione, correzione e configurazione Efforts | TS-US-13–15 |
| TS-EP-06 | Aggiornamenti e accettazione integrata | TS-US-16, TS-US-18 |

## 2. Epic e Stories

### TS-EP-01 — Accesso e filtri tra workspace e team

**Incremento EP-01 implementato (2026-10-20):** pagina navigabile e selettori,
endpoint `/api/timesheets/options`, autorizzazione server con allowlist PAT nel
contesto di dominio. Per scelta esplicita di perimetro, le query griglia,
aggregati e dettaglio sono rinviate alle Epic successive. TS-US-01–03 hanno
fondazioni e selettori verificabili, ma l'accettazione su contributi/totali e
integrazione dati resta pendente; TS-US-17 ed EP-01 restano **parziali**.
Vedere [contratti operativi e futuri](../requirements/timesheets/contracts.md)
e [verifica dell'incremento](../requirements/timesheets/acceptance-ep01.md).

**Valore:** vedere insieme i dati autorizzati, con un ambito comprensibile e
senza estendere i privilegi da un workspace a un altro.

#### TS-US-01 — Autorizzare ogni workspace separatamente

**Come** utente con ruoli diversi nei workspace, **voglio** che il timesheet
usi i miei permessi correnti in ciascuno, **per** consultare il lavoro consentito.

**Ambito e accettazione:**

- Calcolare sul server gli ambiti personale e amministrativo. Il primo comprende
  i propri worklog visibili in tutte le membership correnti; il secondo solo
  workspace con ruolo owner/admin, in qualunque raggruppamento.
- Admin in A/member in B: personale A+B, report altrui solo A. Non inserire
  silenziosamente le sole ore proprie di B nel report amministrativo.
- Applicare visibilità prima di somme, conteggi, dettagli e selettori; nessuna
  deroga per lead di progetto o `project:manage`.
- Rifiutare una selezione esplicita non autorizzata senza distinguere risorsa
  inesistente/inaccessibile; risolvere «Tutti» solo nell'ambito consentito.
- Verificare guest, non membri, pubblico, team privati e revoca concorrente alla
  lettura. Nessuna modifica alla policy dei worklog nelle viste issue esistenti.

**Requisiti:** TS-008, TS-022; sezione 3. **AC:** TS-AC-07–10, TS-AC-23.
**Dipendenze:** nessuna. **Verifica:** repository PostgreSQL e HTTP reali.

#### TS-US-02 — Selezionare uno o più workspace

**Come** utente, **voglio** selezionare uno, più o tutti i workspace ammessi,
**per** consultare il mio lavoro o gli aggregati nel perimetro scelto.

**Ambito e accettazione:**

- Prima apertura personale su tutti i workspace ammessi, indipendentemente
  dallo slug di navigazione; selettore con ambito effettivo esplicito.
- Passaggio a modalità amministrativa: conservare l'intersezione ammessa e
  segnalare il cambiamento; senza workspace amministrabili, non offrire la modalità.
- Aggiornare dati e filtri figli al cambio selezione, senza ampliare una
  selezione vuota a «Tutti». Rimuovere opzioni revocate prima di nuova lettura.
- Distinguere nessun workspace ammesso, nessun lavoro e errore di caricamento.

**Requisiti:** TS-001, TS-019, TS-022. **AC:** TS-AC-01, TS-AC-10, TS-AC-23–24.
**Dipendenze:** TS-US-01, TS-US-17. **Verifica:** API selettori e E2E con ruoli misti.

#### TS-US-03 — Filtrare team e progetti nell'ambito selezionato

**Come** utente, **voglio** restringere il timesheet a più team e progetti,
**per** isolare il lavoro di mio interesse anche tra workspace diversi.

**Ambito e accettazione:**

- Team selezionabili solo nei workspace scelti e secondo visibilità corrente;
  progetti qualificati dal workspace e filtro esplicito Senza progetto.
- OR tra valori della stessa dimensione, AND tra dimensioni; nessun filtro
  basato sul nome o sulla membership team del titolare.
- Rimuovere i soli filtri figli incompatibili con la nuova selezione workspace,
  indicando il cambiamento; non confondere «nessuno selezionato» con «tutti».
- Nomi uguali distinti per ID e workspace; i totali corrispondono ai medesimi
  filtri dei dettagli. Non mostrare opzioni o conteggi inaccessibili.

**Requisiti:** TS-006, TS-011, TS-019–020. **AC:** TS-AC-06, TS-AC-09, TS-AC-24–25.
**Dipendenze:** TS-US-02. **Verifica:** query con filtri combinati ed E2E.

#### TS-US-17 — Consultare Timesheet via API e PAT

**Come** utilizzatore delle API, **voglio** leggere il timesheet con un PAT
entro i miei permessi, **per** ottenere gli stessi risultati della UI.

**Ambito e accettazione:**

- Definire e documentare route/DTO per selettori, griglia, aggregati e dettaglio
  paginato; dati minimi di contesto entro il medesimo ambito Efforts autorizzato.
- Input con periodo, modalità, workspace/team/persona/progetto e riferimento;
  output con ambito effettivo, minuti, totali completi, paginazione e capacità
  locali necessarie alla UI. Rifiutare combinazioni e periodi non validi.
- Intersecare `efforts:read`, allowlist `workspace_slugs`, membership, ruolo
  e accesso per ogni workspace, anche quando la route non ha uno slug.
- Un PAT limitato ad A non legge B con «Tutti» o ID espliciti; scope mancanti,
  token revocati e downgrade negano i dati. Le scritture conservano anche
  `efforts:write`; selezione delega con i vincoli di sessione Efforts esistenti.
- Query autorizzate server-side, nessun download completo per sommare nel client;
  risposta coerente e nessun totale parziale in caso di errore.

**Requisiti:** TS-007–011, TS-019, TS-022; sezione 6.
**AC:** TS-AC-07, TS-AC-09–11, TS-AC-20–21, TS-AC-23.
**Dipendenze:** TS-US-01. **Verifica:** contratti, manifest route e test HTTP PAT.

### TS-EP-02 — Timesheet personale

**Incremento EP-02:** implementati calendario settimana/mese, griglia personale
cross workspace con totali completi e dettaglio paginato dei worklog, con
letture PostgreSQL/PAT autorizzate. Ritorno dall'issue tramite stato di sessione
della cronologia browser, per scelta esplicita; URL completo resta a TS-US-12.
Azioni di scrittura, aggregati e notifiche multi-workspace restano alle Epic
successive. Vedere [verifica EP-02](../requirements/timesheets/acceptance-ep02.md).
Le parti degli AC condivise con tali Epic restano pendenti: questo incremento
non dichiara completata l'intera TS-01 né tutte le letture TS-US-17.

**Valore:** ricostruire il lavoro svolto giorno per giorno, attraverso tutti i
workspace e team autorizzati, fino alla singola registrazione.

#### TS-US-04 — Scegliere settimana/mese e gestire i fusi

**Come** utente, **voglio** navigare un periodo comune, **per** confrontare le
ore dei workspace senza spostare le date registrate.

**Ambito e accettazione:**

- Settimana lunedì–domenica, mese di calendario, precedente/successivo e corrente;
  weekend e giorni vuoti visibili. Riferimento iniziale dal workspace di apertura.
- Cambiare filtri non sposta il periodo; mostrare il riferimento e il fuso/oggi
  del destinatario quando si registra lavoro. Riferimento revocato: sceglierne
  uno accessibile conservando le date già selezionate.
- Testare fine anno, bisestile, DST e fusi con oggi diverso. Non convertire
  `work_date` né usare i preset analytics basati sul browser.

**Requisiti:** TS-002–003, TS-023. **AC:** TS-AC-02, TS-AC-15, TS-AC-26–27.
**Dipendenze:** TS-US-17. **Verifica:** unit test calendari e test HTTP/E2E fusi.

#### TS-US-05 — Leggere la griglia del mio lavoro

**Come** utente, **voglio** una griglia giornaliera del mio lavoro,
**per** vedere totali e ripartizione per workspace, team, progetto e issue.

**Ambito e accettazione:**

- Righe workspace → team → progetto → issue; dati attribuiti al titolare, mai
  all'inseritore o agli assegnatari. Ore/minuti esatti, nessun obiettivo di 8h.
- Subtotali e totali completi anche con paginazione; sommare ogni worklog una
  volta senza aggiungere riepiloghi parent o eventi eliminati.
- Includere lavoro su issue chiuse/non assegnate/senza stima; gruppo Senza
  progetto e trattamento neutro di riferimenti non disponibili.
- Guest consulta il proprio eventuale storico in sola lettura. Dare stati
  distinti per periodo vuoto, filtri senza risultati e errore.

**Requisiti:** TS-001, TS-004–006, TS-017–018.
**AC:** TS-AC-01, TS-AC-03–04, TS-AC-08–09, TS-AC-12, TS-AC-20–21, TS-AC-25.
**Dipendenze:** TS-US-03, TS-US-04, TS-US-11.
**Verifica:** repository PostgreSQL con totali attesi ed E2E della griglia.

#### TS-US-06 — Aprire il dettaglio delle registrazioni

**Come** utente autorizzato, **voglio** aprire una cella e vedere i worklog,
**per** comprendere la somma e raggiungere l'issue corretta.

**Ambito e accettazione:**

- Esporre singoli worklog con contesto workspace/team/issue, titolare, inseritore,
  data, durata e descrizione; due worklog nello stesso giorno restano distinti.
- Collegare l'issue nel suo workspace, mantenendo il ritorno al timesheet filtrato.
- Dettaglio oltre 50 registrazioni interamente raggiungibile, ordinato stabilmente
  su dataset immutato; riepilogo e dettaglio usano lo stesso ambito autorizzato.
- La cella aggregata non è un record modificabile. Predisporre le azioni sui
  singoli worklog, abilitate solo con TS-US-13/14 e relativi controlli.

**Requisiti:** TS-005, TS-007, TS-018. **AC:** TS-AC-03, TS-AC-09, TS-AC-20–21, TS-AC-28.
**Dipendenze:** TS-US-05. **Verifica:** API dettaglio e E2E navigazione/paginazione.

### TS-EP-03 — Aggregazioni amministrative

**Incremento EP-03:** implementate TS-US-07–10: quattro gerarchie amministrative,
filtro persone, apertura del timesheet individuale nella stessa vista Per persona
e dettaglio per titolare. I contributi storici derivano dai record correnti,
comprese identità non disponibili e rimappate dall'import. Vedere
[verifica EP-03](../requirements/timesheets/acceptance-ep03.md).
Scritture, URL condivisibili e notifiche multi-workspace restano alle Epic dedicate.

**Valore:** leggere gli stessi worklog per persona, progetto, workspace o team,
senza cambiare totale né perdere la provenienza dei contributi.

#### TS-US-07 — Consultare per persona

**Come** owner/admin, **voglio** raggruppare il lavoro per persona tra gli ambiti
amministrabili, **per** aprirne il timesheet e analizzare i contributi.

**Ambito e accettazione:**

- Gerarchia persona → workspace → team → progetto → issue; totali giornalieri
  e del periodo, filtro persone combinabile con gli altri.
- Stesso ID account sommato tra workspace autorizzati; nomi uguali non uniscono
  account diversi. Delega attribuita al titolare.
- Aprire il timesheet altrui mantenendo filtri/periodo; nessun allargamento
  implicito ai workspace dove il chiamante è member/guest.

**Requisiti:** TS-008–009, TS-011, TS-022.
**AC:** TS-AC-05–07, TS-AC-13, TS-AC-23–25.
**Dipendenze:** TS-US-06. **Verifica:** aggregati PostgreSQL e flusso E2E.

#### TS-US-08 — Consultare per progetto

**Come** owner/admin, **voglio** raggruppare il lavoro per progetto,
**per** vedere contributi di persone e team senza duplicare i minuti.

**Ambito e accettazione:**

- Gerarchia progetto qualificato dal workspace → team → persona → issue;
  progetto corrente dell'issue, senza ereditarlo dal parent.
- Progetti omonimi distinti, Senza progetto separato per workspace. Progetto
  condiviso tra team ripartito secondo il team di ciascuna issue.
- Il passaggio da Per persona conserva filtri, periodo e totale generale;
  paginazione dei gruppi non riduce i totali.

**Requisiti:** TS-008, TS-010–011. **AC:** TS-AC-05–06, TS-AC-12, TS-AC-20, TS-AC-25.
**Dipendenze:** TS-US-07. **Verifica:** query e UI con progetti condivisi/omonimi.

#### TS-US-09 — Consultare per workspace e per team

**Come** owner/admin, **voglio** riepiloghi per workspace e team,
**per** confrontare la distribuzione del lavoro nei miei ambiti autorizzati.

**Ambito e accettazione:**

- Per workspace: workspace → team → progetto → persona → issue. Per team:
  team qualificato dal workspace → progetto → persona → issue.
- Stessi filtri, totale generale e dettaglio delle altre viste amministrative;
  team corrente dell'issue, non team del titolare o del progetto.
- Verificare l'esempio W1/W2 del requisito, team omonimi e cambio di team con
  immediata applicazione della nuova visibilità.

**Requisiti:** TS-008, TS-011, TS-020–022. **AC:** TS-AC-23–25.
**Dipendenze:** TS-US-08. **Verifica:** quattro raggruppamenti sullo stesso dataset.

#### TS-US-10 — Conservare attribuzioni e coerenza dello storico

**Come** owner/admin, **voglio** ritrovare il lavoro delle persone rimosse e
capire la classificazione corrente, **per** interpretare correttamente i totali.

**Ambito e accettazione:**

- Conservare contribuzioni di ex membri e account non disponibili per ID,
  senza inventare nomi storici o rendere nuovamente ammissibile la delega.
- Esplicitare uso del progetto/team attuale; spostamenti riclassificano date
  pregresse senza cambiare minuti, applicando le nuove regole di visibilità.
- Eliminare worklog/issue rimuove i contributi effettivamente eliminati, non i
  figli conservati; gli eventi storici non ricreano ore.
- Export/import ricostruisce i risultati secondo i dati importati e l'accesso
  di destinazione. Identità rimappate non si fondono per nome tra workspace.

**Requisiti:** TS-012, TS-020; sezioni 5–6.
**AC:** TS-AC-12–13, TS-AC-16, TS-AC-22, TS-AC-25.
**Dipendenze:** TS-US-09. **Verifica:** lifecycle e fixture import/storico.

### TS-EP-04 — UI e navigazione coerenti con Insights

**Incremento EP-04:** implementate TS-US-11–12: cinque tab con primitive Insights,
ripristino URL e cronologia con rivalidazione degli accessi, tastiera e gestione
del focus, resa mobile e azioni tradotte. Vedere
[verifica EP-04](../requirements/timesheets/acceptance-ep04.md).
Scritture e notifiche multi-workspace restano a EP-05/06.

**Valore:** una nuova vista riconoscibile nell'applicazione, con componenti
esistenti e comportamenti di navigazione verificabili.

#### TS-US-11 — Usare la struttura UI di Insights

**Come** utente di Insights, **voglio** ritrovare convenzioni familiari nel
Timesheet, **per** orientarmi tra modalità, filtri e risultati.

**Ambito e accettazione:**

- Riprendere header/SidebarToggle, toolbar, tab, spaziature, token, selettori,
  icone team, LoadingState ed ErrorState, come indicato nella matrice di riuso.
- Adattare i selettori alla scelta multipla; riusare primitive calendario e
  popover con settimana/mese e fuso esplicito. Nessuna copia dei preset 30/90/180.
- Griglia timesheet dedicata; nessun riuso delle metriche issue Insights come
  consuntivo. Estrarre componenti comuni solo se necessario al riuso effettivo.
- Verificare confronto visivo Insights/Timesheet, tema chiaro/scuro e viewport
  stretto senza modificare comportamenti Insights estranei alla richiesta.

**Requisiti:** TS-018, TS-024. **AC:** TS-AC-21, TS-AC-27.
**Dipendenze:** nessuna per struttura e riuso; integrazione dati con TS-US-02/04.
**Verifica:** check/build UI e confronto visivo, regressione Insights se toccato.

#### TS-US-12 — Ripristinare filtri e navigare con la tastiera

**Come** utente, **voglio** ritrovare la vista dall'URL e usarla con tastiera,
**per** consultare e confrontare periodi senza perdere contesto.

**Ambito e accettazione:**

- Parametri URL per modalità, periodo, riferimento e filtri; apertura e
  avanti/indietro ripristinano lo stato autorizzato corrente.
- Un URL non concede permessi; ambiti revocati vengono rimossi con feedback,
  senza mostrare dati vecchi. Il riferimento non più accessibile viene sostituito
  esplicitamente mantenendo il periodo.
- Intestazioni di tabella, etichette tradotte, focus visibile, selettori e
  dettagli utilizzabili da tastiera, focus restituito alla chiusura del dialogo.
- Scorrimento orizzontale della griglia mensile e dettaglio leggibile su mobile;
  link all'issue usa il workspace della riga, non quello dell'URL timesheet.

**Requisiti:** TS-005, TS-016, TS-018, TS-024.
**AC:** TS-AC-10, TS-AC-19, TS-AC-21, TS-AC-28.
**Dipendenze:** TS-US-06, TS-US-11. **Verifica:** E2E URL, storia browser e tastiera.

### TS-EP-05 — Operazioni Efforts nel workspace corretto

**Incremento EP-05:** implementate TS-US-13–15: selezione esplicita di workspace
e issue, anche senza lavoro nella griglia; celle vuote per registrare, dettaglio
con inserimento/correzione/eliminazione e riuso del dialogo Efforts. Capacità,
delega, data locale e revisione sono rilette nel workspace destinatario;
scritture, conflitti e retry conservano gli invarianti Efforts. Dopo salvataggio
si aggiornano griglia e dettaglio senza cambiare filtri o periodo. Vedere
[verifica EP-05](../requirements/timesheets/acceptance-ep05.md).
Aggiornamenti remoti multi-workspace e accettazione integrata restano EP-06.

**Valore:** registrare e correggere lavoro dal Timesheet usando le garanzie già
implementate in Efforts, anche quando la vista comprende molti workspace.

#### TS-US-13 — Registrare lavoro dalla griglia

**Come** utente abilitato, **voglio** registrare lavoro scegliendo workspace e
issue, **per** aggiornare il timesheet senza aprire ogni issue separatamente.

**Ambito e accettazione:**

- Selezione esplicita workspace/issue e precompilazione dalla cella; issue
  assente dalla griglia comunque selezionabile se accessibile.
- Riutilizzare EffortWorklogDialog, parser e comandi; titolare proprio oppure
  delegato solo se ruolo e destinatario lo consentono in quel workspace.
- Unità giorni da 480 minuti, giorni/periodi e weekend come Efforts attuale;
  data futura rispetto al destinatario rifiutata, limite per workspace invariato.
- Periodo atomico su una sola issue/workspace, retry idempotente, storico e
  residuo invariati nelle regole. Nessuna scrittura multipla tra workspace.

**Requisiti:** TS-003, TS-013, TS-017, TS-023.
**AC:** TS-AC-14–15, TS-AC-17, TS-AC-26.
**Dipendenze:** TS-US-06. **Verifica:** E2E, test HTTP e regressione periodi Efforts.

#### TS-US-14 — Correggere o eliminare un worklog

**Come** titolare o owner/admin autorizzato, **voglio** correggere una
registrazione dal dettaglio, **per** mantenere corretto il consuntivo.

**Ambito e accettazione:**

- Rileggere worklog/revisione nel workspace corretto; nessuna modifica del
  titolare o del totale della cella. Guest e utenti non autorizzati non scrivono.
- Cambio data/durata aggiorna giorni interessati senza ricalcolare residuo;
  eliminazione con conferma mantiene lo storico senza contribuire ai totali.
- Conflitto esplicito, retry senza duplicati, bozza preservata; azioni rifiutate
  se ruolo/accesso sono cambiati tra apertura e invio.

**Requisiti:** TS-014. **AC:** TS-AC-08, TS-AC-13, TS-AC-16–17.
**Dipendenze:** TS-US-13. **Verifica:** E2E con revisioni e regressione CRUD Efforts.

#### TS-US-15 — Rispettare configurazione e sola lettura per workspace

**Come** utente che lavora in più workspace, **voglio** distinguere dove posso
registrare lavoro, **per** operare secondo configurazione e ruolo locali.

**Ambito e accettazione:**

- Efforts disabilitato in A mantiene letture e blocca scritture in A; non
  disabilita B. La UI espone le capacità locali e non usa lo stato globale
  del solo workspace di navigazione.
- Rileggere oggi/fuso/configurazione del destinatario; un form già aperto non
  aggira disattivazione, downgrade, revoca o limite giornaliero.
- Superare 24h aggregate tra workspace non crea un nuovo errore globale;
  restano i limiti separati e i controlli sui worklog nascosti.

**Requisiti:** TS-015, TS-023. **AC:** TS-AC-15, TS-AC-18, TS-AC-26.
**Dipendenze:** TS-US-13. **Verifica:** HTTP/E2E con stati e fusi diversi.

### TS-EP-06 — Aggiornamenti e accettazione

**Valore:** mantenere coerenti dati e permessi di una vista trasversale e
verificare il risultato completo prima di dichiarare la funzionalità pronta.

#### TS-US-16 — Ricevere aggiornamenti da tutti gli ambiti selezionati

**Come** utente, **voglio** che modifiche e revoche nei workspace selezionati
aggiornino la vista, **per** consultare risultati coerenti con l'accesso corrente.

**Ambito e accettazione:**

- Coprire refresh remoto anche di B mentre la pagina è aperta dallo slug A;
  il socket attuale del layout, da solo, non soddisfa la storia.
- Definire sottoscrizione autorizzata, cleanup al cambio selezione, invalidazione
  e reconnect; nessun payload indiscriminato con ore o identità private.
- Isolare cache e risposte per utente/modalità/ambito/periodo/filtri, invalidare
  il precedente insieme e impedire risposte tardive; non sostituire bozze aperte.
- Errore in parte dell'ambito: non pubblicare un totale completo apparente.
  Revoca: rimuovere i dati e riallineare la selezione con feedback prima del retry.

**Requisiti:** TS-016, TS-018, TS-022. **AC:** TS-AC-09–10, TS-AC-19, TS-AC-21, TS-AC-28.
**Dipendenze:** TS-US-09, TS-US-12, TS-US-14–15.
**Verifica:** E2E multi-sessione, reconnect, revoche e risposte fuori ordine.

#### TS-US-18 — Accettazione integrata e misure delle query

**Tipo:** storia tecnica di verifica, non una funzionalità utente aggiuntiva.

**Risultato e completamento:**

- Registrare evidenza riproducibile per tutti i 28 AC con test repository
  PostgreSQL, HTTP/PAT, unit UI ed E2E; non sostituire i controlli backend con mock UI.
- Dataset con A admin/B member/C guest, workspace fuori allowlist, team privati,
  fusi differenti, omonimi, persone rimosse, gerarchie, progetto tra team,
  worklog delegati e oltre 50 registrazioni con più pagine di gruppi.
- Verificare totali delle quattro aggregazioni e dettaglio sullo stesso snapshot;
  controllare overflow, ordinamento e assenza di duplicati su dataset immutato.
- Misurare settimana/mese personale e aggregati su più workspace; documentare
  dataset, ambiente, tempi e piani EXPLAIN. Aggiungere indici solo se motivati;
  evitare caricamento totale nel browser e query per ogni cella/issue.
- Eseguire regressioni Efforts pertinenti e Insights se coinvolto nel riuso;
  verificare portabilità v1/v2 e lifecycle, accessibilità e resa visiva.
- Pubblicare una nota di accettazione con risultati e limiti effettivi, senza
  dichiarare già eseguite prove o promettere SLA non misurati.

**Requisiti:** TS-001–024 e invarianti delle sezioni 3/5/6.
**AC:** TS-AC-01–28. **Dipendenze:** tutte le altre Stories completate.
**Verifica:** report finale con comandi, esiti, dataset e riferimenti alle suite.

## 3. Tracciabilità requisiti → Stories

TS-US-18 verifica trasversalmente tutti i requisiti; la tabella identifica le
storie che ne realizzano il comportamento.

| Requisito | Stories |
| --- | --- |
| TS-001 | TS-US-02, TS-US-05 |
| TS-002 | TS-US-04 |
| TS-003 | TS-US-04, TS-US-13 |
| TS-004 | TS-US-05 |
| TS-005 | TS-US-05, TS-US-06, TS-US-12 |
| TS-006 | TS-US-03, TS-US-05 |
| TS-007 | TS-US-06, TS-US-17 |
| TS-008 | TS-US-01, TS-US-07–09, TS-US-17 |
| TS-009 | TS-US-07, TS-US-17 |
| TS-010 | TS-US-08, TS-US-17 |
| TS-011 | TS-US-03, TS-US-07–09, TS-US-17 |
| TS-012 | TS-US-10 |
| TS-013 | TS-US-13 |
| TS-014 | TS-US-14 |
| TS-015 | TS-US-15 |
| TS-016 | TS-US-12, TS-US-16 |
| TS-017 | TS-US-05, TS-US-13 |
| TS-018 | TS-US-05–06, TS-US-11–12, TS-US-16 |
| TS-019 | TS-US-02–03, TS-US-17 |
| TS-020 | TS-US-03, TS-US-09–10 |
| TS-021 | TS-US-09 |
| TS-022 | TS-US-01–02, TS-US-07, TS-US-09, TS-US-16–17 |
| TS-023 | TS-US-04, TS-US-13, TS-US-15 |
| TS-024 | TS-US-11–12 |

## 4. Tracciabilità criteri → Stories

Ogni AC è verificato anche in TS-US-18. Nessun AC è rinviato soltanto alla
verifica finale: le storie riportate devono già soddisfarlo nel proprio ambito.

| Criterio | Stories |
| --- | --- |
| TS-AC-01 | TS-US-02, TS-US-05 |
| TS-AC-02 | TS-US-04 |
| TS-AC-03 | TS-US-05–06 |
| TS-AC-04 | TS-US-05 |
| TS-AC-05 | TS-US-07–08 |
| TS-AC-06 | TS-US-03, TS-US-07–08 |
| TS-AC-07 | TS-US-01, TS-US-07, TS-US-17 |
| TS-AC-08 | TS-US-01, TS-US-05, TS-US-14 |
| TS-AC-09 | TS-US-01, TS-US-03, TS-US-05–06, TS-US-16–17 |
| TS-AC-10 | TS-US-01–02, TS-US-12, TS-US-16–17 |
| TS-AC-11 | TS-US-17 |
| TS-AC-12 | TS-US-05, TS-US-08, TS-US-10 |
| TS-AC-13 | TS-US-07, TS-US-10, TS-US-14 |
| TS-AC-14 | TS-US-13 |
| TS-AC-15 | TS-US-04, TS-US-13, TS-US-15 |
| TS-AC-16 | TS-US-10, TS-US-14 |
| TS-AC-17 | TS-US-13–14 |
| TS-AC-18 | TS-US-15 |
| TS-AC-19 | TS-US-12, TS-US-16 |
| TS-AC-20 | TS-US-05–06, TS-US-08, TS-US-17 |
| TS-AC-21 | TS-US-05–06, TS-US-11–12, TS-US-16–17 |
| TS-AC-22 | TS-US-10 |
| TS-AC-23 | TS-US-01–02, TS-US-07, TS-US-09, TS-US-17 |
| TS-AC-24 | TS-US-02–03, TS-US-07, TS-US-09 |
| TS-AC-25 | TS-US-03, TS-US-05, TS-US-07–10 |
| TS-AC-26 | TS-US-04, TS-US-13, TS-US-15 |
| TS-AC-27 | TS-US-04, TS-US-11 |
| TS-AC-28 | TS-US-06, TS-US-12, TS-US-16 |

## 5. Sequenza di implementazione

1. **Confini e contratti:** TS-US-01 → TS-US-17. Verificare prima autorizzazione
   cross workspace, allowlist PAT, DTO, paginazione e totali coerenti.
2. **Selezione e struttura:** TS-US-02 → TS-US-03; TS-US-04 e TS-US-11 preparano
   periodo e UI. Verificare filtri multipli, riferimenti temporali e riuso Insights.
3. **Personale:** TS-US-05 → TS-US-06 → TS-US-12. Verificare griglia, dettaglio,
   navigazione URL, stati vuoti/errori e tastiera.
4. **Aggregati:** TS-US-07 → TS-US-08 → TS-US-09 → TS-US-10. Verificare quattro
   raggruppamenti, identità storiche e invarianza dei totali.
5. **Operazioni:** TS-US-13 → TS-US-14/15. Verificare scritture sul workspace
   corretto, fusi, configurazioni locali, retry e conflitti.
6. **Integrazione:** TS-US-16 → TS-US-18. Verificare aggiornamenti da tutti gli
   ambiti, revoche, performance e copertura finale degli AC.

## 6. Decisioni tecniche da chiudere nel refinement

- TS-US-17: nomi endpoint/DTO, rappresentazione di «Tutti» e selezione vuota,
  ordinamento/paginazione dei gruppi e snapshot coerente senza successi parziali.
- TS-US-11/12: minimo riuso di componenti Insights, codifica URL e modalità
  accessibile di selezione multipla; nessuna riscrittura generale di analytics.
- TS-US-16: trasporto delle notifiche autorizzate per più workspace e cleanup;
  non presumere che il singolo socket del layout corrente copra l'intera vista.
- TS-US-18: volumi di prova e indici giustificati dalle misure; nessuno SLA o
  cambiamento del limite giornaliero globale aggiunto a questo requisito.

Le decisioni devono rispettare TS-01 v0.2 e sono attività del backlog, non
richieste di approvazione per la sola predisposizione di questi documenti.

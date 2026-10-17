# Efforts — Epic e User Story per il backlog

Data: 2026-10-16. Base: `918cc47`, branch `feature/efforts`.
Stato: **T-01 e T-02 completati il 2026-10-17**; tutti gli altri elementi sono
**Proposti — da raffinare e stimare**.

Fonte: [requisiti Efforts](../requirements/efforts.md).
Vincoli e codice verificato: [analisi architetturale](../requirements/efforts-analysis.md).

## 1. Regole di composizione

Questo backlog contiene **8 Epic, 25 User Story e 3 attività tecniche**.
Gli ID sono locali al documento; non sono ticket creati su un servizio esterno.
Le storie descrivono risultati utente; T-01–T-03 sono abilitatori/verifiche
tecniche e non sono presentati come finte User Story.

I criteri AC citati rimandano alla baseline. Le verifiche aggiuntive specificano
come renderla testabile, senza sostituirla. Ogni storia include, nel proprio
ambito, controlli server, storico atomico, concorrenza e privacy definiti dagli
elementi trasversali. Una storia non è completata se funziona soltanto dalla UI.

Tutti gli elementi sono necessari alla V1. Le ondate della sezione 5 indicano
l'ordine di costruzione e integrazione, non rilasci parziali autorizzati.
Stime e assegnatari vanno definiti dal team dopo T-01/T-02; non sono deducibili
in modo attendibile dalla sola lettura del repository.

## 2. Attività tecniche trasversali

### T-01 — Contratti e persistenza Efforts nel backend esistente

**Stato:** completato il 2026-10-17 secondo il perimetro concordato.
Contratti e confini: [efforts-contracts.md](../requirements/efforts-contracts.md).
Migrazione 47, letture API e scope verificati; le scritture restano non esposte.

**Scopo:** permettere alle storie di usare un modello coerente con i livelli Go,
SQL/PostgreSQL, DTO e frontend già presenti.

**Attività:**

- Definire modello di valori propri e worklog, minuti interi, data di calendario,
  titolare immutabile, operatore, revisione e identificazione del tentativo.
- Definire configurazione workspace e default; scegliere colonne o tabella
  collegata per i valori propri e documentare la scelta.
- Produrre nuove migrazioni up/down, FK e vincoli su workspace/issue, durata,
  valori non negativi, cancellazione e conservazione delle attribuzioni.
- Integrare le guardie privacy sulle nuove tabelle: la migrazione 46 non le
  installerà automaticamente sulle tabelle create successivamente.
- Definire contratti API, scope Efforts, revisione attesa, idempotenza, errori
  contestuali, riepilogo, aggregati e paginazione da 50; riusare DTO comuni.
- Definire come rappresentare snapshot storici e operatore di sistema e come
  rimapparli negli archivi. Non ripristinare la vecchia scala di stima rimossa.
- Estendere `domain`, `dto`, interfacce repository, wiring in `main.go`, route,
  tipi/client TypeScript e catalogo scope nei punti esistenti.

**Completamento:** decisioni registrate con esempi di payload e migrazioni
verificabili su database esistente e nuovo; assente/zero distinti; nessun nuovo
servizio, ORM o framework introdotto. Nessuna API di scrittura esposta prima
dei controlli previsti da T-02 e US-18/19.

**Dipendenze:** nessuna. **Supporta:** tutte le Epic.

### T-02 — Confine transazionale, lock e autorizzazione al salvataggio

**Stato:** completato il 2026-10-17 secondo il perimetro concordato.
Implementazione e verifiche: [efforts-transactions.md](../requirements/efforts-transactions.md).
Comandi interni, chiusure normali/GitHub e relativi scope PAT integrati;
nessuna nuova route di scrittura Efforts. Gli altri canali restano in US-18/19.

**Scopo:** fornire alle operazioni Efforts le garanzie richieste da EFF-040,
senza riscrivere l'architettura del tracker.

**Attività:**

- Implementare un percorso repository transazionale condiviso per dati Efforts,
  variazioni del residuo e storico. Non usare il solo `recordHistory` attuale,
  che tratta l'errore come non bloccante.
- Mantenere READ COMMITTED ordinario e guardie privacy; definire ordine dei lock
  compatibile con migrazione 46, issue, workspace e budget giornaliero.
- Serializzare le variazioni del budget per workspace/titolare/data anche se non
  esistono ancora worklog; nel cambio data proteggere origine e destinazione.
- Ricontrollare accesso, ruolo, assegnazione, titolarità e attivazione dopo le
  eventuali attese; serializzare configurazione e riattivazione con le scritture.
- Predisporre revisione attesa e deduplicazione persistente dei tentativi;
  documentare conflitti e retry ammessi. Pubblicare eventi solo dopo commit.
- Integrare gli stessi invarianti nei percorsi di chiusura normali e GitHub;
  confinare l'adattamento ai punti necessari per Efforts.

**Completamento:** prove su PostgreSQL con due connessioni dimostrano rollback,
limite giornaliero, conflitti, revoche concorrenti e assenza di deadlock nei
percorsi coperti. Nessun affidamento a mutex del singolo processo.

**Dipendenze:** T-01. **Supporta:** tutte le scritture e US-21–23.

### T-03 — Accettazione integrata e preparazione del rilascio

**Scopo:** verificare il risultato completo, oltre alle prove svolte dentro ogni
storia. **Dipendenze:** US-01–US-25 completate per la chiusura dell'attività.

**Attività e completamento:**

- Tradurre AC-001–AC-051 in una suite tracciata, mantenendo test Go di dominio,
  repository PostgreSQL, handler/route e test UI/E2E negli strumenti esistenti.
- Eseguire migrazioni su base pre-Efforts e installazione nuova; verificare
  default, conservazione dati e comportamento distruttivo di eventuale rollback.
- Eseguire test concorrenti e con errori iniettati; verificare revoca di ruolo,
  accesso, funzione disattivata, PAT e assenza di esposizione pubblica.
- Verificare con tastiera entrambe le superfici di dettaglio, traduzioni,
  errori contestuali, pagina oltre 50 righe e riconnessione realtime.
- Eseguire i controlli pertinenti del progetto: Go test con PostgreSQL e `-race`,
  controlli/build UI, test unitari ed E2E delle aree coinvolte. Verificare che i
  test DB non siano saltati per mancanza delle variabili d'ambiente richieste.
- Misurare query di aggregazione e riattivazione su dati rappresentativi,
  documentando volumi e risultati; non introdurre SLA arbitrari.
- Aggiornare documentazione API/uso e nota di rilascio: Efforts attivo di default,
  `Europe/Rome`, permessi, disattivazione e limiti del perimetro V1.

## 3. Epic e User Story

### EP-01 — Configurazione e disponibilità nel workspace

**Risultato:** owner/admin governano Efforts; gli utenti conservano i dati anche
quando la funzione viene disabilitata. **Storie:** US-01–03.

#### US-01 — Disponibilità predefinita nei workspace

**Storia:** come utente di un workspace, voglio trovare Efforts disponibile al
rilascio e nei nuovi workspace, per poterlo usare senza attivazioni tecniche.

**Requisiti:** EFF-001, EFF-012, EFF-030. **AC fonte:** AC-010, AC-039.
**Dipendenze:** T-01, T-02.

**Accettazione:**

- Workspace esistenti e nuovi hanno Efforts attivo e fuso `Europe/Rome`.
- L'impostazione vale per tutte le issue del workspace, senza flag di team/issue.
- Una disattivazione salvata successivamente non viene sovrascritta a un riavvio
  o da una successiva esecuzione delle migrazioni.

**Attività:** migrazione/default e costruzione workspace; lettura configurazione
nei DTO e nello stato UI; prova su dati preesistenti e creazione successiva.

#### US-02 — Disabilitare e riattivare Efforts preservando i dati

**Storia:** come owner/admin, voglio sospendere e riattivare Efforts per l'intero
workspace, per governarne l'utilizzo senza perdere il lavoro storico.

**Requisiti:** EFF-001, EFF-008, EFF-012, EFF-033.
**AC fonte:** AC-004, AC-009, AC-011, AC-043.
**Dipendenze:** US-01, US-18, US-21; T-02.

**Accettazione:**

- Solo sessioni interattive di owner/admin possono cambiare l'attivazione;
  membri, guest e PAT non possono farlo.
- Disabilitando, dati e letture autorizzate rimangono; tutte le scritture dirette
  Efforts sono rifiutate, anche con un modulo aperto prima della disattivazione.
- Una chiusura a funzione disabilitata conserva il residuo salvato.
- Riattivando, stime/worklog sono conservati; residui delle issue aperte restano
  invariati e quelli delle issue attualmente chiuse vengono azzerati con storico.
- Se un passaggio fallisce, attivazione, azzeramenti e storico non rimangono
  parzialmente applicati. Impostazioni e fuso restano amministrabili da disattivo.

**Attività:** percorso di configurazione owner/admin distinto dal PATCH generale
owner-only; transazione di riattivazione; controlli comuni sulle scritture;
comandi UI e aggiornamento delle sessioni. Verificare concorrenza con US-11/22.

#### US-03 — Configurare il fuso senza spostare il lavoro storico

**Storia:** come owner/admin, voglio scegliere il fuso del workspace, per applicare
la data corrente del contesto di lavoro senza alterare le registrazioni passate.

**Requisiti:** EFF-020, EFF-029, EFF-030, EFF-035.
**AC fonte:** AC-024, AC-039, AC-045.
**Dipendenze:** US-01, US-18; T-02.

**Accettazione:**

- Solo owner/admin con sessione possono salvare un identificativo IANA valido,
  anche con Efforts disabilitato; un identificativo non valido è rifiutato.
- «Oggi» e il valore proposto dai moduli usano il fuso del workspace, inclusa
  l'ora legale, senza dipendere dal fuso del server o del browser.
- Cambio fuso non cambia date e totali già salvati. Un worklog divenuto futuro
  rimane conservato; un suo successivo salvataggio applica il nuovo «oggi».

**Attività:** validazione/calcolo server, controllo impostazioni, riuso dei
componenti e traduzioni UI; test ai confini del giorno e dell'ora legale.

### EP-02 — Pianificazione dell'effort proprio della issue

**Risultato:** gli utenti autorizzati distinguono previsione, lavoro svolto e
lavoro ancora necessario. **Storie:** US-04–05.

#### US-04 — Impostare, rivedere e rimuovere la stima

**Storia:** come assegnatario non guest o owner/admin, voglio gestire una stima
facoltativa, per pianificare l'issue anche dopo che è iniziato il lavoro.

**Requisiti:** EFF-002, EFF-013–016, EFF-018, EFF-032, EFF-034.
**AC fonte:** AC-012, AC-014, AC-017, AC-018, AC-021, AC-022, AC-041, AC-044.
**Dipendenze:** US-01, US-18, US-21; T-02.

**Accettazione:**

- Assegnazione multipla, ruolo e accesso correnti governano l'operazione;
  membri non assegnatari e guest sono esclusi.
- Sono validi zero e durate intere non negative. `90m`, `2h`, `1h 30m` e numero
  senza unità (minuti) sono supportati; niente arrotondamenti silenziosi.
- Su issue aperta con residuo assente, stima 8h e consuntivo 2h inizializzano 6h;
  se consuntivo supera la stima, il residuo inizializzato è zero.
- Un residuo già impostato, anche a zero, non cambia con la revisione/rimozione
  della stima; il consuntivo non cambia. Su issue chiusa il residuo resta zero.

**Attività:** comando e validazioni nel service, persistenza con revisione attesa,
storico; input durata condiviso e controllo integrabile nel riepilogo US-15.

#### US-05 — Correggere manualmente il residuo

**Storia:** come assegnatario non guest o owner/admin, voglio correggere il lavoro
residuo, per aggiornare la previsione senza alterare stima o consuntivo.

**Requisiti:** EFF-004, EFF-006, EFF-013, EFF-018, EFF-024, EFF-032, EFF-034.
**AC fonte:** AC-002, AC-012, AC-021, AC-027, AC-041, AC-042.
**Dipendenze:** US-04.

**Accettazione:**

- Sulle issue aperte l'utente autorizzato imposta minuti interi non negativi;
  zero esplicito viene distinto dal residuo assente.
- Correggere il residuo non modifica stima o worklog e produce storico prima/dopo.
- Su issue chiuse la correzione è bloccata per tutti. Dopo riapertura è ammessa,
  senza ripristinare automaticamente il valore precedente.

**Attività:** comando separato dalla stima, validazione stato/permessi al commit,
controllo UI e rappresentazione esplicita del valore impostato.

### EP-03 — Registrare e correggere il lavoro

**Risultato:** membri e amministratori registrano contributi manuali affidabili,
con attribuzione corretta e limiti coerenti. **Storie:** US-06–10.

#### US-06 — Registrare lavoro proprio

**Storia:** come membro non guest con accesso all'issue, voglio registrare il mio
lavoro manualmente, per aggiornare consuntivo e residuo anche senza assegnazione.

**Requisiti:** EFF-003, EFF-006, EFF-007, EFF-009, EFF-014, EFF-015,
EFF-018–020, EFF-021, EFF-027, EFF-034.
**AC fonte:** AC-001, AC-003, AC-005, AC-013, AC-015, AC-016, AC-022–024,
AC-026, AC-034.
**Dipendenze:** US-01, US-03, US-10, US-18, US-21, US-22; T-02.

**Accettazione:**

- Data e durata da 1 a 1.440 minuti sono obbligatorie; descrizione testuale
  facoltativa, anche multilinea, massimo 2.000 caratteri. Errori non troncano dati.
- Il modulo propone utente corrente e oggi nel fuso workspace. Date passate
  ammesse senza finestra retroattiva; date future rifiutate.
- Si può registrare senza stima, senza assegnazione e su issue chiuse, se ruolo,
  accesso, attivazione e limite giornaliero lo consentono.
- Un nuovo worklog aumenta il consuntivo e porta il residuo impostato a
  `max(residuo - durata, 0)`; residuo assente resta assente, mostrato zero.
- Durata oltre il residuo è interamente registrata. Su issue chiusa il residuo
  rimane zero. Retry del medesimo invio non crea una seconda registrazione.

**Attività:** comando atomico con storico e budget; form accessibile condiviso,
parser durata e validazioni, gestione tentativo/errore e refresh dopo successo.

#### US-07 — Modificare un worklog senza ricalcolare il residuo

**Storia:** come titolare non guest o owner/admin, voglio correggere data, durata
e descrizione di un worklog, per rettificare il consuntivo mantenendo la previsione.

**Requisiti:** EFF-010, EFF-017, EFF-019, EFF-020, EFF-028, EFF-034, EFF-037.
**AC fonte:** AC-007, AC-019, AC-023, AC-024, AC-026, AC-036, AC-037, AC-047.
**Dipendenze:** US-06.

**Accettazione:**

- Il membro corregge solo worklog propri, owner/admin tutti quelli accessibili;
  i permessi correnti prevalgono su quelli al momento dell'inserimento.
- Durata/data/descrizione rispettano le validazioni di creazione e il limite
  giornaliero, escludendo dal nuovo totale il contributo precedente del worklog.
- Cambiare data protegge entrambi i giorni; un rifiuto non altera nessuno dei due.
- Cambiare 2h in 3h aggiorna il consuntivo ma lascia il residuo invariato. Il
  titolare è immutabile; conflitti mantengono la bozza e richiedono riallineamento.

**Attività:** update con revisione attesa, lock sui budget interessati e storico
prima/dopo; riuso form con titolare non editabile e messaggi contestuali.

#### US-08 — Eliminare un worklog conservando la traccia

**Storia:** come titolare non guest o owner/admin, voglio eliminare una
registrazione errata, per correggere il consuntivo mantenendo lo storico.

**Requisiti:** EFF-010, EFF-017, EFF-026, EFF-041.
**AC fonte:** AC-007, AC-009, AC-020, AC-032, AC-051.
**Dipendenze:** US-06.

**Accettazione:**

- Eliminazione richiede conferma; accesso, ruolo e attivazione sono riesaminati
  dal server. Guest e membri su registrazioni altrui sono esclusi.
- Eliminare 2h sottrae 2h dal consuntivo e dal budget giornaliero, lasciando il
  residuo invariato, anche su issue chiusa.
- Lo storico conserva i dati eliminati e l'operatore. Un retry non ripete gli
  effetti né produce eventi di eliminazione duplicati.

**Attività:** eliminazione atomica con controllo di concorrenza, snapshot storico,
dialogo di conferma condiviso e riallineamento registro/riepilogo.

#### US-09 — Registrare per conto di un'altra persona

**Storia:** come owner/admin, voglio inserire lavoro svolto da un'altra persona
ammissibile, per riportare il consuntivo distinguendo titolare e operatore.

**Requisiti:** EFF-025, EFF-028, EFF-037.
**AC fonte:** AC-030, AC-031, AC-035, AC-047.
**Dipendenze:** US-06, US-18.

**Accettazione:**

- Solo owner/admin possono scegliere un altro owner, admin o membro attuale
  del workspace con accesso all'issue; guest e persone non ammissibili sono esclusi.
- Il limite giornaliero e la data sono valutati per il titolare nel workspace,
  indipendentemente dall'operatore e dal suo fuso personale.
- Registro e storico distinguono il titolare da chi ha inserito il lavoro;
  l'operatore non acquisisce la titolarità. Il titolare non è riassegnabile.
- Se ruolo/accesso cambiano mentre il modulo è aperto, il server ricontrolla
  l'ammissibilità e rifiuta una nuova registrazione non più consentita.

**Attività:** selezione server degli utenti ammissibili e controllo al commit;
selettore nel form e indicazione «inserito da» nel registro.

#### US-10 — Rispettare il limite giornaliero tra tutte le issue

**Storia:** come persona a cui è attribuito lavoro, voglio che il totale per
giorno non superi 24 ore nel workspace, per evitare consuntivi impossibili.

**Requisiti:** EFF-028, EFF-034, EFF-040.
**AC fonte:** AC-035–038, AC-050.
**Dipendenze:** T-01, T-02, US-18.

**Accettazione:**

- Con 23h, aggiungere 1h è consentito; 1h 1m è rifiutato. Il controllo comprende
  tutte le issue del workspace e gli inserimenti per conto altrui.
- Con 24h comprensive di un worklog da 2h, correggerlo a 1h è valido, a 3h no.
  Spostare 2h verso una data già a 23h è rifiutato senza effetti parziali.
- Due sessioni su issue diverse non superano insieme il limite, anche se il
  giorno non aveva worklog prima degli inserimenti.
- Il controllo comprende lavoro su issue nascoste, senza esporne valori,
  identificativi o numero nel messaggio d'errore.

**Attività:** query interna limitata a workspace/persona/data e lock persistenti;
test con connessioni reali, delega simultanea e spostamento tra giorni.

### EP-04 — Coerenza con il ciclo di vita delle issue

**Risultato:** le azioni già disponibili sulle issue conservano le regole Efforts.
**Storie:** US-11–14.

#### US-11 — Chiudere, annullare e riaprire con residui coerenti

**Storia:** come utente del tracker, voglio che le transizioni di stato gestiscano
il residuo in modo uniforme, per avere previsioni coerenti con lo stato reale.

**Requisiti:** EFF-021, EFF-024, EFF-032, EFF-033, EFF-036.
**AC fonte:** AC-025–027, AC-041–043.
**Dipendenze:** US-02, US-05, US-21; T-02.

**Accettazione:**

- Con Efforts attivo, entrare in categoria completed/cancelled azzera il residuo
  proprio con storico, lasciando stima e worklog invariati.
- Vale per update singolo, bulk, rifiuto triage, chiusure automatiche parent/figli
  e transizioni GitHub, anche con stati personalizzati.
- Riaprire lascia zero; nessuna revisione della stima o correzione worklog fa
  risalire il residuo mentre la issue è chiusa.
- Chiudere una parent agisce sul suo valore proprio. Se l'automazione esistente
  chiude dei figli, ciascuno segue la regola; figli ancora aperti mantengono i dati.
- Con funzione disabilitata chiusura/riapertura non cambiano i dati congelati;
  la riattivazione segue US-02. Un errore non lascia stato e residuo incoerenti.

**Attività:** individuare e integrare tutti i percorsi in IssueService,
GitHubService e repository, preservando trigger lifecycle; attribuire causa e
operatore di sistema; test diretti e automatici, senza ampliare le automazioni.

#### US-12 — Duplicare soltanto la pianificazione prevista

**Storia:** come utente autorizzato a duplicare una issue, voglio ritrovare la
stima nella copia senza copiarne il lavoro, per iniziare un'attività distinta.

**Requisiti:** EFF-038. **AC fonte:** AC-048.
**Dipendenze:** US-04, US-11, US-19.

**Accettazione:**

- Con funzione attiva, la copia riceve la stima, zero worklog e nessuno storico
  Efforts copiato dall'originale; residuo pari alla stima se presente, altrimenti
  assente. Per copia in stato chiuso prevale residuo zero.
- La stessa regola vale per ciascuna sotto-issue copiata dal flusso esistente;
  non si estende la duplicazione attuale a ulteriori livelli.
- Con Efforts disabilitato non si copiano dati Efforts. Un PAT non può usare la
  duplicazione per leggere o scrivere Efforts senza gli scope necessari.

**Attività:** estendere `duplicateIssue`/creazione con copia esplicita della sola
stima, controlli API su effetti Efforts e test con/senza figli e stima.

#### US-13 — Conservare dati in spostamenti, gerarchie e conversione

**Storia:** come utente che organizza le issue, voglio conservare gli effort
quando cambio progetto o gerarchia, per non perdere il lavoro già registrato.

**Requisiti:** EFF-022, EFF-036, EFF-038.
**AC fonte:** AC-046, AC-048.
**Dipendenze:** T-01, T-02, US-18.

**Accettazione:**

- Gli spostamenti consentiti dall'architettura conservano stime, residui,
  worklog e storico sulle issue originali; l'accesso risultante vale subito.
- Cambiare parent aggiorna gli aggregati della vecchia e nuova catena di
  antenati, senza alterare i valori propri. La prevenzione dei cicli resta attiva.
- La conversione attuale conserva effort sulle issue associate al progetto,
  senza creare effort propri del progetto o introdurre effort nei template.
- Le regole esistenti sui confini dei team privati restano vincolanti; non si
  aggiungono percorsi di spostamento oggi non supportati.

**Attività:** integrare update parent/bulk e `ConvertToProject`, raccogliere
antenati interessati e invalidare le letture tramite US-23; test conservazione.

#### US-14 — Eliminare una issue con avviso e conservare i figli rimasti

**Storia:** come utente autorizzato a eliminare issue, voglio conoscere l'effetto
sui dati Efforts, per confermare consapevolmente la cancellazione.

**Requisiti:** EFF-038. **AC fonte:** AC-048.
**Dipendenze:** T-01, T-02, US-18.

**Accettazione:**

- La conferma avvisa esplicitamente che worklog e storico Efforts della issue
  verranno eliminati, anche con funzione disabilitata; copre i percorsi bulk.
- La cancellazione rimuove i dati della issue e il loro contributo ai totali;
  i figli conservati dall'attuale comportamento mantengono i propri effort.
- Eliminare l'intero workspace include i dati Efforts; rimuovere un membro
  invece non elimina il suo lavoro. Errori non lasciano dati orfani.

**Attività:** vincoli/cascade mirati, dialoghi esistenti in menu/pannello/pagina
e bulk, refresh degli antenati e test di distinzione issue/membro/workspace.

### EP-05 — Consultare Efforts nel dettaglio issue

**Risultato:** gli utenti leggono dati propri, registro e totali visibili senza
confonderli o poter modificare valori derivati. **Storie:** US-15–17.

#### US-15 — Riepilogo laterale e stato di sola lettura

**Storia:** come utente con accesso all'issue, voglio leggere stima, consuntivo e
residuo nel riepilogo laterale, per valutare rapidamente l'avanzamento.

**Requisiti:** EFF-005, EFF-008, EFF-011, EFF-022, EFF-031, EFF-033, EFF-041.
**AC fonte:** AC-006, AC-028, AC-040, AC-051.
**Dipendenze:** US-02, US-04–05, US-17, US-18.

**Accettazione:**

- Pagina completa e dettaglio in pannello presentano valori propri e totali
  separati, con unità coerenti, stima assente riconoscibile e residuo assente
  visualizzato zero senza perderne la semantica.
- Consuntivo e aggregati sono sempre in sola lettura; comandi stima/residuo
  rispettano ruolo, assegnazione, attivazione e stato.
- Guest leggono ma non scrivono. Con Efforts disabilitato i dati presenti sono
  leggibili con indicazione esplicita; senza dati propri o aggregati il pannello
  vuoto è nascosto.

**Attività:** componente riepilogo condiviso e integrazione nei due contenitori,
collegamento comandi US-04/05, traduzioni/cataloghi e gestione aggiornamenti.

#### US-16 — Registro paginato e moduli accessibili

**Storia:** come utente con accesso all'issue, voglio consultare le registrazioni
nel corpo e usare azioni accessibili, per ricostruire e gestire il lavoro svolto.

**Requisiti:** EFF-003, EFF-007, EFF-011, EFF-025, EFF-031, EFF-041.
**AC fonte:** AC-003, AC-006, AC-040, AC-051.
**Dipendenze:** US-06–09, US-15.

**Accettazione:**

- La sezione mostra data, titolare, durata, descrizione e «inserito da» quando
  pertinente; ordina per data lavoro decrescente, poi inserimento decrescente.
- Le righe successive alla cinquantesima sono consultabili tramite pagine o
  caricamento progressivo da 50; a parità dei criteri l'ordine resta stabile.
- «Registra lavoro», modifica ed eliminazione usano i comandi delle storie
  precedenti e rispettano permessi correnti; eliminazione richiede conferma.
- Moduli, focus e messaggi sono utilizzabili da tastiera; gli errori sono
  contestuali e non cancellano la bozza. La sola lettura è esplicita.
- Funziona sia in pagina sia in pannello, con componenti e traduzioni esistenti;
  non aggiunge timesheet, ricerca o filtri trasversali.

**Attività:** query/DTO paginati con default 50, componente registro condiviso,
integrazione dei form, stati vuoto/errore/caricamento e prove oltre 50 righe.

#### US-17 — Totali dei discendenti visibili a ogni profondità

**Storia:** come utente che consulta una parent issue, voglio vedere i totali
delle issue accessibili della gerarchia, per valutare il lavoro complessivo.

**Requisiti:** EFF-022, EFF-036.
**AC fonte:** AC-028, AC-046.
**Dipendenze:** T-01, US-13, US-18.

**Accettazione:**

- L'aggregato comprende l'issue e tutti i discendenti visibili una sola volta;
  8/2/6h proprie più 4/1/3h di un figlio danno 12/3/9h, senza scrivere sul parent.
- Stime note formano un subtotale con indicazione parziale se ne mancano; se
  mancano tutte si mostra «Non stimato». Residui assenti contribuiscono zero con
  indicazione di parzialità.
- La dicitura «Totale delle issue visibili» non rivela quantità o dettagli dei
  discendenti esclusi. L'API applica la visibilità prima di restituire le somme.
- Chiusura della parent non azzera figli ancora aperti; spostamenti, modifica
  dei dati e cambi accesso aggiornano i totali degli antenati interessati.

**Attività:** query ricorsiva workspace-scoped su valori propri/worklog con
predicati di accesso e marcatori di completezza; test su tre livelli, privacy,
spostamenti e query plan; rendering nel riepilogo US-15.

### EP-06 — Autorizzazioni, identità e canali di accesso

**Risultato:** ogni persona e token opera nei limiti correnti, preservando dati
storici e riservatezza. **Storie:** US-18–20.

#### US-18 — Applicare ruoli, accesso e identità correnti

**Storia:** come utente del workspace, voglio che accessi e attribuzioni siano
rispettati anche dopo cambi di ruolo, per mantenere dati corretti e riservati.

**Requisiti:** EFF-009–011, EFF-013, EFF-025, EFF-037.
**AC fonte:** AC-005–008, AC-012, AC-030, AC-031, AC-047.
**Dipendenze:** T-01, T-02.

**Accettazione:**

- Accesso all'issue è necessario per tutte le operazioni; conoscere UUID di
  issue/worklog o appartenere al workspace non supera la visibilità del team.
- Owner/admin operano su tutte le registrazioni accessibili, membri solo sulle
  proprie; stima/residuo ai membri assegnatari, mai ai guest.
- Diventare guest o perdere accesso revoca le scritture sui vecchi worklog;
  rimozione dal workspace non cancella dati né riduce i totali.
- Owner/admin con accesso possono correggere lavoro storico di una persona
  rimossa, guest o senza accesso, ma non crearle nuove registrazioni.
- Attribuzioni storiche rimangono disponibili; per account eliminato si gestisce
  «Utente non più disponibile». Il modello non cancella worklog per cascade utente.

**Attività:** policy comune server per operazione e oggetto, assegnazione
multipla, predicati repository e controllo corrente nella transazione; capacità
UI coerenti e prove di revoca concorrente/UUID cross-workspace.

#### US-19 — Usare le API con scope Efforts espliciti

**Storia:** come utilizzatore delle API, voglio autorizzare esplicitamente un PAT
per Efforts, per automatizzare le operazioni entro i permessi del proprietario.

**Requisiti:** EFF-039. **AC fonte:** AC-049.
**Dipendenze:** T-01, T-02, US-18.

**Accettazione:**

- API e UI applicano le stesse validazioni e permessi; gli scope Efforts sono
  espliciti e non attribuiti retroattivamente ai token già creati.
- Scope non elevano ruolo/accesso del proprietario. Attivazione e fuso restano
  riservati alla sessione interattiva owner/admin.
- Un token senza scope Efforts non ottiene dati o scritture Efforts attraverso
  dettaglio/lista/storico issue, duplicazione, export o altri percorsi generici.
- La modifica dei campi non Efforts continua a rispettare il suo contratto:
  l'estensione definisce quando filtrare una proiezione o negare un'operazione
  che includerebbe Efforts, senza ampliare implicitamente gli scope esistenti.

**Attività:** permessi/catalogo PAT, annotazioni route e relativo manifest test,
filtri delle proiezioni e controlli sugli effetti indiretti, UI scope e preset;
documentare combinazioni necessarie per export e duplicazione in T-01.

#### US-20 — Escludere Efforts dalle condivisioni pubbliche

**Storia:** come membro del workspace, voglio condividere issue senza pubblicare
i relativi effort, per mantenere riservati pianificazione e contributi personali.

**Requisiti:** EFF-011, EFF-023, EFF-026.
**AC fonte:** AC-029.
**Dipendenze:** T-01, US-18, US-21.

**Accettazione:**

- Nessuna pagina o risposta pubblica include stima, consuntivo, residuo,
  worklog, aggregati o eventi dello storico che li rivelino.
- Il vincolo vale con Efforts attivo/disattivo e su issue con discendenti;
  rimuovere controlli visivi non è sufficiente.
- Gli endpoint Efforts non sono accessibili usando soltanto il link pubblico.

**Attività:** mantenere i DTO pubblici espliciti, verificare serializzazione,
eventuali arricchimenti e chiamate della UI pubblica; prove HTTP su payload reali.

### EP-07 — Storico, affidabilità e aggiornamenti tra sessioni

**Risultato:** le operazioni sono ricostruibili e resistono a errori, retry e
modifiche simultanee. **Storie:** US-21–23.

#### US-21 — Storico completo nell'attività dell'issue

**Storia:** come utente con accesso all'issue, voglio ricostruire le operazioni
Efforts, per sapere chi ha cambiato i dati, quando e per quale causa.

**Requisiti:** EFF-025, EFF-026, EFF-033, EFF-037, EFF-040.
**AC fonte:** AC-032, AC-033, AC-043, AC-050.
**Dipendenze:** T-01, T-02, US-18.

**Accettazione:**

- Inserimento/modifica/eliminazione worklog e variazioni stima/residuo producono
  eventi con operatore e dati prima/dopo sufficienti a ricostruire l'operazione.
- Deleghe distinguono titolare e operatore; le variazioni automatiche dichiarano
  la causa: worklog, chiusura o riattivazione, distinta dalla correzione manuale.
- Worklog eliminato non contribuisce al totale ma resta ricostruibile; un
  aggiornamento fallito non lascia storico e un errore dello storico annulla i dati.
- Attività in pannello/pagina rappresenta gli eventi in modo leggibile, inclusi
  operatori di sistema/utenti non disponibili; applica privacy e scope Efforts.

**Attività:** estendere il contratto storico e le scritture transazionali,
snapshot strutturati, cause e rendering nei due componenti; conservare la
compatibilità degli eventi preesistenti e predisporre rimappatura import.

#### US-22 — Retry sicuri e conflitti espliciti

**Storia:** come utente che salva dati Efforts, voglio poter ritentare dopo un
errore di rete e riconoscere conflitti, per non duplicare né perdere lavoro.

**Requisiti:** EFF-040. **AC fonte:** AC-038, AC-050.
**Dipendenze:** T-01, T-02, US-18, US-21.

**Accettazione:**

- Lo stesso tentativo ripetuto dopo risposta persa non duplica worklog,
  decremento residuo o storico; vale anche dopo riavvio del backend.
- Due inserimenti intenzionali identici restano possibili con tentativi distinti;
  il riuso della stessa chiave con contenuto diverso non è accettato silenziosamente.
- Modifiche concorrenti di stima/residuo/worklog, inclusa eliminazione durante
  modifica, segnalano conflitto e non sovrascrivono dati più recenti.
- Il form mantiene la bozza e consente il riallineamento. Errori di persistenza
  non lasciano modifiche parziali; attivazione e permessi sono riesaminati.

**Attività:** deduplicazione persistente e revisione attesa nei contratti T-01,
risposta di conflitto, stato della bozza e retry client con la stessa chiave;
test multi-connessione e perdita della risposta dopo commit.

#### US-23 — Aggiornare le altre sessioni autorizzate

**Storia:** come utente che consulta una issue, voglio vedere le modifiche Efforts
delle altre sessioni, per prendere decisioni su dati aggiornati.

**Requisiti:** EFF-036, EFF-040. **AC fonte:** AC-046, AC-050.
**Dipendenze:** T-02, US-15–17, US-21–22.

**Accettazione:**

- Dopo commit le sessioni autorizzate aggiornano riepilogo, registro, attività
  e aggregati di tutti gli antenati coinvolti, anche dopo cambi parent/delete.
- Rollback non pubblica un successo; riconnessione recupera lo stato dal server.
- Nei workspace con privacy si mantiene il refresh generico del Hub e si
  ricaricano dati tramite API autorizzate, senza payload con dati/ID di issue
  nascoste. Perdita accesso invalida le viste precedentemente caricate.
- Non vengono create email o notifiche inbox dedicate alle operazioni Efforts.

**Attività:** riuso Hub, `app.refresh`, layout e stato UI; invalidazioni mirate
quando consentite, listener condivisi e teardown, test con due sessioni e privacy.

### EP-08 — Portabilità del workspace

**Risultato:** esportare e ripristinare un workspace conserva Efforts e
attribuzioni, entro i limiti di trasferimento già presenti. **Storie:** US-24–25.

#### US-24 — Includere Efforts nell'export workspace

**Storia:** come owner/admin autorizzato al trasferimento, voglio esportare anche
Efforts, per conservare pianificazione, lavoro e storico insieme al workspace.

**Requisiti:** EFF-039. **AC fonte:** AC-049.
**Dipendenze:** T-01, US-18–19, US-21.

**Accettazione:**

- Archivio comprende configurazione, fuso, stima, residuo con distinzione
  assente/zero, worklog e storico, con titolari e operatori rimappabili.
- L'export conserva lo stato disabilitato e segue autorizzazioni e blocchi
  esistenti per workspace con team privati, senza aggirarli con nuovi endpoint.
- I PAT richiedono gli scope espliciti definiti per trasferimento ed Efforts;
  non si aggiunge export timesheet/CSV.
- I dati Efforts esportati sono coerenti tra loro anche in presenza di
  scritture concorrenti, senza worklog e storico di versioni incompatibili.

**Attività:** estendere TableSpecs, riferimenti utente e versione archivio;
definire lettura coerente riusando il trasferimento e la barriera privacy,
verificando che questa da sola non sostituisca uno snapshot dei dati.

#### US-25 — Importare Efforts e supportare archivi precedenti

**Storia:** come utente autorizzato a importare un workspace, voglio ripristinare
Efforts con attribuzioni e regole corrette, per proseguire senza perdere lo storico.

**Requisiti:** EFF-030, EFF-034, EFF-035, EFF-037, EFF-039, EFF-040.
**AC fonte:** AC-045, AC-047, AC-049, AC-050.
**Dipendenze:** US-10, US-21, US-24.

**Accettazione:**

- Il round-trip conserva stime, residui assenti/zero, worklog, date, fuso,
  attivazione e storico; rimappa issue, titolari e operatori, anche negli snapshot.
- Importare un workspace disabilitato mantiene i dati congelati. Non si
  riapplicano riduzioni residuo o nuove registrazioni di storico come se ogni
  worklog importato fosse un inserimento interattivo.
- Archivi legacy privi di Efforts sono accettati secondo la compatibilità
  definita in T-01, usando attivo e `Europe/Rome` come default.
- Durate o totali giornalieri incoerenti vengono segnalati e rifiutati senza
  troncamenti o import parziale. Le date storiche sono conservate come date:
  non si applica indiscriminatamente il controllo «nuova registrazione futura».
- Restano vincoli di privacy e rimappatura utenti già presenti; non si creano
  nuovi accessi per persone rimosse e non si cancellano le loro attribuzioni.

**Attività:** estendere validazione ante-import, piano di rimappatura, ordine
dipendenze e transazione esistente; gestire gli effetti automatici del ripristino,
fixture legacy/invalide, rollback e round-trip con lavoro delegato/storico.

## 4. Tracciabilità completa

Le tabelle associano ogni requisito e ogni criterio della baseline ad almeno
una storia. T-01/T-02 forniscono le garanzie comuni; T-03 verifica l'intero insieme.

### Requisiti → User Story

| Requisito | Storie responsabili |
| --- | --- |
| EFF-001 | US-01, US-02 |
| EFF-002 | US-04 |
| EFF-003 | US-06, US-16 |
| EFF-004 | US-05 |
| EFF-005 | US-15, US-16 |
| EFF-006 | US-05, US-06 |
| EFF-007 | US-06, US-16 |
| EFF-008 | US-02, US-15 |
| EFF-009 | US-06, US-18 |
| EFF-010 | US-07, US-08, US-18 |
| EFF-011 | US-15, US-16, US-18, US-20 |
| EFF-012 | US-01, US-02 |
| EFF-013 | US-04, US-05, US-18 |
| EFF-014 | US-04, US-06 |
| EFF-015 | US-04, US-06 |
| EFF-016 | US-04 |
| EFF-017 | US-07, US-08 |
| EFF-018 | US-04, US-05, US-06 |
| EFF-019 | US-06, US-07 |
| EFF-020 | US-03, US-06, US-07 |
| EFF-021 | US-06, US-11 |
| EFF-022 | US-13, US-15, US-17 |
| EFF-023 | US-20 |
| EFF-024 | US-05, US-11 |
| EFF-025 | US-09, US-16, US-18, US-21 |
| EFF-026 | US-08, US-20, US-21 |
| EFF-027 | US-06 |
| EFF-028 | US-07, US-09, US-10 |
| EFF-029 | US-03 |
| EFF-030 | US-01, US-03, US-25 |
| EFF-031 | US-15, US-16 |
| EFF-032 | US-04, US-05, US-11 |
| EFF-033 | US-02, US-11, US-15, US-21 |
| EFF-034 | US-04, US-05, US-06, US-07, US-10, US-25 |
| EFF-035 | US-03, US-25 |
| EFF-036 | US-11, US-13, US-17, US-23 |
| EFF-037 | US-07, US-09, US-18, US-21, US-25 |
| EFF-038 | US-12, US-13, US-14 |
| EFF-039 | US-19, US-24, US-25 |
| EFF-040 | US-10, US-21, US-22, US-23, US-25 |
| EFF-041 | US-08, US-15, US-16 |

### Criteri di accettazione → User Story

| AC fonte | Storie di verifica |
| --- | --- |
| AC-001 | US-06 |
| AC-002 | US-05 |
| AC-003 | US-06, US-16 |
| AC-004 | US-02 |
| AC-005 | US-06, US-18 |
| AC-006 | US-15, US-16, US-18 |
| AC-007 | US-07, US-08, US-18 |
| AC-008 | US-18 |
| AC-009 | US-02, US-08 |
| AC-010 | US-01 |
| AC-011 | US-02 |
| AC-012 | US-04, US-05, US-18 |
| AC-013 | US-06 |
| AC-014 | US-04 |
| AC-015 | US-06 |
| AC-016 | US-06 |
| AC-017 | US-04 |
| AC-018 | US-04 |
| AC-019 | US-07 |
| AC-020 | US-08 |
| AC-021 | US-04, US-05 |
| AC-022 | US-04, US-06 |
| AC-023 | US-06, US-07 |
| AC-024 | US-03, US-06, US-07 |
| AC-025 | US-11 |
| AC-026 | US-06, US-07, US-11 |
| AC-027 | US-05, US-11 |
| AC-028 | US-15, US-17 |
| AC-029 | US-20 |
| AC-030 | US-09, US-18 |
| AC-031 | US-09, US-18 |
| AC-032 | US-08, US-21 |
| AC-033 | US-21 |
| AC-034 | US-06 |
| AC-035 | US-09, US-10 |
| AC-036 | US-07, US-10 |
| AC-037 | US-07, US-10 |
| AC-038 | US-10, US-22 |
| AC-039 | US-01, US-03 |
| AC-040 | US-15, US-16 |
| AC-041 | US-04, US-05, US-11 |
| AC-042 | US-05, US-11 |
| AC-043 | US-02, US-11, US-21 |
| AC-044 | US-04, US-06 |
| AC-045 | US-03, US-25 |
| AC-046 | US-13, US-17, US-23 |
| AC-047 | US-07, US-09, US-18, US-25 |
| AC-048 | US-12, US-13, US-14 |
| AC-049 | US-19, US-24, US-25 |
| AC-050 | US-10, US-21, US-22, US-23, US-25 |
| AC-051 | US-08, US-15, US-16 |

## 5. Ordine proposto di realizzazione

Gli elementi nella stessa ondata possono essere pianificati separatamente solo
dopo aver soddisfatto le dipendenze dichiarate. La preparazione dei componenti
UI può iniziare sui contratti T-01; la chiusura delle storie richiede integrazione.

| Ondata | Elementi | Risultato verificabile |
| --- | --- | --- |
| 0 — Fondazioni | T-01 → T-02 | Contratti, migrazioni, confini di transazione, privacy e lock verificati. |
| 1 — Policy e tracciabilità | US-18 → US-19/21; US-01 | Accesso/scopi, default e storico utilizzabili dalle operazioni successive. |
| 2 — Configurazione e comandi | US-02, US-03, US-04 → US-05; US-10, US-22 | Attivazione/fuso, pianificazione e garanzie su budget, retry e conflitti. |
| 3 — Worklog e lifecycle | US-06 → US-07/08/09; US-11/12/13/14 | Registrazioni e ciclo di vita integrati, inclusi bulk e automazioni. |
| 4 — Consultazione | US-17 → US-15 → US-16; US-20 | Riepilogo/registro in entrambe le superfici, aggregati e pubblico senza Efforts. |
| 5 — Realtime e portabilità | US-23; US-24 → US-25 | Aggiornamento tra sessioni e round-trip del workspace. |
| 6 — Accettazione | T-03 | Tutti i 41 requisiti e 51 AC verificati, documentazione pronta. |

Accesso, storico, idempotenza e integrazione API si costruiscono prima di
esporre le scritture al pubblico: non sono hardening rinviabile a fine progetto.
US-23 si integra dopo le viste, ma la pubblicazione post-commit e il rispetto
del Hub esistente valgono già durante le ondate precedenti.

## 6. Definition of Ready e Definition of Done

**Ready per una storia:** contratto T-01 applicabile, dipendenze soddisfatte,
casi della matrice permessi identificati, AC fonte associati e decisioni tecniche
necessarie risolte. Per iniziare il refinement non è richiesta l'approvazione di
nuove funzionalità fuori perimetro.

**Done per una storia:** comportamento completo nel proprio ambito API/UI,
validazioni e autorizzazioni server, storico atomico quando pertinente, assenza
di esposizione pubblica/PAT impropria, gestione errore/conflitto, test significativi
dei casi coperti e traduzioni dei nuovi testi nelle lingue del progetto.

**Done della V1:** tutte le storie e T-03 completate. Nessuna delle funzioni fuori
perimetro deve essere introdotta implicitamente per chiudere una storia.

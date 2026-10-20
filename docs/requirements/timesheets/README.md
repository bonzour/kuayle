# Timesheets — viste personali e aggregate tra workspace e team

Data: 2026-10-20. Identificativo: **TS-01**. Versione: **0.2**.
Stato: baseline 0.1 accettata dall'utente; revisione richiesta per consultazione
cross workspace/team e riuso della UI Insights. Le scelte di dettaglio aggiunte
sono esplicitate sotto; funzionalità non implementata con questa attività.
Backlog: [Epic e Stories](../../backlog/timesheets.md).

**Implementazione successiva all'analisi:** sono disponibili le fondazioni e i
selettori EP-01 e il timesheet personale EP-02 con calendario, griglia e
dettaglio, oltre agli aggregati amministrativi EP-03 con filtro persone e
attribuzioni storiche. Restano da implementare scritture dalla nuova vista e
aggiornamenti multi-workspace. Vedere [contratti API](contracts.md),
[verifica EP-01](acceptance-ep01.md), [verifica EP-02](acceptance-ep02.md)
e [verifica EP-03](acceptance-ep03.md);
questa nota non dichiara completata TS-01.

Baseline esaminata: branch `feature/efforts`, commit
`9007650e384e40d0ee86b42b61a267d1b1c055eb`, incluse le modifiche locali
presenti a Efforts al momento dell'analisi. Non è una verifica del deployment.

## 1. Obiettivo e decisioni proposte

Ogni utente autenticato di un workspace deve poter consultare il proprio
timesheet, ottenuto dai worklog Efforts. Gli utenti con permessi superiori
devono poter consultare il lavoro aggregato per persona o per progetto e
risalire alle singole registrazioni. La consultazione deve includere più
workspace e più team contemporaneamente, con filtri e riepiloghi anche per
queste dimensioni. L'ambito è la stessa istanza KUAYLE e lo stesso account.

La vista prende come riferimento l'esperienza tabellare di Jira Tempo:
periodo selezionabile, giorni in colonna, righe raggruppate, totali e accesso
al dettaglio. Il requisito riguarda KUAYLE e i suoi dati; non richiede
un'integrazione, una sincronizzazione o una dipendenza da Jira/Tempo.

La revisione conferma l'obiettivo cross workspace/team e il riuso, ove possibile,
della UI Insights. La tabella consolida la baseline e specifica le scelte per
realizzare l'estensione; default, fusi e composizione degli ambiti con ruoli
diversi sono proposte di dettaglio di questa revisione.

| Tema | Proposta per la prima versione |
| --- | --- |
| «Livelli più alti di permission» | Ruoli workspace `owner` e `admin`; nessun nuovo ruolo. |
| «Ogni utente» | Anche `guest` può consultare il proprio storico accessibile, in sola lettura. |
| Ambito | Uno, più o tutti i workspace autorizzati nella modalità scelta; uno, più o tutti i team visibili al loro interno. |
| Permessi misti | Personale su tutti i workspace con membership corrente; viste altrui solo sui workspace in cui il chiamante è owner/admin. |
| UI | Struttura e componenti di Insights ove compatibili, con selettori multipli workspace/team e griglia timesheet dedicata. |
| Periodo | Settimana, predefinita lunedì–domenica, oppure mese di calendario. |
| Progetto del lavoro | Progetto attuale dell'issue; nessuna attribuzione storica al progetto alla data del lavoro. |
| Registrazione dalla vista | Riutilizzo dei comandi e del dialogo Efforts; nessuna modifica diretta dei totali. |
| Unità visiva | Ore e minuti nella griglia; input coerente con Efforts, incluso `1d = 8h`. |
| Fusi diversi | Periodo comune di date, senza conversione dei worklog. Il workspace da cui si apre la pagina fornisce il fuso di riferimento del periodo corrente; ogni scrittura usa il fuso del proprio workspace. |

## 2. Stato attuale verificato

L'analisi Efforts del 16 ottobre descriveva una base ancora senza worklog.
Il codice attuale e la guida d'uso del 20 ottobre sono successivi: Efforts è
già implementato. Le esclusioni relative ai timesheet nella specifica V1
restano corrette per quella versione; TS-01 ne costituisce un'estensione.

| Area | Evidenza attuale | Conseguenza per TS-01 |
| --- | --- | --- |
| Dati | `effort_worklogs` contiene workspace, issue, titolare, inseritore, data di lavoro, minuti, descrizione e timestamp. | I worklog esistenti sono la fonte unica del timesheet. |
| Pianificazione | Stima e residuo sono distinti dal consuntivo; il riepilogo issue separa valori propri e totali dei discendenti visibili. | Il timesheet somma i worklog, senza sommare riepiloghi gerarchici o ripartire stime. |
| Registrazione | CRUD, delega owner/admin e inserimento per periodo con esclusione facoltativa dei weekend; periodo salvato atomicamente. | Nessun secondo flusso di scrittura o nuovo modello di registrazione. |
| Integrità | Revisione attesa, idempotenza, storico transazionale, massimo 1.440 minuti per persona/data/workspace; date future vietate. | Le operazioni avviate dal timesheet mantengono questi vincoli. |
| UI issue | Pagina e pannello integrano Effort; il working tree mostra le barre di stima/consuntivo/residuo e WORK LOG sotto ACTIVITY, con correzioni accessibili dagli eventi. | La nuova pagina affianca queste superfici; gli eventi della timeline non sono la fonte dei totali. |
| Letture | API dei worklog e riepiloghi circoscritte a una issue; worklog paginati a 50. | Manca una query autorizzata trasversale per periodo/persona/progetto. |
| Permessi | `owner`, `admin`, `member`, `guest`; `efforts:read` per tutti i ruoli, scrittura per i primi tre con vincoli di titolarità. | `efforts:read` da solo non distingue chi può vedere i report trasversali altrui. |
| Team e progetti | Membership team senza ruolo manager; progetto con `lead_id`; `project:manage` è presente anche per member. | Né essere lead né possedere `project:manage` equivale a essere owner/admin. |
| Privacy | Predicati di accesso applicati prima delle aggregazioni; nessun Efforts nei link pubblici. | Estendere la stessa protezione a somme, filtri, identità e dettagli dei timesheet. |
| Configurazione | Efforts attivo di default, fuso iniziale `Europe/Rome`; disattivazione conserva i dati leggibili. | Timesheet consultabile anche da disabilitato, con scritture bloccate. |
| Aggiornamenti | Eventi `app.refresh`, rilettura autorizzata e gestione riconnessione. | Riutilizzare il refresh per invalidare griglia, totali e dettagli. |
| Trasferimento | Export/import workspace v2 conserva Efforts; v1 compatibile senza dati Efforts. | Una vista derivata non richiede un secondo archivio timesheet. |
| Insights | Pagina legata a un workspace, selezione singola di team, tab Overview/Explore, stato nell'URL e componenti condivisi. | Riutilizzare presentazione e primitive; selezione multipla e aggregazione tra workspace sono nuove capacità Timesheet. |
| Accesso trasversale | `domain.Access` prevede richieste utente tra workspace; PAT con allowlist `workspace_slugs`; WebSocket del layout connesso al solo workspace aperto. | Verificare ruoli e allowlist per ogni workspace e coprire gli aggiornamenti anche degli altri workspace selezionati. |

### Fonti locali dell'analisi

- [Guida Efforts aggiornata](../efforts-usage.md),
  [contratti](../efforts-contracts.md),
  [accettazione registrata](../efforts-acceptance.md),
  [analisi storica del 16 ottobre](../efforts-analysis.md).
- [Schema, vincoli e indici](../../../BE/migrations/000047_efforts.up.sql),
  [modello Efforts](../../../BE/internal/domain/efforts.go),
  [letture e aggregati](../../../BE/internal/repository/efforts.go),
  [comandi transazionali](../../../BE/internal/repository/efforts_commands.go).
- [Ruoli e permission](../../../BE/internal/domain/permission.go),
  [scope Efforts](../../../BE/internal/domain/effort_access.go),
  [visibilità](../../../BE/internal/repository/visibility.go),
  [modello team](../../../BE/internal/domain/team.go),
  [modello progetto](../../../BE/internal/domain/project.go),
  [route HTTP](../../../BE/cmd/server/routes.go).
- [Client API Efforts](../../../UI/src/lib/api/efforts.ts),
  [WORK LOG](../../../UI/src/lib/features/issues/EffortWorklogs.svelte),
  [dialogo registrazione](../../../UI/src/lib/features/issues/EffortWorklogDialog.svelte),
  [conversione durate](../../../UI/src/lib/features/issues/effort-duration.ts).
- [Pagina Insights](../../../UI/src/routes/[workspaceSlug]/insights/+page.svelte),
  [InsightsExplorer](../../../UI/src/lib/features/analytics/InsightsExplorer.svelte),
  [selettore date Analytics](../../../UI/src/lib/features/analytics/AnalyticsDateRangePicker.svelte),
  [contesto di accesso](../../../BE/internal/domain/access.go),
  [modello PAT](../../../BE/internal/domain/personal_access_token.go),
  [middleware workspace](../../../BE/internal/middleware/workspace.go),
  [layout e WebSocket](../../../UI/src/routes/[workspaceSlug]/+layout.svelte).

L'analisi è statica. Gli esiti dei test riportati nel documento di accettazione
sono evidenze precedenti, non test rieseguiti durante questa attività.

## 3. Permessi e ambito dei dati

| Ruolo valutato in ciascun workspace | Il mio timesheet | Viste aggregate altrui, in ogni raggruppamento | Scritture dalla vista |
| --- | --- | --- | --- |
| Owner/admin | Sì | Sì, sulle issue accessibili | Proprie e delegate secondo Efforts |
| Member | Sì | No | Solo propri worklog secondo Efforts |
| Guest | Sì, anche storico da precedente member | No | Nessuna |
| Utente senza membership corrente | No | No | Nessuna |
| Visitatore tramite link pubblico | No | No | Nessuna |

**Accesso alla funzione e accesso ai dati sono controlli distinti.** Il ruolo
owner/admin abilita le viste aggregate, ma la nuova funzione non introduce
deroghe alla visibilità corrente di team, issue e progetti.

La matrice si applica separatamente a ciascun workspace incluso nella query.
Esempio: un account admin in A e member in B vede il proprio lavoro di A+B;
nella modalità amministrativa vede il lavoro delle persone solo in A. Non si
mescolano nel medesimo report amministrativo tutti i dati di A e solo i propri
di B, perché il totale sarebbe ambiguo. La UI indica l'ambito effettivo.

Il passaggio dalla vista personale a quella amministrativa conserva i workspace
selezionati solo dove il ruolo lo permette e segnala la variazione di ambito;
se nessuno è ammissibile, le modalità amministrative non sono disponibili.
Una richiesta API con workspace espliciti non autorizzati viene rifiutata
integralmente, senza dettagli che ne rivelino l'esistenza. «Tutti» viene risolto
dal server nell'insieme ammissibile alla modalità, mai nell'intera istanza.
Essere membro di un team o lead di progetto non aggiunge privilegi amministrativi.

La vista personale include solo `owner_user_id = utente autenticato`. L'autore
dell'inserimento e gli assegnatari dell'issue non determinano la persona a cui
attribuire le ore. La stessa regola vale per i raggruppamenti amministrativi.

Oggi member e guest possono leggere i worklog altrui nelle issue accessibili.
TS-01 limita la nuova consultazione trasversale altrui a owner/admin; **non
introduce riservatezza individuale dei worklog nelle API e viste issue
esistenti**. Cambiare anche quella policy richiederebbe un requisito distinto.

Per i PAT, la lettura richiede `efforts:read` e il medesimo controllo sul ruolo
attuale del proprietario. Il token di un member resta limitato al timesheet
personale anche se il chiamante manipola persona o raggruppamento. Le scritture
richiedono anche `efforts:write` e tutte le autorizzazioni Efforts esistenti.
I selettori di delega mantengono i vincoli di sessione già presenti.
Per ogni lettura cross workspace si applica anche l'allowlist `workspace_slugs`
del PAT: `nil` significa tutti i workspace consentiti al proprietario, non
accesso indiscriminato. Nessun ruolo del workspace nella URL può autorizzarne
un altro. I dati minimi di workspace/team/progetto dei nuovi selettori devono
essere restituiti entro lo stesso ambito autorizzato della lettura Timesheet.

## 4. Requisiti funzionali

### Accesso e periodo

| ID | Requisito |
| --- | --- |
| TS-001 | Aggiungere una voce **Timesheet** nella navigazione del workspace. L’apertura senza filtri espliciti mostra **Il mio timesheet** su tutti i workspace autorizzati, sulla settimana corrente nel fuso di riferimento. Il workspace di navigazione non limita i dati. |
| TS-002 | Consentire settimana/mese, periodo precedente/successivo e ritorno al periodo corrente. La settimana inizia lunedì; il mese include tutte le proprie date. Esplicitare periodo e fuso di riferimento, mantenendoli stabili quando cambiano i filtri workspace/team. |
| TS-003 | Visualizzare tutti i giorni del periodo, inclusi weekend e giorni senza registrazioni. Le date future possono comparire, ma ogni inserimento viene validato rispetto a oggi nel workspace destinatario. Un giorno vuoto non significa assenza, ferie o mancato adempimento. |

### Timesheet personale

| ID | Requisito |
| --- | --- |
| TS-004 | Mostrare una griglia personale workspace → team → progetto → issue con date sulle colonne. Ogni cella issue/giorno somma i worklog propri; mostrare subtotali per workspace, team e progetto, totale per issue, per giorno e per periodo. |
| TS-005 | Mostrare identificativo e titolo delle issue, workspace, team e progetto, con collegamento al dettaglio nel workspace corretto. Includere issue con lavoro nel periodo anche se chiuse, non assegnate o prive di stima. Nessuna riga vuota automatica per issue senza lavoro. |
| TS-006 | Consentire filtri multipli per workspace, team e progetto, includendo **Senza progetto**. Celle e totali riflettono gli stessi filtri. La vista personale non offre un selettore di altri utenti a member/guest. |
| TS-007 | Aprendo una cella, mostrare le singole registrazioni con workspace/team/issue, data, durata, descrizione, titolare e inseritore. Più registrazioni sullo stesso giorno/issue restano distinguibili. Il totale della cella non è un worklog modificabile. |

### Viste aggregate

| ID | Requisito |
| --- | --- |
| TS-008 | Nei workspace in cui il chiamante è owner/admin aggiungere le modalità **Per persona**, **Per progetto**, **Per workspace** e **Per team**. Tutte usano lo stesso periodo, gli stessi filtri e lo stesso insieme autorizzato di worklog. |
| TS-009 | **Per persona** raggruppa persona → workspace → team → progetto → issue. Mostra totali giornalieri e del periodo per persona, sommando lo stesso account fra workspace autorizzati, e consente di aprirne il timesheet mantenendo ambito e filtri. |
| TS-010 | **Per progetto** raggruppa progetto qualificato dal workspace → team dell’issue → persona → issue. Mostra totali giornalieri e del periodo per progetto, con dettaglio fino ai worklog. Progetti omonimi di workspace diversi restano distinti. |
| TS-011 | Consentire filtri combinabili per workspace, team, persone e progetti nelle viste aggregate. Più valori dello stesso filtro sono in OR; filtri diversi sono in AND. Il cambio fra raggruppamenti amministrativi conserva periodo, filtri e totale generale. |
| TS-012 | I gruppi derivano dai worklog autorizzati del periodo, comprese persone rimosse dal workspace o account non più disponibili. Non introdurre un report di persone senza ore; i gruppi senza contributi dopo i filtri non sono mostrati. |

### Collegamento a Efforts

| ID | Requisito |
| --- | --- |
| TS-013 | Offrire **Registra lavoro** a chi può scrivere Efforts in almeno un workspace: selezionare esplicitamente workspace e issue accessibile, anche assente dalla griglia. Dalla cella precompilare workspace, issue e giorno; nel timesheet altrui precompilare il titolare solo se delegabile nel workspace destinatario. Riutilizzare dialogo, parser e comandi Efforts, inclusi i periodi; una scrittura interessa una sola issue di un solo workspace. |
| TS-014 | Dal dettaglio consentire modifica/eliminazione secondo ruolo, titolarità e accesso nel workspace del worklog. Rileggere dati e revisione dell’issue prima di aprire la modifica. Conferma eliminazione, titolare immutabile, conflitti e retry secondo Efforts. Non usare implicitamente slug o permessi del workspace di navigazione. |
| TS-015 | Nei workspace con Efforts disabilitato mantenere dati leggibili e indicare sola lettura. Le registrazioni negli altri workspace restano possibili secondo permessi. Bloccare sul server anche invii da dialoghi aperti prima della disattivazione nel workspace destinatario. |
| TS-016 | Aggiornare griglia e dettagli dopo scritture, modifiche remote, refresh e riconnessioni per tutti i workspace selezionati. Revoche e cambi di ambito invalidano i dati; risposte tardive non li ripristinano né sostituiscono bozze. Non mostrare come completo un totale ottenuto con errori su parte dei workspace: segnalare errore e richiedere una rilettura coerente. |

### Presentazione e stati

| ID | Requisito |
| --- | --- |
| TS-017 | Presentare celle e totali in ore/minuti esatti, ad esempio `8h`, `1h 30m`, `40h`, senza arrotondamenti o conversioni implicite in giornate lavorative. L'input mantiene la convenzione Efforts: un numero senza unità indica giorni da 480 minuti; esplicitarlo nel dialogo. Non assumere un obbligo di 8 ore al giorno. |
| TS-018 | Distinguere caricamento, nessun lavoro nel periodo, nessun risultato con i filtri, errore e accesso revocato. Non presentare un errore come zero ore. Usare etichette tradotte, intestazioni di tabella, navigazione da tastiera, focus visibile e apertura/chiusura del dettaglio accessibile; su schermi stretti consentire scorrimento della griglia e dettaglio leggibile. |

### Workspace, team e riuso Insights

| ID | Requisito |
| --- | --- |
| TS-019 | Il selettore Workspace permette uno, più o tutti i workspace autorizzati per la modalità. Il selettore Team permette uno, più o tutti i team visibili dei workspace scelti. Mostrare l'ambito effettivo e qualificare team/progetti omonimi con il workspace; selettori e API non rivelano ambiti inaccessibili. |
| TS-020 | Filtrare per il team attuale dell'issue, non per membership del titolare né per team del progetto. Cambiare selezione workspace elimina solo filtri team/progetto non più pertinenti, con indicazione della variazione; non trasformare silenziosamente una selezione vuota in «Tutti». I nomi non sono chiavi di filtro o raggruppamento. |
| TS-021 | **Per workspace** raggruppa workspace → team → progetto → persona → issue; **Per team** raggruppa team qualificato dal workspace → progetto → persona → issue. Mostrare totali giornalieri e di periodo per ciascun gruppo, invarianti rispetto agli altri raggruppamenti amministrativi. Nella vista personale workspace/team restano visibili e filtrabili senza accesso ai dati altrui. |
| TS-022 | Applicare ruolo, membership, visibilità e restrizioni PAT separatamente per ogni workspace prima di sommare. Un privilegio in A non si propaga a B. Le viste amministrative includono soltanto workspace con ruolo owner/admin; il timesheet personale include i propri worklog visibili anche negli altri workspace ammessi. |
| TS-023 | Usare le stesse date di calendario per confrontare i workspace, senza conversione di `work_date`. Esporre fuso di riferimento e, nel dettaglio/registrazione, fuso e oggi del workspace interessato. La selezione del periodo corrente usa il riferimento dichiarato; future date, attivazione e limite 24h si valutano per workspace. |
| TS-024 | Riutilizzare ove possibile struttura, componenti e convenzioni di Insights: header, toolbar dei filtri, tab, icone team, selettori, date, stati loading/error, token e layout responsive. Conservare nell'URL modalità, periodo, riferimento e filtri autorizzati; rileggerli all'apertura e con navigazione avanti/indietro, validando sempre l'accesso corrente. Adattare il selettore date ai periodi Timesheet, senza importare metriche o preset estranei. |

## 5. Regole di aggregazione e casi limite

1. **Fonte e misura.** Sommare `duration_minutes` dei record correnti di
   `effort_worklogs`, selezionati tramite `work_date` inclusa tra gli estremi
   del periodo. Non usare `created_at`, eventi storici, stime o residui.
2. **Filtraggio prima della somma.** Applicare insieme autorizzato di workspace/team, accesso corrente,
   ambito personale/amministrativo, periodo e filtri prima di raggruppare,
   calcolare conteggi o paginare. Identificativi forniti dal client non
   costituiscono autorizzazione.
3. **Una sola contribuzione.** Ogni worklog contribuisce una volta al totale
   generale, anche con parent/sub-issue o assegnatari multipli. L'espansione di
   una riga non cambia i totali; subtotali e dettagli non si risommano fra loro.
4. **Progetto corrente.** Usare `issues.project_id`, senza ereditare il progetto
   del parent. `NULL` confluisce in **Senza progetto**. Uno spostamento da A a B
   riclassifica anche le ore pregresse in B, lasciando invariati minuti e date.
   La vista deve spiegare che il raggruppamento usa il progetto attuale.
5. **Privacy dei riferimenti.** Non esporre titoli, nomi, identificativi,
   descrizioni, suggerimenti o conteggi di dati non accessibili. Se un riferimento
   progetto non è visibile, non mostrarne i metadati né confonderlo con `NULL`:
   l'eventuale contributo dell'issue visibile resta in un gruppo neutro
   **Progetto non disponibile**, senza consentire di ricostruirne l'identità.
6. **Completezza relativa all'accesso.** I totali rappresentano il lavoro
   consultabile dall'operatore, anche quando guarda il timesheet altrui.
   Mostrare una nota generale «Totali relativi ai dati a cui hai accesso»;
   non rivelare se esistano record esclusi o quante ore contengano.
7. **Identità storiche.** Raggruppare per ID stabile del titolare, non per nome
   o membership corrente. Account eliminati distinti restano gruppi distinti,
   con etichetta neutra e senza inventare nomi storici non conservati. La
   rimozione della membership non elimina il lavoro né abilita nuova delega.
8. **Date.** `work_date` è una data di calendario. Il workspace di apertura
   fornisce il fuso di riferimento, esplicito e stabile mentre si filtrano altri
   workspace. «Periodo corrente» usa quel fuso; oggi e validazione delle
   scritture usano il fuso del workspace destinatario. Il dettaglio distingue
   i due quando necessario. Alla revoca del workspace di riferimento occorre
   selezionarne uno accessibile, conservando il periodo già scelto. Cambiare
   fuso non sposta date storiche, neppure nei cambi dell'ora legale.
9. **Correzioni e cancellazioni.** Una modifica data/durata sposta o aggiorna
   il contributo; eliminare un worklog lo rimuove dai totali. Gli eventi
   storici di un worklog eliminato non ricreano ore. Eliminare una issue
   rimuove i suoi worklog secondo il comportamento Efforts attuale; il
   timesheet non è un archivio contabile immutabile.
10. **Limite giornaliero.** Resta 1.440 minuti per titolare/data/workspace,
    verificato anche sulle issue non visibili all'operatore. Il totale
    visibile non è una misura affidabile del budget disponibile. Gli errori
    non devono rivelare dettagli nascosti.
    Il totale cross workspace può superare 24h per data: non introdurre un
    limite globale né interpretare il timesheet come rilevazione di presenze.
11. **Paginazione.** Totali generali e giornalieri devono riferirsi all'intero
    insieme filtrato, non alla pagina caricata. La paginazione dei gruppi o
    dei dettagli deve essere esplicita; un insieme incompleto non può apparire
    completo. Totali e righe di una risposta devono essere coerenti rispetto
    allo stesso snapshot di lettura.
12. **Identità e team.** Usare ID stabili qualificati dal workspace per team,
    progetto e issue, mai nomi o solo codici issue. La stessa persona/account
    può essere sommata tra workspace ammessi; account distinti con nomi uguali
    non vengono fusi. Il team è `issues.team_id` attuale; uno spostamento di
    team riclassifica lo storico senza cambiare i minuti, applicando subito
    la nuova visibilità. Un progetto condiviso fra team riceve ogni worklog
    una volta, nel team della sua issue.
13. **Errori e selezioni.** Un errore tecnico su parte dell'ambito impedisce di
    pubblicare nuovi totali come completi. Una selezione esplicita non più
    autorizzata richiede riallineamento; la UI rimuove i dati revocati e mostra
    il cambiamento senza divulgarne dettagli. Le cache devono distinguere
    utente, modalità, workspace, team, periodo e filtri; non basta lo slug della pagina.

### Esempio verificabile

Nello stesso giorno e workspace, con tutte le issue accessibili:

| Titolare | Progetto attuale | Issue | Inseritore | Minuti |
| --- | --- | --- | --- | ---: |
| Anna | Alfa | A-1 | Anna | 60 |
| Anna | Alfa | A-1 | Admin | 30 |
| Anna | Nessuno | A-2 | Anna | 30 |
| Luca | Alfa | A-1 | Luca | 120 |

- Il timesheet di Anna mostra `1h 30m` su A-1, `30m` su A-2 e `2h` totali.
- Per persona: Anna `2h`, Luca `2h`; totale generale `4h`.
- Per progetto: Alfa `3h 30m`, Senza progetto `30m`; totale generale `4h`.
- I 30 minuti inseriti da Admin restano di Anna. Se A-2 è figlia di A-1,
  il totale generale resta `4h`.
- Spostando A-1 da Alfa a Beta, `3h 30m` passano a Beta anche consultando
  lo stesso giorno storico; il totale generale resta `4h`.

Estensione: i quattro record sopra sono nel workspace W1, team T1. Nel workspace
W2, team T2, Anna ha altri 60 minuti su un progetto anch'esso chiamato Alfa.
Il suo timesheet personale W1+W2 totalizza `3h`; per un admin di entrambi il
report completo totalizza `5h`, ripartito W1/T1 `4h` e W2/T2 `1h`. I due Alfa
non si fondono. Un chiamante admin solo in W1 e member in W2 vede nel report
amministrativo solo W1 (`4h`), anche se può consultare il proprio lavoro in W2.

## 6. Direzione tecnica proposta

Questa sezione indica come integrare il requisito nella base attuale, senza
fissare un contratto API come se fosse già implementato.

| Area | Intervento previsto |
| --- | --- |
| Navigazione UI | Pagina proposta `/{workspaceSlug}/timesheet`, collegata dalla [Sidebar](../../../UI/src/lib/components/layout/Sidebar.svelte). Lo slug fornisce contesto di navigazione e fuso di riferimento, non un limite ai dati selezionati. Filtri cross workspace espliciti e link alle issue nel rispettivo workspace. |
| Letture backend | Estendere Efforts nei livelli handler → service → repository con letture utente trasversali e insieme esplicito di workspace autorizzati. Il contesto cross workspace già previsto da `domain.Access` non sostituisce il controllo di ruolo/PAT su ciascun workspace. Non enumerare issue e worklog dal browser. |
| Contratto | Periodo comune, riferimento, modalità e filtri multipli workspace/team/persona/progetto; risposta con ambito effettivo, impostazioni/capacità per workspace, gruppi, totali completi e paginazione. Dettaglio e opzioni dei filtri seguono gli stessi controlli. Definire contratto di errore senza successo parziale implicito. |
| Autorizzazione | Controllare sul server membership e ruolo di ogni workspace, `efforts:read`, allowlist PAT e visibilità issue/team/progetto prima delle somme. Non riusare il ruolo del workspace nella URL come autorizzazione globale; richieste con ID espliciti non autorizzati vengono rifiutate. |
| Scritture | Riutilizzare route issue Efforts e dialogo, passando esplicitamente il workspace destinatario, configurazione e capacità locali. Nessuna scrittura multipla tra workspace o nuovo comando di modifica totali. |
| Persistenza | Nessuna tabella di totali o copia dei worklog necessaria. Valutare eventuali nuovi indici con `EXPLAIN (ANALYZE, BUFFERS)` sulle query reali; gli indici attuali coprono issue/pagina e persona/data. |
| Aggiornamento | Coprire tutti i workspace selezionati: il WebSocket del layout attuale copre solo lo slug aperto. Estendere la sottoscrizione autorizzata e la relativa invalidazione, con cleanup al cambio ambito, riconnessione e rilettura dei permessi. Definire il trasporto nel refinement senza payload privati indiscriminati. |
| Portabilità | Ricostruire le viste dai dati Efforts importati; nessuna duplicazione di worklog o dipendenza da aggregati esportati. |

Nel refinement tecnico vanno definiti nomi finali degli endpoint, DTO, ordinamento
stabile e paginazione dei gruppi/dettagli. Le query devono avere intervalli
limitati ai periodi supportati, controlli sugli interi e nessun troncamento
silenzioso. Misurare viste mensili personali e aggregate con più progetti,
persone, workspace con ruoli/fusi diversi, team privati e oltre 50 worklog; documentare dataset, ambiente e tempi,
senza dedurre SLA dalle misure dei riepiloghi issue già esistenti.

### Riuso concreto della pagina Insights

| Elemento verificato | Impiego in Timesheet |
| --- | --- |
| Header con `SidebarToggle`, toolbar e contenitore responsive | Riprendere gerarchia, spaziature, colori e token esistenti; consentire alla griglia mensile lo scorrimento orizzontale. |
| Primitive `Tabs`, `Select`, `TeamIcon` | Riutilizzarle dove compatibili; adattare i filtri alla selezione multipla e qualificare le opzioni con il workspace. |
| `AnalyticsDateRangePicker`, `Popover`, `RangeCalendar` | Riutilizzare primitive o estrarre il minimo componente condiviso necessario. I preset esistenti 30/90/180 giorni e il loro calcolo nel fuso browser non soddisfano TS-002/023: non copiarli nel Timesheet. |
| `LoadingState`, `ErrorState` e gestione risposte tardive | Riutilizzare componenti e convenzioni, estendendo l'identità della richiesta a tutto l'ambito cross workspace. |
| Parametri URL di Insights/Explorer | Applicare la stessa convenzione per ripristinare modalità/filtri; ogni apertura rivalida i permessi. Il link non concede accesso e non costituisce condivisione pubblica. |
| `InsightsExplorer` e grafici analytics | Non riutilizzare le query sulle issue come fonte del consuntivo; la griglia usa worklog e relativi titolari. Nessuna riscrittura generale di Insights necessaria. |

## 7. Criteri di accettazione

| ID | Scenario ed esito atteso | Copertura |
| --- | --- | --- |
| TS-AC-01 | Un member apre Timesheet senza filtri: settimana corrente nel fuso di riferimento, tutti i workspace ammessi e soli propri worklog, incluse registrazioni delegate. | TS-001–004, TS-019 |
| TS-AC-02 | Settimana/mese, precedente/successivo e periodo corrente producono estremi corretti, inclusi cambio anno, febbraio bisestile e cambio ora; nessuna data storica viene spostata. | TS-002–003 |
| TS-AC-03 | Due registrazioni sulla stessa issue/giorno sommano in una cella e restano due elementi distinti nel dettaglio. I totali dell'esempio della sezione 5 coincidono. | TS-004, TS-007, TS-017 |
| TS-AC-04 | Issue chiusa, non assegnata al titolare, senza stima o senza progetto compare se ha lavoro autorizzato nel periodo. | TS-005–006 |
| TS-AC-05 | Owner/admin passa da Per persona a Per progetto: gerarchie corrette, totale generale identico e filtri conservati; può aprire il timesheet individuale e i worklog. | TS-008–011 |
| TS-AC-06 | Filtri persona/progetto combinati aggiornano insieme righe, celle, subtotali e totale; Senza progetto seleziona solo issue senza progetto. | TS-006, TS-011 |
| TS-AC-07 | Member/guest tenta API amministrativa, persona altrui o modalità manipolata: accesso negato senza restituire dati aggregati altrui. Lead di progetto/member con `project:manage` non ottiene privilegi aggiuntivi. | Sezione 3, TS-008 |
| TS-AC-08 | Guest vede il proprio eventuale storico accessibile ma non può creare/modificare/eliminare; senza storico vede uno stato vuoto. | Sezione 3, TS-012, TS-018 |
| TS-AC-09 | Issue/team privati non accessibili, anche usando ID noti, non compaiono in dettagli, somme, conteggi o suggerimenti. La nota sui totali non rivela l'esistenza di dati nascosti. | Sezioni 3 e 5 |
| TS-AC-10 | Workspace non selezionati non contribuiscono; ID di workspace non autorizzati in URL/API vengono rifiutati senza esporre dati. Il semplice cambio dello slug di navigazione non concede accesso. Un non membro e un link pubblico non accedono ai dati. | Sezione 3, TS-019, TS-022 |
| TS-AC-11 | PAT senza efforts:read non legge; con lo scope legge entro ruolo corrente e allowlist workspace_slugs. Tutti esclude workspace fuori allowlist; selezionarli esplicitamente fallisce. Un downgrade del proprietario revoca l’accesso amministrativo solo negli ambiti interessati. | Sezione 3, TS-022 |
| TS-AC-12 | Parent/sub-issue e assegnatari multipli non duplicano minuti. Lo spostamento di progetto riclassifica lo storico come dichiarato, senza cambiare il totale; non viene ereditato il progetto del parent. | TS-004, TS-009–010; sezione 5 |
| TS-AC-13 | Rimozione membership e account non disponibile non fanno sparire i contributi amministrativi autorizzati; ID distinti non vengono fusi e non sono disponibili per nuova delega. | TS-012–013 |
| TS-AC-14 | Registra lavoro sceglie esplicitamente workspace e issue anche assente dalla griglia; da cella precompila il workspace corretto. 1 e 1d valgono 480 minuti, 1h 30m vale 90; la griglia mostra ore/minuti esatti. Nessuna scrittura va allo slug sbagliato. | TS-013, TS-017 |
| TS-AC-15 | Registrazione per periodo crea le celle giornaliere attese e rispetta l'opzione weekend. Data futura, durata invalida o superamento di 24h in una data annullano l'intera operazione, anche con contributi nascosti. | TS-003, TS-013; sezione 5 |
| TS-AC-16 | Modificare data/durata aggiorna entrambi i giorni interessati; eliminare rimuove il contributo ma non lo storico. Titolare immutabile; correzione/delete non ricalcolano il residuo. | TS-014; sezione 5 |
| TS-AC-17 | Due modifiche concorrenti non si sovrascrivono silenziosamente; un retry dopo risposta persa non duplica ore. Nuovo lavoro mantiene il comportamento Efforts sul residuo. | TS-013–014 |
| TS-AC-18 | Efforts disabilitato in A resta leggibile e non scrivibile, anche da dialogo già aperto; Efforts attivo in B resta scrivibile secondo ruolo. Riabilitare A non duplica i worklog. | TS-015 |
| TS-AC-19 | Modifiche remote in A e B aggiornano una vista aperta dallo slug A. Reconnect, revoca e cambio selezione invalidano gli ambiti interessati; risposte tardive non ripristinano dati precedenti né sovrascrivono bozze. | TS-016 |
| TS-AC-20 | Oltre 50 worklog e più pagine di gruppi: totale e subtotali sono completi, pagine ordinate stabilmente, nessun duplicato su dataset immutato e dettaglio interamente raggiungibile. | TS-007, TS-009–010; sezione 5 |
| TS-AC-21 | Errore di rete, anche su uno dei workspace selezionati, non produce zero o un totale parziale presentato come completo. Stati vuoti distinti, tastiera, viewport stretto e focus funzionano. | TS-016, TS-018 |
| TS-AC-22 | Eliminare una issue rimuove solo i contributi effettivamente eliminati; worklog dei figli conservati restano nel report. Export/import v2 conserva i totali a parità di dati/accesso; import v1 senza Efforts produce vista vuota. | Sezioni 5–6 |
| TS-AC-23 | Un account admin in A e member/guest in B consulta il proprio lavoro in A+B; nelle quattro viste amministrative B è escluso e il cambio ambito è esplicito. Una richiesta amministrativa esplicita A+B fallisce; il ruolo di A non si propaga. | TS-008, TS-019, TS-022 |
| TS-AC-24 | Selezione multipla workspace/team e filtri persona/progetto applicano OR interno e AND fra dimensioni. Eliminare un workspace rimuove i soli filtri figli non pertinenti, con feedback; nessuna selezione vuota si amplia a Tutti. Team/progetti omonimi restano distinguibili. | TS-006, TS-011, TS-019–020 |
| TS-AC-25 | Le quattro aggregazioni producono gli stessi totali cross workspace dell'esempio della sezione 5; Per workspace e Per team mostrano subtotali corretti. Un progetto tra team non duplica ore; il team dell'issue, non quello del titolare/progetto, decide filtro e gruppo. Spostarlo riclassifica lo storico e applica la nuova visibilità. | TS-004–005, TS-009–010, TS-020–021 |
| TS-AC-26 | Workspace con fusi in cui oggi differisce: le date registrate non si spostano, il periodo comune resta stabile cambiando filtri e una nuova registrazione è validata con oggi locale del destinatario. 24h in A e 1h in B producono 25h aggregate senza introdurre un limite globale. | TS-002–003, TS-023 |
| TS-AC-27 | Header, toolbar, tab, stati e token Timesheet sono coerenti con Insights; i selettori consentono più workspace/team. Il periodo usa settimana/mese e il riferimento dichiarato, senza preset analytics o dipendenza implicita dal fuso browser. | TS-018, TS-024 |
| TS-AC-28 | Riapertura URL e navigazione avanti/indietro ripristinano modalità, periodo, riferimento e filtri. Revoche impediscono il ripristino di dati non più accessibili; link issue e dialoghi operano nel workspace corretto. | TS-005, TS-016, TS-024 |

### Verifica prevista per l'implementazione

- Test repository con PostgreSQL reale per aggregazioni, privacy, identità
  storiche e coerenza di righe/totali; test HTTP per matrice ruoli/PAT e filtri
  manipolati.
- Test unitari per periodi e formati e test E2E per navigazione, griglia,
  dettaglio, riuso dialoghi, errori, revoche e aggiornamenti.
- Regressione mirata delle suite Efforts esistenti per scritture, residui,
  limite giornaliero, idempotenza, conflitti e disattivazione.
- Verifica della resa settimanale/mensile e misure delle query su dataset
  rappresentativi, con risultati registrati prima di dichiarare completato TS-01.

## 8. Fuori perimetro

Approvazioni, invio/chiusura dei timesheet, blocco dei periodi, ferie e festività,
capacità/orario contrattuale, ore mancanti o straordinari, fatturazione/costi,
timer e orari inizio/fine, notifiche dedicate, export CSV/PDF, report condivisi,
intervalli arbitrari, aggregazione tra istanze KUAYLE diverse, nuovi ruoli manager o deleghe
configurabili, integrazione Jira/Tempo e attribuzione storica dei progetti.

Non è previsto un confronto stima/consuntivo per persona: Efforts memorizza
stime sulle issue e non una loro ripartizione attendibile fra persone e giorni.

## 9. Riferimento Tempo

Fonti ufficiali consultate il 2026-10-20; Cloud e Data Center sono indicati
separatamente. Sono riferimenti funzionali, non specifiche di compatibilità.

- [Tempo Timesheets Data Center — Selecting View Options in the Timesheet View](https://help.tempo.io/timesheets-dc/latest/selecting-view-options-in-the-timesheet-view-tempo):
  selezione del periodo, presentazione a griglia/lista e opzioni di consultazione.
  TS-01 adotta griglia e periodi settimana/mese; il dettaglio copre la lettura
  delle singole registrazioni.
- [Tempo Timesheets Cloud — Generating Reports](https://help.tempo.io/timesheets/latest/generating-reports):
  report sui worklog con filtri, raggruppamenti e autorizzazioni per consultare
  il lavoro altrui. TS-01 traduce questi concetti nei ruoli e nelle regole di
  visibilità KUAYLE.

## 10. Esito di questa attività

Aggiornato il requisito in `docs/requirements/timesheets/` e predisposto il
[backlog Epic e Stories](../../backlog/timesheets.md). Esaminati codice Efforts,
Insights e confini di accesso; verificati collegamenti e tracciabilità. Nessuna implementazione
applicativa o esecuzione delle suite di prodotto fa parte di questa consegna.

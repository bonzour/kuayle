# Timesheets — contratti API EP-01–EP-05

Data: **2026-10-21**. Fonte: [TS-01 v0.2](README.md).

## Stato e autorizzazione

Sono operativi `GET /api/timesheets/options`, `GET /api/timesheets`,
`GET /api/timesheets/worklogs` e `GET /api/timesheets/issues`. Griglia e dettaglio
supportano `personal`, `person`, `project`, `workspace` e `team`. EP-05 integra
le scritture attraverso le route issue Efforts esistenti.

Ogni endpoint richiede autenticazione e `efforts:read` per i PAT. Il repository
interseca membership corrente, ruolo, visibilità e allowlist `workspace_slugs`.
Allowlist `null` significa nessuna restrizione aggiuntiva, `[]` nessun workspace.
Owner/admin abilita le modalità amministrative soltanto nei rispettivi workspace;
lead e `project:manage` non conferiscono questo privilegio. Le sessioni guest
possono usare la modalità personale. Efforts disabilitato conserva le letture.

Lo slug della pagina non limita né autorizza il dominio dei dati. Non è necessario
che un PAT abbia `workspaces:read`, `teams:read` o `projects:read` per ottenere i
metadati minimi del nuovo endpoint. Le API Efforts esistenti non cambiano policy.

## Selezioni comuni

| Parametro | Valori |
| --- | --- |
| `mode` | `personal` (default), `person`, `project`, `workspace`, `team` |
| `workspace_ids` | `all`, lista UUID separati da virgole, oppure stringa vuota |
| `team_ids` | Stessa codifica |
| `project_ids` | Stessa codifica; `none` indica progetto SQL `NULL`, combinabile con UUID |

L'omissione di una dimensione equivale a `all`; un parametro presente ma vuoto
è una selezione esplicita senza risultati. Non viene ampliata automaticamente.
`all` non è combinabile con UUID o `none`. UUID duplicati vengono deduplicati;
UUID nullo, elementi vuoti nelle liste, parametri sconosciuti e chiavi ripetute
vengono rifiutati. Si applica OR entro una dimensione e AND fra dimensioni.

Workspace, team e progetti sono identificati per UUID; i metadati dei figli
comprendono `workspace_id`. La risposta non rivela dati o conteggi inaccessibili.
Selezionare esplicitamente un figlio esterno ai workspace scelti fallisce.
La UI rimuove prima i filtri figli incompatibili quando cambia workspace.

## Endpoint operativo: opzioni

`GET /api/timesheets/options` accetta esclusivamente i quattro parametri comuni.
Non richiede un periodo e non enumera persone. Team e progetti derivano dalle
risorse attualmente accessibili, anche senza worklog; le opzioni progetto non
vengono limitate al team del progetto, perché il filtro lavoro usa il
team dell'issue. I progetti invisibili non diventano opzioni «Senza progetto».

Risposta `200`:

```json
{
  "mode": "personal",
  "available_modes": ["personal", "person", "project", "workspace", "team"],
  "workspaces": [{
    "id": "11111111-1111-4111-8111-111111111111",
    "name": "Workspace A",
    "slug": "a",
    "efforts_enabled": true,
    "timezone": "Europe/Rome",
    "today": "2026-10-20",
    "can_administer": true,
    "can_record_work": true
  }],
  "teams": [],
  "projects": [],
  "scope": {
    "workspace_ids": ["11111111-1111-4111-8111-111111111111"],
    "team_ids": [],
    "project_ids": [],
    "include_no_project": true
  }
}
```

`workspaces` contiene tutti gli ammessi alla modalità, anche se non selezionati;
`teams` e `projects` contengono le opzioni dei soli workspace scelti. Ogni team
espone `id`, `workspace_id`, `name`, `key`, `icon`, `color`; ogni progetto
`id`, `workspace_id`, `name`. `scope` espone gli ID effettivi dopo la selezione.
`include_no_project` è vero con progetto `all` o `none`; non implica che esistano
contributi e non allarga un'altra dimensione vuota.

Le liste sono ordinate per nome, poi workspace/ID per distinguere omonimi;
gli ID di `scope` sono ordinati per UUID. Le opzioni non sono paginate né
accompagnate da conteggi di lavoro. Tutto deriva da un unico statement SQL.

`can_record_work` considera attivazione, ruolo e scope PAT read/write. È una
capacità di base: non garantisce accesso a una specifica issue, delegabilità o
budget giornaliero disponibile. I comandi Efforts applicano sempre i controlli
definitivi. Il selettore delega resta soggetto ai vincoli di sessione esistenti.

| Stato | Significato |
| --- | --- |
| `400 INVALID_PARAMS` | Modalità/codifica non valida o parametri non supportati |
| `401 UNAUTHORIZED` | Autenticazione assente, scaduta o PAT revocato |
| `403 FORBIDDEN` | Scope mancante o nessun workspace amministrabile per una modalità amministrativa |
| `404 NOT_FOUND` | Selezione esplicita non disponibile; risorsa inesistente e inaccessibile indistinguibili |
| `500 INTERNAL_ERROR` | Nessuna risposta parziale; ricaricare l'intero ambito |

Le risposte autenticate usano `Cache-Control: private, no-store`. Un ambito
personale privo di workspace restituisce liste vuote; non equivale a un errore.

## Lettura operativa: griglia personale e amministrativa

`GET /api/timesheets` accetta i filtri comuni e `mode=personal` (default).
Le modalità amministrative applicano il ruolo owner/admin separatamente in
ciascun workspace; senza ambiti amministrabili restituiscono `403 FORBIDDEN`.
Sono obbligatori `period=week|month`, `from`, `to` in formato `YYYY-MM-DD` e
`reference_workspace_id`. Il periodo deve essere un'intera settimana
lunedì–domenica o un intero mese; intervalli arbitrari non sono ammessi.
`page` parte da 1; `per_page`, se presente, deve essere `50`.
Parametri sconosciuti o ripetuti, pagine non valide e filtri del dettaglio
su questa route restituiscono `400 INVALID_PARAMS`.

La UI ricava il periodo corrente da `today` del riferimento. Il riferimento
richiede membership e rispetto dell'allowlist, anche quando non selezionato
nei dati; non autorizza altri ambiti. Se viene revocato l'API risponde `404`:
la UI sceglie esplicitamente il primo workspace personale accessibile,
conservando le date e segnalando il cambiamento.

In `personal` il titolare è l'utente autenticato: `owner_user_ids`, se presente,
accetta soltanto il proprio UUID; altri ID restituiscono `403 FORBIDDEN`.
Nelle modalità amministrative `owner_user_ids` accetta omissione o `all` per
tutte le persone, una lista UUID deduplicata oppure stringa vuota per nessuno.
Valori nulli, liste miste con `all`, elementi vuoti o non UUID sono invalidi.
Un UUID valido senza contributi autorizzati produce zero risultati senza
rivelare l'esistenza dell'account. I titolari derivano dai worklog, inclusi ex
membri, e non acquisiscono capacità di delega. L'accesso rimane circoscritto ai
workspace owner/admin, senza aggiungere il proprio lavoro in altri workspace.

La risposta comprende `mode`, `period`, `from`, `to`, `days`, `reference`,
`workspaces` con capacità locali, `scope`, `groups`, `daily_totals`,
`total_minutes`, `people`, `empty_reason` e paginazione
`total_count`, `page`, `per_page`, `has_more`. La paginazione riguarda i gruppi radice
con contributi (workspace nel personale, persone/progetti/workspace/team negli
aggregati); ogni gruppo restituisce gerarchia e subtotali completi.
Sono previsti questi raggruppamenti:

- personale: workspace → team → progetto → issue;
- persona: titolare → workspace → team → progetto → issue;
- progetto: progetto qualificato dal workspace → team → titolare → issue;
- workspace: workspace → team → progetto → titolare → issue;
- team: team qualificato dal workspace → progetto → titolare → issue.

Ogni gruppo espone `key`, `type`, `id`, `name`, `workspace_id`,
`workspace_slug`, `daily_totals`, `total_minutes` e `children`; le issue
aggiungono `identifier`, i team i metadati `team`. `daily_totals` è una lista
`{date, minutes}` completa di tutti i giorni, inclusi gli zeri.
I gruppi sono ordinati stabilmente per nome/tipo e ID, qualificati dal workspace.
`type=no_project` e `type=unavailable_project` hanno `id=null` e nome vuoto:
la UI traduce le due etichette senza ottenere l'identità del progetto privato.
I gruppi `type=person` aggiungono `person` con `id`, `available`, `name` e
`display_name`, come l'identità Efforts. Nei gruppi persona alla radice
`workspace_id` e `workspace_slug` sono nulli perché il titolare può contribuire
a più workspace. Le righe subordinate conservano il workspace corretto.
I nomi di un'identità non disponibile sono nulli; il nome generico del gruppo
è vuoto e la UI mostra l'etichetta tradotta con ID abbreviato. Le chiavi dei
gruppi comprendono il percorso completo e non identificano record modificabili.

`people` contiene le identità distinte dei titolari con contributi nel periodo
e nell'ambito workspace/team/progetto selezionato, prima del filtro persone
e della paginazione. Non enumera membri senza contributi e non contiene conteggi.
Una selezione persone senza risultati non nasconde le altre opzioni autorizzate.

Totali e subtotali includono l'intero insieme filtrato, indipendentemente
dalla pagina. Minuti e conteggi devono rientrare nell'intervallo JSON sicuro;
un superamento causa errore, senza troncamenti o arrotondamenti.
`empty_reason` è `null` quando esistono gruppi, `period` se non esiste lavoro
accessibile nella modalità richiesta nel periodo, `filters` se il lavoro esiste
ma i filtri lo escludono. La selezione vuota resta vuota. Una pagina oltre l'ultima
mantiene conteggi e totali e restituisce `groups=[]`; la UI riallinea la pagina.

## Lettura operativa: dettaglio personale e amministrativo

`GET /api/timesheets/worklogs`.

Usa lo stesso contratto di ambito/periodo, più `issue_id` e `work_date`
facoltativi per restringere una cella. Nelle celle amministrative la UI aggiunge
`owner_user_ids` con il solo titolare della riga, mantenendo modalità e altri
filtri. Restituisce i campi comuni della griglia
(senza `groups`/`people`/`empty_reason`) e `data`. Ogni worklog
contiene ID, contesto `workspace`/`team`, `issue_identifier`, `issue_title`,
titolare `owner`, inseritore `operator`,
`work_date`, `duration_minutes`, descrizione e timestamp. Date e minuti sono
quelli dei record correnti, non degli eventi storici.

Ordinamento `work_date DESC, created_at DESC, id DESC`; tutte le pagine sono
raggiungibili su dataset immutato. Righe, somme e conteggi di ciascuna risposta
derivano dallo stesso snapshot, con autorizzazione prima di aggregare/paginare.
Non è promesso uno snapshot persistente fra richieste diverse.

In entrambe le letture, un errore in qualunque ambito impedisce la risposta
completa. `work_date` non viene convertita fra fusi. Team/progetto sono quelli
attuali dell'issue; i comandi di scrittura restano sulle route Efforts esistenti.

## Ricerca issue e operazioni dell'incremento EP-05

`GET /api/timesheets/issues?workspace_id=<UUID>&search=<testo>&page=1`.
Richiede autenticazione e `efforts:read`, senza richiedere `issues:read`.
`workspace_id` è obbligatorio e identifica un solo workspace con membership
corrente, entro l'allowlist PAT. `search` è facoltativo e cerca sottostringhe
case-insensitive nel titolo o identificativo; vuoto mostra le issue recenti.
`page` parte da 1 e le pagine contengono 50 elementi; `per_page`, se presente,
deve essere `50`. Parametri ripetuti/sconosciuti o non validi producono 400
`INVALID_PARAMS`; workspace inesistenti/inaccessibili producono lo stesso 404.

La risposta espone `workspace` con le capacità locali e `data` con sole issue
`{id, identifier, title, team}`, più `total_count`, `page`, `per_page`, `has_more`.
Ordine `updated_at DESC, id DESC`; autorizzazione, metadati, conteggio e pagina
condividono un unico snapshot SQL. La ricerca include issue prive di worklog,
senza applicare periodo o filtri della griglia. Guest, PAT in sola lettura ed
Efforts disabilitato conservano letture autorizzate ma non capacità di scrittura.
Non sono esposti descrizioni o altri metadati completi delle issue.

La toolbar offre Registra lavoro se almeno un workspace personale è scrivibile,
anche quando fuori dalla selezione amministrativa. La scelta globale richiede
workspace e issue espliciti; le celle issue vuote scrivibili precompilano
workspace, issue e giorno. Le celle con ore aprono il dettaglio, che offre
registrazione, modifica ed eliminazione dei singoli worklog autorizzati.
Subtotali e totali non sono record modificabili.

Prima di modificare/eliminare si rileggono capacità locali, worklog corrente e
revisione, raggiungendo anche le pagine successive delle API Efforts. Ogni
scrittura usa le route issue Efforts nel workspace destinatario; `efforts:read`
e `efforts:write` restano obbligatori per i PAT. La selezione delega conserva il
vincolo di sessione e i destinatari ammissibili per l'issue. Un titolare
precompilato non delegabile viene sostituito dal chiamante con feedback esplicito.

Il dialogo condiviso rilegge configurazione, fuso e oggi del destinatario;
conserva parser, periodi atomici, weekend, limite di 1.440 minuti per
titolare/data/workspace, storico, residuo, revisioni e retry idempotenti Efforts.
Un rifiuto conserva la bozza e blocca nuovi invii; il riallineamento esplicito
sostituisce la bozza con i dati correnti. Il dialogo è indipendente dai
caricamenti della griglia. Dopo successo si ricaricano griglia e dettaglio,
riallineando l'eventuale ultima pagina eliminata, senza ampliare filtri o periodo.
Un salvataggio fuori dai filtri correnti viene comunque confermato.

Le proprietà aggiuntive di `EffortWorklogDialog` sono opzionali: i chiamanti issue
mantengono i propri default. Le sottoscrizioni remote multi-workspace e
l'accettazione integrata restano EP-06.

## Navigazione UI dell'incremento EP-02

Il calendario usa settimana/mese, precedente/successivo/corrente e il `today`
del riferimento, senza preset analytics. Cambiare settimana/mese usa la data
iniziale del periodo visualizzato come ancora. La griglia usa ore/minuti esatti,
non conversioni in giorni da 480 minuti. Celle issue/giorno con contributi aprono
il dettaglio; i subtotali non sono registrazioni modificabili.

Il ritorno dall'issue usa lo stato della voce di cronologia SvelteKit, associato
all'utente e allo slug della pagina Timesheet. Periodo, riferimento, modalità,
filtri e pagina vengono rivalidati alla riapertura. Questa scelta non implementa
parametri URL condivisibili, né sottoscrizioni remote multi-workspace (EP-04/06).

## Navigazione UI dell'incremento EP-03

Le quattro viste amministrative usano la stessa griglia, periodo e filtri. Il
cambio fra raggruppamenti conserva il filtro persone e reimposta la pagina a 1.
«Apri timesheet» in Per persona imposta il filtro sul titolare nella stessa
pagina; non apre la modalità personale né amplia i workspace. Tornando al
personale il filtro persone viene azzerato e il titolare resta il chiamante.
Il riferimento temporale viene rivalidato nell'ambito personale e può quindi
restare un workspace dove il chiamante è member/guest. Lo stato di cronologia
include persone e modalità; i parametri URL condivisibili rimangono a EP-04.

## Navigazione UI dell'incremento EP-04

L'URL sostituisce lo stato di sessione descritto per EP-02/03 come fonte della
vista. Il percorso resta `/{workspaceSlug}/timesheet`; lo slug fornisce il
contesto iniziale, non restringe né autorizza i dati.

| Parametro | Significato |
| --- | --- |
| `mode` | `personal`, `person`, `project`, `workspace`, `team`; predefinito `personal`. |
| `period` | `week` o `month`; predefinito `week`. |
| `from`, `to` | Coppia di date `YYYY-MM-DD` che descrive un'intera settimana lunedì–domenica o un intero mese. Se assenti, periodo corrente secondo `today` del riferimento. |
| `reference_workspace_id` | Workspace di riferimento, anche esterno alla selezione dati amministrativa purché accessibile nell'ambito personale. Se assente, workspace di apertura o primo accessibile. |
| `workspace_ids`, `team_ids`, `project_ids` | ID separati da virgole; omissione o `all` significa Tutti, valore vuoto significa selezione vuota. Solo i progetti ammettono `none` per Senza progetto. |
| `owner_user_ids` | Stessa semantica delle selezioni, solo nelle viste amministrative. Nel personale viene rimosso e il titolare resta l'utente autenticato. |
| `page` | Pagina dei gruppi, predefinita 1; cambia solo la pagina, non i totali. |

Dopo la risoluzione del riferimento e del periodo l'URL esplicita la vista.
Parametri estranei alla vista e frammento vengono conservati, senza inoltrarli
alle API. Parametri noti duplicati, periodi incompleti o non canonici e selezioni
malformate mostrano un errore tradotto; il ripristino dei default richiede
un'azione esplicita. Le API mantengono la validazione autorevole degli ID.

SvelteKit `goto` crea una voce per ogni cambiamento utente di modalità, periodo,
filtro o pagina, mantenendo focus e scorrimento. Inizializzazione, correzione
della pagina e riallineamenti di accesso sostituiscono la voce corrente.
Refresh, apertura/chiusura e paginazione del dialogo non aggiungono voci.
I cambi ai filtri, al periodo o alla modalità riportano la pagina dei gruppi a 1.

Apertura, reload e avanti/indietro rileggono selettori e dati autorizzati.
Revoche rimuovono le sole selezioni incompatibili senza trasformare una lista
vuota in Tutti; un downgrade elimina la modalità amministrativa e il filtro
persone, senza ampliare ai workspace personali estranei. Il riferimento revocato
viene sostituito esplicitamente conservando le date scelte. Dati e dialoghi
precedenti vengono invalidati prima della lettura; risposte obsolete non
riscrivono né risultati né URL.

I cinque tab riusano le primitive Insights, hanno attivazione manuale con
Invio/Spazio e mostrano soltanto le modalità ammesse dal server. I link alle
issue usano il workspace della riga; il ritorno browser riapre l'URL filtrato.
La chiusura del dettaglio riporta il focus alla cella o, se rimossa, al titolo
della pagina. La griglia mensile mantiene lo scorrimento orizzontale al proprio
interno; barra dei tab e dettaglio sono utilizzabili sui viewport stretti.

## Aggiornamenti remoti dell'incremento EP-06

`GET /api/workspaces/:slug/ws?channel=timesheets` riusa il WebSocket esistente,
con sessione JWT e membership corrente nel workspace richiesto. I PAT restano
esclusi dal socket e leggono attraverso le API REST autorizzate sopra descritte.
Un valore `channel` diverso da `timesheets` o dal valore vuoto viene rifiutato
con HTTP 400. Senza canale il protocollo applicativo preesistente resta valido.

Il canale Timesheet è solo in uscita e produce esclusivamente:

```json
{"type":"app.refresh","payload":{"resources":["timesheets"]}}
```

Non trasporta ore, identificativi di persone/issue, titoli, descrizioni,
presenze o cursori. Le mutazioni issue, Efforts, team, progetti e accessi
invalidano la lettura; le invalidazioni vengono accorpate prima dell'accodamento.
I messaggi applicativi in ingresso non sono inoltrati. La membership viene
ricontrollata prima della consegna e ogni cinque secondi sui socket inattivi;
revoca o errore di verifica chiudono la connessione con codice 1008.

La pagina mantiene un socket per workspace nell'ambito effettivo più il
riferimento temporale, eliminando duplicati. Queste sottoscrizioni non
estendono l'ambito delle query. Cambi selezione, account e navigazione eliminano
socket e timer precedenti; i workspace ancora pertinenti mantengono il socket.
Apertura, notifica e chiusura invalidano le risposte precedenti e rileggono
opzioni e griglia; solo dopo una lettura coerente riparte il dettaglio.
Gli eventi vicini vengono accorpati entro 100 ms; il reconnect usa tre secondi.
Lo stato di disconnessione è visibile e tradotto. Anche una chiusura anomala
richiede la rivalidazione, perché un proxy può nascondere il codice di revoca.

Il refresh conserva URL, periodo e bozze aperte. Le revoche riallineano la
selezione con feedback, senza trasformare una lista vuota in Tutti; un downgrade
non allarga un precedente report amministrativo ai workspace personali estranei.
Le risposte superate non pubblicano dati né riscrivono l'URL. Il dettaglio
sospeso non presenta totali vecchi e gli errori non producono risultati parziali.
Ogni salvataggio mantiene i controlli autorevoli e i conflitti Efforts esistenti.

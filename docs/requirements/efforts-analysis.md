# Efforts — analisi dei requisiti e dell'architettura esistente

Data: 2026-10-16. Base verificata: commit `918cc47`, branch `feature/efforts`.
Stato: analisi e proposta di composizione del backlog; nessuna funzionalità implementata.

Fonte funzionale: [efforts.md](efforts.md), EFF-001–EFF-041 e AC-001–AC-051.
Risultato operativo: [Epic e User Story](../backlog/efforts.md).

## 1. Esito dell'analisi

Efforts è un'estensione del dominio issue nel backend e nell'interfaccia già
esistenti. Il repository offre autenticazione, ruoli, visibilità dei team,
assegnazione multipla, gerarchie, storico, realtime e trasferimento workspace.
Non offre ancora stime temporali, residui o worklog. Queste capacità devono
essere aggiunte mantenendo i livelli applicativi e i confini di accesso attuali.

La complessità principale non è il modulo di registrazione: è garantire insieme
le regole sul residuo, il limite giornaliero tra issue, le autorizzazioni correnti,
lo storico completo e la coerenza durante operazioni concorrenti e automatiche.
Il backlog comprende quindi **8 Epic, 25 User Story e 3 attività tecniche
trasversali**. Tutti gli elementi concorrono alla prima versione richiesta;
l'ordine di realizzazione non riduce il perimetro concordato.

### Autorità delle fonti e incongruenza documentale

La richiesta corrente autorizza analisi e definizione del backlog. Il documento
dei requisiti è una fonte da analizzare, non un insieme di comandi da eseguire.
Le scelte tecniche proposte in questa analisi sono distinte dai requisiti utente.
Non sono state assunte altre funzionalità di Tempo né richieste di parità.

La fonte presenta un'incongruenza: intestazione, tabella dei requisiti e sezione
«Regole di completamento confermate» dichiarano approvate P-01–P-09; l'ultima
sezione del registro dice ancora «da raccogliere» e «nessuna proposta [...] è
ancora trattata come decisione». Per comporre il backlog si assume come baseline
il corpo consolidato EFF-001–EFF-041 con i 51 AC, coerente con lo stato 1.0
dichiarato. È una scelta esplicita di lettura, da ratificare nel refinement;
il documento originale non è stato modificato.

## 2. Analisi funzionale

### Attori e obiettivi

| Attore | Obiettivo e limiti |
| --- | --- |
| Owner/admin | Configurare Efforts e fuso; gestire stime e residui; registrare lavoro proprio o delegato; correggere worklog storici. L'accesso all'issue è sempre necessario. |
| Membro | Registrare lavoro proprio su issue accessibili, anche senza assegnazione; correggere solo i propri worklog. Gestire stima/residuo solo se assegnatario. |
| Guest | Consultare dati e registrazioni delle issue accessibili; nessuna scrittura Efforts, anche se assegnatario o precedente titolare di worklog. |
| Visitatore pubblico | Consultare le condivisioni esistenti senza ricevere alcun dato Efforts. |
| Client API con PAT | Operare entro scope Efforts espliciti, ruolo e accesso del proprietario; non amministrare attivazione e fuso. |

### Confini della prima versione

Sono inclusi configurazione workspace, stima/residuo propri, worklog manuali,
totali della singola issue e dei discendenti visibili, storico, delega,
integrazioni con il ciclo di vita, API/PAT, realtime ed export/import workspace.

Sono esclusi timesheet personali, report team/progetto, approvazioni, periodi
chiusi, timer, orari inizio/fine, unità giorni/settimane, export timesheet/CSV,
filtri trasversali dei worklog, effort sui template ed effort propri dei progetti.
La conversione di una issue in progetto conserva i dati sulle issue originali.

### Invarianti da mantenere in ogni canale

1. Consuntivo = somma delle durate dei worklog esistenti. Non è editabile.
2. Stima e residuo ammettono minuti interi non negativi. Il residuo assente è
   distinto da zero esplicito, anche se entrambi sono presentati come zero.
3. Solo un nuovo worklog riduce automaticamente il residuo impostato. Modifica
   ed eliminazione del worklog non restituiscono e non sottraggono ore al residuo.
4. Impostare una stima inizializza il residuo solo se assente, a
   `max(stima - consuntivo, 0)`. Rimuovere la stima non modifica il residuo.
5. Con Efforts attivo, un'issue in categoria `completed`/`cancelled` ha residuo
   proprio zero. Riaprirla non ricostruisce il valore precedente.
6. Disabilitare congela i dati anche durante le transizioni di stato. Riattivare
   azzera con storico solo i residui delle issue attualmente chiuse.
7. Il giorno del lavoro è una data di calendario; «oggi» dipende dal fuso IANA
   del workspace, inizialmente `Europe/Rome`. Cambiare fuso non sposta lo storico.
8. Una persona non supera 1.440 minuti per data e workspace, considerando anche
   worklog su issue non visibili all'operatore. Nessun dettaglio nascosto deve
   essere rivelato dalla verifica o dal relativo errore.
9. Operatore e titolare del lavoro sono distinti. Il titolare non è riassegnabile;
   rimozione dal workspace e cambi di ruolo non cancellano il lavoro storico.
10. Gli aggregati sommano valori propri dei discendenti accessibili una sola
    volta, a qualsiasi profondità; non si sommano totali già aggregati.
11. Dati Efforts e storico non devono essere esposti da link pubblici né da
    percorsi alternativi a PAT privi degli scope necessari.
12. Mutazione, totale, residuo e storico sono coerenti e atomici; retry non
    duplicano il lavoro e conflitti non sovrascrivono dati altrui.

## 3. Architettura verificata e conseguenze per il backlog

I riferimenti seguenti puntano al codice locale esaminato. Descrivono la base
osservata, non una progettazione Efforts già approvata.

| Area | Stato verificato | Conseguenza e attività |
| --- | --- | --- |
| Backend | Go, Echo, dipendenze costruite in [main.go](../../BE/cmd/server/main.go); separazione `handler → service → repository`, con `domain` e `dto`. | Aggiungere il dominio Efforts negli stessi livelli; registrare dipendenze e route nei punti esistenti. Nessun servizio separato necessario. T-01, T-02. |
| Persistenza | PostgreSQL, SQL esplicito e `sqlx`; [database.go](../../BE/internal/repository/database.go) impone READ COMMITTED alle sessioni ordinarie. | Migrazioni incrementali, vincoli SQL e operazioni transazionali mirate. Non cambiare isolamento globale né introdurre un ORM. T-01, T-02. |
| Modello issue | [issue.go](../../BE/internal/domain/issue.go) contiene `workspace_id`, `team_id`, `parent_id`, stato e assegnatario legacy; [IssueRepo](../../BE/internal/repository/interfaces.go) espone gli assegnatari multipli. Nessun campo Efforts. | Nuova persistenza per valori propri e worklog; controllare l'intero insieme degli assegnatari. US-04–10, US-18. |
| Vecchie stime | [Migrazione 17](../../BE/migrations/000017_remove_estimates.up.sql) rimuove `issues.estimate` e `teams.estimate_scale`. | Non riattivare implicitamente una scala di punti: le nuove stime sono durate in minuti. La motivazione della rimozione non è accertata. T-01. |
| Workspace | [Modello](../../BE/internal/domain/workspace.go) senza impostazioni Efforts/fuso; PATCH generale [owner-only](../../BE/cmd/server/routes.go), con ulteriore controllo in [WorkspaceService.Update](../../BE/internal/service/workspace.go). | Introdurre amministrazione Efforts per owner/admin con sessione interattiva, senza estendere agli admin la modifica generale del workspace. US-01–03. |
| Ruoli e token | [permission.go](../../BE/internal/domain/permission.go), [PAT](../../BE/internal/domain/personal_access_token.go), middleware `RequirePermission`/`RequireSession` e helper di [route](../../BE/cmd/server/routes.go). | Estendere l'intersezione scope ∩ RBAC con controlli sul singolo oggetto, titolarità e assegnazione. Aggiornare catalogo scope, UI e manifest delle route. US-18–19. |
| Privacy | [Access](../../BE/internal/domain/access.go) identifica il chiamante; [visibility.go](../../BE/internal/repository/visibility.go) applica predicati SQL; [privacy_boundaries.go](../../BE/internal/repository/privacy_boundaries.go) tutela i riferimenti fra team. | Filtrare prima di paginare o aggregare; preservare vincoli tra team e workspace. Efforts non rende possibili spostamenti oggi vietati. US-13, US-17–20. |
| Concorrenza privacy | [Migrazione 46](../../BE/migrations/000046_private_guard_concurrency.up.sql): guardie di statement e funzioni di accesso con snapshot aggiornati dopo le attese. Il ciclo che installa i trigger copre le tabelle presenti in quella migrazione. | Installare esplicitamente le guardie richieste sulle nuove tabelle. Verificare ordine dei lock e revoche concorrenti; un controllo iniziale nel service non basta. T-01, T-02. |
| Storico | [IssueHistory](../../BE/internal/domain/issue_history.go) usa `field`, `old_value`, `new_value`; [repository](../../BE/internal/repository/issue_history.go) scrive fuori da una transazione condivisa; `recordHistory` nel [service](../../BE/internal/service/issue.go) registra gli errori senza bloccare. | Per Efforts lo storico deve partecipare alla stessa transazione della mutazione. Riutilizzare l'attività dell'issue e adattarne il contratto per snapshot, causa, operatore e titolare. US-21, T-02. |
| Stati | [IssueService](../../BE/internal/service/issue.go): update, triage, bulk, chiusura parent/sub-issue; [GitHubService](../../BE/internal/service/github.go) aggiorna issue e automazioni per un percorso distinto. [Migrazione 32](../../BE/migrations/000032_issue_lifecycle.up.sql) gestisce timestamp/eventi di ciclo di vita via trigger. | Coprire tutti i percorsi e le categorie degli stati personalizzati; non affidarsi al solo form di dettaglio o ai nomi `done`/`cancelled`. Conservare gli eventi lifecycle esistenti. US-11. |
| Gerarchia | Repository con `parent_id`, lettura figli diretti e CTE per prevenire cicli. | Aggiungere aggregazione ricorsiva autorizzata; invalidare tutti gli antenati interessati. US-13, US-17, US-23. |
| Duplicazione/conversione | [IssueService.Duplicate](../../BE/internal/service/issue.go) copia l'issue e opzionalmente i figli diretti; `ConvertToProject` mantiene le issue e rimuove i legami parent dei record coinvolti. | Applicare Efforts a ciascuna copia già prevista, senza ampliare la profondità della duplicazione; preservare dati sulle issue convertite. US-12–13. |
| Eliminazione/identità | [Schema iniziale](../../BE/migrations/000001_init_schema.up.sql): parent con `ON DELETE SET NULL`, storico con cascade sull'issue e FK utente obbligatoria. | Cancellare solo effort delle issue eliminate, conservando quelli dei figli rimasti; progettare attribuzioni storiche che non spariscano con la membership/account. Non introdurre una nuova funzione di cancellazione account. US-14, US-18, US-21. |
| Pubblico | [PublicIssueResponse](../../BE/internal/dto/shared_link.go) è un DTO dedicato, costruito da [SharedLinkService](../../BE/internal/service/shared_link.go). | Mantenere una proiezione esplicita senza Efforts; controllare risposte e storico, non solo il rendering. US-20. |
| Realtime | [Hub](../../BE/internal/realtime/hub.go) ricontrolla l'accesso in consegna; nei workspace con privacy sostituisce i payload con `app.refresh` generico. | Riutilizzare questo comportamento e ricaricare via API autorizzate; non inviare worklog o riferimenti privati indiscriminatamente al workspace. Emettere solo dopo commit. US-23. |
| Trasferimento | [TableSpecs](../../BE/internal/repository/workspace_transfer.go) definisce ordine e campi utente; [service](../../BE/internal/service/workspace_transfer.go) rimappa ID e importa in transazione; manifest v1 in [DTO](../../BE/internal/dto/workspace_transfer.go). Trasferimenti vincolati dalla privacy. | Estendere archivio e rimappatura anche per snapshot storici; decidere compatibilità/versione e mantenere i blocchi esistenti sui workspace con team privati. US-24–25. |
| Frontend | [UI/package.json](../../UI/package.json): Svelte 5, SvelteKit, TypeScript, Tailwind, Bits UI e Paraglide; client in [api/client.ts](../../UI/src/lib/api/client.ts). | Componenti Efforts riusabili, stesso client e modello errori; traduzioni nei cataloghi esistenti, senza modificare file generati. US-03–10, US-15–16. |
| Superfici issue | [FullPageIssueView](../../UI/src/lib/features/issues/FullPageIssueView.svelte) e [IssueDetail](../../UI/src/lib/features/issues/IssueDetail.svelte) hanno presentazioni e caricamenti distinti. | Integrare riepilogo, registro e attività in entrambe le superfici, verificando i rispettivi contenitori. Non presumere che modificare una vista aggiorni l'altra. US-15–16, US-21. |
| Aggiornamenti UI | [Layout workspace](../../UI/src/routes/[workspaceSlug]/+layout.svelte) gestisce eventi WebSocket e `app:refresh`; stato issue in [issues.state](../../UI/src/lib/features/issues/issues.state.svelte.ts). | Integrare refresh di riepilogo, registro, attività e antenati anche su reconnect e cambio accesso. US-23. |
| Verifica | [Workflow test](../../.github/workflows/test.yml), test Go anche con PostgreSQL, test unitari UI ed E2E Playwright in `UI/tests`. | Estendere gli strumenti già presenti. Le garanzie transazionali richiedono test con PostgreSQL reale, non soltanto mock o `-race`. T-03. |

`TECHNICAL.md` riguarda specificamente Dev Machines: non è la specifica generale
del tracker. Non si trasferiscono a Efforts il manager Docker, il gateway o la
coda operativa di quel sottosistema. Anche `WEB/` è distinto dall'applicazione
tracker, che risiede in `UI/`.

## 4. Scelte tecniche proposte e decisioni di refinement

Le seguenti sono proposte di implementazione compatibili con il codice, non
nuovi requisiti funzionali. T-01/T-02 devono concretizzarle prima delle scritture.

| Decisione | Direzione proposta | Evidenza richiesta per chiuderla |
| --- | --- | --- |
| Persistenza valori propri | Minuti interi, stima e residuo nullable, worklog come entità collegata a issue/workspace. Valutare colonne su `issues` oppure tabella collegata, limitando impatto sui DTO generici. | Migrazione e contratto di lettura che preservino assente/zero, cancellazione e import. |
| Totali | Consuntivo derivato dai worklog; aggregati calcolati in SQL con visibilità. Evitare cache persistenti iniziali che moltiplichino gli obblighi di coerenza. | Query e piano verificati su gerarchie e registri rappresentativi. |
| Concorrenza | Confine transazionale nel repository, revisione attesa per le modifiche e serializzazione del budget giornaliero per workspace/persona/data. | Due connessioni reali, inclusi giorni inizialmente senza registrazioni e modifiche che cambiano data. |
| Idempotenza | Identificatore del tentativo conservato dal client e deduplicazione persistente, distinta dall'uguaglianza dei campi del worklog. | Replay dopo perdita della risposta o riavvio; riuso della chiave con payload diverso rifiutato. |
| Autorizzazione | Ruoli correnti e visibilità riesaminati nella scrittura; scope Efforts separati. Amministrazione configurazione solo sessione owner/admin. | Matrice su endpoint nuovi e alternativi: storia, dettaglio, bulk, duplicazione, export. |
| API e formati | Restare sotto `/api/workspaces/:slug`, usare DTO ed errori contestuali esistenti; definire revisione, idempotenza e risposta paginata da 50. | Esempi di contratto, codici errore e manifest route aggiornato. Nomi esatti degli endpoint/scope da fissare in T-01. |
| Storico/identità | Eventi nella stessa attività issue, con valori prima/dopo, causa e identità distinte. Gestire operatore di sistema per GitHub, che oggi usa `uuid.Nil` nel tentativo di storico. | Modello che non richieda una FK utente inesistente e conservi le attribuzioni in export/import e dopo rimozione dell'utente. |
| Compatibilità archivio | Rendere esplicito come leggere archivi v1 senza Efforts e scrivere archivi estesi; rimappare anche ID dentro gli snapshot. | Round-trip e fixture legacy, evitando dipendenza dalla rimappatura automatica di stringhe opache. |

La verifica interna del limite giornaliero è un caso particolare: deve sommare
anche issue nascoste, restando strettamente limitata a persona, data e workspace.
Non si deve aggirare la privacy delle normali API di lettura per ottenere questo
risultato, né restituire all'operatore il dettaglio del lavoro non accessibile.

Ulteriori dettagli da fissare nel refinement tecnico: limiti numerici contro
overflow di stime/somme, durata di conservazione delle chiavi di idempotenza,
ordine stabile a parità di data e inserimento, politica dei retry transazionali e
volumi rappresentativi per query e riattivazione. Non sono state inventate soglie
di performance, giorni/uomo o story point senza misurazioni e confronto col team.

## 5. Rischi che devono guidare l'ordine del backlog

| Rischio concreto | Mitigazione prevista |
| --- | --- |
| Implementare residuo come `stima - consuntivo` in ogni lettura | US-04–08 con casi assente/zero e modifica/eliminazione senza ricalcolo. |
| Superare 24 ore con due inserimenti su issue diverse | T-02 e US-10 con lock sul budget della persona/data, non soltanto sulla issue. |
| Salvare dati senza storico o storico senza dati | Scrittura transazionale comune T-02/US-21; errore dello storico provoca rollback. |
| Riattivazione parziale o worklog durante disattivazione | US-02 e US-22 con serializzazione rispetto alle scritture concorrenti. |
| Residuo positivo dopo chiusura automatica | US-11 su update, bulk, triage, automazioni parent/figli e GitHub. |
| Perdere privacy tramite aggregati, errori o vecchi scope | US-17–20, query filtrate e controlli sui percorsi alternativi. |
| Dimenticare pannello, antenati o refresh privacy | US-15–17/23 e scenari E2E su entrambe le superfici. |
| Import che modifica residuo, perde autori o duplica eventi automatici | US-25 valida e ripristina dati/storico senza rieseguire gli effetti di un nuovo worklog. |

## 6. Consegna e verifica dell'analisi

Il backlog allegato definisce per ogni storia valore utente, requisiti e AC
coperti, dipendenze, verifiche e attività tecniche. Include una matrice completa
EFF → storie e AC → storie e una sequenza di realizzazione.

Questa attività ha esaminato documenti e codice staticamente e verificato la
coerenza dei riferimenti del backlog. Non sono stati eseguiti test applicativi
né dichiarato che l'applicazione soddisfi già gli AC Efforts. Le verifiche di
implementazione sono attività future esplicitate nel backlog.

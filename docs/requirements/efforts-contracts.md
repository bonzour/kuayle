# Efforts — contratti API e archivio V1

Decisioni del 2026-10-17, branch `feature/efforts`, baseline precedente `918cc47`.
Fonte funzionale: [requisiti confermati](efforts.md), EFF-001–041 e AC-001–051.
Attività: [T-01](../backlog/efforts.md).

Aggiornamento V1 del 2026-10-20: EP-01–08 implementate. Configurazione riservata
alle sessioni owner/admin; pianificazione e worklog disponibili anche ai PAT con
scope espliciti. Aggregati, attività, realtime e trasferimento v2 sono operativi.
Le garanzie transazionali sono in [efforts-transactions.md](efforts-transactions.md),
le prove in [efforts-acceptance.md](efforts-acceptance.md) e l'uso in
[efforts-usage.md](efforts-usage.md). Nessun arricchimento dei DTO pubblici o dei
template; i valori propri restano negli endpoint Efforts dedicati.

## 1. Persistenza

| Tabella | Identità e significato |
| --- | --- |
| `workspace_effort_settings` | PK `workspace_id`, `enabled`, `timezone`, `revision`, `updated_at`. Assenza della riga = `true`, `Europe/Rome`, revisione 0, anche sui workspace esistenti. |
| `issue_efforts` | PK `issue_id`, FK composta `(workspace_id, issue_id)` verso `issues`; `estimate_minutes`, `remaining_minutes` nullable e `revision`. Assenza della riga = entrambi assenti e revisione 0. |
| `effort_worklogs` | UUID, FK composta verso issue, `owner_user_id`, `created_by_user_id`, `work_date`, `duration_minutes`, `description`, `created_at`, `updated_at`. |
| `effort_events` | UUID, FK composta verso issue, `event_type`, identità operatore, causa, versione snapshot, snapshot JSON prima/dopo, revisione risultante e timestamp. Nessuna FK verso un worklog cancellabile. |
| `effort_operations` | PK `(workspace_id, actor_user_id, request_id)`, SHA-256 di 32 byte, `issue_id` logico nullable per i comandi workspace, `resource_id`, revisione e timestamp del commit. FK soltanto al workspace. |

Stima e residuo usano `INTEGER`, da 0 a 2.147.483.647 minuti, oppure SQL NULL.
Non esiste un consuntivo persistito: si sommano i worklog presenti. Le somme
usano `BIGINT`; l'API rifiuta valori oltre 9.007.199.254.740.991 anziché perdere
precisione JavaScript. Anche le revisioni rispettano questo limite.

`work_date` è `DATE`, serializzato `YYYY-MM-DD`, da `0001-01-01` a `9999-12-31`.
Nessuna conversione UTC o dipendenza dal fuso del browser. La durata è un intero
fra 1 e 1.440 minuti. Descrizione nullable, testo semplice, massimo 2.000 caratteri
Unicode, senza troncamento. I form convertono `1h 30m` in 90 e `1d` in 480; le API
ricevono minuti interi, non stringhe di durata o numeri frazionari.

Il fuso deve esistere in `pg_timezone_names`, essere un nome IANA con `/` oppure
`UTC`, escludendo gli alberi `posix/` e `right/`. Non accetta offset numerici o
abbreviazioni come `CET`. Il controllo applicativo sui salvataggi usa lo stesso
nome per calcolare oggi; cambiare fuso non modifica le date persistite.

ID, issue, workspace, titolare, operatore iniziale e inserimento di un worklog
sono immutabili. L'operatore di una correzione compare nel nuovo evento.
Gli UUID umani sono riferimenti storici intenzionalmente privi di FK a `users`
o `workspace_members`: rimuovere l'account o la membership non cancella lavoro
e non lo attribuisce ad altri. Le scritture validano le identità
prima di inserirle; conoscere un riferimento storico non concede accesso.

La cancellazione dell'issue elimina valori, worklog ed eventi; i figli conservati
mantengono i propri dati. La ricevuta del tentativo sopravvive come marcatore
minimo, senza descrizioni, valori o snapshot. Eliminare il workspace elimina
anche ricevute e configurazione. Le cinque tabelle installano esplicitamente
`privacy_statement_shared`, riusando la barriera della migrazione 46.

La down della 47 è ammessa soltanto quando tutte le tabelle Efforts sono vuote.
Non elimina vincoli o funzioni precedenti e non cancella silenziosamente dati.

## 2. API di lettura implementate

Prefisso: `/api/workspaces/:slug`. Sessione autenticata o PAT con
`efforts:read`, membership corrente e visibilità dell'issue sono obbligatori.
Il repository richiede un chiamante autenticato anche fuori dal middleware HTTP.
Disabilitare Efforts non blocca queste letture.

| Metodo e route | Risposta |
| --- | --- |
| `GET /efforts/settings` | `{"enabled":true,"timezone":"Europe/Rome","revision":0,"today":"2026-10-17"}` |
| `GET /issues/:identifier/efforts` | Valori propri e aggregati autorizzati. |
| `GET /issues/:identifier/efforts/worklogs?page=1&per_page=50` | Involucro `ListResponse<EffortWorklog>` con `revision` dell’issue. |

`today` è una proiezione, non un'impostazione persistita: PostgreSQL la calcola
nello stesso statement usando il fuso del workspace e l'istante corrente.
Non dipende dal fuso del server/browser e non converte le date dei worklog.

Issue senza dati:

```json
{
  "revision": 0,
  "has_data": false,
  "has_visible_descendants": false,
  "own": {
    "estimate_minutes": null,
    "spent_minutes": 0,
    "remaining_minutes": null
  },
  "total": {
    "estimate_minutes": null,
    "spent_minutes": 0,
    "remaining_minutes": 0,
    "estimate_partial": true,
    "remaining_partial": true
  }
}
```

Uno zero esplicito resta `0`; i campi nullable non sono omessi. `total` comprende
l'issue e tutti i discendenti visibili una volta sola. Il grafo è percorso entro
il workspace, poi la visibilità filtra i contributi prima delle somme. Questo
non perde un eventuale discendente visibile dietro un nodo non accessibile.
Le stime note sono sommate; se tutte assenti la stima totale è `null`.
`estimate_partial` e `remaining_partial` indicano dati mancanti soltanto tra le
issue visibili. Nessun numero o dettaglio delle issue escluse è restituito.
`has_data` indica almeno un valore esplicito (anche zero) o lavoro registrato
nei contributi visibili; `has_visible_descendants` indica almeno un discendente
visibile. Entrambi sono letti nello stesso statement delle somme. Il riepilogo
vuoto è nascosto quando Efforts è disabilitato; i dati esistenti restano visibili.

Esempio di pagina:

```json
{
  "revision": 7,
  "data": [{
    "id": "10000000-0000-0000-0000-000000000001",
    "issue_id": "20000000-0000-0000-0000-000000000001",
    "owner": {
      "id": "30000000-0000-0000-0000-000000000001",
      "available": false, "name": null, "display_name": null
    },
    "operator": {
      "id": "30000000-0000-0000-0000-000000000002",
      "available": true, "name": "Alice", "display_name": "Alice"
    },
    "work_date": "2026-10-16",
    "duration_minutes": 90,
    "description": null,
    "created_at": "2026-10-17T08:00:00Z",
    "updated_at": "2026-10-17T08:00:00Z"
  }],
  "total_count": 1, "page": 1, "per_page": 50, "has_more": false
}
```

`available=false` permette alla UI di mostrare «Utente non più disponibile»;
un utente rimosso dal workspace ma con account esistente resta identificabile.
Non si espongono email nelle attribuzioni. Pagine da 50 fisse; `page` predefinita 1,
positiva e compatibile con il limite numerico; `per_page`, se fornito, deve essere
50. Ordine `work_date DESC, created_at DESC, id DESC`. Conteggio e righe sono letti
nello stesso statement, insieme alla revisione Efforts dell’issue (zero senza
stato persistito). Una pagina oltre l'ultima restituisce `data: []` con il
conteggio effettivo. Una risorsa non accessibile restituisce lo stesso 404 di una
inesistente. L'ordinamento è stabile sullo stato letto; pagine successive possono
cambiare durante modifiche concorrenti: il client le ricarica al refresh/reconnect.

## 3. Contratti di scrittura e comandi interni

DTO Go/TypeScript e comandi transazionali sono condivisi tra sessioni e PAT.
Per pianificazione e worklog un PAT richiede `efforts:read` + `efforts:write`,
oltre al ruolo/accesso corrente del proprietario. Configurazione e selettore
dei titolari restano riservati alle sessioni owner/admin:

| Comando | Campi oltre all'involucro comune |
| --- | --- |
| `SetEffortEstimateRequest` | `estimate_minutes`: intero o null esplicito per rimuovere; ometterlo non equivale a rimuovere. |
| `SetEffortRemainingRequest` | `remaining_minutes`: intero non negativo obbligatorio. |
| `CreateEffortWorklogRequest` | `owner_user_id`, `work_date`, `duration_minutes`, `description` opzionale/null; `end_date` e `skip_weekends` opzionali per un periodo. |
| `UpdateEffortWorklogRequest` | Data, durata e descrizione desiderate; nessun titolare modificabile. |
| `DeleteEffortWorklogRequest` | Solo involucro comune. |
| `SetEffortSettingsRequest` | Configurazione completa: `enabled`, `timezone`; solo sessione owner/admin. |

### Configurazione EP-01

`PUT /api/workspaces/:slug/efforts/settings` riceve la configurazione completa:

```json
{
  "request_id": "40000000-0000-0000-0000-000000000001",
  "expected_revision": 0,
  "enabled": false,
  "timezone": "Europe/Rome"
}
```

Risposta 200: `EffortMutationReceipt` con `resource_id` uguale all'UUID workspace.
Sessioni owner/admin possono salvare anche da disabilitato; membri, guest e PAT
sono esclusi. Ruolo e membership sono ricontrollati nella transazione. JSON
malformato, campi ignoti e valori invalidi sono rifiutati; i dettagli di validazione
indicano il campo, incluso `timezone`. Si applicano i codici 400/409 sotto descritti.

Disabilitare conserva i dati; riattivare azzera soltanto i residui delle issue
attualmente chiuse, con eventi `remaining_set`, causa `reactivation` e operatore.
Storico, residui, impostazioni e ricevuta sono atomici. I replay non rieseguono
la riattivazione e devono ancora superare i permessi correnti. Dopo un nuovo commit
il Hub pubblica `app.refresh` per `workspace` e `issues`, senza dati delle issue;
rollback e replay non pubblicano. Nei workspace privati resta il refresh generico.

Le Impostazioni generali mostrano la configurazione a tutti i membri autorizzati,
con modifica limitata a owner/admin. Il client conserva bozza e payload del retry;
un conflitto richiede la sostituzione esplicita della bozza con i valori correnti.
Il dialogo worklog rilegge `today` dal server a ogni apertura.

### Pianificazione EP-02

Le due route ammettono sessioni utente e PAT con entrambi gli scope Efforts.
Prefisso comune:
`/api/workspaces/:slug/issues/:identifier/efforts`.

| Metodo e route | Campi oltre a `request_id` e `expected_revision` |
| --- | --- |
| `PUT /estimate` | `estimate_minutes`: intero da 0 a 2.147.483.647, oppure `null` esplicito per rimuovere la stima. |
| `PUT /remaining` | `remaining_minutes`: intero obbligatorio da 0 a 2.147.483.647; `null` non è ammesso. |

L'identificativo è risolto nel workspace prima del comando. Accesso, ruolo,
assegnazioni primarie/multiple, attivazione e revisione sono ricontrollati nella
transazione. Owner/admin con accesso e membri assegnatari possono scrivere;
guest e membri non assegnatari sono esclusi. Una destinazione non accessibile o
appartenente a un altro workspace restituisce 404. JSON malformato, campi ignoti,
campi obbligatori assenti, frazioni e valori fuori limite restituiscono 400.
La risposta 200 è `EffortMutationReceipt`, con `resource_id` uguale all'UUID issue.

Su issue aperta, una stima non nulla inizializza soltanto il residuo assente a
`max(stima - consuntivo, 0)`. Revisione/rimozione della stima preservano un residuo
già impostato, incluso zero; stima e residuo non cambiano i worklog. Sulle issue
concluse/annullate il residuo resta zero e ogni nuovo comando manuale sul residuo
è rifiutato, anche se propone zero. Riaprire non ripristina il valore precedente.
Un replay già committato restituisce la ricevuta dopo i controlli correnti di
accesso e attribuzione, senza rieseguire la modifica.

Pianificazione, eventi prima/dopo e ricevuta sono atomici. Si riusano i codici
400/409 descritti sotto e il refresh generico post-commit per `issues`; rollback
e replay non pubblicano. Nessun arricchimento dei DTO generici o pubblici.

Pagina completa e pannello nei progetti condividono la sezione richiudibile
«Effort», con tre righe stima/consuntivo/residuo nello stesso formato di DETAILS,
senza icone nei valori. In pagina è accodata sotto PROJECTS. Stima e residuo si
modificano inline e si salvano su Invio o all'uscita; Esc
annulla la modifica. Non sono presenti pulsanti Salva. Il consuntivo proprio
resta in sola lettura. Le due scritture usano la revisione comune dell'issue.
Stima assente è «Non stimato»; residuo assente è
visualizzato zero con «Non impostato», distinto dallo zero esplicito. Rimozione
della stima è un comando separato, non un campo vuoto.
Dal 2026-10-20 l’unità principale UI è il giorno da 8 ore: numeri senza unità
indicano giorni; sono supportati `1d`, `0.5d`/`0,5d`, `1d 2h 30m`, `90m`, `2h`,
`1h 30m` e `1h30m`. Le frazioni sono ammesse soltanto per giorni e soltanto se
corrispondono a minuti interi. Output con giorni, ore e minuti conserva la
precisione. Segni, settimane, unità duplicate e overflow sono rifiutati.
Il contratto API e la persistenza continuano a usare minuti interi.
Errori conservano la bozza; un 404 è segnalato come indisponibilità, distinto dal
rifiuto dei permessi (403). Il caricamento del ruolo non implica un rifiuto.
Retry incerti conservano payload e chiave. Conflitti
richiedono il riallineamento esplicito. Letture tardive non possono sovrascrivere
una lettura più recente o i dati di un'altra issue/workspace/persona.
Lo storico è visibile in Activity. I totali delle issue visibili sono separati
dai valori propri e sempre in sola lettura. Non sono necessarie nuove migrazioni.

### Registrazioni EP-03

Prefisso: `/api/workspaces/:slug/issues/:identifier/efforts`. Le mutazioni
ammettono sessioni e PAT con read+write Efforts. Il solo selettore dei titolari
è un supporto UI riservato alle sessioni owner/admin.

| Metodo e route | Richiesta / risposta |
| --- | --- |
| `POST /worklogs` | `CreateEffortWorklogRequest` → 200 `EffortMutationReceipt`. |
| `PUT /worklogs/:worklogId` | `UpdateEffortWorklogRequest` → 200 `EffortMutationReceipt`. |
| `DELETE /worklogs/:worklogId` | Involucro comune nel corpo → 200 `EffortMutationReceipt`. |
| `GET /worklog-owners` | Solo sessioni owner/admin con accesso: array `EffortUser[]`, senza email. |

La selezione ammette soltanto owner/admin/member attuali con accesso all’issue,
applicando i predicati privacy esistenti. Il comando ricontrolla l’ammissibilità
dopo le attese. Titolare e operatore iniziale sono immutabili; gli admin possono
correggere/eliminare lavoro storico di persone rimosse o diventate guest.
I membri registrano lavoro proprio anche senza assegnazione o stima e su issue
chiuse. Modifica ed eliminazione richiedono titolarità oppure ruolo owner/admin.

Durata 1–1.440 minuti, data di calendario non futura nel fuso workspace,
descrizione nullable multilinea fino a 2.000 caratteri Unicode. JSON malformato,
campi ignoti/immutabili e valori invalidi restituiscono 400; il limite giornaliero
usa 400 `EFFORT_DAILY_LIMIT` senza dettagli o conteggi. Restano i conflitti 409
e i controlli transazionali/idempotenti comuni. I replay dopo eliminazione sono
riconosciuti senza richiedere preventivamente l’esistenza del worklog.

Pagina e pannello condividono WORK LOG dopo l’attività. La sottosezione
«Registrazioni» conserva il registro con pagine da 50,
data, titolare, durata, descrizione e «inserito da» nelle deleghe. L’intestazione
è richiudibile e le righe riprendono lo stile compatto della timeline ACTIVITY.
Si mostrano inizialmente tre registrazioni, con comando per espandere le
precedenti nella pagina corrente e tornare alla vista ridotta. I dialoghi di
inserimento/modifica e conferma eliminazione conservano le bozze sugli errori;
solo owner/admin scelgono altri titolari in creazione. All’apertura si rilegge
`today` dal server. Le date non attraversano conversioni UTC/browser; i testi
sono resi come testo semplice e non vengono troncati.

Ogni dialogo conserva la revisione della pagina da cui nasce: refresh remoti
non riallineano una bozza. Il riallineamento esplicito sostituisce la bozza con
i dati correnti e usa una nuova chiave al salvataggio. I retry incerti conservano
chiave e payload, anche se nel frattempo Efforts viene disabilitato: il server
riconosce un replay oppure rifiuta un tentativo non committato. Dopo successo
si ricaricano la prima pagina e il riepilogo tramite il refresh generico esistente.
Risposte tardive e callback di componenti distrutti non popolano altre viste.

Il comando per mostrare le registrazioni precedenti è sopra le righe, come in
Activity. Il completamento V1 include aggregati e storico nelle due superfici,
con refresh remoto e riconnessione; non introduce timesheet o notifiche dedicate.

### Inserimento per periodo e timeline Efforts (2026-10-20)

`POST /worklogs` conserva il contratto singolo ed accetta anche:

```json
{
  "request_id": "40000000-0000-0000-0000-000000000001",
  "expected_revision": 7,
  "owner_user_id": "30000000-0000-0000-0000-000000000001",
  "work_date": "2026-03-27",
  "end_date": "2026-03-30",
  "skip_weekends": true,
  "duration_minutes": 480,
  "description": "Effort giornaliero"
}
```

Il periodo è inclusivo; la durata si applica a ciascuna data scelta. Il server
espande le date di calendario e, se richiesto, esclude soltanto sabato/domenica.
`end_date` deve essere valido, non precedente a `work_date` e non futuro nel fuso
workspace; un periodo privo di date selezionate è rifiutato con 400 e dettaglio
`end_date`. Senza `end_date` rimane la semantica singola, anche nei weekend.

Autorizzazioni e delega sono quelle dei worklog singoli. Prima degli inserimenti
si acquisiscono tutti i lock budget in ordine stabile; ogni data viene validata
entro la stessa transazione. Un errore su qualsiasi data annulla l’intero periodo,
compresi residuo, eventi e ricevuta. La revisione issue avanza una sola volta;
ogni worklog ha il suo evento, e il residuo impostato diminuisce della somma.
La ricevuta del periodo usa l’UUID issue come `resource_id`. Hash e ricevuta
includono estremi e scelta weekend; i replay non ricreano le righe. I tentativi
singoli precedenti mantengono il loro hash, senza alterare i retry già persistiti.
Nessun nuovo modello persistito o migrazione.

WORK LOG mostra gli eventi Efforts con la forma precedente di Activity; gli
eventi generici restano in Activity. Le variazioni `remaining_set` appaiono solo
per operatore utente, causa manuale e comando esplicito sul residuo. Gli eventi
storici automatici marcati manuali sono riconosciuti dal comando stima/worklog
con la stessa issue e revisione. La proiezione UI non elimina gli eventi dalla
risposta API né dall’archivio. Il registro precedente rimane in «Registrazioni»,
richiudibile, con le azioni esistenti. Testi nelle quattro lingue del progetto.

### Comandi sulle issue e garanzie comuni

Esempio di creazione:

```json
{
  "request_id": "40000000-0000-0000-0000-000000000001",
  "expected_revision": 7,
  "owner_user_id": "30000000-0000-0000-0000-000000000001",
  "work_date": "2026-10-16",
  "duration_minutes": 90,
  "description": "Verifica del contratto"
}
```

Una revisione Efforts per issue copre pianificazione e qualsiasi mutazione dei
worklog, anche della sola descrizione; ogni comando completato la incrementa una
volta. La configurazione ha una revisione separata. L'assenza iniziale equivale a
revisione 0; le righe di stato sono create nella transazione del primo comando.
Campi immutabili/ignoti delle scritture sono rifiutati dagli handler.

Idempotenza: chiave `(workspace, operatore, request_id)` senza TTL. SHA-256 del
JSON canonico della richiesta normalizzata, contenente nome operazione, destinazione,
revisione attesa e campi, esclusi credenziali e intestazioni di trasporto. Ordinare
le chiavi JSON; risolvere gli identificatori di destinazione a UUID prima del hash;
normalizzare descrizione omessa a null. La chiave non deriva dal contenuto: due
invii intenzionali identici usano UUID differenti. Non condividere chiavi fra comandi.

T-02 ricontrolla autorizzazioni e attivazione dopo le attese, riconosce
un tentativo già committato prima del controllo di revisione e salva ricevuta,
dati, variazioni del residuo ed eventi nella stessa transazione. La ricevuta ha
`request_id`, `resource_id`, `revision`; il client rilegge le proiezioni correnti.
Un replay restituisce la ricevuta precedente soltanto dopo i controlli correnti
sull'issue logica; se l'issue è stata eliminata restituisce 404 senza riesecuzione.
Il replay di una cancellazione del worklog resta riconoscibile dalla ricevuta e
dall'`issue_id` logico anche quando il worklog non esiste più.

Gli errori riusano `ErrorResponse` e `details: [{field, message}]`:

| HTTP / codice | Semantica |
| --- | --- |
| 400 `VALIDATION_ERROR` | Campi invalidi, data futura o fuso non valido; dettagli contestuali. |
| 400 `EFFORT_DAILY_LIMIT` | Superamento di 1.440 minuti per titolare/data/workspace; mai dettagli, conteggi o riferimenti al lavoro nascosto. |
| 409 `EFFORT_REVISION_CONFLICT` | Revisione obsoleta; nessuna sovrascrittura. |
| 409 `EFFORT_REQUEST_CONFLICT` | Stessa chiave con operazione/contenuto diverso. |
| 409 `EFFORT_DISABLED` | Tentativo nuovo di mutazione mentre Efforts è disabilitato. |
| 401 / 403 / 404 | Convenzioni comuni di autenticazione, autorizzazione e risorsa non accessibile. |
| 500 `INTERNAL_ERROR` | Errore di persistenza/overflow; nessun dettaglio SQL nel payload. |

Limite giornaliero, controlli sulle date future, assegnazione, delega e transizioni
di stato non sono vincoli locali alla singola riga. Sono implementati nei percorsi
transazionali interni T-02; la sola persistenza T-01 non li garantisce. Le API/UI usano questi percorsi condivisi.

## 4. Eventi e archivio v2

Gli eventi hanno `snapshot_version=1`, `before`/`after` nullable e una proiezione
`planning` oppure `worklog`. Gli snapshot worklog conservano UUID, titolare,
operatore iniziale, data, durata, descrizione e timestamp. Non sono stringhe JSON
in `old_value/new_value`. I tipi evento sono `estimate_set`, `remaining_set`,
`worklog_created`, `worklog_updated`, `worklog_deleted`; le cause sono `manual`,
`worklog`, `closure`, `reactivation`. Il residuo automatico produce il proprio
evento nella stessa transazione del comando causale.

L'operatore è `{ "kind": "user", "user_id": "..." }` oppure
`{ "kind": "system", "system_source": "github" }`. Le automazioni dichiarano
l'origine reale, senza inventare account o usare `uuid.Nil`. L'operatore della
riattivazione è l'admin che la richiede. Questi eventi sono proiettati nella
risposta storico dell'issue mantenendo compatibili gli eventi generici esistenti.

`GET /issues/:identifier/history` mantiene l'involucro degli eventi generici e
aggiunge gli eventi con `field: "efforts"` e `effort: EffortEvent`. Il campo
`effort` comprende actor, cause, snapshot_version, before/after, revision e
`users: EffortUser[]` per operatore, titolare e inseritore storici, senza email.
Per gli eventi di sistema il `user_id` esterno è omesso. Gli eventi sono ordinati
per istante decrescente e UUID a parità di istante; quelli Efforts non sono
raggruppati eliminando passaggi intermedi. Senza `efforts:read` sono omessi;
nessuna proiezione pubblica include Efforts.

Il formato emesso è `kuayle.workspace`, versione **2**:

- Configurazione e dati nelle quattro nuove tabelle di dominio, ordinate in import
  come settings, issue_efforts, worklog, eventi dopo le issue. Assenza di settings
  resta equivalente ai default. Le ricevute `effort_operations` non si trasferiscono.
- Nuovo elenco `data.unavailable_users: [{ "id": "UUID-storico" }]` per gli account
  assenti ma referenziati. I riferimenti ad account esistenti restano in `data.users`.
  L'import assegna nuovi riferimenti storici agli unavailable, senza creare account
  o membership e senza sostituirli con l'importatore.
- Rimappatura tipizzata di workspace, issue, evento, worklog, titolari e operatori.
  Allocare prima anche gli ID dei worklog presenti solo negli snapshot di eliminazione.
  Rimappare `worklog.id`, `owner_user_id`, `created_by_user_id` negli snapshot;
  non modificare UUID citati incidentalmente nel testo della descrizione.
- Accettare archivio v1 senza Efforts con default attivo/Europe-Rome; tutti i nuovi
  export emettono v2. I client precedenti che accettano soltanto v1 devono essere
  aggiornati per importare i nuovi archivi.
- Validare durate, riferimenti e somme giornaliere prima dell'import atomico.
  Ripristinare date, stato disabilitato e snapshot senza rieseguire effetti worklog,
  azzeramenti o nuovi eventi. L'export richiede una vista coerente dei dati, oltre
  alla barriera privacy esistente, e mantiene i blocchi per workspace privati.

L'export acquisisce la barriera privacy del workspace prima dello snapshot
`REPEATABLE READ READ ONLY`, usando una sola connessione per tutte le tabelle.
Le scritture concorrenti non producono combinazioni incoerenti di worklog e storico.
Il lock di sessione è rilasciato prima del ritorno della connessione al pool.
L'import valida anche identità immutabili, snapshot, revisioni, riferimenti e
limiti giornalieri dopo la rimappatura utenti. Gli UUID storici non disponibili
non possono essere usati per creare membership. La descrizione viene conservata
letteralmente, anche quando contiene UUID o URL di allegati.

## 5. Scope e canali indiretti

`efforts:read` è nel limite di ruolo di owner/admin/member/guest;
`efforts:write` soltanto di owner/admin/member. Il limite di ruolo non sostituisce
titolarità, assegnazione e visibilità. I PAT esistenti non cambiano. La nuova riga
Efforts nella UI espande Read a read, Read & write a read+write. I preset con tutte
le letture includono read; solo Full access include anche write. Triage resta invariato.

Regole applicate alle route e ai percorsi di dominio:

| Percorso | Regola PAT |
| --- | --- |
| Mutazioni Efforts | `efforts:read` + `efforts:write` e policy corrente. |
| Configurazione in scrittura | Solo sessione interattiva owner/admin, nessun PAT. |
| Storico issue | Gli eventi Efforts sono omessi senza `efforts:read`; gli altri eventi restano leggibili con `issues:read`. |
| Export workspace v2 | `workspace:transfer` + `efforts:read`; nessun archivio Efforts parziale. |
| Import workspace | Mantiene il vincolo attuale di sessione interattiva. |
| Duplicazione | Richiede `issue:create` + read/write Efforts; negazione uniforme senza dipendere dalla presenza di una stima. |
| Update/bulk generici | I campi senza effetti Efforts mantengono il contratto attuale. Le operazioni che possono mutare Efforts richiedono read/write, prima di qualsiasi effetto; niente mutazioni parziali o salti degli invarianti. |
| Delete issue/bulk | Richiede anche read/write; non dipende dalla presenza dei dati. |
| Condivisione pubblica | Nessun dato o evento Efforts, nessuna credenziale Efforts derivata dal link. |

Negazioni per duplicazione/eliminazione sono uniformi, indipendenti dalla presenza
di dati Efforts o dall'attivazione. Cambi di stato, bulk e rifiuto triage richiedono
read/write prima degli effetti. Gli scope non aumentano i permessi del ruolo.
Il repository filtra anche le letture prive del contesto autenticato Efforts.

## 6. Verifica T-01

Verificato il 2026-10-17:

- Suite backend completa `go test ./... -race -count=1`, dopo le migrazioni su
  un database temporaneo distinto dal database di sviluppo: superata.
- Test dedicati `TestEffortMigration` e `TestEffortsHTTP`: installazione nuova,
  upgrade con dati dalla 46, down vuota/protetta, vincoli e conservazione delle
  attribuzioni, scope, revoche, privacy, gerarchie e paginazione oltre 50 righe.
- `NODE_OPTIONS=--experimental-strip-types npm run test:unit`: 31 test superati.
  Il flag serve con Node 22.15 locale per importare i moduli TypeScript dei test.
- `npm run check`: zero errori; warning nel layout workspace non modificato
  relativo a `shortcutDefs` catturato al valore iniziale.
- Nessuna migrazione applicata al database di sviluppo durante questa verifica.

Questa sezione registra la verifica T-01. Le garanzie delle scritture T-02 sono
verificate separatamente in [efforts-transactions.md](efforts-transactions.md).
Il completamento V1, inclusi round-trip e snapshot concorrente dell’export, è
verificato in [efforts-acceptance.md](efforts-acceptance.md).

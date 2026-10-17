# Efforts — contratti T-01

Decisioni del 2026-10-17, branch `feature/efforts`, baseline precedente `918cc47`.
Fonte funzionale: [requisiti confermati](efforts.md), EFF-001–041 e AC-001–051.
Attività: [T-01](../backlog/efforts.md).

Questo documento distingue i contratti implementati dalle scritture future.
T-01 introduce migrazione 47, modelli e API di lettura; non completa T-02,
le policy di scrittura US-18/19, l'attività US-21 o il trasferimento US-24/25.
Non espone endpoint Efforts di scrittura, né aggiunge Efforts ai DTO generici
delle issue, ai DTO pubblici o ai template.

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
Unicode, senza troncamento. I futuri form convertiranno `1h 30m` in 90; le API
riceveranno minuti interi, non stringhe di durata o numeri frazionari.

Il fuso deve esistere in `pg_timezone_names`, essere un nome IANA con `/` oppure
`UTC`, escludendo gli alberi `posix/` e `right/`. Non accetta offset numerici o
abbreviazioni come `CET`. Il controllo applicativo sui salvataggi userà lo stesso
nome per calcolare oggi; cambiare fuso non modifica le date persistite.

ID, issue, workspace, titolare, operatore iniziale e inserimento di un worklog
sono immutabili. L'operatore di una correzione compare nel nuovo evento.
Gli UUID umani sono riferimenti storici intenzionalmente privi di FK a `users`
o `workspace_members`: rimuovere l'account o la membership non cancella lavoro
e non lo attribuisce ad altri. Le scritture future dovranno validare le identità
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
| `GET /efforts/settings` | `{"enabled":true,"timezone":"Europe/Rome","revision":0}` |
| `GET /issues/:identifier/efforts` | Valori propri e aggregati autorizzati. |
| `GET /issues/:identifier/efforts/worklogs?page=1&per_page=50` | `ListResponse<EffortWorklog>` comune. |

Issue senza dati:

```json
{
  "revision": 0,
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

Esempio di pagina:

```json
{
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

`available=false` permette alla UI futura di mostrare «Utente non più disponibile»;
un utente rimosso dal workspace ma con account esistente resta identificabile.
Non si espongono email nelle attribuzioni. Pagine da 50 fisse; `page` predefinita 1,
positiva e compatibile con il limite numerico; `per_page`, se fornito, deve essere
50. Ordine `work_date DESC, created_at DESC, id DESC`. Conteggio e righe sono letti
nello stesso statement. Una pagina oltre l'ultima restituisce `data: []` con il
conteggio effettivo. Una risorsa non accessibile restituisce lo stesso 404 di una
inesistente. L'ordinamento è stabile sullo stato letto; pagine successive possono
cambiare durante modifiche concorrenti e il client dovrà ricaricarle con US-23.

## 3. Contratti delle scritture future

Sono definiti DTO Go/TypeScript, ma nessuna route o funzione client di mutazione.
I seguenti contratti richiedono transazioni e policy T-02/US-18/19 prima dell'uso:

| Comando | Campi oltre all'involucro comune |
| --- | --- |
| `SetEffortEstimateRequest` | `estimate_minutes`: intero o null esplicito per rimuovere; ometterlo non equivale a rimuovere. |
| `SetEffortRemainingRequest` | `remaining_minutes`: intero non negativo obbligatorio. |
| `CreateEffortWorklogRequest` | `owner_user_id`, `work_date`, `duration_minutes`, `description` opzionale/null. |
| `UpdateEffortWorklogRequest` | Data, durata e descrizione desiderate; nessun titolare modificabile. |
| `DeleteEffortWorklogRequest` | Solo involucro comune. |
| `SetEffortSettingsRequest` | Configurazione completa: `enabled`, `timezone`; solo sessione owner/admin. |

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
revisione 0; le righe di stato verranno create nella transazione del primo comando.
Campi immutabili/ignoti delle scritture devono essere rifiutati dai futuri handler.

Idempotenza: chiave `(workspace, operatore, request_id)` senza TTL. SHA-256 del
JSON canonico della richiesta normalizzata, contenente nome operazione, destinazione,
revisione attesa e campi, esclusi credenziali e intestazioni di trasporto. Ordinare
le chiavi JSON; risolvere gli identificatori di destinazione a UUID prima del hash;
normalizzare descrizione omessa a null. La chiave non deriva dal contenuto: due
invii intenzionali identici usano UUID differenti. Non condividere chiavi fra comandi.

T-02 dovrà ricontrollare autorizzazioni e attivazione dopo le attese, riconoscere
un tentativo già committato prima del controllo di revisione e salvare ricevuta,
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
di stato non sono implementati da vincoli locali alla singola riga. Sono obblighi
espliciti della transazione T-02 e delle rispettive storie, non garanzie di T-01.

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
riattivazione è l'admin che la richiede. US-21 proietterà questi eventi nella
stessa attività dell'issue mantenendo compatibili gli eventi generici esistenti.

Per US-24/25, il formato esteso sarà `kuayle.workspace`, versione **2**:

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
- Accettare archivio v1 senza Efforts con default attivo/Europe-Rome; emettere v2
  soltanto quando l'estensione di trasferimento sarà operativa. Il backend T-01
  mantiene il formato v1 attuale e non dichiara un round-trip Efforts già disponibile.
- Validare durate, riferimenti e somme giornaliere prima dell'import atomico.
  Ripristinare date, stato disabilitato e snapshot senza rieseguire effetti worklog,
  azzeramenti o nuovi eventi. L'export richiede una vista coerente dei dati, oltre
  alla barriera privacy esistente, e mantiene i blocchi per workspace privati.

## 5. Scope e canali indiretti

`efforts:read` è nel limite di ruolo di owner/admin/member/guest;
`efforts:write` soltanto di owner/admin/member. Il limite di ruolo non sostituisce
titolarità, assegnazione e visibilità. I PAT esistenti non cambiano. La nuova riga
Efforts nella UI espande Read a read, Read & write a read+write. I preset con tutte
le letture includono read; solo Full access include anche write. Triage resta invariato.

Contratto per le storie successive, da applicare prima di esporre scritture o archivi:

| Percorso | Regola PAT |
| --- | --- |
| Mutazioni Efforts | `efforts:read` + `efforts:write` e policy corrente. |
| Configurazione in scrittura | Solo sessione interattiva owner/admin, nessun PAT. |
| Storico issue | Gli eventi Efforts sono omessi senza `efforts:read`; gli altri eventi restano leggibili con `issues:read`. |
| Export workspace v2 | `workspace:transfer` + `efforts:read`; nessun archivio Efforts parziale. |
| Import workspace | Mantiene il vincolo attuale di sessione interattiva. |
| Duplicazione | Con il comportamento Efforts operativo richiede `issue:create` + read/write Efforts; negazione uniforme senza dipendere dalla presenza di una stima. |
| Update/bulk generici | I campi senza effetti Efforts mantengono il contratto attuale. Le operazioni che possono mutare Efforts richiedono read/write, prima di qualsiasi effetto; niente mutazioni parziali o salti degli invarianti. |
| Delete issue/bulk | Con il lifecycle Efforts operativo richiede anche read/write; non dipende dalla presenza dei dati. |
| Condivisione pubblica | Nessun dato o evento Efforts, nessuna credenziale Efforts derivata dal link. |

La tabella descrive i controlli da integrare in US-18/19 e nel lifecycle: T-01 non
modifica le autorizzazioni delle scritture generiche esistenti. Finché non sono
implementati quei controlli, le scritture Efforts restano non registrate.

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

Restano fuori da questa verifica le garanzie delle scritture T-02 e il round-trip
degli archivi Efforts US-24/25. I soli vincoli di riga non le sostituiscono.

# Efforts — confine transazionale T-02

Implementazione del 2026-10-17, branch `feature/efforts`.
Base: contratti [T-01](efforts-contracts.md); attività [T-02](../backlog/efforts.md).

## Perimetro

Il repository espone comandi interni tipizzati `SetEstimate`, `SetRemaining`,
`CreateWorklog`, `UpdateWorklog`, `DeleteWorklog` e `SetSettings`. Il service
restituisce le ricevute T-01 e pubblica il refresh dopo il commit. Non sono
registrate nuove route HTTP di scrittura Efforts, né funzioni client di mutazione.

Le chiusure esistenti sono già integrate: update singolo, bulk, triage,
automazioni parent/figli e GitHub. Stato issue, trigger lifecycle, eventuale
azzeramento del residuo ed eventi Efforts partecipano alla stessa transazione.
Gli eventi Efforts non usano `recordHistory`: un errore dello storico annulla
il salvataggio. Lo storico generico preesistente rimane distinto e best effort;
per update/triage il suo evento di stato è ora registrato dopo il salvataggio,
così una chiusura fallita non lascia un falso evento di stato.

US-18/19 devono ancora completare l'integrazione dei canali indiretti prima di
esporre le nuove scritture. Restano successive la proiezione degli eventi
nell'attività, l'integrazione delle viste, il refresh UI specifico e gli archivi.
Non è necessaria una nuova migrazione: T-02 usa le cinque tabelle della 47.

## Ordine dei lock

Ogni comando apre una transazione esplicita READ COMMITTED. Tutti i lock durano
fino al commit o rollback; non ci sono mutex applicativi né retry automatici.

1. Barriera privacy condivisa `(610046, 0)`, prima dei lock sulle righe.
2. Advisory lock configurazione per workspace, hash con seed `470`: condiviso
   per comandi issue e aggiornamenti issue, esclusivo per impostazioni/riattivazione.
3. Advisory lock del tentativo `(workspace, operatore, request_id)`, seed `471`.
   È presente solo nei comandi Efforts espliciti.
4. Membership dell'operatore e, per nuovi worklog, del titolare: team prima di workspace,
   `FOR SHARE`, ordinate per identificativi. Il lock workspace protegge anche il
   ruolo, non solo la presenza. `RemoveMember` acquisisce i grant team nello stesso
   ordine prima di eliminarli; le attribuzioni storiche non vengono cancellate.
5. Stati referenziati `FOR SHARE`, poi issue `FOR UPDATE`, in ordine di UUID.
   Anche `SetAssignees` blocca l'issue prima di sostituire le righe della junction.
6. Advisory lock budget `(workspace, titolare, data)`, seed `472`. Tutte le date
   coinvolte sono ordinate e deduplicate; il cambio data protegge entrambi i giorni.
7. Riga worklog da aggiornare/eliminare `FOR UPDATE`.

Le chiavi advisory dei budget esistono anche senza worklog. Il budget somma tutto
il lavoro della persona nel workspace, comprese issue nascoste; l'errore non
contiene somme o riferimenti a quel lavoro. L'aggiornamento esclude il worklog
corrente dalla somma prima di verificare la durata desiderata.

Dopo le attese si leggono le membership correnti, la visibilità e lo stato issue.
Ruolo e accesso restano protetti dai lock membership e dalla barriera privacy;
assegnazione e worklog dalla riga issue; attivazione e fuso dal lock configurazione.
Queste decisioni non possono quindi essere revocate durante una successiva attesa
sul budget. Una revoca arrivata dopo l'acquisizione dei lock si serializza dopo
il salvataggio. La data corrente viene calcolata dopo il lock budget con
`clock_timestamp() AT TIME ZONE`, senza usare l'ora di inizio transazione.

Le nuove garanzie passano dai repository applicativi. Scritture SQL dirette e il
futuro import devono rispettare un proprio protocollo; la sola somma SQL non
serializza i budget. Rimane vietato aggiornare la classificazione privacy dopo
scritture ordinarie nella stessa transazione, come previsto dalla migrazione 46.

## Revisioni, idempotenza ed errori

Valgono involucro, hash canonico e codici T-01. La ricevuta è inserita insieme a
worklog, pianificazione, revisione ed eventi. Un comando completato incrementa
una volta la revisione issue, anche se produce più eventi. Le impostazioni hanno
una revisione propria; ogni issue riconciliata alla riattivazione avanza una volta.

Il tentativo viene cercato prima della revisione attesa. Un replay riesamina i
permessi correnti e restituisce la ricevuta originale senza nuove scritture né
refresh. Può riuscire anche a funzione disabilitata; non riesegue l'operazione.
Gli snapshot conservano la titolarità per autorizzare replay dopo eliminazione
del worklog. Se l'issue logica è eliminata si restituisce risorsa non trovata.
Una chiave riutilizzata con comando/contenuto diverso produce
`EFFORT_REQUEST_CONFLICT`, senza sostituire la ricevuta.

- Risposta persa dopo commit: ritentare lo stesso comando con la stessa chiave.
- Errore di persistenza o rollback transazionale, inclusi `40001`/`40P01`:
  ritentare l'intera operazione con la stessa chiave dopo il rollback; nessun
  recupero parziale e nessun retry nascosto nel repository.
- `EFFORT_REVISION_CONFLICT`: rileggere lo stato, riallineare la bozza e usare
  un nuovo tentativo con la revisione corrente.
- Errori di autorizzazione, disattivazione o validazione: correggere la causa;
  nessuna ricevuta è stata consumata. I nuovi handler futuri dovranno mappare
  `EffortError` sui codici HTTP/dettagli già definiti in T-01.

La distinzione assente/zero si conserva anche nella prima lettura transazionale:
una riga SQL assente restituisce esplicitamente uno snapshot vuoto, senza usare i
puntatori che `sqlx` può allocare prima di scoprire l'assenza della riga.

## Chiusure, configurazione e canali

La categoria dello stato personalizzato determina conclusione/annullamento;
il legacy è usato quando non è disponibile uno stato referenziato. Le transizioni
con il solo slug legacy, incluso bulk, allineano `status_id` allo stato del team;
i relativi stati sono bloccati prima delle issue. Uno slug indisponibile per il
team rifiuta il salvataggio, senza mantenere una categoria precedente incoerente.
Con Efforts
attivo una chiusura azzera il residuo esistente e registra la causa `closure`.
Non crea righe Efforts per issue che non hanno mai avuto pianificazione.
Riaprire non ricostruisce il residuo. Disabilitare congela i valori; riattivare
azzera atomicamente le issue attualmente chiuse, con causa `reactivation`, senza
ricalcolare i residui aperti. Le date storiche non cambiano con il fuso.

I comandi interni applicano la matrice di ruoli, assegnazione, delega e titolarità
T-01. La configurazione richiede sessione owner/admin. L'admin può correggere
lavoro storico di un titolare rimosso/guest, ma non creargli nuovi worklog.
GitHub attribuisce gli eventi Efforts a `kind=system, system_source=github`,
senza account fittizi. Le regole di visibilità pubblica di GitHub restano valide.

I PAT portano il limite degli scope anche nel context Go. Update/bulk con campi
stato e rifiuto triage richiedono entrambi gli scope Efforts prima degli effetti,
indipendentemente dai valori presenti. Il repository ripete il controllo per
le effettive transizioni. Aggiornamenti ordinari senza transizione mantengono il
contratto scope esistente. Questo adattamento non completa tutto US-19.

Il nuovo service emette soltanto `app.refresh` generico dopo il commit, senza
worklog o ID issue; replay e rollback non emettono nuovi eventi. Le chiusure
mantengono i broadcast esistenti dopo il salvataggio. Il Hub ricontrolla membership
e privacy in consegna e conserva il comportamento generico dei workspace privati.
Non vengono aggiunte email o notifiche dedicate ai worklog.

## Verifica

Le prove T-02 usano PostgreSQL reale e connessioni con PID distinti. Le attese
sono osservate con `pg_blocking_pids`, invece di assumere un ordine tramite sleep.
Le fixture concorrenti hanno database temporanei dedicati, per non interferire
con le barriere privacy delle altre suite. Nessuna migrazione viene applicata al
database di sviluppo.

Copertura: rollback per errore su eventi/ricevute; budget inizialmente vuoto;
inserimenti su issue diverse; cambi data opposti e spostamento rifiutato;
budget con lavoro nascosto; replay concorrente/persistente e dopo cancellazione;
conflitti di revisione; revoche concorrenti di ruolo, workspace, team,
assegnazione e privacy; disattivazione in attesa; riattivazione atomica fallita;
chiusura/riapertura singola, bulk, triage e automazioni parent/figli; servizio
GitHub reale; scope PAT via HTTP e refresh WebSocket dopo commit.

Verifica conclusa il 2026-10-17 su PostgreSQL 17.11 locale:

- `go test ./... -race -count=1`: superato; 983 test riusciti, 6 saltati.
- Tutti i test Efforts PostgreSQL, migrazione, HTTP, GitHub e realtime sono stati
  eseguiti; nessuno è stato saltato per assenza del database.
- Un ulteriore passaggio mirato con `-race` su `TestEffortsT02HTTPAndRealtime`
  verifica anche il bulk con il solo slug legacy, il rollback della chiusura via HTTP,
  l'assenza di un falso evento
  di stato e la risposta 500 senza dettagli SQL.
- I 6 skip preesistenti riguardano analytics/issue con fixture esterne richieste,
  tre prove su agent run/log/eventi persistiti assenti e l'integrazione Docker
  opt-in. Non sono prove Efforts e non attestano una loro esecuzione.
- Installazioni nuove e upgrade/down della 47 sono coperti dai test migrazione
  T-01 eseguiti insieme alla suite. T-02 non modifica lo schema.

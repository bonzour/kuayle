# Efforts — requisiti di dettaglio

Stato: versione 1.0 — requisiti confermati; sessione di raccolta conclusa.
Data di apertura: 2026-10-16.
Data di conferma finale: 2026-10-16.
Branch di lavoro: `feature/efforts`.

## Obiettivo

Introdurre in Kuayle una funzionalità opzionale per gestire gli effort sulle issue,
usando Tempo Timesheets per Jira come riferimento funzionale. Le funzionalità di
Tempo non sono automaticamente requisiti di Kuayle: fanno fede le decisioni
raccolte in questo documento.

Il documento comprende 41 requisiti e 51 criteri di accettazione. Le proposte
P-01–P-09, approvate nella revisione finale, sono integrate come EFF-033–EFF-041;
la corrispondenza è riportata nel registro della sessione. Le scelte di schema,
endpoint e componenti interni appartengono alla successiva progettazione tecnica.

## Requisiti confermati

| ID | Requisito | Origine |
| --- | --- | --- |
| EFF-001 | La funzionalità deve poter essere abilitata e disabilitata a livello di workspace. Quando abilitata, si applica a tutte le issue del workspace. | Sessione, risposta 1.2 |
| EFF-002 | Ogni issue deve supportare una stima iniziale dell'effort. | Sessione, risposta 1.1 |
| EFF-003 | Ogni issue deve supportare registrazioni del lavoro effettivamente svolto e la visualizzazione del relativo totale. | Richiesta iniziale e risposte 1.1, 1.3 |
| EFF-004 | Ogni issue deve supportare la gestione del lavoro residuo. | Sessione, risposta 1.1 |
| EFF-005 | La prima versione comprende la gestione sulla singola issue, inclusi i totali delle sotto-issue previsti in EFF-022. Timesheet personali, report aggregati per team/progetto, approvazioni e chiusura dei periodi sono fuori dal perimetro iniziale. | Sessione, risposte 1.3 e 7.2 |
| EFF-006 | Inserendo una nuova registrazione di lavoro, un residuo impostato diminuisce della durata registrata, con minimo zero; un residuo assente resta assente ed è mostrato a zero. Il residuo è correggibile manualmente da utenti autorizzati sulle issue aperte. Modifiche ed eliminazioni dei worklog seguono invece EFF-017. | Sessione, risposte 2.1, 5.3, 6.1, 9.1 e 10.3 |
| EFF-007 | Il lavoro viene registrato manualmente tramite data e durata obbligatorie e descrizione facoltativa. Timer e inserimento tramite orari di inizio/fine non fanno parte della prima versione. | Sessione, risposte 2.2 e 6.2 |
| EFF-008 | Disabilitando Efforts, i dati effort già presenti devono essere conservati e mostrati in sola lettura. | Sessione, risposta 2.3 |
| EFF-009 | Con Efforts attivo, owner, admin e membri possono registrare il proprio lavoro su qualunque issue a cui hanno accesso, anche se non ne sono assegnatari. I guest non possono registrare lavoro. | Sessione, risposta 3.1 |
| EFF-010 | Il titolare del lavoro può modificare ed eliminare le proprie registrazioni se ha ancora un ruolo non guest e accesso all'issue; owner e admin possono modificare ed eliminare tutte le registrazioni sulle issue a cui hanno accesso. Un membro non può modificare o eliminare le registrazioni altrui. | Sessione, risposta 3.2 e revisione finale P-05 |
| EFF-011 | Gli utenti del workspace con accesso all'issue possono vedere il dettaglio delle registrazioni, inclusi autore, durata e descrizione, oltre al totale. I visitatori dei link pubblici sono esclusi secondo EFF-023. | Sessione, risposte 3.3 e 7.3 |
| EFF-012 | Efforts è attivo per default sia nei workspace nuovi sia in quelli esistenti al rilascio della funzionalità. Solo owner e admin possono attivarlo e disattivarlo. | Sessione, risposta 4.1 |
| EFF-013 | Con Efforts attivo, solo gli assegnatari dell'issue, gli owner e gli admin possono impostare o modificare la stima iniziale e correggere manualmente il residuo. I guest sono esclusi anche se assegnatari. L'accesso all'issue resta necessario. Sulle issue concluse/annullate il residuo è bloccato a zero secondo EFF-032. | Sessione, risposte 4.2 e 10.3 |
| EFF-014 | La stima iniziale è facoltativa: si può registrare lavoro anche senza una stima. Gli utenti autorizzati possono rivederla anche dopo la prima registrazione di lavoro. | Sessione, risposta 4.3 |
| EFF-015 | Registrando lavoro su un'issue aperta senza stima né residuo impostati, il consuntivo aumenta e il residuo mostrato è zero. Questo zero è solo di presentazione: il residuo resta assente ai fini di EFF-016. Esempio: dopo 2 ore lavorate senza stima, impostare la stima a 8 ore inizializza il residuo a 6 ore. Resta possibile impostare manualmente il residuo con i permessi previsti. | Sessione, risposte 5.1 e 6.1 |
| EFF-016 | Impostando o modificando la stima su un'issue aperta, se il residuo è assente viene inizializzato a `max(stima − consuntivo, 0)`; se il residuo è già impostato, resta invariato. Sulle issue concluse/annullate prevale il residuo zero di EFF-032. | Sessione, risposte 5.2 e 10.3 |
| EFF-017 | Modificare o eliminare una registrazione aggiorna il consuntivo ma lascia invariato il residuo. Non deve essere applicato alcun ricalcolo automatico del residuo per differenza di durata o restituzione delle ore eliminate. | Sessione, risposta 5.3 |
| EFF-018 | Stima, residuo e lavoro registrato sono espressi in ore/minuti, con precisione al minuto; deve essere supportato l'inserimento di durate come `1h 30m`. | Sessione, risposta 6.2 |
| EFF-019 | Ogni worklog richiede una data e una durata strettamente positiva; la descrizione è facoltativa. Le medesime regole valgono durante la modifica. | Sessione, risposta 6.2 |
| EFF-020 | È possibile registrare e correggere lavoro per oggi e per qualsiasi data passata. Le date future non sono ammesse. «Oggi» è determinato dal fuso orario del workspace, anche nelle registrazioni per conto altrui. | Sessione, risposte 6.3 e 9.3 |
| EFF-021 | Con Efforts attivo, quando l'issue viene conclusa o annullata il residuo proprio viene azzerato. Lo stato concluso/annullato non impedisce registrazioni e correzioni di lavoro, nel rispetto degli altri permessi. Con Efforts disabilitato si applica EFF-033. | Sessione, risposta 7.1 e revisione finale P-01 |
| EFF-022 | Un'issue con sotto-issue deve mostrare separatamente i propri valori di stima, consuntivo e residuo e i totali comprensivi delle sotto-issue. La visualizzazione aggregata non modifica i valori propri delle issue. | Sessione, risposta 7.2 |
| EFF-023 | Nei link pubblici delle issue non deve essere esposto alcun dato Efforts: né stima/consuntivo/residuo, né registrazioni individuali. | Sessione, risposta 7.3 |
| EFF-024 | Alla riapertura di un'issue, il residuo azzerato alla chiusura rimane zero; può essere impostato manualmente da assegnatari non guest, owner o admin. Non si ripristina il valore precedente e non si ricalcola dalla stima. | Sessione, risposta 8.1 |
| EFF-025 | Owner e admin possono inserire lavoro per conto di altri membri del workspace che hanno accesso all'issue. Devono essere conservate sia l'identità della persona che ha svolto il lavoro sia quella di chi ha effettuato l'inserimento. | Sessione, risposta 8.2 |
| EFF-026 | L'attività dell'issue deve contenere uno storico completo delle operazioni su stima, residuo e worklog, inclusi inserimenti, modifiche ed eliminazioni, con autore dell'operazione e valori precedenti e successivi. | Sessione, risposta 8.3 |
| EFF-027 | È consentito registrare una durata superiore al residuo. L'intera durata viene aggiunta al consuntivo e il residuo viene limitato a zero, senza valori negativi: con residuo 1 ora, registrare 2 ore porta il residuo a zero. | Sessione, risposta 9.1 |
| EFF-028 | Il totale giornaliero del lavoro attribuito alla stessa persona, su tutte le issue dello stesso workspace, non può superare 24 ore. Inserimenti e modifiche che supererebbero il limite devono essere rifiutati, anche se effettuati per conto altrui da owner/admin. | Sessione, risposta 9.2 |
| EFF-029 | Il fuso orario usato da Efforts è configurato a livello di workspace ed è modificabile solo da owner e admin. | Sessione, risposta 9.3 |
| EFF-030 | Il fuso orario iniziale è `Europe/Rome` per workspace nuovi ed esistenti. | Sessione, risposta 10.1 |
| EFF-031 | Nel dettaglio dell'issue, Efforts presenta un riepilogo nella barra laterale e una sezione con le registrazioni nel corpo dell'issue. I valori propri e i totali comprensivi delle sotto-issue restano distinti. | Sessione, risposta 10.2 |
| EFF-032 | Con Efforts attivo, il residuo proprio di un'issue conclusa o annullata è obbligatoriamente zero e non è modificabile finché l'issue non viene riaperta. Owner e admin non fanno eccezione. La riapertura lascia il residuo a zero secondo EFF-024. | Sessione, risposta 10.3 |
| EFF-033 | Disabilitare Efforts congela i dati, anche in caso di successiva chiusura dell'issue. Riattivarlo conserva stime/worklog e azzera, con storico, solo i residui delle issue attualmente concluse/annullate; quelli delle issue aperte restano invariati. Fuso e attivazione restano amministrabili. | Revisione finale, P-01 |
| EFF-034 | Durate intere in minuti, stima/residuo non negativi, formati ore/minuti senza arrotondamenti silenziosi; descrizione facoltativa in testo semplice fino a 2.000 caratteri. Rimuovere la stima non cambia worklog o residuo. | Revisione finale, P-02 |
| EFF-035 | La data del lavoro è una data di calendario. Cambiare il fuso IANA del workspace non sposta le date salvate o i totali storici; i salvataggi successivi usano il nuovo riferimento temporale. | Revisione finale, P-03 |
| EFF-036 | Gli aggregati comprendono l'issue e tutti i discendenti visibili, una sola volta; sono in sola lettura, distinguono valori mancanti e non rivelano issue non accessibili. Chiudere la parent non azzera i residui delle sotto-issue ancora aperte. | Revisione finale, P-04 |
| EFF-037 | Nuove registrazioni sono attribuibili solo a utenti non guest con accesso all'issue. Il titolare del lavoro è distinto dall'operatore e non è riassegnabile dopo l'inserimento. Ruolo e accesso correnti regolano le operazioni; rimozione dell'utente o cambio di ruolo non cancellano lo storico. | Revisione finale, P-05 |
| EFF-038 | Duplicare copia la stima ma non worklog/storico; spostare o convertire in progetto conserva gli effort sulle issue. Eliminare un'issue elimina i suoi effort/storico, con avviso esplicito anche a funzione disabilitata. Le sotto-issue conservate mantengono i propri dati. | Revisione finale, P-06 |
| EFF-039 | API e token applicano autorizzazioni e validazioni Efforts; i token richiedono scope espliciti. Export/import del workspace conservano configurazione, dati e attribuzioni, nel rispetto dei vincoli esistenti. Nessun export timesheet dedicato nella prima versione. | Revisione finale, P-07 |
| EFF-040 | Salvataggi, totali, residuo e storico restano coerenti anche in caso di errore o concorrenza. Un invio ritentato non duplica il lavoro; i conflitti non sovrascrivono modifiche altrui. Le sessioni autorizzate si aggiornano in realtime, senza notifiche email/inbox dedicate. | Revisione finale, P-08 |
| EFF-041 | Il dettaglio presenta riepilogo e registro secondo EFF-031, ordinamento decrescente per data/inserimento, pagine o caricamenti da 50 righe, moduli accessibili con errori contestuali, conferma di eliminazione e sola lettura esplicita. | Revisione finale, P-09 |

## Permessi confermati

La matrice seguente riguarda gli utenti del workspace. Le operazioni sugli effort
di un'issue richiedono l'accesso all'issue; la sola appartenenza al workspace non
supera i vincoli dei team privati. Le scritture dirette sui dati effort richiedono
inoltre che Efforts sia attivo.
Le condivisioni pubbliche non espongono dati Efforts, secondo EFF-023.

| Operazione | Owner | Admin | Membro | Guest |
| --- | --- | --- | --- | --- |
| Leggere totale e dettagli | Sì | Sì | Sì | Sì |
| Registrare lavoro proprio | Sì | Sì | Sì | No |
| Registrare lavoro per conto di altri membri con accesso all'issue | Sì | Sì | No | No |
| Modificare/eliminare registrazioni proprie | Sì | Sì | Sì | No, anche se registrate prima del cambio di ruolo |
| Modificare/eliminare registrazioni altrui | Sì | Sì | No | No |
| Impostare/modificare stima e residuo manualmente | Sì | Sì | Solo se assegnatario | No |
| Attivare/disattivare Efforts per il workspace | Sì | Sì | No | No |
| Modificare il fuso orario del workspace | Sì | Sì | No | No |

Attivazione/disattivazione e fuso sono impostazioni del workspace e restano
disponibili a owner/admin anche quando Efforts è disattivato. Le operazioni sui
singoli dati Efforts richiedono la funzione attiva. L'eliminazione dell'intera
issue/workspace e il ripristino tramite import seguono invece EFF-038–EFF-039.
Con Efforts attivo, il residuo proprio delle issue concluse/annullate rimane zero
per tutti i ruoli: i permessi della matrice non superano questa regola. La stima
resta rivedibile.

La riduzione automatica del residuo conseguente all'inserimento di lavoro è un
effetto del worklog consentito da EFF-009. Non richiede che l'autore sia anche
autorizzato a correggere manualmente il residuo: un membro non assegnatario può
quindi registrare lavoro e causarne la riduzione automatica, ma non impostarne
liberamente il valore.

Una registrazione distingue l'autore del lavoro dall'operatore che l'ha inserita
o modificata. Inserire lavoro per conto di altri non trasferisce la titolarità
del lavoro all'operatore. Lo storico permette di attribuire le operazioni
amministrative alla persona che le ha effettuate.

## Glossario di lavoro

- **Stima iniziale:** effort previsto per completare l'issue, facoltativo e
  rivedibile anche dopo aver registrato lavoro. Il nome non implica un valore
  immutabile. È espresso in ore/minuti; le revisioni sono tracciate nello storico.
- **Registrazione di lavoro / worklog:** registrazione di un contributo di lavoro
  svolto su una issue, inserita manualmente con data e durata positiva obbligatorie
  e descrizione facoltativa.
  I permessi sono definiti sopra e le validazioni sono specificate in EFF-034.
- **Tempo lavorato / consuntivo:** totale del lavoro registrato sull'issue.
- **Residuo:** effort che si prevede ancora necessario per completare l'issue.
  Diminuisce automaticamente quando si inserisce nuovo lavoro ed è correggibile
  manualmente. Modificare o eliminare lavoro già registrato non lo cambia.
  Non coincide necessariamente con stima iniziale meno consuntivo.
- **Titolare del lavoro:** persona a cui sono attribuiti durata e data del worklog;
  è l'«autore» ai fini della modifica delle proprie registrazioni.
- **Operatore:** persona che effettua un inserimento, una modifica o un'eliminazione;
  nelle deleghe può essere diversa dal titolare e viene registrata nello storico.
- **Issue chiusa:** issue in una categoria di stato conclusa o annullata, incluse
  le categorie equivalenti degli stati personalizzati dei team. Le regole si
  applicano anche alle transizioni automatiche e alle operazioni in blocco.

## Regole di calcolo confermate

Il consuntivo corrisponde alla somma delle durate delle registrazioni esistenti.
Il residuo ha un aggiornamento distinto, secondo le seguenti regole.

| Operazione | Consuntivo | Residuo |
| --- | --- | --- |
| Nuova registrazione di durata `d` | Aumenta di `d`, se il nuovo totale giornaliero non supera 24 ore | Se impostato a `r`, diventa `max(r − d, 0)`; se assente, resta assente e viene mostrato zero. Sulle issue chiuse rimane zero. |
| Modifica durata da `d1` a `d2` | Varia di `d2 − d1` | Invariato. |
| Modifica data o descrizione | Invariato | Invariato. |
| Eliminazione di una registrazione di durata `d` | Diminuisce di `d` | Invariato. |
| Impostazione/revisione della stima a `s` | Invariato | Su issue aperta: se assente, `max(s − consuntivo, 0)`; altrimenti invariato. Su issue chiusa con Efforts attivo: zero. |
| Correzione manuale del residuo su issue aperta | Invariato | Assume il valore indicato dall'utente autorizzato. Sulle issue chiuse la correzione non è consentita. |
| Conclusione/annullamento dell'issue con Efforts attivo | Invariato | Il residuo proprio viene azzerato e rimane bloccato a zero fino alla riapertura. Il caso Efforts disabilitato segue EFF-033. |
| Riapertura dopo azzeramento alla chiusura | Invariato | Resta zero, senza ripristino o ricalcolo automatico. |
| Chiusura/riapertura con Efforts disabilitato | Invariato | Il valore salvato resta congelato. |
| Riattivazione di Efforts | Invariato | Zero sulle issue attualmente chiuse; invariato sulle issue aperte. |

Esempio confermato dalle regole: con stima 8 ore, consuntivo 2 ore e residuo
6 ore, correggere la registrazione da 2 a 3 ore porta il consuntivo a 3 ore e
lascia il residuo a 6 ore. Se invece si elimina la registrazione da 2 ore,
il consuntivo diventa zero e il residuo resta 6 ore.

È necessario distinguere il residuo assente da quello impostato a zero, pur
mostrando zero in entrambi i casi. Registrare lavoro quando il residuo è assente
non lo inizializza. L'inserimento successivo di una stima applica quindi EFF-016.
Un residuo impostato esplicitamente a zero, invece, rimane tale alla revisione
della stima. La rappresentazione tecnica di questa distinzione sarà definita
durante la progettazione.

## Issue concluse, sotto-issue e condivisione

Con Efforts attivo, concludere o annullare un'issue azzera il suo residuo proprio,
senza modificare stima e registrazioni. Le registrazioni e le correzioni restano consentite agli
utenti autorizzati anche dopo la chiusura. Riaprire l'issue non ripristina il
residuo precedente: un utente autorizzato può impostare una nuova previsione.
Finché l'issue resta conclusa o annullata con Efforts attivo, nessun utente può correggere il residuo
proprio a un valore diverso da zero; la revisione della stima non lo modifica.

I valori propri e quelli aggregati devono essere riconoscibili e separati.
Esempio: un'issue con stima/consuntivo/residuo di 8/2/6 ore e una sotto-issue con
4/1/3 ore mostra valori propri 8/2/6 e totali 12/3/9. Le somme non devono contare
due volte il lavoro. Profondità della gerarchia e presentazione dei valori
mancanti seguono EFF-036. I totali devono rispettare l'accesso alle
issue coinvolte; non possono esporre dati di issue non accessibili.

Le pagine e risposte pubbliche non espongono dati Efforts, compresi eventuali
eventi dello storico che ne rivelino i valori. La sola rimozione dei controlli
visivi non soddisfa EFF-023 se i dati rimangono accessibili nella risposta.

## Storico delle operazioni

Ogni inserimento, modifica o eliminazione di worklog e ogni variazione di stima
o residuo deve essere ricostruibile nello storico dell'issue. La scelta dello
storico completo comprende gli aggiornamenti automatici del residuo: il sistema
deve distinguere la causa (ad esempio registrazione di lavoro o chiusura)
dalla correzione manuale e attribuire l'operazione a chi l'ha attivata.

Per un worklog inserito per conto di un altro membro, lo storico conserva la
distinzione fra autore del lavoro e autore dell'operazione. L'eliminazione di un
worklog lo rimuove dal consuntivo ma non deve rendere irricostruibile l'operazione
di eliminazione. Le modalità tecniche di conservazione saranno definite in fase
di progettazione.

## Campi e validazioni confermati

| Campo | Obbligatorietà | Regola |
| --- | --- | --- |
| Stima iniziale | Facoltativa | Ore/minuti, precisione al minuto, valore non negativo; rivedibile dopo l'inserimento di lavoro. |
| Residuo | Può essere assente | Ore/minuti, precisione al minuto, valore non negativo. Se assente viene visualizzato zero, mantenendo la distinzione descritta sopra. |
| Data del lavoro | Obbligatoria | Oggi o qualunque data passata; mai futura. «Oggi» è calcolato nel fuso del workspace, inizialmente `Europe/Rome`. |
| Durata del worklog | Obbligatoria | Da 1 a 1.440 minuti; inoltre la somma giornaliera della persona nel workspace non può superare 1.440 minuti. |
| Descrizione del worklog | Facoltativa | Testo semplice multilinea fino a 2.000 caratteri; il salvataggio è consentito anche senza descrizione. |

Il limite giornaliero riguarda la persona a cui il lavoro è attribuito, non
l'operatore che inserisce per conto altrui. Il controllo include tutte le issue
del workspace, anche quelle non visibili all'operatore; un rifiuto non deve
rivelare identificativi, titoli o dettagli di tali issue. I workspace sono
conteggiati separatamente. Il limite è sempre di 24 ore per data di lavoro,
anche nelle giornate di cambio dell'ora legale.

Nel modificare una registrazione, il controllo deve usare il totale risultante,
senza sommare contemporaneamente la vecchia e la nuova durata. Cambiare la data
deve liberare il totale del giorno originale e verificare quello del giorno di
destinazione. La regola deve reggere anche in presenza di salvataggi simultanei.

I dettagli di validazione e conservazione delle date sono definiti in EFF-034–EFF-035.

## Criteri di accettazione

Gli scenari seguenti, insieme ad AC-043–AC-051 nelle regole di completamento,
costituiscono la base di verifica funzionale. Salvo indicazione diversa, le
operazioni di scrittura assumono Efforts attivo e i permessi necessari.
Le modifiche manuali del residuo assumono un'issue aperta.

| ID | Requisiti | Scenario e risultato atteso |
| --- | --- | --- |
| AC-001 | EFF-002, EFF-003, EFF-004, EFF-006 | Data un'issue con 8 ore di stima, 8 ore residue e nessun lavoro registrato, quando viene salvata una registrazione di 2 ore, il totale lavorato è 2 ore e il residuo è 6 ore. |
| AC-002 | EFF-006 | Dato un residuo calcolato di 6 ore, quando un utente autorizzato lo corregge manualmente a 5 ore, il valore salvato e visualizzato è 5 ore. |
| AC-003 | EFF-007 | Con Efforts attivo, un utente autorizzato può registrare manualmente lavoro indicando data, durata e descrizione; la registrazione compare nell'issue e contribuisce al totale. |
| AC-004 | EFF-008, EFF-033 | Dato un workspace con dati effort, quando la funzionalità viene disabilitata, i dati restano conservati e consultabili; inserimento/modifica/eliminazione diretti dei worklog e modifiche dirette di stima/residuo sono bloccati. Impostazioni, eliminazione dell'issue e trasferimento workspace seguono EFF-033/038/039. |
| AC-005 | EFF-009 | Con Efforts attivo, un membro che ha accesso all'issue può registrarvi lavoro proprio anche se l'issue è assegnata a un'altra persona o non è assegnata. |
| AC-006 | EFF-009, EFF-011 | Un guest che ha accesso all'issue può consultarne totale e dettagli effort, ma non può creare registrazioni di lavoro. |
| AC-007 | EFF-010 | Con Efforts attivo, un membro può correggere o eliminare una propria registrazione, ma non quelle di un altro utente; owner e admin con accesso all'issue possono intervenire anche su registrazioni altrui. |
| AC-008 | EFF-009, EFF-010, EFF-011 | Un utente senza accesso a un'issue non può consultarne né modificarne gli effort, anche conoscendo l'identificativo dell'issue o di una registrazione. |
| AC-009 | EFF-008, EFF-010 | Con Efforts disabilitato, anche owner, admin e titolare non possono modificare o eliminare singoli worklog tramite le operazioni Efforts; i dati restano leggibili a chi ha accesso all'issue. |
| AC-010 | EFF-012 | Al rilascio della funzionalità i workspace esistenti hanno Efforts attivo; anche un workspace creato successivamente nasce con Efforts attivo. Le successive scelte esplicite di disattivazione devono essere rispettate. |
| AC-011 | EFF-001, EFF-012 | Owner e admin possono disattivare e riattivare Efforts per tutte le issue del workspace; membri e guest non possono cambiare questa impostazione. |
| AC-012 | EFF-013 | Con Efforts attivo e accesso all'issue, un membro assegnatario può modificare stima e residuo; un membro non assegnatario e un guest non possono farlo. Owner e admin possono farlo anche senza assegnazione. |
| AC-013 | EFF-014 | Un utente abilitato a registrare lavoro può salvare una registrazione su un'issue senza stima; l'assenza della stima non blocca il salvataggio. |
| AC-014 | EFF-013, EFF-014 | Su un'issue con lavoro già registrato, un utente autorizzato può rivedere la stima. Il consuntivo continua a rappresentare le registrazioni effettive e non viene alterato dalla revisione della stima. |
| AC-015 | EFF-006, EFF-009, EFF-013 | Un membro non assegnatario registra 2 ore su un'issue accessibile con residuo di 8 ore: il residuo passa automaticamente a 6 ore, ma l'utente non può correggerlo manualmente. |
| AC-016 | EFF-014, EFF-015 | Su un'issue senza stima né residuo impostati e senza registrazioni, inserire 2 ore di lavoro produce un consuntivo di 2 ore e un residuo visualizzato di zero. |
| AC-017 | EFF-015, EFF-016 | Quando il residuo è assente, anche se mostrato come zero, impostare una stima di 8 ore con 2 ore di consuntivo inizializza il residuo a 6 ore; con 10 ore di consuntivo lo inizializza a zero. |
| AC-018 | EFF-016 | Con stima 8 ore, consuntivo 2 ore e residuo impostato a 5 ore, modificare la stima a 10 ore lascia il residuo a 5 ore e il consuntivo a 2 ore. |
| AC-019 | EFF-017 | Con un'unica registrazione da 2 ore e residuo 6 ore, correggere la durata a 3 ore porta il consuntivo a 3 ore e lascia il residuo a 6 ore. Anche le sole correzioni di data o descrizione lasciano il residuo invariato. |
| AC-020 | EFF-017 | Con un'unica registrazione da 2 ore e residuo 6 ore, eliminare la registrazione porta il consuntivo a zero e lascia il residuo a 6 ore. |
| AC-021 | EFF-015, EFF-016 | Su un'issue con 2 ore lavorate, senza stima e con residuo mostrato a zero, impostare una stima di 8 ore produce residuo 6 ore se il residuo è ancora assente. Se un utente autorizzato lo aveva esplicitamente impostato a zero, il residuo resta zero. |
| AC-022 | EFF-018, EFF-019 | Una durata di `1h 30m` equivale a 90 minuti; una registrazione di 1 minuto è valida. Una durata nulla, negativa o inferiore al minuto non costituisce un worklog valido. |
| AC-023 | EFF-007, EFF-019 | Con data e durata valide, un worklog può essere salvato senza descrizione. La mancanza della data o della durata impedisce il salvataggio, sia in inserimento sia in modifica. |
| AC-024 | EFF-020, EFF-029 | Sono ammesse registrazioni per oggi e date passate senza una finestra retroattiva prefissata, valutando «oggi» nel fuso del workspace. Una registrazione con data futura viene rifiutata; non è possibile spostare una registrazione esistente a una data futura. |
| AC-025 | EFF-021 | Con Efforts attivo, concludere o annullare un'issue con residuo 5 ore lo porta a zero, lasciando invariati stima e consuntivo. |
| AC-026 | EFF-021 | Su un'issue conclusa o annullata, un utente autorizzato può inserire o correggere lavoro quando Efforts è attivo; lo stato dell'issue non costituisce un blocco al salvataggio. |
| AC-027 | EFF-024 | Riaprendo un'issue il cui residuo è stato azzerato alla chiusura, il residuo resta zero anche se stima meno consuntivo è positivo. Solo un utente con i permessi previsti può correggerlo manualmente. |
| AC-028 | EFF-022 | Un'issue con valori propri 8/2/6 ore e una sotto-issue con 4/1/3 ore mostra separatamente i propri valori e i totali 12/3/9; il totale non viene salvato come nuovo valore proprio. |
| AC-029 | EFF-023 | Aprendo un link pubblico, né la pagina né le relative risposte dati espongono stime, consuntivi, residui, worklog o eventi di storico Efforts. |
| AC-030 | EFF-025 | Owner o admin con accesso all'issue possono inserirvi 2 ore per un altro membro con accesso all'issue: il lavoro è attribuito al membro e l'inserimento all'operatore. Un membro ordinario non può effettuare lo stesso inserimento per conto altrui. |
| AC-031 | EFF-025 | Owner e admin non possono inserire lavoro per conto di una persona che non appartiene al workspace o non ha accesso all'issue. |
| AC-032 | EFF-026 | Dopo una modifica da 2 a 3 ore e la successiva eliminazione del worklog, lo storico consente di ricostruire inserimento, modifica ed eliminazione, indicando autore e valori prima/dopo; il worklog eliminato non contribuisce al consuntivo. |
| AC-033 | EFF-026 | Una revisione della stima e una correzione manuale del residuo producono eventi con autore e valori prima/dopo; l'azzeramento alla chiusura è distinguibile da una correzione manuale. |
| AC-034 | EFF-027 | Con residuo 1 ora e 2 ore da registrare, se il limite giornaliero lo consente, vengono salvate tutte le 2 ore e il residuo diventa zero; la registrazione non è troncata né rifiutata per superamento del residuo. |
| AC-035 | EFF-028 | Con 23 ore attribuite a una persona in una data del workspace, inserire un'altra ora è consentito; inserire 1 ora e 1 minuto è rifiutato. Vale anche su un'altra issue e per inserimenti effettuati da un admin per conto della persona. |
| AC-036 | EFF-028 | Con un totale giornaliero di 24 ore che include una registrazione da 2 ore, modificarla in 1 ora è consentito e porta il totale a 23; modificarla in 3 ore è rifiutato. |
| AC-037 | EFF-028 | Spostare un worklog di 2 ore verso una data che già contiene 23 ore per la stessa persona nel workspace è rifiutato, senza alterare il worklog o i totali di entrambi i giorni. |
| AC-038 | EFF-028 | Due inserimenti simultanei non possono portare il totale giornaliero oltre 24 ore: se insieme eccedono il limite, almeno uno è rifiutato e i dati restano coerenti. |
| AC-039 | EFF-020, EFF-029, EFF-030 | Alla prima introduzione di Efforts, «oggi» è calcolato usando `Europe/Rome`, anche se il server o il browser usano un altro fuso. Owner/admin possono cambiare il fuso; membri/guest non possono farlo. |
| AC-040 | EFF-005, EFF-031 | Nel dettaglio issue sono presenti il riepilogo laterale e la sezione Efforts con le registrazioni. Un'issue con sotto-issue distingue valori propri e totali aggregati. L'esperienza Efforts della prima versione non introduce timesheet, report team/progetto o approvazioni. |
| AC-041 | EFF-021, EFF-032 | Su un'issue conclusa/annullata con Efforts attivo, anche owner e admin non possono impostare un residuo non nullo. Registrare o correggere lavoro e rivedere la stima non fanno risalire il residuo da zero. |
| AC-042 | EFF-024, EFF-032 | Dopo la riapertura il residuo resta zero, ma un assegnatario non guest, owner o admin può nuovamente impostarlo manualmente. |

## Tracciabilità delle decisioni

Tutti i punti raccolti durante la sessione sono stati risolti. La tabella mantiene
gli identificativi D-01–D-17 per collegare le domande iniziali alle regole definitive.

| ID | Ambito | Riferimenti definitivi |
| --- | --- | --- |
| D-01 | Calcolo e validazione del residuo | EFF-006, EFF-015, EFF-016, EFF-017, EFF-027, EFF-034 |
| D-02 | Modalità di registrazione | EFF-007 |
| D-03 | Disattivazione, riattivazione e dati in sola lettura | EFF-008, EFF-033 |
| D-04 | Default e amministrazione | EFF-012, EFF-029, EFF-030 |
| D-05 | Campi, unità e limiti | EFF-018, EFF-019, EFF-028, EFF-034 |
| D-06 | Permessi, delega e cambi di ruolo | EFF-009, EFF-010, EFF-011, EFF-025, EFF-037 |
| D-07 | Stima facoltativa, revisione e rimozione | EFF-002, EFF-013, EFF-014, EFF-016, EFF-026, EFF-034 |
| D-08 | Date, fuso e limite giornaliero | EFF-020, EFF-028, EFF-029, EFF-030, EFF-035 |
| D-09 | Chiusura, annullamento e riapertura | EFF-021, EFF-024, EFF-032, EFF-033 |
| D-10 | Gerarchia e aggregazione | EFF-022, EFF-036 |
| D-11 | Duplicazione, spostamento, conversione ed eliminazione | EFF-037, EFF-038 |
| D-12 | Interfaccia e consultazione | EFF-031, EFF-041 |
| D-13 | Condivisione, storico e notifiche | EFF-023, EFF-026, EFF-040 |
| D-14 | API e personal access token | EFF-039 |
| D-15 | Export/import workspace | EFF-039 |
| D-16 | Concorrenza e realtime | EFF-040 |
| D-17 | Accettazione | AC-001–AC-051 |

## Regole di completamento confermate

Le regole seguenti dettagliano EFF-033–EFF-041 e sono state approvate nella
revisione finale della sessione.

### EFF-033 — Disattivazione, riattivazione e chiusura

Con Efforts disabilitato, i valori salvati sono congelati: anche una chiusura o
un annullamento dell'issue non cambia il residuo conservato. La funzione resta
in sola lettura; se non esistono dati propri o aggregati, si nasconde il pannello
vuoto. Alla riattivazione si conservano stime e worklog e si azzerano i residui
delle issue attualmente concluse/annullate, registrando la causa nello storico.
I residui delle issue aperte non vengono ricalcolati.

Esempio confermato: residuo 5 ore → disabilitazione → chiusura dell'issue →
residuo storico ancora 5 ore in sola lettura → riattivazione → residuo zero.
L'azzeramento e la riattivazione devono essere coerenti e non lasciare dati
parzialmente aggiornati in caso di errore. L'impostazione del fuso resta gestibile
da owner/admin anche a funzione disabilitata, senza modificare le date salvate.

### EFF-034 — Durate, descrizione e rimozione della stima

Stima e residuo impostati manualmente ammettono zero e valori positivi interi in
minuti, mai valori negativi. Il worklog ammette da 1 a 1.440 minuti e rispetta il
limite giornaliero già confermato. Le durate non vengono arrotondate silenziosamente.
Si supportano minuti, ore e combinazioni, ad esempio `90m`, `2h`, `1h 30m`;
un numero senza unità indica minuti. Giorni e settimane non sono unità di input
nella prima versione.

La descrizione è testo semplice, anche multilinea, fino a 2.000 caratteri. Un
campo vuoto è valido. Togliere la stima la rende assente, senza cancellare lavoro
o cambiare il residuo già impostato; l'operazione è tracciata. Un residuo impostato
non torna assente mediante la rimozione della stima.

### EFF-035 — Fuso orario e date conservate

La data del lavoro è una data di calendario, distinta dal momento in cui viene
inserita la registrazione. Il fuso del workspace usa un identificativo IANA
valido, inizialmente `Europe/Rome`, con gestione dell'ora legale. Cambiare il
fuso non sposta le date dei worklog e non ridistribuisce i totali già registrati;
la nuova impostazione si applica ai controlli sui salvataggi successivi.
Non si modificano automaticamente dati storici diventati temporaneamente futuri
rispetto al nuovo «oggi»; un loro successivo salvataggio segue EFF-020.

### EFF-036 — Aggregazione delle sotto-issue

I totali comprendono l'issue e tutti i suoi discendenti, a qualunque profondità,
conteggiando ciascuna issue una sola volta. Sono calcolati dalle registrazioni e
dai valori propri, non sommando totali già aggregati. Gli aggregati sono in sola
lettura; per correggerli si interviene sui valori propri delle issue interessate.

Per stime mancanti si mostra il subtotale delle stime note con indicazione di
stima parziale; se mancano tutte, si mostra «Non stimato». I residui assenti
contribuiscono zero al totale, con indicazione di residuo parziale. Si considerano
solo le issue accessibili al lettore, senza rivelare numero o dettagli di quelle
escluse; la dicitura generale è «Totale delle issue visibili».

Con Efforts attivo, chiudere una parent issue azzera solo il suo residuo proprio. Il totale aggregato
può rimanere positivo per sotto-issue aperte. Se le automazioni già presenti in
Kuayle chiudono anche le sotto-issue, ciascuna segue la stessa regola di azzeramento.

### EFF-037 — Identità, delega e cambi di ruolo

Per inserire una nuova registrazione, il titolare del lavoro deve essere un utente
non guest con accesso all'issue. Owner/admin possono inserire per conto di owner,
admin o membri, non di guest.
Il titolare rimane distinto dal creatore e dagli autori delle successive modifiche.
Per correggere un'attribuzione errata si elimina il worklog e lo si ricrea per la
persona corretta; non si riassegna silenziosamente una registrazione esistente.

I permessi sono valutati al momento dell'operazione: diventando guest si perdono
anche le scritture sulle proprie vecchie registrazioni. La rimozione dal workspace
o la perdita di accesso all'issue non cancella il lavoro storico, ma revoca l'accesso
e le modifiche. I totali restano invariati; gli utenti autorizzati vedono il titolare
storico oppure «Utente non più disponibile» se l'account è stato eliminato.
Owner e admin con accesso all'issue mantengono la possibilità di correggere i
worklog storici di una persona successivamente rimossa, diventata guest o priva
di accesso all'issue. Queste correzioni non costituiscono nuove registrazioni
per conto di quella persona e restano soggette alle validazioni sui dati.

### EFF-038 — Ciclo di vita delle issue

Duplicare un'issue copia la stima, ma non worklog e storico Efforts. Il residuo
della copia parte dalla stima se presente, altrimenti rimane assente; per una copia
già conclusa/annullata è zero. Se la duplicazione include sotto-issue, la regola si
applica a ciascuna copia. A funzione disabilitata non si copiano dati Efforts.

Spostamenti fra team/progetti e la conversione in progetto conservano gli effort
sulle issue originali, applicando subito le nuove regole di accesso. I cambi di
gerarchia aggiornano i totali delle parent interessate. La conversione attuale
conserva le issue e le associa al progetto: non serve introdurre un effort proprio
del progetto. La prima versione non aggiunge effort ai template di issue.

L'eliminazione di un'issue elimina anche i suoi worklog e lo storico Efforts; la
conferma di eliminazione lo dichiara, anche se Efforts è disabilitato. Le sotto-issue
che il comportamento attuale conserva mantengono i propri effort. La rimozione di
un membro, invece, conserva i worklog come previsto in EFF-037. L'eliminazione di un
intero workspace include i dati Efforts, coerentemente con gli altri dati.

### EFF-039 — API, token, export e import

Le operazioni Efforts sono disponibili nelle API applicative e applicano gli stessi
permessi e validazioni dell'interfaccia. I personal access token richiedono scope
espliciti per Efforts, sempre limitati dai permessi e dall'accesso alle issue del
proprietario. I token esistenti non acquisiscono automaticamente nuovi scope.
Attivazione e fuso restano impostazioni amministrative gestite con sessione
interattiva di owner/admin.

Export/import del workspace comprendono configurazione, fuso, stime, distinzione
fra residuo assente e zero, worklog e storico, preservando autori del lavoro e
operatori mediante rimappatura degli identificativi. L'import conserva un'eventuale
disattivazione salvata. Archivi precedenti senza Efforts usano i nuovi default;
worklog importati incoerenti con durate o limite giornaliero non vengono troncati
silenziosamente, ma segnalati e rifiutati. Queste operazioni continuano a rispettare
i vincoli già presenti sui workspace con team privati. Non si aggiunge un export
timesheet/CSV dedicato nella prima versione.

### EFF-040 — Coerenza, concorrenza e aggiornamenti

Worklog, totale, eventuale variazione del residuo e storico devono essere salvati
insieme: un errore non lascia scritture parziali. Il limite giornaliero deve essere
rispettato anche da due sessioni o due operatori simultanei. Ripetere lo stesso
tentativo di invio per un errore di rete non deve duplicare il lavoro; due
registrazioni intenzionali con gli stessi dati restano possibili.

In caso di modifica concorrente a stima, residuo o worklog, non si sovrascrive
silenziosamente l'ultima modifica altrui: si segnala il conflitto mantenendo i dati
inseriti dall'utente. Disattivazione o cambio dei permessi durante un modulo aperto
sono ricontrollati al salvataggio.

Le altre sessioni autorizzate aggiornano riepilogo, registro, storico e aggregati
delle parent tramite il meccanismo realtime esistente. Gli eventi non espongono
dati di issue non accessibili. Non si introducono email o notifiche inbox specifiche
per ogni worklog; lo storico resta la traccia completa delle operazioni.

### EFF-041 — Dettagli dell'interfaccia

Il riepilogo laterale mostra stima, consuntivo e residuo propri; quando presenti
sotto-issue mostra separatamente i totali aggregati. Il consuntivo e gli aggregati
non sono modificabili direttamente. I controlli per stima/residuo rispettano ruolo,
assegnazione e stato dell'issue.

La sezione nel corpo contiene «Registra lavoro» e l'elenco con data, titolare,
durata, descrizione ed eventuale indicazione «inserito da» per le deleghe. Ordine:
data del lavoro decrescente, poi momento di inserimento decrescente; caricamento
progressivo o paginato da 50 righe. La prima versione non aggiunge filtri o ricerche
trasversali tra issue.

Il modulo propone oggi nel fuso del workspace e l'utente corrente come titolare;
owner/admin possono selezionare un'altra persona ammissibile. Gli errori restano
accanto ai campi pertinenti senza perdere quanto digitato. Eliminare un worklog
richiede conferma. Lo stato di sola lettura è esplicito. Si riutilizzano componenti,
scorciatoie accessibili e traduzioni esistenti di Kuayle, anche nel dettaglio aperto
in pannello oltre che nella pagina completa.

### Criteri di accettazione delle regole di completamento

Questi criteri continuano la numerazione AC-001–AC-042 e sono parte della stessa
base di accettazione.

| ID | Requisito | Verifica |
| --- | --- | --- |
| AC-043 | EFF-033 | Disabilitare con residuo 5 ore, chiudere e riattivare conserva prima 5 ore in sola lettura e poi azzera con storico; un'issue rimasta aperta conserva il proprio residuo. |
| AC-044 | EFF-034 | Stima zero valida; rimozione stima non cambia worklog/residuo; descrizione oltre 2.000 caratteri e durate con frazioni di minuto sono rifiutate senza troncamenti. |
| AC-045 | EFF-035 | Cambiare fuso lascia invariate le date e i totali storici; le nuove registrazioni usano il nuovo «oggi». |
| AC-046 | EFF-036 | Un albero su tre livelli conta ogni issue una sola volta; spostare una sotto-issue cambia i due aggregati, non i valori propri; issue non accessibili non contribuiscono al risultato restituito. |
| AC-047 | EFF-037 | Un titolare diventato guest non può modificare un vecchio worklog; la sua rimozione dal workspace non cambia i totali; un admin non può creare nuovo lavoro per un guest, ma può correggerne il lavoro storico se ha accesso all'issue. |
| AC-048 | EFF-038 | Duplicazione copia solo la stima e inizializza il residuo secondo lo stato; conversione conserva effort sulle issue; eliminare la parent rimuove i suoi dati ma non gli effort delle sotto-issue conservate. |
| AC-049 | EFF-039 | Un export/import conserva worklog, storico, fuso e stato disabilitato; un token privo di scope Efforts non ottiene i relativi dati tramite un endpoint alternativo. |
| AC-050 | EFF-040 | Invio ritentato non duplica il worklog; conflitto concorrente è segnalato; errore di salvataggio non produce storico o variazioni parziali; altre sessioni autorizzate si aggiornano. |
| AC-051 | EFF-041 | Dopo 50 registrazioni sono consultabili anche le successive; controlli e errori sono utilizzabili da tastiera; in sola lettura nessun comando Efforts consente modifiche dirette. |

## Contesto verificato nel repository

Verifica effettuata sulla base `176dfa1`, senza modificare il codice applicativo.

- Il modello attuale di issue contiene workspace, team, progetto opzionale,
  assegnazione e collegamento alla parent issue; non contiene campi effort o
  worklog: [modello issue](../../BE/internal/domain/issue.go).
- La migrazione 17 ha rimosso `issues.estimate` e `teams.estimate_scale`:
  [migrazione](../../BE/migrations/000017_remove_estimates.up.sql).
  Questo è un dato storico del progetto; la motivazione non è stata accertata.
- Esistono ruoli workspace `owner`, `admin`, `member` e `guest`:
  [modello workspace](../../BE/internal/domain/workspace.go).
- Le issue appartengono a team, anche privati, e possono avere più assegnatari.
  I requisiti effort devono definire come rispettare le autorizzazioni esistenti.
- Le API distinguono permessi applicativi, scope dei token e operazioni riservate
  alla sessione interattiva: [route](../../BE/cmd/server/routes.go).

Questi elementi orientano l'analisi; non stabiliscono nuove regole funzionali.

## Riferimenti Tempo

Documentazione Cloud consultata il 2026-10-16. Il riferimento Cloud è una base di
confronto; l'utente non ha richiesto parità con una specifica edizione di Tempo.

- [The Log Time Form](https://help.tempo.io/timesheets/latest/the-log-time-form):
  registrazione associata a una issue, con data e durata; possibilità di includere
  descrizione, orari e informazioni aggiuntive.
- [Overview of the Timesheet](https://help.tempo.io/timesheets/latest/overview-of-the-timesheet):
  consultazione del lavoro registrato per periodo e raggruppamento.
- [Timesheet Approvals](https://help.tempo.io/timesheets/latest/timesheet-approvals):
  approvazioni opzionali e gestione dei periodi, fuori dal perimetro iniziale scelto.

## Registro della sessione

### Round 1 — perimetro (confermato)

1. Contenuto effort: tempo lavorato, stima iniziale e residuo.
2. Attivazione: per workspace, su tutte le sue issue.
3. Prima versione: registrazioni e totale sulla singola issue; nessun timesheet o
   report aggregato.

### Round 2 — regole di utilizzo (confermato)

1. Residuo: riduzione automatica con possibilità di correzione manuale.
   Esempio: stima/residuo di 8 ore, registrazione di 2 ore, nuovo residuo di 6 ore.
2. Inserimento manuale di data, durata e descrizione.
3. Disattivazione: conservare i dati e mostrarli in sola lettura.

### Round 3 — permessi sulle registrazioni (confermato)

1. Owner, admin e membri possono registrare lavoro proprio anche senza essere
   assegnatari, purché abbiano accesso all'issue.
2. L'autore può correggere/eliminare le proprie registrazioni; owner e admin
   possono intervenire su tutte le registrazioni delle issue accessibili.
3. Chiunque abbia accesso all'issue può leggere i dettagli delle registrazioni.

Le scritture restano subordinate allo stato abilitato della funzionalità.

### Round 4 — attivazione e stime (confermato)

1. Efforts attivo per default nei workspace nuovi ed esistenti; gestione
   dell'attivazione riservata a owner e admin.
2. Stima iniziale e residuo modificabili manualmente dagli assegnatari e da owner
   e admin; guest esclusi.
3. Stima iniziale facoltativa e modificabile anche dopo aver registrato lavoro.

### Round 5 — calcolo del residuo (confermato)

1. Registrando lavoro senza stima né residuo, mostrare residuo zero e permetterne
   la correzione manuale agli utenti autorizzati.
2. Quando si imposta o rivede la stima, inizializzare il residuo assente a stima
   meno consuntivo, con minimo zero; lasciare invariato un residuo già impostato.
3. Modificare o eliminare registrazioni cambia solo il consuntivo, non il residuo.

Il raccordo fra i punti 1 e 2 è stato chiarito nel round 6: dopo aver registrato
2 ore senza stima, impostare una stima di 8 ore produce residuo 6, se il residuo
non è stato impostato manualmente nel frattempo.

### Round 6 — inizializzazione e campi (confermato)

1. Lo zero mostrato quando manca la stima non costituisce un residuo impostato:
   dopo 2 ore registrate, una prima stima di 8 ore inizializza il residuo a 6 ore.
2. Stime, residui e lavoro sono espressi in ore/minuti con precisione al minuto;
   data e durata positiva sono obbligatorie, descrizione facoltativa.
3. Sono ammesse registrazioni e correzioni per oggi e qualsiasi data passata;
   le date future non sono ammesse.

### Round 7 — stato, sotto-issue e condivisione (confermato)

1. Conclusione/annullamento azzerano il residuo; registrazioni e correzioni restano
   consentite nel rispetto dei permessi.
2. L'issue principale mostra valori propri e, separatamente, totali comprensivi
   delle sotto-issue.
3. I link pubblici non mostrano alcun dato Efforts.

### Round 8 — riapertura, delega e storico (confermato)

1. Alla riapertura il residuo resta zero; gli utenti autorizzati possono impostarlo
   manualmente.
2. Owner e admin possono registrare per conto di altri membri con accesso
   all'issue, conservando chi ha effettuato l'inserimento.
3. Lo storico dell'attività deve essere completo per stime, residui e worklog,
   mostrando autore dell'operazione e valori precedenti e successivi.

### Round 9 — superamento del residuo, limiti e fuso orario (confermato)

1. Registrare l'intera durata anche oltre il residuo, portando quest'ultimo a zero.
2. Bloccare inserimenti e modifiche che superano 24 ore attribuite alla stessa
   persona nella stessa data, su tutte le issue del workspace.
3. Usare il fuso del workspace, configurabile da owner e admin.

### Round 10 — fuso iniziale, interfaccia e issue chiuse (confermato)

1. Fuso iniziale `Europe/Rome` per workspace nuovi ed esistenti.
2. Riepilogo nella barra laterale e sezione Efforts nel corpo dell'issue.
3. Il residuo rimane obbligatoriamente zero sulle issue concluse/annullate fino
   alla riapertura.

### Revisione finale — proposte di completamento (da raccogliere)

Conferma del raccordo fra disattivazione e azzeramento alla riattivazione (P-01)
e revisione delle proposte di dettaglio P-02–P-09. Nessuna proposta di questa
sezione è ancora trattata come decisione dell'utente.

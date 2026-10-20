# Efforts V1 — uso e nota di rilascio

Rilascio implementato il 2026-10-20. Copre EP-01–08, US-01–25.

## Impostazioni

Efforts è attivo per impostazione predefinita nei workspace nuovi ed esistenti,
con fuso `Europe/Rome`. Owner e admin possono cambiare attivazione e fuso nelle
Impostazioni generali, usando una sessione interattiva. I PAT non possono farlo.

Disabilitare conserva i dati in sola lettura e blocca i comandi diretti anche da
un modulo già aperto. Riattivare azzera il residuo delle issue attualmente chiuse
con una voce nell’attività; conserva quello delle issue aperte. Il cambio fuso
modifica soltanto la valutazione di «oggi», senza spostare il lavoro storico.

## Pianificazione e registro

Pagina completa e pannello mostrano la sezione richiudibile **Effort**. Stima e
residuo si modificano inline: Invio o uscita salva, Esc annulla. L’unità principale è il giorno da 8 ore: `1` o `1d` vale 480 minuti, `0,5d` vale
4 ore. Sono accettati anche `1d 2h`, `90m`, `2h` e `1h 30m`; le frazioni di giorno
sono valide solo se producono minuti interi, senza arrotondamenti.
Stima assente e zero sono distinti; residuo assente è visualizzato zero con
«Non impostato». Impostare una stima inizializza soltanto il residuo assente a
stima meno consuntivo, con minimo zero. Cambiare o rimuovere la stima non cambia
un residuo già impostato.

**Effort** mostra soltanto i valori propri della issue. La scheda separata
**Effort totale** compare quando almeno un discendente visibile ha lavoro
registrato e somma i valori propri e quelli dei discendenti accessibili a ogni
profondità. I totali sono in sola lettura e segnalano i dati mancanti; non indicano
quante issue siano escluse per privacy. Entrambe le schede mostrano tre barre:
stima nel colore principale del tema, consuntivo verde e residuo ambra. La scala
comune di ciascuna scheda è il maggiore tra stima e consuntivo più residuo;
stime assenti e valori zero mantengono la barra vuota. Da disabilitato,
il riepilogo senza dati propri o aggregati viene nascosto.

**WORK LOG** è sotto **ACTIVITY**, chiuso per impostazione predefinita ed
espandibile nello stesso stile. Raccoglie
le operazioni Efforts prima mostrate in Activity: stima, registrazioni e sole
correzioni manuali del residuo. Sono escluse le variazioni automatiche dovute a
stima, worklog, chiusura o riattivazione; restano conservate nello storico server.
All’apertura mostra soltanto l’evento più recente; il comando per mostrare gli
eventi precedenti è sopra la riga. «Mostra meno» ripristina il singolo evento.
La sottosezione **Registrazioni / Entries** è rimossa. Dettagli prima/dopo,
titolare, data, durata, descrizione e «inserito da» restano negli eventi.
Modifica ed eliminazione sono disponibili sull’ultimo evento di ogni worklog
ancora esistente, secondo i permessi correnti. Prima di aprire il dialogo si
rilegge il worklog attuale, anche oltre le prime 50 registrazioni.

«Registra lavoro» propone la data corrente riletta dal server nel fuso workspace.
I campi **Dal/Al** includono entrambi gli estremi e applicano lo stesso effort
per ciascuna data selezionata. Sabato e domenica sono esclusi per impostazione
predefinita; deselezionando l’opzione vengono inclusi. Le festività non sono
escluse automaticamente. L’anteprima mostra il numero di giorni e l’effort totale.
Con Dal uguale ad Al si registra un solo giorno. L’intero periodo è salvato in
un’unica transazione: una data che supera il limite annulla tutte le registrazioni,
lo storico e la variazione del residuo. Un retry ripete lo stesso tentativo
senza duplicare il periodo. La durata per giorno è da 1 a 1.440 minuti; il totale giornaliero della persona nel workspace
non può superare 24 ore. La descrizione è facoltativa, multilinea, fino a 2.000
caratteri Unicode. Sono ammesse date passate e oggi, anche su issue chiuse, senza
stima o assegnazione. Un nuovo worklog riduce il residuo impostato fino a zero;
correggere o eliminare un worklog cambia il consuntivo senza ricalcolare il residuo.
L’eliminazione richiede conferma.

## Permessi e affidabilità

| Ruolo con accesso all’issue | Operazioni |
| --- | --- |
| Owner/admin | Pianificazione; lavoro proprio o delegato a membri attualmente ammissibili; correzione/eliminazione del lavoro accessibile, anche storico. |
| Member | Lavoro proprio; stima/residuo soltanto se assegnatario. |
| Guest | Lettura. |

Il titolare di una registrazione resta fisso. Rimuovere una persona dal workspace
non elimina il lavoro. Un account eliminato appare come utente non disponibile.
Chiudere/annullare azzera il residuo con storico; riaprire non lo ricostruisce.
Duplicare copia soltanto la stima e inizializza il residuo (zero se chiusa), senza
worklog o storico copiati; da disabilitato non copia Efforts. Eliminare una issue
rimuove i suoi worklog e storico, mantenendo quelli dei figli conservati.

WORK LOG mostra valori prima/dopo, operatore e causa delle operazioni Efforts
visibili, incluso il lavoro eliminato; ACTIVITY contiene le altre modifiche issue. Le altre sessioni aggiornano riepilogo,
registro, attività e totali; alla riconnessione rileggono i dati autorizzati.
I refresh non cambiano la revisione di una bozza aperta. Dopo un errore di rete
il retry conserva chiave e payload; un conflitto richiede riallineamento esplicito.

## API e trasferimento

I PAT richiedono `efforts:read` per leggere e anche `efforts:write` per scrivere.
Gli scope non aumentano ruolo o accesso del proprietario. Anche duplicazione,
eliminazione e cambi di stato richiedono gli scope Efforts previsti nei
[contratti API](efforts-contracts.md#5-scope-e-canali-indiretti). I token esistenti
non ricevono automaticamente nuovi scope. Nessun dato Efforts è esposto nei
link pubblici.

L’export workspace produce archivi **v2** con configurazione, pianificazione,
worklog e storico coerenti. Il PAT deve avere `workspace:transfer` e
`efforts:read`. Rimane il blocco del trasferimento per workspace con team privati.
L’import accetta v2 e archivi v1 privi di Efforts (default attivo/Europe-Rome).
Conserva date, stato disabilitato e attribuzioni, senza ricreare accessi per utenti
rimossi. Archivi incoerenti sono rifiutati senza import parziale.

Non sono inclusi timesheet, report team/progetto, export CSV, approvazioni,
notifiche dedicate o Efforts nei template. La V1 usa la migrazione 47 già presente;
questo completamento non aggiunge migrazioni. Le verifiche e le misure sono in
[efforts-acceptance.md](efforts-acceptance.md).

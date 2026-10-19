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
residuo si modificano inline: Invio o uscita salva, Esc annulla. Sono accettati
minuti interi, `90m`, `2h`, `1h 30m`; non sono ammessi arrotondamenti di frazioni.
Stima assente e zero sono distinti; residuo assente è visualizzato zero con
«Non impostato». Impostare una stima inizializza soltanto il residuo assente a
stima meno consuntivo, con minimo zero. Cambiare o rimuovere la stima non cambia
un residuo già impostato.

**Totale delle issue visibili** somma i valori propri e quelli dei discendenti
accessibili a ogni profondità. I totali sono in sola lettura e segnalano i dati
mancanti; non indicano quante issue siano escluse per privacy. Da disabilitato,
il riepilogo senza dati propri o aggregati viene nascosto.

**WORK LOG** è sotto **ACTIVITY**, espandibile e nello stesso stile. Il comando
per mostrare le registrazioni precedenti è sopra i record. La navigazione usa
pagine da 50, ordinate per data lavoro e inserimento decrescenti. Le righe mostrano
titolare, durata, descrizione e «inserito da» nelle deleghe.

«Registra lavoro» propone la data corrente riletta dal server nel fuso workspace.
La durata è da 1 a 1.440 minuti; il totale giornaliero della persona nel workspace
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

Activity mostra valori prima/dopo, operatore e causa delle variazioni, inclusi
sistema/GitHub e lavoro eliminato. Le altre sessioni aggiornano riepilogo,
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

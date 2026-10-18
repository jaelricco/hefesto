# ADR 0027 — Der Start einer geplanten Einheit ohne Verbindung

- Status: proposed (zur Prüfung mit dem Offline-Start, 18.10.2026)
- Date: 2026-10-18
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Spec §10.5 sieht vor, dass der Client eine geplante Einheit auch ohne
Verbindung startet: nach demselben Schema wie der Server, mit neuen IDs und
`planned_item_id`. Der Sync lädt sie hoch, und der Server passt danach an.
Gebaut war das nicht:
- Die Sync-Operationen trugen `planned_session_id` und `planned_item_id`
  nicht (§15.3).
- Ohne Netz sagte das Check-in-Blatt nur, dass der Start eine Verbindung
  braucht (ADR 0022).

Wer in einem Keller oder einer Halle ohne Empfang trainiert, konnte seinen
Plan also nicht starten. Das freie Training ging, aber ohne Ziele und ohne
Zuordnung zum Plan.

## Entscheidungen

### 1. Der Sync trägt die Verweise auf den Plan

- `SessionPut` hat das Feld `planned_session_id`, `SetEntryWrite` das Feld
  `planned_item_id`.
- Der Server übernimmt beide nur beim Anlegen der Zeile. Danach gehören sie
  ihm: Der Abgleich (ADR 0017) verlinkt Sätze auf neue Items, und ein
  späterer Push des Clients überschreibt das nicht.
- Eine geplante Einheit, die der Server für diesen User nicht kennt, wird
  verworfen (`NULL`). Die Session bleibt dann eine Session des Logs. Ein
  Verweis auf den Plan darf das Log nie blockieren (ADR 0009).

### 2. Die App schreibt den Entwurf wie der Server

Die App baut den Entwurf aus dem gehaltenen Plan nach `Materialize`
(`internal/domain/planning/materialize.go`), Schritt für Schritt nachgebaut:
- Blöcke ohne geplante Sätze und Angebote fehlen.
- Ein Paar-Block mit mehr als einem Item wird ein Supersatz, Runde für Runde.
- Jeder Satz trägt Messart, Ziel, Pause, Zusatzlast und «exzentrisch».
- Die Ziel-Reserve ist gekappt wie im Log (RIR bis 10, SIR bis 60 s).

Der Planer ordnet Sätze ihren Items zu und zählt sie. Beide Seiten müssen
deshalb dieselben Blöcke, dieselbe Zahl Sätze je Item und dieselbe
Reihenfolge erzeugen. Ein Test in HefestoKit hält das Schema fest.

Session, Blöcke und Sätze gehen in einer Transaktion in den Ausgang, die
Eltern zuerst. Ein zweiter Start derselben Einheit auf diesem Gerät öffnet die
erste Session. Fehlt eine Übung im lokalen Katalog, startet nichts.

**Ohne Check-in.** Das Leichter-Machen nach kurzem Schlaf oder grosser
Müdigkeit (ADAPT-17, ADR 0019) rechnet der Server. Die App bildet es nicht
nach, denn dafür bräuchte sie die Wissensbasis. Offline startet die Einheit
deshalb wie geplant.

### 3. Der Server verknüpft nach dem Commit

Der Planer sucht Sessions, die eine geplante Einheit nennen, aber von keiner
geplanten Einheit verknüpft sind. Er tut das zu vier Zeitpunkten:
- nach jedem Push mit mindestens einer angewandten Operation, damit der Pull
  danach schon den angepassten Entwurf bringt;
- vor jedem Lesen des Plans;
- vor jedem Start;
- vor jeder Auswertung eines Abschlusses.

Für jede solche Session gilt:
- **Verknüpft wird** die Einheit desselben Tages im aktuellen Plan der Woche,
  wie beim Übertrag auf einen neuen Plan (ADR 0017). Ihr Status wird
  `started`, oder `completed`, wenn die Session schon abgeschlossen ist.
  `check_in_applied` bleibt falsch.
- **Ein Entwurf aus einem ersetzten Plan** folgt dem aktuellen, wie jeder
  gestartete Entwurf (ADR 0017).
- **Ohne Einheit an dem Tag**, etwa nach einem Stopp, gehen seine offenen
  Sätze.
- **Wurde der Tag anderswo zuerst gestartet**, behält die geplante Einheit
  jene Session. Die Offline-Session bleibt eine Session des Logs, mit ihren
  Zielen.
- Verknüpft wird nur, was ab dem Montag der Vorwoche geplant war. Jede
  Verknüpfung läuft in einer eigenen Transaktion unter der Planer-Sperre. Ein
  Fehler wird geloggt und beim nächsten Anlass wiederholt; er blockiert
  weder den Push noch den Plan.

Ein Start auf einem anderen Gerät antwortet danach mit der Offline-Session,
wie jeder zweite Start (U-51).

### 4. Die App zeigt den lokalen Start sofort

Was die App selbst gestartet oder abgeschlossen hat, zeigt «Heute», bevor der
gehaltene Plan es kennt:
- Ein Offline-Start erscheint als begonnene Einheit, «Weiter» öffnet sie.
- Ein Abschluss, der noch nicht synchronisiert ist, erscheint als «Für heute
  erledigt».

Nennt der Plan für die Einheit eine andere Session, gilt der Plan.

### 5. Das Blatt bietet den Start ohne Verbindung an

Erreicht der Start den Server nicht, sagt das Check-in-Blatt:
- dass die Einheit ohne Verbindung wie geplant startet;
- dass das Leichter-Machen den Server braucht;
- dass der Plan die Einheit beim nächsten Sync übernimmt.

Es bietet «Ohne Verbindung starten» und «Nochmals versuchen» an. Die Wahl
bleibt beim Athleten: Wer müde ist, kann auch frei und leichter trainieren.

«Weiter» bei einer Einheit, deren Session nicht auf dem Gerät ist, braucht
weiter eine Verbindung und sagt das.

### 6. Demo

Die Demo hat keinen Server. Ihr Start führt deshalb zum Angebot ohne
Verbindung. Der neue Screen `offlinestart` zeigt das Blatt in diesem Zustand.

## Folgen

- Eine geplante Einheit startet überall. Was ohne Netz geschieht, landet mit
  dem nächsten Sync im Plan, und der Planer passt danach an wie nach jedem
  Start.
- Der Port von `Materialize` muss mit dem Server Schritt halten. Eine
  Änderung auf einer Seite braucht die andere und den Test in HefestoKit.
- **Nicht gebaut:**
  - der Check-in ohne Verbindung;
  - der Offline-Start einer Einheit, die nicht von heute ist. Die Screens
    starten nur heute (ADR 0022).
- **Befunde für den Review:**
  - **Ein Ausgang über mehrere Pushes.** Die App sendet höchstens 200 Ops
    je Push. Liegt die Grenze mitten in einem Offline-Start und wurde der Plan
    seither ersetzt, kann der Server den Entwurf verknüpfen, bevor alle Sätze
    da sind. Fehlende Sätze zählen dann als erledigt. Nachkommende Sätze
    behalten ihr altes Item und folgen späteren Plänen nicht. Doppelte Sätze
    entstehen nicht.
  - **Zwei Geräte starten dieselbe Einheit offline.** Die erste Verknüpfung
    gewinnt. Die zweite Session bleibt mit offenen Zielen im Log, bis der
    Athlet sie löscht oder abschliesst. Spec §10.2 hatte für diesen Fall eine
    Zuordnung angenommen.
  - **Offline gibt es keinen Check-in.** Ein müder Athlet bekommt ohne Netz
    die volle Einheit, sofern er sie wählt. Das Blatt sagt es ihm.

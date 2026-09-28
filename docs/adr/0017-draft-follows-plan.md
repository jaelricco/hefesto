# ADR 0017 — Ein gestarteter Entwurf folgt dem Plan

- Status: proposed (zur Prüfung mit dem Abgleich, 28.09.2026)
- Date: 2026-09-28
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

ADR 0016 schreibt eine geplante Einheit beim Start als Entwurf ins Log.
Danach änderte der Server den Entwurf nicht mehr. Jedes Ereignis erzeugt aber
einen neuen Plan: ein Schmerzbericht, Symptome, Red-Flag-Antworten, ein
Widerruf der Einwilligung, eine Profil- oder Zieländerung. Der neue Plan
schliesst dann vielleicht eine Region aus, senkt eine Dosis oder stoppt das
Training. Der Entwurf behielt trotzdem seine offenen geplanten Sätze. ADR 0016
hat das als Sicherheitsbefund festgehalten und vorgeschlagen: Der Server
ersetzt die offenen geplanten Sätze, bei einem Stopp löscht er sie, und
ausgeführte Sätze bleiben. Der Vorschlag ist angenommen.

Beim Umsetzen galten zwei Regeln aus ADR 0009, dem Sync:
- «Der Server ändert nie die Werte eines Elements.» Ein Ziel kann also nicht
  an Ort und Stelle sinken; ein Satz mit neuem Ziel ist ein neuer Satz.
- «Löschungen sind endgültig.» Ein Satz, den der Server löscht, kann ein
  Client später nicht mehr schreiben. Hat der User ihn offline schon
  ausgeführt, geht die Ausführung verloren.

Dazu kamen drei Fragen:
- Was zählt als erledigt?
- Was geschieht mit Zielen, die der User selbst angepasst hat?
- Wie wird der User informiert?

## Entscheidungen

### 1. Abgeglichen wird bei jedem neuen Plan der Woche

Der Abgleich läuft in derselben Transaktion wie der neue Plan. Er betrifft
jede Einheit des ersetzten Plans, die gestartet ist und deren Session im Log
noch ein Entwurf ist. Eine abgeschlossene, abgebrochene oder gelöschte
Session folgt dem Plan nicht.

### 2. Verglichen werden Plan-Items, nicht Log-Werte

Die reine Funktion `Reconcile` vergleicht die geplante Einheit, aus der der
Entwurf zuletzt entstand, mit ihrer Nachfolgerin im neuen Plan. Items werden
über Block-Rolle, Übung und Art einander zugeordnet, in ihrer Reihenfolge.

Die offenen geplanten Sätze (`is_planned`, ohne `completed_at`) folgen so:
- **Item unverändert:** Die Sätze bleiben, samt den Zielen, die der User
  vielleicht selbst angepasst hat. Nur `planned_item_id` zeigt danach auf das
  neue Item. `updated_at` bleibt, denn das ist die Uhr des Users.
- **Ziel geändert:** Jeder offene Satz wird ersetzt. Der alte wird ein
  Grabstein, der neue nimmt seinen Platz im Block ein.
- **Weniger Sätze:** Die letzten offenen Sätze gehen.
- **Mehr Sätze:** Sie kommen nach den offenen Sätzen des Items in denselben
  Block.
- **Item entfallen:** Seine offenen Sätze gehen.
- **Item neu:** Es kommt in einen neuen Block am Ende der Session.
- **Keine Einheit mehr an dem Tag:** Das gilt etwa nach einem Stopp
  (SAFE-02) oder wenn der Tag ein Ruhetag wird. Dann gehen alle offenen
  geplanten Sätze, auch die, die der User selbst als geplant angelegt hat.

Ausgeführte Sätze ändern sich nie. Ein Block, in dem kein Satz mehr steht,
wird ein Grabstein.

### 3. Erledigt ist, was nicht mehr offen ist

Für ein Item gilt: erledigt = geplante Sätze des alten Items − noch offene
Sätze. Das umfasst ausgeführte Sätze und Sätze, die der User gelöscht hat;
beides kommt nicht zurück. Der neue Plan bekommt so viele offene Sätze, wie
seine Sätze die erledigten übersteigen. Sätze, die der User selbst angelegt
hat, zählen nicht; sie haben kein Item.

### 4. Die Antwort sagt es: `session_adjusted` (ADAPT-19)

Sieht der User eine Änderung, zeichnet das Ereignis die Änderung
`session_adjusted` mit der Log-Session auf (`session_id`). Grund ist die neue
Regel ADAPT-19 («Deine laufende Einheit folgt dem geänderten Plan: offene
Sätze sind angepasst, erledigte bleiben.», Projektvorgabe). Ein nur neu
verknüpfter Satz gilt nicht als Änderung. Die Änderung steht im
Änderungsprotokoll, eine Wiederholung des Ereignisses antwortet also gleich.
Profil- und Zieländerungen und die Neuerzeugung haben keine Ereignis-Antwort;
dort zeigt der Plan den Start, und der Sync bringt die Sätze.

### 5. Der Server schreibt ohne Gerät

Grabsteine und neue Zeilen tragen `client_id = NULL` und als `updated_at` die
Zeit des Servers. So erreicht der Sync jedes Gerät.

## Folgen

- Der Sicherheitsbefund aus ADR 0016 ist behoben. Nach einem Stopp stehen
  keine offenen geplanten Sätze mehr im Entwurf. Nach einem Ausschluss oder
  einer gesenkten Dosis stehen dort die Sätze des neuen Plans.
- **Restrisiko (Review):** Löschungen sind endgültig (ADR 0009).
  - Führt der User einen Satz offline aus, den der Server inzwischen
    ersetzt oder gelöscht hat, lehnt der Sync die Ausführung ab
    (`superseded`).
  - Betroffen sind nur Sätze, deren Item sich geändert hat oder entfallen
    ist. Unveränderte Sätze bleiben dieselben Zeilen.
  - Die App sollte ihren Ausgang synchronisieren, bevor sie ein Ereignis
    sendet.
  - Wenn das nicht reicht: Ein ausgeführter Satz dürfte einen Grabstein des
    Servers wiederbeleben, wenn er vor der Löschung ausgeführt wurde. Das
    ändert ADR 0009 und braucht eine eigene Entscheidung.
- Nach einem Stopp hat der Plan keine Einheiten. Der Start wird deshalb nicht
  weitergetragen. Nach der Freigabe kann die Einheit des Tages neu starten;
  der alte Entwurf behält seine ausgeführten Sätze.
- Ein Satz, den der User selbst als geplant angelegt hat, bleibt bei jeder
  Änderung ausser «keine Einheit mehr»; der Server kennt sein Ziel nicht.
- Profil- und Zieländerungen stehen weiter nicht im Änderungsprotokoll,
  also auch nicht ihre Abgleiche.

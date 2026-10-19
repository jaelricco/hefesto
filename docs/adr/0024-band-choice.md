# ADR 0024 — Die Wahl des Bands in der iOS-App

- Status: proposed (zur Prüfung mit der Wahl des Bands, 18.10.2026)
- Date: 2026-10-18
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Das Log verlangt für eine Band-Unterstützung das Band selbst: `band_id` und
`band_count` (`set_element_assistance_band_ck`). Der Server hat dafür alles:
- `GET /v1/bands` listet den Katalog und die eigenen Bänder;
- `POST /v1/me/bands` und `DELETE /v1/me/bands/{id}` legen eigene Bänder an
  und entfernen sie;
- der Planer zählt Band-Sätze als Band-Kapazität ihrer Übung (ADR 0018).

Die App konnte kein Band wählen. Deshalb bot der Eingabe-Screen keine
Band-Unterstützung an. Ein geplanter Band-Satz liess sich nicht ausführen
(ADR 0022, Entscheidung 6), denn ohne Band hätte er als unassistiert
gegolten.

Der globale Katalog ist leer, bis echte Herstellerangaben recherchiert sind
(`content/bands/README.md`). Fast jedes Band ist also ein eigenes.

## Entscheidungen

### 1. Die App hält die Bänder des Servers

Eine lokale Tabelle `band` (Migration v5) hält die Liste von `GET /v1/bands`.
- Bänder haben keine Sync-Spalten (ADR 0009). Die App ruft die Liste nach
  jedem Sync ab, wie die Übungen.
- Ein Band, das der Server nicht mehr listet, bleibt als «entfernt» stehen.
  Sätze, die es nennen, zeigen weiter seinen Namen; zur Wahl steht es nicht
  mehr.

### 2. Bänder entstehen online

«Band hinzufügen» fragt Marke, Farbe oder Name und den Widerstand laut
Hersteller (von–bis, kg) und legt das Band mit `POST /v1/me/bands` an.
- Ohne Netz sagt die App, dass Bänder online gespeichert werden. Die Lücke
  «Bänder offline» aus ADR 0009 bleibt offen.
- Ein eigenes Band lässt sich in der Liste wegwischen. Der Server löscht es
  nur weich (`deleted_at`); Sätze, die es nennen, behalten es.
- Die App sagt dazu, dass Hersteller oft mehr angeben, als ein Band hergibt,
  und dass Bänder derselben Farbe streuen (`onboarding.md` §3.8, C-79, C-80).

### 3. Der Eingabe-Screen fragt nach dem Band

«Band» steht wieder unter den Unterstützungen. Bei «Band» erscheinen:
- **das Band**, gewählt aus den eigenen Bändern, dann dem Katalog;
- **«Bänder gleichzeitig»**, 1 bis 4.

Ohne gewähltes Band lässt sich der Satz nicht sichern. Die geschätzte Hilfe
in kg bleibt freiwillig wie bei jeder Unterstützung. Der Planer rechnet nicht
mit ihr.

Ein wiederholter Satz übernimmt Band und Anzahl, wie jede Unterstützung.

### 4. Geplante Band-Sätze werden ausführbar

Entscheidung 6 von ADR 0022 fällt:
- Ein geplanter Satz, dessen Item `assist: band` trägt, öffnet mit «Band» als
  Unterstützung.
- Der Eingabe-Screen sagt: «Mit Band geplant. Wähl das Band, das du nutzt,
  oder «Keine», wenn du ohne trainiert hast.»
- «Nächster Satz» überspringt Band-Sätze nicht mehr.

Ein Band-Satz ohne Band ist damit nicht mehr möglich. Ein Satz ohne Band ist
ehrlich als unassistiert eingetragen.

### 5. Sätze nennen ihr Band

Logger und Verlauf zeigen bei einem Band-Satz «Band: Marke Farbe», bei
mehreren Bändern mit «×2». Ein Band, das die App nicht kennt, heisst nur
«Band».

### 6. Demo

Die Demo hält zwei eigene Bänder als Testdaten, ohne echte Herstellerangaben.
Der neue Screen `band` zeigt einen Klimmzug mit Band im Eingabe-Screen.

## Folgen

- Band-Sätze lassen sich ehrlich eintragen, frei und geplant. Der Planer
  bekommt sie als Band-Kapazität (ADR 0018).
- **Nicht gebaut:**
  - Bänder offline anlegen (ADR 0009);
  - ein Band bearbeiten; die API kennt nur Anlegen und Entfernen;
  - die Verankerung (`anchor`).
- **Befund für den Review:** Der Planer unterscheidet Bänder nicht. Ein
  Klimmzug mit leichtem und einer mit starkem Band zählen beide als
  Band-Kapazität derselben Übung. Wer das Band wechselt, verschiebt die
  Schätzung, ohne dass der Planer es merkt. Abhilfe bräuchte eine Kapazität je
  Band oder die geschätzte Hilfe in kg; beides ist eine fachliche Entscheidung.

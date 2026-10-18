# ADR 0019 — Check-in beim Start einer Einheit

- Status: proposed (zur Prüfung mit dem Check-in, 28.09.2026)
- Date: 2026-09-28
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

`spec.md` §6.12 (ADAPT-17) sieht einen optionalen Check-in beim Start vor. Der
User gibt Schlaf (Stunden) und Tagesform (1–10) an (ENT-7). Bei Schlaf ≤ 6 h
(PAR-E-41) oder Ermüdung ≥ 8 (`PAR-S-28`) werden die Maximalblöcke dieser
Einheit Technik (PAR-E-43, PAR-E-19); Kraftübungen bleiben (PAR-E-42). Der
Check-in hat keine Folgen für Streak, XP oder Fortschritt und wird ohne
Einwilligung nicht gespeichert.

Offen war:
- Was genau heisst «Maximalblöcke werden Technik»?
- In welche Richtung läuft die Skala 1–10?
- Was bleibt gespeichert?
- Wie verträgt sich der Check-in mit dem Abgleich (ADR 0017)? Jedes spätere
  Ereignis erzeugt einen neuen Plan. Ohne Vorkehrung ersetzte der Abgleich die
  Technik-Sätze wieder durch die Maximalsätze des Plans.

## Entscheidungen

### 1. Eingaben und Skala

`POST /v1/me/plan/sessions/{id}/start` nimmt optional `check_in` mit
`sleep_hours` (0–24, Schlaf in den letzten 24 Stunden) und `fatigue` an.
`fatigue` hat die Skala von `perceived_fatigue`: 1 frisch, 10 erschöpft.
`PAR-S-28` ist auf dieser Skala definiert, als Ermüdung ≥ 8 wie PAR-B-49 b.
Müde ist, wer höchstens 6 h geschlafen hat oder Ermüdung ≥ 8 angibt.

### 2. Was leichter wird

Nur der Block `skill_max` ändert sich:
- **Angebote gehen:** Prüfversuche und erste Versuche sind Maximalversuche.
- **Jeder Halt wird Technik, ob Skill- oder Konditionshalt.** Der
  Maximalblock einer Ziel-Leiter enthält auch Konditionshalte, etwa eine
  Planche Lean unter der Stufenschwelle. Die Dosis ist die Technik-Dosis des
  Plans (DOSE-09, PAR-S-26):
  - Dosiswert `d = Halt + Reserve`, also der Wert, mit dem der Plan dosiert
    hat;
  - Halt `min(0.5 · d, 10 s)`, drei Versuche, Reserve `⌊d − Halt⌋`;
  - Pause wie bei Balance und Technik;
  - bleibt keine Reserve, fällt der Halt weg.
- **Wiederholungsarbeit bleibt:** Kraft, Wiederholungs-Skills und Exzentrik.
  Oberkörperkraft war unter Schlafmangel unbeeinflusst (PAR-E-42), und
  Wiederholungs-Skills sind Skills mit gemischter Begrenzung
  (`limiting_factor: mixed`), bei denen Kraft mitbegrenzt.
- **Die Sprosse bleibt:** Technik auf leichten Tagen nimmt eine Sprosse
  tiefer, wenn sie machbar ist. Hier bleibt die geplante Sprosse, denn die
  Technik-Dosis ist bereits weit unter der Grenze. So braucht der Check-in
  keine neue Auswahl.
- **Aufwärmen, Balance, Volumen, Kraft und Ende bleiben**, auch die
  Aufwärm-Rampe vor dem Maximalblock.

Die Einheit trägt den Grund ADAPT-17, wenn sich etwas geändert hat.

### 3. Gespeichert wird nur die Entscheidung

Die Antworten gehen nicht über den Aufruf hinaus, mit oder ohne Einwilligung.
`planned_sessions.check_in_applied` hält nur fest, dass ein müder Check-in
die Einheit leichter gemacht hat. Das ist wie bei den Auflagen (ENT-S-7): Es
bleibt die Entscheidung, nicht die Antwort. Das Flag hängt am Start. Wird
der Entwurf gelöscht, gilt die Einheit wieder als geplant, und das Flag
fällt weg.

### 4. Plan, Entwurf und Abgleich stimmen überein

- Der Plan-Payload bleibt, wie der Kern ihn erzeugt hat.
- Die Plan-Ansichten zeigen eine Einheit mit `check_in_applied` so, wie ihr
  Entwurf sie hält. Das gilt für die Woche, eine frühere Woche, die
  Neuerzeugung und die einzelne Einheit.
- Ein neuer Plan der Woche übernimmt das Flag mit dem Start (U-54).
- Der Abgleich macht beide Seiten leichter, bevor er vergleicht. Die
  Technik-Sätze bleiben also, solange sich der Plan für das Item nicht
  ändert. Ändert er sich, folgen sie ihm in der leichteren Form.
- Das `ETag` des Plans nimmt das Flag auf.

### 5. Keine Folgen ausserhalb der Einheit

Der Check-in ändert weder Streak noch XP noch Fortschritt (ADR 0003). Er ist
kein Ereignis im Änderungsprotokoll, und der Planer lernt nichts aus ihm.

## Folgen

- Ein müder User bekommt eine leichtere Einheit ohne Maximalversuche. Kraft
  und Volumen bleiben.
- Der Check-in gilt nur beim ersten Start. Ein wiederholter Start antwortet
  mit dem bestehenden Entwurf und übergeht einen neuen Check-in.
- **Befunde für den Review:**
  - PAR-E-41 ist die Einschlussdefinition der Meta-Analyse, keine dort
    ermittelte Schwelle.
  - Die Übertragung von «Präzisionsaufgaben» auf Calisthenics-Skills ist eine
    Analogie (PAR-E-42).
  - Die Schwelle ≥ 8 übernimmt PAR-B-49 b (Deload), ist aber für den Tag
    einer einzelnen Einheit nicht untersucht.
- Die App sollte den Check-in freiwillig und ohne Druck fragen (ADR 0003).

# ADR 0030 — Die Fragen zum Schmerz vor und nach einer geplanten Einheit

- Status: proposed (zur Prüfung mit den Fragen, 19.10.2026)
- Date: 2026-10-19
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Spec §8.6 beobachtet den Schmerz der Regionen in der Rampe (`rtt_1`–`rtt_5`)
zu festen Zeitpunkten:

- vor der Einheit der aktuelle Wert (`before_session`);
- nach der Einheit «während» und «danach», mit «hielt länger als 1 h an»;
- am nächsten Morgen «heute früh» (`next_morning`).

Die Rampe rückt nur mit einem grünen Morgenbericht nach einer Einheit vor,
die die Region belastet (PAR-D-26). Ohne Bericht bleibt sie stehen
(`PAR-S-34`).

Bisher gab es nur den freien Schmerzbericht im Screen «Körper» und im Log
(ADR 0028). Die App wusste nicht, welche Regionen eine Einheit beobachtet.
Wer nicht von sich aus berichtete, blieb auf seiner Stufe. Ein Bericht ohne
Verbindung ging verloren.

## Entscheidungen

### 1. Der Plan nennt die Regionen

- Jede geplante Einheit nennt in `monitor` die Regionen in `rtt_1`–`rtt_5`.
- Regionen mit S-Zellen (§8.4) sind darin enthalten: Die Matrix wirkt nur in
  der Rampe.
- Der Plan nennt zu jeder beobachteten Region ihren Namen
  (`RegionMonitor.name`). So zeigen die Fragen ihn auch ohne Verbindung.
- Der Server sendet beide Felder immer. Im Schema sind sie optional: Ein
  Plan, den die App von vorher hält, bleibt lesbar.

### 2. Drei Momente

- **Vor der Einheit, im Check-in:** «Schmerz gerade» je Region, als
  `before_session`.
  - Der Bericht geht nach dem Start, mit der Log-Session als `session_id`.
    Erst der Start gibt sie.
  - Das gilt auch für den Start ohne Verbindung.
- **Nach der Einheit, im Abschluss:** «Schmerz während der Einheit» und
  «Schmerz gerade», als `during` und `after`.
  - Die App synchronisiert zuerst den Abschluss. So kennt der Planer die
    Sätze der Einheit, wenn er die Berichte auswertet.
  - Das Blatt nimmt dann den ganzen Bildschirm ein.
- **Am Morgen danach, auf «Heute»:** eine Karte zur letzten abgeschlossenen
  geplanten Einheit von gestern.
  - Sie fragt «Schmerz heute früh» (`next_morning`).
  - Sie fragt auch «Hielt der Schmerz nach der Einheit länger als eine Stunde
    an?».

Die Fragen nennen jede Region mit ihrem Namen. Die Skala 0–10 erklärt sich in
einem Satz.

### 3. «Länger als 1 h» am Morgen

Spec §8.6 und das Datenmodell (§4.9) fragen «hielt länger als 1 h an» mit dem
Bericht «danach». Das ist ein Widerspruch zur Praxis: Direkt nach der Einheit
ist die Stunde nicht vorbei, und die Antwort wäre geraten.

- Die App fragt es am Morgen danach und sendet es mit dem Bericht
  `next_morning`.
- Der Planer wertet das Flag für die Einheit, gleich an welchem ihrer
  Berichte es hängt.
- Server und Schema bleiben unverändert.

### 4. Freiwillig und ohne Strafe (ADR 0003)

- Jede Antwort ist optional. Eine Frage ohne Antwort sendet nichts.
- Die Morgenkarte hat «Nicht heute» und verschwindet mit ihrem Tag. Es gibt
  keine Erinnerung und keine Benachrichtigung.
- Ohne Bericht läuft der Plan weiter, nur die Rampe rückt nicht vor
  (`PAR-S-34`). Die Karte sagt das sachlich: Der Plan entscheidet mit der
  Antwort, ob die Last dort wachsen kann.
- Eine Region, die für eine Session und einen Zeitpunkt beantwortet ist,
  fragt die App nicht noch einmal.

### 5. Warnzeichen vor dem Log

Fragt der Planer nach dem Schmerz vor der Einheit Warnzeichen
(`ask_red_flags`), kommen sie, bevor sich das Log öffnet. Stoppen sie das
Training, öffnet es sich nicht; der Hinweis sagt, warum. Nach der Einheit
und am Morgen öffnen sich die Warnzeichen wie beim freien Bericht
(ADR 0028).

### 6. Berichte werden gehalten

Das gilt für jeden Schmerzbericht der App, auch für den freien aus ADR 0028.

- Die App hält jeden Bericht lokal (Tabelle `painReport`) und sendet ihn
  sofort.
- Was nicht gleich gehen kann, wartet. Beim nächsten Sync geht es nach dem
  Push der Sessions, ältester zuerst und mit derselben ID (U-40).
- Der Hinweis sagt dann «Gespeichert» statt «Gesendet».
- Lehnt der Server einen Bericht endgültig ab, verwirft ihn die App. Das
  sind:
  - keine Einwilligung;
  - kein Onboarding;
  - eine unbekannte Region;
  - eine ungültige Eingabe;
  - eine andere Antwort 4xx ausser 401, 408 und 429.

  Beim direkten Senden sagt die App den Grund.
- Ungesendete Berichte zählen als ausstehend. Das Abmelden warnt wie bei
  ungesendeten Sätzen.

### 7. Demo

Zwei neue Screens kommen zu den Screenshots und zum Accessibility-Audit:

- `painafter`: der Abschluss mit den Fragen;
- `morning`: die Morgenkarte.

## Folgen

- Die Rampe kann vorrücken, ohne dass der Athlet selbst an einen Bericht
  denkt. Die Morgenfrage liefert den Bericht, den PAR-D-26 braucht.
- Kein Schmerzbericht geht ohne Verbindung verloren.
- **Nicht gebaut:**
  - Die Frage im Aufwärmen (`warmup`, «hält länger als 15 min an») bleibt
    beim freien Bericht im Log (U-87). §8.6 fragt sie nur, wenn Schmerz
    auftritt.
  - Den täglichen Alltagsschmerz fragt die App nicht eigens. Er bleibt
    freiwillig im Screen «Körper».
  - Die App holt keine Berichte vom Server (`GET /v1/me/pain-reports`). Auf
    einem anderen Gerät Beantwortetes fragt sie noch einmal.
  - Eine Einheit, die nicht aus dem Check-in startet (fortgesetzt oder frei),
    hat keine Frage davor.
  - Die Morgenfrage gilt nur der letzten Einheit von gestern.
  - Lehnt der Server einen wartenden Bericht beim Sync endgültig ab, etwa
    nach einem Widerruf der Einwilligung, verwirft ihn die App ohne Hinweis.

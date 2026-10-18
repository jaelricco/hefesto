# ADR 0028 — Körper und Sicherheit in der iOS-App

- Status: proposed (zur Prüfung mit den Körper-Screens, 19.10.2026)
- Date: 2026-10-19
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Der Server hat für den Körper alles, was der Planer braucht (ADR 0014,
ADR 0015, spec §8):
- `POST /v1/me/symptoms` meldet Belastungssymptome (RF-10). Das Training
  stoppt sofort (SAFE-02).
- `POST /v1/me/pain-reports` nimmt einen Schmerzbericht an. Der Planer kann
  daraufhin die Warnzeichen der Region erfragen (`ask_red_flags`).
- `GET /v1/me/regions` liefert den Zustand jeder Region mit ihren
  Warnzeichen und dem Disclaimer.
- `POST /v1/me/regions/{region}/red-flags` nimmt die Antworten auf die
  Warnzeichen an.
- `POST /v1/me/regions/{region}/clearance` und
  `POST /v1/me/screening/clearance` bestätigen die Freigabe durch eine
  Fachperson.

Die App nutzte davon nichts. Wer beim Training Brustschmerz hatte, konnte es
nicht melden. Wer gestoppt war, konnte den Stopp nicht aufheben, auch nicht
nach der Freigabe durch eine Ärztin. Der Plan konnte also in der App nie
wieder starten.

## Entscheidungen

### 1. Jede Meldung geht sofort an den Server

Meldungen zum Körper laufen nicht über den Ausgang, sondern als einzelne
Anfragen, wie der Start (ADR 0021). Gemeint sind Symptome, Schmerz,
Warnzeichen und Freigaben.
- Sie sind Sicherheit und Gesundheitsdaten. Ihre Wirkung muss der Athlet
  sofort sehen, nicht nach dem nächsten Sync.
- Jede Meldung trägt eine ID. Eine Wiederholung nach verlorener Antwort
  sendet dieselbe ID, und der Server antwortet wie beim ersten Mal (U-40).
- Nach jeder angenommenen Meldung holt die App den Plan und die Regionen
  neu.
- Ohne Verbindung sagt die App, dass die Meldung eine braucht. Bei Symptomen
  sagt sie zuerst: Hör jetzt auf und folge dem Rat. Der Rat steht
  wörtlich aus der Wissensbasis da (RF-10).

Der Sync der Schmerzberichte (§10.5) bleibt offen.

### 2. Ein Screen «Körper»

«Heute» führt mit einer Zeile «Körper: Schmerz und Symptome» dorthin. Der
Screen zeigt:
- oben «Schmerz melden» und «Symptome beim Training»;
- jede Region mit Beschwerde oder Einschränkung, mit ihrem Namen aus der
  Wissensbasis und ihrem Zustand: «Ohne Einschränkung», «Rückkehr, Stufe n
  von 5» oder «Pausiert»;
- was der Zustand für den Plan heisst. Ein Beispiel: «Der Plan lässt diese
  Region weg, bis du eine Freigabe bestätigst»;
- eine Pause bis zu einem Datum, eine empfohlene Abklärung und die
  Einschränkungen einer Fachperson aus dem Onboarding;
- ohne Einwilligung den Hinweis, dass Hefesto keinen Zustand behält, Sperren
  und Ausschlüsse aber gelten;
- den Disclaimer der Antwort, wörtlich.

Je Region gibt es zwei Aktionen. Die Warnzeichen lassen sich jederzeit
beantworten. «Ich wurde freigegeben» erscheint nur, wo eine Freigabe etwas
ändert: bei einer gesperrten Region, in Stufe 0, bei einer empfohlenen
Abklärung oder wenn ihre Warnzeichen alles stoppen.

Die App benennt keine Diagnose und gibt keinen eigenen Rat. Die Zustände
sagen nur, was der Plan tut.

### 3. Schmerz melden

Der Bericht fragt die Region aus dem ganzen Körperbild, den Zeitpunkt und den
Wert von 0 bis 10. Zum Zeitpunkt passend fragt er, was der Planer dazu
braucht (spec §8.6):
- im Aufwärmen, ob der Schmerz länger als 15 Minuten anhält;
- nach dem Training, ob er länger als eine Stunde anhielt;
- während des Trainings, ob er plötzlich und stechend kam (DOSE-23).

Der Text sagt: «Der Plan passt sich an; nichts davon zählt gegen dich»
(ADR 0003).

Danach zeigt die App die Gründe der Änderungen wörtlich. Fragt der Planer
die Warnzeichen einer Region, öffnen sie sich gleich danach. Aus dem Logger
ist der Zeitpunkt «während» vorbelegt, und der Bericht nennt die Session.

Ohne Einwilligung nimmt der Server keinen Schmerzbericht an
(`consent-required`). Die App sagt das in diesen Worten.

### 4. Warnzeichen

Gefragt werden die Fragen, die der Server für die Region nennt, mit ihren
Folgefragen. Ein «Ja» zeigt sofort den Rat der Wissensbasis, wie im
Onboarding (ADR 0023). Gesendet wird erst, wenn jede Frage beantwortet ist,
eine Folgefrage genau dann, wenn ihre Frage «Ja» hatte.

### 5. Symptome beim Training

Das Blatt zeigt die Frage RF-10 und ihren Rat wörtlich. Es sagt, dass die
Meldung das Training sofort pausiert, bis eine Freigabe bestätigt ist. Es
ist aus dem Logger, vom Screen «Körper» und damit von «Heute» erreichbar.

### 6. Freigabe durch eine Fachperson

- **Eine Region:** «Ich wurde freigegeben» fragt nach: «Bestätige nur, was
  dir eine Ärztin, ein Arzt oder eine Physiotherapeutin gesagt hat.» Die
  Region beginnt ihre Rückkehr dann bei der ersten Stufe (spec §8.3).
- **Das ganze Training:** Die Karte «Das Training pausiert» auf «Heute»
  bietet «Eine Fachperson hat mich freigegeben» an. Das bestätigt die
  Freigabe für das Training als Ganzes: Screening, Symptome und gesperrte
  Regionen.
- **Für Minderjährige** fehlt dieser Knopf (SAFE-07). Ihr Stopp hängt nicht
  an einer Freigabe.

### 7. Demo

Die Demo hält zwei Regionen als Testdaten: «innen am Ellbogen» in Stufe 2
der Rückkehr und «am Handrücken …» gesperrt, mit empfohlener Abklärung. Namen,
Fragen und Rat stammen aus der Wissensbasis, die Zustände sind erfunden. Neue
Screens: `body`, `pain`, `symptoms`.

## Folgen

- Wer Symptome oder Schmerz hat, kann es in der App melden, und der Plan
  reagiert sofort. Ein Stopp lässt sich nach einer Freigabe aufheben.
- **Nicht gebaut:**
  - die Einwilligung ändern. Eine Erteilung fragt Screening und
    Vorverletzungen neu (ADR 0015) und braucht dafür einen eigenen Ablauf;
  - die Fragen vor und nach einer Einheit bei beobachteten Regionen
    (spec §8.6); der Bericht ist freiwillig und vom Athleten ausgelöst;
  - die Liste der eigenen Schmerzberichte;
  - der Sync der Schmerzberichte. Ohne Verbindung geht keine Meldung.
- **Befunde für den Review:**
  - **Die Freigabe ist eine Selbstauskunft.** Die App kann nicht prüfen, ob
    eine Fachperson freigegeben hat. Das war schon im Onboarding so
    (spec §8.3).
  - **Symptome ohne Verbindung.** Die App zeigt den Rat und sagt, dass man
    aufhören soll, pausiert den Plan aber erst, wenn die Meldung den Server
    erreicht. Eine Meldung im Ausgang hätte es auch nicht früher getan.

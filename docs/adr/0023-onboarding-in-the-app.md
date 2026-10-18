# ADR 0023 — Das Onboarding in der iOS-App

- Status: proposed (zur Prüfung mit dem Onboarding, 18.10.2026)
- Date: 2026-10-18
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Ohne Onboarding hat ein User keinen Plan: `GET /v1/me/plan` antwortet
`onboarding-required`. Die API dazu gibt es seit ADR 0014:
- `GET /v1/planner/catalogue` liefert Skills, Übungen, Regionen mit
  Red-Flag-Fragen und Antwortklassen.
- `POST /v1/me/onboarding` nimmt alle Antworten auf einmal.

Die App kannte beides nicht. Was sie fragen soll, legt `onboarding.md` §3 fest.

Beim Bauen zeigten sich zwei Lücken im Server:
1. **Die sechs Vorabfragen hatten keinen Wortlaut.** `onboarding.md` §3.7
   nennt nur ihre Themen, die API verlangt sechs Antworten. Die Red Flags
   stehen dagegen mit ihrem Text in der Wissensbasis.
2. **Ohne Einwilligung verlangte die API trotzdem sechs Antworten.** Laut
   `onboarding.md` §3.1 werden die Vorabfragen ohne Einwilligung nicht
   gestellt, und der Server verwirft die Antworten. Die App hätte also
   Fragen stellen müssen, deren Antworten verworfen werden, oder Antworten
   senden, die niemand gab.

## Entscheidungen

### 1. Der Katalog trägt die Sicherheitsfragen

Die sechs Vorabfragen stehen in der Wissensbasis (`body.yaml`, `SCR-01` bis
`SCR-06`), in der Reihenfolge ihrer Antworten.
- Der Wortlaut folgt `onboarding.md` §3.7 Punkt für Punkt, in eigenen Worten
  nach dem PAR-Q+-Konzept. Ob er als Bearbeitung von PAR-Q+ gilt, bleibt die
  offene Rechtsfrage OE-3.
- Die Prüfung KB-10 verlangt genau sechs Fragen; KB-12 prüft auch ihre
  Sprache.

`GET /v1/planner/catalogue` liefert zusätzlich:
- `screening`: diese Fragen;
- `exertion`: die Frage nach Belastungssymptomen, RF-10;
- `minor_age_years`: die Altersgrenze aus PAR-D-23.

Die Texte des Planers kommen damit aus einer Quelle, wie bei den Red Flags
(ADR 0016, KB-11). Die App fragt genau, was der Server prüft, auch die Red
Flags nur für Minderjährige.

### 2. Ohne Einwilligung keine Vorabfragen

`screening` darf ohne Einwilligung leer sein; mit Einwilligung bleiben sechs
Antworten Pflicht. Ein Client, der trotzdem sechs sendet, wird nicht
abgelehnt.

Ohne Einwilligung fragt die App:
- die Belastungssymptome;
- die Körperkarte, nur «jetzt»;
- Details und Red Flags jeder aktuellen Beschwerde.

Der Server braucht diese Angaben für die Entscheidung über Sperren und
Stopps. Er speichert sie nicht (`onboarding.md` §3.1). Vorabfragen und
Verletzungen der letzten 12 Monate fragt die App ohne Einwilligung nicht.

### 3. Ein Formular im Paket, Seiten in der App

`HefestoPlan` bildet den Katalog auf eigene Typen ab
(`OnboardingCatalogue`) und hält die Antworten in `OnboardingForm`.

Das Formular entscheidet:
- welche Fragen gelten (`onboarding.md` §3.6):
  - Hängen und Rudern nur ohne Klimmzug, Rudern nur mit tiefer Stange,
    Ringen oder Park;
  - Stütz nur ohne Dip;
  - Hollow Body bei Planche, Front Lever und L-Sitz;
  - Mobilität je Ziel;
  - die Stufe vor der Pause nach mindestens sieben Wochen Pause;
- wann eine Seite vollständig ist;
- was hinausgeht: nur Antworten, die noch gelten. Eine Antwort auf eine
  Frage, die nach einer Änderung nicht mehr gilt, bleibt zurück.

Die Views zeigen das Formular nur an und füllen es. Die Logik ist in Swift
getestet.

`OnboardingModel` lädt den Katalog, sendet die Antworten und hält den
Schlüssel der gesendeten Antworten (`Idempotency-Key`). Eine Wiederholung
nach einer verlorenen Antwort bekommt so das erste Ergebnis; neue Antworten
bekommen einen neuen Schlüssel. Das Modell lebt im `AppModel`: Wer das
Onboarding schliesst, verliert nichts.

### 4. Die Seiten

Eine Seite je Block von `onboarding.md` §2, mit Fortschritt, «Zurück» und
«Weiter» als Hauptaktion. «Weiter» ist erst möglich, wenn die Seite
vollständig ist.

| Seite | Inhalt |
|---|---|
| Rahmen | Geburtsjahr, Einwilligung mit Erklärung, Hinweis der Wissensbasis zum Bestätigen |
| Ziele | bis zu drei Skills mit Meilenstein, in Reihenfolge; je Ziel die Stufe (Vorgabe: erster Meilenstein) und optional ein Datum |
| Zeit | Einheiten pro Woche, Minuten pro Einheit |
| Ausrüstung | Kacheln; Boden und Wand sind immer da |
| Hintergrund | Körpergewicht, Niveau, Trainingsalter, letzte regelmässige Woche, nach langer Pause die Stufe davor |
| Leistungsstand | die geltenden Klassenfragen, die Stufe je Ziel mit «wie lange» oder «wie viele», Mobilität, Sicherheit der Zahlen |
| Gesundheit | Belastungssymptome, mit Einwilligung die Vorabfragen, die Körperkarte |
| je Beschwerde | Schmerz im Alltag und beim Training, Beginn, Dauer, Verdacht, Fachperson, Einschränkungen, Warnzeichen |
| Abschluss | senden; offene Rückfragen beantworten und erneut senden |

Ein paar Regeln gelten für alle Seiten:
- **Fragen und Antworten** stehen in der App (Englisch und Deutsch, nach
  `onboarding.md`). Fragen der Wissensbasis stehen wörtlich da: Regionen, Red
  Flags, Vorabfragen, Rückfragen und der Hinweis.
- **«Weiss nicht»** gibt es bei jeder Klassen- und Stufenfrage (§5.2).
- **Belastungssymptome** fragt die App einmal. Jede Region verlangt RF-10
  auch; die App sendet dort dieselbe Antwort.
- **Ein «Ja» bei einer Red Flag** zeigt sofort die Empfehlung der
  Wissensbasis und die Folgefrage, falls es eine gibt. Die App fügt keinen
  eigenen Rat hinzu (keine medizinischen Aussagen).
- **Die Dauer einer Beschwerde** ist eine von vier Klassen (§3.7). Die App
  sendet je Klasse eine Wochenzahl darin: 1, 3, 8 und 13. Der Hinweis nach
  mehr als vier Wochen (PAR-D-19) folgt so der Klasse.
- **Ziele** sind Skills mit mindestens einem Meilenstein. Grundlagen ohne
  Meilenstein, wie Hängen oder Hollow Body, sind Weg, nicht Ziel.

### 5. Ergebnis und Einstieg

Der Server antwortet mit einem von drei Ergebnissen:
- **`needs_answers`**: Die Abschlussseite zeigt die Rückfragen. Ohne Antwort
  löst der Planer konservativ auf (§5.5).
- **`complete`**: «Dein Plan ist bereit» mit Realismus-Hinweisen und
  Gründen.
- **`blocked`**: «Das Training pausiert» mit den Gründen.

Bei `complete` und `blocked` hält die App den ersten Plan und ruft ihn neu
ab.

Fehler erscheinen über der Leiste unten. Die Antworten bleiben erhalten.
Bei «schon beantwortet» ruft die App den Plan ab.

Auf «Heute» führt ohne Plan die Karte «Hol dir deinen Trainingsplan». Das
freie Training bleibt als zweite Aktion (ADR 0022).

### 6. Demo

Die Demo hält einen Katalog im JSON der API. Er wurde einmalig aus
`content/training` geschrieben, mit allen Skills, Regionen und Red Flags im
Wortlaut. Drei neue Screens nehmen die Screenshots auf:
- `onboarding`: die Ziele;
- `standing`: der Leistungsstand;
- `complaint`: eine Beschwerde mit Warnzeichen.

Der Demo-Plan trägt jetzt den ganzen Hinweis der Wissensbasis; vorher war es
nur sein erster Satz.

## Folgen

- Ein neuer User bekommt seinen Plan in der App.
- **Nicht gebaut:**
  - der optionale Testtag (OE-6);
  - «Profil verfeinern» (§3.8) und spätere Änderungen von Profil und Zielen;
  - bevorzugte Tage;
  - der Freitext zu «an einer anderen Stelle»;
  - das Bestätigen einer Freigabe nach einem Stopp.
- **Befunde für den Review:**
  - **Bedingungen im Code.** Welche Ziele welche Zusatzfragen bringen, sagt
    der Katalog nicht. Die App hält die Bedingungen aus §3.6 mit den Slugs der
    Wissensbasis. Ein umbenannter Skill lässt seine Frage still fallen; der
    Planer plant dann mit seiner konservativen Vorgabe.
  - **Minderjährige** beantworten das ganze Onboarding und erfahren erst am
    Ende, dass die App für sie keinen Plan erstellt (SAFE-07). Eine frühe
    Auskunft bräuchte die Entscheidung OE-1.
  - **Texte nur auf Deutsch.** Die Wissensbasis hat nur deutsche Texte
    (KB-11). In der englischen App stehen Regionen, Red Flags und
    Vorabfragen deshalb auf Deutsch.

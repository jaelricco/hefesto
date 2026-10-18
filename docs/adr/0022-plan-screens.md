# ADR 0022 — Die Screens des Trainingsplans in der iOS-App

- Status: proposed (zur Prüfung mit den Screens, 18.10.2026)
- Date: 2026-10-18
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Der Unterbau des Plans liegt im Paket (ADR 0021): Die App hält den Plan der
Woche, startet eine geplante Einheit mit Check-in und führt geplante Sätze an
ihrem Platz aus. Es fehlten die Screens. Das Design von ADR 0020 ist in
`main`. Seine Regeln gelten auch hier:
- eine Hauptaktion je Screen;
- Gold nur für Erreichtes;
- Views lesen kleine eigene Modelle, nie die Typen der API.

Offen war:
- Wie verhalten sich der Plan und das freie Training auf «Heute»?
- Was zeigt die App bei einem Stopp?
- Wie kommt das Ziel eines geplanten Satzes in den Logger, und wann fragt
  die App nach der Reserve?
- Was geschieht mit einem geplanten Satz, für den das Log ein Band braucht?

## Entscheidungen

### 1. Ein eigenes Lesemodell: `HefestoPlan`

Das neue Modul `HefestoPlan` bildet den gehaltenen Plan (`TrainingPlan`) an
einer Stelle auf eigene Typen ab: `PlanWeek`, `PlanDay`, `PlanSession`,
`PlanBlock`, `PlanItem` und `PlanNote`. Die Views kennen nur diese Typen.
- Ein unbekannter Status einer Einheit gilt als geplant.
- `PlanModel` beobachtet den gehaltenen Plan von heute. Es ruft ihn nach
  jedem Sync neu ab und startet Einheiten.
- Fehlt das Onboarding, vergisst das Modell den Plan, und «Heute» zeigt keinen
  Plan. Ist der Planer nicht verfügbar oder das Netz weg, bleibt der
  gehaltene Plan stehen.
- Die Demo hat keinen Server; ihr `PlanModel` erreicht das Netz nie.

### 2. Auf «Heute» führt der Plan

Hat der Plan heute eine offene Einheit, ist ihr Start die Hauptaktion. Das
freie Training rückt zur zweiten Aktion («Freies Training») und bleibt immer
möglich. Ist die Einheit erledigt oder hat die App keinen Plan, ist das freie
Training wieder die Hauptaktion.
- **Die Karte von heute** zeigt die Art der Einheit, die Dauer, bis zu vier
  Übungen mit Sätzen und Ziel, «Starten» oder «Weiter» und den Weg zur ganzen
  Einheit. Eine erledigte Einheit zeigt «Für heute erledigt» in Gold.
- **Ein Ruhetag des Plans** ist ein Tag wie jeder andere: «Dein Plan ruht
  heute. Ruhetage zählen zur Serie.» (ADR 0003).
- **Die Woche** steht unter den Aktionen, Tag für Tag, mit Ruhetagen. Ein Tag
  mit Einheit öffnet sie.
- **Ein Stopp** zeigt «Das Training pausiert», die Gründe des Plans wörtlich
  und den Hinweis der Wissensbasis. Die App fügt keinen eigenen Rat hinzu
  (keine medizinischen Aussagen). Das freie Training bleibt als zweite Aktion.

### 3. Eine Einheit ganz

Die Einheit zeigt jeden Block mit Rolle, Minuten und seinem ersten Grund und
jedes Item mit Sätzen, Ziel, Reserve, Pause und Zusatzlast. Chips markieren:
- «Optional» für ein Angebot;
- «Kalibrierung»;
- «Band»;
- «Aufwärmsatz»;
- «Auf Schmerz achten» für ein beobachtetes Item.

Darunter stehen die Gründe der Einheit und der Hinweis der Wissensbasis.
- Die Texte der Gründe kommen aus der Wissensbasis und stehen wörtlich da.
- Starten oder weitermachen kann man nur die Einheit von heute.
- Die Einheit wird bei jedem Anzeigen im aktuellen Plan gesucht. Ersetzt ein
  neuer Plan sie, sagt der Screen das, statt die alte zu zeigen.

### 4. Check-in und Start

Vor dem Start fragt ein Blatt zwei Dinge, beide freiwillig:
- **Schlaf:** 4 bis 9 Stunden; die Enden heissen «oder weniger» und «oder
  mehr».
- **Müdigkeit:** 1 bis 10, «1 ist frisch, 10 ist erschöpft».

Das Blatt sagt, was die Antworten bewirken: Ab 6 Stunden Schlaf oder weniger
oder einer Müdigkeit ab 8 wird der schwerste Block Technikarbeit (ADR 0019).
Es sagt auch, dass die Antworten nicht gespeichert werden. Nur Beantwortetes
geht in die Anfrage. Eine leichtere Einheit trägt danach «Heute leichter».

Der Start:
- **Erst synchronisieren.** Vor dem Start sendet die App ihren Ausgang. So
  plant der Server mit dem letzten Log. Das ist die Antwort der App auf den
  Befund zu offline ausgeführten Sätzen (ADR 0017, spec §15.4).
- **Dann starten** (ADR 0021). Danach ruft das Modell den Plan neu ab, und der
  Logger öffnet sich mit der Session.
- **Fehler** erscheinen im Blatt, in den Worten des Athleten:
  - gestoppt;
  - nicht mehr im Plan;
  - Planer nicht verfügbar;
  - Onboarding fehlt;
  - keine Verbindung.

  Bei einem Stopp und bei «nicht mehr im Plan» ruft das Modell den Plan neu
  ab, damit «Heute» den Grund zeigt.
- **«Weiter»** öffnet die gehaltene Session. Hält die App sie nicht, startet
  sie die Einheit erneut. Der Server antwortet mit derselben Session
  (U-51), und der Logger öffnet sie.

### 5. Der Logger mit Zielen

Der Logger einer geplanten Session heisst «Geplantes Training».
- **Offene geplante Sätze** stehen an ihrem Platz in den Blöcken, als Ziele
  in der Sekundärfarbe: gestrichelte Nummer, Übung, «Ziel · 2 Wdh. Reserve»
  und der Zielwert. Ein Knopf führt den Satz aus.
- **«Nächster Satz»** ist die Hauptaktion, solange geplante Sätze offen sind.
  Er öffnet den nächsten in der Reihenfolge der Blöcke. Ein eigener Satz ist
  der kleine Knopf daneben. Erst ohne offene Ziele kehrt die bisherige Leiste
  zurück: Block anlegen, Satz eintragen.
- **Der Eingabe-Screen** öffnet einen geplanten Satz mit seinen Zielen und
  seiner geplanten Pause. Der Athlet ändert sie in das, was er gemacht hat.
  - Darüber steht das Ziel mit der Ziel-Reserve.
  - Eine Kombination lässt sich hier nicht bauen: Ein geplanter Satz ist ein
    Item des Plans. Es bleibt ein Pfad für Sätze; die Ausführung ist
    `perform(plannedSet:)` (ADR 0021).
- **Die Reserve** wird wie bisher abgefragt und bleibt freiwillig.
  - Ein Kalibrierungssatz trägt den Chip «Kalibrierung» und sagt, warum die
    Reserve hier zählt: «Dieser Satz kalibriert deinen Plan: Was du noch übrig
    hattest, zeigt ihm, wo du stehst.»
  - Kalibrierungssatz ist nur der erste Arbeitssatz eines Items mit
    `calibration`; Aufwärmsätze nie. Auf ihn stützt sich die Kalibrierung
    (SEL-08); die weiteren Sätze sind ermüdet.
  - Die App erzwingt die Reserve nicht. Ein Satz ohne Reserve bleibt eine
    Untergrenze (§4.3).
- **Der Verlauf** zeigt nur Ausgeführtes. Geplante Sätze, die offen blieben,
  erscheinen dort nicht; für den Planer zählen sie ohnehin nicht (ADR 0018).

### 6. Ein geplanter Band-Satz wird noch nicht ausgeführt

> Aufgehoben durch ADR 0024: Der Eingabe-Screen fragt nach dem Band, und
> geplante Band-Sätze sind ausführbar.

Das Log verlangt für eine Band-Unterstützung das Band selbst (ADR 0016), und
die App kann noch kein Band wählen. Der Draft trägt das Band nicht, und der
Eingabe-Screen bietet es nicht an. Ein Band-Satz, wie geplant ausgeführt, stünde
deshalb als unassistierter Satz im Log. Er wäre ein Beleg für die Skill-Karte,
und Freischaltungen werden nie zurückgenommen.

Ein geplanter Satz, dessen Item `assist: band` trägt, bleibt deshalb sichtbar,
lässt sich aber nicht ausführen:
- Er sagt: «Mit Band geplant. Die App kann Bänder noch nicht eintragen.»
- «Nächster Satz» überspringt ihn.

Sobald die Bänder kommen, fragt der Eingabe-Screen nach dem Band, und die
Sperre fällt.

### 7. Eine Demo des Plans

Die Demo hält einen Plan der laufenden Woche im JSON der API:
- heute ein volles Training;
- vor zwei Tagen das erledigte Training des Verlaufs;
- in zwei Tagen ein leichtes Training;
- sonst Ruhetage.

Übungsnamen und Begründungen stehen wörtlich in `content/training/`. Vier neue
Screens zeigen den Plan; `scripts/ios-screenshots.sh` nimmt sie auf:
- `plansession`: die Einheit von heute ganz;
- `checkin`: das Blatt vor dem Start;
- `planned`: der Logger der gestarteten Einheit, das Aufwärmen erledigt;
- `target`: der Eingabe-Screen auf dem Kalibrierungssatz.

## Folgen

- Ein Athlet mit Onboarding sieht seinen Plan auf «Heute» und kann eine
  geplante Einheit mit Check-in starten. Er führt ihre Sätze mit vorbelegten
  Zielen aus. Die Woche und jede Einheit mit ihren Gründen sind offline
  lesbar.
- Die Wahl zwischen Plan und freiem Training bleibt beim Athleten.
- **Nicht gebaut:**
  - **Angebote auf aktive Wahl.** Die Einheit zeigt ein Angebot als
    «Optional», aber der Logger bietet es nicht an. Der Start schreibt
    Angebote nicht in den Draft (ADR 0016).
  - **Die Frage nach Schmerz bei beobachteten Items.** Die Einheit zeigt
    «Auf Schmerz achten»; die Frage braucht Schmerzberichte aus der App.
  - **Das Onboarding in der App.** Ohne Onboarding zeigt «Heute» keinen
    Plan.
  - **Die Wahl des Bands** (Entscheidung 6).
  - **Der Offline-Start.** Ohne Netz sagt das Blatt, dass der Start eine
    Verbindung braucht.
  - **Ein geplanter Satz lässt sich nicht ausdrücklich auslassen.** Er
    bleibt offen.
- **Befunde für den Review:**
  - Gehört die Session zu einem Plan, den die App nicht mehr hält, etwa eine
    über Mitternacht am Sonntag weitergeführte Einheit, findet die App ihre
    Items nicht. Sie zeigt dann keine Kalibrierung und erkennt keinen
    Band-Satz.
  - Ein freier Satz kann kein Band tragen; das galt schon vorher. Wer mit
    Band trainiert, kann seine Sätze bis zum Screen der Bänder nicht ehrlich
    eintragen.

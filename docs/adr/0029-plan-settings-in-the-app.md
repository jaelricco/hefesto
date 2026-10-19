# ADR 0029 — Die Einstellungen des Plans in der iOS-App

- Status: proposed (zur Prüfung mit den Einstellungen, 19.10.2026)
- Date: 2026-10-19
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Nach dem Onboarding konnte der Athlet in der App nichts mehr ändern: weder
Ziele noch Zeit, Ausrüstung oder Körpergewicht. Der Server hat dafür
`GET`/`PUT /v1/me/training-profile` und `GET`/`PUT /v1/me/goals`
(ADR 0014). Beide `PUT` planen die Woche neu (WEEK-08). Das Profil-`PUT`
ersetzt den änderbaren Teil: Optionale Felder, die fehlen, werden gelöscht.

## Entscheidungen

### 1. Dieselben Seiten wie im Onboarding

Ein Screen «Einstellungen des Plans» zeigt die Seiten des Onboardings für
Ziele, Zeit und Ausrüstung und die Frage nach dem Körpergewicht. Er nutzt
dasselbe Formular (`OnboardingForm`).
- Fragen und Regeln sind dieselben (onboarding.md §3); es gibt keine zweite
  Fassung.
- «Speichern» gibt es erst, wenn etwas geändert ist und Ziele und Zeit
  vollständig sind.

### 2. Nur Geändertes geht hinaus, der Rest bleibt

- Ziele und Profil werden je einzeln gesendet, und nur, wenn sie sich
  geändert haben. Jedes `PUT` plant die Woche neu.
- Was der Screen nicht bearbeitet, geht unverändert zurück:
  - bevorzugte Tage;
  - Mobilitäts-Checks;
  - Grenze der Zusatzlast;
  - kleinste Scheibe.

  Sonst löschte der Server diese Felder.
- Danach holt die App den Plan neu. Der Realismus datierter Ziele steht
  wörtlich da (GOAL-05).
- Der Screen sagt, dass Speichern die Woche neu plant und eine begonnene
  Einheit behält, was getan ist (ADR 0017).

### 3. Online

Einstellungen ändern sich nur online. Ohne Verbindung sagt der Screen das.

### 4. Zugang und Demo

«Heute» führt über das Konto-Menü dorthin, sobald ein Plan da ist. Die Demo
zeigt den Screen `settings` mit den Einstellungen ihres Athleten.

## Folgen

- Der Athlet passt Ziele, Zeit, Ausrüstung und Gewicht selbst an, ohne ein
  zweites Onboarding.
- Beim Abmelden vergisst die App, was ihre Modelle vom Athleten im Speicher
  halten (REVIEW_DECISIONS D-16):
  - die Antworten des Onboardings;
  - die Einstellungen;
  - die Regionen des Körpers;
  - die gezeigte Woche.

  Sonst sähe das nächste Konto sie bis zum Neustart der App. Der Katalog
  bleibt: Er ist Inhalt, nicht Daten des Athleten.
- **Nicht gebaut:**
  - bevorzugte Tage, Mobilitäts-Checks, Grenze der Zusatzlast und kleinste
    Scheibe bearbeiten. Das Onboarding fragt sie auch nicht (ADR 0023);
  - Trainingshintergrund und Geburtsjahr. Die API lässt sie nach dem
    Onboarding nicht ändern (U-39).

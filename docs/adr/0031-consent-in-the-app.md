# ADR 0031 — Die Einwilligung in der App und ihr Nachweis

- Status: proposed (zur Prüfung mit dem Screen, 21.10.2026)
- Date: 2026-10-21
- Deciders: Jaelricco

Dieses ADR ist auf Deutsch, wie alle Dokumente des Planer-Tracks.

## Kontext

Die Einwilligung zu Gesundheitsdaten gab es in der App nur im Onboarding. Der
Server kann sie seit ADR 0015 widerrufen und neu erteilen
(`POST /v1/me/health-consent`), die App konnte es nicht.

Einen Nachweis gab es auch nicht. Das Profil hielt nur fest, ob die
Einwilligung jetzt gilt (`health_data_consent`). Das Änderungsprotokoll
zeigte, wann sie sich änderte, aber nicht, welchem Text zugestimmt wurde.
ADR 0015 hat das der rechtlichen Prüfung überlassen. Den Nachweis jetzt zu
bauen ist die vorsichtigere Wahl: Was nicht festgehalten wurde, lässt sich
später nicht nachholen.

## Entscheidungen

### 1. Ein Nachweis je Entscheidung

Eine neue Tabelle `health_consent_records` (Migration `00010`) hält jede
Entscheidung fest:

- erteilt oder nicht;
- die Quelle: das Onboarding oder eine Änderung, mit der ID der Änderung;
- der Zeitpunkt;
- Version und Sprache des Texts, den die App zeigte.

Der Nachweis ist keine Gesundheitsangabe.

- Ein Widerruf behält ihn: Er belegt, wann die Einwilligung galt und wann
  sie endete.
- Er geht mit dem Konto.
- Er ist nur anhängbar, wie das Änderungsprotokoll.
- Eine Wiederholung derselben Änderung hält nichts fest, eine abgelehnte
  Änderung auch nicht. Beides prüft dieselbe Transaktion wie das Ereignis.
- Erteilen, was schon erteilt ist, ändert am Plan nichts, wird aber
  festgehalten: Es ist eine Entscheidung des Athleten.

### 2. Die API

- `GET /v1/me/health-consent` gibt die Einwilligung und die letzten 50
  Entscheidungen zurück, neueste zuerst.
- Das Onboarding nimmt `health_data_consent_text`, eine Änderung `text`,
  beide als `{version, locale}`.
- Beide sind optional, so bleibt die laufende App-Version gültig. Eine
  Entscheidung ohne Text steht ohne Text im Nachweis.
- Weil es noch keinen produktiven Betrieb gibt, wird nichts nachgetragen.

### 3. Der Text gehört der App und hat eine Version

Den Text zeigt die App in zwei Sprachen. Der Server kennt ihn nicht:
Die Wissensbasis ist nur deutsch (KB-11).

- `ConsentTextView` zeigt die Frage und ihre Erklärung, im Onboarding wie auf
  dem neuen Screen.
- `HealthConsentText.version` ist die Version, die mit jeder Entscheidung
  geht. Die Sprache ist die, in der die App den Text zeigt.
- Jede Version liegt wörtlich in `docs/consent/health-data-<version>.json`,
  englisch und deutsch.
- `ios/scripts/check-strings.py` prüft:
  - Der gezeigte Text ist der archivierte.
  - Die Übersetzung im Katalog ist die archivierte.

  Wer ein Wort ändert, muss eine neue Version anlegen. Die alte bleibt im
  Archiv.
- Version 1 ist der bisherige Text des Onboardings (ADR 0023), unverändert.

### 4. Der Screen «Gesundheitsdaten»

Er ist über das Konto-Menü auf «Heute» und über den Screen «Körper»
erreichbar. Er zeigt:

- ob Hefesto Gesundheitsangaben speichert, seit wann und zu welcher
  Textversion;
- den Einwilligungstext;
- was als Gesundheitsangabe zählt (ADR 0015 §2);
- die Entscheidungen des Athleten.

Er bietet zwei Aktionen:

- **Widerruf:** nur nach einer Bestätigung, die sagt, was gelöscht wird und
  was als Schutz bleibt.
  - Die App löscht dabei auch die Schmerzberichte, die sie auf dem Telefon
    hält (ADR 0030).
  - Ohne Verbindung geht nichts: Der Widerruf muss den Server erreichen.
- **Neue Erteilung:** Sie fragt, was ohne Einwilligung nicht gespeichert
  wurde (ADR 0015 §3):
  - die sechs Sicherheitsfragen, Pflicht;
  - die Regionen mit einer Verletzung in den letzten 12 Monaten, freiwillig.

  Die Antworten bleiben nur bis zum Senden im Speicher. Startet danach eine
  Region auf Stufe 0, sagt der Screen, dass Alltagsschmerz und Warnzeichen
  sie dort herausführen, und führt zur Schmerzmeldung. Das ist die Folge,
  die ADR 0015 verlangt hat.
- Eine Wiederholung nach verlorener Antwort trägt dieselbe ID (U-40).

### 5. Demo

Der neue Demo-Screen `consent` zeigt eine Einwilligung, die im Onboarding
gegeben, später widerrufen und wieder gegeben wurde. Er kommt zu den
Screenshots und zum Accessibility-Audit.

## Folgen

- Für jede Entscheidung ab dieser Version lässt sich sagen, wann sie fiel und
  welchem Wortlaut sie galt.
- Der Athlet kann die Einwilligung in der App widerrufen und wieder geben.
  Bisher ging das nur über die API.
- Eine Textänderung braucht eine neue Version. Die App fragt bei einer neuen
  Version nicht von sich aus neu.
- **Nicht gebaut:**
  - Bei einer neuen Textversion erneut fragen. Eine neue Erteilung verlangt
    die Sicherheitsfragen. Ob eine geänderte Fassung eine neue Zustimmung
    braucht, ist eine rechtliche Frage.
  - Ein Export des Nachweises für den Athleten. Der Screen zeigt ihn, der
    Endpunkt liefert ihn.
- **Offen für die rechtliche Prüfung (ENT-4, OE-2):**
  - der Wortlaut selbst;
  - ob der Nachweis nach einem Widerruf so bleiben darf;
  - wie lange er aufbewahrt wird. Heute bleibt er bis zur Löschung des
    Kontos.

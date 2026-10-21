# Einwilligungstexte

Hier liegt jede Version des Texts, mit dem ein Athlet über die Einwilligung zu
Gesundheitsdaten entscheidet (ADR 0031). Der Server hält zu jeder
Entscheidung die Version und die Sprache fest (`health_consent_records`). Mit
diesen Dateien lässt sich später sagen, welchem Wortlaut zugestimmt wurde.

- Eine Datei je Version: `health-data-<version>.json`, mit dem Text auf
  Englisch und auf Deutsch, wörtlich wie in der App.
- Die App zeigt den Text in `ConsentTextView` (`ios/Hefesto/ConsentViews.swift`)
  und sendet `HealthConsentText.version`.
- `ios/scripts/check-strings.py` prüft, dass der gezeigte Text, die
  Übersetzung im Katalog und die archivierte Version übereinstimmen. Wer ein
  Wort ändert, legt eine neue Version an und lässt die alte Datei stehen.

Der Text ist ein Vorschlag aus dem Onboarding (ADR 0023). Seine rechtliche
Prüfung steht aus (ENT-4, OE-2).

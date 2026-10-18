# Unabhängiger Review der Stufe 5

Stand: 28.09.2026, Regelwerk `0.1.0`.

## Methode

Drei Reviewer ohne Vorwissen und ohne Schreibzugriff haben die Spezifikation,
die Wissensbasis, den Code und die Persona-Pläne gegen die Recherche
(`docs/research/`) geprüft. Jeder hatte einen eigenen Schwerpunkt:

- **A — Sicherheit:** Überlastung, Schmerz- und Red-Flag-Logik, Wortlaut.
- **B — Zahlen und Quellen:** jede Zahl gegen ihren Beleg, Umrechnungen,
  Heuristiken.
- **C — Plausibilität der Pläne:** Woche 1 und zwölf simulierte Wochen je
  Persona, aus Sicht eines Trainers.

Beleg war nur, was die Recherche sagt; eigene Einschätzungen der Reviewer sind
als solche markiert. A und C haben ihre Befunde zusätzlich mit eigenen
Testläufen nachgestellt.

Geprüft und ohne Befund: die Beschwerde-Matrix (alle Zeilen), Dringlichkeit und
Aktion von RF-01 bis RF-13 (bis auf RF-05), die Schmerzschwellen und
Verweisfristen, alle 117 Umrechnungen in `parameters.yaml`, alle 280
Belastungswerte in `body.yaml` und die Rechenbeispiele der Spezifikation.

## Ergebnis

| Reviewer | kritisch | wichtig | klein | behoben | teilweise | dokumentiert oder offen |
|---|---|---|---|---|---|---|
| A Sicherheit | 3 | 7 | 4 | 12 | 1 | 1 |
| B Zahlen | 1 | 3 | 12 | 13 | – | 3 |
| C Pläne | 2 | 8 | 5 | 10 | 4 | 1 |

Alle kritischen Befunde sind behoben. Die Korrekturen stehen in den Commits
`259063b`, `4d930f7`, `d69b3ee`, `19c5b7a` und `7421f7e`; die Entscheidungen der
Umsetzung in `spec.md` §15.2 (U-24 bis U-32). Der Checkpoint hat die Fragen
ENT-R-1 bis ENT-R-5 entschieden und alle Vorschläge übernommen (§15.5); die
Umsetzung steht in den Commits `227f4a9` (ENT-R-3), `769798c` (ENT-R-1),
`6a663c6` (ENT-R-2) und `4a88397` (ENT-R-5), die Entscheidungen der Umsetzung in
§15.2 (U-33 bis U-35).

## A — Sicherheit

| Nr. | Schwere | Befund | Behandlung | Stand |
|---|---|---|---|---|
| A-1 | kritisch | Eine Schmerzverletzung in der Rampe senkte die Last kaum: Stufe zurück ohne Volumenschritt, keine Tage Pause, 8/10 über 1 h ohne Deload. | PAR-D-28 umgesetzt: Schritt wiederholen oder zurück, 1 bzw. 2 Tage Pause der Region; PAR-D-18 zusätzlich bei jeder Schwellenverletzung. Tabellentest je Fall. | behoben (U-24) |
| A-2 | kritisch | Die erkannte Pause veraltete; nach 44–65 Tagen wurde das Zielvolumen zur Basis, eine längere Pause erlaubte mehr Last. | Pause wächst während der Abwesenheit, Referenz vor der Pause festgehalten, Pause je Kontengruppe. Property-Test: eine längere Pause erlaubt nie mehr Last. | behoben (U-25) |
| A-3 | kritisch | U-13 liess die Straight-Arm-Last eines Wiedereinsteigers in drei Wochen versechsfachen; Stufen zählten jede Einheit, kein 72-h-Abstand, f(a) wirkungslos, zwei Sprossen in einer Woche. | U-13 zurückgenommen; Stufe zählt nur Einheiten mit Straight-Arm-Last; 72 h; Sprosse eine unter der Stufe vor der Pause; Aufstieg über Kalibrierungssätze. Nach ENT-R-1 rampt eine Pause aus dem Onboarding auf dem eingefrorenen Zielvolumen der ersten Woche, mit f(a) auf den Schritten (U-33). | behoben; ENT-R-1 umgesetzt (U-33) |
| A-4 | wichtig | Einstieg LOAD-04b: Einheitsdeckel × 1.5 statt 10 %, Schritte eines wachsenden Ziels (+82–96 %), «weiss nicht» qualifizierte. | «Weiss nicht» ausgeschlossen, Schritte an die Vorwoche gebunden, kein Abfall bei der Übergabe an LOAD-02. Nach ENT-R-3 gilt der Einheitsdeckel LOAD-03 auch im Einstieg und in der Pausenrampe. | behoben (ENT-R-3) |
| A-5 | wichtig | Eine Verletzung setzte den Zähler der Stufe nicht zurück. | Jede Verletzung startet die Stufe neu. | behoben |
| A-6 | wichtig | In der Rampe wuchs die Haltezeit ungebremst (Lean 4 → 16 s). | M-Zelle: kürzere Halte mit Technik-Dosierung (DOSE-09). | behoben |
| A-7 | wichtig | RF-05 «Gelenk sichtbar verschoben» (N) war nicht auslösbar. | Folgefrage im Katalog; «ja» stoppt das Training (SAFE-02). | behoben |
| A-8 | wichtig | Die Stopp-Regel nannte plötzlichen stechenden Schmerz nicht. | Text ergänzt. | behoben |
| A-9 | wichtig | Stufe 0 hatte keinen Ausgang. | Ausgang nach §8.3; nach einem Verweis nur mit bestätigter Freigabe. Test. | behoben (U-30) |
| A-10 | wichtig | Der Trainingsschmerz aus dem Onboarding wurde ignoriert. | > 5/10 startet die Rampe mit 0.25; der Alltagswert ist der erste Basiswert. | behoben (U-30, Heuristik) |
| A-11 | klein | Texte: Verweis im Onboarding «trotz angepasster Belastung», Grammatik der Regionen, Sehnenanpassung als Tatsache, RF-10 ohne «sofort», `onboarding.md` «Jedes Ja → gesperrt». | Alle Texte korrigiert. | behoben |
| A-12 | klein | Angebote wurden in Rampen angekündigt; Handgelenk-Last war in der Pause nicht gesperrt. | Ein gemeinsames Tor (ADAPT-06a) für Plan und Adaption; Straight-Arm- und Handgelenk-Konten in der Pause gesperrt. Ein angekündigtes Angebot kann weiter an knappen Deckeln scheitern (§15.3). | teilweise |
| A-13 | klein | Kein Disclaimer im Onboarding-Ergebnis; Hinweise fehlten in den Golden Files. | Beides ergänzt. Adaptionsänderungen gehen mit dem neu erzeugten Plan und dessen Disclaimer an den Client. | behoben |
| A-14 | klein | PAR-D-28 enthält nur zwei der vier Soreness Rules. | Lücke der Recherche-Tabelle in §15.3 dokumentiert. | dokumentiert |

## B — Zahlen und Quellen

| Nr. | Schwere | Befund | Behandlung | Stand |
|---|---|---|---|---|
| B-1 | kritisch | Die Folgetag-Regel war abgeschwächt: «höher als vor der Einheit» statt «Schmerz am Folgetag» (PAR-D-28), und ≤ 2 zählte als schmerzfrei (PAR-D-26). | `PAR-S-47`: Soreness zählt gegen den Wert vor der Einheit, ohne Wert gegen 0; eine Einheit zählt für die Rampe nur ohne Soreness. | behoben; bestätigt (ENT-R-4) |
| B-2 | wichtig | Der Rückfall-Halt (2 × 5 s) lag über dem geschätzten Maximum. | Technik-Halte aus μ, ohne Reserve kein Satz und ein Hinweis; mit Band ohne eigenen Wert ein Kalibrierungsstart. | behoben (U-31) |
| B-3 | wichtig | Die Lean zählte schwerer als die Tuck, die Wand-Liegestütz schwerer als die erhöhte. | Lean 0.59 (PAR-C-10, 31°), Wand höchstens 0.55 (PAR-C-20). | behoben |
| B-4 | wichtig | = A-7 | | behoben |
| B-5 | klein | Die Begründung von `PAR-S-36` passte nicht zu den Werten; drei Vorlagenfelder waren ungenutzt. | Begründung korrigiert; Felder in §15.3. | behoben |
| B-6 | klein | Zwei Schwellen für Kantengewichte (0.3 und 0.5). | §6.8 erklärt die tiefere Schwelle der Plateau-Unterstützung; diese ist noch nicht umgesetzt. | dokumentiert |
| B-7 | klein | Beispiel «Dip ≥ 5» statt 8. | Korrigiert. | behoben |
| B-8 | klein | Liegestütz 0.64 statt des Standardwerts 0.66 (PAR-C-18). | 0.66; Push-up Plus 0.69. | behoben |
| B-9 | klein | Stütz am Barren OG 1 ohne Beleg. | OG 0, unter dem Ringstütz. | behoben |
| B-10 | klein | Mehr als 4 Jahre Training lag im Risikofenster. | Fenster endet vor 48 Monaten. | behoben |
| B-11 | klein | Zahlen im Code statt im Katalog. | PAR-A-17, PAR-S-14, PAR-S-20, PAR-S-42, PAR-D-19 aus dem Katalog; die Bandgrenzen des Realismus sind Struktur. | behoben |
| B-12 | klein | Veralteter Text von `PAR-S-31`; kein Test für PAR-S-Texte. | Neu erzeugt; Test gegen Anhang B. | behoben |
| B-13 | klein | Konditionshalte unter 10 s. | In U-9 begründet (sichere Richtung). | dokumentiert |
| B-14 | klein | Zahlen in `personas.md`, die die Golden Files nicht zeigen. | `personas.md` neu geschrieben. | behoben |
| B-15 | klein | Spanne des Etappenziels gegen PAR-A-47. | §3.6 und `personas.md` nennen beides. | behoben |
| B-16 | klein | Forschungsfragen: Plateau ohne MDC (PAR-F-03), offene Halteklassen. | Plateau bleibt Befund in §15.4. | dokumentiert |

## C — Plausibilität der Pläne

| Nr. | Schwere | Befund | Behandlung | Stand |
|---|---|---|---|---|
| C-1 | kritisch | Das Volumen verliess den halbierten Start nie: LOAD-03 mit ganzen Sätzen liess keinen zusätzlichen Satz zu. | `PAR-S-48`: ein ganzer Satz über M ist immer erlaubt. Bent-Arm-Volumen wächst wieder (Persona 5: 2 + 2 → 3 + 3 Sätze); kleine Straight-Arm-Konten wachsen seit ENT-R-2 mit dem Mindestschritt LOAD-12 um einen Satz je 3 Wochen (U-34). | behoben (ENT-R-2) |
| C-2 | kritisch | Deload-Wochen waren nicht leichter. | Deload auf dem gekürzten Plan, Reserve von Wdh. und Halt abgezogen, Sprosse gehalten, keine Angebote (Persona 2: 16 statt 20 Sätze, 6 statt 9 Straight-Arm-Sätze). | behoben (U-27) |
| I-1 | wichtig | Gestapelte Vorsicht: Lean 2 × 4 s für eine Tuck von 10–19 s, Front Lever zwei Regressionen, Advanced Tuck nie erreichbar. | Leans als Konditionshalt; anteiliger σ-Boden (Persona 2 erreicht an beiden Skills die Advanced Tuck). Der Einstieg bleibt eine Untergrenze (PAR-F-26); ein Frog Stand fehlt der Wissensbasis. | teilweise (U-28) |
| I-2 | wichtig | Schätzungen froren ein oder sanken bei wachsender Leistung. | Untergrenzen auf Höhe der Schätzung halten σ an; grosse Reserven lösen einen Kalibrierungssatz aus. | behoben (U-28) |
| I-3 | wichtig | Persona 2 ohne Kraftarbeit ab Woche 2. | Erreichte Empfehlungen mit Erhaltungsdosis; Klimmzüge bleiben. | teilweise (U-29) |
| I-4 | wichtig | Fast leere Einheiten (ein Plank-Satz). | Nach ENT-R-5 wird eine Einheit mit weniger als 2 Arbeitssätzen Ruhetag, ihre Sätze gehen an Einheiten mit derselben Übung oder, bei leichten Übungen, an eine andere Einheit (WEEK-09, U-35). Einheiten, die eine Rampe zählt, bleiben; die Frequenz der verschobenen Übung sinkt. | behoben (ENT-R-5) |
| I-5 | wichtig | Persona 4 blieb neun Wochen an der Lean. | Kalibrierungssatz an der nächsten Sprosse: ab Woche 2 an der Tuck. Nach ENT-R-1: 3 Straight-Arm-Sätze je Woche ab Woche 4, 4 in Woche 12. | behoben (ENT-R-1) |
| I-6 | wichtig | Angebote nannten die falsche Sprosse, wurden angekündigt, ohne im Plan zu erscheinen, und wiederholten sich jede Einheit. | Name der angebotenen Sprosse; Ankündigung nur mit Tor. Ein Abstand zwischen Wiederholungen fehlt. | teilweise |
| I-7 | wichtig | Klimmzüge bei Ellbogenbeschwerde unverändert. | M: Band bei Wiederholungsübungen. | behoben |
| I-8 | wichtig | Persona 5 ohne Zug und ohne Hinweis. | Hinweis GOAL-06 mit fehlender Ausrüstung. Hollow und Stütz hängen laut `02` §8 an späteren Stufen; neue Kanten ohne Beleg wurden nicht erfunden. | teilweise |
| M-1 | klein | Persona 6 las zweimal «eine Stufe unter deiner Angabe». | Eigene Regel SEL-12. | behoben |
| M-2 | klein | ADAPT-07 meldete «Ziel 26 Wdh.» und jede Woche dasselbe Ziel. | Nur neue Ziele im Anfängerbereich. | behoben |
| M-3 | klein | = B-13 | | dokumentiert |
| M-4 | klein | Ungenauigkeiten in `personas.md`. | Neu geschrieben. | behoben |
| M-5 | klein | «Datum in 7 Wochen» für ein Ziel in 8 Wochen. | Gerundet. | behoben |

## Grenzen des Reviews

- Die Reviewer haben den Code gelesen und Szenarien nachgestellt, aber keine
  Messdaten echter Nutzer gesehen; der simulierte Athlet wächst linear.
- Die Recherche selbst wurde nur auf Stichproben geprüft (Stufe 2 hatte eigene
  Zitat-Audits).
- Die rechtlichen Fragen (Minderjährige, Gesundheitsdaten, Screening-Wortlaut)
  und die fachliche Abnahme der Wissensbasis (ENT-10) sind nicht Teil dieses
  Reviews.

# ADR 0020 — The iOS app's visual design: direction B, "Glut"

- Status: accepted
- Date: 2026-09-28
- Deciders: Jaelricco

## Context

Phases 5 and 6 built the iOS app's screens with system styling: light and
dark lists for Today and History, while the brief (§10) asks for a dark,
high-contrast logger, and the skill map is a dark sky by nature. Moving
between them switched the whole app between light and dark.

Three simple directions were drawn as mockups of the same four screens
(Today, logger, skill map, skill detail): A, close to stock iOS; B, dark and
warm throughout; C, light paper and typography. The owner chose B.

## Decisions

### One dark, warm world

- **The app is always dark** (`UIUserInterfaceStyle: Dark` and
  `.preferredColorScheme(.dark)`). There is no light mode. The logger and the
  map were dark already; now nothing switches.
- **One palette** (`ios/Hefesto/Theme.swift`). The ground is warm graphite
  (`#15120F`), cards sit on `#211D19`, and hairlines are `#3A332C`.
  - **Ember** (`#FF7A33`) is for actions: the one thing to press on a screen.
  - **Gold** (`#F2C14E`) is for achievement only: unlocks, bests, XP. It
    never marks something missing (ADR 0003).
  - Every text colour has at least 4.5:1 contrast on the surface it sits on.
- **Barlow Condensed** carries titles and the numbers read at a glance (reps,
  seconds, the rest timer). **Barlow** carries the text. Both are bundled
  under the SIL Open Font License (`ios/Hefesto/Fonts/OFL.txt`). Every size is
  relative to a text style, so Dynamic Type scales it. The digits are set
  tabular, so the timer does not jitter.
- **The system's controls stay**: lists, forms, sheets, swipe to delete. The
  theme colours them; it does not replace them. From iOS 26 the tab bar and
  toolbars are Liquid Glass and keep the system look. Before iOS 26 the tab
  bar gets the warm ground.

### Changes to the screens that come with it

- **Today** shows XP and the streak. The streak tile says that rest days
  count, and it is hidden at zero rather than showing a loss (ADR 0003).
- **The logger's main action sits at the bottom**, in reach of the thumb:
  "Log a set" goes into the last block, next to "Add a block". An earlier
  block keeps a smaller way to log into it. The rest timer is a large card
  with the share of the planned rest used.
- **The map shows a skill below it on the first tap**: its standing, levels
  and next level. A second tap, or the card, opens the skill. VoiceOver reads
  each node as before.
- **The skill page** shows the levels as a ladder and each level as a card.
  Injury notes keep the API's disclaimer first.
- **The celebration reads its own small model**, built from the API's answer
  in `AppModel`, rather than the generated type. API changes to
  `CompletionResult` then stop at one mapping function.

### A demo mode and screenshots, for design review

- **A Debug build started with `-HefestoDemo YES`** opens on seeded data in an
  in-memory database: signed in, offline, nothing sent anywhere
  (`ios/Hefesto/DemoMode.swift`). `-HefestoDemoScreen` picks the screen. The
  "Hefesto Demo" scheme starts it in German. The seed uses only local store
  types, never generated API types, so API changes cannot break the build
  through it. None of it is compiled into Release builds.
- The real skills and criteria in the seed follow `content/`. The locked
  skills on the demo map carry only names from the project brief. This is
  fixture data, not content.
- **`.github/workflows/ios-screenshots.yml`** runs `scripts/ios-screenshots.sh`
  on the self-hosted Mac when the app's views change. It boots one simulator,
  "Hefesto Screenshots", captures every demo screen in German and uploads them
  as an artifact for two weeks. The simulator is created once, reused, and
  shut down after each run. It is the one thing the job keeps outside its
  workspace.

## Consequences

- There is no light mode. In direct sunlight, a light UI would read better.
  If athletes ask for it, a light palette can come later behind the same
  names in `Palette`.
- The app ships about 640 KB of fonts.
- A custom font does not follow the system's Bold Text setting by itself.
  That is left for the accessibility pass (Phase 7).
- Every push that changes the app's views also runs the screenshot job on the
  owner's Mac, about two minutes after the first run.

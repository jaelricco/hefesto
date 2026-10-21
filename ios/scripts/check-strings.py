#!/usr/bin/env python3
"""Checks the app's string catalog against the Swift sources (brief §10).

  1. Every user-facing string literal in ios/Hefesto/*.swift is a key in
     Localizable.xcstrings, so nothing ships in English only.
  2. Every key has a German translation, marked translated and not empty,
     with the same placeholders as the key.
  3. Every key is still used, so the catalog does not rot.
  4. A count in front of a noun has plural variations in both languages.
  5. The consent text the app shows is the one archived for its version in
     docs/consent/, in both languages (ADR 0031).

Swift turns each interpolation into a format specifier (%lld, %@, %lf...)
whose type the source does not show, so both sides are compared with every
placeholder reduced to one token.

Literals that are not shown to the user (identifiers, SF Symbol names, log
messages) are recognised by shape, or listed in IGNORED below with a reason.

Usage: ios/scripts/check-strings.py [app source dir]
"""

import json
import re
import sys
from pathlib import Path

APP = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).resolve().parent.parent / "Hefesto"
CATALOG = APP / "Localizable.xcstrings"
CONSENT_VIEW = APP / "ConsentViews.swift"
CONSENT_ARCHIVE = APP.parent.parent / "docs" / "consent"

# Literals that look like prose but are not shown as UI text.
IGNORED = {
    "Europe/Zurich": "a time zone identifier",
}

# Files that are never shown to athletes.
EXCLUDED_FILES = {
    "DemoMode.swift": "Debug-only fixture data for the demo mode and the UI tests",
    "DemoCatalogue.swift": "Debug-only fixture: the planner's catalogue, verbatim from content/training, for the demo",
}

SPECIFIER = re.compile(r"%(?:\d+\$)?(?:lld|ld|d|lf|f|@|u|llu)")


def norm(s: str) -> str:
    return SPECIFIER.sub("%*", s)


def placeholders(s: str) -> list[str]:
    """The specifier types in a string, ignoring positions."""
    return sorted(re.sub(r"\d+\$", "", m) for m in SPECIFIER.findall(s))


def literals(src: str):
    """Yields (line, text, verbatim) for each string literal, with each
    interpolation replaced by %*. Handles escapes and nested parentheses."""
    i, n, line = 0, len(src), 1
    while i < n:
        c = src[i]
        if c == "\n":
            line += 1
        if src.startswith("//", i):
            j = src.find("\n", i)
            i = n if j < 0 else j
            continue
        if src.startswith("/*", i):
            j = src.find("*/", i)
            line += src[i:j].count("\n")
            i = n if j < 0 else j + 2
            continue
        if c == '"':
            if src.startswith('"""', i):  # multi-line literals are not UI text here
                j = src.find('"""', i + 3)
                line += src[i:j].count("\n")
                i = j + 3
                continue
            start_line = line
            before = src[max(0, i - 12):i]
            verbatim = bool(re.search(r"verbatim:\s*$", before))
            out, i = [], i + 1
            while i < n and src[i] != '"':
                if src[i] == "\\" and i + 1 < n and src[i + 1] == "(":
                    depth, i = 1, i + 2
                    while i < n and depth:
                        if src[i] == "(":
                            depth += 1
                        elif src[i] == ")":
                            depth -= 1
                        elif src[i] == '"':  # a literal inside the interpolation
                            i += 1
                            while i < n and src[i] != '"':
                                i += 2 if src[i] == "\\" else 1
                        i += 1
                    out.append("%*")
                    continue
                if src[i] == "\\" and i + 1 < n:
                    out.append({"n": "\n", "t": "\t", '"': '"', "\\": "\\"}.get(src[i + 1], src[i + 1]))
                    i += 2
                    continue
                out.append(src[i])
                i += 1
            i += 1
            yield start_line, "".join(out), verbatim
            continue
        i += 1


def user_facing(text: str) -> bool:
    """Whether a literal reads like UI text rather than an identifier."""
    if not re.search(r"[A-Za-z]", text):
        return False  # "·", "%*", "0", " "
    if text in IGNORED:
        return False
    if not re.search(r"\s", text) and (re.search(r"[._%/:\\-]", text) or re.search(r"[A-Z]", text[1:])
                                        or text[0].islower()):
        return False  # measures, states, SF Symbols, slugs, Info.plist and defaults keys, formats
    if text.startswith(("http://", "https://", "Bearer ")):
        return False
    return True


def main() -> int:
    catalog = json.loads(CATALOG.read_text(encoding="utf-8"))
    strings = catalog["strings"]
    # "%lld reps" and "%@ reps" look the same from the source.
    by_norm: dict[str, list[str]] = {}
    for k in strings:
        by_norm.setdefault(norm(k), []).append(k)
    errors: list[str] = []
    used: set[str] = set()

    for path in sorted(APP.glob("*.swift")):
        if path.name in EXCLUDED_FILES:
            continue
        src = path.read_text(encoding="utf-8")
        for line, text, verbatim in literals(src):
            keys = by_norm.get(text)
            if keys:
                used.update(keys)
                continue
            if verbatim or not user_facing(text):
                continue
            # Log and error messages for developers are not localized.
            context = src.splitlines()[line - 1]
            if re.search(r"\b(Logger|logger|log|print|fatalError|precondition|assert)\b", context):
                continue
            errors.append(f"{path.name}:{line}: {text!r} is not in {CATALOG.name}")

    for key, entry in sorted(strings.items()):
        if entry.get("shouldTranslate") is False:
            continue
        if key not in used:
            errors.append(f"{CATALOG.name}: {key!r} is not used by the app any more")
        locs = entry.get("localizations", {})
        de = locs.get("de")
        if de is None:
            errors.append(f"{CATALOG.name}: {key!r} has no German translation")
            continue
        units = [de["stringUnit"]] if "stringUnit" in de else [
            v["stringUnit"] for v in de.get("variations", {}).get("plural", {}).values()]
        if not units:
            errors.append(f"{CATALOG.name}: {key!r} has an empty German entry")
        for u in units:
            if u.get("state") != "translated" or not u.get("value", "").strip():
                errors.append(f"{CATALOG.name}: {key!r} German is not translated")
            elif placeholders(u["value"]) != placeholders(key) and "variations" not in de \
                    and "substitutions" not in de:
                errors.append(f"{CATALOG.name}: {key!r} German {u['value']!r} has different placeholders")
        # "%lld sets" must not read "1 sets": a count before a word needs
        # plural forms, unless the count can never be one (listed below).
        # Unit abbreviations ("%lld m") and "of"/"to" do not inflect.
        if re.search(r"%(?:\d+\$)?lld[ -][a-z]{3,}", key) and key not in SINGULAR_NEVER:
            for lang in ("en", "de"):
                loc = locs.get(lang, {})
                if "variations" not in loc and "substitutions" not in loc:
                    errors.append(f"{CATALOG.name}: {key!r} needs plural variations in {lang!r}")

    check_consent_text(strings, errors)

    for e in errors:
        print(e)
    print(f"check-strings: {len(strings)} keys, {len(errors)} problems")
    return 1 if errors else 0


def check_consent_text(strings: dict, errors: list[str]) -> None:
    """The words of ConsentTextView are those archived for the version the
    app sends with each decision, and so is their German: a changed word is a
    new version, and the old one stays archived."""
    if not CONSENT_VIEW.exists():
        return
    src = CONSENT_VIEW.read_text(encoding="utf-8")
    m = re.search(r'static let version = "([^"]+)"', src)
    view = re.search(r"struct ConsentTextView: View \{(.*?)\n\}", src, re.S)
    if not m or not view:
        errors.append(f"{CONSENT_VIEW.name}: no consent text version or no ConsentTextView")
        return
    version = m.group(1)
    archive = CONSENT_ARCHIVE / f"health-data-{version}.json"
    if not archive.exists():
        errors.append(f"{CONSENT_VIEW.name}: consent text version {version} is not archived in {archive}")
        return
    texts = json.loads(archive.read_text(encoding="utf-8"))["texts"]
    shown = {text for _, text, verbatim in literals(view.group(1)) if not verbatim and user_facing(text)}
    archived = {t["en"] for t in texts}
    for text in sorted(shown - archived):
        errors.append(f"{CONSENT_VIEW.name}: {text!r} is not in {archive.name}; a changed text is a new version")
    for text in sorted(archived - shown):
        errors.append(f"{archive.name}: {text!r} is not shown any more; a changed text is a new version")
    for t in texts:
        de = strings.get(t["en"], {}).get("localizations", {}).get("de", {}).get("stringUnit", {}).get("value")
        if de != t["de"]:
            errors.append(f"{archive.name}: the German of {t['en']!r} is {de!r} in {CATALOG.name}, "
                          f"not the archived {t['de']!r}")


# Counted strings whose count is never one, so they need no plural forms.
SINGULAR_NEVER = {
    "%lld times": "only shown when occurrences > 1",
    "%lld times within %lld days": "only shown when occurrences > 1",
}

if __name__ == "__main__":
    sys.exit(main())

# Status

The project is built in the numbered phases of `docs/PROJECT_BRIEF.md` §12.
Each phase ends with a summary and a stop for review. This file records where
that stands.

## Done

- **Phase 0** — scaffold, Compose, CI, deploy pipeline, ADRs 0001–0004, DDL proposal.
- **Phase 1** — migrations, content pipeline, seed (ADRs 0005–0006). Merged in #4.

## Current phase: 2 — auth and the core API (complete, awaiting review)

Built:

- **OpenAPI** `api/openapi.yaml` v0.2.0 covers auth, me, exercises, bands and
  sessions/blocks/sets. It is embedded in the binary and **enforced**: every
  request body is validated against its schema, integration tests validate
  every response against its schema, and a unit test fails if routes and spec
  drift apart (ADR 0007).
- **Auth** (`internal/auth`): email/password with argon2id; Sign in with Apple
  (identity token checked against Apple's keys, nonce, audience; verified
  emails link to existing accounts); 15-minute HS256 access tokens; rotating
  refresh tokens with family-wide revocation on reuse; per-device sign-out;
  per-address rate limiting on credential endpoints.
- **Account deletion**: `DELETE /v1/me` starts the 30-day grace period and
  signs out everywhere; signing in cancels it; an hourly, advisory-locked job
  hard-deletes accounts past it.
- **Logging API**: create/list/get/patch/delete sessions; `PUT`/`DELETE`
  blocks; `PUT`/`DELETE` sets — the one write path, a set with its complete
  ordered `elements`, one for a plain set, N for a combo; assistance and
  derived `assistance_class`; a reorder endpoint; "repeat last set".
  Client-generated UUIDv7 ids throughout, as the Phase 4 sync contract needs.
- **Catalogue**: exercise search (fuzzy name, family, equipment) with an
  `ETag` keyed on the content version; bands (catalogue + my own).
- **Domain** (`internal/domain/training`): set/element/block/session rules and
  the assistance-class derivation, pure and unit-tested.
- **Tests**: unit tests for the domain, tokens, passwords, Apple verification
  and the route/spec contract; 23 HTTP integration tests against real Postgres
  covering the auth flows, reuse detection, deletion and reaping, the
  catalogue, bands, combos, element replacement, assistance, validation,
  ordering, tombstones, pagination, last-set and cross-user isolation.

### Verification

Run in the development container against PostgreSQL 16.13: `golangci-lint`
clean; `go test -race -short ./...` and `go test -race -tags integration ./...`
green; `redocly lint` clean apart from four warnings carried over from Phase 0;
`oasdiff` reports no breaking change against `main`; the API was exercised by
hand with curl against the dev database.

As in Phase 1, Docker image pulls are blocked here, so the integration tests
ran through `HEFESTO_TEST_DATABASE_URL`; CI runs them on testcontainers.

### Deliberately not in this phase

- **Email verification and password reset.** Not in the brief's §4 list; the
  `email_verifications` table is ready. Worth doing before a public release.
- **`POST /v1/sessions/{id}/complete`** arrives with the unlock engine in
  Phase 3, so a session cannot be completed yet (only drafted or abandoned).
- **Templates** have tables but no endpoints yet.

### Open questions for review

1. **Registration reveals a taken email** (409). Acceptable, or should
   registration go through email verification so it cannot be probed?
2. **Rate limits** are in memory, per API process: 10 credential attempts per
   minute per address. Fine for one instance; revisit if we scale out.
3. **Device ids are per account.** A phone switching accounts must mint a new
   device id. The iOS client needs to know this in Phase 5.
4. Phase 1's questions still stand: the implicit level ladder, permanent
   levels, criteria `assistance: none | any`, and user bands in Phase 4 sync.

## Next: Phase 3

Unlock rules engine with golden tests, `/v1/me/skill-map`, `/complete`
returning unlocks, XP and streaks.

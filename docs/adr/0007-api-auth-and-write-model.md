# ADR 0007 — API, authentication and the logging write model

- Status: accepted
- Date: 2026-09-24
- Deciders: Jaelricco

## Context

Phase 2 puts the first real surface on the API: accounts, the exercise
catalogue, bands, and the logger's sessions → blocks → sets → elements. Several
choices here are hard to change once an iOS client depends on them.

## Decisions

### The OpenAPI document is enforced, not just published

`api/openapi.yaml` is embedded in the binary. Every request body is validated
against its `components/schemas` entry before a handler sees it (OpenAPI 3.1
schemas are JSON Schema 2020-12). Enums, ranges, required fields and
`additionalProperties: false` are therefore defined once, in the file the
Swift client is generated from. Integration tests validate every response
against its schema, and a unit test fails if a route is served but not
documented, or documented but not served.

Rules a schema cannot express (a value for the wrong `measure`, a completed set
missing a value, a set whose block belongs to another session) live in
`internal/domain/training` and the store, and fail with the same
`validation` problem, keyed by JSON pointer.

### Client-generated ids everywhere

Sessions, blocks, sets, elements, assistance rows and user bands all carry an
id the client generates (UUIDv7). This is the Phase 4 sync contract applied
early, so an object created over REST and one created offline look the same.
Consequences: blocks and sets are written with `PUT .../{id}` (create or
replace, 201 or 200); `POST /v1/sessions` returns 409 for an id that exists.

### One write path for a set

`PUT /v1/sessions/{id}/sets/{setId}` takes the set with its complete, ordered
`elements` array — one element for a plain set, several for a combo. There is
no endpoint that writes an element on its own. Elements missing from the new
list are tombstoned; array order is element order.

### Ordering

Sibling order is `order_index` under a unique constraint that is deferrable,
so it cannot also be partial. Tombstoned rows are therefore **parked** above
999 999 999 when deleted, which frees their position at once. A write that
lands on a live sibling's position is a 409 `order-conflict`;
`POST /v1/sessions/{id}/reorder` renumbers whole sibling lists in one
transaction with constraints deferred.

### Tombstones are final

`PUT` on the id of a deleted block or set is a 404. Resurrecting ids would
make "which write wins" ambiguous for sync, and the client can always mint a
new id.

### Tokens

- **Access tokens** are HS256 JWTs, 15 minutes, carrying the user and the
  device. Only this service verifies them, so a shared secret is enough.
- **Refresh tokens** are 32 random bytes, stored as SHA-256 only, rotated on
  every use. Each sign-in starts a *family*; presenting a spent token revokes
  the whole family (the standard reuse-detection response to theft).
  Logout revokes one family: that device.
- **Devices.** A device id is bound to the first account that uses it and is
  never re-owned. A client uses one device id per account. The device becomes
  the `client_id` on every row written with its tokens.
- **Passwords** are argon2id (64 MiB, 3 iterations, 2 lanes by default), with
  a dummy verification for unknown emails so a login costs the same either
  way. Registration does reveal that an email is taken (409); hiding it
  would need email verification to be the only path, which is later work.
- **Credential endpoints are rate limited** per client address, in memory.
  Behind Caddy the address is the rightmost `X-Forwarded-For` entry, and only
  when `HEFESTO_TRUST_PROXY_HEADERS=true`.

### Sign in with Apple

The identity token is verified against Apple's published keys (RS256, issuer,
audience = our client ids, expiry, and the SHA-256 nonce). A known Apple
subject signs in to its account. Otherwise a **verified** email that matches
an account links to it; an unverified email is neither trusted nor stored.
Otherwise a new account is created.

### Account deletion

`DELETE /v1/me` starts the 30-day grace period decided in ADR 0005 and
revokes every refresh token. Signing in during the grace period cancels the
deletion, and the sign-in response says so. An hourly job in the API process
hard-deletes accounts past the grace period; it is advisory-locked, so any
number of API processes can run it.

## Consequences

- Changing the API means changing `openapi.yaml` first; the server then
  enforces the change and the tests notice drift.
- Two small risks accepted: an in-memory rate limiter allows the full rate per
  process, and access tokens stay valid for up to 15 minutes after the account
  signs out everywhere.
- Deferred to later phases: email verification and password reset (the
  `email_verifications` table exists), session completion (Phase 3, with the
  unlock engine), and user bands in sync (Phase 4).

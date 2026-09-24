// Package auth issues and checks credentials: argon2id password hashes, JWT
// access tokens, rotating opaque refresh tokens with reuse detection, and
// Sign in with Apple identity tokens.
//
// Authorisation goes through one helper, Authorize, with an explicit actor
// and subject (ADR 0002). In v1 they always match; adding coaching later
// changes that helper, not every handler.
package auth

//go:build integration

package http_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestRegisterAndLogin(t *testing.T) {
	a := newAPI(t)
	u := a.register("athlete@example.test")

	me := a.call("GET", "/v1/me", u.access, nil).ok(200, "User")
	if me["email"] != "athlete@example.test" || me["timezone"] != "Europe/Zurich" || me["status"] != "active" {
		t.Fatalf("unexpected profile %v", me)
	}

	a.call("POST", "/v1/auth/register", "", map[string]any{
		"email": "ATHLETE@example.test", "password": "another good password",
	}).problem(409, "email-taken")

	errs := a.call("POST", "/v1/auth/register", "", map[string]any{
		"email": "no", "password": "short", "surprise": true,
	}).problem(422, "validation")
	hasField(t, errs, "/email")
	hasField(t, errs, "/password")

	a.call("POST", "/v1/auth/register", "", []byte("{not json")).problem(400, "bad-request")

	a.call("POST", "/v1/auth/login", "", map[string]any{
		"email": "athlete@example.test", "password": "wrong password!",
	}).problem(401, "invalid-credentials")
	a.call("POST", "/v1/auth/login", "", map[string]any{
		"email": "nobody@example.test", "password": "correct horse battery",
	}).problem(401, "invalid-credentials")

	// Email lookup is case-insensitive.
	a.call("POST", "/v1/auth/login", "", map[string]any{
		"email": "Athlete@Example.test", "password": "correct horse battery",
	}).ok(200, "AuthResponse")
}

func TestAccessTokenIsRequired(t *testing.T) {
	a := newAPI(t)
	a.call("GET", "/v1/me", "", nil).problem(401, "unauthorized")
	a.call("GET", "/v1/me", "not-a-jwt", nil).problem(401, "unauthorized")
	a.call("GET", "/v1/sessions", "", nil).problem(401, "unauthorized")
}

func TestRefreshRotationAndReuseDetection(t *testing.T) {
	a := newAPI(t)
	u := a.register("rotate@example.test")

	first := a.call("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": u.refresh}).ok(200, "AuthResponse")
	second := first["refresh_token"].(string)
	if second == u.refresh {
		t.Fatal("refresh did not rotate the token")
	}
	// The new access token works.
	a.call("GET", "/v1/me", first["access_token"].(string), nil).ok(200, "User")

	// Presenting the spent token again is theft: the whole family dies,
	// including the legitimately rotated one.
	a.call("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": u.refresh}).problem(401, "invalid-refresh-token")
	a.call("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": second}).problem(401, "invalid-refresh-token")

	// Other sign-ins (other families) are unaffected.
	other := a.call("POST", "/v1/auth/login", "", map[string]any{
		"email": "rotate@example.test", "password": "correct horse battery",
	}).ok(200, "AuthResponse")
	a.call("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": other["refresh_token"]}).ok(200, "AuthResponse")

	a.call("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": "made-up"}).problem(401, "invalid-refresh-token")
}

func TestLogoutRevokesTheDevice(t *testing.T) {
	a := newAPI(t)
	u := a.register("logout@example.test")
	a.call("POST", "/v1/auth/logout", "", map[string]any{"refresh_token": u.refresh}).ok(204, "")
	a.call("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": u.refresh}).problem(401, "invalid-refresh-token")
	// Unknown tokens are accepted silently.
	a.call("POST", "/v1/auth/logout", "", map[string]any{"refresh_token": "made-up"}).ok(204, "")
}

func TestExpiredRefreshToken(t *testing.T) {
	a := newAPI(t)
	u := a.register("expired@example.test")
	a.exec(`UPDATE refresh_tokens SET expires_at = now() - interval '1 minute'`)
	a.call("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": u.refresh}).problem(401, "invalid-refresh-token")
}

func TestDeviceIDIsPerAccount(t *testing.T) {
	a := newAPI(t)
	u := a.register("first@example.test")
	errs := a.call("POST", "/v1/auth/register", "", map[string]any{
		"email": "second@example.test", "password": "correct horse battery",
		"device": map[string]any{"id": u.device, "platform": "ios"},
	}).problem(422, "validation")
	hasField(t, errs, "/device/id")
}

func TestSignInWithApple(t *testing.T) {
	a := newAPI(t)
	tok := a.appleToken(jwt.MapClaims{
		"sub": "001.apple.one", "email": "one@privaterelay.appleid.com", "email_verified": "true",
		"is_private_email": "true", "nonce": nonceHash("raw-nonce"),
	})
	created := a.call("POST", "/v1/auth/apple", "", map[string]any{
		"identity_token": tok, "nonce": "raw-nonce", "display_name": "One",
	}).ok(201, "AuthResponse")
	uid := created["user"].(map[string]any)["id"]
	if created["user"].(map[string]any)["display_name"] != "One" {
		t.Fatalf("display name not stored: %v", created["user"])
	}

	again := a.call("POST", "/v1/auth/apple", "", map[string]any{"identity_token": tok, "nonce": "raw-nonce"}).ok(200, "AuthResponse")
	if again["user"].(map[string]any)["id"] != uid {
		t.Fatal("second Apple sign-in created another account")
	}

	a.call("POST", "/v1/auth/apple", "", map[string]any{"identity_token": tok, "nonce": "other"}).problem(401, "invalid-identity-token")
	a.call("POST", "/v1/auth/apple", "", map[string]any{
		"identity_token": a.appleToken(jwt.MapClaims{"sub": "x", "aud": "com.evil.app"}),
	}).problem(401, "invalid-identity-token")
}

func TestAppleLinksAVerifiedEmail(t *testing.T) {
	a := newAPI(t)
	pw := a.register("linked@example.test")

	m := a.call("POST", "/v1/auth/apple", "", map[string]any{
		"identity_token": a.appleToken(jwt.MapClaims{"sub": "001.linked", "email": "linked@example.test", "email_verified": true}),
	}).ok(200, "AuthResponse")
	if m["user"].(map[string]any)["id"] != pw.id {
		t.Fatal("a verified Apple email did not link to the existing account")
	}

	// An unverified email never links, and is not recorded.
	m = a.call("POST", "/v1/auth/apple", "", map[string]any{
		"identity_token": a.appleToken(jwt.MapClaims{"sub": "001.unverified", "email": "linked@example.test", "email_verified": "false"}),
	}).ok(201, "AuthResponse")
	if u := m["user"].(map[string]any); u["id"] == pw.id || u["email"] != nil {
		t.Fatalf("unverified email was trusted: %v", u)
	}
}

func TestAccountDeletionGraceAndReaper(t *testing.T) {
	a := newAPI(t)
	u := a.register("leaving@example.test")

	d := a.call("DELETE", "/v1/me", u.access, nil).ok(202, "DeletionScheduled")
	req, _ := time.Parse(time.RFC3339, d["deletion_requested_at"].(string))
	after, _ := time.Parse(time.RFC3339, d["hard_delete_after"].(string))
	if got := after.Sub(req); got != 30*24*time.Hour {
		t.Fatalf("grace period %s", got)
	}
	// Every device is signed out.
	a.call("POST", "/v1/auth/refresh", "", map[string]any{"refresh_token": u.refresh}).problem(401, "invalid-refresh-token")

	// Signing in within the grace period cancels the deletion.
	back := a.call("POST", "/v1/auth/login", "", map[string]any{
		"email": "leaving@example.test", "password": "correct horse battery",
	}).ok(200, "AuthResponse")
	if back["deletion_cancelled"] != true || back["user"].(map[string]any)["status"] != "active" {
		t.Fatalf("login did not cancel the deletion: %v", back)
	}

	// Delete again, let the grace period pass, and reap.
	a.call("DELETE", "/v1/me", back["access_token"].(string), nil).ok(202, "DeletionScheduled")
	a.exec(`UPDATE users SET deletion_requested_at = now() - interval '31 days'`)
	n, err := a.store.ReapDeletedUsers(context.Background(), 30*24*time.Hour)
	if err != nil || n != 1 {
		t.Fatalf("reaped %d, err %v", n, err)
	}
	a.call("POST", "/v1/auth/login", "", map[string]any{
		"email": "leaving@example.test", "password": "correct horse battery",
	}).problem(401, "invalid-credentials")
}

func TestUpdateProfile(t *testing.T) {
	a := newAPI(t)
	u := a.register("profile@example.test")
	m := a.call("PATCH", "/v1/me", u.access, map[string]any{
		"display_name": "Ada", "unit_system": "imperial", "week_start": 7, "timezone": "America/New_York",
	}).ok(200, "User")
	if m["display_name"] != "Ada" || m["unit_system"] != "imperial" || m["week_start"] != float64(7) {
		t.Fatalf("update not applied: %v", m)
	}
	hasField(t, a.call("PATCH", "/v1/me", u.access, map[string]any{"timezone": "Mars/Olympus"}).problem(422, "validation"), "/timezone")
	hasField(t, a.call("PATCH", "/v1/me", u.access, map[string]any{"week_start": 8}).problem(422, "validation"), "/week_start")
	a.call("PATCH", "/v1/me", u.access, map[string]any{}).problem(422, "validation")
}

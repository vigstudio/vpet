package httpapi

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestPasswordRoundTrip(t *testing.T) {
	encoded, err := hashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword(encoded, "correct horse battery staple") {
		t.Fatal("correct password rejected")
	}
	if verifyPassword(encoded, "wrong password") {
		t.Fatal("wrong password accepted")
	}
}

func TestVerifyPasswordRejectsMalformedAndExpensiveHashes(t *testing.T) {
	valid, err := hashPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(valid, "$")
	cases := []string{
		"",
		"$argon2i$v=19$" + parts[3] + "$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=262145,t=3,p=2$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=65536,t=0,p=2$" + parts[4] + "$" + parts[5],
		"$argon2id$v=19$m=65536,t=3,p=0$" + parts[4] + "$" + parts[5],
		valid + "$extra",
	}
	for _, encoded := range cases {
		if verifyPassword(encoded, "correct horse battery staple") {
			t.Fatalf("malformed hash accepted: %q", encoded)
		}
	}
}

func TestSameOriginAllowsConfiguredURLAndLocalLoopbackAlias(t *testing.T) {
	s := &Server{PublicURL: mustURL(t, "http://localhost:18080")}
	for _, origin := range []string{"", "http://localhost:18080", "http://127.0.0.1:18080"} {
		r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:18080/api", nil)
		r.Host = "127.0.0.1:18080"
		r.Header.Set("Origin", origin)
		if !s.sameOrigin(r) {
			t.Fatalf("local origin rejected: %q", origin)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:18080/api", nil)
	r.Host = "127.0.0.1:18080"
	r.Header.Set("Origin", "http://evil.example")
	if s.sameOrigin(r) {
		t.Fatal("foreign origin accepted")
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func TestRateLimitIgnoresUntrustedForwardedFor(t *testing.T) {
	s := &Server{limiter: newLimiter()}
	for i := 0; i < 8; i++ {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = "192.0.2.1:1234"
		r.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i))
		if !s.limit(httptest.NewRecorder(), r, "login:user") {
			t.Fatalf("attempt %d rejected early", i+1)
		}
	}
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.RemoteAddr = "192.0.2.1:9999"
	r.Header.Set("X-Forwarded-For", "203.0.113.10")
	if s.limit(httptest.NewRecorder(), r, "login:user") {
		t.Fatal("spoofed forwarded address bypassed limiter")
	}
}

func TestLimiterCapsAndSweepsExpiredKeys(t *testing.T) {
	l := newLimiter()
	now := time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 8; i++ {
		if !l.allow("client", now) {
			t.Fatalf("attempt %d rejected early", i+1)
		}
	}
	if l.allow("client", now) {
		t.Fatal("ninth attempt accepted")
	}
	if !l.allow("other", now.Add(2*time.Minute)) {
		t.Fatal("new key rejected after window")
	}
	if _, exists := l.values["client"]; exists {
		t.Fatal("expired limiter key retained")
	}
}

func TestVerifyClaimCode(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	batch := base64.RawURLEncoding.EncodeToString([]byte("cutover-2026-10-30"))
	user := base64.RawURLEncoding.EncodeToString([]byte("user-id"))
	payload := "v1." + batch + "." + user
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	code := payload + "." + hex.EncodeToString(mac.Sum(nil))

	gotBatch, gotUser, err := verifyClaimCode(code, secret)
	if err != nil || gotBatch != "cutover-2026-10-30" || gotUser != "user-id" {
		t.Fatalf("valid code rejected: batch=%q user=%q err=%v", gotBatch, gotUser, err)
	}
	if _, _, err = verifyClaimCode(code+"0", secret); err == nil {
		t.Fatal("tampered code accepted")
	}
}

func TestAllianceRateKeyBoundsUnknownActions(t *testing.T) {
	if allianceRateKey("u", "attack") == allianceRateKey("u", "garrison") {
		t.Fatal("known actions shared a bucket")
	}
	if allianceRateKey("u", "invented-a") != allianceRateKey("u", "invented-b") {
		t.Fatal("unknown actions created unbounded buckets")
	}
}

func TestAllianceActionsUseSeparateRateBuckets(t *testing.T) {
	s := &Server{limiter: newLimiter()}
	for i := 0; i < 8; i++ {
		r := httptest.NewRequest(http.MethodPost, "/api/v1/alliance", nil)
		r.RemoteAddr = "192.0.2.1:1234"
		if !s.limit(httptest.NewRecorder(), r, allianceRateKey("u", "attack")) {
			t.Fatalf("attack %d rejected early", i+1)
		}
	}
	other := httptest.NewRequest(http.MethodPost, "/api/v1/alliance", nil)
	other.RemoteAddr = "192.0.2.1:1234"
	if !s.limit(httptest.NewRecorder(), other, allianceRateKey("u", "garrison")) {
		t.Fatal("garrison shared the attack bucket")
	}
	ninth := httptest.NewRequest(http.MethodPost, "/api/v1/alliance", nil)
	ninth.RemoteAddr = "192.0.2.1:1234"
	if s.limit(httptest.NewRecorder(), ninth, allianceRateKey("u", "attack")) {
		t.Fatal("ninth attack request bypassed limiter")
	}
}

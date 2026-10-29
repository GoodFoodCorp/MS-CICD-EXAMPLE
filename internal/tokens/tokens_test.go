package tokens

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testConfig(t *testing.T) Config {
	t.Helper()
	t.Setenv("JWT_SECRET", "access-secret")
	t.Setenv("JWT_REFRESH_SECRET", "")
	cfg, err := FromEnv()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestFromEnv_RequiresSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	if _, err := FromEnv(); err == nil {
		t.Fatal("expected an error without JWT_SECRET")
	}
}

func TestFromEnv_DerivesDistinctRefreshKey(t *testing.T) {
	cfg := testConfig(t)
	if string(cfg.RefreshSecret) == string(cfg.AccessSecret) {
		t.Fatal("refresh key must differ from the access key")
	}
	if cfg.Issuer != DefaultIssuer || cfg.Audience != DefaultAudience {
		t.Fatalf("unexpected defaults: %q %q", cfg.Issuer, cfg.Audience)
	}
}

func TestAccessToken_RoundTrip(t *testing.T) {
	cfg := testConfig(t)
	raw, err := cfg.IssueAccess("u1", map[string]any{"role_slugs": []string{"user"}, "sub": "ignored"}, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	claims, err := cfg.ParseAccess(raw)
	if err != nil {
		t.Fatal(err)
	}
	if claims["sub"] != "u1" || claims["typ"] != TypeAccess {
		t.Fatalf("unexpected claims: %v", claims)
	}
}

func TestTokens_AreNotInterchangeable(t *testing.T) {
	cfg := testConfig(t)
	now := time.Now()
	access, _ := cfg.IssueAccess("u1", nil, now)
	refresh, _ := cfg.IssueRefresh("u1", now)

	if _, err := cfg.ParseAccess(refresh); !errors.Is(err, ErrInvalid) {
		t.Fatal("a refresh token must never be accepted as an access token")
	}
	if _, err := cfg.ParseRefresh(access); !errors.Is(err, ErrInvalid) {
		t.Fatal("an access token must never be accepted as a refresh token")
	}
	if sub, err := cfg.ParseRefresh(refresh); err != nil || sub != "u1" {
		t.Fatalf("refresh round trip failed: %q %v", sub, err)
	}
}

func TestParseAccess_RejectsExpired(t *testing.T) {
	cfg := testConfig(t)
	raw, _ := cfg.IssueAccess("u1", nil, time.Now().Add(-time.Hour))
	if _, err := cfg.ParseAccess(raw); err == nil {
		t.Fatal("expired token accepted")
	}
}

func TestParseAccess_RejectsWrongAudienceAndAlgorithm(t *testing.T) {
	cfg := testConfig(t)
	other := cfg
	other.Audience = "another-platform"
	raw, _ := other.IssueAccess("u1", nil, time.Now())
	if _, err := cfg.ParseAccess(raw); err == nil {
		t.Fatal("token for another audience accepted")
	}

	none, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"sub": "u1", "typ": TypeAccess, "iss": cfg.Issuer, "aud": cfg.Audience,
		"exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := cfg.ParseAccess(none); err == nil {
		t.Fatal("alg=none token accepted")
	}
}

func TestIssueRefresh_UniquePerCall(t *testing.T) {
	cfg := testConfig(t)
	now := time.Now()
	a, _ := cfg.IssueRefresh("u1", now)
	b, _ := cfg.IssueRefresh("u1", now)
	if a == b || Hash(a) == Hash(b) {
		t.Fatal("two refresh tokens issued at the same instant must differ")
	}
}

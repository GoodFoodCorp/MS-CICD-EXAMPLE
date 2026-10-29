// Package tokens issues and verifies the JWTs of the platform.
//
// Access and refresh tokens are two different kinds of credential and must
// never be interchangeable:
//   - an access token (typ=access, 15 min) is what every service accepts;
//   - a refresh token (typ=refresh, 7 days) is only ever accepted by
//     auth-service's /refresh endpoint and is signed with a different key, so
//     the other services cannot even verify it.
//
// Both carry the issuer and audience so services can reject a token minted
// for another platform that happens to share the secret.
package tokens

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TypeAccess  = "access"
	TypeRefresh = "refresh"

	DefaultIssuer   = "goodfood-auth"
	DefaultAudience = "goodfood-api"

	AccessTTL  = 15 * time.Minute
	RefreshTTL = 7 * 24 * time.Hour

	// refreshKeyLabel derives the refresh signing key from JWT_SECRET when no
	// JWT_REFRESH_SECRET is configured: the derived key differs from the
	// access key, so a refresh token never verifies as an access token.
	refreshKeyLabel = "goodfood/refresh-token/v1"
)

// ErrInvalid is returned for any token that must not be accepted.
var ErrInvalid = errors.New("invalid token")

// Config holds the signing material. Build it with FromEnv.
type Config struct {
	AccessSecret  []byte
	RefreshSecret []byte
	Issuer        string
	Audience      string
}

// FromEnv reads JWT_SECRET (required), JWT_REFRESH_SECRET, JWT_ISSUER and
// JWT_AUDIENCE.
func FromEnv() (Config, error) {
	access := os.Getenv("JWT_SECRET")
	if access == "" {
		return Config{}, errors.New("JWT_SECRET is required")
	}
	cfg := Config{
		AccessSecret:  []byte(access),
		RefreshSecret: []byte(os.Getenv("JWT_REFRESH_SECRET")),
		Issuer:        envOr("JWT_ISSUER", DefaultIssuer),
		Audience:      envOr("JWT_AUDIENCE", DefaultAudience),
	}
	if len(cfg.RefreshSecret) == 0 {
		mac := hmac.New(sha256.New, cfg.AccessSecret)
		mac.Write([]byte(refreshKeyLabel))
		cfg.RefreshSecret = mac.Sum(nil)
	}
	return cfg, nil
}

// MustFromEnv is FromEnv for call sites that run after startup validation.
func MustFromEnv() Config {
	cfg, err := FromEnv()
	if err != nil {
		panic(err)
	}
	return cfg
}

// IssueAccess signs an access token carrying the given business claims
// (email, roles, tenant...). Registered claims are set here and override any
// value of the same name in extra.
func (c Config) IssueAccess(subject string, extra map[string]any, now time.Time) (string, error) {
	claims := jwt.MapClaims{}
	for k, v := range extra {
		claims[k] = v
	}
	c.stamp(claims, subject, TypeAccess, now, AccessTTL)
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(c.AccessSecret)
}

// IssueRefresh signs a refresh token. Each one gets a unique id so two tokens
// issued in the same second never collide in storage.
func (c Config) IssueRefresh(subject string, now time.Time) (string, error) {
	claims := jwt.MapClaims{}
	c.stamp(claims, subject, TypeRefresh, now, RefreshTTL)
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(c.RefreshSecret)
}

// ParseAccess verifies an access token and returns its claims.
func (c Config) ParseAccess(raw string) (jwt.MapClaims, error) {
	return c.parse(raw, c.AccessSecret, TypeAccess)
}

// ParseRefresh verifies a refresh token and returns its subject.
func (c Config) ParseRefresh(raw string) (string, error) {
	claims, err := c.parse(raw, c.RefreshSecret, TypeRefresh)
	if err != nil {
		return "", err
	}
	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", ErrInvalid
	}
	return sub, nil
}

func (c Config) stamp(claims jwt.MapClaims, subject, typ string, now time.Time, ttl time.Duration) {
	claims["sub"] = subject
	claims["typ"] = typ
	claims["iss"] = c.Issuer
	claims["aud"] = c.Audience
	claims["iat"] = now.Unix()
	claims["exp"] = now.Add(ttl).Unix()
	claims["jti"] = newID()
}

func (c Config) parse(raw string, key []byte, typ string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(raw,
		func(t *jwt.Token) (any, error) { return key, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(c.Issuer),
		jwt.WithAudience(c.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("%w: %v", ErrInvalid, err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["typ"] != typ {
		return nil, ErrInvalid
	}
	return claims, nil
}

// Hash is how refresh tokens are stored: a leaked database must not hand out
// usable sessions.
func Hash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	return hex.EncodeToString(b)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

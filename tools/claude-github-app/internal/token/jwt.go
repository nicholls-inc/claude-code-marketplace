// Package token mints GitHub App installation access tokens, caches them on
// disk under 0600/0700 permissions, and resolves bot-user identities. It is
// the only package that touches the network or the user's secrets on disk.
package token

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// MaxJWTLifetime is the longest GitHub will accept for an App JWT.
const MaxJWTLifetime = 10 * time.Minute

// JWTLifetime is the lifetime we actually request — short enough that a single
// failed mint and retry stays well under the 10-minute cap.
const JWTLifetime = 9 * time.Minute

// ClockSkew is subtracted from `iat` to tolerate small clock drift.
const ClockSkew = 60 * time.Second

// BuildAppJWT signs a short-lived RS256 JWT used to mint installation tokens.
// `issuer` accepts either the App's client ID (e.g. "Iv23li...") or its
// numeric App ID — GitHub accepts both per the App JWT docs.
func BuildAppJWT(issuer string, pemBytes []byte) (string, error) {
	if issuer == "" {
		return "", errors.New("issuer (client_id) is required")
	}
	key, err := jwt.ParseRSAPrivateKeyFromPEM(pemBytes)
	if err != nil {
		return "", fmt.Errorf("parse private key: %w", err)
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-ClockSkew)),
		ExpiresAt: jwt.NewNumericDate(now.Add(JWTLifetime)),
		Issuer:    issuer,
	}
	t := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := t.SignedString(key)
	if err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	return signed, nil
}

// ReadPrivateKey reads a PEM file, enforcing 0600 permissions to satisfy the
// Properties §A confidentiality invariant. Returns a structured error so the
// caller can render the user-facing fix ("chmod 600 <path>").
func ReadPrivateKey(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat private key %s: %w", path, err)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return nil, &PrivateKeyPermsError{Path: path, Mode: perm}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key %s: %w", path, err)
	}
	return data, nil
}

// PrivateKeyPermsError is returned when a PEM file's mode is broader than
// 0600 (any bit set in `0o077`). The wrapper translates this into the §8
// "Private key file mode broader than 0600" failure mode.
type PrivateKeyPermsError struct {
	Path string
	Mode os.FileMode
}

func (e *PrivateKeyPermsError) Error() string {
	return fmt.Sprintf("private key file %s is mode %#o; must be 0600. Run: chmod 600 %s",
		e.Path, e.Mode, e.Path)
}

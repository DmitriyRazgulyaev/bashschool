package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Tokens struct{ secret []byte }

func New(secret string) (*Tokens, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT_SECRET must contain at least 32 bytes")
	}
	return &Tokens{secret: []byte(secret)}, nil
}
func (t *Tokens) Issue(id, role string) (string, error) {
	h := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(map[string]any{"sub": id, "role": role, "exp": time.Now().Add(12 * time.Hour).Unix()})
	p := base64.RawURLEncoding.EncodeToString(payload)
	msg := h + "." + p
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(msg))
	return msg + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func (t *Tokens) Parse(value string) (string, string, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return "", "", errors.New("invalid token")
	}
	msg := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(msg))
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(sig, mac.Sum(nil)) {
		return "", "", errors.New("invalid signature")
	}
	b, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", err
	}
	var claims struct {
		Sub  string `json:"sub"`
		Role string `json:"role"`
		Exp  int64  `json:"exp"`
	}
	if err = json.Unmarshal(b, &claims); err != nil || claims.Exp < time.Now().Unix() || claims.Sub == "" {
		return "", "", errors.New("expired or invalid token")
	}
	return claims.Sub, claims.Role, nil
}

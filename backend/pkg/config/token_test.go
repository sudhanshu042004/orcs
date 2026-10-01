package config

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestTokenRoundTrip(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	token, err := CreateToken(42, "a@example.com")
	if err != nil {
		t.Fatalf("CreateToken: %s", err)
	}

	payload, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken: %s", err)
	}
	if payload.Id != 42 || payload.Email != "a@example.com" {
		t.Errorf("round trip gave %+v", payload)
	}
}

func TestVerifyTokenRejectsWrongSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "the-real-secret")
	token, err := CreateToken(1, "a@example.com")
	if err != nil {
		t.Fatalf("CreateToken: %s", err)
	}

	t.Setenv("JWT_SECRET", "an-attackers-secret")
	if _, err := VerifyToken(token); err == nil {
		t.Error("a token signed with a different secret was accepted")
	}
}

func TestVerifyTokenRejectsTamperedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")
	token, err := CreateToken(1, "a@example.com")
	if err != nil {
		t.Fatalf("CreateToken: %s", err)
	}

	// Flip the last byte of the signature
	tampered := token[:len(token)-1] + string(token[len(token)-1]^1)
	if _, err := VerifyToken(tampered); err == nil {
		t.Error("a token with a broken signature was accepted")
	}
}

func TestVerifyTokenRejectsExpiredToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	// CreateToken always issues 7 days out, so an expired one has to be built by hand
	expired := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":    float64(1),
		"email": "a@example.com",
		"exp":   time.Now().Add(-time.Hour).Unix(),
	})
	signed, err := expired.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("signing: %s", err)
	}

	if _, err := VerifyToken(signed); err == nil {
		t.Error("an expired token was accepted")
	}
}

// VerifyToken's keyfunc returns the HMAC secret without checking which algorithm the
// token claims. This pins the behaviour: an unsigned token must never be accepted.
func TestVerifyTokenRejectsUnsignedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	unsigned := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"id":    float64(1),
		"email": "attacker@example.com",
		"exp":   time.Now().Add(time.Hour).Unix(),
	})
	signed, err := unsigned.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("signing: %s", err)
	}

	if _, err := VerifyToken(signed); err == nil {
		t.Error("a token with alg=none was accepted")
	}
}

func TestVerifyTokenRejectsMissingClaims(t *testing.T) {
	t.Setenv("JWT_SECRET", "test-secret")

	for name, claims := range map[string]jwt.MapClaims{
		"no email": {"id": float64(1), "exp": time.Now().Add(time.Hour).Unix()},
		"no id":    {"email": "a@example.com", "exp": time.Now().Add(time.Hour).Unix()},
	} {
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
		if err != nil {
			t.Fatalf("%s: signing: %s", name, err)
		}
		if _, err := VerifyToken(signed); err == nil {
			t.Errorf("%s: token was accepted", name)
		}
	}
}

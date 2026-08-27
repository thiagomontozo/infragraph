package security

import (
	"crypto/ed25519"
	"github.com/thiagomontozo/infragraph/internal/domain"
	"testing"
	"time"
)

func TestPasswordAndEnvelope(t *testing.T) {
	h, e := HashPassword("correct horse battery staple")
	if e != nil || !VerifyPassword(h, "correct horse battery staple") || VerifyPassword(h, "wrong password!") {
		t.Fatal("password invariant failed")
	}
	s, _ := NewSecretStore(make([]byte, 32), 1)
	v, _ := s.Encrypt([]byte("secret"), nil)
	got, e := s.Decrypt(v)
	if e != nil || string(got) != "secret" {
		t.Fatal("envelope roundtrip failed")
	}
	v.Ciphertext = "AAAA"
	if _, e = s.Decrypt(v); e == nil {
		t.Fatal("tampering accepted")
	}
}

func TestKeyedTokenHashIsBoundToSecretAndToken(t *testing.T) {
	one := KeyedTokenHash("secret-one", "token")
	if one == KeyedTokenHash("secret-two", "token") || one == KeyedTokenHash("secret-one", "other") {
		t.Fatal("keyed token hash is not bound to both inputs")
	}
}
func TestSnapshotSignatureMutation(t *testing.T) {
	pub, priv, _ := GenerateCollectorKey()
	s := domain.SnapshotEnvelope{ProtocolVersion: "1.0", SnapshotID: "s1", OrganizationID: "o1", CollectorID: "c1", Sequence: 1, CompletedAt: time.Now()}
	s, e := SignSnapshot(s, priv)
	if e != nil || VerifySnapshot(s, pub) != nil {
		t.Fatal("valid snapshot rejected")
	}
	s.Sequence++
	if VerifySnapshot(s, pub) == nil {
		t.Fatal("mutated snapshot accepted")
	}
	wrong, _, _ := ed25519.GenerateKey(nil)
	if VerifySnapshot(s, wrong) == nil {
		t.Fatal("wrong collector accepted")
	}
}

func TestTOTPAgainstRFCVector(t *testing.T) {
	// RFC 6238 uses the RFC 4226 SHA-1 secret. At Unix time 59 the moving
	// counter is 1 and the six-digit truncation is 287082.
	const secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	if !VerifyTOTP(secret, "287082", time.Unix(59, 0)) {
		t.Fatal("valid TOTP vector rejected")
	}
	if VerifyTOTP(secret, "287083", time.Unix(59, 0)) {
		t.Fatal("invalid TOTP accepted")
	}
}

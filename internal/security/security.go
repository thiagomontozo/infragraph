package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/thiagomontozo/infragraph/internal/domain"
	"golang.org/x/crypto/argon2"
)

func HashPassword(password string) (string, error) {
	if len(password) < 12 {
		return "", errors.New("password must contain at least 12 characters")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	sum := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=2$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(sum)), nil
}
func VerifyPassword(encoded, password string) bool {
	p := strings.Split(encoded, "$")
	if len(p) != 6 {
		return false
	}
	salt, e1 := base64.RawStdEncoding.DecodeString(p[4])
	expected, e2 := base64.RawStdEncoding.DecodeString(p[5])
	if e1 != nil || e2 != nil {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, 3, 64*1024, 2, 32)
	return hmac.Equal(got, expected)
}

type SecretEnvelope struct {
	Ciphertext, Nonce string
	KeyVersion        int
	Metadata          map[string]string
}
type SecretStore struct {
	key     []byte
	version int
}

func NewSecretStore(key []byte, version int) (*SecretStore, error) {
	if len(key) != 32 {
		return nil, errors.New("AES-256-GCM requires exactly 32 key bytes")
	}
	return &SecretStore{append([]byte(nil), key...), version}, nil
}
func (s *SecretStore) Encrypt(plaintext []byte, metadata map[string]string) (SecretEnvelope, error) {
	b, e := aes.NewCipher(s.key)
	if e != nil {
		return SecretEnvelope{}, e
	}
	g, e := cipher.NewGCM(b)
	if e != nil {
		return SecretEnvelope{}, e
	}
	n := make([]byte, g.NonceSize())
	if _, e = rand.Read(n); e != nil {
		return SecretEnvelope{}, e
	}
	aad := []byte(strconv.Itoa(s.version))
	c := g.Seal(nil, n, plaintext, aad)
	return SecretEnvelope{base64.StdEncoding.EncodeToString(c), base64.StdEncoding.EncodeToString(n), s.version, metadata}, nil
}
func (s *SecretStore) Decrypt(e SecretEnvelope) ([]byte, error) {
	if e.KeyVersion != s.version {
		return nil, errors.New("unsupported key version")
	}
	c, e1 := base64.StdEncoding.DecodeString(e.Ciphertext)
	n, e2 := base64.StdEncoding.DecodeString(e.Nonce)
	if e1 != nil || e2 != nil {
		return nil, errors.New("invalid envelope encoding")
	}
	b, _ := aes.NewCipher(s.key)
	g, _ := cipher.NewGCM(b)
	return g.Open(nil, n, c, []byte(strconv.Itoa(s.version)))
}

func GenerateCollectorKey() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	return ed25519.GenerateKey(rand.Reader)
}
func SignSnapshot(s domain.SnapshotEnvelope, key ed25519.PrivateKey) (domain.SnapshotEnvelope, error) {
	s.SignatureAlgorithm = "Ed25519"
	s.Signature = ""
	s.ContentHash = ""
	b, e := s.SigningBytes()
	if e != nil {
		return s, e
	}
	h := sha256.Sum256(b)
	s.ContentHash = hex.EncodeToString(h[:])
	b, e = s.SigningBytes()
	if e != nil {
		return s, e
	}
	s.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(key, b))
	return s, nil
}
func VerifySnapshot(s domain.SnapshotEnvelope, key ed25519.PublicKey) error {
	if s.SignatureAlgorithm != "Ed25519" {
		return errors.New("unsupported signature algorithm")
	}
	sig, e := base64.StdEncoding.DecodeString(s.Signature)
	if e != nil {
		return errors.New("invalid signature encoding")
	}
	b, e := s.SigningBytes()
	if e != nil {
		return e
	}
	if !ed25519.Verify(key, b, sig) {
		return errors.New("invalid snapshot signature")
	}
	claimed := s.ContentHash
	s.Signature = ""
	s.ContentHash = ""
	hashBytes, e := s.SigningBytes()
	if e != nil {
		return e
	}
	h := sha256.Sum256(hashBytes)
	if subtle.ConstantTimeCompare([]byte(claimed), []byte(hex.EncodeToString(h[:]))) != 1 {
		return errors.New("invalid snapshot content hash")
	}
	return nil
}
func TokenHash(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func KeyedTokenHash(secret, token string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))
}
func RandomToken(bytes int) (string, error) {
	b := make([]byte, bytes)
	if _, e := rand.Read(b); e != nil {
		return "", e
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func VerifyTOTP(secret, code string, now time.Time) bool {
	raw, e := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.ReplaceAll(secret, " ", "")))
	if e != nil {
		return false
	}
	for d := -1; d <= 1; d++ {
		counter := uint64(now.Unix()/30) + uint64(d)
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, counter)
		m := hmac.New(sha1.New, raw)
		m.Write(buf)
		sum := m.Sum(nil)
		off := sum[len(sum)-1] & 15
		if int(off)+4 > len(sum) {
			continue
		}
		num := binary.BigEndian.Uint32(sum[off:off+4]) & 0x7fffffff
		if fmt.Sprintf("%06d", num%1000000) == code {
			return true
		}
	}
	return false
}

func AuditHash(previous string, payload []byte) string {
	h := sha256.New()
	h.Write([]byte(previous))
	h.Write([]byte{0})
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

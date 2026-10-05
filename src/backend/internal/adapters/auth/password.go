// Package auth contains password hashing and token helpers (driven adapters).
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/youmei295/something-something/src/backend/internal/ports"
)

// Argon2Hasher implements ports.PasswordHasher using argon2id, the recommended
// password-hashing function for new systems.
type Argon2Hasher struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

// NewArgon2Hasher returns a hasher with conservative defaults suitable for a
// small VPS (64 MiB memory, 1 iteration, 4 lanes).
func NewArgon2Hasher() *Argon2Hasher {
	return &Argon2Hasher{time: 1, memory: 64 * 1024, threads: 4, keyLen: 32}
}

func (h *Argon2Hasher) Hash(plain string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(plain), salt, h.time, h.memory, h.threads, h.keyLen)
	// PHC string format, e.g. $argon2id$v=19$m=65536,t=1,p=4$<salt>$<hash>
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, h.memory, h.time, h.threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

func (h *Argon2Hasher) Verify(encoded, plain string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errors.New("invalid hash format")
	}

	var memory uint32
	var t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &t, &p); err != nil {
		return false, fmt.Errorf("parse argon2 params: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}

	got := argon2.IDKey([]byte(plain), salt, t, memory, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

var _ ports.PasswordHasher = (*Argon2Hasher)(nil)

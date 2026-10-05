package auth

import "testing"

func TestArgon2Hasher_HashAndVerify(t *testing.T) {
	h := NewArgon2Hasher()

	encoded, err := h.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if encoded == "correct horse battery staple" {
		t.Fatal("password was not hashed")
	}

	ok, err := h.Verify(encoded, "correct horse battery staple")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("correct password rejected")
	}

	ok, err = h.Verify(encoded, "wrong password")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if ok {
		t.Fatal("incorrect password accepted")
	}
}

func TestArgon2Hasher_UniqueSalts(t *testing.T) {
	h := NewArgon2Hasher()
	a, _ := h.Hash("same")
	b, _ := h.Hash("same")
	if a == b {
		t.Fatal("hashes of the same password must differ (unique salts)")
	}
}

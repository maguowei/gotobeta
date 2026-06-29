package security

import "testing"

func TestRandomSecretGeneratorHashesToken(t *testing.T) {
	gen := NewRandomSecretGenerator()
	token, err := gen.NewToken()
	if err != nil {
		t.Fatalf("NewToken() error = %v", err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if got := gen.HashToken(token); len(got) != 64 {
		t.Fatalf("hash length = %d, want 64", len(got))
	}
}

func TestBcryptPasswordHasherComparesPassword(t *testing.T) {
	hasher := NewBcryptPasswordHasher(4)
	hash, alg, err := hasher.Hash("password-123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if alg != bcryptAlgorithm {
		t.Fatalf("alg = %q, want %q", alg, bcryptAlgorithm)
	}
	if err := hasher.Compare(hash, "password-123"); err != nil {
		t.Fatalf("Compare() error = %v", err)
	}
	if err := hasher.Compare(hash, "wrong-password"); err == nil {
		t.Fatal("Compare() error = nil, want mismatch")
	}
}

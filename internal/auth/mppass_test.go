package auth

import "testing"

func TestMppassMatchesClassicAlgorithm(t *testing.T) {
	t.Parallel()

	if got := Mppass("alice", "salt"); got != "36264c5ce84d59a4da2a6716eb0f3ff0" {
		t.Fatalf("Mppass = %q", got)
	}
	if !ValidMppass("alice", "salt", "36264C5CE84D59A4DA2A6716EB0F3FF0") {
		t.Fatal("ValidMppass rejected uppercase token")
	}
	if ValidMppass("alice", "salt", "bad") {
		t.Fatal("ValidMppass accepted invalid token")
	}
}

func TestGenerateSalt(t *testing.T) {
	t.Parallel()

	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt returned error: %v", err)
	}
	if !ValidSalt(salt) {
		t.Fatalf("GenerateSalt returned invalid salt %q", salt)
	}
	if len(salt) != SaltLength {
		t.Fatalf("GenerateSalt length = %d, want %d", len(salt), SaltLength)
	}
}

func TestValidSaltRejectsUnsafeValues(t *testing.T) {
	t.Parallel()

	for _, salt := range []string{"short", "has space 123456789", "symbols-not-allowed"} {
		if ValidSalt(salt) {
			t.Fatalf("ValidSalt(%q) = true", salt)
		}
	}
}

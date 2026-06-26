package world

import "testing"

func TestMultiManagerBasic(t *testing.T) {
	mm := NewMultiManager()
	main := NewManager()
	mm.SetMain("Main", main, "/tmp/main.swld")

	if mm.MainName() != "Main" {
		t.Fatalf("MainName = %q, want %q", mm.MainName(), "Main")
	}
	if mm.MainManager() != main {
		t.Fatal("MainManager mismatch")
	}
	if !mm.Has("main") { // case-insensitive
		t.Fatal("Has(main) should be true (case-insensitive)")
	}
	if mm.Get("MAIN") != main {
		t.Fatal("Get(MAIN) should return main manager (case-insensitive)")
	}
	if got := mm.Path("MAIN"); got != "/tmp/main.swld" {
		t.Fatalf("Path(MAIN) = %q", got)
	}

	second := NewManager()
	mm.Add("Other", second, "/tmp/other.swld")
	names := mm.Names()
	if len(names) != 2 {
		t.Fatalf("Names() = %v, want 2 entries", names)
	}

	if !mm.Remove("other") {
		t.Fatal("Remove(other) should succeed")
	}
	if mm.Has("Other") {
		t.Fatal("Has(Other) should be false after remove")
	}

	if !mm.Remove("main") {
		t.Fatal("Remove(main) should succeed at MultiManager level")
	}
	if mm.Has("main") {
		t.Fatal("Has(main) should be false after remove")
	}
}

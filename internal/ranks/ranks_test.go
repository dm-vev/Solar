package ranks

import "testing"

func TestDefaultRanks(t *testing.T) {
	r := NewRegistry()
	all := r.All()
	if len(all) != 7 {
		t.Fatalf("expected 7 default ranks, got %d", len(all))
	}
}

func TestGetByName(t *testing.T) {
	r := NewRegistry()
	guest := r.Get("guest")
	if guest == nil || guest.Permission != PermGuest {
		t.Fatal("guest rank not found")
	}
	op := r.Get("Operator") // case-insensitive
	if op == nil || op.Permission != PermOperator {
		t.Fatal("Operator rank not found (case-insensitive)")
	}
}

func TestGetByPerm(t *testing.T) {
	r := NewRegistry()
	builder := r.GetByPerm(PermBuilder)
	if builder == nil || builder.Name != "builder" {
		t.Fatal("builder rank not found by perm")
	}
}

func TestHasRank(t *testing.T) {
	if !HasRank(PermOperator, PermGuest) {
		t.Fatal("operator should have guest rank")
	}
	if HasRank(PermGuest, PermOperator) {
		t.Fatal("guest should not have operator rank")
	}
}

func TestIsOperator(t *testing.T) {
	if !IsOperator(PermOperator) {
		t.Fatal("operator should be operator")
	}
	if IsOperator(PermGuest) {
		t.Fatal("guest should not be operator")
	}
}

func TestDefaultRank(t *testing.T) {
	r := NewRegistry()
	dr := r.DefaultRank()
	if dr == nil || dr.Name != "guest" {
		t.Fatal("default rank should be guest")
	}
}

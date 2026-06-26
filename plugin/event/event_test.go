package event

import "testing"

func TestUnregisterRemovesHandler(t *testing.T) {
	t.Parallel()

	ev := NewEvent[int]()
	calls := 0
	handler := func(*Context, int) {
		calls++
	}
	ev.Register(handler, PriorityNormal)
	ev.Fire(1)
	ev.Unregister(handler)
	ev.Fire(1)

	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
	if ev.HasHandlers() {
		t.Fatal("HasHandlers returned true after Unregister")
	}
}

func TestFireRecoversHandlerPanic(t *testing.T) {
	t.Parallel()

	ev := NewEvent[int]()
	ev.Register(func(*Context, int) {
		panic("boom")
	}, PriorityHigh)
	called := false
	ev.Register(func(*Context, int) {
		called = true
	}, PriorityLow)

	ctx := ev.Fire(1)
	if !ctx.Cancelled() {
		t.Fatal("Fire did not cancel after handler panic")
	}
	if called {
		t.Fatal("Fire called lower-priority handler after panic")
	}
}

func TestFireUsesHandlerSnapshot(t *testing.T) {
	t.Parallel()

	ev := NewEvent[int]()
	calls := 0
	ev.Register(func(*Context, int) {
		calls++
		ev.Register(func(*Context, int) {
			calls++
		}, PriorityLow)
	}, PriorityHigh)

	ev.Fire(1)
	if calls != 1 {
		t.Fatalf("calls after first Fire = %d, want 1", calls)
	}
	ev.Fire(1)
	if calls != 3 {
		t.Fatalf("calls after second Fire = %d, want 3", calls)
	}
}

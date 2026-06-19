package worker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolExecutesSubmittedJobs(t *testing.T) {
	t.Parallel()

	pool := NewPool(context.Background(), 2)
	defer pool.Close()

	done := make(chan struct{})
	if !pool.Submit(func() { close(done) }) {
		t.Fatal("Submit returned false")
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("job was not executed")
	}
}

func TestPoolRejectsAfterClose(t *testing.T) {
	t.Parallel()

	pool := NewPool(context.Background(), 1)
	pool.Close()
	pool.Close()
	if pool.Submit(func() {}) {
		t.Fatal("Submit returned true after Close")
	}
}

func TestPoolRunsJobsConcurrently(t *testing.T) {
	t.Parallel()

	pool := NewPool(context.Background(), 4)
	defer pool.Close()

	var count atomic.Int32
	done := make(chan struct{})
	for i := 0; i < 4; i++ {
		if !pool.Submit(func() {
			if count.Add(1) == 4 {
				close(done)
			}
		}) {
			t.Fatal("Submit returned false")
		}
	}

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatalf("only %d jobs executed", count.Load())
	}
}

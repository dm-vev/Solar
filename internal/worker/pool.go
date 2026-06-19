package worker

import (
	"context"
	"runtime"
	"sync"
)

// Pool executes background jobs on a fixed worker budget.
type Pool struct {
	Size   int
	jobs   chan func()
	ctx    context.Context
	cancel context.CancelFunc
	once   sync.Once
	mu     sync.RWMutex
	closed bool
	wg     sync.WaitGroup
}

// NewPool starts a fixed-size worker pool derived from ctx. When ctx is
// canceled, the pool stops accepting jobs and workers exit.
func NewPool(ctx context.Context, size int) *Pool {
	if ctx == nil {
		ctx = context.Background()
	}
	if size < 1 {
		size = runtime.NumCPU()
	}
	ctx, cancel := context.WithCancel(ctx)
	p := &Pool{
		Size:   size,
		jobs:   make(chan func(), size*4),
		ctx:    ctx,
		cancel: cancel,
	}
	for i := 0; i < p.Size; i++ {
		p.wg.Add(1)
		go p.loop()
	}
	return p
}

// Submit queues a job for execution.
func (p *Pool) Submit(job func()) bool {
	if p == nil || job == nil {
		return false
	}
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.closed {
		return false
	}
	select {
	case p.jobs <- job:
		return true
	case <-p.ctx.Done():
		return false
	default:
		return false
	}
}

// Close stops the pool and waits for workers to exit.
func (p *Pool) Close() {
	if p == nil {
		return
	}
	p.once.Do(func() {
		p.mu.Lock()
		p.closed = true
		p.mu.Unlock()
		p.cancel()
	})
	p.wg.Wait()
}

func (p *Pool) loop() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case job, ok := <-p.jobs:
			if !ok {
				return
			}
			if job != nil {
				job()
			}
		}
	}
}

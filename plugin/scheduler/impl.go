package scheduler

import (
	"sync"
	"sync/atomic"
	"time"
)

// schedulerImpl implements Scheduler. Safe for concurrent use.
type schedulerImpl struct {
	mu     sync.Mutex
	nextID atomic.Uint64
	tasks  map[uint64]*Task
	tickCh chan struct{}
}

var _ Scheduler = (*schedulerImpl)(nil)

// New creates a new scheduler instance.
func New() *schedulerImpl {
	return &schedulerImpl{tasks: make(map[uint64]*Task)}
}

// Default is the default scheduler instance.
var Default = New()

func (s *schedulerImpl) After(delay Duration, fn func()) {
	time.AfterFunc(delay, fn)
}

func (s *schedulerImpl) Every(interval Duration, fn func()) *Task {
	t := &Task{ID: s.nextID.Add(1)}
	stop := make(chan struct{})
	ticker := time.NewTicker(interval)
	t.Stop = func() { ticker.Stop(); close(stop) }
	s.mu.Lock()
	s.tasks[t.ID] = t
	s.mu.Unlock()
	go func() {
		for {
			select {
			case <-ticker.C:
				select {
				case <-stop:
					return
				default:
				}
				fn()
			case <-stop:
				return
			}
		}
	}()
	return t
}

func (s *schedulerImpl) NextTick(fn func()) {
	s.mu.Lock()
	ch := s.tickCh
	s.mu.Unlock()
	if ch == nil {
		time.AfterFunc(50*time.Millisecond, fn)
		return
	}
	go func() {
		select {
		case <-ch:
			fn()
		case <-time.After(50 * time.Millisecond):
			fn()
		}
	}()
}

// Tick fires all pending next-tick callbacks. Called by the server tick loop.
func (s *schedulerImpl) Tick() {
	s.mu.Lock()
	old := s.tickCh
	s.tickCh = make(chan struct{})
	s.mu.Unlock()
	if old != nil {
		close(old)
	}
}

func (s *schedulerImpl) Cancel(task *Task) {
	if task == nil {
		return
	}
	s.mu.Lock()
	if task.Cancelled {
		s.mu.Unlock()
		return
	}
	task.Cancelled = true
	delete(s.tasks, task.ID)
	stop := task.Stop
	s.mu.Unlock()
	if stop != nil {
		stop()
	}
}

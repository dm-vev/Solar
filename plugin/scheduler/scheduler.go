// Package scheduler defines the Scheduler interface that plugins use
// to schedule delayed and repeating tasks.
package scheduler

import "time"

// Scheduler lets plugins schedule delayed and repeating tasks.
type Scheduler interface {
	// After runs fn once after delay.
	After(delay Duration, fn func())
	// Every runs fn repeatedly every interval until the task is cancelled.
	Every(interval Duration, fn func()) *Task
	// NextTick runs fn on the next server tick.
	NextTick(fn func())
	// Cancel cancels a scheduled task.
	Cancel(task *Task)
}

// Task represents a scheduled operation.
type Task struct {
	ID        uint64
	Cancelled bool
	Stop      func() // called by the scheduler to stop the underlying ticker/timer
}

// Duration is re-exported from time.Duration for plugin convenience.
type Duration = time.Duration

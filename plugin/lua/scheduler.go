//go:build lua

package lua

import (
	glua "github.com/yuin/gopher-lua"

	"github.com/solar-mc/solar/plugin"
	"github.com/solar-mc/solar/plugin/scheduler"
)

func checkScheduler(L *glua.LState, idx int) plugin.Scheduler {
	if v, ok := udValue(L, idx).(plugin.Scheduler); ok {
		return v
	}
	L.ArgError(idx, "expected scheduler")
	return nil
}

func checkTask(L *glua.LState, idx int) *scheduler.Task {
	if v, ok := udValue(L, idx).(*scheduler.Task); ok {
		return v
	}
	L.ArgError(idx, "expected task")
	return nil
}

var schedulerMethods = map[string]glua.LGFunction{
	// after(ms, fn) — no return (fire-and-forget)
	"after": func(L *glua.LState) int {
		sched := checkScheduler(L, 1)
		fn := L.CheckFunction(2)
		sched.After(durationFromMS(L, 2), func() {
			defer func() { _ = recover() }()
			_ = L.CallByParam(glua.P{Fn: fn, NRet: 0, Protect: true})
		})
		return 0
	},
	// every(ms, fn) -> task
	"every": func(L *glua.LState) int {
		sched := checkScheduler(L, 1)
		fn := L.CheckFunction(2)
		task := sched.Every(durationFromMS(L, 2), func() {
			defer func() { _ = recover() }()
			_ = L.CallByParam(glua.P{Fn: fn, NRet: 0, Protect: true})
		})
		L.Push(wrapUD(L, typeTask, &taskWrapper{task: task, sched: sched}))
		return 1
	},
	// next_tick(fn) — no return
	"next_tick": func(L *glua.LState) int {
		sched := checkScheduler(L, 1)
		fn := L.CheckFunction(2)
		sched.NextTick(func() {
			defer func() { _ = recover() }()
			_ = L.CallByParam(glua.P{Fn: fn, NRet: 0, Protect: true})
		})
		return 0
	},
	// cancel(task) — cancel a task returned by every()
	"cancel": func(L *glua.LState) int {
		sched := checkScheduler(L, 1)
		tw, ok := udValue(L, 2).(*taskWrapper)
		if !ok {
			L.ArgError(2, "expected task")
			return 0
		}
		sched.Cancel(tw.task)
		return 0
	},
}

// taskMethods — standalone task table with cancel (for convenience).
// Since cancel lives on scheduler, we store the scheduler on the task userdata
// via a wrapper struct.
type taskWrapper struct {
	task  *scheduler.Task
	sched plugin.Scheduler
}

var taskMethods = map[string]glua.LGFunction{
	"cancel": func(L *glua.LState) int {
		tw, ok := udValue(L, 1).(*taskWrapper)
		if !ok {
			L.ArgError(1, "expected task")
			return 0
		}
		tw.sched.Cancel(tw.task)
		return 0
	},
}

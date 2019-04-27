package loop

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	lua "github.com/yuin/gopher-lua"
)

// Task is a task that should be executed inside the loop
type Task func(*lua.LState)

// Loop is an async event loop for lua
type Loop interface {
	// Start starts the loop
	Start(context.Context) error

	// Schedule a new task to be executed inside the loop
	Schedule(Task)

	// ScheduleAndWait schedules a task on the loop and waits for it to finish
	ScheduleAndWait(Task)

	// Stop the loop
	Stop()

	// Wait for the loop to finish
	Wait()
}

// Options used when creating a new event loop
type Options struct {
	// InitVM is called with the new lua State before the event loop is initialized
	InitVM func(*lua.LState) error
}

// loop is the actual implementation of the Loop interface
type loop struct {
	vm *lua.LState

	queue *Queue

	wg      sync.WaitGroup
	running bool
}

// LGet returns the current event loop from the given VM
func LGet(state *lua.LState) Loop {
	g := state.GetGlobal("__loop")
	ud, ok := g.(*lua.LUserData)
	if !ok {
		state.RaiseError("Failed to find event loop global")
		return nil
	}

	l, ok := ud.Value.(Loop)
	if !ok {
		state.RaiseError("Failed to find event loop")
		return nil
	}

	return l
}

// New returns a new event loop
func New(opts *Options) (Loop, error) {
	vm := lua.NewState()

	l := &loop{
		vm:    vm,
		queue: NewQueue("default", "jobs"),
	}

	ud := vm.NewUserData()
	ud.Value = l

	vm.SetGlobal("__loop", ud)
	vm.SetGlobal("__schedule", vm.NewFunction(l.scheduleLua))

	if opts != nil {
		if opts.InitVM != nil {
			if err := opts.InitVM(vm); err != nil {
				return nil, err
			}
		}
	}

	return l, nil
}

// Start starts the loop and implements Loop.Start
func (l *loop) Start(ctx context.Context) error {
	l.wg.Add(1)
	go l.run(ctx)

	return nil
}

func (l *loop) scheduleLua(state *lua.LState) int {
	fn := state.CheckFunction(1)

	l.Schedule(func(state *lua.LState) {
		state.CallByParam(lua.P{
			Fn:   fn,
			NRet: 0,
		})
	})

	return 0
}

// Schedule schedules a task to be executed on the loop
func (l *loop) Schedule(task Task) {
	l.wg.Add(1)

	l.queue.Push(func(L *lua.LState) {
		defer l.wg.Done()
		task(L)
	})
}

// ScheduleAndWait schedules a task and waits for it to be executed
func (l *loop) ScheduleAndWait(task Task) {
	ch := make(chan bool)

	l.Schedule(func(vm *lua.LState) {
		defer func() {
			ch <- true
		}()
		task(vm)
	})

	<-ch
}

// Stop asks the loop to stop
func (l *loop) Stop() {
	l.Schedule(func(_ *lua.LState) {
		l.running = false
	})
}

// Wait waits for the loop to stop
func (l *loop) Wait() {
	l.wg.Wait()
}

func (l *loop) run(ctx context.Context) {
	defer l.wg.Done()
	l.running = true

	for l.running {
		job := l.queue.PopWait(ctx)
		if ctx.Err() != nil {
			l.running = false
			break
		}

		l.runJob(job)
	}
}

func (l *loop) runJob(task Task) {
	timer := prometheus.NewTimer(prometheus.ObserverFunc(jobExecDuration.With(prometheus.Labels{
		"loop": "default",
	}).Observe))
	defer timer.ObserveDuration()

	task(l.vm)
}

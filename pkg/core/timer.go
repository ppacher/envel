package core

import (
	"fmt"
	"sync"
	"time"

	"github.com/ppacher/envel/pkg/callback"
	"github.com/ppacher/envel/pkg/loop"
	lua "github.com/yuin/gopher-lua"
)

const timerTypeName = "timer"

// AddTimer adds the exec package to the lua table m
func AddTimer(L *lua.LState, m *lua.LTable) {
	t := L.NewTable()

	L.SetMetatable(t, L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"__call": newTimer,
	}))

	typeMt := L.NewTypeMetatable(timerTypeName)

	L.SetField(typeMt, "__index", L.SetFuncs(L.NewTable(), timerTypeAPI))
	t.RawSetString("__timer_mt", typeMt)

	m.RawSetString("timer", t)
}

var timerTypeAPI = map[string]lua.LGFunction{
	"start":      timerStart,
	"stop":       timerStop,
	"again":      timerAgain,
	"is_started": timerStarted,
}

// TimerOptions holds configuration options for a new timer
type TimerOptions struct {
	// Timeout for the timer. After each timeout, the Callback function is invoked
	// this field MUST be set
	Timeout time.Duration

	// Autostart defines whether the time should start immediately
	Autostart bool

	// CallNow defines whether the callback function should be triggered immediately
	CallNow bool

	// Callback is the actual callback function to invoke
	Callback callback.Callback

	// SingleShot configures the timer to automaticall stop after the first timeout
	SingleShot bool
}

type Timer struct {
	*TimerOptions

	wg     sync.WaitGroup
	stopCh chan struct{}
	lock   sync.Mutex
}

// Start starts the timer if its not running
func (t *Timer) Start() {
	t.lock.Lock()
	defer t.lock.Unlock()

	if t.stopCh == nil {
		t.stopCh = make(chan struct{}, 1)
		t.wg.Add(1)
		go t.run()
	}
}

// Init initializes the timer. If CallNow is set, Init() will try to execute the callback.
// In this case, if L is provided, the callback function will run immediately. If L is nil,
// the callback is scheduled on the loop and Init() will wait for it to finish. Note that
// the caller MUST provide L if it's currently running inside the loop. Otherwise
// it will deadlock. The returned error will always be nil if CallNow is set to false. If
// the callback is invoked immediately, the returned error may be a lua error
func (t *Timer) Init(L *lua.LState) error {
	if t.Autostart {
		t.Start()
	}

	var err error

	if t.CallNow {
		if L != nil {
			err = L.CallByParam(lua.P{
				Fn:      t.Callback.Callable(),
				NRet:    0,
				Protect: true,
			})
		} else {
			// schedule an immediate invocation of the callback
			// and wait for it to finish
			err = <-t.Callback.Do()
		}
	}

	return err
}

func (t *Timer) run() {
	t.lock.Lock()
	ch := t.stopCh
	t.lock.Unlock()

	defer func() {
		t.lock.Lock()
		defer t.lock.Unlock()
		t.stopCh = nil
		t.wg.Done()
	}()

	ticker := time.NewTicker(t.Timeout)

	for {
		select {
		case _, _ = <-ch:
			return
		case <-ticker.C:
			go func() { <-t.Callback.Do() }()

			if t.SingleShot {
				go t.Stop()
			}
		}
	}
}

// Stop stops the timer if its running
func (t *Timer) Stop() {
	t.lock.Lock()
	if t.stopCh != nil {
		close(t.stopCh)
	}
	t.lock.Unlock()

	t.wg.Wait()
}

// IsStarted returns true if the timer is started
func (t *Timer) IsStarted() bool {
	t.lock.Lock()
	defer t.lock.Unlock()

	return t.stopCh != nil
}

// Again restart the time. This is equalent to calling .Stop() and .Start()
func (t *Timer) Again() {
	t.Stop()
	t.Start()
}

// NewTimer creates a new timer for the given lua.LState. Note that the caller
// is responsible for initializing the timer by calling .Init(). Passing the
// timer-object to lua without calling .Init() may have strange side-effects
func NewTimer(L *lua.LState, opts TimerOptions) (*lua.LUserData, *Timer) {
	timer := &Timer{
		TimerOptions: &opts,
	}

	ud := L.NewUserData()
	ud.Value = timer
	L.SetMetatable(ud, L.GetTypeMetatable(timerTypeName))

	return ud, timer
}

// newTimer creates a new time inside the Lua VM
func newTimer(L *lua.LState) int {
	argsTable := L.CheckTable(2)

	opts := TimerOptions{}

	timeout := argsTable.RawGetString("timeout")
	if val, ok := timeout.(lua.LNumber); ok {
		opts.Timeout = time.Duration(float64(val) * float64(time.Second))
	} else {
		L.ArgError(1, fmt.Sprintf("timeout must be a number. got: %s (%v)", timeout.Type().String(), timeout))
	}

	autostart := argsTable.RawGetString("autostart")
	if autostart != lua.LNil {
		if val, ok := autostart.(lua.LBool); ok {
			opts.Autostart = bool(val)
		} else {
			L.ArgError(1, "autostart must be boolean or nil")
		}
	}

	callNow := argsTable.RawGetString("call_now")
	if callNow != lua.LNil {
		if val, ok := callNow.(lua.LBool); ok {
			opts.CallNow = bool(val)
		} else {
			L.ArgError(1, "call_now must be boolean or nil")
		}
	}

	cb := argsTable.RawGetString("callback")
	if fn, ok := cb.(*lua.LFunction); ok {
		opts.Callback = callback.New(fn, loop.LGet(L))
	} else {
		L.ArgError(1, "callback must be set to a function")
	}

	singleShot := argsTable.RawGetString("single_shot")
	if singleShot != lua.LNil {
		if val, ok := singleShot.(lua.LBool); ok {
			opts.SingleShot = bool(val)
		} else {
			L.ArgError(1, "single_shot must be boolean or nil")
		}
	}

	ud, timer := NewTimer(L, opts)

	err := timer.Init(L)
	if err != nil {
		L.RaiseError(err.Error())
	}

	L.Push(ud)
	return 1
}

func checkTimer(L *lua.LState) *Timer {
	ud := L.CheckUserData(1)
	if t, ok := ud.Value.(*Timer); ok {
		return t
	}

	L.ArgError(1, "Expected a timer object")
	return nil
}

func timerStart(L *lua.LState) int {
	t := checkTimer(L)
	t.Start()
	return 0
}

func timerStop(L *lua.LState) int {
	t := checkTimer(L)
	t.Stop()
	return 0
}

func timerAgain(L *lua.LState) int {
	t := checkTimer(L)
	t.Again()
	return 0
}

func timerStarted(L *lua.LState) int {
	t := checkTimer(L)
	L.Push(lua.LBool(t.IsStarted()))
	return 1
}

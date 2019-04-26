package callback

import (
	"log"

	"github.com/ppacher/envel/pkg/loop"
	lua "github.com/yuin/gopher-lua"
)

// Callback represents a lua callback that can be scheduled on the event loop
type Callback interface {
	// Do schedules the callback
	Do(args ...lua.LValue) <-chan error

	// From schedules the callback to be executed on the loop
	// The passed function is executed just before the callback and can
	// be used to construct lua objects
	From(func(*lua.LState) []lua.LValue) <-chan error

	// Callable returns the callbacks callable. Use with care
	Callable() *lua.LFunction

	// BindChannelErrors binds the callback to the given channel and executes/schedules
	// a callback invocation per message. For each execution, either nil or an error is
	// returned
	BindChannelErrors(ch <-chan []lua.LValue) <-chan error

	// BindChannel works like BindChannelErrors but ignores any errors returned by
	// a callback invocation
	BindChannel(ch <-chan []lua.LValue)
}

type callback struct {
	callable *lua.LFunction
	loop     loop.Loop
}

// New returns a new callback
func New(callable *lua.LFunction, loop loop.Loop) Callback {
	return &callback{
		callable: callable,
		loop:     loop,
	}
}

func (cb *callback) Callable() *lua.LFunction {
	return cb.callable
}

func (cb *callback) From(fn func(L *lua.LState) []lua.LValue) <-chan error {
	err := make(chan error, 1)

	if cb == nil {
		err <- nil
		return err
	}

	cb.loop.Schedule(func(state *lua.LState) {
		args := fn(state)
		e := state.CallByParam(lua.P{
			Fn:      cb.callable,
			NRet:    0,
			Protect: true,
		}, args...)

		if e != nil {
			log.Printf("error in callback: %s\n", e.Error())
		}
		err <- e
	})

	return err
}

// Do executes the callback
func (cb *callback) Do(args ...lua.LValue) <-chan error {
	return cb.From(func(_ *lua.LState) []lua.LValue {
		return args
	})
}

func (cb *callback) BindChannelErrors(ch <-chan []lua.LValue) <-chan error {
	errors := make(chan error)
	go func() {
		defer close(errors)
		for args := range ch {
			err := <-cb.Do(args...)
			errors <- err
		}

	}()

	return errors
}

func (cb *callback) BindChannel(ch <-chan []lua.LValue) {
	errCh := cb.BindChannelErrors(ch)
	go func() {
		for range errCh {
		}
	}()
}

// LGet returns a Callback for the function passed at parameter index `stack`
func LGet(stack int, L *lua.LState) Callback {
	val := L.CheckFunction(stack)
	l := loop.LGet(L)

	return New(val, l)
}

// LGetOpt works like LGet but returns nil if no function has been passed
func LGetOpt(stack int, L *lua.LState) Callback {
	val := L.OptFunction(stack, nil)
	if val != nil {
		l := loop.LGet(L)
		return New(val, l)
	}

	return nil
}

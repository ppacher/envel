package testing

import (
	"context"
	"testing"

	"github.com/ppacher/envel/pkg/loop"
	lua "github.com/yuin/gopher-lua"
)

type Loader func(*lua.LState)

func GetTestLoop(t *testing.T, Preloaders ...Loader) (loop.Loop, chan struct{}) {
	l, _ := loop.New(nil)
	l.Start(context.Background())

	// load the reader library
	l.ScheduleAndWait(func(L *lua.LState) {
		for _, l := range Preloaders {
			l(L)
		}
	})

	ch := make(chan struct{}, 1)

	l.ScheduleAndWait(func(L *lua.LState) {
		L.SetGlobal("error", L.NewFunction(func(L *lua.LState) int {
			t.Error(L.CheckString(1))
			return 0
		}))

		L.SetGlobal("done", L.NewFunction(func(L *lua.LState) int {
			ch <- struct{}{}
			return 0
		}))
	})

	return l, ch
}

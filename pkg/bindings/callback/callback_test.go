package callback

import (
	"context"
	"testing"

	"github.com/ppacher/envel/pkg/loop"
	lua "github.com/yuin/gopher-lua"
)

func Test_SimpleCallback(t *testing.T) {
	loop, _ := loop.New(nil)
	loop.Start(context.Background())

	var cb Callback

	loop.ScheduleAndWait(func(vm *lua.LState) {
		vm.DoString("function test(a1, a2) error(tostring(a1)..tostring(a2)) end")

		fn := vm.GetGlobal("test").(*lua.LFunction)

		cb = New(fn, loop)
	})

	err := <-cb.Do(lua.LString("hello"), lua.LString("world"))
	if err == nil {
		t.Errorf("Expected lua callback to return an error but got nil")
		t.FailNow()
	} else {
		apiErr, ok := err.(*lua.ApiError)
		if !ok {
			t.Errorf("Expected an API error but got %v", err)
		} else {
			if apiErr.Object.(lua.LString).String() != "<string>:1: helloworld" {
				t.Errorf("Expected error object to be LString('<string>:1: helloworld') but got %#v", apiErr.Object)
			}
		}
	}

	loop.Stop()
	loop.Wait()
}

func Test_BindChannel(t *testing.T) {
	loop, _ := loop.New(nil)
	loop.Start(context.Background())

	// prepare
	var cb Callback
	loop.ScheduleAndWait(func(state *lua.LState) {
		state.SetGlobal("i", lua.LNumber(0))
		state.DoString("function test(a) i = i + a end")
		fn := state.GetGlobal("test").(*lua.LFunction)

		cb = New(fn, loop)
	})

	ch := make(chan []lua.LValue)
	cb.BindChannel(ch)
	defer close(ch)

	ch <- []lua.LValue{lua.LNumber(1)}
	ch <- []lua.LValue{lua.LNumber(3)}
	ch <- []lua.LValue{lua.LNumber(-1)}

	loop.ScheduleAndWait(func(state *lua.LState) {
		i := state.GetGlobal("i").(lua.LNumber)

		if i != 3 {
			t.Errorf("expected the callback to run 3 times and produce 2 but it did: %v", i)
		}
	})

	loop.Stop()
	loop.Wait()
}

package loop

import (
	"context"
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func Test_LoopSchedule(t *testing.T) {
	loop, err := New(nil)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}

	loop.Start(context.Background())

	ch := make(chan bool)
	loop.Schedule(func(vm *lua.LState) {
		ch <- true
		if err := vm.DoString(`print("hello")`); err != nil {
			t.Errorf("Expected method to run successfully but got error: %s", err.Error())
		}
	})

	<-ch

	loop.Stop()
	loop.Wait()
}

func Test_LoopSequence(t *testing.T) {
	loop, err := New(nil)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	loop.Start(context.Background())

	i := 0
	for a := 0; a < 10; a++ {
		loop.Schedule(func(*lua.LState) {
			i++
		})
	}

	loop.Stop()
	loop.Wait()

	if i != 10 {
		t.Errorf("not all jobs executed. Expected i to be 10 but got %d", i)
	}
}

func Test_LoopAccess(t *testing.T) {
	loop, err := New(nil)
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	loop.Start(context.Background())

	loop.ScheduleAndWait(func(state *lua.LState) {
		l := state.NewUserData()
		l.Value = loop
		state.SetGlobal("__loop", l)
	})

	loop.ScheduleAndWait(func(state *lua.LState) {
		l := state.GetGlobal("__loop")

		ud, ok := l.(*lua.LUserData)
		if !ok {
			t.Errorf("expected user data but gut %#v", l)

		} else {
			_, ok = ud.Value.(Loop)
			if !ok {
				t.Errorf("expected Loop but got %#v", ud.Value)
			}
		}
	})
}

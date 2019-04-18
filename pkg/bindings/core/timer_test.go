package core

import (
	"testing"
	"time"

	lua "github.com/yuin/gopher-lua"
)

func Test_Timer(t *testing.T) {
	l, done := getLibTestLoop(t)

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		res = 0
		
		t = _G.__core.timer {
			timeout = 0.01,
			callback = function()
				res = res + 1
				if res >= 4 then done() end
			end
		}
		
		t:start()
		`)
		if err != nil {
			t.Error(err)
		}
	})

	go func() {
		// timeout after 3 seconds
		<-time.After(time.Second * 3)
		close(done)
	}()

	<-done

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		if res ~= 4 then	
			error("expected res to be 4 but got "..tostring(res))	
		end
		`)

		if err != nil {
			t.Error(err)
		}
	})

	l.Stop()
	l.Wait()
}

package core

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func Test_ExecCall(t *testing.T) {
	l, done := getLibTestLoop(t)

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		
		exec = _G.__core.exec
		
		local function outs(outline)
			if outline ~= "test\n" then
				--error("expected outline to be test but got "..outline)
			end
		end
		
		local function done_cb()
			done()
		end

		exec("echo test", true, outs, done_cb)
		`)

		if err != nil {
			t.Error(err)
			close(done)
		}
	})

	<-done

	l.Stop()
	l.Wait()
}

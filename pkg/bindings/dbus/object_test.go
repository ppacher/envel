package dbus

import (
	"testing"

	helper "github.com/ppacher/envel/pkg/bindings/testing"
	lua "github.com/yuin/gopher-lua"
)

func Test_ObjectCall(t *testing.T) {
	l, ch := helper.GetTestLoop(t, Preload)

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		o = require("envel.dbus").object.bus()
		o:call("org.freedesktop.DBus.ListNames", 0, function(res, err)
			done()	

			found = false
			for _, v in res() do
				if v == "org.freedesktop.DBus" then
					found = true
				end
			end
			
			if not found then
				error("Expected to find org.freedesktop.DBus in the name list")
			end
		end)
		
		`)
		if err != nil {
			t.Error(err)
			close(ch)
		}
	})

	<-ch

	l.Stop()
	l.Wait()
}

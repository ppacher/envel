package dbus

import (
	"testing"

	helper "github.com/ppacher/envel/pkg/bindings/testing"
	lua "github.com/yuin/gopher-lua"
)

func Test_Notify(t *testing.T) {
	l, ch := helper.GetTestLoop(t, Preload)

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		notify = require("envel.dbus").notify

		notify({
			title = "Foobar",
			text = "body"
		})
		
		done()
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

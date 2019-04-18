package mqtt

import (
	"testing"
	"time"

	helper "github.com/ppacher/envel/pkg/bindings/testing"
	lua "github.com/yuin/gopher-lua"
)

func Test_ObjectCall(t *testing.T) {
	l, ch := helper.GetTestLoop(t, Preload)

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		c = require("envel.mqtt")({
			broker = "tcp://localhost:1883",
			client_id="mqtt-test"
		})
		
		c:subscribe {
			topic = "test",
			callback = function(msg)
				done()
				c:close()
				if msg.body ~= "foobar" then
					error("Expected test but got "..msg.body)	
				end
			end
		}
		
		c:publish{
			topic = "test",
			payload = "foobar",
		}
		
		`)
		if err != nil {
			t.Error(err)
			close(ch)
		}
	})

	// make sure we timeout
	go func() {
		<-time.After(time.Second * 2)
		close(ch)
	}()

	<-ch

	l.Stop()
	l.Wait()
}

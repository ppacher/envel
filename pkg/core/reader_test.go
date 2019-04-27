package core

import (
	"testing"

	lua "github.com/yuin/gopher-lua"
)

func Test_ReaderString(t *testing.T) {
	l, _ := getLibTestLoop(t)

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		reader = _G.__core.reader

		r = reader.from_string("foobar")
		res = r:read()

		if res ~= "foobar" then
			error("expected foobar but got "..res)
		end
		`)

		if err != nil {
			t.Error(err)
		}
	})

	l.Stop()
	l.Wait()
}

func Test_ReaderLine(t *testing.T) {
	l, _ := getLibTestLoop(t)

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		reader = _G.__core.reader
		r = reader.from_string("line1\nline2\nline3")
		
		d = r:read("*line")
		if d ~= "line1" then
			error("Expected line1 but got '"..d.."'")
		end
		
		d = r:read("*l")
		if d ~= "line2" then
			error("Expected line2 but got '"..d.."'")
		end
		
		d = r:read()
		if d ~= "line3" then
			error("Expected line3 but got '"..d.."'")
		end
		
		`)
		if err != nil {
			t.Error(err)
		}
	})

	l.Stop()
	l.Wait()
}

func Test_ReaderBytes(t *testing.T) {
	l, _ := getLibTestLoop(t)

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		reader = _G.__core.reader
		r = reader.from_string("1234567890")
		
		d = r:read(5)
		if d ~= "12345" then
			error("Expected 12345 but got '"..d.."'")
		end
		
		d = r:read(10)
		if d ~= "67890" then
			error("Expected 67890 but got '"..d.."'")
		end
		
		d = r:read(10)
		if d ~= nil then
			error("Expected nil due to EOF but got "..tostring(d))
		end
		
		`)
		if err != nil {
			t.Error(err)
		}
	})

	l.Stop()
	l.Wait()
}

func Test_ReaderAll(t *testing.T) {
	l, _ := getLibTestLoop(t)

	l.ScheduleAndWait(func(L *lua.LState) {
		err := L.DoString(`
		reader = _G.__core.reader
		r = reader.from_string("1234567890")
		
		d = r:read("*a")
		if d ~= "1234567890" then
			error("Expected 1234567890 but got '"..d.."'")
		end
		
		d = r:read("*a")
		if d ~= nil then
			error("Expected nil due to EOF but got "..#d)
		end

		r = reader.from_string("1234567890")
		
		d = r:read("*all")
		if d ~= "1234567890" then
			error("Expected 1234567890 but got '"..d.."'")
		end
		
		d = r:read("*all")
		if d ~= nil then
			error("Expected nil due to EOF but got "..#d)
		end
		
		`)
		if err != nil {
			t.Error(err)
		}
	})

	l.Stop()
	l.Wait()
}

func Test_ReaderLineCallback(t *testing.T) {
	l, done := getLibTestLoop(t)

	l.ScheduleAndWait(func(L *lua.LState) {
		L.SetGlobal("res", lua.LString(""))
		err := L.DoString(`
		reader = _G.__core.reader

		r = reader.from_string("hello\nworld")
		
		r:with_line_callback(function(line) 
			if line == nil then
				-- EOF	
				done()
			end

			-- only get the first line
			if res == "" then res = line end
		end)
		`)

		if err != nil {
			t.Error(err)
		}
	})

	<-done

	l.ScheduleAndWait(func(L *lua.LState) {
		res := L.GetGlobal("res").(lua.LString)
		if res.String() != "hello" {
			t.Errorf("expect hello but got %#v", res)
		}
	})

	l.Stop()
	l.Wait()
}

package dbus

import (
	"github.com/godbus/dbus"
	luar "github.com/layeh/gopher-luar"
	"github.com/ppacher/envel/pkg/callback"
	"github.com/ppacher/envel/pkg/loop"
	lua "github.com/yuin/gopher-lua"
)

const objectTypeName = "dbusObject"

// AddObject adds a DBUS object to lua
func AddObject(L *lua.LState, t *lua.LTable) {
	typeMt := L.NewTypeMetatable(objectTypeName)

	indexTable := L.NewTable()
	L.SetField(typeMt, "__index", L.SetFuncs(indexTable, dbusObjectTypeAPI))

	indexTable.RawSetString("call", luar.New(L, objectCall))

	object := L.NewTable()
	L.SetFuncs(object, map[string]lua.LGFunction{
		"bus": busObject,
	})

	L.SetMetatable(object, L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"__call": objectNew,
	}))

	t.RawSetString("__dbus_object_mt", typeMt)
	L.SetField(t, "object", object)
}

var dbusObjectTypeAPI = map[string]lua.LGFunction{}

type Object struct {
	dbus.BusObject
	L *lua.LState
}

func busObject(L *lua.LState) int {
	conn, err := GetConnection(L)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	obj := conn.BusObject()

	ud := L.NewUserData()
	ud.Value = &Object{
		BusObject: obj,
		L:         L,
	}

	L.SetMetatable(ud, L.GetTypeMetatable(objectTypeName))

	L.Push(ud)
	return 1
}

func objectNew(L *lua.LState) int {
	conn, err := GetConnection(L)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	dest := L.CheckString(2)
	path := L.CheckString(3)

	obj := conn.Object(dest, dbus.ObjectPath(path))

	ud := L.NewUserData()
	ud.Value = &Object{
		BusObject: obj,
		L:         L,
	}

	L.SetMetatable(ud, L.GetTypeMetatable(objectTypeName))

	L.Push(ud)

	return 1
}

func objectCall(ud *lua.LUserData, method string, flags byte, fn *lua.LFunction, args ...interface{}) {
	obj, ok := ud.Value.(*Object)
	if !ok {
		panic("Expected an dbusObject")
	}

	call := obj.Go(method, dbus.Flags(flags), nil, args...)

	l := loop.LGet(obj.L)
	cb := callback.New(fn, l)

	go func() {
		<-call.Done
		var x interface{}
		err := call.Store(&x)

		cb.From(func(L *lua.LState) []lua.LValue {
			val := luar.New(L, x)
			args := []lua.LValue{
				val,
			}

			if err != nil {
				args = append(args, lua.LString(err.Error()))
			}

			return args
		})
	}()
}

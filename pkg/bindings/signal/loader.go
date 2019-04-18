package signal

import (
	lua "github.com/yuin/gopher-lua"
)

// OpenSignalWithName opens the signal library and also
// adds a package path `name` so it can be require()d
func OpenSignalWithName(name string, L *lua.LState) {
	L.Push(L.NewFunction(Loader))
	L.Call(0, 0)

	L.PreloadModule(name, func(L *lua.LState) int {
		mod := L.GetGlobal("__signal")
		L.Push(mod)
		return 1
	})
}

// OpenSignal opens the signal library
func OpenSignal(L *lua.LState) {
	L.Push(L.NewFunction(Loader))
	L.Call(0, 0)
}

// Loader loads the actual mqtt package
func Loader(L *lua.LState) int {
	t := L.RegisterModule("__signal", map[string]lua.LGFunction{}).(*lua.LTable)

	createSignalTypeMetatable(L, t)

	L.SetMetatable(t, L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"__call": newSignal,
	}))

	L.Push(t)
	return 1
}

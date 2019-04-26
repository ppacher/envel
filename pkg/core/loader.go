package core

import lua "github.com/yuin/gopher-lua"

func OpenCore(L *lua.LState) {
	L.Push(L.NewFunction(Loader))
	L.Call(0, 0)
}

func OpenCoreWithName(name string, L *lua.LState) {
	L.Push(L.NewFunction(Loader))
	L.Call(0, 0)

	L.PreloadModule(name, func(L *lua.LState) int {
		mod := L.GetGlobal("__core")
		L.Push(mod)
		return 1
	})
}

func Loader(L *lua.LState) int {
	mod := L.RegisterModule("__core", map[string]lua.LGFunction{}).(*lua.LTable)

	AddReader(L, mod)
	AddExec(L, mod)
	AddTimer(L, mod)

	L.Push(mod)

	return 1
}

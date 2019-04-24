package mqtt

import (
	lua "github.com/yuin/gopher-lua"
)

// Preload reloads the envel.mqtt package
func Preload(L *lua.LState) {
	L.PreloadModule("envel.bindings.mqtt", Loader)
}

func PreloadWithName(name string, L *lua.LState) {
	L.PreloadModule(name, Loader)
}

// Loader loads the actual mqtt package
func Loader(L *lua.LState) int {
	t := L.NewTable()

	createMQTTTypeTable(L, t)

	L.SetMetatable(t, L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"__call": newMQTT,
	}))

	L.Push(t)
	return 1
}

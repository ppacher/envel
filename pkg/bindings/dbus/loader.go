package dbus

import (
	lua "github.com/yuin/gopher-lua"
)

// Preload adds plugin to the given Lua state's package.preload table. After it
// has been preloaded, it can be loaded using require:
//
//  local plugin = require("plugin")
func Preload(L *lua.LState) {
	L.PreloadModule("envel.dbus", Loader)
}

func PreloadWithName(name string, L *lua.LState) {
	L.PreloadModule(name, Loader)
}

// Loader is the module loader function.
func Loader(L *lua.LState) int {
	t := L.NewTable()

	AddNotify(L, t)
	AddObject(L, t)

	L.Push(t)
	return 1
}

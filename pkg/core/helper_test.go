package core

import (
	"testing"

	helper "github.com/ppacher/envel/pkg/testing"
	"github.com/ppacher/envel/pkg/loop"
	lua "github.com/yuin/gopher-lua"
)

func getLibTestLoop(t *testing.T) (loop.Loop, chan struct{}) {
	return helper.GetTestLoop(t, func(L *lua.LState) {
		L.Push(L.NewFunction(OpenCore))
		L.Call(0, 0)

		L.PreloadModule("envel.core", func(L *lua.LState) int {
			mod := L.GetGlobal("__core")
			L.Push(mod)
			return 1
		})
	})
}

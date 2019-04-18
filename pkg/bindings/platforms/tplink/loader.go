package tplink

import (
	"github.com/ppacher/envel/pkg/bindings/platforms/tplink/api"
	"github.com/ppacher/envel/pkg/bindings/platforms/tplink/hs1xx"
	lua "github.com/yuin/gopher-lua"
)

func PreloadPlatform(L *lua.LState) {
	api.Preload(L)
	hs1xx.Preload(L)
}

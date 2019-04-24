package api

import (
	"context"

	"github.com/ppacher/envel/pkg/bindings/callback"
	tpshp "github.com/ppacher/tplink-smart-home-protocol"
	lua "github.com/yuin/gopher-lua"
)

const tplinkAPITypeName = "tplink_api"

func Preload(L *lua.LState) {
	L.PreloadModule("envel.bindings.platform.tplink.api", Loader)
}

func Loader(L *lua.LState) int {
	tbl := L.NewTable()

	typeMt := L.NewTypeMetatable(tplinkAPITypeName)
	L.SetField(typeMt, "__index", L.SetFuncs(L.NewTable(), tplinkAPITypeAPI))

	L.SetField(tbl, "__api_mt", typeMt)
	L.SetMetatable(tbl, L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"__call": newAPIClient,
	}))

	L.Push(tbl)
	return 1
}

var tplinkAPITypeAPI = map[string]lua.LGFunction{
	"send": tpshpSend,
}

func newAPIClient(L *lua.LState) int {
	ip := L.CheckString(2)

	cli := tpshp.New(ip)

	ud := L.NewUserData()
	ud.Value = cli
	L.SetMetatable(ud, L.GetTypeMetatable(tplinkAPITypeName))

	L.Push(ud)
	return 1
}

func checkAPIClient(L *lua.LState, arg int) tpshp.Client {
	ud := L.CheckUserData(arg)
	if v, ok := ud.Value.(tpshp.Client); ok {
		return v
	}

	L.ArgError(arg, "expected a tplink smart home client")
	return nil
}

func tpshpSend(L *lua.LState) int {
	cli := checkAPIClient(L, 1)
	message := L.CheckString(2)
	cb := callback.LGet(3, L)

	go func() {
		response, err := cli.Send(context.Background(), []byte(message))
		if err != nil {
			<-cb.Do(lua.LNil, lua.LString(err.Error()))
		} else {
			<-cb.Do(lua.LString(response))
		}
	}()

	return 0
}

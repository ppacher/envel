package hs1xx

import (
	"context"
	"io"
	"log"
	"time"

	tpsmartapi "github.com/ppacher/tplink-hs1xx"

	"github.com/ppacher/envel/pkg/bindings/callback"
	"github.com/ppacher/envel/pkg/bindings/core"
	"github.com/ppacher/envel/pkg/loop"

	"github.com/ppacher/envel/pkg/bindings/signal"
	hs1xx "github.com/ppacher/tplink-hs1xx/plug"
	lua "github.com/yuin/gopher-lua"
)

const hs1xxxTypeName = "tplink_hs1xxx"

func Preload(L *lua.LState) {
	L.PreloadModule("envel.bindings.platform.tplink.hs1xx", Loader)
}

func PreloadWithName(name string, L *lua.LState) {
	L.PreloadModule(name, Loader)
}

func Loader(L *lua.LState) int {
	tbl := L.NewTable()

	mt := L.NewTypeMetatable(hs1xxxTypeName)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), hs1xxTypeAPI))

	L.SetField(tbl, "__hs1xx_mt", mt)
	L.SetMetatable(tbl, L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"__call": newHs1xx,
	}))

	L.Push(tbl)
	return 1
}

var hs1xxTypeAPI = map[string]lua.LGFunction{
	"turn_on":        hs1xxTurnOn,
	"turn_off":       hs1xxTurnOff,
	"sysinfo":        hs1xxSysinfo,
	"watch_relay":    hs1xxWatchRelay,
	"realtime":       hs1xxGetRealtime,
	"watch_realtime": hs1xxWatchRealtime,
}

type HS1xx struct {
	hs1xx.HS1xx
	currentRelayState bool
}

func newHs1xx(L *lua.LState) int {
	ip := L.CheckString(2)

	hs := &HS1xx{
		HS1xx: hs1xx.New(ip),
	}

	ud := L.NewUserData()
	ud.Value = hs
	L.SetMetatable(ud, L.GetTypeMetatable(hs1xxxTypeName))

	signal.Extend(L, ud)

	L.Push(ud)

	return 1
}

func checkHS1xx(L *lua.LState, arg int) *HS1xx {
	ud := L.CheckUserData(arg)
	if hs, ok := ud.Value.(*HS1xx); ok {
		return hs
	}

	L.ArgError(arg, "expected a "+hs1xxxTypeName)

	return nil
}

func hs1xxTurnOn(L *lua.LState) int {
	hs := checkHS1xx(L, 1)

	res := <-hs.TurnOn(context.Background())
	if res.Err() != nil {
		L.Push(lua.LString(res.Err().Error()))
		return 1
	}

	sig, err := signal.GetSignal(L, L.Get(1))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}
	hs.currentRelayState = true

	sig.Emit("state::on")
	sig.Emit("state")

	return 0
}

func hs1xxTurnOff(L *lua.LState) int {
	hs := checkHS1xx(L, 1)

	res := <-hs.TurnOff(context.Background())
	if res.Err() != nil {
		L.Push(lua.LString(res.Err().Error()))
		return 1
	}

	sig, err := signal.GetSignal(L, L.Get(1))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	hs.currentRelayState = false

	sig.Emit("state::off")
	sig.Emit("state")

	return 0
}

func hs1xxSysinfo(L *lua.LState) int {
	hs := checkHS1xx(L, 1)
	sysinfo := <-hs.SysInfo(context.Background())
	if sysinfo.Err() != nil {
		L.RaiseError(sysinfo.Err().Error())
		return 0
	}

	result := L.NewTable()

	result.RawSetString("active_mode", lua.LString(sysinfo.ActiveMode))
	result.RawSetString("alias", lua.LString(sysinfo.Alias))
	result.RawSetString("device_id", lua.LString(sysinfo.DeviceID))
	result.RawSetString("device_name", lua.LString(sysinfo.DeviceName))
	result.RawSetString("feature", lua.LString(sysinfo.Feature))
	result.RawSetString("fw_id", lua.LString(sysinfo.FwID))
	result.RawSetString("hw_ver", lua.LString(sysinfo.HWVer))
	result.RawSetString("hw_id", lua.LString(sysinfo.HwID))
	result.RawSetString("icon_hash", lua.LString(sysinfo.IconHash))
	result.RawSetString("latitude", lua.LNumber(sysinfo.Latitude))
	result.RawSetString("led_off", lua.LNumber(sysinfo.LedOff))
	result.RawSetString("longitude", lua.LNumber(sysinfo.Longitude))
	result.RawSetString("mac", lua.LString(sysinfo.MAC))
	result.RawSetString("model", lua.LString(sysinfo.Model))
	result.RawSetString("oem_id", lua.LString(sysinfo.OEMID))
	result.RawSetString("on_time", lua.LNumber(sysinfo.OnTime))
	result.RawSetString("relay_state", lua.LBool(sysinfo.RelayState))
	result.RawSetString("rssi", lua.LNumber(sysinfo.Rssi))
	result.RawSetString("sw_ver", lua.LString(sysinfo.SWVer))
	result.RawSetString("type", lua.LString(sysinfo.Type))
	result.RawSetString("updating", lua.LNumber(sysinfo.Updating))

	L.Push(result)

	return 1
}

func realtimeToTable(L *lua.LState, realtime *tpsmartapi.RealtimeInfo) *lua.LTable {
	result := L.NewTable()
	result.RawSetString("voltage", lua.LNumber(realtime.Voltage()))
	result.RawSetString("current", lua.LNumber(realtime.Current()))
	result.RawSetString("power", lua.LNumber(realtime.Power()))
	result.RawSetString("total", lua.LNumber(realtime.Total()))

	return result
}

func hs1xxGetRealtime(L *lua.LState) int {
	hs := checkHS1xx(L, 1)
	cb := callback.LGet(2, L)

	go func() {
		realtime := <-hs.EMeter().GetRealtime(context.Background())
		<-cb.From(func(L *lua.LState) []lua.LValue {
			if realtime.Err() != nil {
				return []lua.LValue{
					lua.LNil,
					lua.LString(realtime.Err().Error()),
				}
			}

			return []lua.LValue{realtimeToTable(L, realtime)}
		})
	}()

	return 0
}

func hs1xxWatchRelay(L *lua.LState) int {
	hs := checkHS1xx(L, 1)
	timeout := L.CheckNumber(2)
	lo := loop.LGet(L)

	sig, err := signal.GetSignal(L, L.Get(1))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	duration := time.Duration(float64(timeout) * float64(time.Second))

	var timer *core.Timer
	var timerUserData *lua.LUserData

	fn := L.NewFunction(func(L *lua.LState) int {
		timer.Stop()

		go func() {
			defer timer.Start()

			sysinfo := <-hs.SysInfo(context.Background())
			if sysinfo.Err() != nil {
				if sysinfo.Err() == io.EOF {
					// It seems like hs1xx plugs will start failing requests if the happen to often
					// if we receive an EOF for a call, give the plug some time to recover (skip two intervals)
					log.Printf("failed to poll hs1xx, try to increase the polling interval: %s\n", sysinfo.Err().Error())

					// sleeping here works because the timer has been stopped and will
					// be restarted once this function returns
					<-time.After(duration * 2)
				} else {
					log.Printf("failed to poll hs1xx: %s\n", sysinfo.Err().Error())
				}
				return
			}
			newState := bool(sysinfo.RelayState)

			if hs.currentRelayState != newState {
				hs.currentRelayState = newState

				sig.Emit("state", lua.LBool(newState))
				if newState {
					sig.Emit("state::on", lua.LBool(newState))
				} else {
					sig.Emit("state::off", lua.LBool(newState))
				}
			}
		}()

		return 0
	})

	nativeCb := callback.New(fn, lo)

	timerUserData, timer = core.NewTimer(L, core.TimerOptions{
		Autostart: true,
		Timeout:   duration,
		Callback:  nativeCb,
	})

	if err := timer.Init(L); err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	L.Push(timerUserData)
	return 1
}

func hs1xxWatchRealtime(L *lua.LState) int {
	hs := checkHS1xx(L, 1)
	timeout := L.CheckNumber(2)
	lo := loop.LGet(L)

	sig, err := signal.GetSignal(L, L.Get(1))
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	duration := time.Duration(float64(timeout) * float64(time.Second))

	var timer *core.Timer
	var timerUserData *lua.LUserData

	fn := L.NewFunction(func(L *lua.LState) int {
		timer.Stop()

		go func() {
			defer timer.Start()
			realtime := <-hs.EMeter().GetRealtime(context.Background())

			if realtime.Err() != nil {
				log.Printf("failed to poll realtime information: %s\n", realtime.Err().Error())
				return
			}

			sig.EmitFrom("realtime", func(L *lua.LState) []lua.LValue {
				return []lua.LValue{
					realtimeToTable(L, realtime),
				}
			})
		}()

		return 0
	})

	nativeCb := callback.New(fn, lo)

	timerUserData, timer = core.NewTimer(L, core.TimerOptions{
		Autostart: true,
		Timeout:   duration,
		Callback:  nativeCb,
	})

	if err := timer.Init(L); err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	L.Push(timerUserData)
	return 1
}

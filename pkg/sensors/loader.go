package sensors

import lua "github.com/yuin/gopher-lua"

const registryTypeName = "registry"

// OpenSensorRegistry openes the sensor registry for the given lua Vm
func OpenSensorRegistry(L *lua.LState, registry Registry) {
	L.Push(L.NewFunction(func(L *lua.LState) int {
		mod := L.RegisterModule("__sensors", map[string]lua.LGFunction{}).(*lua.LTable)

		mt := L.NewTypeMetatable(registryTypeName)
		L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
			"register_sensor": luaRegisterSensor,
		}))

		ud := L.NewUserData()
		ud.Value = registry
		L.SetMetatable(ud, mt)

		L.SetField(mod, "registry", ud)

		return 1
	}))

	L.Call(0, 0)
}

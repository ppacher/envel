package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	lua "github.com/yuin/gopher-lua"
)

// promCounterTypeName is the name for prometeus counter types
const promCounterTypeName = "prometheus_counter"

// promCounterTypeAPI holds the API definition of lua objects of
// type prometheus_counter
var promCounterTypeAPI = map[string]lua.LGFunction{
	"inc": counterInc,
	"add": counterAdd,
}

// getCounter creates methods to create a new prometheus counter
func getCounter(L *lua.LState) lua.LValue {
	typeTable := L.NewTypeMetatable(promCounterTypeName)

	L.SetField(typeTable, "__index", L.SetFuncs(L.NewTable(), promCounterTypeAPI))

	return L.NewFunction(createNewCounter)
}

// createNewCounter creates a new prometheus counter object
func createNewCounter(L *lua.LState) int {
	options := L.CheckTable(1)
	opts := prometheus.Opts{}
	getCommonMetricOpts(L, options, &opts)

	counter := prometheus.NewCounter(prometheus.CounterOpts(opts))

	prometheus.MustRegister(counter)

	// prepare the actual user-data for the Lua VM
	ud := L.NewUserData()
	ud.Value = counter
	L.SetMetatable(ud, L.GetTypeMetatable(promCounterTypeName))
	L.Push(ud)

	return 1
}

func checkCounter(L *lua.LState, arg int) prometheus.Counter {
	ud := L.CheckUserData(arg)
	if c, ok := ud.Value.(prometheus.Counter); ok {
		return c
	}

	L.ArgError(arg, "Expected a "+promCounterTypeName)
	return nil
}

func counterInc(L *lua.LState) int {
	c := checkCounter(L, 1)
	c.Inc()
	return 0
}

func counterAdd(L *lua.LState) int {
	c := checkCounter(L, 1)
	n := L.CheckNumber(2)

	c.Add(float64(n))
	return 0
}

package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	lua "github.com/yuin/gopher-lua"
)

// promGaugeName is the name for prometeus gauge types
const promGaugeName = "prometheus_gauge"

// promGaugeAPI holds the API definition of lua objects of
// type prometheus_counter
var promGaugeAPI = map[string]lua.LGFunction{
	"inc": gaugeInc,
	"dec": gaugeDec,
	"set": gaugeSet,
	"add": gaugeAdd,
}

// getGauge creates methods to create a new prometheus gauge
func getGauge(L *lua.LState) lua.LValue {
	typeTable := L.NewTypeMetatable(promGaugeName)

	L.SetField(typeTable, "__index", L.SetFuncs(L.NewTable(), promGaugeAPI))

	return L.NewFunction(createNewGauge)
}

// createNewGauge creates a new prometheus counter object
func createNewGauge(L *lua.LState) int {
	options := L.CheckTable(1)
	opts := prometheus.Opts{}

	getCommonMetricOpts(L, options, &opts)

	gauge := prometheus.NewGauge(prometheus.GaugeOpts(opts))

	prometheus.MustRegister(gauge)

	// prepare the actual user-data for the Lua VM
	ud := L.NewUserData()
	ud.Value = gauge
	L.SetMetatable(ud, L.GetTypeMetatable(promGaugeName))
	L.Push(ud)

	return 1
}

func checkGauge(L *lua.LState, arg int) prometheus.Gauge {
	ud := L.CheckUserData(arg)
	if c, ok := ud.Value.(prometheus.Gauge); ok {
		return c
	}

	L.ArgError(arg, "Expected a "+promGaugeName)
	return nil
}

func gaugeAdd(L *lua.LState) int {
	g := checkGauge(L, 1)
	n := L.CheckNumber(2)

	g.Add(float64(n))

	return 0
}

func gaugeInc(L *lua.LState) int {
	g := checkGauge(L, 1)
	g.Inc()

	return 0
}

func gaugeDec(L *lua.LState) int {
	g := checkGauge(L, 1)
	g.Dec()

	return 0
}

func gaugeSet(L *lua.LState) int {
	g := checkGauge(L, 1)
	n := L.CheckNumber(2)

	g.Set(float64(n))

	return 0
}

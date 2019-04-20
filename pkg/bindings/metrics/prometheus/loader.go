package prometheus

import (
	"log"
	"net/http"
	"sync"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	lua "github.com/yuin/gopher-lua"
)

// Preload preloads the event.metrics.prometheus library
func Preload(L *lua.LState) {
	L.PreloadModule("envel.metrics.prometheus", Loader)
}

var metricTypeMap = map[string]lua.LGFunction{
	"counter": createNewCounter,
	"gauge":   createNewGauge,
}

// Loader implements a lua.LGFunction and is used
// to preload the envel.metrics.prometheus library
func Loader(L *lua.LState) int {
	// make sure we serve any metrics via HTTP
	// this will do nothing if we are already serving them
	serveMetrics()

	// t is the actual table that we expose as "event.metrics.prometheus"
	t := L.NewTable()

	L.SetMetatable(t, L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"__call": createMetric,
	}))

	t.RawSetString("counter", getCounter(L))
	t.RawSetString("gauge", getGauge(L))

	L.Push(t)
	return 1
}

func createMetric(L *lua.LState) int {
	options := L.CheckTable(2)
	metricTypeValue := options.RawGetString("type")

	metricType, ok := metricTypeValue.(lua.LString)
	if !ok {
		L.ArgError(2, "metric type must be set to one of 'gauge', 'counter'")
		return 0
	}

	handler := metricTypeMap[string(metricType)]
	if handler == nil {
		L.ArgError(2, "metric type must be set to one of 'gauge', 'counter'")
		return 0
	}

	L.Push(L.NewFunction(handler))
	L.Push(options)
	L.Call(1, 1)
	result := L.Get(3)

	L.Push(result)
	return 1
}

var serve sync.Once

func serveMetrics() {
	serve.Do(func() {
		// TODO(ppacher): move this to main
		http.Handle("/metrics", promhttp.Handler())
		go func() {
			log.Fatal(http.ListenAndServe(":9091", nil))
		}()
	})
}

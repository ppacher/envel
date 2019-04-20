package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
	lua "github.com/yuin/gopher-lua"
)

func getCommonMetricOpts(L *lua.LState, options *lua.LTable, opts *prometheus.Opts) {
	if options != nil {
		if h, ok := L.GetField(options, "name").(lua.LString); ok {
			opts.Name = string(h)
		}

		if h, ok := L.GetField(options, "help").(lua.LString); ok {
			opts.Help = string(h)
		}

		if ns, ok := L.GetField(options, "namespace").(lua.LString); ok {
			opts.Namespace = string(ns)
		}

		if sub, ok := L.GetField(options, "subsystem").(lua.LString); ok {
			opts.Subsystem = string(sub)
		}
	}

	if opts.Name == "" {
		L.RaiseError("Metric name must be set")
	}
}

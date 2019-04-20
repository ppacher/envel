package main

import (
	"context"
	"fmt"

	"github.com/ppacher/envel/pkg/sensors"

	"github.com/alecthomas/kingpin"
	"github.com/ppacher/envel/pkg/bindings/core"
	"github.com/ppacher/envel/pkg/bindings/dbus"
	"github.com/ppacher/envel/pkg/bindings/http"
	"github.com/ppacher/envel/pkg/bindings/metrics/prometheus"
	"github.com/ppacher/envel/pkg/bindings/mqtt"
	"github.com/ppacher/envel/pkg/bindings/platforms/tplink"
	"github.com/ppacher/envel/pkg/bindings/signal"
	"github.com/ppacher/envel/pkg/loop"
	"github.com/sirupsen/logrus"
	json "layeh.com/gopher-json"

	lua "github.com/yuin/gopher-lua"
)

var loadPaths = kingpin.Flag("lua-path", "Lua include paths").Short('p').Strings()
var filePath = kingpin.Arg("file", "Path to the file to execute").String()

func main() {
	kingpin.Parse()

	registry := sensors.NewRegistry()

	l, err := loop.New(&loop.Options{
		InitVM: func(L *lua.LState) error {
			core.OpenCore(L)
			signal.OpenSignal(L)
			json.Preload(L)
			dbus.Preload(L)
			mqtt.Preload(L)
			http.Preload(L)
			tplink.PreloadPlatform(L)
			prometheus.Preload(L)

			sensors.OpenSensorRegistry(L, registry)

			// TODO(ppacher): this is ugly ...
			for _, p := range *loadPaths {
				L.DoString(fmt.Sprintf(`
				package.path = "%s" .. [[/?.lua;]] .. package.path	
				`, p))
			}

			err := L.DoFile(*filePath)

			return err
		},
	})
	if err != nil {
		logrus.Fatal(err)
	}

	l.Start(context.Background())

	l.Wait()
}

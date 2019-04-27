package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ppacher/envel/pkg/plugin"

	"github.com/alecthomas/kingpin"

	// built-in bindings
	_ "github.com/ppacher/envel/pkg/bindings/dbus"
	_ "github.com/ppacher/envel/pkg/bindings/metrics/prometheus"
	_ "github.com/ppacher/envel/pkg/bindings/mqtt"
	_ "github.com/ppacher/envel/pkg/bindings/platforms/tplink"

	// default lua bindings
	"github.com/ppacher/envel/pkg/core"
	"github.com/ppacher/envel/pkg/http"
	"github.com/ppacher/envel/pkg/loop"
	signalBinding "github.com/ppacher/envel/pkg/signal"

	"github.com/sirupsen/logrus"
	json "layeh.com/gopher-json"

	lua "github.com/yuin/gopher-lua"
)

var loadPaths = kingpin.Flag("lua-path", "Lua include paths").Short('p').Strings()
var filePath = kingpin.Arg("file", "Path to the file to execute").String()
var pluginPaths = kingpin.Flag("plugins", "Path to a plugin file or directory to load on startup").Short('P').Strings()

func main() {
	kingpin.Parse()

	// TODO(ppacher) move to command line flag
	logrus.SetLevel(logrus.DebugLevel)

	var p = append([]plugin.Instance(nil), plugin.Builtin()...)
	countBuiltin := len(p)

	for _, path := range *pluginPaths {
		plugins, err := plugin.LoadDirectory(path)
		if err != nil {
			logrus.Fatalf("failed to load plugins from %s: %s", path, err.Error())
			return
		}

		p = append(p, plugins...)
	}

	logrus.Debugf("built-in plugins: %d", countBuiltin)
	logrus.Debugf("found %d plugins in %d paths", len(p)-countBuiltin, len(*pluginPaths))

	for _, plugin := range p {
		if err := plugin.Init(); err != nil {
			logrus.Fatalf("failed to initialize plugin %s: %s", plugin.Name(), err.Error())
		}

		logrus.Debugf("plugin: %s initialized", plugin.Name())
	}

	l, err := loop.New(&loop.Options{
		InitVM: func(L *lua.LState) error {
			core.OpenCore(L)
			signalBinding.OpenSignal(L)
			json.Preload(L)
			http.Preload(L)

			// preload all plugins
			for _, plugin := range p {
				for _, b := range plugin.Bindings() {
					b.Preload(L)
					logrus.Debugf("plugin: %s preloaded into LState", plugin.Name())
				}
			}

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

	exitSig := make(chan os.Signal, 1)
	signal.Notify(exitSig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	<-exitSig

	logrus.Info("shutting down")

	l.Stop()
	l.Wait()

	logrus.Info("shutdown completed")
}

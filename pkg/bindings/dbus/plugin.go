//go:generate go run ../../../hacks/build-plugin.go -o ../../../plugins/ github.com/ppacher/envel/pkg/bindings/dbus

package dbus

import (
	"github.com/ppacher/envel/pkg/plugin"
	lua "github.com/yuin/gopher-lua"
)

// Binding implements plugin.Binding
type Binding struct{}

// Preload preloads the dbus module
func (Binding) Preload(L *lua.LState) error {
	Preload(L)
	return nil
}

var Plugin = plugin.New(
	plugin.WithBinding(Binding{}),
)

func init() {
	plugin.Register("dbus", Plugin)
}

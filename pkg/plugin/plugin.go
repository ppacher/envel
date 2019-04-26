package plugin

import lua "github.com/yuin/gopher-lua"

// SymbolName defines the name of the symbol that should be exposed by all plugins
// and should implement the Plugin interface
const SymbolName = "Plugin"

// Binding describes a plugin type that provides additional native
// bindings for envel
type Binding interface {
	// Preload is called when the binding should preload itself into the provided
	// lua state. Note that this function may be called mutliple times with different
	// lua states. The plugin must NOT share objects between different states
	Preload(*lua.LState) error
}

// Plugin describes a plugin for envel
type Plugin interface {
	// Init is called to initialize the plugin. It is called exactly once when
	// the plugin is loaded
	Init() error

	// Bindings should return a list of native lua bindings or nil if none are supported
	// by the plugin
	Bindings() []Binding
}

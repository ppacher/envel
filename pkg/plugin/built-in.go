package plugin

var builtinPlugins []Instance

// Register registeres a new built-in plugin
func Register(name string, p Plugin) {
	builtinPlugins = append(builtinPlugins, &instance{
		Plugin: p,
		name:   "[builtin: " + name + "]",
		path:   "<built-in>",
	})
}

// Builtin returns a list of built-in plugins
func Builtin() []Instance {
	return builtinPlugins
}

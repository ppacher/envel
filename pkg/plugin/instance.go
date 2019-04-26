package plugin

// Instance represents a loaded plugin
type Instance interface {
	Plugin

	// Name returns the filename of the plugin
	Name() string

	// Path returns the path to the plugin file
	Path() string
}

type instance struct {
	Plugin

	name string
	path string
}

func (p *instance) Name() string {
	return p.name
}

func (p *instance) Path() string {
	return p.path
}

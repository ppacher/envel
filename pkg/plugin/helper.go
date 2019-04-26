package plugin

// Option defines the type of function that serves as a plugin option
type Option func(p *pluginImpl)

// WithInit configures the init function to call when the plugin is loaded
func WithInit(init func() error) Option {
	return func(p *pluginImpl) {
		p.initFunction = init
	}
}

// WithBindings adds one or more bindings to the plugin
func WithBindings(b ...Binding) Option {
	return func(p *pluginImpl) {
		p.bindings = append(p.bindings, b...)
	}
}

// WithBinding adds a new binding the the plugin
// This is a singular alias for WithBindings
func WithBinding(b Binding) Option {
	return WithBindings(b)
}

// New creates a new plugin
func New(opts ...Option) Plugin {
	p := &pluginImpl{}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

type pluginImpl struct {
	initFunction func() error
	bindings     []Binding
}

// Bindings implements plugin.Bindings
func (p *pluginImpl) Bindings() []Binding {
	return p.bindings
}

// Init implements Plugin.Init
func (p *pluginImpl) Init() error {
	if p.initFunction != nil {
		return p.initFunction()
	}

	return nil
}

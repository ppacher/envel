package sensors

import (
	"sync"
)

// Registry stores sensor values
type Registry interface {
	// Add adds a new sensor
	Add(Sensor) error

	// List returns a list of available sensors
	List() []Sensor

	// Get searches for a sensor by name
	Get(string) Sensor

	// FindByLabel returns all sensors that have the given
	// label
	FindByLabel(string, string) []Sensor
}

// NewRegistry returns a new sensor registry
func NewRegistry() Registry {
	return nil
}

type registry struct {
	lock   sync.RWMutex
	sensor map[string]*Sensor
}

// Add adds a new sensor to the registry
func (reg *registry) Add(s Sensor) error {
	return nil
}

// List returns all sensors registered in the registry
func (reg *registry) List() []Sensor {
	return nil
}

// Get returns the sensor with the given name or nil
func (reg *registry) Get(name string) Sensor {
	return nil
}

// FindByLabel returns all sensor that have the given label key and value
func (reg *registry) FindByLabel(name, value string) []Sensor {
	return nil
}

// compile time check
var _ Registry = &registry{}

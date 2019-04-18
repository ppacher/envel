package sensors

import (
	"sync"
)

// Value wraps a value monitored/measured by a sensor
type Value interface {
	// Name is the name of the sensor value
	Name() string

	// Unit eventually returns the unit of the sensor
	Unit() string

	// Current returns the current value
	Current() interface{}
}

// Sensor describes a sensor
type Sensor interface {
	// Name is the name of the sensor
	Name() string

	// Description eventually returns the description of the sensor
	Description() string

	// Labels returns any labels set to the sensor
	Lables() map[string]string

	// Values returns all values the sensor monitores
	Values() []Value

	// ValueByName returns the value with the given name or nil
	ValueByName(string) Value
}

// value implements the Value interface
type value struct {
	name string
	unit string

	valueLock    sync.RWMutex
	currentValue interface{}
}

// Name returns the name of the value
func (val *value) Name() string {
	return val.name
}

// Unit returns the unit of the value
func (val *value) Unit() string {
	return val.unit
}

// Current returns the current sensor value
func (val *value) Current() interface{} {
	val.valueLock.RLock()
	defer val.valueLock.RUnlock()

	return val.currentValue
}

// Set sets a new value
func (val *value) Set(v interface{}) {
	val.valueLock.Lock()
	defer val.valueLock.Unlock()

	val.currentValue = v
}

// sensor implements the Sensor interface
type sensor struct {
	name        string
	description string
	labels      map[string]string
	values      map[string]*value
}

// Name returns the name of the sensor
func (s *sensor) Name() string {
	return s.name
}

// Description returns the description of the sensor
func (s *sensor) Description() string {
	return s.description
}

// Labels returns the sensors labels
func (s *sensor) Lables() map[string]string {
	copy := make(map[string]string, len(s.labels))

	for name, value := range s.labels {
		copy[name] = value
	}

	return copy
}

// Values returns all values the sensor is monitoring/measuring
func (s *sensor) Values() []Value {
	values := make([]Value, len(s.values))

	for _, val := range s.values {
		values = append(values, val)
	}

	return values
}

// ValueByName returns the value with the given name or nil
func (s *sensor) ValueByName(name string) Value {
	return s.values[name]
}

// compile time checks
var _ Value = &value{}
var _ Sensor = &sensor{}

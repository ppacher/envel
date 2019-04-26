package plugin

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"plugin"
	"syscall"

	"github.com/sirupsen/logrus"
)

// Loader provides methods for loading plugins
type Loader interface {
	// LoadDirectory loads all plugins from the given directory
	// If the specified path is a file, it should try to directly load the file
	LoadDirectory(path string) ([]Instance, error)

	// LoadFile loads an envel plugin from the given file
	LoadFile(path string) (Instance, error)
}

type defaultLoader struct{}

// DefaultLoader is the default plugin loader used by LoadDirectory() and LoadFile()
var DefaultLoader = &defaultLoader{}

func (l *defaultLoader) LoadDirectory(path string) ([]Instance, error) {
	content, err := ioutil.ReadDir(path)
	if err != nil {
		if syscallError, ok := err.(*os.SyscallError); ok {
			if errno, ok := syscallError.Err.(syscall.Errno); ok && errno == syscall.ENOTDIR {
				res, err := l.LoadFile(path)
				return []Instance{res}, err
			}
		}
		return nil, err
	}

	var loadedPlugins []Instance

	for _, entry := range content {
		// we skip any sub-directories
		if entry.IsDir() {
			continue
		}

		p, err := l.LoadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			// skip the plugin
			logrus.Warnf("failed to load plugin %s: %s", entry.Name(), err.Error())
			continue
		}

		loadedPlugins = append(loadedPlugins, p)
	}

	return loadedPlugins, nil
}

// LoadFile loads an evenl plugin from the given file
func (l *defaultLoader) LoadFile(path string) (Instance, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	sym, err := p.Lookup(SymbolName)
	if err != nil {
		return nil, err
	}

	if plugin, ok := sym.(*Plugin); ok {
		return &instance{
			Plugin: *plugin,
			name:   filepath.Base(path),
			path:   path,
		}, nil
	}

	return nil, fmt.Errorf("Symbol found but it does not implement the Plugin interface")
}

// LoadDirectory loads all plugins from the given directory.
// It used DefaultLoader as the plugin Loader
func LoadDirectory(path string) ([]Instance, error) {
	return DefaultLoader.LoadDirectory(path)
}

// LoadFile loads an envel plugin from the given file.
// It uses DefaultLoader as the plugin Loader
func LoadFile(path string) (Instance, error) {
	return DefaultLoader.LoadFile(path)
}

// compile time check if the interface is implemented correctly
var _ Loader = DefaultLoader

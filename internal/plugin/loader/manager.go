package pluginconf

import (
	"fmt"
	"plugin"
	"sync"
)

var (
	registry = make(map[string]Plugin)
	mu       sync.Mutex
)

// loads a plugin into memory from a .so file
func LoadPlugin(path string, name string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("error opening plugin: %v", err)
	}

	// Look up the symbol "PluginInstance" in the plugin
	sym, err := p.Lookup("PluginInstance")
	if err != nil {
		return fmt.Errorf("error looking up symbol: %v", err)
	}
	// Verify the type of the symbol
	pluginPtr, ok := sym.(*Plugin)
	if !ok {
		return fmt.Errorf("type %T does not implement Plugin", sym)
	}

	pluginInstance := *pluginPtr

	// Register the plugin in our list of plugins (Registry)
	mu.Lock()
	registry[name] = pluginInstance
	mu.Unlock()

	return nil
}

// returns a registered plugin by its name
func GetPlugin(name string) (Plugin, error) {
	mu.Lock()
	defer mu.Unlock()

	// Look for the plugin in the registry
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("plugin not found")
	}
	return p, nil
}

package pluginconf

import (
	"context"
	"fmt"
	"os"
	"plugin"
	"sync"

	"github.com/sirupsen/logrus"
)

var (
	registry = make(map[string]Plugin)
	mu       sync.Mutex

	// GetPluginFunc is a variable that allows mocking the GetPlugin function in tests
	GetPluginFunc = defaultGetPlugin
)

// LoadPlugin loads a plugin into memory from a .so file
func LoadPlugin(ctx context.Context, path string, name string, config map[string]interface{}, logger *logrus.Logger) error {
	// Check if plugin file exists before attempting to load
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("plugin file '%s' does not exist", path)
	}

	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("error opening plugin '%s': %w", path, err)
	}

	// Look up the symbol "PluginInstance" in the plugin
	sym, err := p.Lookup("PluginInstance")
	if err != nil {
		return fmt.Errorf("error looking up symbol 'PluginInstance' in plugin '%s': %w", name, err)
	}
	// Verify the type of the symbol
	pluginPtr, ok := sym.(*Plugin)
	if !ok {
		return fmt.Errorf("type %T does not implement Plugin interface", sym)
	}

	pluginInstance := *pluginPtr

	if err := pluginInstance.Initialize(ctx, config, logger); err != nil {
		return fmt.Errorf("error initializing plugin: '%s': %w", name, err)
	}

	// Register the plugin in our list of plugins (Registry)
	mu.Lock()
	registry[name] = pluginInstance
	mu.Unlock()

	return nil
}

// Implementación por defecto de GetPlugin
func defaultGetPlugin(name string) (Plugin, error) {
	mu.Lock()
	defer mu.Unlock()

	// Look for the plugin in the registry
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("plugin not found")
	}
	return p, nil
}

// GetPlugin returns a registered plugin by its name
func GetPlugin(name string) (Plugin, error) {
	return GetPluginFunc(name)
}

// GetMetricsFunc checks if a metrics function exists by name
func GetMetricsFunc(funcName string) (interface{}, error) {
	return nil, fmt.Errorf("metrics function not accessible")
}

// UpdateMetric updates a metric by name with a value
func UpdateMetric(funcName string, value float64) error {
	return fmt.Errorf("metrics update not implemented")
}

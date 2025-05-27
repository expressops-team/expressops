// Package v1alpha1 provides API types for the configuration of ExpressOps
// api/v1alpha1/config_types.go
package v1alpha1

// Config represents the root configuration structure for the application
type Config struct {
	Logging LoggingConfig `yaml:"logging"`
	Server  ServerConfig  `yaml:"server"`
	Plugins []Plugin      `yaml:"plugins"`
	Flows   []Flow        `yaml:"flows"`
}

// LoggingConfig represents the logging-related configuration options
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// ServerConfig represents the server-related configuration options
type ServerConfig struct {
	Port       int        `yaml:"port"`
	Address    string     `yaml:"address"`
	TimeoutSec int        `yaml:"timeoutSeconds"`
	HTTP       HTTPConfig `yaml:"http"`
}

// HTTPConfig represents HTTP-specific configuration settings
type HTTPConfig struct {
	ProtocolVersion int `yaml:"protocolVersion"`
}

// Plugin represents a plugin configuration entry
type Plugin struct {
	Name   string                 `yaml:"name"`
	Path   string                 `yaml:"path"`
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
}

// Flow represents a workflow definition
type Flow struct {
	Name          string `yaml:"name"`
	CustomHandler string `yaml:"customHandler,omitempty"`
	Pipeline      []Step `yaml:"pipeline"`
}

// Step represents each step in a flow pipeline
type Step struct {
	PluginRef  string                 `yaml:"pluginRef"`
	Parameters map[string]interface{} `yaml:"parameters,omitempty"`
}

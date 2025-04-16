// api/v1alpha1/config_types.go
package v1alpha1

// Config represents the configuration defined in the YAML file.
type Config struct {
	Logging LoggingConfig `yaml:"logging"`
	Server  ServerConfig  `yaml:"server"`
	Plugins []Plugin      `yaml:"plugins"`
	Flows   []Flow        `yaml:"flows"`
}

// represents the "logging" section
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// server section
type ServerConfig struct {
	Port       int        `yaml:"port" default:"8080"`
	Address    string     `yaml:"address" default:"0.0.0.0"`
	TimeoutSec int        `yaml:"timeoutSeconds" default:"4"`
	HTTP       HTTPConfig `yaml:"http"`
}

// http section
type HTTPConfig struct {
	ProtocolVersion int `yaml:"protocolVersion"`
}

// plugins section
type Plugin struct {
	Name   string                 `yaml:"name"`
	Path   string                 `yaml:"path"`
	Type   string                 `yaml:"type"`
	Config map[string]interface{} `yaml:"config"`
}

// flow section
type Flow struct {
	Name          string `yaml:"name"`
	CustomHandler string `yaml:"customHandler,omitempty"`
	Pipeline      []Step `yaml:"pipeline"`
}

// represents each step in the pipeline
type Step struct {
	PluginRef  string                 `yaml:"pluginRef"`
	Parameters map[string]interface{} `yaml:"parameters,omitempty"`
}

package v1alpha1

import (
	"encoding/json"
	"testing"
	"gopkg.in/yaml.v2"
	
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigDeserialization(t *testing.T) {
	// YAML configuration example.
	yamlConfig := `
logging:
  level: info
  format: json
server:
  port: 8080
  address: 0.0.0.0
  timeoutSeconds: 10
  http:
    protocolVersion: 1
plugins:
  - name: test-plugin
    path: /path/to/plugin
    type: go
    config:
      key1: value1
      key2: 42
flows:
  - name: test-flow
    pipeline:
      - pluginRef: test-plugin
        parameters:
          param1: value1
          param2: value2
`
	
	// Deserialize YAML
	var config Config
	err := yaml.Unmarshal([]byte(yamlConfig), &config)
	require.NoError(t, err)
	
	// Check logging fields
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "json", config.Logging.Format)
	
	// Check server fields
	assert.Equal(t, 8080, config.Server.Port)
	assert.Equal(t, "0.0.0.0", config.Server.Address)
	assert.Equal(t, 10, config.Server.TimeoutSec)
	assert.Equal(t, 1, config.Server.HTTP.ProtocolVersion)
	
	// Check plugins
	require.Len(t, config.Plugins, 1)
	assert.Equal(t, "test-plugin", config.Plugins[0].Name)
	assert.Equal(t, "/path/to/plugin", config.Plugins[0].Path)
	assert.Equal(t, "go", config.Plugins[0].Type)
	assert.Equal(t, "value1", config.Plugins[0].Config["key1"])
	
	// Obtener el valor y comprobar solo el valor numérico, independientemente del tipo
	key2Value := config.Plugins[0].Config["key2"]
	switch v := key2Value.(type) {
	case float64:
		assert.Equal(t, float64(42), v)
	case int:
		assert.Equal(t, 42, v)
	default:
		assert.Fail(t, "El valor key2 debería ser un número (float64 o int)")
	}
	
	// Verify flows
	require.Len(t, config.Flows, 1)
	assert.Equal(t, "test-flow", config.Flows[0].Name)
	require.Len(t, config.Flows[0].Pipeline, 1)
	assert.Equal(t, "test-plugin", config.Flows[0].Pipeline[0].PluginRef)
	assert.Equal(t, "value1", config.Flows[0].Pipeline[0].Parameters["param1"])
	assert.Equal(t, "value2", config.Flows[0].Pipeline[0].Parameters["param2"])
}

func TestConfigSerialization(t *testing.T) {
	// Create a sample configuration
	config := Config{
		Logging: LoggingConfig{
			Level:  "debug",
			Format: "text",
		},
		Server: ServerConfig{
			Port:       9000,
			Address:    "localhost",
			TimeoutSec: 5,
			HTTP: HTTPConfig{
				ProtocolVersion: 2,
			},
		},
		Plugins: []Plugin{
			{
				Name: "plugin1",
				Path: "/plugins/plugin1.so",
				Type: "go",
				Config: map[string]interface{}{
					"option1": "value1",
					"option2": 123,
				},
			},
		},
		Flows: []Flow{
			{
				Name: "flow1",
				Pipeline: []Step{
					{
						PluginRef: "plugin1",
						Parameters: map[string]interface{}{
							"param1": "value1",
						},
					},
				},
			},
		},
	}
	
	// Serialize to YAML
	yamlData, err := yaml.Marshal(config)
	require.NoError(t, err)
	
	// Deserialize again and verify
	var configFromYaml Config
	err = yaml.Unmarshal(yamlData, &configFromYaml)
	require.NoError(t, err)
	
	// Verify that the structure remains intact
	assert.Equal(t, config.Logging.Level, configFromYaml.Logging.Level)
	assert.Equal(t, config.Logging.Format, configFromYaml.Logging.Format)
	
	assert.Equal(t, config.Server.Port, configFromYaml.Server.Port)
	assert.Equal(t, config.Server.Address, configFromYaml.Server.Address)
	assert.Equal(t, config.Server.TimeoutSec, configFromYaml.Server.TimeoutSec)
	assert.Equal(t, config.Server.HTTP.ProtocolVersion, configFromYaml.Server.HTTP.ProtocolVersion)
	
	// Serialize to JSON (for compatibility)
	jsonData, err := json.Marshal(config)
	require.NoError(t, err)
	
	// Deserialize from JSON and verify
	var configFromJSON Config
	err = json.Unmarshal(jsonData, &configFromJSON)
	require.NoError(t, err)
	
	// Verify that the structure is maintained after JSON
	assert.Equal(t, config.Logging.Level, configFromJSON.Logging.Level)
	assert.Equal(t, config.Plugins[0].Name, configFromJSON.Plugins[0].Name)
	assert.Equal(t, config.Flows[0].Name, configFromJSON.Flows[0].Name)
}
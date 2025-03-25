package pluginconf

type Plugin interface {
	Initialize(config map[string]interface{}) error
	Execute(params map[string]interface{}) (interface{}, error)
}

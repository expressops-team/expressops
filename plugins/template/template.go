// This is the format that a plugin MUST FOLLOW
package template

import (
	"context"
	"fmt"
	"net/http"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type TemplatePlugin struct {
	logger *logrus.Logger
}

func (p *TemplatePlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
}

func (p *TemplatePlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	return nil, nil
}

func (p *TemplatePlugin) FormatResult(result interface{}) (string, error) {
	return fmt.Sprintf("%v", result), nil
}

var PluginInstance pluginconf.Plugin = &TemplatePlugin{}

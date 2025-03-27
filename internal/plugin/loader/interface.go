package pluginconf

import (
	"context"

	"github.com/sirupsen/logrus"
)

type Plugin interface {
	Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error
	Execute(ctx context.Context, params map[string]interface{}) (interface{}, error)
	FormatResult(result interface{}) (string, error)
}

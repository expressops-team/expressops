package pluginconf

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

type Plugin interface {
	Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error
	Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error)
	FormatResult(result interface{}) (string, error)
}

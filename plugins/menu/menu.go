package main

import (
	"context"
	"fmt"
	"net/http"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type Menu struct {
	logger *logrus.Logger
	flows  []string
}

func (m *Menu) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	m.logger = logger
	return nil
}

func (m *Menu) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	if shared == nil || (*shared)["flows"] == nil {
		return nil, fmt.Errorf("no flows available")
	}

	flows, ok := (*shared)["flows"].([]string)
	if !ok {
		return nil, fmt.Errorf("invalid flows format")
	}

	fmt.Println("\n🔄 Available flows:")
	for i, flow := range flows {
		fmt.Printf("%d. ▶️ %s\n", i+1, flow)
	}

	var selection int
	fmt.Print("\nSelect a flow to execute (1-", len(flows), "): ")
	fmt.Scanf("%d", &selection)

	if selection < 1 || selection > len(flows) {
		return nil, fmt.Errorf("invalid selection")
	}

	selectedFlow := flows[selection-1]
	m.logger.Infof("Selected flow: %s", selectedFlow)

	return map[string]interface{}{
		"selected_flow": selectedFlow,
	}, nil
}

// FormatResult formats the result of the execution
func (m *Menu) FormatResult(result interface{}) (string, error) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if flow, ok := resultMap["selected_flow"].(string); ok {
			return fmt.Sprintf("Selected flow: %s", flow), nil
		}
	}
	return "Menu selection completed", nil
}

var PluginInstance pluginconf.Plugin = &Menu{}

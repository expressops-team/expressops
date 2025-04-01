// to fix:
// - add a way to get the flows from the config
// - add a way to get the port from the config
// - choose the flow from where i execute curl
package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type Menu struct {
	logger *logrus.Logger
	flows  []string
}

func (m *Menu) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	m.logger = logger

	// harcoded rn
	m.flows = []string{
		"incident-flow",
		"healthz",
		"test-context",
		"dr-house",
		"weekly-cleanup",
		"alert-flow",
	}

	return nil
}

func (m *Menu) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	if r != nil && r.URL.Query().Get("flows") != "" {
		customFlows := r.URL.Query().Get("flows")
		m.logger.Infof("Custom flows provided: %s", customFlows)
	}

	fmt.Println("\n🔄 Available flows:")
	flowMap := make(map[int]string)

	for i, flow := range m.flows {
		flowMap[i+1] = flow
		fmt.Printf("%d. ▶️ %s\n", i+1, flow)
	}

	var selection int
	fmt.Print("\nSelect a flow to execute (1-", len(m.flows), "): ")
	fmt.Scanf("%d", &selection)

	if selection < 1 || selection > len(m.flows) {
		return nil, fmt.Errorf("invalid selection %d, must be between 1 and %d", selection, len(m.flows))
	}

	selectedFlow := flowMap[selection]
	m.logger.Infof("Selected flow: %s", selectedFlow)

	port := 8080 //in case of change the port
	url := fmt.Sprintf("http://localhost:%d/flow?flowName=%s", port, selectedFlow)

	m.logger.Infof("Executing flow via curl: %s", url)
	cmd := exec.Command("curl", "-s", url)
	output, err := cmd.CombinedOutput()

	if err != nil {
		m.logger.Errorf("Error executing flow: %v", err)
		return nil, fmt.Errorf("error executing flow: %v", err)
	}

	m.logger.Infof("Flow execution result: %s", string(output))

	return map[string]interface{}{
		"selected_flow": selectedFlow,
		"executed":      true,
		"result":        string(output),
	}, nil
}

func (m *Menu) FormatResult(result interface{}) (string, error) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		if flow, ok := resultMap["selected_flow"].(string); ok {
			if executed, ok := resultMap["executed"].(bool); ok && executed {
				return fmt.Sprintf("✅ Executed flow: %s", flow), nil
			}
			return fmt.Sprintf("Selected flow: %s", flow), nil
		}
	}
	return "Menu selection completed", nil
}

var PluginInstance pluginconf.Plugin = &Menu{}

// ==================================================================================================================
// func (m *Menu) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
// 	m.logger = logger
// 	return nil
// }
//
// func (m *Menu) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
// 	if shared == nil || (*shared)["flows"] == nil {
// 		return nil, fmt.Errorf("no flows available")
// 	}
//
// 	flows, ok := (*shared)["flows"].([]string)
// 	if !ok {
// 		return nil, fmt.Errorf("invalid flows format")
// 	}
//
// 	fmt.Println("\n🔄 Available flows:")
// 	for i, flow := range flows {
// 		fmt.Printf("%d. ▶️ %s\n", i+1, flow)
// 	}
//
// 	var selection int
// 	fmt.Print("\nSelect a flow to execute (1-", len(flows), "): ")
// 	fmt.Scanf("%d", &selection)
//
// 	if selection < 1 || selection > len(flows) {
// 		return nil, fmt.Errorf("invalid selection")
// 	}
//
// 	selectedFlow := flows[selection-1]
// 	m.logger.Infof("Selected flow: %s", selectedFlow)
//
// 	return map[string]interface{}{
// 		"selected_flow": selectedFlow,
// 	}, nil
// }
//
//
// func (m *Menu) FormatResult(result interface{}) (string, error) {
// 	if resultMap, ok := result.(map[string]interface{}); ok {
// 		if flow, ok := resultMap["selected_flow"].(string); ok {
// 			return fmt.Sprintf("Selected flow: %s", flow), nil
// 		}
// 	}
// 	return "Menu selection completed", nil
// }
//
// var PluginInstance pluginconf.Plugin = &Menu{}

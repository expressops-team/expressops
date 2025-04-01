// to fix:
// - add a way to get the flows from the config
// - add a way to get the port from the config
// - choose the flow from where i execute curl
// IGNORE THIS PLUGIN 
package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type Menu struct {
	logger *logrus.Logger
	flows  []string
	port   int
}

func (m *Menu) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	m.logger = logger

	// Default values in case we can't get them from config
	m.port = 8080

	// Check if port is provided in config
	if portValue, ok := config["port"].(float64); ok {
		m.port = int(portValue)
	}

	m.flows = []string{
		"incident-flow",
		"healthz",
		"test-context",
		"dr-house",
		"weekly-cleanup",
		"alert-flow",
		"logs-view",
	}

	return nil
}

func (m *Menu) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	// Try to get configured flows from shared context
	if shared != nil {
		// Try to get server port from shared context
		if serverConfig, ok := (*shared)["server"].(map[string]interface{}); ok {
			if port, ok := serverConfig["port"].(int); ok {
				m.port = port
				m.logger.Infof("Using port from config: %d", m.port)
			}
		}

		// Try to get flows from context
		if flowsList, ok := (*shared)["flows"].([]interface{}); ok {
			configFlows := make([]string, 0, len(flowsList))

			for _, flow := range flowsList {
				if flowMap, ok := flow.(map[string]interface{}); ok {
					if name, ok := flowMap["name"].(string); ok {
						configFlows = append(configFlows, name)
					}
				}
			}

			if len(configFlows) > 0 {
				m.flows = configFlows
				m.logger.Infof("Using flows from config: %v", m.flows)
			}
		}
	}

	// Get any custom flows passed in URL
	if r != nil && r.URL.Query().Get("flows") != "" {
		customFlows := r.URL.Query().Get("flows")
		customFlowsList := strings.Split(customFlows, ",")

		// Only use custom flows if valid
		if len(customFlowsList) > 0 {
			m.flows = customFlowsList
			m.logger.Infof("Using custom flows: %v", m.flows)
		}
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

	url := fmt.Sprintf("http://localhost:%d/flow?flowName=%s", m.port, selectedFlow)

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

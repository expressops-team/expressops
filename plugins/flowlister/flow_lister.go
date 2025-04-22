package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"expressops/api/v1alpha1"
	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type FlowListerPlugin struct {
	logger *logrus.Logger
	config map[string]interface{}
}

// FlowRegistry shared with the server package
var FlowRegistry map[string]v1alpha1.Flow

func (p *FlowListerPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.config = config

	logger.Info("Flow Lister Plugin initialized successfully")
	return nil
}

func (p *FlowListerPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	p.logger.Info("Flow Lister Plugin executing")

	var flows map[string]v1alpha1.Flow
	if registry, ok := (*shared)["flow_registry"].(map[string]v1alpha1.Flow); ok {
		flows = registry
	} else {
		p.logger.Warn("Flow registry not found in shared map")
		flows = make(map[string]v1alpha1.Flow)
	}

	// Create a result object with the list of flows
	result := map[string]interface{}{
		"count":    len(flows),
		"flows":    []map[string]interface{}{},
		"flowList": []string{}, // New field for separate log lines
	}

	// Get a sorted list of flow names for consistent output
	var flowNames []string
	for name := range flows {
		flowNames = append(flowNames, name)
	}
	sort.Strings(flowNames)

	// Create the flow list
	var flowList []map[string]interface{}
	var logLines []string // Store each flow as a separate log line

	// Add header to log lines
	logLines = append(logLines, fmt.Sprintf("Available Flows (%d):", len(flows)))
	logLines = append(logLines, "=====================")
	logLines = append(logLines, "") // Empty line

	for _, name := range flowNames {
		flow := flows[name]
		flowInfo := map[string]interface{}{
			"name":         name,
			"description":  flow.CustomHandler,
			"plugin_count": len(flow.Pipeline),
			"plugins":      []string{},
		}

		// Add the list of plugins in this flow
		var plugins []string
		for _, step := range flow.Pipeline {
			if step.PluginRef != "" {
				plugins = append(plugins, step.PluginRef)
			}
		}
		flowInfo["plugins"] = plugins

		flowList = append(flowList, flowInfo)

		// Create a separate log line for this flow
		flowLine := fmt.Sprintf("📋 %s", name)
		logLines = append(logLines, flowLine)

		if flow.CustomHandler != "" {
			descLine := fmt.Sprintf("   Description: %s", flow.CustomHandler)
			logLines = append(logLines, descLine)
		}

		pluginsLine := fmt.Sprintf("   Plugins (%d): ", len(plugins))

		for i, plugin := range plugins {
			if i > 0 {
				pluginsLine += " → "
			}
			pluginsLine += plugin
		}

		logLines = append(logLines, pluginsLine)
		logLines = append(logLines, "") // Empty line between flows
	}

	result["flows"] = flowList
	result["flowList"] = logLines

	return result, nil
}

// Format the result for separate log lines
func (p *FlowListerPlugin) FormatResult(result interface{}) (string, error) {
	if result == nil {
		return "No flow information available", nil
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected result format")
	}

	// Get the list of log lines
	logLines, ok := data["flowList"].([]string)
	if !ok {
		return "Could not format flow list", nil
	}

	// Use special separator that will be recognized by the server
	return "__MULTILINE_LOG__" + strings.Join(logLines, "__MULTILINE_LOG__"), nil
}

// important to export the plugin, same interface as the other plugins
var PluginInstance pluginconf.Plugin = &FlowListerPlugin{}

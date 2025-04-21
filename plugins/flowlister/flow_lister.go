package main

import (
	"context"
	"fmt"
	"net/http"
	"sort"

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
		"count": len(flows),
		"flows": []map[string]interface{}{},
	}

	// Get a sorted list of flow names for consistent output
	var flowNames []string
	for name := range flows {
		flowNames = append(flowNames, name)
	}
	sort.Strings(flowNames)

	// Create the flow list
	var flowList []map[string]interface{}
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
	}

	result["flows"] = flowList

	return result, nil
}

// Format the result as a complete list
func (p *FlowListerPlugin) FormatResult(result interface{}) (string, error) {
	if result == nil {
		return "No flow information available", nil
	}

	data, ok := result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected result format")
	}

	output := fmt.Sprintf("Available Flows (%d):\n", data["count"])
	output += "=====================\n\n"

	flows, ok := data["flows"].([]map[string]interface{})
	if !ok {
		return output + "No flows found", nil
	}

	for _, flow := range flows {
		output += fmt.Sprintf("📋 %s\n", flow["name"])

		if desc, ok := flow["description"].(string); ok && desc != "" {
			output += fmt.Sprintf("   Description: %s\n", desc)
		}

		output += fmt.Sprintf("   Plugins (%d): ", flow["plugin_count"])

		plugins, ok := flow["plugins"].([]string)
		if ok {
			for i, plugin := range plugins {
				if i > 0 {
					output += " → "
				}
				output += plugin
			}
		}

		output += "\n\n"
	}

	return output, nil
}

// important to export the plugin, same interface as the other plugins
var PluginInstance pluginconf.Plugin = &FlowListerPlugin{}

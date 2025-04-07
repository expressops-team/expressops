package main

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

// Feature toggle - set to 0 to disable permissions and show GCP message
var enablePermissionsFeature = 0

// to change it
var DefaultConfig = struct {
	DefaultUsername    string
	DefaultPaths       []string
	DefaultPermissions string
}{
	DefaultUsername:    "example-user",
	DefaultPaths:       []string{"it-school-2025-2-nacho", "it-school-2025-3-david"},
	DefaultPermissions: "rwx",
}

type PermissionsPlugin struct {
	logger          *logrus.Logger
	config          map[string]interface{}
	defaultUsername string
	defaultPaths    []string
	defaultPerms    string
	baseDir         string
}

func (p *PermissionsPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.config = config
	p.logger.Info("Initializing Permissions Plugin")

	p.defaultUsername = DefaultConfig.DefaultUsername
	p.defaultPaths = DefaultConfig.DefaultPaths
	p.defaultPerms = DefaultConfig.DefaultPermissions

	if username, ok := config["default_username"].(string); ok && username != "" {
		p.defaultUsername = username
		p.logger.Infof("Setting default username to: %s", username)
	}

	if paths, ok := config["default_paths"].([]interface{}); ok && len(paths) > 0 {
		p.defaultPaths = make([]string, 0, len(paths))
		for _, path := range paths {
			if pathStr, ok := path.(string); ok && pathStr != "" {
				p.defaultPaths = append(p.defaultPaths, pathStr)
			}
		}
		p.logger.Infof("Setting default paths to: %v", p.defaultPaths)
	}

	if perms, ok := config["default_permissions"].(string); ok && perms != "" {
		p.defaultPerms = perms
		p.logger.Infof("Setting default permissions to: %s", perms)
	}

	if baseDir, ok := config["base_directory"].(string); ok && baseDir != "" {
		p.baseDir = baseDir
		p.logger.Infof("Setting base directory to: %s", baseDir)
	}

	return nil
}

func (p *PermissionsPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	p.logger.Info("Executing Permissions Plugin")

	// Check feature toggle
	if enablePermissionsFeature == 0 {
		p.logger.Info("Permissions feature is disabled, returning GCP integration message")
		message := map[string]interface{}{
			"message": "🚧 Próximamente: Integración con Google Cloud Platform (GCP) 🚧\n\n" +
				"🔄 Working on implementing permissions directly to GCP.\n" +
				"🔐 This functionality will allow managing permissions at the project and folder level.\n" +
				"📅 Available in the future.\n\n" +
				"👨‍💻 Att. David and Nacho",
		}
		return message, nil
	}

	username := p.defaultUsername
	if userVal, ok := (*shared)["username"].(string); ok && userVal != "" {
		username = userVal
	}

	paths := p.defaultPaths
	if pathsVal, ok := (*shared)["paths"].([]string); ok && len(pathsVal) > 0 {
		paths = pathsVal
	} else if pathsVal, ok := (*shared)["paths"].([]interface{}); ok && len(pathsVal) > 0 {
		// Convert from []interface{} to []string if needed
		paths = make([]string, len(pathsVal))
		for i, p := range pathsVal {
			if str, ok := p.(string); ok {
				paths[i] = str
			}
		}
	}

	permissions := p.defaultPerms
	if permVal, ok := (*shared)["permissions"].(string); ok && permVal != "" {
		permissions = permVal
	}

	results := make(map[string]interface{})
	errors := []string{}

	// Change permissions for each path
	for _, path := range paths {
		fullPath := path
		if p.baseDir != "" {
			fullPath = filepath.Join(p.baseDir, path)
		}

		p.logger.Infof("Changing permissions for user %s on path %s", username, fullPath)

		// Execute the chmod command to give user permissions
		// We use setfacl to modify ACLs for greater flexibility
		cmd := exec.CommandContext(ctx, "setfacl", "-m", fmt.Sprintf("u:%s:%s", username, permissions), fullPath)
		output, err := cmd.CombinedOutput()

		pathResult := map[string]interface{}{
			"path":        fullPath,
			"permissions": permissions,
			"success":     err == nil,
		}

		if err != nil {
			errMsg := fmt.Sprintf("Failed to change permissions for %s: %v - %s", fullPath, err, string(output))
			p.logger.Error(errMsg)
			pathResult["error"] = errMsg
			errors = append(errors, errMsg)
		} else {
			p.logger.Infof("Successfully set permissions for %s on %s", username, fullPath)
			pathResult["output"] = string(output)
		}

		results[path] = pathResult
	}

	response := map[string]interface{}{
		"username":   username,
		"paths":      paths,
		"results":    results,
		"successful": len(errors) == 0,
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	return response, nil
}

// FormatResult creates a human-readable response
func (p *PermissionsPlugin) FormatResult(result interface{}) (string, error) {
	if result == nil {
		return "No result received", nil
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("%v", result), nil
	}

	// Check if this is a GCP integration message
	if msg, ok := resultMap["message"].(string); ok {
		if _, hasUsername := resultMap["username"]; !hasUsername {
			// This is our GCP message since it doesn't have username field
			return msg, nil
		}
	}
	//temporary message
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🔐 Permission Changes for User: %s\n\n", resultMap["username"]))

	if success, ok := resultMap["successful"].(bool); ok && success {
		sb.WriteString("✅ All permission changes successful!\n\n")
	} else {
		sb.WriteString("⚠️ Some permission changes failed!\n\n")
	}

	if results, ok := resultMap["results"].(map[string]interface{}); ok {
		sb.WriteString("Results by path:\n")
		for path, result := range results {
			pathResult, ok := result.(map[string]interface{})
			if !ok {
				continue
			}

			success := pathResult["success"].(bool)
			if success {
				sb.WriteString(fmt.Sprintf("  ✅ %s: Permissions set to %s\n", path, pathResult["permissions"]))
			} else {
				sb.WriteString(fmt.Sprintf("  ❌ %s: Failed - %s\n", path, pathResult["error"]))
			}
		}
	}

	return sb.String(), nil
}

var PluginInstance pluginconf.Plugin = &PermissionsPlugin{}

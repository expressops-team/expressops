// https://gist.github.com/salrashid123/e894e856c2851fe437eee5fc2b72c8ad
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

var enableUserCreationFeature = 0

// Default configuration for the plugin
var DefaultConfig = struct {
	DefaultUsername    string
	DefaultGroups      []string
	DefaultHomeDirBase string
	DefaultShell       string
}{
	DefaultUsername:    "example-user",
	DefaultGroups:      []string{"users", "developers"},
	DefaultHomeDirBase: "/home",
	DefaultShell:       "/bin/bash",
}

type UserCreationPlugin struct {
	logger             *logrus.Logger
	config             map[string]interface{}
	defaultUsername    string
	defaultGroups      []string
	defaultHomeDirBase string
	defaultShell       string
}

func (p *UserCreationPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.config = config
	p.logger.Info("Initializing User Creation Plugin")

	// Initialize with defaults
	p.defaultUsername = DefaultConfig.DefaultUsername
	p.defaultGroups = DefaultConfig.DefaultGroups
	p.defaultHomeDirBase = DefaultConfig.DefaultHomeDirBase
	p.defaultShell = DefaultConfig.DefaultShell

	// Override with config if provided
	if username, ok := config["default_username"].(string); ok && username != "" {
		p.defaultUsername = username
		p.logger.Infof("Setting default username to: %s", username)
	}

	if groups, ok := config["default_groups"].([]interface{}); ok && len(groups) > 0 {
		p.defaultGroups = make([]string, 0, len(groups))
		for _, group := range groups {
			if groupStr, ok := group.(string); ok && groupStr != "" {
				p.defaultGroups = append(p.defaultGroups, groupStr)
			}
		}
		p.logger.Infof("Setting default groups to: %v", p.defaultGroups)
	}

	if homeDir, ok := config["default_homedir_base"].(string); ok && homeDir != "" {
		p.defaultHomeDirBase = homeDir
		p.logger.Infof("Setting default home directory base to: %s", homeDir)
	}

	if shell, ok := config["default_shell"].(string); ok && shell != "" {
		p.defaultShell = shell
		p.logger.Infof("Setting default shell to: %s", shell)
	}

	return nil
}

func (p *UserCreationPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	p.logger.Info("Executing User Creation Plugin")

	// Check feature toggle
	if enableUserCreationFeature == 0 {
		p.logger.Info("User creation feature is disabled, returning info message")
		message := map[string]interface{}{
			"message": "👤 User Account Creation Service 👤\n\n" +
				"🛠️ Creating user accounts for IT School participants\n" +
				"🔑 This would normally create Linux user accounts in the system\n" +
				"📋 Groups: developers, users\n" +
				"🏠 Home directory: /home/{username}\n\n" +
				"⚠️ Running in simulation mode - no actual users created\n" +
				"✅ This is how it will work in the future ;D",
		}
		return message, nil
	}

	username := p.defaultUsername
	if userVal, ok := (*shared)["username"].(string); ok && userVal != "" {
		username = userVal
	}

	// Get groups from shared context or use default
	groups := p.defaultGroups
	if groupsVal, ok := (*shared)["groups"].([]string); ok && len(groupsVal) > 0 {
		groups = groupsVal
	} else if groupsVal, ok := (*shared)["groups"].([]interface{}); ok && len(groupsVal) > 0 {
		// Convert from []interface{} to []string
		groups = make([]string, len(groupsVal))
		for i, g := range groupsVal {
			if str, ok := g.(string); ok {
				groups[i] = str
			}
		}
	}

	homeDirBase := p.defaultHomeDirBase
	if homeVal, ok := (*shared)["homedir_base"].(string); ok && homeVal != "" {
		homeDirBase = homeVal
	}

	shell := p.defaultShell
	if shellVal, ok := (*shared)["shell"].(string); ok && shellVal != "" {
		shell = shellVal
	}

	p.logger.Infof("Creating user %s with groups %v, home directory base %s, and shell %s",
		username, groups, homeDirBase, shell)

	// Build the useradd command
	groupsStr := strings.Join(groups, ",")
	cmd := exec.CommandContext(ctx, "useradd",
		"-m",
		"-d", fmt.Sprintf("%s/%s", homeDirBase, username),
		"-s", shell,
		"-G", groupsStr,
		username)

	output, err := cmd.CombinedOutput()

	result := map[string]interface{}{
		"username": username,
		"groups":   groups,
		"homedir":  fmt.Sprintf("%s/%s", homeDirBase, username),
		"shell":    shell,
		"success":  err == nil,
	}

	if err != nil {
		errMsg := fmt.Sprintf("Failed to create user %s: %v - %s", username, err, string(output))
		p.logger.Error(errMsg)
		result["error"] = errMsg
	} else {
		p.logger.Infof("Successfully created user %s", username)
		result["output"] = string(output)

		// Set or create a random password (if user creation succeeded)
		passwdCmd := exec.CommandContext(ctx, "passwd", username)
		passwdOut, passwdErr := passwdCmd.CombinedOutput()

		if passwdErr != nil {
			p.logger.Warnf("Failed to set password for user %s: %v - %s", username, passwdErr, string(passwdOut))
			result["password_set"] = false
			result["password_error"] = fmt.Sprintf("%v - %s", passwdErr, string(passwdOut))
		} else {
			p.logger.Infof("Successfully set password for user %s", username)
			result["password_set"] = true
		}
	}

	// Store username in shared context for other plugins to use
	(*shared)["created_username"] = username

	return result, nil
}

// FormatResult creates a human-readable response
func (p *UserCreationPlugin) FormatResult(result interface{}) (string, error) {
	if result == nil {
		return "No result received", nil
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return fmt.Sprintf("%v", result), nil
	}

	// Check if this is an info message
	if msg, ok := resultMap["message"].(string); ok {
		return msg, nil
	}

	var sb strings.Builder

	username, _ := resultMap["username"].(string)
	success, _ := resultMap["success"].(bool)

	sb.WriteString(fmt.Sprintf("👤 User Creation: %s\n\n", username))

	if success {
		sb.WriteString("✅ User created successfully!\n\n")

		if groups, ok := resultMap["groups"].([]string); ok {
			sb.WriteString(fmt.Sprintf("👥 Groups: %s\n", strings.Join(groups, ", ")))
		}

		if homedir, ok := resultMap["homedir"].(string); ok {
			sb.WriteString(fmt.Sprintf("🏠 Home Directory: %s\n", homedir))
		}

		if shell, ok := resultMap["shell"].(string); ok {
			sb.WriteString(fmt.Sprintf("🐚 Shell: %s\n", shell))
		}

		if passwordSet, ok := resultMap["password_set"].(bool); ok {
			if passwordSet {
				sb.WriteString("🔑 Password: Set successfully\n")
			} else {
				sb.WriteString("⚠️ Password: Not set - manual setup required\n")
			}
		}
	} else {
		sb.WriteString("❌ User creation failed!\n\n")

		if errMsg, ok := resultMap["error"].(string); ok {
			sb.WriteString(fmt.Sprintf("Error: %s\n", errMsg))
		}
	}

	return sb.String(), nil
}

var PluginInstance pluginconf.Plugin = &UserCreationPlugin{}

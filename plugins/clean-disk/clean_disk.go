// plugins/clean-disk/clean_disk.go
package main

import (
	"context"
	"net/http"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

// Default thresholds and settings
var DefaultConfig = struct {
	ThresholdMB    int64
	TargetDirPath  string
	AgeThresholdH  int
	DryRun         bool
	DeletePatterns []string
}{
	ThresholdMB:    500,           // 500 MB
	TargetDirPath:  "/tmp",        // Default target directory
	AgeThresholdH:  24,            // 24 hours
	DryRun:         true,          // Default to dry run for safety
	DeletePatterns: []string{"*"}, // All files by default
}

type CleanDiskPlugin struct {
	logger        *logrus.Logger
	thresholdMB   int64
	targetDirPath string
	ageThresholdH int
	dryRun        bool
	patterns      []string
}

// Initialize sets up the plugin
func (p *CleanDiskPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger
	p.logger.Info("Initializing Clean Disk Plugin")

	// Set default values
	p.thresholdMB = DefaultConfig.ThresholdMB
	p.targetDirPath = DefaultConfig.TargetDirPath
	p.ageThresholdH = DefaultConfig.AgeThresholdH
	p.dryRun = DefaultConfig.DryRun
	p.patterns = DefaultConfig.DeletePatterns

	// Override with config if provided
	if threshold, ok := config["threshold_mb"].(float64); ok {
		p.thresholdMB = int64(threshold)
		p.logger.Infof("Setting threshold to %d MB", p.thresholdMB)
	}

	if targetDir, ok := config["target_dir"].(string); ok && targetDir != "" {
		p.targetDirPath = targetDir
		p.logger.Infof("Setting target directory to: %s", p.targetDirPath)
	}

	if ageHours, ok := config["age_hours"].(float64); ok {
		p.ageThresholdH = int(ageHours)
		p.logger.Infof("Setting age threshold to %d hours", p.ageThresholdH)
	}

	if dryRun, ok := config["dry_run"].(bool); ok {
		p.dryRun = dryRun
		p.logger.Infof("Dry run mode: %v", p.dryRun)
	}

	if patterns, ok := config["delete_patterns"].([]interface{}); ok && len(patterns) > 0 {
		p.patterns = make([]string, 0, len(patterns))
		for _, pattern := range patterns {
			if patternStr, ok := pattern.(string); ok && patternStr != "" {
				p.patterns = append(p.patterns, patternStr)
			}
		}
		p.logger.Infof("Setting delete patterns to: %v", p.patterns)
	}

	return nil
}

// Execute performs disk cleanup based on age
func (p *CleanDiskPlugin) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	result := struct {
		DryRun       bool     `json:"dry_run"`
		FilesDeleted int      `json:"files_deleted"`
		BytesFreed   int64    `json:"bytes_freed"`
		DeletedFiles []string `json:"deleted_files,omitempty"`
	}{
		DryRun:       p.dryRun,
		FilesDeleted: 0,
		BytesFreed:   0,
		DeletedFiles: []string{},
	}

	// Logic to clean up old files would go here
	// This is simplified for the example
	p.logger.Infof("Would clean up files older than %d hours in %s", p.ageThresholdH, p.targetDirPath)

	return result, nil
}

// FormatResult returns a formatted string representation of the result
func (p *CleanDiskPlugin) FormatResult(result interface{}) (string, error) {
	if result == nil {
		return "No cleanup performed", nil
	}

	// Return a formatted string based on the result
	return "clean-disk", nil
}

// Exported plugin instance
var PluginInstance pluginconf.Plugin = &CleanDiskPlugin{}

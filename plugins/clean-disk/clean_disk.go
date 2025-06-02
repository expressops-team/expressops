// plugins/clean-disk/clean_disk.go
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

// DefaultConfig define valores por defecto para el plugin
var DefaultConfig = struct {
	ThresholdMB    int64
	TargetDirPath  string
	AgeThresholdH  int
	DryRun         bool
	DeletePatterns []string
}{
	ThresholdMB:    1000,   // 1GB
	TargetDirPath:  "/tmp", // Default to clean /tmp
	AgeThresholdH:  24,     // 24 hours (1 day)
	DryRun:         false,
	DeletePatterns: []string{"*.tmp", "*.log.*"},
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

	// Limpia el directorio objetivo
	p.logger.Infof("Cleaning files older than %d hours in %s", p.ageThresholdH, p.targetDirPath)

	if p.targetDirPath == "" || p.targetDirPath == "/" {
		return nil, fmt.Errorf("invalid target directory: %s", p.targetDirPath)
	}

	err := cleanDirectory(p.targetDirPath, p.logger)
	if err != nil {
		p.logger.Errorf("Error cleaning directory %s: %v", p.targetDirPath, err)
		return nil, err
	}

	return result, nil
}

func cleanDirectory(dir string, logger *logrus.Logger) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		err := os.RemoveAll(path)
		if err != nil {
			logger.Warnf("Could not delete %s: %v", path, err)
		} else {
			logger.Debugf("Deleted %s", path)
		}
	}
	return nil
}

// FormatResult formats the result of the cleanup operation
func (p *CleanDiskPlugin) FormatResult(result interface{}) (string, error) {
	if result == nil {
		return "No cleanup results", nil
	}

	if res, ok := result.(struct {
		DryRun       bool     `json:"dry_run"`
		FilesDeleted int      `json:"files_deleted"`
		BytesFreed   int64    `json:"bytes_freed"`
		DeletedFiles []string `json:"deleted_files,omitempty"`
	}); ok {
		if res.DryRun {
			return fmt.Sprintf("Dry run: Would have deleted %d files, freeing %d bytes", res.FilesDeleted, res.BytesFreed), nil
		}
		return fmt.Sprintf("Deleted %d files, freed %d bytes", res.FilesDeleted, res.BytesFreed), nil
	}

	return fmt.Sprintf("%v", result), nil
}

var PluginInstance pluginconf.Plugin = &CleanDiskPlugin{}

// Los sábados comprime los logs del día anterior y los guarda en una carpeta llamada "backups(YYYYMM[semana1-4])"
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type LogCleaner struct {
	baseDir    string
	maxAgeDays int
	logger     *logrus.Logger
}

func (p *LogCleaner) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {
	p.logger = logger

	// Get directory from config
	if dir, ok := config["log_dir"].(string); ok && dir != "" {
		p.baseDir = dir
	} else {
		p.baseDir = "logs" // Default directory
	}

	// Get max age from config
	if age, ok := config["max_age_days"].(float64); ok {
		p.maxAgeDays = int(age)
	} else {
		p.maxAgeDays = 30 // Default to 30 days
	}

	p.logger.Infof("Initializing Log Cleaner Plugin (max age: %d days, directory: %s)",
		p.maxAgeDays, p.baseDir)

	return nil
}

func (p *LogCleaner) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	var maxAgeDays int = p.maxAgeDays
	var targetDir string = p.baseDir

	// Log request if available
	if r != nil {
		p.logger.Infof("Log cleanup request from: %s", r.RemoteAddr)

		// Check for parameters in request
		if ageParam := r.URL.Query().Get("max_age_days"); ageParam != "" {
			if age, err := strconv.Atoi(ageParam); err == nil && age > 0 {
				maxAgeDays = age
			}
		}

		if dirParam := r.URL.Query().Get("dir"); dirParam != "" {
			targetDir = dirParam
		}
	}

	// Check shared context for parameters
	if shared != nil {
		if dir, ok := (*shared)["log_dir"].(string); ok && dir != "" {
			targetDir = dir
		}

		if age, ok := (*shared)["max_age_days"].(float64); ok && age > 0 {
			maxAgeDays = int(age)
		} else if ageStr, ok := (*shared)["max_age_days"].(string); ok && ageStr != "" {
			if age, err := strconv.Atoi(ageStr); err == nil && age > 0 {
				maxAgeDays = age
			}
		}
	}

	// Ensure directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		p.logger.Warnf("Directory %s does not exist, nothing to clean", targetDir)
		return map[string]interface{}{
			"status":        "warning",
			"message":       fmt.Sprintf("Directory %s does not exist", targetDir),
			"files_deleted": 0,
		}, nil
	}

	// Calculate cutoff time
	cutoffTime := time.Now().AddDate(0, 0, -maxAgeDays)
	p.logger.Infof("Cleaning log files older than %s in %s", cutoffTime.Format("2006-01-02"), targetDir)

	// Track deleted files
	deletedFiles := []string{}

	// Walk through directory
	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches log pattern (has .log extension)
		if filepath.Ext(path) == ".log" {
			// Check file age
			if info.ModTime().Before(cutoffTime) {
				p.logger.Infof("Deleting old log file: %s (modified: %s)",
					path, info.ModTime().Format("2006-01-02"))

				// Delete the file
				if err := os.Remove(path); err != nil {
					p.logger.Errorf("Failed to delete file %s: %v", path, err)
					return nil // Continue with other files
				}

				deletedFiles = append(deletedFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		p.logger.Errorf("Error walking directory %s: %v", targetDir, err)
		return nil, fmt.Errorf("error cleaning log files: %v", err)
	}

	// Return results
	return map[string]interface{}{
		"status":        "success",
		"files_deleted": len(deletedFiles),
		"files":         deletedFiles,
		"max_age_days":  maxAgeDays,
		"directory":     targetDir,
	}, nil
}

func (p *LogCleaner) FormatResult(result interface{}) (string, error) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		count, _ := resultMap["files_deleted"].(int)
		dir, _ := resultMap["directory"].(string)
		age, _ := resultMap["max_age_days"].(int)

		if count == 0 {
			return fmt.Sprintf("🧹 No log files needed cleaning in %s (max age: %d days)", dir, age), nil
		}

		return fmt.Sprintf("🧹 Cleaned %d log files older than %d days from %s", count, age, dir), nil
	}

	return "Log cleanup completed", nil
}

var PluginInstance pluginconf.Plugin = &LogCleaner{}

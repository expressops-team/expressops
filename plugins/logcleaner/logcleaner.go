// it would be cool to delete the logs weekly not only the 30 days old ones <==
package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	if dir, ok := config["log_dir"].(string); ok && dir != "" {
		p.baseDir = dir
	} else {
		p.baseDir = "logs"
	}

	if age, ok := config["max_age_days"].(float64); ok {
		p.maxAgeDays = int(age)
	} else {
		p.maxAgeDays = 30
	}

	p.logger.Infof("Initializing Log Cleaner Plugin (max age: %d days, directory: %s)",
		p.maxAgeDays, p.baseDir)

	return nil
}

// Execute cleans old log files and compresses weekly logs
func (p *LogCleaner) Execute(ctx context.Context, r *http.Request, shared *map[string]interface{}) (interface{}, error) {
	var maxAgeDays int = p.maxAgeDays
	var targetDir string = p.baseDir
	var zipFolderPath string = filepath.Join(p.baseDir, "zips")

	if r != nil {
		p.logger.Infof("Log cleanup request from: %s", r.RemoteAddr)

		if ageParam := r.URL.Query().Get("max_age_days"); ageParam != "" {
			if age, err := strconv.Atoi(ageParam); err == nil && age > 0 {
				maxAgeDays = age
			}
		}

		if dirParam := r.URL.Query().Get("dir"); dirParam != "" {
			targetDir = dirParam
		}
	}

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

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		p.logger.Warnf("Directory %s does not exist, nothing to clean", targetDir)
		return map[string]interface{}{
			"status":        "warning",
			"message":       fmt.Sprintf("Directory %s does not exist", targetDir),
			"files_deleted": 0,
		}, nil
	}

	zippedFiles, err := p.zipWeeklyLogs(targetDir, zipFolderPath)
	zipResult := ""

	if err != nil {
		p.logger.Warnf("Error zipping weekly logs: %v", err)
		zipResult = fmt.Sprintf("Error compressing logs: %v", err)
	} else if len(zippedFiles) > 0 {
		p.logger.Infof("Zipped %d log files for the week", len(zippedFiles))
		zipResult = fmt.Sprintf(" %d log files were compressed", len(zippedFiles))
	} else {
		zipResult = "There are no files to compress this week."
	}

	cutoffTime := time.Now().AddDate(0, 0, -maxAgeDays)
	p.logger.Infof("Cleaning log files older than %s in %s", cutoffTime.Format("2006-01-02"), targetDir)

	deletedFiles := []string{}

	err = filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) == ".log" {
			if info.ModTime().Before(cutoffTime) {
				p.logger.Infof("Deleting old log file: %s (modified: %s)",
					path, info.ModTime().Format("2006-01-02"))

				if err := os.Remove(path); err != nil {
					p.logger.Errorf("Failed to delete file %s: %v", path, err)
					return nil
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

	return map[string]interface{}{
		"status":        "success",
		"files_deleted": len(deletedFiles),
		"files":         deletedFiles,
		"max_age_days":  maxAgeDays,
		"directory":     targetDir,
		"zip_result":    zipResult,
		"zipped_count":  len(zippedFiles),
	}, nil
}

// zipWeeklyLogs compresses log files from the current week into a single archive
func (p *LogCleaner) zipWeeklyLogs(logsDir, zipDir string) ([]string, error) {
	if err := os.MkdirAll(zipDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create zip directory: %v", err)
	}

	now := time.Now()

	// Calculate the start of the current week (we are using monday)
	daysFromMonday := (int(now.Weekday()) - 1) % 7
	if daysFromMonday < 0 {
		daysFromMonday += 7
	}

	startOfWeek := now.AddDate(0, 0, -daysFromMonday)
	startDate := startOfWeek.Format("20060102")

	month := now.Format("January")
	_, weekNum := now.ISOWeek() // week number OF THE YEAR
	zipFileName := fmt.Sprintf("%s%dweek.zip", strings.ToLower(month), weekNum)
	zipFilePath := filepath.Join(zipDir, zipFileName)

	// Create the zip file :0
	zipFile, err := os.Create(zipFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	// Create a zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	var zippedFiles []string

	err = filepath.Walk(logsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and files not matching .log extension
		if info.IsDir() || filepath.Ext(path) != ".log" {
			return nil
		}

		// Get the date part from the log directory (format: YYYYMMDD)
		dirName := filepath.Base(filepath.Dir(path))

		// Only include log files from the current week
		if len(dirName) == 8 && dirName >= startDate {
			header, err := zip.FileInfoHeader(info)
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(logsDir, path)
			if err != nil {
				relPath = filepath.Base(path)
			}
			header.Name = relPath

			writer, err := zipWriter.CreateHeader(header)
			if err != nil {
				return err
			}

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(writer, file)
			if err != nil {
				return err
			}

			zippedFiles = append(zippedFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking logs directory: %v", err)
	}

	if len(zippedFiles) == 0 {
		zipFile.Close()
		os.Remove(zipFilePath)
		return nil, nil
	}

	p.logger.Infof("Created weekly zip archive: %s with %d log files", zipFilePath, len(zippedFiles))

	return zippedFiles, nil
}

// user-friendly output message describing the cleanup operation
func (p *LogCleaner) FormatResult(result interface{}) (string, error) {
	if resultMap, ok := result.(map[string]interface{}); ok {
		count, _ := resultMap["files_deleted"].(int)
		dir, _ := resultMap["directory"].(string)
		age, _ := resultMap["max_age_days"].(int)
		zipResult, _ := resultMap["zip_result"].(string)

		var output strings.Builder

		if zipResult != "" {
			output.WriteString(fmt.Sprintf("📦 %s\n", zipResult))
		}

		if count == 0 {
			output.WriteString(fmt.Sprintf("🧹 No log files needed cleaning in %s (max age: %d days)", dir, age))
		} else {
			output.WriteString(fmt.Sprintf("🧹 Cleaned %d log files older than %d days from %s", count, age, dir))
		}

		return output.String(), nil
	}

	return "Log cleanup completed", nil
}

var PluginInstance pluginconf.Plugin = &LogCleaner{}

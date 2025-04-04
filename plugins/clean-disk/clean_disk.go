// clean disk, cache and tmp files on a daily schedule
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type CleanDisk struct {
	logger  *logrus.Logger
	request *http.Request
	shared  *map[string]any
	ticker  *time.Ticker // daily
	done    chan bool
}

func (c *CleanDisk) Initialize(ctx context.Context, params map[string]interface{}, logger *logrus.Logger) error {
	c.logger = logger
	c.done = make(chan bool)
	c.ticker = time.NewTicker(24 * time.Hour)

	logger.Info("Initializing CleanDisk plugin")
	go func() {
		for {
			select {
			case <-c.ticker.C:
				if err := c.cleanDisk(logger, params); err != nil {
					logger.Errorf("Cleanup failed: %v", err)
				}
				logger.Info("Cleanup completed successfully 😎 \n")
			case <-c.done:
				c.ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (c *CleanDisk) Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error) {
	return nil, c.cleanDisk(c.logger, *shared)
}

// cleanDisk removes temporary and cache files from the system
func (c *CleanDisk) cleanDisk(logger *logrus.Logger, params map[string]interface{}) error {
	logger.Info("Starting disk cleanup")

	// Clean temp directory
	logger.Info("Cleaning /tmp directory")
	err := cleanDirectory("/tmp", logger)
	if err != nil {
		return fmt.Errorf("error cleaning /tmp: %v", err)
	}

	// Clean cache directories
	logger.Info("Cleaning cache directories")
	cacheDirs := []string{
		"/var/cache/apt",
		"/var/tmp",
	}
	for _, dir := range cacheDirs {
		err := cleanDirectory(dir, logger)
		if err != nil {
			logger.Warnf("Error cleaning %s: %v", dir, err)
		}
	}

	// Sync filesystem
	logger.Info("Cleaning filesystem")
	cmd := exec.Command("sync")
	if err := cmd.Run(); err != nil {
		logger.Warnf("Error running sync: %v", err)
		return err
	}
	logger.Info("Filesystem cleaned successfully")

	return nil
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

func (c *CleanDisk) Name() string {
	return "clean-disk"
}

// FormatResult formats the result of the cleanup operation
func (c *CleanDisk) FormatResult(result interface{}) (string, error) {
	return "", nil
}

var PluginInstance pluginconf.Plugin = &CleanDisk{}

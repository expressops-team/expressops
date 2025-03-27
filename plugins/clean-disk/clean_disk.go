// clean disk, cache and tmp files on a daily schedule
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type CleanDisk struct {
	logger *logrus.Logger
	ticker *time.Ticker // daily
	done   chan bool
}

func (c *CleanDisk) cleanDisk(logger *logrus.Logger, params map[string]interface{}) error {
	logger.Info("Iniciando limpieza de disco")

	// Clean /tmp directory
	logger.Info("Limpiando directorio /tmp")
	err := cleanDirectory("/tmp", logger)
	if err != nil {
		return fmt.Errorf("error cleaning /tmp: %v", err)
	}

	// Clean cache directories
	logger.Info("Limpiando directorios de caché")
	cacheDirs := []string{
		"/var/cache",
		filepath.Join(os.Getenv("HOME"), ".cache"),
	}
	for _, dir := range cacheDirs {
		err := cleanDirectory(dir, logger)
		if err != nil {
			logger.Warnf("Error cleaning %s: %v", dir, err)
		}
	}

	// Run system disk cleanup
	logger.Info("Limpiando sistema de archivos")
	cmd := exec.Command("sync") // after this you can reboot
	if err := cmd.Run(); err != nil {
		logger.Warnf("Error running sync: %v", err)
	}

	logger.Info("Limpiado sistema de archivos exitosamente")
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
			logger.Warnf("No se pudo eliminar %s: %v", path, err)
		} else {
			logger.Debugf("Eliminado %s", path)
		}
	}
	return nil
}

func (c *CleanDisk) Initialize(ctx context.Context, params map[string]interface{}, logger *logrus.Logger) error {
	c.logger = logger
	c.done = make(chan bool)
	c.ticker = time.NewTicker(24 * time.Hour)

	logger.Info("Inicializando CleanDisk plugin")
	go func() {
		for {
			select {
			case <-c.ticker.C:
				if err := c.cleanDisk(logger, params); err != nil {
					logger.Errorf("Cleanup falló: %v", err)
				}
				logger.Info("Cleanup completado exitosamente 😎 \n")
			case <-c.done:
				c.ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (c *CleanDisk) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	return nil, c.cleanDisk(c.logger, params)
}

func (c *CleanDisk) Name() string {
	return "clean-disk"
}

var PluginInstance pluginconf.Plugin = &CleanDisk{}

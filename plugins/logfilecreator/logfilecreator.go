// TO DO
// add parameter to specify the time of the day to create the log file
// add parameter to specify what plugins was executed and what was the output

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type LogFileCreator struct {
	logger   *logrus.Logger
	filename string
}

func (l *LogFileCreator) Initialize(ctx context.Context, params map[string]interface{}, logger *logrus.Logger) error {
	l.logger = logger
	filename := "logs/logfilecreator.log"

	l.filename = filename

	return nil
}

func (l *LogFileCreator) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	logger := l.logger
	logger.Infof("Creando fichero de log: %s", l.filename)

	dir := filepath.Dir(l.filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("no se pudo crear el directorio: %w", err)
	}

	file, err := os.Create(l.filename)
	if err != nil {
		return nil, fmt.Errorf("no se pudo crear el fichero: %w", err)
	}
	defer file.Close()

	header := fmt.Sprintf("Fichero de log creado el %s\n", time.Now().Format(time.RFC3339))
	if _, err := file.WriteString(header); err != nil {
		return nil, fmt.Errorf("no se pudo escribir en el fichero de log: %w", err)
	}

	logger.Info("Fichero de log creado correctamente")
	return nil, nil
}

func NewLogFileCreator(logger *logrus.Logger) pluginconf.Plugin {
	return &LogFileCreator{logger: logger}
}

var PluginInstance = NewLogFileCreator(logrus.New())

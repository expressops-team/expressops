// still working on it ;(
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
	logger       *logrus.Logger
	baseFilename string
	logPath      string
	logFile      *os.File
}

func (l *LogFileCreator) Initialize(ctx context.Context, params map[string]interface{}, logger *logrus.Logger) error {
	l.logger = logger

	if path, ok := params["log_path"].(string); ok && path != "" {
		l.logPath = path
	} else {
		l.logPath = "logs"
	}

	if base, ok := params["base_filename"].(string); ok && base != "" {
		l.baseFilename = base
	} else {
		l.baseFilename = "logfile"
	}

	return nil
}

func (l *LogFileCreator) generateFilename() string {
	currentDate := time.Now().Format("02012006")
	return filepath.Join(l.logPath, fmt.Sprintf("%s%s.log", l.baseFilename, currentDate))
}

func (l *LogFileCreator) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	filename := l.generateFilename()
	l.logger.Infof("Preparando para crear/abrir fichero de log: %s", filename)

	// ===> NOT NECESSARY, BECAUSE WE ARE USING THE DAILY FOLDER STRUCTURE <====
	// dir := filepath.Dir(filename)
	// if err := os.MkdirAll(dir, 0755); err != nil {
	// 	return nil, fmt.Errorf("no se pudo crear el directorio: %w", err)
	// }

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("no se pudo abrir el fichero: %w", err)
	}

	if l.logFile != nil {
		l.logFile.Close()
	}

	l.logFile = file

	fileLogger := logrus.New()
	fileLogger.SetOutput(file)
	fileLogger.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})

	timeNow := time.Now().Format("2006-01-02 15:04:05")
	fileLogger.Infof("===== Entrada de registro en %s =====", timeNow)

	if pluginName, ok := params["plugin_name"].(string); ok {
		fileLogger.Infof("Plugin ejecutado: %s", pluginName)
	} else {
		fileLogger.Info("Plugin ejecutado: desconocido")
	}

	if pluginOutput, ok := params["plugin_output"]; ok {
		fileLogger.Infof("Salida del plugin: %v", pluginOutput)
	}

	fileLogger.Info("Salida del plugin:")
	for k, v := range params {
		if k != "plugin_name" && k != "plugin_output" {
			fileLogger.Infof("  %s: %v", k, v)
		}
	}

	l.logger.Info("Entrada de registro creada exitosamente")
	return map[string]string{"status": "éxito", "filename": filename}, nil
}

func (l *LogFileCreator) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

func (l *LogFileCreator) FormatResult(result interface{}) (string, error) {
	if res, ok := result.(map[string]string); ok {
		return fmt.Sprintf("Registro creado exitosamente en %s", res["filename"]), nil
	}
	return fmt.Sprintf("Operación de registro completada: %v", result), nil
}

func NewLogFileCreator(logger *logrus.Logger) pluginconf.Plugin {
	return &LogFileCreator{logger: logger}
}

var PluginInstance pluginconf.Plugin = NewLogFileCreator(nil)

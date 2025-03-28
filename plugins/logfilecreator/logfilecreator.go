package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func (l *LogFileCreator) extractHealthChecks(output string) map[string]string {
	checksMap := make(map[string]string)

	if !strings.Contains(output, "HealthCheckPlugin") {
		return checksMap
	}

	parts := strings.Split(output, ":")
	if len(parts) < 2 {
		return checksMap
	}

	checksString := strings.TrimSpace(parts[1])
	checksList := strings.Split(checksString, ",")

	for _, check := range checksList {
		check = strings.TrimSpace(check)
		if check == "" {
			continue
		}

		if strings.Contains(check, "HealthCheckPlugin.") {
			checkName := strings.TrimPrefix(check, "HealthCheckPlugin.")
			checksMap[checkName] = "ejecutado"
		}
	}

	return checksMap
}

func (l *LogFileCreator) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	filename := l.generateFilename()
	l.logger.Infof("Preparando para crear/abrir fichero de log: %s", filename)

	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("no se pudo crear el directorio: %w", err)
	}

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

	pluginName := "desconocido"
	if name, ok := params["plugin_name"].(string); ok {
		if name != "health-check-plugin" && name != "sleep-plugin" && name != "clean-disk" {
			return nil, fmt.Errorf("plugin no soportado: %s", name)
		}
		pluginName = name
		fileLogger.Infof("Plugin ejecutado: %s", pluginName)
	}

	if output, ok := params["plugin_output"].(string); ok {
		fileLogger.Infof("Salida del plugin: %s", output)

		if pluginName == "health-check-plugin" {
			checks := l.extractHealthChecks(output)
			if len(checks) > 0 {
				fileLogger.Info("Verificaciones detectadas:")
				for check, status := range checks {
					fileLogger.Infof("  - %s: %s", check, status)
				}
			}
		} else if pluginName == "clean-disk" {
			if paths, ok := params["cleanupPaths"].([]interface{}); ok {
				fileLogger.Info("Rutas limpiadas:")
				for _, path := range paths {
					fileLogger.Infof("  - %v", path)
				}
			}
		}
	}

	l.logger.Info("Entrada de registro creada exitosamente")
	return map[string]interface{}{
		"status":   "éxito",
		"filename": filename,
	}, nil
}

func (l *LogFileCreator) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

func (l *LogFileCreator) FormatResult(result interface{}) (string, error) {
	if res, ok := result.(map[string]interface{}); ok {
		return fmt.Sprintf("Registro creado exitosamente en %s", res["filename"]), nil
	}
	return fmt.Sprintf("Operación de registro completada: %v", result), nil
}

func NewLogFileCreator(logger *logrus.Logger) pluginconf.Plugin {
	return &LogFileCreator{logger: logger}
}

var PluginInstance pluginconf.Plugin = NewLogFileCreator(nil)

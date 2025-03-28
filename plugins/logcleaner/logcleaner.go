// Los sábados comprime los logs del día anterior y los guarda en una carpeta llamada "backups(YYYYMM[semana1-4])"
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	pluginconf "expressops/internal/plugin/loader"

	"github.com/sirupsen/logrus"
)

type LogCleaner struct {
	logger      *logrus.Logger
	rutaLogs    string
	weeksToKeep int
}

func (l *LogCleaner) Initialize(ctx context.Context, params map[string]interface{}, logger *logrus.Logger) error {
	l.logger = logger
	if ruta, ok := params["log_path"].(string); ok && ruta != "" {
		l.rutaLogs = ruta
	} else {
		l.rutaLogs = "logs"
	}

	if weeks, ok := params["weeks_to_keep"].(int); ok && weeks > 0 {
		l.weeksToKeep = weeks
	} else {
		l.weeksToKeep = 4
	}

	return nil
}

func (l *LogCleaner) obtenerSemanaMes(fecha time.Time) int {
	dia := fecha.Day()
	if dia <= 7 {
		return 1
	} else if dia <= 14 {
		return 2
	} else if dia <= 21 {
		return 3
	} else {
		return 4
	}
}

func (l *LogCleaner) obtenerArchivosDiaAnterior() []string {
	ayer := time.Now().AddDate(0, 0, -1)
	fechaAyer := ayer.Format("02012006") // DDMMYYYY

	var archivos []string
	filepath.Walk(l.rutaLogs, func(ruta string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.Contains(ruta, fechaAyer) && strings.HasSuffix(ruta, ".log") {
			archivos = append(archivos, ruta)
		}
		return nil
	})

	return archivos
}

func (l *LogCleaner) crearCarpetaRespaldo(fecha time.Time) string {
	año := fecha.Format("2006")
	mes := fecha.Format("01")
	semana := l.obtenerSemanaMes(fecha)

	dirRespaldo := filepath.Join("backups", fmt.Sprintf("%s%s-semana%d", año, mes, semana))
	os.MkdirAll(dirRespaldo, 0755)

	return dirRespaldo
}

func (l *LogCleaner) crearArchivoTar(archivos []string, dirRespaldo string) string {
	if len(archivos) == 0 {
		return ""
	}

	horaActual := time.Now().Format("02012006-150405")
	nombreTar := filepath.Join(dirRespaldo, fmt.Sprintf("logs-%s.tar.gz", horaActual))

	archivoTar, _ := os.Create(nombreTar)
	defer archivoTar.Close()

	escritorGzip := gzip.NewWriter(archivoTar)
	defer escritorGzip.Close()

	escritorTar := tar.NewWriter(escritorGzip)
	defer escritorTar.Close()

	for _, archivo := range archivos {
		tarFile(escritorTar, archivo)
	}

	return nombreTar
}

func tarFile(escritorTar *tar.Writer, rutaArchivo string) {
	archivo, _ := os.Open(rutaArchivo)
	defer archivo.Close()

	info, _ := archivo.Stat()
	cabecera, _ := tar.FileInfoHeader(info, "")

	cabecera.Name = filepath.Base(rutaArchivo)

	escritorTar.WriteHeader(cabecera)
	io.Copy(escritorTar, archivo)
}

func (l *LogCleaner) borrarRespaldosAntiguos() {
	fechaLimite := time.Now().AddDate(0, 0, -l.weeksToKeep*7)

	filepath.Walk("backups", func(ruta string, info os.FileInfo, err error) error {
		if !info.IsDir() && strings.HasSuffix(ruta, ".tar.gz") {
			if info.ModTime().Before(fechaLimite) {
				os.Remove(ruta)
				l.logger.Infof("Respaldo antiguo borrado: %s", ruta)
			}
		}
		return nil
	})
}

func (l *LogCleaner) Execute(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	if time.Now().Weekday() != time.Saturday {
		return map[string]string{"estado": "omitido", "razón": "no es sábado"}, nil
	}

	archivosAyer := l.obtenerArchivosDiaAnterior()

	if len(archivosAyer) == 0 {
		l.logger.Info("No se encontraron archivos de registro de ayer para archivar")
		return map[string]string{"estado": "omitido", "razón": "sin logs"}, nil
	}

	dirRespaldo := l.crearCarpetaRespaldo(time.Now())

	archivoTar := l.crearArchivoTar(archivosAyer, dirRespaldo)

	l.borrarRespaldosAntiguos()

	resultado := map[string]interface{}{
		"estado":           "éxito",
		"archivo_respaldo": archivoTar,
		"total_archivos":   len(archivosAyer),
		"archivos":         archivosAyer,
	}

	l.logger.Infof("Se archivaron correctamente %d archivos de registro en %s", len(archivosAyer), archivoTar)
	return resultado, nil
}

func (l *LogCleaner) FormatResult(result interface{}) (string, error) {
	if res, ok := result.(map[string]interface{}); ok {
		if res["estado"] == "omitido" {
			return fmt.Sprintf("🔄 Respaldo omitido: %s", res["razón"]), nil
		}

		if res["estado"] == "éxito" {
			return fmt.Sprintf("📦 Se archivaron correctamente %d logs en %s",
				res["total_archivos"], res["archivo_respaldo"]), nil
		}
	}

	return fmt.Sprintf("Operación Limpiador de Logs: %v", result), nil
}

func NewLogCleaner(logger *logrus.Logger) pluginconf.Plugin {
	return &LogCleaner{logger: logger}
}

var PluginInstance pluginconf.Plugin = NewLogCleaner(nil)

// plugins/slack/slack.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	pluginconf "expressops/internal/plugin/loader"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

type SlackPlugin struct {
	webhook string
	logger  *logrus.Logger
}

// Initialize initializes the plugin with the provided configuration
func (s *SlackPlugin) Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error {

	s.logger = logger
	webhook, ok := config["webhook_url"].(string)
	if !ok {
		return fmt.Errorf("slack webhook URL required")
	}
	s.webhook = webhook
	s.logger.Info("Inicializando Slack Plugin")
	return nil
}

// Execute sends a message to a Slack channel
func (s *SlackPlugin) Execute(ctx context.Context, _ *http.Request, shared *map[string]any) (interface{}, error) {
	// 1. Intentar obtener 'message' desde shared
	var message string
	if msgVal, ok := (*shared)["message"]; ok {
		if msgStr, isString := msgVal.(string); isString {
			message = msgStr
			s.logger.Debugf("Mensaje obtenido de shared[\"message\"]: %s", message)
		} else {
			s.logger.Warnf("shared[\"message\"] existe pero no es un string (tipo: %T)", msgVal)
		}
	}

	// 2. Si no hay 'message', intentar usar el resultado anterior ('previous_result')
	if message == "" {
		s.logger.Debug("shared[\"message\"] no encontrado o vacío, buscando en shared[\"previous_result\"]")
		if prevResultVal, ok := (*shared)["previous_result"]; ok {
			// Intentamos convertir el resultado anterior a string.
			// Puede ser cualquier tipo, así que usamos fmt.Sprintf como último recurso.
			if prevResultStr, isString := prevResultVal.(string); isString {
				message = prevResultStr
				s.logger.Debugf("Mensaje obtenido de shared[\"previous_result\"] (era string): %s", message)
			} else {
				// Si no es string, intentamos una representación genérica
				message = fmt.Sprintf("%v", prevResultVal)
				s.logger.Debugf("Mensaje obtenido de shared[\"previous_result\"] (convertido a string): %s", message)
			}
		}
	}

	// 3. Comprobar si finalmente tenemos un mensaje
	if message == "" {
		err := fmt.Errorf("no se pudo determinar un mensaje para enviar a Slack (ni 'message' ni 'previous_result' encontrados/utilizables en shared)")
		s.logger.Error(err.Error())
		return nil, err
	}

	// 4. Obtener 'channel' y 'severity' desde shared, con valores por defecto o manejo de ausencia
	var channel string
	if channelVal, ok := (*shared)["channel"]; ok {
		if channelStr, isString := channelVal.(string); isString {
			channel = channelStr
		} else {
			s.logger.Warnf("shared[\"channel\"] existe pero no es un string (tipo: %T), se usará el canal por defecto del webhook", channelVal)
		}
	} else {
		s.logger.Debug("shared[\"channel\"] no encontrado, se usará el canal por defecto del webhook")
	}

	var severity string = "normal" // Valor por defecto
	if severityVal, ok := (*shared)["severity"]; ok {
		if severityStr, isString := severityVal.(string); isString {
			severity = severityStr
		} else {
			s.logger.Warnf("shared[\"severity\"] existe pero no es un string (tipo: %T), se usará '%s'", severityVal, severity)
		}
	} else {
		s.logger.Debugf("shared[\"severity\"] no encontrado, se usará '%s'", severity)
	}

	s.logger.Infof("Enviando a Slack: Channel='%s', Severity='%s', Message='%.50s...'", channel, severity, message) // Loguear antes de enviar

	// 5. Preparar y enviar payload (esta parte no cambia mucho)
	payload := map[string]interface{}{
		"text": fmt.Sprintf("[%s] %s", severity, message),
	}
	// Solo incluir 'channel' si se especificó uno, sino Slack usa el default del webhook
	if channel != "" {
		payload["channel"] = channel
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		s.logger.Errorf("Error codificando payload JSON para Slack: %v", err)
		return nil, fmt.Errorf("error interno al codificar payload: %w", err) // Envolver error
	}

	// Usar el contexto en la petición HTTP
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.webhook, bytes.NewBuffer(jsonData))
	if err != nil {
		s.logger.Errorf("Error creando request para Slack: %v", err)
		return nil, fmt.Errorf("error interno al crear request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{} // Reutilizar cliente podría ser mejor en producción, pero está bien por ahora
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Errorf("Error enviando mensaje a Slack: %v", err)
		// Comprobar si el error es por timeout del contexto
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("timeout enviando a Slack: %w", err)
		}
		return nil, fmt.Errorf("error de red enviando a Slack: %w", err)
	}
	defer resp.Body.Close() // Asegurar que el cuerpo se cierre

	if resp.StatusCode >= 300 { // Comprobar cualquier status code no exitoso (no solo != 200)
		s.logger.Errorf("Error en la respuesta de Slack API: %s", resp.Status)
		// Leer cuerpo de la respuesta para más detalles (opcional pero útil)
		// bodyBytes, _ := io.ReadAll(resp.Body)
		// s.logger.Errorf("Cuerpo de la respuesta de error de Slack: %s", string(bodyBytes))
		return nil, fmt.Errorf("error de Slack API: %s", resp.Status)
	}

	s.logger.Info("Mensaje enviado correctamente a Slack.")
	return map[string]interface{}{"status": "success", "message_sent": message}, nil // Devolver un resultado más informativo
}

// FormatResult sigue igual
func (s *SlackPlugin) FormatResult(result interface{}) (string, error) {
	// Podrías hacer este formato más útil si el resultado es el mapa que devolvemos ahora
	if resultMap, ok := result.(map[string]interface{}); ok {
		return fmt.Sprintf("Resultado Slack: Status=%v", resultMap["status"]), nil
	}
	return fmt.Sprintf("Resultado Slack: %v", result), nil
}

// PluginInstance sigue igual
var PluginInstance pluginconf.Plugin = &SlackPlugin{}

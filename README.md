
3º Construir un plugin que se compile como .so 

Ejemplos de plugins que podría haber en nuestro proyecto:

1. Definir valores generales (contrato) de la estructura de la interfaz que todos los plugins deben cumplir(plugin de notificaciones y plugin de orquestacion de flujos)
2. Crear la estructura del plugin que implemente la interfaz definida.
3. Implementar las funciones del plugin
4. Que el plugin se compile como en un archivo compartido .so para que pueda ser cargado dinámicamente
5. Hacer plugins que sea funcional



// Propuesta variable global, accesible en nuestro codigo que se guarden cosas que necesitamos en todos lados. Application container


- Ejemplos de plugins:
1. Plugin de Logger o Registro de eventos

2. Plugin de Contexto para Peticiones

3. Plugin de credenciales o Autenticación

4. Plugin de notificaciones
Función → Envío de alertas o notificaciones a un canal de Slack, Mail cuando se detecte un incidente.

Uso en la aplicación → Permite notificar a los SRE o a otros equipos de manera automática cuando se activa un flujo de incidentes.

5. Plugin de Orquestación de flujos
Función → Define y controla el flujo de acciones que se ejecutarán en respuesta a un incidente (Pasos prioritario = cómo actuar)

Uso en la aplicación → Facilita la creación de flujos configurables sin necesidad de reescribir código en Go → ya que el SRE puede modificar el YAML de configuración.


Preguntas:
preguntar si hacer una encuesta en el grupo SRE-team para saber qué tipo de plugin les vendria mejor?

Proponer plugin 4 y plugin 5 --> enviar alerta por incidente concreto(4) --> y definir el flujo de acciones o pasos principales que hay que seguir 

Preguntar que es webhook: mecanismo que permite a una aplicación enviar datos en tiempo real a otra cuando ocurre un evento específico. Los webhooks se utilizan comúnmente para integrar diferentes servicios y automatizar flujos de trabajo.

Pregunta que es: Implementar la lógica para cargar dinámicamente los plugins en tiempo de ejecución 


Puntero al contexto, al logger, a su propia configuración, a las credenciales que necesite (map string string), quiza un puntero a al request (poruqe se saca los parametros)

Refinar

https://webhook.site/ para probar





                                                            Que he cambiado
Comentarios en ingles
myplugin --> pluginconf-- >plugin/loader X
variable myplugin --> pluginManager
Logrus


                                                            A arreglar/hacer
Health check plugin --> controla uso de ram cpu...

=============================
DEMASIADOS COMENTARIOS

=============================
Mover los .so a una carpeta de compilados: 
/build/plugins/
=============================
api/
  v1alpha1/
cmd/
  expressops.go
docs/
  samples/
    config.yaml
    config_example.yaml
internal/
  config/
  pluginconf/ (o plugins/loader)
  server/
plugins/
  slack/
    slack.go
  healthcheck/
    health_check.go
build/
  plugins/
    slack.so
    health_check.so
go.mod
README.md

## ExpressOps 🚀 <img src="docs/img/RistrettOps.png" alt="ExpressOps Logo" align="right" width="300" style="margin-top: 20px;">

ExpressOps is a lightweight flow orchestrator powered by dynamically loaded plugins. It allows you to define operational workflows (such as health checks, formatting, notifications, and logging) via a simple YAML configuration. Each plugin handles one task and flows chain them together.

## 📦 Docker Hub

The ExpressOps Docker image is available on Docker Hub at:
https://hub.docker.com/r/expressopsfreepik/expressops

## 📜 Table of Contents

- [Features](#-features)
- [Architecture](#-architecture-overview)
- [Requirements](#-requirements)
- [Installation](#-installation)
- [Usage](#-usage)
- [Configuration](#-configuration)
- [Plugins](#-plugins)
- [Example Flows](#-example-flows)
- [Contributing](#-contributing)
- [License](#-license)
- [Acknowledgements](#-acknowledgements)

## 🧭 Architecture Overview

![Functional perspective](docs/img/architecture.png)

ExpressOps follows a modular architecture with three main components:
1. **Core Engine**: Handles configuration loading, plugin management, and server initialization
2. **Plugin System**: Dynamically loads and executes plugins through a standardized interface
3. **Flow Orchestration**: Manages the execution of plugin pipelines based on YAML configuration

## ✨ Features

- 🔌 Dynamic plugin loading from `.so` files at runtime
- 🛠️ **Extensive plugin ecosystem**:
  - **System Operations**: Health checks, disk cleanup, and system monitoring
  - **Kubernetes**: K8s cluster health and management
  - **User Management**: User creation and permission management
  - **Notifications**: Slack integration for alerts and reports
  - **Utilities**: Data formatting, timeouts, and testing tools
- ⚙️ YAML-based flow configuration for defining execution pipelines
- 🌐 HTTP server with endpoints to trigger flows dynamically
- 📊 Metrics collection for monitoring system performance
- 📜 Detailed logging for debugging and traceability

## 📦 Requirements

> 🐧 ExpressOps runs on Linux (due to the Go plugin system)
- Golang 1.20+
- Linux environment (for plugin support)

## 🔧 Installation

```bash
git clone https://github.com/freepik-company/expressops
cd expressops
make build
```

This will build the main application and all available plugins. If you want to build specific plugins manually:

```bash
go build -buildmode=plugin -o plugins/slack/slack.so plugins/slack/slack.go
go build -buildmode=plugin -o plugins/healthcheck/health_check.so plugins/healthcheck/health_check.go
go build -buildmode=plugin -o plugins/formatters/health_alert_formatter.so plugins/formatters/health_alert_formatter.go
```

## 🚀 Usage

Start the server with your configuration file:

```bash
./expressops -config path/to/your/config.yaml
```

Or use the default configuration:

```bash
./expressops
```

Trigger a flow:

```bash
curl "http://localhost:8080/flow?flowName=alert-flow"
```

## 🛠️ Makefile Commands

ExpressOps includes a comprehensive Makefile with various commands to simplify development, building, and deployment:

### Basic Commands
- `make build` - Build plugins and application locally
- `make run` - Run application locally
- `make help` - Display all available commands and current configuration

### Docker Workflow
- `make docker-build` - Build Docker image (auto-versioned) and update Helm values
- `make docker-run` - Run container with the last built tag
- `make docker-run-build` - Build and run Docker container
- `make docker-clean` - Clean Docker resources (stops/removes container and old tags)
- `make docker-push` - Tag and push the last built image to Docker Hub

### Helm Deployment
- `make helm-deploy` - Deploy/upgrade Helm chart using tag from values.yaml

### Combined Workflow
- `make release` - Complete release cycle: build, push, and deploy

You can configure various parameters like ports, log levels, and paths by setting environment variables or editing the Makefile.

## ⚙️ Configuration

ExpressOps uses a YAML configuration file that defines logging settings, server configuration, plugins, and flows:

```yaml
logging:
  level: info
  format: text

server:
  port: 8080
  address: 0.0.0.0
  timeoutSeconds: 4

plugins:
  - name: slack-notifier
    path: plugins/slack/slack.so
    type: notification
    config:
      webhook_url: $SLACK_WEBHOOK_URL

flows:
  - name: alert-flow
    description: "Health check with notification"
    pipeline:
      - pluginRef: health-check-plugin
      - pluginRef: formatter-plugin
      - pluginRef: slack-notifier
```

## 🔌 Plugins

ExpressOps comes with a variety of plugins:

| Plugin | Type | Description |
|--------|------|-------------|
| health-check-plugin | health | Collects CPU, memory, and disk usage stats |
| kube-health-plugin | k8s | Monitors Kubernetes cluster health |
| formatter-plugin | utils | Transforms health data into a clean report |
| slack-notifier | notification | Sends messages to a Slack channel |
| sleep-plugin | test | Delays flow execution to test timeouts |
| test-print-plugin | test | Debug plugin that prints test data |
| permissions-plugin | management | Manages file permissions |
| user-creation-plugin | management | Creates system users |
| clean-disk-plugin | maintenance | Handles disk cleanup operations |

## 📋 Example Flows

### Health Check with Notification (alert-flow)

This flow performs a system health check, formats the results, and sends an alert to Slack:

```bash
curl "http://localhost:8080/flow?flowName=alert-flow"
```

## 🤝 Contributing

Contributions are welcome! Feel free to open an issue, fork the repo, or submit a pull request.

Please follow the convention of exporting your plugin as PluginInstance, and ensure it implements the Plugin interface:

```go
type Plugin interface {
    Initialize(ctx context.Context, config map[string]interface{}, logger *logrus.Logger) error
    Execute(ctx context.Context, request *http.Request, shared *map[string]any) (interface{}, error)
    FormatResult(result interface{}) (string, error)
}
```

## 🪪 License

Copyright 2025.

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for more information.

## 🙏 Acknowledgements

Thanks to all contributors and plugin authors who made this modular system possible.

Happy hacking ✨
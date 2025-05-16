## ExpressOps 🚀 <img src="docs/img/RistrettOps.png" alt="ExpressOps Logo" align="right" width="190" style="margin-top: 20px;">

ExpressOps is a lightweight flow orchestrator powered by dynamically loaded plugins. It allows you to define operational workflows (such as health checks, formatting, notifications, and logging) via a simple YAML configuration. Each plugin handles one task and flows chain them together.

## 📦 Docker Hub

The ExpressOps Docker image is available on Docker Hub at:
https://hub.docker.com/r/expressopsfreepik/expressops


## 📜 Table of Contents

- [Features](#-features)
- [Requirements](#-requirements)
- [Installation](#-installation)
- [Usage](#-usage)
- [Configuration](#-configuration)
- [Example Flow](#-example-flow-dr-house)
- [Contributing](#-contributing)
- [License](#-license)
- [Acknowledgements](#-acknowledgements)


## 🧭 Architecture Overview

![Functional perspective ](docs/img/architecture.png)


## ✨ Features

- 🔌 Dynamic plugin loading from `.so` files at runtime.
- 🛠️ **Plugin system**:
  - **health-check-plugin**: collects CPU, memory, and disk usage stats.
  - **formatter-plugin**: transforms health data into a clean report.
  - **slack-notifier**: sends messages to a Slack channel.
  - **sleep-plugin**: delays flow execution to test timeouts.
  - **test-print-plugin**: debug plugin that prints test data.
- ⚙️ YAML-based flow configuration (define execution pipelines).
- 🌐 HTTP server with endpoints to trigger flows dynamically.
- 📜 Detailed logging for debugging and traceability.


## 📦 Requirements

> 🐧 ExpressOps runs on Linux (due to the Go plugin system).
- Golang 1.20+



## 🔧 Installation

```bash
git clone https://github.com/freepik-company/expressops
cd expressops
make build
```


To build the plugins manually:

```bash
go build -buildmode=plugin -o plugins/slack/slack.so plugins/slack/slack.go
go build -buildmode=plugin -o plugins/healthcheck/health_check.so plugins/healthcheck/health_check.go
go build -buildmode=plugin -o plugins/formatters/health_alert_formatter.so plugins/formatters/health_alert_formatter.go
```

## 🚀 Usage

Start the server:
```bash
./expressops -config docs/samples/config.yaml
```
Trigger a flow:
```bash
curl "http://localhost:8080/flow?flowName=dr-house&format=verbose"
```

```yaml
plugins:
  - name: slack-notifier
    path: plugins/slack/slack.so
    type: notification
    config:
      webhook_url: $SLACK_WEBHOOK_URL

flows:
  - name: alert-flow
    pipeline:
      - pluginRef: health-check-plugin
      - pluginRef: formatter-plugin
      - pluginRef: slack-notifier
```


## 🧪 Example Flow: dr-house

This flow performs:

1. System health check

2. Formats the result

3. Prints a test message

```bash
curl "http://localhost:8080/flow?flowName=dr-house&format=verbose"
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
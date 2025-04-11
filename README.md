## ExpressOps 🚀

ExpressOps is a lightweight flow orchestrator powered by dynamically loaded plugins. It allows you to define operational workflows (such as health checks, formatting, notifications, and logging) via a simple YAML configuration. Each plugin handles one task and flows chain them together.

## 📦 Docker Hub

The ExpressOps Docker image is available on Docker Hub at:
https://hub.docker.com/r/davidnull/expressops

You can pull it with:
```bash
docker pull davidnull/expressops:latest
```

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
go build -o expressops main.go
```


To build the plugins manually:

```bash
go build -buildmode=plugin -o plugins/slack/slack.so plugins/slack/slack.go
go build -buildmode=plugin -o plugins/healthcheck/health_check.so plugins/healthcheck/health_check.go
go build -buildmode=plugin -o plugins/formatters/health_alert_formatter.so plugins/formatters/health_alert_formatter.go
```

Or use the helper script:
```bash
./.plugin_builder/script.sh
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

## 🛥️ Kubernetes Deployment

ExpressOps can be deployed to Kubernetes using the provided Makefile commands:

```bash
# Connect to Kubernetes (keep this terminal open)
gcloud compute ssh --zone "europe-west1-d" "it-school-2025-1" --tunnel-through-iap --project "fc-it-school-2025" --ssh-flag "-N -L 6443:127.0.0.1:6443"

# Build, tag and push Docker image to Docker Hub (optional)
# The deployment is already configured to use the public image davidnull/expressops:latest
make docker-push

# Set up your secrets (needed for Slack notifications)
# Option 1: Set SLACK_WEBHOOK_URL in your environment:
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/REAL/TOKEN"

# Option 2: Edit secrets.yaml manually
make k8s-generate-secrets
# Then edit k8s/secrets.yaml with your actual webhook URL

# Deploy to Kubernetes
make k8s-deploy

# Check deployment status
make k8s-status

# Forward port to access the application
make k8s-port-forward

# View logs
make k8s-logs

# Delete deployment
make k8s-delete
```

The application will be accessible at http://localhost:8080 after port forwarding.

## ⚙️ Configuration
```yaml
plugins:
  - name: slack-notifier
    path: plugins/slack/slack.so
    type: notification
    config: {}

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
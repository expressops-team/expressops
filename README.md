## ExpressOps 🚀 <img src="docs/img/LOGO_EXPRESSOPS.png" alt="ExpressOps Logo" align="right" width="100">

ExpressOps is a lightweight flow orchestrator powered by dynamically loaded plugins. It allows you to define operational workflows (such as health checks, formatting, notifications, and logging) via a simple YAML configuration. Each plugin handles one task and flows chain them together.

## 📦 Docker Hub

The ExpressOps Docker image is available on Docker Hub at:
https://hub.docker.com/r/davidnull/expressops

You can pull it with:
```bash
docker pull davidnull/expressops:1.0.0
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

### Environment Variables

ExpressOps supports the following environment variables for configuration:

- `SERVER_PORT`: Override the HTTP server port (default: 8080)
- `SERVER_ADDRESS`: Override the HTTP server bind address (default: 0.0.0.0)
- `TIMEOUT_SECONDS`: Override the flow execution timeout in seconds (default: 4)
- `LOG_LEVEL`: Set logging level (info, debug, warn, error)
- `LOG_FORMAT`: Set logging format (text, json)
- `SLACK_WEBHOOK_URL`: Required for Slack notifications

## 🛥️ Kubernetes Deployment

ExpressOps can be deployed to Kubernetes using the provided Makefile commands:

```bash
# Connect to Kubernetes (keep this terminal open)
gcloud compute ssh --zone "europe-west1-d" "it-school-2025-1" --tunnel-through-iap --project "fc-it-school-2025" --ssh-flag "-N -L 6443:127.0.0.1:6443"

# Install External Secrets Operator (first time setup)
make k8s-install-eso

# Build, tag and push Docker image to Docker Hub (optional)
# The deployment is already configured to use davidnull/expressops:1.0.0
# You can change the version by setting the VERSION variable:
# VERSION=1.0.1 make docker-push
make docker-push

# OPTIONAL: Set SLACK_WEBHOOK_URL environment variable before deployment
# If not set, a fake webhook URL will be used for development
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/REAL/TOKEN"

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

### Secrets Management (€0 Cost)

ExpressOps uses External Secrets Operator with a Fake provider for managing secrets in development environments. This approach allows you to:

1. Use the same secrets management pattern as in production 
2. Not depend on cloud provider APIs (no need to pay)
3. Easily switch to a real secrets provider when needed

#### How It Works

- `k8s/secrets/fake-secretstore.yaml`: Defines a SecretStore using the Fake provider, which stores secrets directly in the manifest.
- `k8s/secrets/slack-externalsecret.yaml`: Defines an ExternalSecret that references the Fake SecretStore to create a Kubernetes Secret named `expressops-secrets`.

The Fake provider is used for development and testing environments where:
- You don't want to activate or pay (€0) for cloud provider secret management services
- You want a simple solution for local development
- You want to maintain the same External Secrets structure as in production

#### For Production

In a production environment, you would replace the Fake provider with a real secret management solution like:
- Google Secret Manager (€)
- AWS Secrets Manager (€)
- HashiCorp Vault (€)
- Or other supported providers

Then update the SecretStore configuration accordingly.

## ⚙️ Configuration example

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

## 🔍 Flow Discovery

ExpressOps provides a built-in flow to discover all available flows in the system:

```bash
curl "http://localhost:8080/flow?flowName=all-flows"
```

This will return a formatted list of all flows with their descriptions and plugins. The `all-flows` output is:

- Automatically displayed in full without truncation
- Formatted with each flow appearing on separate log lines in Kubernetes logs for better readability
- A great way to explore available flows when you're first getting started

When running in Kubernetes, you can view the formatted output with:

```bash
make k8s-port-forward  # In terminal 1
curl "http://localhost:8080/flow?flowName=all-flows"  # In terminal 2
make k8s-logs  # In terminal 3 to see the nicely formatted logs
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
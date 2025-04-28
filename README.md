## ExpressOps  <img src="docs/img/LOGO_EXPRESSOPS.png" alt="ExpressOps Logo" align="right" width="150" style="margin-top: 20px;">

> 🚨 <span style="color:red">**Note: Currently under active development**</span> - API and features may change without notice

ExpressOps: A lightweight flow orchestrator that:
- Loads plugins dynamically
- Chains plugins into workflows via YAML config
- Each plugin = one task (health checks, formatting, notifications, etc.)

## 📦 Docker Hub

The ExpressOps Docker image is available on Docker Hub at:
https://hub.docker.com/r/davidnull/expressops

> *Note: Currently only for testing. Will move to **expressopsfreepik/expressops** in the future*

You can pull it with:
```bash
docker pull davidnull/expressops:1.0.0
```

## 📑 Table of Contents

- [Requirements](#-requirements)
- [Installation](#-installation)
- [Usage](#-usage)
- [Getting Help](#-getting-help)
- [Configuration](#-configuration)
- [Secret Management](#-secret-management)
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
- Docker (for containerized deployment)
- Kubernetes (for production deployment)
- External Secrets Operator (for secret management)


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

## 🔍 Getting Help

The Makefile includes a comprehensive help system with information about available commands and configuration options.

Here are the main help commands you can use:

- `make help` - Shows you everything you can do with ExpressOps
- `make quick-help` - Just the essential commands you'll use most often 
- `make about` - Learn what ExpressOps is and how to get started
- `make config` - See all your current settings and how to change them

These commands display information in a paged format (similar to 'more' or 'less'). Press 'q' to exit the view.

The help system is organized into categories:
- Development commands (build, run)
- Docker commands (build, push, run)
- Kubernetes commands (deploy, status, logs)
- Helm commands (install, upgrade, uninstall)

![Make QuickHelp Command](docs/img/help.png)

**IMPORTANT:** The help commands are your best source of information about deployment options and required environment variables.

## 🗝️ Secret Management

ExpressOps uses External Secrets Operator with a ClusterSecretStore for managing secrets:

### How Secrets Work in ExpressOps

1. **ClusterSecretStore**: Stores the webhook URL at the cluster level
   ```yaml
   apiVersion: external-secrets.io/v1beta1
   kind: ClusterSecretStore
   metadata:
     name: expressops-fake-secretstore
   spec:
     provider:
       fake:
         data:
           - key: slack/webhook
             value:
               webhook_url: "https://hooks.slack.com/services/..."
   ```

2. **ExternalSecret**: Creates Kubernetes secrets from the ClusterSecretStore
   ```yaml
   apiVersion: external-secrets.io/v1beta1
   kind: ExternalSecret
   metadata:
     name: expressops-slack-external-secret
   spec:
     secretStoreRef:
       name: expressops-fake-secretstore
       kind: ClusterSecretStore
     target:
       name: expressops-slack-secret
       creationPolicy: Owner
     data:
       - secretKey: SLACK_WEBHOOK_URL
         remoteRef:
           key: slack/webhook
           property: webhook_url
   ```

3. **Deployment**: References the created Kubernetes secret
   ```yaml
   env:
     - name: SLACK_WEBHOOK_URL
       valueFrom:
         secretKeyRef:
           name: expressops-slack-secret
           key: SLACK_WEBHOOK_URL
   ```

### Deploying with Secrets

The recommended way to deploy with secrets is using one of these commands:

```bash
# Set your Slack webhook URL as an environment variable
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/TOKEN/HERE"

# Deploy with Helm (recommended)
make helm-install-with-secrets

# Or deploy with kubectl
make k8s-deploy-with-clustersecretstore
```

This approach keeps your secrets secure by:
- Not storing them in Git
- Only passing them at deployment time
- Storing them securely in Kubernetes

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

# Set SLACK_WEBHOOK_URL environment variable before deployment (required)
export SLACK_WEBHOOK_URL="https://hooks.slack.com/services/YOUR/REAL/TOKEN"

# Deploy to Kubernetes with secrets
make k8s-deploy-with-clustersecretstore

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



Hope you like ExpressOps and consider contributing! 🌟
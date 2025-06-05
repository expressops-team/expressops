# Manual Commands for ExpressOps CI/CD Pipeline

This document provides the manual commands that correspond to each step in the automated CI/CD pipeline. You can run these commands locally to perform the same actions as the pipeline.

## 1. Checkout Code
```bash
git checkout rama_nacho
```

## 2. Set up Go
```bash
# Install Go 1.24 if not already installed
# For Ubuntu/Debian:
wget https://go.dev/dl/go1.24.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

## 3. Run Tests
```bash
go test ./...
```

## 4. Run Linter
```bash
# Install golangci-lint if not already installed
# https://golangci-lint.run/usage/install/
golangci-lint run --timeout=5m
```

## 5. Login to Docker Hub
```bash
docker login -u $DOCKERHUB_USERNAME -p $DOCKERHUB_TOKEN
```

## 6. Build Docker Image
```bash
make docker-build
# This will create a .docker_tag file with the image tag
IMAGE_TAG=$(cat .docker_tag)
```

## 7. Scan Docker Image with Trivy
```bash
# Install Trivy if not already installed
# https://aquasecurity.github.io/trivy/latest/getting-started/installation/

# Run scan
trivy image expressopsfreepik/expressops:$IMAGE_TAG \
  --format table \
  --ignore-unfixed \
  --severity CRITICAL,HIGH
```

## 8. Push Docker Image
```bash
make docker-push
```

## 9. Update and Commit Helm Values (if needed)
```bash
# Check if values.yaml has been modified
git diff k3s/apps/expressops-app/expressops-chart/values.yaml

# If modified, commit and push
git add k3s/apps/expressops-app/expressops-chart/values.yaml
git commit -m "Update Helm chart image tag to $IMAGE_TAG"
git push origin rama_nacho
```

## Complete Pipeline in One Script
You can also create a script to run the entire pipeline:

```bash
#!/bin/bash
set -e  # Exit on any error

# Run tests
go test ./...

# Run linter
golangci-lint run --timeout=5m

# Login to Docker Hub
docker login -u $DOCKERHUB_USERNAME -p $DOCKERHUB_TOKEN

# Build Docker image
make docker-build
IMAGE_TAG=$(cat .docker_tag)
echo "Built image with tag: $IMAGE_TAG"

# Scan with Trivy
trivy image expressopsfreepik/expressops:$IMAGE_TAG \
  --format table \
  --ignore-unfixed \
  --severity CRITICAL,HIGH

# Push Docker image
make docker-push

# Update Helm values if needed
git diff k3s/apps/expressops-app/expressops-chart/values.yaml
if [[ $(git diff --name-only k3s/apps/expressops-app/expressops-chart/values.yaml) ]]; then
  git add k3s/apps/expressops-app/expressops-chart/values.yaml
  git commit -m "Update Helm chart image tag to $IMAGE_TAG"
  git push origin rama_nacho
fi

echo "Pipeline completed successfully!"
```

Save this as `run-pipeline.sh` and make it executable with `chmod +x run-pipeline.sh`. 
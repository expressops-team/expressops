#!/bin/bash

# Update ExpressOps to version 1.1.3
VERSION="1.1.3"
DOCKER_IMAGE="expressops:$VERSION"

echo "Updating ExpressOps to version $VERSION"

# Build the Docker image with the new version
echo "Building Docker image $DOCKER_IMAGE..."
docker build -t $DOCKER_IMAGE .

# Tag the image for Kubernetes use
if [ $? -eq 0 ]; then
    echo "✅ Docker image built successfully"
    echo "Tagging image for Kubernetes..."
    docker tag $DOCKER_IMAGE localhost:5000/$DOCKER_IMAGE
    
    # Push to local registry if available
    if docker info | grep -q "Registry:.*localhost:5000"; then
        echo "Pushing to local registry..."
        docker push localhost:5000/$DOCKER_IMAGE
    else
        echo "⚠️  Local registry not found. Skipping push."
    fi
    
    # Restart ExpressOps deployment with new version
    echo ""
    echo "To update the deployment, run:"
    echo "make upgrade VERSION=$VERSION"
    echo ""
    echo "Then verify metrics with:"
    echo "curl http://localhost:8080/metrics | grep expressops"
else
    echo "❌ Failed to build Docker image"
    exit 1
fi 
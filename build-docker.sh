#!/bin/bash

echo "======================================"
echo "Building EVE SDE Server Docker Image"
echo "======================================"
echo ""

# Build image
echo "Building image..."
docker build -t eve-sde-server:latest .

if [ $? -eq 0 ]; then
    echo ""
    echo "✓ Docker image built successfully!"
    echo ""
    echo "Image details:"
    docker images eve-sde-server:latest
    echo ""
    echo "To run the container:"
    echo "  docker run -p 8080:8080 -v sde-data:/app/data eve-sde-server:latest"
    echo ""
    echo "Or use Docker Compose:"
    echo "  docker-compose up -d"
else
    echo ""
    echo "✗ Build failed!"
    exit 1
fi

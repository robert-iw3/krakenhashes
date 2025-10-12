#!/bin/bash
# Script to build agent binaries for Docker image testing
# This mimics what GitHub Actions does before building the Docker image

set -e

echo "Building agent binaries for all platforms..."

# Change to agent directory
cd "$(dirname "$0")/../agent"

# Clean and build all platforms
make clean
make build-all

echo ""
echo "✅ Agent binaries built successfully!"
echo ""
echo "Binaries are in: ../bin/agent/"
echo ""
echo "Directory structure:"
ls -lR ../bin/agent/
echo ""
echo "You can now build the Docker image with:"
echo "  cd .."
echo "  docker build -f Dockerfile.prod -t krakenhashes:test ."

#!/bin/bash

set -e

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT_DIR"

echo "==> Building backend image with ko..."
KO_DOCKER_REPO=ko.local/shiru ko build . --bare

echo "==> Building and starting services..."
docker compose up --build -d "$@"

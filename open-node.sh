#!/bin/bash

# Start a temporary Node container mounting the project directory and entering webui
docker run -it --rm \
  -v "$(pwd):/app" \
  -w /app/webui \
  node:20 \
  bash

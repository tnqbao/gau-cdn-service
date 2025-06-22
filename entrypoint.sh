#!/bin/sh
# Start the application
echo "Starting API..."
if [ -f "gau-cdn-service.bin" ]; then
    ./gau-cdn-service.bin
else
    echo "Running main.go..."
    go run main.go
fi
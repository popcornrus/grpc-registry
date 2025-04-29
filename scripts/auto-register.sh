#!/bin/bash

echo "Starting extension server..."
cd /app
go run ./examples/extension/main.go --port=53000 --data-dir=/app/data &
EXTENSION_PID=$!

echo "Waiting for services to start..."
sleep 10

echo "Attempting auto-registration..."
go run ./examples/client/main.go --action=register --extension-id=rrweb-extension --extension-addr=extension-server:53000 --proxy=proxy-manager:50051
REGISTRATION_STATUS=$?

if [ $REGISTRATION_STATUS -eq 0 ]; then
  echo "Auto-registration completed successfully!"
else
  echo "Auto-registration failed with status: $REGISTRATION_STATUS"
fi

# Keep the extension server running
wait $EXTENSION_PID

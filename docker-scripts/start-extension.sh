#!/bin/sh

# Start the extension server
echo "Starting extension server..."
/app/extension-server --port=53000 --data-dir=/app/data &
EXTENSION_PID=$!

# Wait for the proxy manager and extension server to be ready
echo "Waiting for services to start..."
sleep 10

# Register the extension server with the proxy
echo "Attempting to register with proxy manager..."
/app/registration-client --extension-id=rrweb-extension --extension-addr=extension-server:53000 --proxy=proxy-manager:50051

# Keep the extension server running
wait $EXTENSION_PID

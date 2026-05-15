#!/usr/bin/env bash
set -e

echo "Waiting for services to be ready..."

# Wait for gateway
for i in {1..30}; do
    if curl -s http://localhost:8080/health > /dev/null; then
        echo "Gateway is up!"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "Timeout waiting for gateway"
        docker compose logs gateway
        exit 1
    fi
    sleep 2
done

echo "Testing Gateway -> Health Service communication via NATS..."
RESPONSE=$(curl -s http://localhost:8080/health)

echo "Response: $RESPONSE"

if echo "$RESPONSE" | grep -qE '"status":"(OK|UP)"'; then
    echo "SUCCESS: Services are communicating correctly."
else
    echo "FAILURE: Unexpected response format or content."
    docker compose logs health-service
    exit 1
fi

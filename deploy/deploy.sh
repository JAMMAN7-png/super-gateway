#!/bin/bash
# Super AI Gateway — Production Deploy Script
set -e

echo "=== Super AI Gateway Deploy ==="

# 1. Generate master key
MASTER_KEY=$(openssl rand -hex 32)
echo "Master key: sk-${MASTER_KEY:0:32}"
export MASTER_KEY="sk-${MASTER_KEY:0:32}"

# 2. Check config exists
if [ ! -f config.yaml ]; then
    echo "ERROR: config.yaml not found. Copy config.example.yaml → config.yaml and add your API keys."
    exit 1
fi

# 3. Ensure data directory
mkdir -p data

# 4. Build if needed
if [ ! -f gateway ] || [ cmd/gateway/main.go -nt gateway ]; then
    echo "Building gateway..."
    go build -o gateway ./cmd/gateway
fi

# 5. Install systemd service
sed "s/%MASTER_KEY%/${MASTER_KEY}/g" deploy/super-gateway.service > /etc/systemd/system/super-gateway.service
systemctl daemon-reload
systemctl enable super-gateway
systemctl restart super-gateway

# 6. Start SearXNG if docker available
if command -v docker &>/dev/null; then
    docker compose up -d searxng 2>/dev/null || echo "SearXNG start skipped (check docker-compose.yml)"
fi

echo ""
echo "=== Deploy Complete ==="
echo "Gateway:  http://localhost:3000"
echo "Dashboard: http://localhost:3000/dashboard"
echo "Master Key: sk-${MASTER_KEY:0:32}"
echo ""
echo "Test: curl http://localhost:3000/health"
echo "Create virtual key:"
echo "  curl -X POST http://localhost:3000/v1/keys \\"
echo "    -H 'Authorization: Bearer sk-${MASTER_KEY:0:32}' \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"label\":\"my-app\",\"rpm\":60}'"

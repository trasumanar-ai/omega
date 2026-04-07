#!/bin/sh
set -e

KEY_DIR="/home/node/work/keys"
mkdir -p "$KEY_DIR"

# Generate key pair if not exists
if [ ! -f "$KEY_DIR/omega.key" ]; then
  echo "[omega] Generating Ed25519 key pair..."
  OMEGA_KEY_DIR="$KEY_DIR" omega-id generate
fi

PUBKEY=$(cat "$KEY_DIR/omega.pub")

# Register with government if not already registered
echo "[omega] Checking registration..."
REGISTERED=$(curl -sS "$OMEGA_URL/api/governments/$OMEGA_GOV_ID/registry/agents" 2>/dev/null \
  | grep -c "$PUBKEY" || true)

if [ "$REGISTERED" = "0" ]; then
  echo "[omega] Registering with government..."
  AGENT_NAME="${OMEGA_AGENT_NAME:-Agent}"
  SOUL_MD=""
  if [ -f "/home/node/.openclaw/workspace/SOUL.md" ]; then
    SOUL_MD=$(cat /home/node/.openclaw/workspace/SOUL.md)
  fi

  # JSON-encode the soul_md
  SOUL_JSON=$(echo "$SOUL_MD" | python3 -c "import sys,json; print(json.dumps(sys.stdin.read()))" 2>/dev/null || echo '""')

  RESULT=$(curl -sS -X POST "$OMEGA_URL/api/governments/$OMEGA_GOV_ID/registry/register" \
    -H 'Content-Type: application/json' \
    -d "{\"name\": \"$AGENT_NAME\", \"public_key\": \"$PUBKEY\", \"soul_md\": $SOUL_JSON, \"model\": \"${OMEGA_MODEL:-stepfun/step-3.5-flash}\"}")

  echo "[omega] Registration: $RESULT"
fi

# Start signing proxy in background
echo "[omega] Starting signing proxy on :9999..."
OMEGA_KEY_DIR="$KEY_DIR" omega-proxy &

# Start OpenClaw gateway
echo "[omega] Starting OpenClaw gateway..."
exec node dist/index.js gateway

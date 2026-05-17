#!/bin/bash
# One-shot helper to plug an OpenRouter (or OpenAI-compatible) API key
# into the platform.
#
# Usage:
#   ./scripts/set-llm-key.sh <api-key>
#   ./scripts/set-llm-key.sh <api-key> <model-slug>
#
# Examples:
#   # OpenRouter free Llama 3.3 (default model)
#   ./scripts/set-llm-key.sh sk-or-v1-abc...
#
#   # OpenRouter free Qwen 2.5
#   ./scripts/set-llm-key.sh sk-or-v1-abc... 'qwen/qwen-2.5-72b-instruct:free'
#
# After running this, the AI service is restarted automatically.

set -eu

ENV_FILE="infrastructure/docker/.env"

if [ "$#" -lt 1 ]; then
  cat >&2 <<'USAGE'
Usage: ./scripts/set-llm-key.sh <api-key> [model-slug]

Get a free key at https://openrouter.ai/keys (no credit card needed).
USAGE
  exit 1
fi

KEY="$1"
MODEL="${2:-openai/gpt-oss-120b:free}"

if [ ! -f "$ENV_FILE" ]; then
  echo "Creating $ENV_FILE from .env.example" >&2
  cp infrastructure/docker/.env.example "$ENV_FILE"
fi

upsert() {
  local name="$1" value="$2"
  if grep -qE "^${name}=" "$ENV_FILE"; then
    # macOS-safe in-place sed.
    sed -i.bak "s|^${name}=.*|${name}=${value}|" "$ENV_FILE" && rm "$ENV_FILE.bak"
  else
    printf "\n%s=%s\n" "$name" "$value" >> "$ENV_FILE"
  fi
}

upsert "LLM_API_KEY" "$KEY"
upsert "LLM_MODEL" "$MODEL"
upsert "LLM_BASE_URL" "https://openrouter.ai/api/v1"

echo "Updated $ENV_FILE:"
echo "  LLM_API_KEY=${KEY:0:10}...   (length ${#KEY})"
echo "  LLM_MODEL=$MODEL"
echo "  LLM_BASE_URL=https://openrouter.ai/api/v1"
echo
echo "Restarting ai-service + interview-service..."

docker compose -f infrastructure/docker/docker-compose.yml \
  restart ai-service interview-service >/dev/null 2>&1 || true

echo "Done. Try a short interview turn — AI should now respond live."

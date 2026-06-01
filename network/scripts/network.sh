#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/env.sh"

ACTION="${1:-}"

compose() {
  docker compose -f "${COMPOSE_FILE}" "$@"
}

case "${ACTION}" in
  generate)
    bash "${SCRIPT_DIR}/generate.sh"
    ;;
  up)
    bash "${SCRIPT_DIR}/generate.sh"
    compose up -d
    ;;
  down)
    compose down -v
    ;;
  config)
    compose config
    ;;
  *)
    echo "Usage: $0 {generate|up|down|config}" >&2
    exit 1
    ;;
esac

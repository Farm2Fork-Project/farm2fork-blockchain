#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/env.sh"

ACTION="${1:-}"

compose() {
  docker compose -f "${COMPOSE_FILE}" "$@"
}

cleanup_generated_state() {
  rm -rf "${ORGANIZATIONS_DIR}" "${CHANNEL_ARTIFACTS_DIR}"
}

run_cli() {
  compose exec -T "${CLI_SERVICE}" "$@"
}

retry() {
  local attempts="${1}"
  shift
  local attempt

  for attempt in $(seq 1 "${attempts}"); do
    if "$@"; then
      return 0
    fi
    sleep 2
  done

  return 1
}

join_orderer_channel() {
  run_cli osnadmin channel list \
    --channelID "${CHANNEL_NAME}" \
    -o "${ORDERER_ADMIN_ADDRESS}" \
    --ca-file "${CONTAINER_ORDERER_CA}" \
    --client-cert "${CONTAINER_ORDERER_ADMIN_TLS_SIGN_CERT}" \
    --client-key "${CONTAINER_ORDERER_ADMIN_TLS_PRIVATE_KEY}" >/dev/null 2>&1 && return 0

  run_cli osnadmin channel join \
    --channelID "${CHANNEL_NAME}" \
    --config-block "${CONTAINER_CHANNEL_BLOCK_FILE}" \
    -o "${ORDERER_ADMIN_ADDRESS}" \
    --ca-file "${CONTAINER_ORDERER_CA}" \
    --client-cert "${CONTAINER_ORDERER_ADMIN_TLS_SIGN_CERT}" \
    --client-key "${CONTAINER_ORDERER_ADMIN_TLS_PRIVATE_KEY}"
}

join_peer_channel() {
  run_cli peer channel getinfo -c "${CHANNEL_NAME}" >/dev/null 2>&1 && return 0

  run_cli peer channel join -b "${CONTAINER_CHANNEL_BLOCK_FILE}"
}

case "${ACTION}" in
  generate)
    bash "${SCRIPT_DIR}/generate.sh"
    ;;
  up)
    bash "${SCRIPT_DIR}/generate.sh"
    compose up -d
    if [ ! -f "${CHANNEL_BLOCK_FILE}" ]; then
      echo "Missing channel block: ${CHANNEL_BLOCK_FILE}" >&2
      exit 1
    fi
    retry 10 join_orderer_channel
    retry 10 join_peer_channel
    ;;
  down)
    compose down -v
    ;;
  reset)
    compose down -v --remove-orphans
    cleanup_generated_state
    ;;
  config)
    compose config
    ;;
  *)
    echo "Usage: $0 {generate|up|down|reset|config}" >&2
    exit 1
    ;;
esac

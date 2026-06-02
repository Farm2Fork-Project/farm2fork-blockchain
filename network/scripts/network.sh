#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/env.sh"

ACTION="${1:-}"

compose() {
  docker compose -f "${COMPOSE_FILE}" "$@"
}

cleanup_generated_state() {
  rm -rf "${ORGANIZATIONS_DIR}" "${CHANNEL_ARTIFACTS_DIR}" "${NETWORK_DIR}/system-genesis-block"
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
  local channel_status

  channel_status="$(orderer_admin_cmd channel list \
    --channelID "${CHANNEL_NAME}" \
    2>/dev/null || true)"

  if [[ "${channel_status}" == *"Status: 200"* ]]; then
    return 0
  fi

  orderer_admin_cmd channel join \
    --channelID "${CHANNEL_NAME}" \
    --config-block "${CHANNEL_BLOCK_FILE}"
}

join_peer_channel() {
  peer_cmd peer channel getinfo -c "${CHANNEL_NAME}" >/dev/null 2>&1 && return 0

  peer_cmd peer channel join -b "${CHANNEL_BLOCK_FILE_CONTAINER}"
}

case "${ACTION}" in
  generate)
    bash "${SCRIPT_DIR}/generate.sh"
    ;;
  up)
    bash "${SCRIPT_DIR}/generate.sh"
    compose up -d
    ;;
  channel)
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
    echo "Usage: $0 {generate|up|channel|down|reset|config}" >&2
    exit 1
    ;;
esac

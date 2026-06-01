#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/env.sh"

has_generated_state() {
  [ -f "${CHANNEL_BLOCK_FILE}" ] || [ -d "${ORGANIZATIONS_DIR}" ]
}

has_complete_state() {
  [ -f "${CHANNEL_BLOCK_FILE}" ] && [ -f "${ORDERER_ADMIN_TLS_CERT_FILE}" ] && [ -f "${ORDERER_ADMIN_TLS_KEY_FILE}" ] && [ -d "${PEER_ADMIN_MSP_DIR}" ]
}

mkdir -p "${ORGANIZATIONS_DIR}" "${CHANNEL_ARTIFACTS_DIR}"

if has_complete_state; then
  echo "Using existing generated Fabric artifacts in ${ORGANIZATIONS_DIR} and ${CHANNEL_ARTIFACTS_DIR}" >&2
  exit 0
fi

if has_generated_state; then
  echo "Refusing to regenerate partial or stale Fabric artifacts. Run 'bash ${NETWORK_DIR}/scripts/network.sh reset' first." >&2
  exit 1
fi

docker compose -f "${COMPOSE_FILE}" run --rm --no-deps "${CLI_SERVICE}" bash -lc "
  set -euo pipefail
  cryptogen generate --config=/etc/hyperledger/fabric/config/crypto-config.yaml --output=/etc/hyperledger/fabric
  configtxgen -profile Farm2ForkChannel -channelID '${CHANNEL_NAME}' -outputBlock '${CONTAINER_CHANNEL_BLOCK_FILE}'
"

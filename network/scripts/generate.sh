#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/env.sh"

mkdir -p "${ORGANIZATIONS_DIR}" "${CHANNEL_ARTIFACTS_DIR}"

docker compose -f "${COMPOSE_FILE}" run --rm --no-deps "${CLI_SERVICE}" bash -lc "
  set -euo pipefail
  cryptogen generate --config=/etc/hyperledger/fabric/config/crypto-config.yaml --output=/etc/hyperledger/fabric
  configtxgen -profile Farm2ForkChannel -channelID '${CHANNEL_NAME}' -outputBlock '${CONTAINER_CHANNEL_BLOCK_FILE}'
"

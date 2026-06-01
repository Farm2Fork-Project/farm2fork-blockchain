#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "$0")" && pwd)/env.sh"

mkdir -p "${ORGANIZATIONS_DIR}" "${CHANNEL_ARTIFACTS_DIR}"

cryptogen generate --config="${NETWORK_DIR}/config/crypto-config.yaml" --output="${ORGANIZATIONS_DIR}"
configtxgen -profile Farm2ForkChannel -channelID "${CHANNEL_NAME}" -outputBlock "${CHANNEL_BLOCK_FILE}"

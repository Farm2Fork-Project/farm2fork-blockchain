#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export NETWORK_DIR="${ROOT_DIR}/network"
export FABRIC_CFG_PATH="${ROOT_DIR}/network/config"
export CHANNEL_NAME="${CHANNEL_NAME:-farm2forkchannel}"
export FABRIC_VERSION="${FABRIC_VERSION:-3.1.4}"
export ORDERER_NAME="${ORDERER_NAME:-orderer.farm2fork.com}"
export ORDERER_ADMIN_PORT="${ORDERER_ADMIN_PORT:-9443}"
export ORDERER_ADMIN_ADDRESS="${ORDERER_ADMIN_ADDRESS:-localhost:${ORDERER_ADMIN_PORT}}"
export ORGANIZATIONS_DIR="${NETWORK_DIR}/organizations"
export CHANNEL_ARTIFACTS_DIR="${NETWORK_DIR}/channel-artifacts"
export CHANNEL_BLOCK_FILE="${CHANNEL_ARTIFACTS_DIR}/${CHANNEL_NAME}.block"
export COMPOSE_FILE="${NETWORK_DIR}/compose/compose-net.yaml"

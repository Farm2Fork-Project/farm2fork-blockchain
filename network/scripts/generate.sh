#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "$0")" && pwd)/env.sh"

cryptogen generate --config="${ROOT_DIR}/network/config/crypto-config.yaml" --output="${ROOT_DIR}/network/organizations"
configtxgen -profile Farm2ForkGenesis -channelID system-channel -outputBlock "${ROOT_DIR}/network/system-genesis-block/genesis.block"
configtxgen -profile Farm2ForkChannel -outputCreateChannelTx "${ROOT_DIR}/network/channel-artifacts/${CHANNEL_NAME}.tx" -channelID "${CHANNEL_NAME}"

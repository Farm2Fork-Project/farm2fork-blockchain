#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export FABRIC_CFG_PATH="${ROOT_DIR}/network/config"
export CHANNEL_NAME="${CHANNEL_NAME:-farm2forkchannel}"
export FABRIC_VERSION="${FABRIC_VERSION:-3.1.4}"

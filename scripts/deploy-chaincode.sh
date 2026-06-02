#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/network/scripts/env.sh"

bash "${ROOT_DIR}/scripts/network-up.sh"
bash "${ROOT_DIR}/scripts/create-channel.sh"

docker compose -f "${COMPOSE_FILE}" exec -T "${CLI_SERVICE}" bash -lc "
  set -euo pipefail
  rm -f '${CONTAINER_CHAINCODE_PACKAGE_FILE}'

  peer lifecycle chaincode package '${CONTAINER_CHAINCODE_PACKAGE_FILE}' \
    --path '${CONTAINER_CHAINCODE_DIR}' \
    --lang '${CHAINCODE_LANGUAGE}' \
    --label '${CHAINCODE_LABEL}'

  peer lifecycle chaincode install '${CONTAINER_CHAINCODE_PACKAGE_FILE}'

  PACKAGE_ID=\$(peer lifecycle chaincode queryinstalled | awk -F '[, ]+' '/${CHAINCODE_LABEL}/ {print \$3; exit}')
  if [ -z \"\$PACKAGE_ID\" ]; then
    echo 'Unable to determine chaincode package ID' >&2
    exit 1
  fi

  peer lifecycle chaincode approveformyorg \
    -o '${ORDERER_ADMIN_ADDRESS}' \
    --channelID '${CHANNEL_NAME}' \
    --name '${CHAINCODE_NAME}' \
    --version '${CHAINCODE_VERSION}' \
    --package-id \"\$PACKAGE_ID\" \
    --sequence '${CHAINCODE_SEQUENCE}' \
    --tls \
    --cafile '${CONTAINER_ORDERER_CA}'

  peer lifecycle chaincode commit \
    -o '${ORDERER_ADMIN_ADDRESS}' \
    --channelID '${CHANNEL_NAME}' \
    --name '${CHAINCODE_NAME}' \
    --version '${CHAINCODE_VERSION}' \
    --sequence '${CHAINCODE_SEQUENCE}' \
    --peerAddresses 'peer0.farm2fork.com:7051' \
    --tlsRootCertFiles '/etc/hyperledger/fabric/peerOrganizations/farm2fork.com/peers/peer0.farm2fork.com/tls/ca.crt' \
    --tls \
    --cafile '${CONTAINER_ORDERER_CA}'
"

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/network/scripts/env.sh"

bash "${ROOT_DIR}/scripts/network-up.sh"
bash "${ROOT_DIR}/scripts/create-channel.sh"

committed_definition="$(peer_cmd peer lifecycle chaincode querycommitted --channelID "${CHANNEL_NAME}" --name "${CHAINCODE_NAME}" 2>/dev/null || true)"
if [[ "${committed_definition}" == *"Version: ${CHAINCODE_VERSION}, Sequence: ${CHAINCODE_SEQUENCE},"* ]]; then
  echo "Chaincode ${CHAINCODE_NAME} ${CHAINCODE_VERSION} sequence ${CHAINCODE_SEQUENCE} is already committed on ${CHANNEL_NAME}; skipping lifecycle deployment"
  exit 0
fi

if [ -n "${committed_definition}" ]; then
  echo "Committed chaincode definition differs from requested ${CHAINCODE_VERSION} sequence ${CHAINCODE_SEQUENCE}; deploying requested definition"
fi

rm -f "${CHAINCODE_PACKAGE_FILE}"

peer_host_cmd peer lifecycle chaincode package "${CHAINCODE_PACKAGE_FILE}" \
  --path "${CHAINCODE_DIR}" \
  --lang "${CHAINCODE_LANGUAGE}" \
  --label "${CHAINCODE_LABEL}"

docker cp "${CHAINCODE_PACKAGE_FILE}" "${PEER_CONTAINER_NAME}:${CHAINCODE_CONTAINER_PACKAGE_FILE}"

peer_cmd peer lifecycle chaincode install "${CHAINCODE_CONTAINER_PACKAGE_FILE}" || true

PACKAGE_ID="$(peer_cmd peer lifecycle chaincode queryinstalled | awk -F '[, ]+' "/${CHAINCODE_LABEL}/ {print \$3; exit}")"
if [ -z "${PACKAGE_ID}" ]; then
  echo "Unable to determine chaincode package ID" >&2
  exit 1
fi

peer_cmd peer lifecycle chaincode approveformyorg \
  -o "${ORDERER_ADDRESS}" \
  --channelID "${CHANNEL_NAME}" \
  --name "${CHAINCODE_NAME}" \
  --version "${CHAINCODE_VERSION}" \
  --package-id "${PACKAGE_ID}" \
  --sequence "${CHAINCODE_SEQUENCE}" \
  --tls \
  --cafile "${ORDERER_CA_FILE}"

peer_cmd peer lifecycle chaincode commit \
  -o "${ORDERER_ADDRESS}" \
  --channelID "${CHANNEL_NAME}" \
  --name "${CHAINCODE_NAME}" \
  --version "${CHAINCODE_VERSION}" \
  --sequence "${CHAINCODE_SEQUENCE}" \
  --peerAddresses 'peer0.farm2fork.com:7051' \
  --tlsRootCertFiles "${PEER_TLS_CA_FILE}" \
  --tls \
  --cafile "${ORDERER_CA_FILE}"

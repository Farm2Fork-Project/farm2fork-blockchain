#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [ -f "${ROOT_DIR}/.env" ]; then
  set -a
  source "${ROOT_DIR}/.env"
  set +a
fi

export NETWORK_DIR="${ROOT_DIR}/network"
export FABRIC_BIN_DIR="${FABRIC_BIN_DIR:-${ROOT_DIR}/bin}"
export FABRIC_CFG_PATH="${ROOT_DIR}/network/config"
export CHANNEL_NAME="${CHANNEL_NAME:-farm2forkchannel}"
export FABRIC_VERSION="${FABRIC_VERSION:-3.1.4}"
export ORDERER_NAME="${ORDERER_NAME:-orderer.farm2fork.com}"
export ORDERER_CONTAINER_NAME="${ORDERER_CONTAINER_NAME:-orderer.farm2fork.com}"
export ORDERER_ADMIN_PORT="${ORDERER_ADMIN_PORT:-9443}"
export ORDERER_PORT="${ORDERER_PORT:-7050}"
export ORDERER_ADDRESS="${ORDERER_ADDRESS:-${ORDERER_NAME}:${ORDERER_PORT}}"
export ORDERER_ADMIN_ADDRESS="${ORDERER_ADMIN_ADDRESS:-${ORDERER_NAME}:${ORDERER_ADMIN_PORT}}"
export CHAINCODE_NAME="${CHAINCODE_NAME:-farm2fork-chaincode}"
export CHAINCODE_LANGUAGE="${CHAINCODE_LANGUAGE:-golang}"
export CHAINCODE_VERSION="${CHAINCODE_VERSION:-1.0}"
export CHAINCODE_SEQUENCE="${CHAINCODE_SEQUENCE:-1}"
export CHAINCODE_LABEL="${CHAINCODE_LABEL:-${CHAINCODE_NAME}_1.0}"
export CHAINCODE_DIR="${CHAINCODE_DIR:-${ROOT_DIR}/chaincode/${CHAINCODE_NAME}}"
export CHAINCODE_PACKAGE_FILE="${CHAINCODE_PACKAGE_FILE:-${ROOT_DIR}/chaincode/${CHAINCODE_NAME}.tar.gz}"
export ORGANIZATIONS_DIR="${NETWORK_DIR}/organizations"
export CHANNEL_ARTIFACTS_DIR="${NETWORK_DIR}/channel-artifacts"
export CHANNEL_BLOCK_FILE="${CHANNEL_ARTIFACTS_DIR}/${CHANNEL_NAME}.block"
export CHANNEL_BLOCK_FILE_CONTAINER="${CHANNEL_BLOCK_FILE_CONTAINER:-/etc/hyperledger/fabric/channel-artifacts/${CHANNEL_NAME}.block}"
export ORDERER_ORGS_DIR="${ORGANIZATIONS_DIR}/ordererOrganizations/farm2fork.com"
export ORDERER_MSP_DIR="${ORDERER_ORGS_DIR}/orderers/${ORDERER_NAME}/msp"
export ORDERER_TLS_DIR="${ORDERER_ORGS_DIR}/orderers/${ORDERER_NAME}/tls"
export ORDERER_CA_HOST_FILE="${ORDERER_CA_HOST_FILE:-${ORDERER_TLS_DIR}/ca.crt}"
export ORDERER_CA_FILE="${ORDERER_CA_FILE:-/etc/hyperledger/fabric/orderer/tls/ca.crt}"
export ORDERER_ADMIN_TLS_CERT_FILE="${ORDERER_ADMIN_TLS_CERT_FILE:-${ORGANIZATIONS_DIR}/ordererOrganizations/farm2fork.com/users/Admin@farm2fork.com/tls/client.crt}"
export ORDERER_ADMIN_TLS_KEY_FILE="${ORDERER_ADMIN_TLS_KEY_FILE:-${ORGANIZATIONS_DIR}/ordererOrganizations/farm2fork.com/users/Admin@farm2fork.com/tls/client.key}"
export PEER_ORGS_DIR="${ORGANIZATIONS_DIR}/peerOrganizations/farm2fork.com"
export PEER_MSP_DIR="${PEER_ORGS_DIR}/peers/peer0.farm2fork.com/msp"
export PEER_TLS_DIR="${PEER_ORGS_DIR}/peers/peer0.farm2fork.com/tls"
export PEER_TLS_CA_FILE="${PEER_TLS_CA_FILE:-/etc/hyperledger/fabric/tls/ca.crt}"
export PEER_ADMIN_MSP_HOST_DIR="${PEER_ADMIN_MSP_HOST_DIR:-${ORGANIZATIONS_DIR}/peerOrganizations/farm2fork.com/users/Admin@farm2fork.com/msp}"
export PEER_ADMIN_MSP_DIR="${PEER_ADMIN_MSP_DIR:-/etc/hyperledger/fabric/admin/msp}"
export PEER_CONTAINER_NAME="${PEER_CONTAINER_NAME:-peer0.farm2fork.com}"
export CHAINCODE_CONTAINER_PACKAGE_FILE="${CHAINCODE_CONTAINER_PACKAGE_FILE:-/tmp/${CHAINCODE_LABEL}.tar.gz}"
export COMPOSE_FILE="${NETWORK_DIR}/compose/compose-net.yaml"
export PATH="${FABRIC_BIN_DIR}:${PATH}"

peer_cmd() {
  if [ "${1:-}" = peer ]; then
    shift
  fi
  docker exec \
    -e CORE_PEER_LOCALMSPID="${CORE_PEER_LOCALMSPID:-Farm2ForkMSP}" \
    -e CORE_PEER_MSPCONFIGPATH="${PEER_ADMIN_MSP_DIR}" \
    -e CORE_PEER_ADDRESS="${CORE_PEER_ADDRESS:-peer0.farm2fork.com:7051}" \
    -e CORE_PEER_TLS_ENABLED="${CORE_PEER_TLS_ENABLED:-true}" \
    -e CORE_PEER_TLS_ROOTCERT_FILE="${CORE_PEER_TLS_ROOTCERT_FILE:-/etc/hyperledger/fabric/tls/ca.crt}" \
    "${PEER_CONTAINER_NAME}" \
    peer "$@"
}

peer_host_cmd() {
  if [ "${1:-}" = peer ]; then
    shift
  fi
  FABRIC_CFG_PATH="${ROOT_DIR}/config" \
  CORE_PEER_LOCALMSPID="${CORE_PEER_LOCALMSPID:-Farm2ForkMSP}" \
  CORE_PEER_MSPCONFIGPATH="${PEER_ADMIN_MSP_HOST_DIR}" \
  CORE_PEER_ADDRESS="${CORE_PEER_ADDRESS:-peer0.farm2fork.com:7051}" \
  CORE_PEER_TLS_ENABLED="${CORE_PEER_TLS_ENABLED:-true}" \
  CORE_PEER_TLS_ROOTCERT_FILE="${CORE_PEER_TLS_ROOTCERT_FILE:-${PEER_TLS_DIR}/ca.crt}" \
  peer "$@"
}

peer_container_cmd() {
  peer_cmd "$@"
}

orderer_admin_cmd() {
  osnadmin "$@" \
    -o "localhost:${ORDERER_ADMIN_PORT}" \
    --ca-file "${ORDERER_CA_HOST_FILE}" \
    --client-cert "${ORDERER_ADMIN_TLS_CERT_FILE}" \
    --client-key "${ORDERER_ADMIN_TLS_KEY_FILE}"
}

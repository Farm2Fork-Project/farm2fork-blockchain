#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
source "${SCRIPT_DIR}/env.sh"

has_generated_state() {
  [ -f "${CHANNEL_BLOCK_FILE}" ] || [ -d "${ORDERER_ORGS_DIR}" ] || [ -d "${PEER_ORGS_DIR}" ]
}

has_complete_state() {
  [ -f "${CHANNEL_BLOCK_FILE}" ] &&
    [ -d "${ORDERER_MSP_DIR}" ] &&
    [ -d "${ORDERER_TLS_DIR}" ] &&
    [ -f "${ORDERER_ADMIN_TLS_CERT_FILE}" ] &&
    [ -f "${ORDERER_ADMIN_TLS_KEY_FILE}" ] &&
    [ -d "${PEER_MSP_DIR}" ] &&
    [ -d "${PEER_TLS_DIR}" ] &&
    [ -d "${PEER_ADMIN_MSP_HOST_DIR}" ]
}

if has_complete_state; then
  echo "Using existing generated Fabric artifacts in ${ORGANIZATIONS_DIR} and ${CHANNEL_ARTIFACTS_DIR}" >&2
  exit 0
fi

if has_generated_state; then
  echo "Refusing to regenerate partial or stale Fabric artifacts. Run 'bash ${NETWORK_DIR}/scripts/network.sh reset' first." >&2
  exit 1
fi

mkdir -p "${ORGANIZATIONS_DIR}" "${CHANNEL_ARTIFACTS_DIR}"

cryptogen generate --config="${NETWORK_DIR}/config/crypto-config.yaml" --output="${ORGANIZATIONS_DIR}"

ORDERER_TLS_CA_CERT="$(find "${ORGANIZATIONS_DIR}/ordererOrganizations/farm2fork.com/tlsca" -name '*-cert.pem' -type f | head -n 1)"
ORDERER_TLS_CA_KEY="$(find "${ORGANIZATIONS_DIR}/ordererOrganizations/farm2fork.com/tlsca" -name 'priv_sk' -type f | head -n 1)"

if [ -z "${ORDERER_TLS_CA_CERT}" ] || [ -z "${ORDERER_TLS_CA_KEY}" ]; then
  echo "Unable to locate orderer TLS CA materials" >&2
  exit 1
fi

ORDERER_TLS_WORKDIR="$(mktemp -d)"
trap 'rm -rf "${ORDERER_TLS_WORKDIR}"' EXIT

cat > "${ORDERER_TLS_WORKDIR}/openssl.cnf" <<EOF
[ req ]
prompt = no
distinguished_name = dn
req_extensions = v3_req

[ dn ]
CN = orderer.farm2fork.com

[ v3_req ]
subjectAltName = @alt_names

[ alt_names ]
DNS.1 = orderer.farm2fork.com
DNS.2 = orderer
DNS.3 = localhost
IP.1 = 127.0.0.1
EOF

openssl req -new -nodes -newkey rsa:2048 \
  -keyout "${ORDERER_TLS_DIR}/server.key" \
  -out "${ORDERER_TLS_WORKDIR}/server.csr" \
  -config "${ORDERER_TLS_WORKDIR}/openssl.cnf" \
  >/dev/null 2>&1

openssl x509 -req \
  -in "${ORDERER_TLS_WORKDIR}/server.csr" \
  -CA "${ORDERER_TLS_CA_CERT}" \
  -CAkey "${ORDERER_TLS_CA_KEY}" \
  -CAcreateserial \
  -out "${ORDERER_TLS_DIR}/server.crt" \
  -days 3650 \
  -extfile "${ORDERER_TLS_WORKDIR}/openssl.cnf" \
  -extensions v3_req \
  >/dev/null 2>&1

rm -f "${ORDERER_TLS_CA_CERT}.srl"

configtxgen -profile Farm2ForkChannel -channelID "${CHANNEL_NAME}" -outputBlock "${CHANNEL_BLOCK_FILE}"

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/network/scripts/env.sh"

if [ "${SMOKE_RESET_NETWORK:-true}" = true ]; then
  echo "[0/8] reset local Fabric network"
  bash "${ROOT_DIR}/scripts/network-down.sh"
fi

echo "[1/8] network availability"
bash "${ROOT_DIR}/scripts/network-up.sh"
docker ps --format '{{.Names}}' | rg 'orderer\.farm2fork\.com|peer0\.farm2fork\.com' >/dev/null

echo "[2/8] channel readiness"
bash "${ROOT_DIR}/scripts/create-channel.sh"
test -f "${CHANNEL_BLOCK_FILE}"

echo "[3/8] chaincode readiness"
bash "${ROOT_DIR}/scripts/deploy-chaincode.sh"
peer_cmd peer lifecycle chaincode querycommitted --channelID "${CHANNEL_NAME}" --name "${CHAINCODE_NAME}" >/dev/null

echo "[4/8] payment write success"
peer_cmd peer chaincode invoke \
  -o "${ORDERER_ADDRESS}" \
  --tls \
  --cafile "${ORDERER_CA_FILE}" \
  --waitForEvent \
  --waitForEventTimeout 60s \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["RecordPayment","payment-001","order-001","buyer-001","farmer-001","1500","PKR","stripe","2026-06-01T12:00:00Z"]}'

echo "[5/8] payment query success"
peer_cmd peer chaincode query \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["GetTransactionByReferenceId","payment-001"]}' | tee /tmp/payment.json
rg '"orderId":"order-001"' /tmp/payment.json
rg '"buyerId":"buyer-001"' /tmp/payment.json
rg '"farmerId":"farmer-001"' /tmp/payment.json

echo "[6/8] supply chain write success"
peer_cmd peer chaincode invoke \
  -o "${ORDERER_ADDRESS}" \
  --tls \
  --cafile "${ORDERER_CA_FILE}" \
  --waitForEvent \
  --waitForEventTimeout 60s \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["RecordSupplyChainEvent","product-001:event-001","Product","product-001","farmer-001","listed","Lahore","farmer-001","farmer","2026-06-01T12:05:00Z"]}'

echo "[7/8] supply chain query success"
peer_cmd peer chaincode query \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["GetTransactionByReferenceId","product-001:event-001"]}' | tee /tmp/supply-chain.json
rg '"productId":"product-001"' /tmp/supply-chain.json
rg '"eventType":"listed"' /tmp/supply-chain.json
rg '"actorRole":"farmer"' /tmp/supply-chain.json

echo "[8/8] round-trip verification success"
peer_cmd peer chaincode query \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["GetHistoryForKey","product-001:event-001"]}' | tee /tmp/history.json
rg 'product-001:event-001' /tmp/history.json

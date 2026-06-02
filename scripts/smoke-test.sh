#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/network/scripts/env.sh"

run_cli() {
  docker compose -f "${COMPOSE_FILE}" exec -T "${CLI_SERVICE}" "$@"
}

echo "[1/8] network availability"
bash "${ROOT_DIR}/scripts/network-up.sh"
docker ps --format '{{.Names}}' | rg 'orderer\.farm2fork\.com|peer0\.farm2fork\.com|cli\.farm2fork\.com' >/dev/null

echo "[2/8] channel readiness"
bash "${ROOT_DIR}/scripts/create-channel.sh"
test -f "${CHANNEL_BLOCK_FILE}"

echo "[3/8] chaincode readiness"
bash "${ROOT_DIR}/scripts/deploy-chaincode.sh"
run_cli peer lifecycle chaincode querycommitted --channelID "${CHANNEL_NAME}" --name "${CHAINCODE_NAME}" >/dev/null

echo "[4/8] payment write success"
run_cli peer chaincode invoke \
  -o "${ORDERER_ADMIN_ADDRESS}" \
  --tls \
  --cafile "${CONTAINER_ORDERER_CA}" \
  --waitForEvent \
  --waitForEventTimeout 60s \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["RecordPayment","payment-001","order-001","buyer-001","farmer-001","1500","PKR","stripe","2026-06-01T12:00:00Z"]}'

echo "[5/8] payment query success"
run_cli peer chaincode query \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["GetTransactionByReferenceId","payment-001"]}' | tee /tmp/payment.json
rg '"orderId":"order-001"' /tmp/payment.json
rg '"buyerId":"buyer-001"' /tmp/payment.json
rg '"farmerId":"farmer-001"' /tmp/payment.json

echo "[6/8] supply chain write success"
run_cli peer chaincode invoke \
  -o "${ORDERER_ADMIN_ADDRESS}" \
  --tls \
  --cafile "${CONTAINER_ORDERER_CA}" \
  --waitForEvent \
  --waitForEventTimeout 60s \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["RecordSupplyChainEvent","product-001:event-001","Product","product-001","farmer-001","listed","Lahore","farmer-001","farmer","2026-06-01T12:05:00Z"]}'

echo "[7/8] supply chain query success"
run_cli peer chaincode query \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["GetTransactionByReferenceId","product-001:event-001"]}' | tee /tmp/supply-chain.json
rg '"productId":"product-001"' /tmp/supply-chain.json
rg '"eventType":"listed"' /tmp/supply-chain.json
rg '"actorRole":"farmer"' /tmp/supply-chain.json

echo "[8/8] round-trip verification success"
run_cli peer chaincode query \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["GetHistoryForKey","product-001:event-001"]}' | tee /tmp/history.json
rg 'product-001:event-001' /tmp/history.json

#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/network/scripts/env.sh"

if [ "${SMOKE_RESET_NETWORK:-true}" = true ]; then
  echo "[0/9] reset local Fabric network"
  bash "${ROOT_DIR}/scripts/network-down.sh"
fi

echo "[1/9] network availability"
bash "${ROOT_DIR}/scripts/network-up.sh"
docker ps --format '{{.Names}}' | rg 'orderer\.farm2fork\.com|peer0\.farm2fork\.com' >/dev/null

echo "[2/9] channel readiness"
bash "${ROOT_DIR}/scripts/create-channel.sh"
test -f "${CHANNEL_BLOCK_FILE}"

echo "[3/9] chaincode readiness"
bash "${ROOT_DIR}/scripts/deploy-chaincode.sh"
peer_cmd peer lifecycle chaincode querycommitted --channelID "${CHANNEL_NAME}" --name "${CHAINCODE_NAME}" >/dev/null

echo "[4/9] payment write success"
peer_cmd peer chaincode invoke \
  -o "${ORDERER_ADDRESS}" \
  --tls \
  --cafile "${ORDERER_CA_FILE}" \
  --waitForEvent \
  --waitForEventTimeout 60s \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["RecordPayment","outbox-payment-001","payment-001","order-001","buyer-001","farmer-001","1500","PKR","stripe","2026-08-11T12:00:00Z"]}'

echo "[5/9] payment query success"
peer_cmd peer chaincode query \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["GetTransactionByLedgerKey","outbox-payment-001"]}' | tee /tmp/payment.json
rg '"orderId":"order-001"' /tmp/payment.json
rg '"buyerId":"buyer-001"' /tmp/payment.json
rg '"farmerId":"farmer-001"' /tmp/payment.json

echo "[6/9] multi-product shipment write success"
peer_cmd peer chaincode invoke \
  -o "${ORDERER_ADDRESS}" \
  --tls \
  --cafile "${ORDERER_CA_FILE}" \
  --waitForEvent \
  --waitForEventTimeout 60s \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["RecordSupplyChainEvent","outbox-shipment-001-apple","shipment-001","Shipment","product-apple","farmer-001","shipment_assigned","Lahore, Punjab","transporter-001","transporter","2026-08-11T12:05:00Z"]}'

peer_cmd peer chaincode invoke \
  -o "${ORDERER_ADDRESS}" \
  --tls \
  --cafile "${ORDERER_CA_FILE}" \
  --waitForEvent \
  --waitForEventTimeout 60s \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["RecordSupplyChainEvent","outbox-shipment-001-mango","shipment-001","Shipment","product-mango","farmer-001","shipment_assigned","Lahore, Punjab","transporter-001","transporter","2026-08-11T12:05:00Z"]}'

echo "[7/9] shipment reference query success"
peer_cmd peer chaincode query \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["GetTransactionsByReference","Shipment","shipment-001"]}' | tee /tmp/shipment.json
rg '"productId":"product-apple"' /tmp/shipment.json
rg '"productId":"product-mango"' /tmp/shipment.json
rg '"referenceId":"shipment-001"' /tmp/shipment.json

echo "[8/9] product timeline query success"
peer_cmd peer chaincode query \
  -C "${CHANNEL_NAME}" \
  -n "${CHAINCODE_NAME}" \
  -c '{"Args":["GetTransactionsByProductId","product-apple"]}' | tee /tmp/product.json
rg '"productId":"product-apple"' /tmp/product.json
if rg '"productId":"product-mango"' /tmp/product.json; then
  echo "Product query returned a different product" >&2
  exit 1
fi

echo "[9/9] round-trip verification success"

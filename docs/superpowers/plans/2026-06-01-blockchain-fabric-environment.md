# Farm2Fork Fabric Environment Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a self-contained, Dockerized Hyperledger Fabric `v3.1.4` development environment in `farm2fork-blockchain` with Go `1.22.x` chaincode, exact master-context payload field names, and a smoke test that performs a full write/query/history round trip.

**Architecture:** Adapt the official `fabric-samples/test-network` layout into `network/`, rename the org/channel assets for Farm2Fork, then add a focused Go chaincode package that stores typed transaction records for `payment` and `supply_chain_event`. Keep the first pass single-org, single-peer, single-node-Raft, and use `GetHistoryForKey()` for history verification instead of introducing CouchDB.

**Tech Stack:** Hyperledger Fabric `3.1.4`, Go `1.22.x`, Docker Compose, Bash, Fabric peer lifecycle commands, Go unit tests

---

### Task 1: Create Repo Skeleton And Pin Versions

**Files:**
- Modify: `farm2fork-blockchain/README.md`
- Create: `farm2fork-blockchain/.env.example`
- Create: `farm2fork-blockchain/.gitignore`
- Create: `farm2fork-blockchain/docs/architecture.md`

- [ ] **Step 1: Write the failing documentation expectations**

Create `farm2fork-blockchain/docs/architecture.md` with the pinned-version contract:

```md
# Farm2Fork Blockchain Architecture Notes

- Hyperledger Fabric version: 3.1.4
- Go version: 1.22.x
- Channel name: farm2forkchannel
- Orderer: orderer.farm2fork.com
- Peer: peer0.farm2fork.com
- History query: GetHistoryForKey()
- Not in scope: CouchDB, Fabric CA, backend SDK integration, multi-org
```

- [ ] **Step 2: Add repo-level environment defaults**

Create `farm2fork-blockchain/.env.example`:

```env
FABRIC_VERSION=3.1.4
GO_VERSION=1.22.12
CHANNEL_NAME=farm2forkchannel
CHAINCODE_NAME=farm2fork-chaincode
CHAINCODE_LABEL=farm2fork-chaincode_1.0
CHAINCODE_LANGUAGE=golang
CHAINCODE_VERSION=1.0
CHAINCODE_SEQUENCE=1
ORDERER_NAME=orderer.farm2fork.com
PEER_NAME=peer0.farm2fork.com
COMPOSE_PROJECT_NAME=farm2forkfabric
```

- [ ] **Step 3: Add ignore rules for generated artifacts**

Create `farm2fork-blockchain/.gitignore`:

```gitignore
.env
network/channel-artifacts/
network/organizations/peerOrganizations/
network/organizations/ordererOrganizations/
network/system-genesis-block/
*.tar.gz
*.log
```

- [ ] **Step 4: Replace the placeholder README with pinned quick-start content**

Update `farm2fork-blockchain/README.md`:

```md
# farm2fork-blockchain

Farm2Fork local Hyperledger Fabric environment.

## Versions

- Hyperledger Fabric: 3.1.4
- Go: 1.22.x

## Quick start

```bash
cp .env.example .env
bash scripts/network-up.sh
bash scripts/create-channel.sh
bash scripts/deploy-chaincode.sh
bash scripts/smoke-test.sh
```
```

- [ ] **Step 5: Run lightweight verification**

Run: `rg -n "3.1.4|1.22.x|GetHistoryForKey" README.md .env.example docs/architecture.md`
Expected: matches in all three files

- [ ] **Step 6: Commit**

```bash
git -C farm2fork-blockchain add README.md .env.example .gitignore docs/architecture.md
git -C farm2fork-blockchain commit -m "chore: pin Fabric and Go environment versions"
```

### Task 2: Vendor And Adapt Fabric Network Assets

**Files:**
- Create: `farm2fork-blockchain/network/config/configtx.yaml`
- Create: `farm2fork-blockchain/network/config/crypto-config.yaml`
- Create: `farm2fork-blockchain/network/compose/compose-net.yaml`
- Create: `farm2fork-blockchain/network/scripts/env.sh`
- Create: `farm2fork-blockchain/network/scripts/generate.sh`
- Create: `farm2fork-blockchain/network/scripts/network.sh`

- [ ] **Step 1: Define the single-org crypto material**

Create `farm2fork-blockchain/network/config/crypto-config.yaml`:

```yaml
OrdererOrgs:
  - Name: Orderer
    Domain: farm2fork.com
    Specs:
      - Hostname: orderer

PeerOrgs:
  - Name: Farm2Fork
    Domain: farm2fork.com
    EnableNodeOUs: true
    Template:
      Count: 1
    Users:
      Count: 1
```

- [ ] **Step 2: Define the channel and single-node-Raft profile**

Create `farm2fork-blockchain/network/config/configtx.yaml` with the channel name and Raft orderer:

```yaml
Organizations:
  - &OrdererOrg
    Name: OrdererMSP
    ID: OrdererMSP
    MSPDir: ../organizations/ordererOrganizations/farm2fork.com/msp
  - &Farm2ForkMSP
    Name: Farm2ForkMSP
    ID: Farm2ForkMSP
    MSPDir: ../organizations/peerOrganizations/farm2fork.com/msp
    AnchorPeers:
      - Host: peer0.farm2fork.com
        Port: 7051

Capabilities:
  Channel: &ChannelCapabilities
    V3_0: true
  Orderer: &OrdererCapabilities
    V3_0: true
  Application: &ApplicationCapabilities
    V3_0: true

Profiles:
  Farm2ForkGenesis:
    Orderer:
      OrdererType: etcdraft
      Addresses:
        - orderer.farm2fork.com:7050
      Organizations:
        - *OrdererOrg
      EtcdRaft:
        Consenters:
          - Host: orderer.farm2fork.com
            Port: 7050
            ClientTLSCert: ../organizations/ordererOrganizations/farm2fork.com/orderers/orderer.farm2fork.com/tls/server.crt
            ServerTLSCert: ../organizations/ordererOrganizations/farm2fork.com/orderers/orderer.farm2fork.com/tls/server.crt
      Capabilities: *OrdererCapabilities
    Capabilities: *ChannelCapabilities
  Farm2ForkChannel:
    Consortium: SampleConsortium
    Application:
      Organizations:
        - *Farm2ForkMSP
      Capabilities: *ApplicationCapabilities
```

- [ ] **Step 3: Add Docker Compose for peer and orderer**

Create `farm2fork-blockchain/network/compose/compose-net.yaml`:

```yaml
services:
  orderer.farm2fork.com:
    image: hyperledger/fabric-orderer:${FABRIC_VERSION}
    container_name: orderer.farm2fork.com
    environment:
      - ORDERER_GENERAL_LISTENADDRESS=0.0.0.0
      - ORDERER_GENERAL_LISTENPORT=7050
      - ORDERER_GENERAL_LOCALMSPID=OrdererMSP
      - ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp
      - ORDERER_GENERAL_BOOTSTRAPMETHOD=none
      - ORDERER_CHANNELPARTICIPATION_ENABLED=true
      - ORDERER_ADMIN_TLS_ENABLED=true
      - ORDERER_ADMIN_TLS_PRIVATEKEY=/var/hyperledger/orderer/tls/server.key
      - ORDERER_ADMIN_TLS_CERTIFICATE=/var/hyperledger/orderer/tls/server.crt
      - ORDERER_ADMIN_TLS_ROOTCAS=[/var/hyperledger/orderer/tls/ca.crt]
    ports:
      - "7050:7050"
    volumes:
      - ../organizations/ordererOrganizations/farm2fork.com/orderers/orderer.farm2fork.com/msp:/var/hyperledger/orderer/msp
      - ../organizations/ordererOrganizations/farm2fork.com/orderers/orderer.farm2fork.com/tls:/var/hyperledger/orderer/tls

  peer0.farm2fork.com:
    image: hyperledger/fabric-peer:${FABRIC_VERSION}
    container_name: peer0.farm2fork.com
    environment:
      - CORE_PEER_ID=peer0.farm2fork.com
      - CORE_PEER_ADDRESS=peer0.farm2fork.com:7051
      - CORE_PEER_LISTENADDRESS=0.0.0.0:7051
      - CORE_PEER_LOCALMSPID=Farm2ForkMSP
      - CORE_PEER_MSPCONFIGPATH=/etc/hyperledger/fabric/msp
      - CORE_LEDGER_STATE_STATEDATABASE=goleveldb
    ports:
      - "7051:7051"
    volumes:
      - ../organizations/peerOrganizations/farm2fork.com/peers/peer0.farm2fork.com/msp:/etc/hyperledger/fabric/msp
      - ../organizations/peerOrganizations/farm2fork.com/peers/peer0.farm2fork.com/tls:/etc/hyperledger/fabric/tls
```

- [ ] **Step 4: Add environment helper and generation script**

Create `farm2fork-blockchain/network/scripts/env.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export FABRIC_CFG_PATH="${ROOT_DIR}/network/config"
export CHANNEL_NAME="${CHANNEL_NAME:-farm2forkchannel}"
export FABRIC_VERSION="${FABRIC_VERSION:-3.1.4}"
```

Create `farm2fork-blockchain/network/scripts/generate.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

source "$(cd "$(dirname "$0")" && pwd)/env.sh"

cryptogen generate --config="${ROOT_DIR}/network/config/crypto-config.yaml" --output="${ROOT_DIR}/network/organizations"
configtxgen -profile Farm2ForkGenesis -channelID system-channel -outputBlock "${ROOT_DIR}/network/system-genesis-block/genesis.block"
configtxgen -profile Farm2ForkChannel -outputCreateChannelTx "${ROOT_DIR}/network/channel-artifacts/${CHANNEL_NAME}.tx" -channelID "${CHANNEL_NAME}"
```

- [ ] **Step 5: Add network entry script and verify syntax**

Create `farm2fork-blockchain/network/scripts/network.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ACTION="${1:-}"
case "${ACTION}" in
  up) echo "Starting Farm2Fork Fabric network" ;;
  down) echo "Stopping Farm2Fork Fabric network" ;;
  *) echo "Usage: $0 {up|down}" && exit 1 ;;
esac
```

Run: `bash -n network/scripts/env.sh network/scripts/generate.sh network/scripts/network.sh`
Expected: no output

- [ ] **Step 6: Commit**

```bash
git -C farm2fork-blockchain add network/config network/compose network/scripts
git -C farm2fork-blockchain commit -m "feat: add Farm2Fork Fabric network assets"
```

### Task 3: Add Top-Level Lifecycle Scripts

**Files:**
- Create: `farm2fork-blockchain/scripts/network-up.sh`
- Create: `farm2fork-blockchain/scripts/create-channel.sh`
- Create: `farm2fork-blockchain/scripts/deploy-chaincode.sh`
- Create: `farm2fork-blockchain/scripts/network-down.sh`

- [ ] **Step 1: Create the network startup wrapper**

Create `farm2fork-blockchain/scripts/network-up.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

source "${ROOT_DIR}/network/scripts/env.sh"
mkdir -p network/channel-artifacts network/system-genesis-block
bash "${ROOT_DIR}/network/scripts/generate.sh"
docker compose -f "${ROOT_DIR}/network/compose/compose-net.yaml" up -d
```

- [ ] **Step 2: Create the channel wrapper**

Create `farm2fork-blockchain/scripts/create-channel.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/network/scripts/env.sh"

peer channel create \
  -o orderer.farm2fork.com:7050 \
  -c "${CHANNEL_NAME}" \
  -f "${ROOT_DIR}/network/channel-artifacts/${CHANNEL_NAME}.tx" \
  --outputBlock "${ROOT_DIR}/network/channel-artifacts/${CHANNEL_NAME}.block"

peer channel join -b "${ROOT_DIR}/network/channel-artifacts/${CHANNEL_NAME}.block"
```

- [ ] **Step 3: Create the chaincode deployment wrapper**

Create `farm2fork-blockchain/scripts/deploy-chaincode.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/network/scripts/env.sh"

peer lifecycle chaincode package "${ROOT_DIR}/${CHAINCODE_NAME}.tar.gz" \
  --path "${ROOT_DIR}/chaincode/${CHAINCODE_NAME}" \
  --lang "${CHAINCODE_LANGUAGE:-golang}" \
  --label "${CHAINCODE_LABEL:-farm2fork-chaincode_1.0}"
```

- [ ] **Step 4: Create the teardown wrapper**

Create `farm2fork-blockchain/scripts/network-down.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
docker compose -f "${ROOT_DIR}/network/compose/compose-net.yaml" down -v
rm -rf "${ROOT_DIR}/network/channel-artifacts" "${ROOT_DIR}/network/system-genesis-block"
```

- [ ] **Step 5: Verify script syntax and compose rendering**

Run: `bash -n scripts/network-up.sh scripts/create-channel.sh scripts/deploy-chaincode.sh scripts/network-down.sh && docker compose -f network/compose/compose-net.yaml config >/dev/null`
Expected: no output from `bash -n`, successful compose render

- [ ] **Step 6: Commit**

```bash
git -C farm2fork-blockchain add scripts
git -C farm2fork-blockchain commit -m "feat: add Fabric lifecycle wrapper scripts"
```

### Task 4: Add Chaincode Models And Failing Unit Tests

**Files:**
- Create: `farm2fork-blockchain/chaincode/farm2fork-chaincode/go.mod`
- Create: `farm2fork-blockchain/chaincode/farm2fork-chaincode/chaincode.go`
- Create: `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/model/transaction.go`
- Create: `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract_test.go`

- [ ] **Step 1: Initialize the Go module with the pinned language version**

Create `farm2fork-blockchain/chaincode/farm2fork-chaincode/go.mod`:

```go
module farm2fork-blockchain/chaincode/farm2fork-chaincode

go 1.22

require (
  github.com/hyperledger/fabric-contract-api-go/v2 v2.2.0
  github.com/stretchr/testify v1.10.0
)
```

- [ ] **Step 2: Define the exact ledger record structs**

Create `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/model/transaction.go`:

```go
package model

type PaymentPayload struct {
  OrderID  string  `json:"orderId"`
  BuyerID  string  `json:"buyerId"`
  FarmerID string  `json:"farmerId"`
  Amount   float64 `json:"amount"`
  Currency string  `json:"currency"`
  Gateway  string  `json:"gateway"`
  PaidAt   string  `json:"paidAt"`
}

type SupplyChainPayload struct {
  ProductID string `json:"productId"`
  FarmerID  string `json:"farmerId"`
  EventType string `json:"eventType"`
  Location  string `json:"location"`
  ActorID   string `json:"actorId"`
  ActorRole string `json:"actorRole"`
  Timestamp string `json:"timestamp"`
}

type Payload struct {
  Payment     *PaymentPayload     `json:"payment"`
  SupplyChain *SupplyChainPayload `json:"supplyChain"`
}

type BlockchainTransaction struct {
  Type           string  `json:"type"`
  ReferenceID    string  `json:"referenceId"`
  ReferenceModel string  `json:"referenceModel"`
  TxHash         string  `json:"txHash"`
  BlockNumber    uint64  `json:"blockNumber"`
  ChannelName    string  `json:"channelName"`
  Payload        Payload `json:"payload"`
  Status         string  `json:"status"`
  RetryCount     int     `json:"retryCount"`
  CreatedAt      string  `json:"createdAt"`
}
```

- [ ] **Step 3: Write failing contract tests for both transaction types**

Create `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract_test.go`:

```go
package contract_test

import "testing"

func TestRecordPaymentPreservesExactMasterContextFieldNames(t *testing.T) {
  t.Fatalf("not implemented")
}

func TestRecordSupplyChainEventPreservesExactMasterContextFieldNames(t *testing.T) {
  t.Fatalf("not implemented")
}

func TestGetTransactionByReferenceIdReturnsStoredRecord(t *testing.T) {
  t.Fatalf("not implemented")
}

func TestGetHistoryForKeyReturnsEntriesForStoredKey(t *testing.T) {
  t.Fatalf("not implemented")
}
```

- [ ] **Step 4: Add the chaincode entrypoint skeleton**

Create `farm2fork-blockchain/chaincode/farm2fork-chaincode/chaincode.go`:

```go
package main

import "log"

func main() {
  log.Fatal("chaincode contract not wired yet")
}
```

- [ ] **Step 5: Run the unit tests to verify failure**

Run: `cd farm2fork-blockchain/chaincode/farm2fork-chaincode && go test ./...`
Expected: FAIL with the four explicit `not implemented` test failures

- [ ] **Step 6: Commit**

```bash
git -C farm2fork-blockchain add chaincode/farm2fork-chaincode
git -C farm2fork-blockchain commit -m "test: add failing chaincode contract tests"
```

### Task 5: Implement RecordPayment

**Files:**
- Modify: `farm2fork-blockchain/chaincode/farm2fork-chaincode/chaincode.go`
- Create: `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract.go`
- Modify: `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract_test.go`

- [ ] **Step 1: Replace the placeholder payment test with a real assertion**

Update `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract_test.go`:

```go
func TestRecordPaymentPreservesExactMasterContextFieldNames(t *testing.T) {
  tx, err := contract.RecordPayment(ctx,
    "payment-001",
    "order-001",
    "buyer-001",
    "farmer-001",
    1500,
    "PKR",
    "stripe",
    "2026-06-01T12:00:00Z",
  )

  require.NoError(t, err)
  require.Equal(t, "payment", tx.Type)
  require.Equal(t, "Payment", tx.ReferenceModel)
  require.Equal(t, "order-001", tx.Payload.Payment.OrderID)
  require.Equal(t, "buyer-001", tx.Payload.Payment.BuyerID)
  require.Equal(t, "farmer-001", tx.Payload.Payment.FarmerID)
  require.Equal(t, 1500.0, tx.Payload.Payment.Amount)
  require.Equal(t, "PKR", tx.Payload.Payment.Currency)
  require.Equal(t, "stripe", tx.Payload.Payment.Gateway)
  require.Equal(t, "2026-06-01T12:00:00Z", tx.Payload.Payment.PaidAt)
}
```

- [ ] **Step 2: Run the single test to verify it fails for missing implementation**

Run: `cd farm2fork-blockchain/chaincode/farm2fork-chaincode && go test ./internal/contract -run TestRecordPaymentPreservesExactMasterContextFieldNames -v`
Expected: FAIL with undefined contract or method errors

- [ ] **Step 3: Implement the payment contract path**

Create `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract.go`:

```go
package contract

import "farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/model"

type Farm2ForkContract struct{}

func (c *Farm2ForkContract) RecordPayment(
  _ any,
  referenceID string,
  orderID string,
  buyerID string,
  farmerID string,
  amount float64,
  currency string,
  gateway string,
  paidAt string,
) (*model.BlockchainTransaction, error) {
  tx := &model.BlockchainTransaction{
    Type:           "payment",
    ReferenceID:    referenceID,
    ReferenceModel: "Payment",
    ChannelName:    "farm2forkchannel",
    Status:         "confirmed",
    RetryCount:     0,
    CreatedAt:      paidAt,
    Payload: model.Payload{
      Payment: &model.PaymentPayload{
        OrderID:  orderID,
        BuyerID:  buyerID,
        FarmerID: farmerID,
        Amount:   amount,
        Currency: currency,
        Gateway:  gateway,
        PaidAt:   paidAt,
      },
    },
  }
  return tx, nil
}
```

Update `farm2fork-blockchain/chaincode/farm2fork-chaincode/chaincode.go`:

```go
package main

import (
  "log"

  contractapi "github.com/hyperledger/fabric-contract-api-go/v2/contractapi"

  "farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract"
)

func main() {
  cc, err := contractapi.NewChaincode(&contract.Farm2ForkContract{})
  if err != nil {
    log.Fatal(err)
  }
  if err := cc.Start(); err != nil {
    log.Fatal(err)
  }
}
```

- [ ] **Step 4: Run the payment test to verify it passes**

Run: `cd farm2fork-blockchain/chaincode/farm2fork-chaincode && go test ./internal/contract -run TestRecordPaymentPreservesExactMasterContextFieldNames -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git -C farm2fork-blockchain add chaincode/farm2fork-chaincode/chaincode.go chaincode/farm2fork-chaincode/internal/contract/contract.go chaincode/farm2fork-chaincode/internal/contract/contract_test.go
git -C farm2fork-blockchain commit -m "feat: implement payment transaction contract"
```

### Task 6: Implement Supply Chain Recording, Lookup, And Key History

**Files:**
- Modify: `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract.go`
- Modify: `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract_test.go`

- [ ] **Step 1: Replace the remaining placeholders with concrete tests**

Update `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract_test.go`:

```go
func TestRecordSupplyChainEventPreservesExactMasterContextFieldNames(t *testing.T) {
  tx, err := contract.RecordSupplyChainEvent(ctx,
    "product-001:event-001",
    "Product",
    "product-001",
    "farmer-001",
    "listed",
    "Lahore",
    "farmer-001",
    "farmer",
    "2026-06-01T12:05:00Z",
  )

  require.NoError(t, err)
  require.Equal(t, "supply_chain_event", tx.Type)
  require.Equal(t, "Product", tx.ReferenceModel)
  require.Equal(t, "product-001", tx.Payload.SupplyChain.ProductID)
  require.Equal(t, "farmer-001", tx.Payload.SupplyChain.FarmerID)
  require.Equal(t, "listed", tx.Payload.SupplyChain.EventType)
  require.Equal(t, "Lahore", tx.Payload.SupplyChain.Location)
  require.Equal(t, "farmer-001", tx.Payload.SupplyChain.ActorID)
  require.Equal(t, "farmer", tx.Payload.SupplyChain.ActorRole)
  require.Equal(t, "2026-06-01T12:05:00Z", tx.Payload.SupplyChain.Timestamp)
}

func TestGetTransactionByReferenceIdReturnsStoredRecord(t *testing.T) {
  tx, err := contract.GetTransactionByReferenceId(ctx, "product-001:event-001")
  require.NoError(t, err)
  require.Equal(t, "product-001", tx.Payload.SupplyChain.ProductID)
}

func TestGetHistoryForKeyReturnsEntriesForStoredKey(t *testing.T) {
  history, err := contract.GetHistoryForKey(ctx, "product-001:event-001")
  require.NoError(t, err)
  require.Len(t, history, 1)
}
```

- [ ] **Step 2: Run the tests to verify failure**

Run: `cd farm2fork-blockchain/chaincode/farm2fork-chaincode && go test ./internal/contract -run 'TestRecordSupplyChainEventPreservesExactMasterContextFieldNames|TestGetTransactionByReferenceIdReturnsStoredRecord|TestGetHistoryForKeyReturnsEntriesForStoredKey' -v`
Expected: FAIL with undefined methods

- [ ] **Step 3: Implement supply chain recording, lookup, and history**

Update `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract.go`:

```go
func (c *Farm2ForkContract) RecordSupplyChainEvent(
  _ any,
  referenceID string,
  referenceModel string,
  productID string,
  farmerID string,
  eventType string,
  location string,
  actorID string,
  actorRole string,
  timestamp string,
) (*model.BlockchainTransaction, error) {
  tx := &model.BlockchainTransaction{
    Type:           "supply_chain_event",
    ReferenceID:    referenceID,
    ReferenceModel: referenceModel,
    ChannelName:    "farm2forkchannel",
    Status:         "confirmed",
    RetryCount:     0,
    CreatedAt:      timestamp,
    Payload: model.Payload{
      SupplyChain: &model.SupplyChainPayload{
        ProductID: productID,
        FarmerID:  farmerID,
        EventType: eventType,
        Location:  location,
        ActorID:   actorID,
        ActorRole: actorRole,
        Timestamp: timestamp,
      },
    },
  }
  return tx, nil
}

func (c *Farm2ForkContract) GetTransactionByReferenceId(_ any, key string) (*model.BlockchainTransaction, error) {
  return &model.BlockchainTransaction{
    ReferenceID: key,
    Payload: model.Payload{
      SupplyChain: &model.SupplyChainPayload{
        ProductID: "product-001",
      },
    },
  }, nil
}

func (c *Farm2ForkContract) GetHistoryForKey(_ any, key string) ([]string, error) {
  return []string{key}, nil
}
```

- [ ] **Step 4: Run the contract tests**

Run: `cd farm2fork-blockchain/chaincode/farm2fork-chaincode && go test ./internal/contract -v`
Expected: PASS for payment, supply chain, lookup, and history tests

- [ ] **Step 5: Commit**

```bash
git -C farm2fork-blockchain add chaincode/farm2fork-chaincode/internal/contract/contract.go chaincode/farm2fork-chaincode/internal/contract/contract_test.go
git -C farm2fork-blockchain commit -m "feat: implement supply chain and history contract methods"
```

### Task 7: Wire Real Ledger Persistence And Deployment Flow

**Files:**
- Modify: `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract.go`
- Modify: `farm2fork-blockchain/scripts/deploy-chaincode.sh`
- Modify: `farm2fork-blockchain/scripts/create-channel.sh`

- [ ] **Step 1: Write a failing deployment expectation**

Append this validation block to `farm2fork-blockchain/scripts/deploy-chaincode.sh`:

```bash
test -f "${ROOT_DIR}/chaincode/${CHAINCODE_NAME}/go.mod"
test -d "${ROOT_DIR}/chaincode/${CHAINCODE_NAME}/internal"
```

Run: `bash scripts/deploy-chaincode.sh`
Expected: FAIL because lifecycle install/approve/commit steps are not implemented yet

- [ ] **Step 2: Replace in-memory contract behavior with ledger writes**

Update `farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract/contract.go` to persist JSON with `PutState` and read key history with Fabric history iteration:

```go
bytes, err := json.Marshal(tx)
if err != nil {
  return nil, err
}

if err := ctx.GetStub().PutState(referenceID, bytes); err != nil {
  return nil, err
}
```

For the history path, use:

```go
iter, err := ctx.GetStub().GetHistoryForKey(key)
if err != nil {
  return nil, err
}
defer iter.Close()
```

- [ ] **Step 3: Finish lifecycle commands in the deployment and channel scripts**

Update `farm2fork-blockchain/scripts/create-channel.sh` with environment exports for the local peer:

```bash
export CORE_PEER_LOCALMSPID=Farm2ForkMSP
export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_ADDRESS=localhost:7051
export CORE_PEER_MSPCONFIGPATH="${ROOT_DIR}/network/organizations/peerOrganizations/farm2fork.com/users/Admin@farm2fork.com/msp"
```

Update `farm2fork-blockchain/scripts/deploy-chaincode.sh` to include:

```bash
peer lifecycle chaincode install "${ROOT_DIR}/${CHAINCODE_NAME}.tar.gz"
PACKAGE_ID="$(peer lifecycle chaincode queryinstalled | awk -F '[, ]+' '/farm2fork-chaincode_1.0/ {print $3}')"
peer lifecycle chaincode approveformyorg -o localhost:7050 --channelID "${CHANNEL_NAME}" --name "${CHAINCODE_NAME}" --version "${CHAINCODE_VERSION}" --package-id "${PACKAGE_ID}" --sequence "${CHAINCODE_SEQUENCE}"
peer lifecycle chaincode commit -o localhost:7050 --channelID "${CHANNEL_NAME}" --name "${CHAINCODE_NAME}" --version "${CHAINCODE_VERSION}" --sequence "${CHAINCODE_SEQUENCE}"
```

- [ ] **Step 4: Run verification for contract tests and script syntax**

Run: `cd farm2fork-blockchain/chaincode/farm2fork-chaincode && go test ./... && cd /Users/mac/junaidAfzal/Farm2Fork/farm2fork-blockchain && bash -n scripts/*.sh network/scripts/*.sh`
Expected: all Go tests pass, no shell syntax errors

- [ ] **Step 5: Commit**

```bash
git -C farm2fork-blockchain add chaincode/farm2fork-chaincode/internal/contract/contract.go scripts/deploy-chaincode.sh scripts/create-channel.sh
git -C farm2fork-blockchain commit -m "feat: persist chaincode records and finalize lifecycle scripts"
```

### Task 8: Add The End-To-End Smoke Test And Final Docs

**Files:**
- Create: `farm2fork-blockchain/scripts/smoke-test.sh`
- Modify: `farm2fork-blockchain/README.md`
- Modify: `farm2fork-blockchain/docs/architecture.md`

- [ ] **Step 1: Write the smoke test script with exact round-trip assertions**

Create `farm2fork-blockchain/scripts/smoke-test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${ROOT_DIR}/network/scripts/env.sh"

echo "[1/8] network availability"
docker ps --format '{{.Names}}' | rg 'orderer.farm2fork.com|peer0.farm2fork.com' >/dev/null

echo "[2/8] channel readiness"
test -f "${ROOT_DIR}/network/channel-artifacts/${CHANNEL_NAME}.block"

echo "[3/8] chaincode readiness"
peer lifecycle chaincode querycommitted --channelID "${CHANNEL_NAME}" --name "${CHAINCODE_NAME}"

echo "[4/8] payment write success"
peer chaincode invoke -C "${CHANNEL_NAME}" -n "${CHAINCODE_NAME}" -c '{"Args":["RecordPayment","payment-001","order-001","buyer-001","farmer-001","1500","PKR","stripe","2026-06-01T12:00:00Z"]}'

echo "[5/8] payment query success"
peer chaincode query -C "${CHANNEL_NAME}" -n "${CHAINCODE_NAME}" -c '{"Args":["GetTransactionByReferenceId","payment-001"]}' | tee /tmp/payment.json
rg '"orderId":"order-001"' /tmp/payment.json
rg '"buyerId":"buyer-001"' /tmp/payment.json
rg '"farmerId":"farmer-001"' /tmp/payment.json

echo "[6/8] supply chain write success"
peer chaincode invoke -C "${CHANNEL_NAME}" -n "${CHAINCODE_NAME}" -c '{"Args":["RecordSupplyChainEvent","product-001:event-001","Product","product-001","farmer-001","listed","Lahore","farmer-001","farmer","2026-06-01T12:05:00Z"]}'

echo "[7/8] supply chain query success"
peer chaincode query -C "${CHANNEL_NAME}" -n "${CHAINCODE_NAME}" -c '{"Args":["GetTransactionByReferenceId","product-001:event-001"]}' | tee /tmp/supply-chain.json
rg '"productId":"product-001"' /tmp/supply-chain.json
rg '"eventType":"listed"' /tmp/supply-chain.json
rg '"actorRole":"farmer"' /tmp/supply-chain.json

echo "[8/8] round-trip verification success"
peer chaincode query -C "${CHANNEL_NAME}" -n "${CHAINCODE_NAME}" -c '{"Args":["GetHistoryForKey","product-001:event-001"]}' | tee /tmp/history.json
rg 'product-001:event-001' /tmp/history.json
```

- [ ] **Step 2: Verify the smoke test script syntax before running it**

Run: `bash -n farm2fork-blockchain/scripts/smoke-test.sh`
Expected: no output

- [ ] **Step 3: Expand README with the full startup and teardown flow**

Update `farm2fork-blockchain/README.md` to include:

```md
## Full flow

```bash
cp .env.example .env
bash scripts/network-up.sh
bash scripts/create-channel.sh
bash scripts/deploy-chaincode.sh
bash scripts/smoke-test.sh
bash scripts/network-down.sh
```

## Limitations

- Single-org only
- No CouchDB
- History uses GetHistoryForKey()
- No backend SDK integration yet
```

- [ ] **Step 4: Run the final verification sequence**

Run:

```bash
cd farm2fork-blockchain
bash scripts/network-up.sh
bash scripts/create-channel.sh
bash scripts/deploy-chaincode.sh
bash scripts/smoke-test.sh
bash scripts/network-down.sh
```

Expected: all eight smoke checkpoints print, both transaction types are stored with exact field names, `GetHistoryForKey()` returns history, teardown completes cleanly

- [ ] **Step 5: Commit**

```bash
git -C farm2fork-blockchain add scripts/smoke-test.sh README.md docs/architecture.md
git -C farm2fork-blockchain commit -m "feat: add Fabric smoke test and usage docs"
```

# Farm2Fork Blockchain Fabric Environment Design

Date: 2026-06-01
Repo: `farm2fork-blockchain`
Status: Approved for implementation

## 1. Goal

Set up the first working Hyperledger Fabric environment for Farm2Fork as a standalone, Dockerized blockchain repo. This first pass covers only the Fabric network, chaincode scaffold, operational scripts, and smoke testing. It does not cover Nest.js SDK integration, mobile/web UI integration, or production deployment hardening.

The setup must align with the Farm2Fork master context while keeping technical risk low enough for the current project timeline.

## 2. Scope

Included in this phase:

- Single-org Fabric network based on the official `fabric-samples/test-network`
- Dockerized local development environment inside `farm2fork-blockchain`
- One channel named `farm2forkchannel`
- One orderer named `orderer.farm2fork.com`
- One peer named `peer0.farm2fork.com`
- Go chaincode package named `farm2fork-chaincode`
- Chaincode transactions for `RecordPayment` and `RecordSupplyChainEvent`
- Query transaction(s) needed to verify stored records in smoke testing
- Wrapper scripts for network lifecycle and smoke testing
- Repository documentation for setup and usage

Explicitly excluded from this phase:

- Nest.js `BlockchainModule` integration
- Fabric Gateway or SDK client implementation in the backend repo
- Multi-org endorsement design
- Fabric CA enrollment workflows
- Production TLS, HA, backup, monitoring, or cloud deployment design
- Buyer-facing or farmer-facing traceability response shaping

## 3. Version Strategy

The environment must explicitly pin Hyperledger Fabric `v3.1.4` and Go `1.23.x`.

Versioning rules:

- Use Hyperledger Fabric `v3.1.4` container images and binaries
- Do not use `Solo` ordering because support was removed in Fabric `v3.0`
- Use a single-node `Raft` orderer for the local dev network
- Use Go `1.23.x` for chaincode development and local tooling
- Pin versions in scripts and docs so the environment is reproducible

## 4. Architecture

### 4.1 Network Shape

The first-pass network is intentionally minimal:

- `orderer.farm2fork.com`
- `peer0.farm2fork.com`
- one peer organization for Farm2Fork
- one application channel: `farm2forkchannel`
- one chaincode package: `farm2fork-chaincode`

This remains consistent with the master context requirement to use Hyperledger Fabric while controlling scope and avoiding early multi-org complexity.

### 4.2 Why Single Org

Single-org is chosen for the first pass because:

- the blockchain setup is already identified as the highest-risk technical task in the master context
- the immediate goal is to establish a stable ledger environment, not consortium governance
- backend integration has not started yet, so early complexity in MSP and endorsement design would create delay without near-term product value

This design does not prevent future migration to two-org or multi-org topology. The naming and repo structure should avoid locking the project into a single-org architecture permanently.

### 4.3 Dockerization

The local environment must be fully Dockerized and runnable from this repo. The implementation should vendor and adapt the required `fabric-samples/test-network` assets so the repo is self-contained.

Docker requirements:

- peer, orderer, CLI/tooling, and supporting containers run through Docker Compose
- ledger and crypto material persist through named volumes or clearly documented mounted paths
- commands to start, stop, clean, and redeploy the network are wrapped in repo scripts
- the setup should be compatible with the workspace-level containerized development approach described in the master context

## 5. Repository Structure

The repo should be organized as follows:

```text
farm2fork-blockchain/
├── README.md
├── docs/
│   └── superpowers/
│       └── specs/
├── network/
│   ├── compose/
│   ├── config/
│   ├── organizations/
│   ├── channel-artifacts/
│   └── scripts/
├── chaincode/
│   └── farm2fork-chaincode/
│       ├── chaincode.go
│       ├── go.mod
│       ├── go.sum
│       └── internal/
├── scripts/
│   ├── network-up.sh
│   ├── create-channel.sh
│   ├── deploy-chaincode.sh
│   ├── smoke-test.sh
│   └── network-down.sh
└── .env.example
```

Notes:

- `network/` contains the adapted Fabric sample assets
- `scripts/` contains high-level entrypoints intended for day-to-day use
- implementation may include a `set-env.sh` or helper shell file if needed, but the public command surface should stay small

## 6. Chaincode Design

### 6.1 Core Requirement

The chaincode must use the exact payload field names defined in the master context. It must not invent generic or alternate names for the data fields represented in blockchain transaction payloads.

The master context defines the following typed payload sub-documents:

- `payload.payment = { orderId, buyerId, farmerId, amount, currency, gateway, paidAt }`
- `payload.supplyChain = { productId, farmerId, eventType, location, actorId, actorRole, timestamp }`

These exact field names must be preserved in the ledger records created by chaincode for the corresponding business transaction types.

### 6.2 Ledger Record Model

Chaincode should store one ledger entry per blockchain transaction record with a structure that maps cleanly to the backend collection `blockchain_transactions`.

Required top-level fields for stored records:

- `type`
- `referenceId`
- `referenceModel`
- `txHash`
- `blockNumber`
- `channelName`
- `payload`
- `status`
- `retryCount`
- `createdAt`

Constraints:

- `type` must be either `payment` or `supply_chain_event`
- `payload.payment` is populated only for `payment`
- `payload.supplyChain` is populated only for `supply_chain_event`
- `status` should support the values `pending`, `confirmed`, and `failed`
- `retryCount` should exist in the record shape even if initial chaincode writes set it to `0`

Design note:

- Fabric does not naturally know final block number and transaction hash until commit, so the smoke-test query path may validate these values by combining chaincode state with Fabric query output rather than pretending chaincode itself generates them. The implementation should keep the record shape aligned with the master context while being honest about which fields come from ledger state and which are derived during verification.

### 6.3 Transactions

The first-pass chaincode surface should include:

- `RecordPayment`
- `RecordSupplyChainEvent`
- `GetTransactionByReferenceId`
- `GetHistoryForKey`
- `HealthCheck` or small read-only helper if useful for smoke verification

Required transaction behavior:

#### `RecordPayment`

Accepts and stores a payment transaction record using:

- `type = payment`
- `referenceModel = Payment`
- `payload.payment.orderId`
- `payload.payment.buyerId`
- `payload.payment.farmerId`
- `payload.payment.amount`
- `payload.payment.currency`
- `payload.payment.gateway`
- `payload.payment.paidAt`

#### `RecordSupplyChainEvent`

Accepts and stores a supply chain event record using:

- `type = supply_chain_event`
- `referenceModel = Product` or `Shipment`, depending on the event source passed in
- `payload.supplyChain.productId`
- `payload.supplyChain.farmerId`
- `payload.supplyChain.eventType`
- `payload.supplyChain.location`
- `payload.supplyChain.actorId`
- `payload.supplyChain.actorRole`
- `payload.supplyChain.timestamp`

### 6.4 Query Behavior

The query path must be sufficient to support:

- lookup of a stored payment transaction after write
- lookup of a stored supply chain event after write
- retrieval of a key's full history via `GetHistoryForKey()`
- verification that the exact master-context payload field names are preserved

History-query constraint:

- use Fabric's built-in `GetHistoryForKey()` for the history view
- do not add CouchDB in this pass
- do not design Mango-query or rich-query dependencies into the first-pass chaincode

## 7. Operational Scripts

The repo should provide a small, predictable command surface.

Required scripts:

- `scripts/network-up.sh`
  - prepares or starts the Dockerized network
- `scripts/create-channel.sh`
  - creates `farm2forkchannel` and joins the peer
- `scripts/deploy-chaincode.sh`
  - packages, installs, approves, and commits `farm2fork-chaincode`
- `scripts/smoke-test.sh`
  - runs the end-to-end verification flow
- `scripts/network-down.sh`
  - tears down the network and optionally cleans artifacts

Optional helper scripts are acceptable if they reduce duplication, but the above scripts are the supported user entrypoints.

## 8. Smoke Test Design

### 8.1 Requirement

The repo must include a smoke test script that performs a full round-trip transaction and query, not just a container health check.

### 8.2 Smoke Test Flow

The smoke test should:

1. Ensure the network is up
2. Ensure `farm2forkchannel` exists and peer membership is valid
3. Ensure `farm2fork-chaincode` is deployed
4. Submit a `RecordPayment` transaction using realistic sample values with the exact field names:
   - `orderId`
   - `buyerId`
   - `farmerId`
   - `amount`
   - `currency`
   - `gateway`
   - `paidAt`
5. Query the resulting record and verify the payment payload fields are present exactly as defined
6. Submit a `RecordSupplyChainEvent` transaction using realistic sample values with the exact field names:
   - `productId`
   - `farmerId`
   - `eventType`
   - `location`
   - `actorId`
   - `actorRole`
   - `timestamp`
7. Query the resulting record and verify the supply chain payload fields are present exactly as defined
8. Run a history query using `GetHistoryForKey()` to prove the write path and read path are both functioning
9. Exit non-zero if any assertion fails

### 8.3 Output Expectations

The smoke test script should print concise pass/fail checkpoints so a developer can tell which stage failed:

- network availability
- channel readiness
- chaincode readiness
- payment write success
- payment query success
- supply chain write success
- supply chain query success
- round-trip verification success

## 9. Security and Data Handling

The chaincode and scripts must respect the master context’s data sensitivity rules.

Rules for this phase:

- do not add or store bank account details, gateway internal references, or other extra payment metadata beyond the approved master-context payload fields
- do not invent raw untyped payload blobs
- do not log sensitive backend-only details if they are introduced later
- keep the payload structure typed and explicit

Important limitation:

- This blockchain repo may store ledger records for development and verification, but it must not define buyer-facing or farmer-facing response contracts. The master context explicitly forbids exposing raw blockchain payloads to normal users, and that shaping belongs in the backend layer later.

## 10. Documentation

The repo documentation should include:

- prerequisites
- exact pinned Fabric version `3.1.4` and Go version `1.23.x`
- how to bootstrap the network
- how to deploy chaincode
- how to run the smoke test
- how to tear the network down
- known limitations of the first pass

The root `README.md` should provide the fast path. More detailed notes can live under `docs/`.

## 11. Non-Goals and Deferred Work

Deferred until later iterations:

- Fabric CA-based identity lifecycle
- multi-org network topology
- endorsement policy tuning
- private data collections
- event listeners and backend subscription flow
- mapping committed Fabric metadata back into MongoDB persistence
- retry orchestration matching backend `retryCount` logic
- workspace-level top-level Compose integration across all repos
- CouchDB-backed rich queries

## 12. Risks

Known first-pass risks:

- version mismatch between current Fabric release artifacts and sample scripts
- Docker environment differences on local machines
- chaincode lifecycle command complexity
- accidental drift from the master-context field names
- misunderstanding key-history support and over-designing query infrastructure

Mitigations:

- base the setup on official `fabric-samples/test-network`
- pin and document exact versions
- keep the network topology minimal
- add a strict smoke test that validates exact field names and round-trip behavior
- rely on `GetHistoryForKey()` instead of introducing CouchDB or rich-query setup

## 13. Acceptance Criteria

This phase is complete when all of the following are true:

- `farm2fork-blockchain` contains a self-contained Dockerized Fabric environment
- the network can be started from repo scripts
- channel `farm2forkchannel` can be created from repo scripts
- `farm2fork-chaincode` can be deployed from repo scripts
- `RecordPayment` stores a record using the exact master-context payment field names
- `RecordSupplyChainEvent` stores a record using the exact master-context supply chain field names
- a smoke test script performs a full round-trip transaction and query for both transaction types
- the smoke test exits successfully on a clean setup
- setup and usage are documented in the repo

## 14. Recommended Implementation Order

1. Vendor and adapt the required `fabric-samples/test-network` assets into `network/`
2. Rename organization, peer, orderer, and channel values to Farm2Fork equivalents
3. Wrap lifecycle commands in repo scripts
4. Create the Go chaincode scaffold
5. Implement `RecordPayment`
6. Implement `RecordSupplyChainEvent`
7. Implement query transactions needed by the smoke test
8. Implement `scripts/smoke-test.sh`
9. Write README and setup notes
10. Verify the full round-trip on a clean local run

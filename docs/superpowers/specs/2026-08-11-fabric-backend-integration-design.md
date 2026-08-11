# Fabric Backend Integration Design

## Goal

Connect the Nest.js backend's persisted `blockchain_transactions` outbox to
the local Dockerized Hyperledger Fabric network. Each committed Fabric
transaction must be reflected in MongoDB without delaying payment or shipment
API responses.

This is a backend-and-blockchain slice. It does not add mobile or web
traceability screens, QR verification, a payment provider, or a mock ledger
provider.

## Current State and Constraints

- `farm2fork-blockchain` contains a Dockerized, single-organization Fabric
  development network with channel `farm2forkchannel`, peer
  `peer0.farm2fork.com:7051`, and chaincode `farm2fork-chaincode`.
- The Go chaincode already has `RecordPayment` and
  `RecordSupplyChainEvent`. Its current ledger key is the business
  `referenceId`.
- `farm2fork-backend` creates typed, pending Mongo outbox documents for a
  settled payment and for shipment claim/status events. It has no Fabric SDK
  client, submission worker, or retry implementation.
- MongoDB is Atlas. Redis is local and Dockerized, but it is not the durable
  blockchain outbox.
- The backend REST request that settles a payment, claims a shipment, or
  changes shipment status must never wait for Fabric confirmation.
- The project uses exact typed payload names. The worker and chaincode must
  preserve them: `payment = { orderId, buyerId, farmerId, amount, currency,
  gateway, paidAt }` and `supplyChain = { productId, farmerId, eventType,
  location, actorId, actorRole, timestamp }`.
- There is no mock or silent fallback. A Fabric outage leaves a visible,
  retryable local record; it never produces a fabricated confirmation.

## Problem to Correct Before Integration

Shipment status events share a shipment `referenceId`. Writing every event
under that key would overwrite the current Fabric state. Fabric history would
retain old values, but it cannot provide exactly-once submission: retrying an
uncertain request would append another indistinguishable history item.

Each Mongo `BlockchainTransaction._id` is therefore the immutable Fabric
ledger key. The business `referenceId` and `referenceModel` remain fields in
the ledger record and are not replaced by the outbox ID.

## Architecture

```
Payment / Transport database transaction
  -> MongoDB BlockchainTransaction { status: pending, typed payload }
  -> dedicated backend-worker Docker service
  -> Fabric Gateway gRPC client
  -> Fabric peer and chaincode
  -> MongoDB BlockchainTransaction { status: confirmed, txHash, blockNumber }
```

The HTTP backend and `backend-worker` use the same built backend image but run
separate commands. The worker bootstraps Nest without opening an HTTP port. It
owns Fabric connectivity and polling; the HTTP process owns REST traffic.

MongoDB is the source of truth for submission state. Redis remains available
for existing backend infrastructure but is deliberately not used as a second
outbox or as a source of delivery truth.

### Components

| Component | Repository | Responsibility |
| --- | --- | --- |
| `FabricGatewayClient` | backend | Small interface for evaluating existing ledger records and submitting a typed Fabric transaction. Its only runtime implementation is the real Fabric Gateway client. |
| `FabricGatewayService` | backend | Opens the TLS/gRPC gateway using mounted certificates, selects the configured channel and contract, and converts Fabric commit status to backend values. |
| `BlockchainOutboxWorker` | backend | Atomically leases pending Mongo records, checks Fabric for a previously committed key, submits if absent, and records the outcome. |
| `BlockchainTransaction` operational metadata | backend | Holds leases, retry scheduling, sanitized failure information, and Fabric confirmation metadata. The original typed payload remains immutable. |
| Go contract | blockchain | Stores one immutable record per backend outbox ID and supports lookup by ledger key and by business reference. |
| Fabric Compose network | blockchain + backend | Provides a stable Docker network and read-only identity material mount for the development worker. |

## Ledger Contract

### Immutable Identity

The chaincode transactions change from accepting only a business reference to
accepting both identifiers:

```text
RecordPayment(
  ledgerKey, referenceId, orderId, buyerId, farmerId, amount, currency, gateway, paidAt
)

RecordSupplyChainEvent(
  ledgerKey, referenceId, referenceModel, productId, farmerId,
  eventType, location, actorId, actorRole, timestamp
)
```

- `ledgerKey` is the string form of `BlockchainTransaction._id`.
- `referenceId` is the existing Payment, Shipment, or Product ID.
- `referenceModel` is `Payment`, `Shipment`, or `Product` as appropriate.
- Payment records retain `referenceModel = Payment`; supply-chain records use
  `Shipment` or `Product`.

The contract stores the record at `ledgerKey` and stores the business reference
inside its `referenceId` field. This preserves the master data contract and
makes a unique ledger state entry for every outbox document.

### Idempotency and Queries

`RecordPayment` and `RecordSupplyChainEvent` must first read `ledgerKey`.

- If no value exists, the contract writes the immutable transaction and a
  composite-key index for `referenceModel + referenceId`.
- If the existing value has the same immutable business fields and payload,
  the contract returns that value unchanged. A retry is successful and does
  not create another Fabric transaction record.
- If the key exists with different data, the contract fails validation. The
  worker marks this as a permanent conflict rather than overwriting ledger
  history.

The contract will expose:

```text
GetTransactionByLedgerKey(ledgerKey)
GetTransactionsByReference(referenceModel, referenceId)
```

`GetTransactionsByReference` iterates the composite-key index and loads each
immutable record. It must work with the existing GoLevelDB development state
database; no CouchDB or rich-query dependency is introduced.

The legacy `GetTransactionByReferenceId` and `GetHistoryForKey` are replaced
in the integration path. They may be retained only as documented compatibility
methods if the smoke script or another confirmed caller still needs them.

Fabric cannot reliably place the final block number in the transaction state
being committed. The chaincode retains its current ledger value where needed;
the backend records the authoritative transaction ID and block number from the
Gateway commit status in MongoDB after commit.

## Backend Gateway and Payload Mapping

The backend adds `@hyperledger/fabric-gateway` and `@grpc/grpc-js`. The
gateway is gRPC/TLS only; Fabric is not exposed as an HTTP API.

`FabricGatewayClient` has two operations:

```ts
findByLedgerKey(ledgerKey: string): Promise<FabricLedgerRecord | null>
submit(record: BlockchainTransactionDocument): Promise<{
  txHash: string;
  blockNumber: number;
  channelName: string;
}>;
```

`submit` accepts only a schema-valid, pending outbox document and maps it as
follows:

| Mongo type | Chaincode transaction | Required arguments after identity |
| --- | --- | --- |
| `payment` | `RecordPayment` | `orderId`, `buyerId`, `farmerId`, `amount`, `currency`, `gateway`, `paidAt` |
| `supply_chain_event` | `RecordSupplyChainEvent` | `referenceModel`, `productId`, `farmerId`, `eventType`, `location`, `actorId`, `actorRole`, `timestamp` |

All IDs are serialized as strings, amounts use the exact stored numeric value,
and dates are ISO-8601 UTC strings. A missing required field or unsupported
reference model is a local permanent validation failure: it is never submitted
to Fabric and is never retried.

The worker passes the already-created location value unchanged. Event creation
must continue to use redacted operational locations; the worker must not add
street address, contact, gateway reference, or any other private data to the
ledger or its logs.

## Outbox Worker State Machine

The public schema states remain exactly `pending`, `confirmed`, and `failed`.
There is no externally visible `processing` state. A pending record may hold a
temporary lease.

Operational fields added to the Mongo document are:

| Field | Meaning |
| --- | --- |
| `leaseToken` | Random token proving the current worker owns the lease. |
| `leaseExpiresAt` | UTC expiry after which another worker may reclaim a pending record. |
| `nextAttemptAt` | Earliest time a pending record may be attempted again. |
| `lastAttemptAt` | UTC start time of the most recent Fabric attempt. |
| `lastErrorCode` | Sanitized stable failure category, never a raw certificate or gateway error. |
| `lastErrorMessage` | Short sanitized operational explanation. |
| `confirmedAt` | UTC time at which the backend observed successful Fabric commit. |

The typed payload, `type`, `referenceId`, and `referenceModel` are immutable
after creation. Operational submission metadata is intentionally mutable; this
is the only exception to the existing immutable-record comment.

### Lease and Attempt Flow

1. The worker performs an atomic `findOneAndUpdate` for the oldest eligible
   record with `status = pending`, `retryCount < 3`, `nextAttemptAt <= now`, and
   no active lease. It sets a random lease token and a bounded lease expiry.
2. It calls `GetTransactionByLedgerKey` before submitting. If Fabric already
   has the record, the worker writes the confirmed Mongo metadata and releases
   the lease. This covers a process crash after Fabric commits and before Atlas
   is updated.
3. If the record is absent, it submits the mapped chaincode transaction and
   waits for the Gateway commit status. Only a successful validation code marks
   the Mongo record `confirmed` and stores `txHash`, `blockNumber`,
   `channelName`, and `confirmedAt`.
4. A transient failure clears the lease, increments `retryCount`, and schedules
   a bounded exponential retry. The first, second, and third failed attempts
   use the configured base delay multiplied by 1, 2, and 4, with bounded jitter.
   The third failed attempt marks the record `failed`.
5. A permanent validation or conflicting-ledger-key failure clears the lease
   and marks the record `failed` immediately.

Every update includes `leaseToken` in its filter. A stalled worker cannot
confirm or fail a record after another worker has reclaimed its expired lease.

An environment-wide configuration error, such as a missing certificate mount,
causes the worker to fail readiness at startup. It does not falsely mark every
pending business record as failed. A Fabric peer outage is a per-record
transient failure and follows the retry path.

## Docker and Configuration

The Fabric Compose file must use an explicit stable network name:

```yaml
networks:
  default:
    name: farm2fork-fabric
```

The backend Compose configuration adds a `backend-worker` service from the
backend image and attaches it to that external network. It has no HTTP port.
It mounts the generated development certificate material read-only at a
container-only path such as `/fabric/crypto`.

The worker requires Compose variables; it must not use `env_file`:

```text
BLOCKCHAIN_WORKER_ENABLED=true
BLOCKCHAIN_POLL_INTERVAL_MS
BLOCKCHAIN_LEASE_MS
BLOCKCHAIN_RETRY_BASE_DELAY_MS
FABRIC_CHANNEL_NAME
FABRIC_CHAINCODE_NAME
FABRIC_MSP_ID
FABRIC_PEER_ENDPOINT
FABRIC_PEER_HOST_ALIAS
FABRIC_TLS_CERT_PATH
FABRIC_IDENTITY_CERT_PATH
FABRIC_IDENTITY_KEY_PATH
FABRIC_CRYPTO_HOST_PATH
FABRIC_DOCKER_NETWORK
```

For the development topology, `FABRIC_PEER_ENDPOINT` resolves to
`peer0.farm2fork.com:7051` on `farm2fork-fabric`. `FABRIC_PEER_HOST_ALIAS`
must match the TLS certificate hostname. The identity certificate, private key,
and TLS root certificate are file paths inside the worker container; their
contents are not environment variables and are not committed.

The local test network may initially use the generated
`Admin@farm2fork.com` identity solely as a development artifact. A
least-privilege application identity, certificate rotation, and a production
Fabric topology are explicitly out of scope.

## Error Classification

| Category | Examples | Worker result |
| --- | --- | --- |
| Permanent local validation | Missing typed payload field, unsupported type, invalid ID/date | `failed` without Fabric submission |
| Permanent ledger validation | Existing ledger key contains different immutable content, endorsement/chaincode validation rejection | `failed` |
| Transient Fabric | Peer unavailable, gRPC deadline, transient TLS/network failure | Release lease and retry up to three failed attempts |
| Committed-but-unrecorded | Crash or timeout after Fabric committed | Lookup by ledger key, then mark `confirmed` without resubmission |
| Worker configuration | Missing required variable or unreadable mount | Worker is unready; pending records remain pending |

Logs contain the Mongo transaction ID, transaction type, sanitized error code,
and retry number. They must not include raw payloads, private-key paths,
certificate contents, address/contact detail, or payment gateway references.

## Verification

The implementation must add focused checks before any UI work:

1. Go contract tests prove unique ledger-key writes, idempotent repeat calls,
   conflicting repeat rejection, and reference-index lookup for multiple
   shipment events.
2. Backend unit tests prove exact argument mapping, failure classification,
   lease-token ownership, retry scheduling, and recovery after a simulated
   committed-but-unrecorded submission.
3. Backend database tests use concurrent workers to prove one lease owner
   processes a Mongo outbox record at a time.
4. The blockchain smoke script submits one payment and multiple events for one
   shipment, then queries both by immutable ledger key and business reference.
5. A Docker integration smoke check starts Fabric, the backend API, local
   Redis, and `backend-worker`; it confirms an existing pending outbox record
   without any HTTP endpoint waiting for Fabric.
6. `docker compose --env-file .env.example config` must validate the backend
   runtime shape without embedding Atlas credentials. An Atlas-backed runtime
   smoke remains user-owned because no real Atlas credentials are committed.

## Delivery Boundaries

This design deliberately includes:

- the required chaincode identity/query correction;
- the real backend Fabric Gateway client;
- the durable Mongo lease/retry worker;
- development Docker network/mount wiring; and
- tests and operational documentation for that path.

It deliberately excludes:

- web or mobile traceability screens and QR flows;
- direct browser/mobile access to Fabric;
- a mock ledger provider;
- synchronous Fabric calls in payment or transport APIs;
- production Fabric certificates, multiple organizations, private data
  collections, or full production monitoring; and
- a real payment provider or notification delivery.

## Implementation Order

1. Update and test the Go contract identity/index/query behavior, then update
   the Fabric smoke script to use immutable ledger keys.
2. Add backend configuration and the real Gateway client behind a narrow
   interface, with exact payload-mapping tests.
3. Extend Mongo operational metadata and add the lease-based worker plus its
   dedicated Docker service.
4. Run contract, backend, and Docker/Fabric integration checks; then update the
   cross-repository progress tracker with verified results and remaining
   Atlas/manual smoke work.

Each step is independently committable. No branch is pushed and no changes are
made directly on `main`.

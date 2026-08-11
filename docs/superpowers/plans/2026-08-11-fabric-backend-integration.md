# Fabric Backend Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reliably submit the backend's persisted payment and product-level shipment outbox records to the local Fabric network without blocking HTTP requests or fabricating confirmation.

**Architecture:** The Go chaincode stores each immutable Mongo outbox ID under its own ledger key and indexes it by business reference and product. A dedicated Nest worker atomically leases pending Mongo records, queries Fabric before a retry, submits through the real Fabric Gateway gRPC client, and persists confirmed commit metadata. The HTTP API only creates the outbox records.

**Tech Stack:** Hyperledger Fabric 3.1.4, Go 1.23, Node 24, NestJS 11, Mongoose 8/MongoDB Atlas, `@hyperledger/fabric-gateway`, `@grpc/grpc-js`, Docker Compose, Jest, Go `testing`.

## Global Constraints

- Work only on feature/develop branches; do not commit to or push `main`.
- Keep commits small and scoped to the completed task; never push a branch.
- Fabric submission is asynchronous: payment and shipment HTTP APIs must not wait for it.
- There is no runtime mock ledger and no silent fallback. Only the real Fabric Gateway client may be registered at runtime.
- Atlas is the production/development database target; Redis is local Docker infrastructure, not the blockchain outbox.
- Preserve the existing public statuses `pending`, `confirmed`, and `failed`; a lease is internal metadata, not a fourth status.
- Preserve exact typed payload names and one `productId` per supply-chain event.
- A checkout may contain multiple products but only one farmer. Do not introduce multi-farmer orders.
- Do not commit certificate material, generated Fabric organizations, Atlas credentials, or `.DS_Store` files.
- Docker Compose must use required Compose variables rather than `env_file`.

---

## Planned File Structure

### `farm2fork-blockchain`

| File | Responsibility |
| --- | --- |
| `chaincode/farm2fork-chaincode/internal/contract/contract.go` | Immutable ledger-key writes, idempotency, and index-backed queries. |
| `chaincode/farm2fork-chaincode/internal/contract/contract_test.go` | Contract coverage for repeat calls, conflicts, reference/product indexes. |
| `network/compose/compose-net.yaml` | Stable Fabric Docker network and matching chaincode-container network mode. |
| `scripts/smoke-test.sh` | Payment plus two-product shipment write/query smoke flow. |
| `README.md` | Required network and backend-worker environment/mount guidance. |

### `farm2fork-backend`

| File | Responsibility |
| --- | --- |
| `src/config/blockchain.config.ts` | Typed Fabric and worker configuration defaults. |
| `src/modules/blockchain/interfaces/fabric-gateway-client.interface.ts` | Narrow real-gateway boundary and serializable commit/ledger types. |
| `src/modules/blockchain/fabric-gateway.service.ts` | TLS/gRPC connection, certificate loading, query and submit mapping. |
| `src/modules/blockchain/blockchain-outbox.worker.ts` | Atomic lease, retry classification, Fabric recovery, and Mongo state updates. |
| `src/worker.ts` | Nest application-context entrypoint with no HTTP listener. |
| `src/modules/blockchain/schemas/blockchain-transaction.schema.ts` | Mutable operational submission metadata only. |
| `src/modules/transport/transport.service.ts` | One supply-chain outbox document per distinct order product. |
| `src/modules/blockchain/*.spec.ts` | Gateway mapping and worker lease/retry/recovery unit tests. |
| `test/blockchain-outbox.e2e-spec.ts` | Concurrent Mongo lease test using the existing replica-set test pattern. |
| `docker-compose.yml` | Separate worker service, external Fabric network, read-only crypto mount. |
| `package.json`, `pnpm-lock.yaml` | Gateway dependencies and worker scripts. |

### Cross-repository tracker

| File | Responsibility |
| --- | --- |
| `farm2fork-mobile/PROGRESS.md` | Record only verified commits/checks and explicitly retain Atlas/manual Fabric smoke work. |

## Task 1: Make Fabric Ledger Keys Immutable and Queryable

**Repository:** `farm2fork-blockchain`

**Files:**
- Modify: `chaincode/farm2fork-chaincode/internal/contract/contract.go:20-205`
- Modify: `chaincode/farm2fork-chaincode/internal/contract/contract_test.go:71-220`

**Interfaces:**
- Consumes: The existing `model.BlockchainTransaction` shape and Fabric stub APIs.
- Produces: `RecordPayment(ledgerKey, referenceID, ...)`, `RecordSupplyChainEvent(ledgerKey, referenceID, ...)`, `GetTransactionByLedgerKey(ledgerKey)`, `GetTransactionsByReference(referenceModel, referenceID)`, and `GetTransactionsByProductId(productID)`.

- [x] **Step 1: Add failing contract tests for the new identity behavior**

Replace the current reference-key assertions with tests shaped like:

```go
func TestRecordSupplyChainEventKeepsShipmentReferenceAndUsesOutboxLedgerKey(t *testing.T) {
    ctx := newMockTransactionContext("tx-shipment-001", "farm2forkchannel")
    contract := &contract.Farm2ForkContract{}

    payload, err := contract.RecordSupplyChainEvent(
        ctx, "outbox-001", "shipment-001", "Shipment", "product-001",
        "farmer-001", "shipment_assigned", "Lahore, Punjab", "transporter-001",
        "transporter", "2026-08-11T12:00:00Z",
    )

    require.NoError(t, err)
    stored, err := ctx.GetStub().GetState("outbox-001")
    require.NoError(t, err)
    require.Contains(t, string(stored), "shipment-001")
    require.Contains(t, payload, `"productId":"product-001"`)
}

func TestRecordSupplyChainEventReturnsExistingValueForSameLedgerKey(t *testing.T) {
    ctx := newMockTransactionContext("tx-shipment-002", "farm2forkchannel")
    c := &contract.Farm2ForkContract{}
    args := []string{"outbox-001", "shipment-001", "Shipment", "product-001", "farmer-001", "shipment_assigned", "Lahore, Punjab", "transporter-001", "transporter", "2026-08-11T12:00:00Z"}
    first, err := c.RecordSupplyChainEvent(ctx, args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9])
    require.NoError(t, err)
    second, err := c.RecordSupplyChainEvent(ctx, args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9])
    require.NoError(t, err)
    require.Equal(t, first, second)
}

func TestRecordSupplyChainEventRejectsDifferentPayloadForExistingLedgerKey(t *testing.T) {
    ctx := newMockTransactionContext("tx-shipment-003", "farm2forkchannel")
    c := &contract.Farm2ForkContract{}
    _, err := c.RecordSupplyChainEvent(ctx, "outbox-001", "shipment-001", "Shipment", "product-001", "farmer-001", "shipment_assigned", "Lahore, Punjab", "transporter-001", "transporter", "2026-08-11T12:00:00Z")
    require.NoError(t, err)
    _, err = c.RecordSupplyChainEvent(ctx, "outbox-001", "shipment-001", "Shipment", "product-002", "farmer-001", "shipment_assigned", "Lahore, Punjab", "transporter-001", "transporter", "2026-08-11T12:00:00Z")
    require.ErrorContains(t, err, "different immutable content")
}

func TestGetTransactionsByProductIdReturnsTwoShipmentEvents(t *testing.T) {
    ctx := newMockTransactionContext("tx-shipment-004", "farm2forkchannel")
    c := &contract.Farm2ForkContract{}
    _, err := c.RecordSupplyChainEvent(ctx, "outbox-a", "shipment-001", "Shipment", "product-001", "farmer-001", "shipment_assigned", "Lahore, Punjab", "transporter-001", "transporter", "2026-08-11T12:00:00Z")
    require.NoError(t, err)
    _, err = c.RecordSupplyChainEvent(ctx, "outbox-b", "shipment-001", "Shipment", "product-001", "farmer-001", "shipment_in_transit", "Lahore, Punjab", "transporter-001", "transporter", "2026-08-11T12:10:00Z")
    require.NoError(t, err)
    _, err = c.RecordSupplyChainEvent(ctx, "outbox-c", "shipment-001", "Shipment", "product-002", "farmer-001", "shipment_assigned", "Lahore, Punjab", "transporter-001", "transporter", "2026-08-11T12:00:00Z")
    require.NoError(t, err)
    payload, err := c.GetTransactionsByProductId(ctx, "product-001")
    require.NoError(t, err)
    var records []contractmodel.BlockchainTransaction
    require.NoError(t, json.Unmarshal([]byte(payload), &records))
    require.Len(t, records, 2)
}
```

- [x] **Step 2: Run the focused contract tests and confirm the intended failures**

Run: `go test ./chaincode/farm2fork-chaincode/internal/contract -run 'TestRecordSupplyChainEventKeeps|TestRecordSupplyChainEventReturns|TestRecordSupplyChainEventRejects|TestGetTransactionsByProductId' -count=1`

Expected: compilation failures because the new signatures/query methods do not yet exist, or assertion failures because the current code stores by business reference.

- [x] **Step 3: Implement immutable-key persistence and idempotency helpers**

In `contract.go`, replace the current `persistTransaction(ctx, referenceID, tx)` path with helpers that use the outbox key:

```go
func loadTransactionByLedgerKey(ctx contractapi.TransactionContextInterface, ledgerKey string) (*model.BlockchainTransaction, error)
func persistNewTransaction(ctx contractapi.TransactionContextInterface, ledgerKey string, tx *model.BlockchainTransaction) error
func ensureSameTransaction(existing, requested *model.BlockchainTransaction) error
```

`RecordPayment` and `RecordSupplyChainEvent` must:

1. load `ledgerKey` first;
2. return the existing serialized record when immutable fields/payload match;
3. return a conflict error when they differ; and
4. write only when absent.

Use `ledgerKey` only for `GetState`/`PutState`; continue assigning the supplied
business reference to `tx.ReferenceID`.

- [x] **Step 4: Add GoLevelDB-compatible indexes and queries**

Create composite keys with explicit namespaces:

```go
referenceIndexKey, err := ctx.GetStub().CreateCompositeKey(
    "f2f.reference", []string{tx.ReferenceModel, tx.ReferenceID, ledgerKey},
)
productIndexKey, err := ctx.GetStub().CreateCompositeKey(
    "f2f.product", []string{tx.Payload.SupplyChain.ProductID, ledgerKey},
)
```

Write a non-empty sentinel index value such as `[]byte{1}` with the immutable
transaction; an empty Fabric state value is treated as deletion by the test
stub. Implement query
methods using `GetStateByPartialCompositeKey`, `SplitCompositeKey`, and
`loadTransactionByLedgerKey`; skip index values that no longer resolve and
return a JSON array in deterministic iterator order. Only create the product
index when `Payload.SupplyChain != nil`.

- [x] **Step 5: Run the focused tests and the complete Go package tests**

Run: `go test ./chaincode/farm2fork-chaincode/internal/contract -count=1`

Expected: PASS, including existing exact-field-name checks updated to the new
argument positions.

Run: `go test ./chaincode/farm2fork-chaincode/... -count=1`

Expected: PASS.

- [x] **Step 6: Commit the chaincode identity and query change**

```bash
git add chaincode/farm2fork-chaincode/internal/contract/contract.go \
  chaincode/farm2fork-chaincode/internal/contract/contract_test.go
git commit -m "feat: index immutable Fabric transaction records"
```

## Task 2: Exercise Product-Level Records in the Fabric Network

**Repository:** `farm2fork-blockchain`

**Files:**
- Modify: `network/compose/compose-net.yaml:1-68`
- Modify: `scripts/smoke-test.sh:22-72`
- Modify: `README.md:11-38`

**Interfaces:**
- Consumes: Task 1 transaction names and query signatures.
- Produces: A stable Docker network named `farm2fork-fabric` and a smoke path
  that proves two product events can share one shipment reference.

- [x] **Step 1: Add failing smoke expectations for immutable keys and product lookup**

Update `scripts/smoke-test.sh` to invoke these exact chaincode argument forms:

```bash
'{"Args":["RecordPayment","outbox-payment-001","payment-001","order-001","buyer-001","farmer-001","1500","PKR","stripe","2026-08-11T12:00:00Z"]}'
'{"Args":["RecordSupplyChainEvent","outbox-shipment-apple","shipment-001","Shipment","product-apple","farmer-001","shipment_assigned","Lahore, Punjab","transporter-001","transporter","2026-08-11T12:05:00Z"]}'
'{"Args":["RecordSupplyChainEvent","outbox-shipment-mango","shipment-001","Shipment","product-mango","farmer-001","shipment_assigned","Lahore, Punjab","transporter-001","transporter","2026-08-11T12:05:00Z"]}'
```

Query the payment by `GetTransactionByLedgerKey`, the two events by
`GetTransactionsByReference`, and each timeline by
`GetTransactionsByProductId`.

- [ ] **Step 2: Verify the current smoke script fails against the old deployed contract**

Run: `SMOKE_RESET_NETWORK=true bash scripts/smoke-test.sh`

Expected: failure before the query checks because the current chaincode does
not accept the new ledger-key argument layout. Stop containers with
`bash scripts/network-down.sh` if the script exits before its own cleanup.

- [x] **Step 3: Name the Fabric Docker network explicitly**

Append this network declaration to `network/compose/compose-net.yaml` and set
the peer chaincode-container network mode to the same exact name:

```yaml
networks:
  default:
    name: farm2fork-fabric
```

```yaml
- CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE=farm2fork-fabric
```

Do not use the generated `compose_default` name. The backend worker needs a
stable external-network name.

- [x] **Step 4: Complete the smoke checks and document the integration boundary**

Add `rg` assertions for both distinct product IDs and the shared
`shipment-001` reference. Update `README.md` to state that Fabric now uses the
`farm2fork-fabric` Docker network, emits immutable outbox-key records, and is
consumed only by the backend worker over gRPC/TLS.

- [x] **Step 5: Verify Compose shape and the full Fabric smoke flow**

Run: `docker compose --env-file .env.example -f network/compose/compose-net.yaml config`

Expected: PASS and network name `farm2fork-fabric` appears in rendered output.

Run: `SMOKE_RESET_NETWORK=true bash scripts/smoke-test.sh`

Expected: PASS; queried records contain `product-apple`, `product-mango`, and
the same shipment reference without overwriting either event.

Runtime note (2026-08-11): `go test ./... -count=1`, rendered Compose
configuration, and the complete `SMOKE_RESET_NETWORK=true bash
scripts/smoke-test.sh` flow passed. The network generates and verifies its
orderer TLS leaf with SHA-256, and pre-pulls the matching Fabric chaincode
builder image before starting containers.

- [x] **Step 6: Commit the network and smoke update**

```bash
git add network/compose/compose-net.yaml scripts/smoke-test.sh README.md
git commit -m "test: verify product-level Fabric shipment events"
```

## Task 3: Add Typed Backend Fabric Configuration and Gateway Boundary

**Repository:** `farm2fork-backend`

**Files:**
- Create: `src/config/blockchain.config.ts`
- Create: `src/modules/blockchain/interfaces/fabric-gateway-client.interface.ts`
- Create: `src/modules/blockchain/fabric-gateway.service.ts`
- Create: `src/modules/blockchain/fabric-gateway.service.spec.ts`
- Modify: `src/config/index.ts`
- Modify: `src/app.module.ts:38-70`
- Modify: `src/modules/blockchain/blockchain.module.ts:1-21`
- Modify: `package.json`
- Modify: `pnpm-lock.yaml`

**Interfaces:**
- Consumes: Task 1 chaincode transactions and the existing
  `BlockchainTransactionDocument` schema.
- Produces:

```ts
export interface FabricGatewayClient {
  findByLedgerKey(ledgerKey: string): Promise<FabricLedgerRecord | null>;
  submit(record: BlockchainTransactionDocument): Promise<FabricCommit>;
}

export type FabricFailureKind = 'permanent' | 'transient';
export class FabricGatewayFailure extends Error {
  constructor(
    readonly kind: FabricFailureKind,
    readonly code: 'invalid_payload' | 'ledger_conflict' | 'fabric_validation' | 'fabric_unavailable',
    message: string,
  ) { super(message); }
}
```

- [ ] **Step 1: Write failing mapping and configuration tests**

In `fabric-gateway.service.spec.ts`, use a fake contract object and verify the
real service calls it with exact string arguments:

```ts
expect(contract.submitTransaction).toHaveBeenCalledWith(
  'RecordPayment',
  outboxId, paymentId, orderId, buyerId, farmerId, '1500', 'PKR', 'stripe', paidAt,
);

expect(contract.submitTransaction).toHaveBeenCalledWith(
  'RecordSupplyChainEvent',
  outboxId, shipmentId, 'Shipment', productId, farmerId,
  'shipment_in_transit', 'Lahore, Punjab', transporterId, 'transporter', timestamp,
);
```

Add tests that a missing payment/supply-chain field fails before a contract
call, `GetTransactionByLedgerKey` returning a Fabric not-found result maps to
`null`, and a committed status returns `transactionId`, `Number(blockNumber)`,
and the configured channel name.

- [ ] **Step 2: Run the focused test and observe the missing-service failure**

Run: `pnpm test -- fabric-gateway.service.spec.ts --runInBand`

Expected: FAIL because the client interface and service do not exist.

- [ ] **Step 3: Add dependencies, config, and a narrow interface**

Run: `pnpm add @hyperledger/fabric-gateway @grpc/grpc-js`

Create `blockchain.config.ts` with a typed config object for worker enablement,
poll/lease/retry timing, channel/chaincode/MSP, peer endpoint/hostname, and
the three mounted certificate/key paths. Add matching Joi fields in
`AppModule`; only require Fabric paths when `BLOCKCHAIN_WORKER_ENABLED=true`.

Export this config from `src/config/index.ts`. Define the interface plus
`FabricLedgerRecord` and `FabricCommit` types in the new interfaces directory.
Use an injection token such as `FABRIC_GATEWAY_CLIENT`; do not add any mock
provider to `BlockchainModule`.

- [ ] **Step 4: Implement the real Gateway service**

Use the current Node Gateway APIs:

```ts
const client = new grpc.Client(endpoint, grpc.credentials.createSsl(tlsRootCert), {
  'grpc.ssl_target_name_override': peerHostAlias,
});
const gateway = connect({ identity: { mspId, credentials }, signer, client });
const contract = gateway.getNetwork(channelName).getContract(chaincodeName);
```

Load certificates using `node:fs/promises`, create the signer from
`node:crypto.createPrivateKey`, and close both gateway and gRPC client in
`OnModuleDestroy`. Use `contract.evaluateTransaction('GetTransactionByLedgerKey',
ledgerKey)` for lookup. For a new write, use `submitAsync`, then require
`(await submitted.getStatus()).successful`; map its `transactionId` and
`Number(status.blockNumber)`. Parse returned JSON with a `TextDecoder`.

Classify contract validation/endorsement conflicts as a
`FabricGatewayFailure('permanent', 'fabric_validation' | 'ledger_conflict', message)`
and gRPC deadline/unavailable failures as a
`FabricGatewayFailure('transient', 'fabric_unavailable', message)` in explicit
helper functions; tests must exercise both helpers. Never log payloads, PEM
text, private-key paths, or gateway references.

- [ ] **Step 5: Run focused tests, build, and dependency audit**

Run: `pnpm test -- fabric-gateway.service.spec.ts --runInBand`

Expected: PASS.

Run: `pnpm run build`

Expected: PASS.

Run: `pnpm why @hyperledger/fabric-gateway`

Expected: one direct backend dependency recorded in the lockfile.

- [ ] **Step 6: Commit the gateway boundary**

```bash
git add package.json pnpm-lock.yaml src/config src/app.module.ts \
  src/modules/blockchain/interfaces src/modules/blockchain/fabric-gateway.service.ts \
  src/modules/blockchain/fabric-gateway.service.spec.ts src/modules/blockchain/blockchain.module.ts
git commit -m "feat: add Fabric gateway client"
```

## Task 4: Fan Out Shipment Events Per Distinct Product

**Repository:** `farm2fork-backend`

**Files:**
- Modify: `src/modules/transport/transport.service.ts:175-212,280-319`
- Modify: `src/modules/transport/transport.service.spec.ts:220-340`
- Modify: `test/shipment.e2e-spec.ts:80-180`

**Interfaces:**
- Consumes: Existing `order.items[]`, the typed `BlockchainTransaction` schema,
  and Task 3's strict supply-chain mapping requirement.
- Produces: One pending `BlockchainTransaction` for each distinct product ID at
  shipment assignment and every later shipment status transition.

- [x] **Step 1: Add failing transport unit tests for a two-product order**

Add a test that supplies an order with duplicated cart lines for `product-a`
and one line for `product-b`, then asserts `blockchainModel.create` receives
two records, not three:

```ts
expect(blockchainModel.create).toHaveBeenCalledWith(
  expect.arrayContaining([
    expect.objectContaining({
      referenceModel: BlockchainReferenceModel.Shipment,
      referenceId: shipmentId,
      payload: expect.objectContaining({
        supplyChain: expect.objectContaining({ productId: productA }),
      }),
    }),
    expect.objectContaining({
      payload: expect.objectContaining({
        supplyChain: expect.objectContaining({ productId: productB }),
      }),
    }),
  ]),
  { session: expect.anything() },
);
```

Repeat the assertion for `updateStatus`, including the same event type and
timestamp across the two emitted records.

- [x] **Step 2: Run the targeted test and confirm it fails**

Run: `pnpm test -- transport.service.spec.ts --runInBand`

Expected: FAIL because current transport code creates one record and omits
`productId`.

- [x] **Step 3: Create transaction-safe fan-out records**

Inside the existing Mongo session, derive product IDs without duplicate lines:

```ts
const productIds = [...new Set(order.items.map((item) => item.productId.toHexString()))]
  .map((id) => new Types.ObjectId(id));

await this.blockchainModel.create(
  productIds.map((productId) => ({
    type: BlockchainTxType.SupplyChainEvent,
    referenceId: shipment._id,
    referenceModel: BlockchainReferenceModel.Shipment,
    payload: {
      payment: null,
      supplyChain: { productId, farmerId: order.farmerId, eventType, location, actorId, actorRole, timestamp: now },
    },
    status: BlockchainTxStatus.Pending,
  })),
  { session },
);
```

Extract this into one private helper used by both claim and status-transition
paths so all shared values are identical. Do not alter the one-farmer order
guard or create separate shipments.

- [x] **Step 4: Extend the replica-set e2e assertion**

Seed an order with two product IDs, perform a claim, then require two pending
records with the same shipment `referenceId` and two distinct payload product
IDs. Keep the existing concurrent-claim assertion intact.

- [ ] **Step 5: Run focused transport and e2e tests**

Run: `pnpm test -- transport.service.spec.ts --runInBand`

Expected: PASS.

Run: `pnpm test:e2e --runInBand`

Expected: PASS, including the multi-product shipment assertion.

Runtime note (2026-08-11): `pnpm run build`, the focused transport unit suite
(8 tests), and `test/shipment.e2e-spec.ts` (3 replica-set tests) passed. The
full `pnpm test:e2e --runInBand` command still fails while loading the existing
`app.e2e-spec.ts`: Jest cannot transform the ESM-only `@noble/curves` dependency
loaded through the Fabric gateway runtime. That runner/configuration defect is
outside the transport change, so this step remains open.

- [x] **Step 6: Commit the product fan-out change**

```bash
git add src/modules/transport/transport.service.ts \
  src/modules/transport/transport.service.spec.ts test/shipment.e2e-spec.ts
git commit -m "feat: record shipment events per product"
```

## Task 5: Add Durable Lease and Retry Metadata

**Repository:** `farm2fork-backend`

**Files:**
- Modify: `src/modules/blockchain/schemas/blockchain-transaction.schema.ts:103-142`
- Create: `src/modules/blockchain/blockchain-outbox.worker.ts`
- Create: `src/modules/blockchain/blockchain-outbox.worker.spec.ts`
- Modify: `src/modules/blockchain/blockchain.module.ts`

**Interfaces:**
- Consumes: `FabricGatewayClient` from Task 3 and records created by Task 4.
- Produces: A `BlockchainOutboxWorker.processNext(now?: Date)` operation that
  processes at most one atomically leased pending record.

- [x] **Step 1: Write failing worker tests with in-memory model fakes**

Cover these deterministic cases in `blockchain-outbox.worker.spec.ts`:

```ts
it('claims only an expired-or-unleased pending record with retryCount below 3', async () => {
  await worker.processNext(now);
  expect(model.findOneAndUpdate).toHaveBeenCalledWith(
    expect.objectContaining({ status: BlockchainTxStatus.Pending, retryCount: { $lt: 3 } }),
    expect.objectContaining({ $set: expect.objectContaining({ leaseToken: expect.any(String) }) }),
    expect.objectContaining({ sort: { createdAt: 1 }, new: true }),
  );
});

it('confirms an already committed ledger key without submitting again', async () => {
  const leaseToken = 'lease-001';
  model.findOneAndUpdate.mockResolvedValue({ _id: outboxId, leaseToken, status: BlockchainTxStatus.Pending });
  gateway.findByLedgerKey.mockResolvedValue(existingLedgerRecord);
  await worker.processNext(now);
  expect(gateway.submit).not.toHaveBeenCalled();
  expect(model.updateOne).toHaveBeenCalledWith(expect.objectContaining({ leaseToken }), expect.objectContaining({ $set: { status: BlockchainTxStatus.Confirmed } }));
});

it('fails permanently on a mapping conflict and schedules a transient retry', async () => {
  gateway.submit.mockRejectedValueOnce(new FabricGatewayFailure('permanent', 'ledger_conflict', 'immutable conflict'));
  await worker.processNext(now);
  expect(model.updateOne).toHaveBeenLastCalledWith(
    expect.objectContaining({ leaseToken }),
    expect.objectContaining({ $set: expect.objectContaining({ status: BlockchainTxStatus.Failed, lastErrorCode: 'ledger_conflict' }) }),
  );
  gateway.submit.mockRejectedValueOnce(new FabricGatewayFailure('transient', 'fabric_unavailable', 'deadline exceeded'));
  await worker.processNext(now);
  expect(model.updateOne).toHaveBeenLastCalledWith(
    expect.objectContaining({ leaseToken }),
    expect.objectContaining({ $inc: { retryCount: 1 }, $set: expect.objectContaining({ nextAttemptAt: expect.any(Date) }) }),
  );
});
```

- [x] **Step 2: Run the focused worker test and confirm it fails**

Run: `pnpm test -- blockchain-outbox.worker.spec.ts --runInBand`

Expected: FAIL because the worker and operational fields do not exist.

- [x] **Step 3: Extend only operational Mongo metadata**

Add optional schema properties with indexes appropriate to leasing:

```ts
@Prop({ index: true }) nextAttemptAt?: Date;
@Prop({ index: true }) leaseExpiresAt?: Date;
@Prop() leaseToken?: string;
@Prop() lastAttemptAt?: Date;
@Prop() lastErrorCode?: string;
@Prop() lastErrorMessage?: string;
@Prop() confirmedAt?: Date;
```

Keep the existing payload and business identity fields immutable by convention;
only the worker writes these operational fields plus status/Fabric metadata.

- [x] **Step 4: Implement atomic leasing and guarded finalization**

Use one `findOneAndUpdate` query with an `$and` of two `$or` clauses: eligible
`nextAttemptAt` and absent/expired `leaseExpiresAt`. Set a UUID lease token,
`lastAttemptAt`, and `leaseExpiresAt = now + leaseMs`; sort by `createdAt: 1`.

Before `gateway.submit`, call `gateway.findByLedgerKey(record._id.toHexString())`.
On a confirmed lookup or successful submit, call `updateOne` with both `_id`
and `leaseToken`, set `status`, `txHash`, `blockNumber`, `channelName`,
`confirmedAt`, and unset all lease/error fields. On a transient error, use the
same guarded filter to increment `retryCount`, clear the lease, and set
`nextAttemptAt` from `baseDelay * 2 ** retryCount` plus bounded jitter. Mark
the third failed submission `failed`; permanent errors fail immediately.

Use explicit error codes such as `invalid_payload`, `ledger_conflict`,
`fabric_validation`, and `fabric_unavailable`; truncate sanitized messages to a
fixed small length. Do not persist raw PEM, gRPC metadata, or payload JSON.

- [x] **Step 5: Run worker and gateway unit tests plus the backend build**

Run: `pnpm test -- blockchain-outbox.worker.spec.ts fabric-gateway.service.spec.ts --runInBand`

Expected: PASS.

Runtime note (2026-08-11): `blockchain-outbox.worker.spec.ts` and
`fabric-gateway.service.spec.ts` passed together (9 tests), and `pnpm run
build` passed. The worker leases one eligible record at a time, checks Fabric
by immutable outbox key before submitting, and records guarded confirmation,
retry, or terminal failure metadata.

Run: `pnpm run build`

Expected: PASS.

- [x] **Step 6: Commit the durable outbox worker core**

```bash
git add src/modules/blockchain/schemas/blockchain-transaction.schema.ts \
  src/modules/blockchain/blockchain.module.ts \
  src/modules/blockchain/blockchain-outbox.worker.ts \
  src/modules/blockchain/blockchain-outbox.worker.spec.ts
git commit -m "feat: retry Fabric outbox submissions"
```

## Task 6: Run the Worker as an Isolated Docker Service

**Repository:** `farm2fork-backend`

**Files:**
- Create: `src/worker.ts`
- Modify: `package.json`
- Modify: `docker-compose.yml`
- Modify: `src/app.module.ts:38-70`
- Modify: `src/modules/blockchain/blockchain.module.ts`
- Create: `docs/fabric-worker.md`

**Interfaces:**
- Consumes: Task 3 configuration and Task 5 worker service.
- Produces: `pnpm start:worker:dev`, `pnpm start:worker`, and Docker service
  `backend-worker` with no published HTTP port.

- [ ] **Step 1: Write failing bootstrap/configuration tests**

Add an AppModule configuration test that sets `BLOCKCHAIN_WORKER_ENABLED=true`
without `FABRIC_TLS_CERT_PATH`, `FABRIC_IDENTITY_CERT_PATH`, or
`FABRIC_IDENTITY_KEY_PATH` and expects Joi validation to reject startup. Add a
worker test that ensures polling starts only when the enabled flag is true.

- [ ] **Step 2: Run the focused tests and confirm they fail**

Run: `pnpm test -- blockchain-outbox.worker.spec.ts --runInBand`

Expected: FAIL for the missing bootstrap/configuration behavior.

- [ ] **Step 3: Add the non-HTTP worker entrypoint and lifecycle**

Create `src/worker.ts` using `NestFactory.createApplicationContext(AppModule)`.
It must wait for `SIGINT`/`SIGTERM`, close the application context cleanly, and
never call `listen()`. Have `BlockchainOutboxWorker` implement lifecycle hooks
that schedule one non-overlapping `processNext()` loop only when
`BLOCKCHAIN_WORKER_ENABLED` is true. A startup configuration failure must make
the worker exit nonzero rather than marking business records failed.

Add scripts:

```json
"start:worker:dev": "ts-node -r tsconfig-paths/register src/worker.ts",
"start:worker": "node dist/worker"
```

- [ ] **Step 4: Wire Compose without `env_file`**

Add `backend-worker` beside `backend`, reusing the development image and
source/node-module volumes, but set `command: pnpm start:worker:dev` and omit
`ports`. Require `DATABASE_URL` and `FABRIC_CRYPTO_HOST_PATH`; mount the latter
read-only:

```yaml
volumes:
  - ${FABRIC_CRYPTO_HOST_PATH:?FABRIC_CRYPTO_HOST_PATH is required}:/fabric/crypto:ro
networks:
  fabric:
    external: true
    name: ${FABRIC_DOCKER_NETWORK:-farm2fork-fabric}
```

Set Fabric variable values to paths under `/fabric/crypto`; do not copy
certificates into the image. Keep the HTTP backend off the Fabric network
unless it gains a real Fabric dependency later.

- [ ] **Step 5: Document and verify the Docker runtime shape**

In `docs/fabric-worker.md`, document the ordered local startup:

```bash
cd ../farm2fork-blockchain && bash scripts/network-up.sh && bash scripts/create-channel.sh && bash scripts/deploy-chaincode.sh
cd ../farm2fork-backend && docker compose --env-file .env.example config
```

Document every required Fabric variable and that a real Atlas `DATABASE_URL`
is required to run the worker. Run:

`DATABASE_URL='mongodb+srv://example.invalid/farm2fork' FABRIC_CRYPTO_HOST_PATH='/tmp/fabric-crypto' docker compose config`

Expected: configuration renders `backend-worker`, `redis`, the external
`farm2fork-fabric` network, and no `env_file` key.

- [ ] **Step 6: Commit the worker runtime slice**

```bash
git add src/worker.ts src/app.module.ts src/modules/blockchain package.json \
  pnpm-lock.yaml docker-compose.yml docs/fabric-worker.md
git commit -m "feat: run Fabric worker in Docker"
```

## Task 7: Prove Lease Ownership Against a Replica Set and Record Handoff

**Repository:** `farm2fork-backend`, then `farm2fork-mobile`

**Files:**
- Create: `test/blockchain-outbox.e2e-spec.ts`
- Modify: `docs/fabric-worker.md`
- Modify: `farm2fork-mobile/PROGRESS.md:11-57`

**Interfaces:**
- Consumes: Tasks 3-6.
- Produces: Evidence that two workers do not submit the same eligible Mongo
  outbox record concurrently, plus an accurate cross-repository status entry.

- [ ] **Step 1: Write the concurrent-lease e2e test**

Follow `test/shipment.e2e-spec.ts`'s `MongoMemoryReplSet` setup. Seed one
pending payment outbox record, create two worker instances sharing the same
Mongo model and fake real-gateway interface, and execute concurrently:

```ts
await Promise.all([workerA.processNext(now), workerB.processNext(now)]);
expect(gateway.submit).toHaveBeenCalledTimes(1);
expect(await blockchainModel.findById(outboxId).lean()).toMatchObject({
  status: BlockchainTxStatus.Confirmed,
  txHash: 'fabric-tx-001',
});
```

Add a second test that first returns a ledger record from `findByLedgerKey` and
asserts confirmation occurs without `submit`.

- [ ] **Step 2: Run the e2e test and confirm the baseline failure**

Run: `pnpm test:e2e --runInBand --testPathPattern blockchain-outbox`

Expected: FAIL until the worker is wired into the test module and claims are
atomic.

- [ ] **Step 3: Complete test wiring and run focused verification**

Use the same replica-set test configuration as payment/shipment e2e tests;
inject a fake `FABRIC_GATEWAY_CLIENT` in the test module only. This is a test
double, not a runtime mock provider.

Run: `pnpm test:e2e --runInBand --testPathPattern blockchain-outbox`

Expected: PASS.

Run: `pnpm test -- --runInBand`

Expected: PASS.

Run: `pnpm run build`

Expected: PASS.

- [ ] **Step 4: Run the opt-in real Fabric smoke boundary**

With the user-provided Atlas URL and generated Fabric crypto directory set,
start Fabric then the worker. Seed one valid pending outbox record through the
normal payment/shipment flow, wait for `confirmed`, and query it by ledger key.
Record the exact command and result in `docs/fabric-worker.md`. Do not claim
this check passed if Atlas credentials or Docker Fabric are unavailable.

- [ ] **Step 5: Commit the verification work**

```bash
git add test/blockchain-outbox.e2e-spec.ts docs/fabric-worker.md
git commit -m "test: verify Fabric outbox leases"
```

- [ ] **Step 6: Update the project tracker in its own repository commit**

After only verified checks pass, update `farm2fork-mobile/PROGRESS.md` to note:

- Fabric chaincode immutable record/index commit(s);
- backend real gateway/worker commit(s);
- exact unit/e2e/build/Docker-config outcomes; and
- deferred user-owned Atlas + Docker Fabric smoke work, if still unrun.

Commit only that tracker file on an appropriate local feature branch:

```bash
git add PROGRESS.md
git commit -m "docs: record Fabric integration progress"
```

## Plan Self-Review

Coverage map:

- Immutable ledger keys, idempotency, and business/product indexes: Tasks 1-2.
- Exact real Gateway mapping, TLS files, commit status, and no mock provider: Task 3.
- Multiple products in one one-farmer shipment: Task 4.
- Mongo durable lease, retry, permanent/transient classification, and recovery: Task 5.
- Dedicated Docker worker, stable network, required Compose variables, and no
  HTTP listener: Task 6.
- Replica-set concurrency evidence, optional Atlas/Fabric smoke, and status
  tracking: Task 7.

The plan intentionally contains no browser/mobile implementation and does not
weaken the payload schema or add multi-farmer orders.

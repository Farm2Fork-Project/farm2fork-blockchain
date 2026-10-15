# farm2fork-blockchain

Farm2Fork local Hyperledger Fabric environment.

## Versions

- Hyperledger Fabric: 3.1.4
- Go support line: 1.23.x
- Default pinned Go toolchain: 1.23.0 (see `.env.example`)

## Network defaults

- Channel: `farm2forkchannel`
- Orderer: `orderer.farm2fork.com`
- Peer: `peer0.farm2fork.com`
- Docker network: `farm2fork-fabric`
- Immutable ledger key: backend `BlockchainTransaction._id`
- Queries: `GetTransactionByLedgerKey`, `GetTransactionsByReference`, and `GetTransactionsByProductId`

## Full flow

```bash
cp .env.example .env
bash scripts/network-up.sh
bash scripts/create-channel.sh
bash scripts/deploy-chaincode.sh
bash scripts/smoke-test.sh
node scripts/benchmark.js --operations 20 --concurrency 1,5,10,20
bash scripts/network-down.sh
```

`scripts/smoke-test.sh` resets the local Fabric network by default so it always tests
the chaincode currently on disk. Set `SMOKE_RESET_NETWORK=false` if you need to
preserve the current local ledger while running the smoke test.

## Chaincode write validation

Fabric validates every write before looking up an idempotency key or creating ledger
state and composite indexes. The backend remains responsible for Firebase/JWT
authentication and database ownership checks.

`RecordPayment` requires non-empty ledger, payment, order, buyer, and farmer IDs; a
finite positive amount; an uppercase three-letter currency code; `stripe`, `jazzcash`, or `safepay` as the gateway; and an RFC3339 `paidAt` timestamp.

Safepay uses the same immutable payment payload and ledger-key idempotency as
the other gateways. Chaincode tests cover writing, querying, identical retries,
and rejection of conflicting payment amounts. Run `go test ./... -count=1` from
`chaincode/farm2fork-chaincode` to verify the contract.

An already deployed network must upgrade its chaincode before it can accept
Safepay records: update `CHAINCODE_LABEL`, `CHAINCODE_VERSION`, and
`CHAINCODE_SEQUENCE` in the local `.env` for the next package/definition, then
run `bash scripts/deploy-chaincode.sh`. Use the next sequence after the currently
committed definition. Preserve the ledger; do not reset the network for this
upgrade. Outbox records already marked `failed` do not automatically become
pending when chaincode changes and need separate operator review before retry.

`RecordSupplyChainEvent` requires non-empty ledger, reference, product, farmer,
location, actor, actor-role, and timestamp fields. Its RFC3339 timestamp and
permitted event combinations are:

| Reference model | Event type | Actor role |
| --- | --- | --- |
| `Product` | `listed` | `farmer` |
| `Shipment` | `shipment_assigned` | `transporter` |
| `Shipment` | `shipment_picked_up` | `transporter` |
| `Shipment` | `shipment_in_transit` | `transporter` |
| `Shipment` | `shipment_delivered` | `transporter` |
| `Shipment` | `shipment_failed` | `transporter` |
| `Shipment` | `shipment_issue_reported` | `transporter` |
| `Shipment` | `shipment_reassigned` | `admin` |
| `Shipment` | `shipment_issue_resolved` | `admin` |

## Performance benchmark

`node scripts/benchmark.js --operations 20 --concurrency 1,5,10,20` measures
the deployed provenance-event writes and product-provenance query without
resetting the network. See [docs/benchmark.md](docs/benchmark.md) for the exact
paper-workload mapping, timing boundaries, and result retention rules.

## Limitations

- Single-org only
- No CouchDB
- Fabric is consumed by the backend worker over gRPC/TLS, never by an HTTP endpoint
- The backend worker integration is implemented separately from this network setup

### Delivery recovery compatibility

The admin/transporter release adds the three recovery event types above. Upgrade chaincode validation before processing the matching backend outbox events. Existing ledger entries and event types remain valid; do not reset the ledger. Application role authorization and assignment/custody checks remain enforced by the backend. A local chaincode test is not evidence of a production upgrade or ledger readback.

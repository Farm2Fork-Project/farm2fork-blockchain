# Fabric Chaincode Input Validation Design

## Goal

Prevent malformed or semantically invalid payment and supply-chain records from
being written to Fabric's immutable ledger, without changing Firebase, backend,
web, mobile, network topology, or the existing backend Gateway method names.

## Scope

The change is limited to the Go chaincode contract and its unit tests. It adds
validation before the existing idempotency lookup and before any `PutState` or
composite-index write.

It does not introduce a client-facing Fabric API, change the outbox worker,
store private buyer/contact/address data, or migrate existing ledger entries.
Deploying the resulting chaincode revision remains a separate, deliberate local
Fabric upgrade after tests pass.

## Immutable record invariants

Both write functions require nonblank, trimmed values for `ledgerKey`,
`referenceId`, and their relevant IDs. A record never writes a blank identity
or an empty timestamp. `ledgerKey` remains the immutable backend outbox ID;
repeating it with identical content remains idempotent, while different content
continues to fail.

All time values use RFC3339 and are normalized only for validation: the exact
caller-provided string remains the payload value to preserve the established
backend/Fabric contract.

### Payment records

`RecordPayment` accepts only:

- a finite positive amount;
- ISO 4217-style three-uppercase-letter currency values;
- gateway `stripe` or `jazzcash`; and
- nonblank `orderId`, `buyerId`, `farmerId`, and `paidAt`.

The contract retains the current `Payment` reference model and `payment` type;
no caller-supplied alternative is added.

### Supply-chain records

`RecordSupplyChainEvent` accepts only the following coherent combinations:

| Reference model | Event type | Required actor role |
|---|---|---|
| `Product` | `listed` | `farmer` |
| `Shipment` | `shipment_assigned` | `transporter` |
| `Shipment` | `shipment_picked_up` | `transporter` |
| `Shipment` | `shipment_in_transit` | `transporter` |
| `Shipment` | `shipment_delivered` | `transporter` |
| `Shipment` | `shipment_failed` | `transporter` |

Every supply-chain record also requires nonblank `productId`, `farmerId`,
`location`, `actorId`, and timestamp. The chaincode does not validate whether a
particular actor owns a database record—that remains backend authorization—but
it prevents nonsense event labels, reference models, and actor-role pairings
from permanently entering the ledger.

## Implementation structure

`internal/contract/validation.go` will hold small pure validation helpers. The
existing public chaincode method signatures stay unchanged. `RecordPayment` and
`RecordSupplyChainEvent` call their validator after checking the Fabric context
and before building/loading a transaction. This means invalid retries never
read, return, or mutate an existing ledger key.

Errors describe the invalid contract field/value category without exposing
private data, peer configuration, certificates, or backend internals.

## Test strategy

Tests are table-driven and invoke the public contract functions through the
existing mock transaction context. They prove each invalid case returns an
error and leaves both the main ledger key and its reference/product indexes
absent. Positive regression cases cover every allowlisted supply-chain event,
both payment gateways, valid timestamp/currency/amount boundaries, existing
idempotent writes, conflicting duplicate rejection, and index queries.

Required verification:

```bash
cd chaincode/farm2fork-chaincode
go test ./...
go vet ./...
```

The local Fabric smoke script will be rerun only after source-level tests pass.
It writes development records into a local network only; no Atlas, Firebase,
backend, web, or mobile system is invoked.

## Acceptance criteria

- Invalid payment and supply-chain input cannot reach `PutState` or create an
  index entry.
- The existing backend event taxonomy and method signatures remain compatible.
- Valid current payment and shipment records remain idempotent and queryable.
- No source file outside the blockchain repository changes.
- Test and smoke outcomes are recorded accurately; no deployed-network success
  is claimed without running that network.

# Fabric Chaincode Input Validation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reject malformed or unsupported payment and supply-chain writes before they can be read, indexed, or committed to the Fabric ledger, while preserving the existing backend event contract and idempotency behavior for valid events.

**Architecture:** Keep validation at the chaincode trust boundary. Pure validators will normalize and verify command arguments and payload fields; `RecordPayment` and `RecordSupplyChainEvent` will invoke them before their existing idempotency lookup and persistence flow. The backend remains authoritative for identity and database ownership checks, so the contract will validate shape and permitted business-event combinations only.

**Tech Stack:** Go 1.23, Hyperledger Fabric Contract API 2.5, Go testing, Docker-based local Fabric network.

## Global Constraints

- Work on `feature/fabric-backend-integration-design`; do not create or push to `main`.
- Keep commits small and focused. Do not mention Codex in a branch or commit message.
- Do not modify Firebase-authentication or FCM work in backend, web, or mobile repositories.
- Preserve the backend's current Fabric command names and event payload contract.
- Keep local Fabric state local; do not access MongoDB Atlas or external payment systems.

## File Structure

| File | Responsibility |
| --- | --- |
| `chaincode/farm2fork-chaincode/internal/contract/validation.go` | Pure validation and normalization helpers for Fabric write inputs. |
| `chaincode/farm2fork-chaincode/internal/contract/validation_test.go` | Table-driven unit coverage for every accepted and rejected payment/supply-chain input shape. |
| `chaincode/farm2fork-chaincode/internal/contract/contract.go` | Invoke validation before idempotency lookup or ledger writes. |
| `chaincode/farm2fork-chaincode/internal/contract/contract_test.go` | Contract-level proof that invalid requests leave no state or composite indexes. |
| `README.md` | Document the enforced input contract and local verification command. |

---

### Task 1: Add payment input validation through tests

**Files:**

- Create: `chaincode/farm2fork-chaincode/internal/contract/validation_test.go`
- Create: `chaincode/farm2fork-chaincode/internal/contract/validation.go`
- Modify: `chaincode/farm2fork-chaincode/internal/contract/contract.go`

- [ ] **Step 1: Write failing payment validator tests**

  Add table-driven tests for a valid payment and for each invalid case: blank ledger key, blank reference ID, blank order/buyer/farmer IDs, zero or negative amount, `NaN`, infinity, invalid currency, unsupported gateway, and malformed `paidAt` timestamp. Include whitespace-only values to establish trimming behavior.

  Example test shape:

  ```go
  _, err := validatePaymentInput(" ", "payment-id", validPaymentPayload())
  require.ErrorContains(t, err, "ledger key")
  ```

- [ ] **Step 2: Run the focused test and confirm it fails**

  Run: `go test ./chaincode/farm2fork-chaincode/internal/contract -run TestValidatePaymentInput -count=1`

  Expected: failure because validation helpers do not yet exist.

- [ ] **Step 3: Implement payment validation**

  Add a validator that trims required identifiers, requires a finite positive amount, requires an uppercase three-letter ISO-style currency code, permits only `stripe` and `jazzcash`, and validates `paidAt` with `time.RFC3339`. Return canonicalized values so the persisted record has no accidental surrounding whitespace.

- [ ] **Step 4: Invoke validation before the payment idempotency lookup**

  Update `RecordPayment` to validate its ledger key, reference ID, and payload before calling `loadTransactionByLedgerKey`. Keep the duplicate-exact-match logic unchanged for valid requests.

- [ ] **Step 5: Run focused payment tests**

  Run: `go test ./chaincode/farm2fork-chaincode/internal/contract -run 'TestValidatePaymentInput|TestRecordPayment' -count=1`

  Expected: valid payment and existing idempotency cases pass; malformed payment input is rejected.

- [ ] **Step 6: Commit the payment validation slice**

  ```bash
  git add chaincode/farm2fork-chaincode/internal/contract/validation.go chaincode/farm2fork-chaincode/internal/contract/validation_test.go chaincode/farm2fork-chaincode/internal/contract/contract.go
  git commit -m "feat: validate Fabric payment records"
  ```

### Task 2: Enforce the supply-chain event allowlist and non-mutation guarantee

**Files:**

- Modify: `chaincode/farm2fork-chaincode/internal/contract/validation.go`
- Modify: `chaincode/farm2fork-chaincode/internal/contract/validation_test.go`
- Modify: `chaincode/farm2fork-chaincode/internal/contract/contract.go`
- Modify: `chaincode/farm2fork-chaincode/internal/contract/contract_test.go`

- [ ] **Step 1: Write failing supply-chain validator tests**

  Cover accepted pairs exactly:

  | Reference model | Event type | Actor role |
  | --- | --- | --- |
  | `Product` | `listed` | `farmer` |
  | `Shipment` | `shipment_assigned` | `transporter` |
  | `Shipment` | `shipment_picked_up` | `transporter` |
  | `Shipment` | `shipment_in_transit` | `transporter` |
  | `Shipment` | `shipment_delivered` | `transporter` |
  | `Shipment` | `shipment_failed` | `transporter` |

  Add rejected cases for unknown reference model, event type, actor role, invalid model/event/role pair, blank required payload fields, and malformed timestamp.

- [ ] **Step 2: Write a failing contract-level non-mutation test**

  Call `RecordSupplyChainEvent` with an invalid event and assert it returns an error. Query the ledger key plus its expected `reference~ledgerKey` and `product~ledgerKey` composite indexes, and assert all remain absent.

- [ ] **Step 3: Implement supply-chain validation**

  Validate and trim all required identifiers and location. Parse `timestamp` with `time.RFC3339`. Implement the reference-model/event-type/actor-role allowlist as explicit constants or a small map local to `validation.go`; do not infer permissions from user-supplied strings.

- [ ] **Step 4: Invoke validation before supply-chain idempotency lookup**

  Update `RecordSupplyChainEvent` so invalid arguments are rejected before `loadTransactionByLedgerKey` and `persistNewTransaction` can run.

- [ ] **Step 5: Run focused supply-chain and regression tests**

  Run: `go test ./chaincode/farm2fork-chaincode/internal/contract -run 'TestValidateSupplyChainInput|TestRecordSupplyChainEvent|TestGetTransactions' -count=1`

  Expected: all valid events preserve current lookup/history/idempotency behavior; rejected events create no state or indexes.

- [ ] **Step 6: Commit the supply-chain validation slice**

  ```bash
  git add chaincode/farm2fork-chaincode/internal/contract/validation.go chaincode/farm2fork-chaincode/internal/contract/validation_test.go chaincode/farm2fork-chaincode/internal/contract/contract.go chaincode/farm2fork-chaincode/internal/contract/contract_test.go
  git commit -m "feat: validate Fabric supply chain events"
  ```

### Task 3: Verify the deployed contract and document the boundary

**Files:**

- Modify: `README.md`

- [ ] **Step 1: Run static and complete chaincode validation**

  Run from `chaincode/farm2fork-chaincode`:

  ```bash
  gofmt -w internal/contract/validation.go internal/contract/validation_test.go internal/contract/contract.go internal/contract/contract_test.go
  go test ./... -count=1
  go vet ./...
  ```

  Expected: formatting is clean; all unit tests and static checks pass.

- [ ] **Step 2: Run the Docker-based local Fabric smoke flow**

  Run from the repository root:

  ```bash
  SMOKE_RESET_NETWORK=true bash scripts/smoke-test.sh
  ```

  Expected: the network bootstraps locally, writes the valid payment and supply-chain smoke events, and the reference/product lookups return them. This does not contact Atlas, Firebase, or payment gateways.

- [ ] **Step 3: Document the enforced boundary**

  Add a concise README section covering required payment fields, permitted payment gateways, permitted supply-chain model/event/role combinations, RFC3339 timestamp requirement, and the fact that backend authentication/ownership checks remain outside Fabric chaincode.

- [ ] **Step 4: Review the final diff and commit documentation**

  Run:

  ```bash
  git diff --check
  git status --short
  ```

  Commit only the README update:

  ```bash
  git add README.md
  git commit -m "docs: describe Fabric write validation"
  ```

### Task 4: Handoff

- [ ] **Step 1: Record verification evidence**

  Report the exact Go and local-network commands run, their outcomes, and any Docker prerequisite that prevents local smoke verification.

- [ ] **Step 2: Preserve branch boundaries**

  Leave the work committed on `feature/fabric-backend-integration-design`. Do not merge or push to `main`. Request explicit approval before publishing this feature branch, because this repository has no `develop` branch.


package contract_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/hyperledger/fabric-chaincode-go/shim"
	"github.com/hyperledger/fabric-chaincode-go/shimtest"
	contractapi "github.com/hyperledger/fabric-contract-api-go/contractapi"
	"github.com/hyperledger/fabric-protos-go/ledger/queryresult"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	"farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/contract"
	contractmodel "farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/model"
)

type mockTransactionContext struct {
	*contractapi.TransactionContext
}

type historyTrackingStub struct {
	*shimtest.MockStub
	history map[string][]*queryresult.KeyModification
}

type historyIterator struct {
	modifications []*queryresult.KeyModification
	index         int
}

func newHistoryTrackingStub(name string) *historyTrackingStub {
	return &historyTrackingStub{
		MockStub: shimtest.NewMockStub(name, nil),
		history:  make(map[string][]*queryresult.KeyModification),
	}
}

func newMockTransactionContext(txID string, channelID string) *mockTransactionContext {
	stub := newHistoryTrackingStub("farm2fork")
	stub.ChannelID = channelID
	stub.MockTransactionStart(txID)

	ctx := &contractapi.TransactionContext{}
	ctx.SetStub(stub)

	return &mockTransactionContext{TransactionContext: ctx}
}

func (s *historyTrackingStub) PutState(key string, value []byte) error {
	if err := s.MockStub.PutState(key, value); err != nil {
		return err
	}

	copiedValue := append([]byte(nil), value...)
	s.history[key] = append(s.history[key], &queryresult.KeyModification{
		TxId:      s.TxID,
		Value:     copiedValue,
		Timestamp: timestamppb.New(time.Unix(0, 0)),
		IsDelete:  false,
	})

	return nil
}

func (s *historyTrackingStub) GetHistoryForKey(key string) (shim.HistoryQueryIteratorInterface, error) {
	return &historyIterator{modifications: s.history[key]}, nil
}

func (i *historyIterator) HasNext() bool {
	return i.index < len(i.modifications)
}

func (i *historyIterator) Close() error {
	return nil
}

func (i *historyIterator) Next() (*queryresult.KeyModification, error) {
	modification := i.modifications[i.index]
	i.index++

	return modification, nil
}

func TestRecordPaymentPreservesExactMasterContextFieldNames(t *testing.T) {
	ctx := newMockTransactionContext("tx-payment-001", "farm2forkchannel")

	payload, err := (&contract.Farm2ForkContract{}).RecordPayment(
		ctx,
		"outbox-payment-001",
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
	var tx contractmodel.BlockchainTransaction
	require.NoError(t, json.Unmarshal([]byte(payload), &tx))
	require.Equal(t, "payment", tx.Type)
	require.Equal(t, "Payment", tx.ReferenceModel)
	require.Equal(t, "payment-001", tx.ReferenceID)
	stored, err := ctx.GetStub().GetState("outbox-payment-001")
	require.NoError(t, err)
	require.Contains(t, string(stored), `"referenceId":"payment-001"`)
	require.Equal(t, "tx-payment-001", tx.TxHash)
	require.Equal(t, uint64(0), tx.BlockNumber)
	require.Equal(t, "farm2forkchannel", tx.ChannelName)
	require.Equal(t, "confirmed", tx.Status)
	require.Equal(t, 0, tx.RetryCount)
	require.Equal(t, "2026-06-01T12:00:00Z", tx.CreatedAt)
	require.NotNil(t, tx.Payload.Payment)
	require.Nil(t, tx.Payload.SupplyChain)
	require.Equal(t, "order-001", tx.Payload.Payment.OrderID)
	require.Equal(t, "buyer-001", tx.Payload.Payment.BuyerID)
	require.Equal(t, "farmer-001", tx.Payload.Payment.FarmerID)
	require.Equal(t, 1500.0, tx.Payload.Payment.Amount)
	require.Equal(t, "PKR", tx.Payload.Payment.Currency)
	require.Equal(t, "stripe", tx.Payload.Payment.Gateway)
	require.Equal(t, "2026-06-01T12:00:00Z", tx.Payload.Payment.PaidAt)
}

func TestRecordSupplyChainEventPreservesExactMasterContextFieldNames(t *testing.T) {
	ctx := newMockTransactionContext("tx-supply-001", "farm2forkchannel")

	payload, err := (&contract.Farm2ForkContract{}).RecordSupplyChainEvent(
		ctx,
		"outbox-product-001-event-001",
		"product-001",
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
	var tx contractmodel.BlockchainTransaction
	require.NoError(t, json.Unmarshal([]byte(payload), &tx))
	require.Equal(t, "supply_chain_event", tx.Type)
	require.Equal(t, "Product", tx.ReferenceModel)
	require.Equal(t, "product-001", tx.ReferenceID)
	stored, err := ctx.GetStub().GetState("outbox-product-001-event-001")
	require.NoError(t, err)
	require.Contains(t, string(stored), `"referenceId":"product-001"`)
	require.Equal(t, "tx-supply-001", tx.TxHash)
	require.Equal(t, uint64(0), tx.BlockNumber)
	require.Equal(t, "farm2forkchannel", tx.ChannelName)
	require.Equal(t, "confirmed", tx.Status)
	require.Equal(t, 0, tx.RetryCount)
	require.Equal(t, "2026-06-01T12:05:00Z", tx.CreatedAt)
	require.Nil(t, tx.Payload.Payment)
	require.NotNil(t, tx.Payload.SupplyChain)
	require.Equal(t, "product-001", tx.Payload.SupplyChain.ProductID)
	require.Equal(t, "farmer-001", tx.Payload.SupplyChain.FarmerID)
	require.Equal(t, "listed", tx.Payload.SupplyChain.EventType)
	require.Equal(t, "Lahore", tx.Payload.SupplyChain.Location)
	require.Equal(t, "farmer-001", tx.Payload.SupplyChain.ActorID)
	require.Equal(t, "farmer", tx.Payload.SupplyChain.ActorRole)
	require.Equal(t, "2026-06-01T12:05:00Z", tx.Payload.SupplyChain.Timestamp)
}

func TestGetTransactionByReferenceIdReturnsStoredRecord(t *testing.T) {
	ctx := newMockTransactionContext("tx-supply-002", "farm2forkchannel")

	_, err := (&contract.Farm2ForkContract{}).RecordSupplyChainEvent(
		ctx,
		"outbox-product-002-event-001",
		"product-002",
		"Product",
		"product-002",
		"farmer-002",
		"listed",
		"Multan",
		"farmer-002",
		"farmer",
		"2026-06-01T12:10:00Z",
	)
	require.NoError(t, err)

	payload, err := (&contract.Farm2ForkContract{}).GetTransactionByLedgerKey(ctx, "outbox-product-002-event-001")
	require.NoError(t, err)
	var tx contractmodel.BlockchainTransaction
	require.NoError(t, json.Unmarshal([]byte(payload), &tx))
	require.Equal(t, "product-002", tx.ReferenceID)
	require.Equal(t, "product-002", tx.Payload.SupplyChain.ProductID)
	require.Equal(t, "Multan", tx.Payload.SupplyChain.Location)
}

func TestGetHistoryForKeyReturnsEntriesForStoredKey(t *testing.T) {
	ctx := newMockTransactionContext("tx-history-001", "farm2forkchannel")

	_, err := (&contract.Farm2ForkContract{}).RecordSupplyChainEvent(
		ctx,
		"outbox-product-003-event-001",
		"product-003",
		"Product",
		"product-003",
		"farmer-003",
		"listed",
		"Faisalabad",
		"farmer-003",
		"farmer",
		"2026-06-01T12:15:00Z",
	)
	require.NoError(t, err)

	payload, err := (&contract.Farm2ForkContract{}).GetHistoryForKey(ctx, "outbox-product-003-event-001")
	require.NoError(t, err)
	var history []contractmodel.BlockchainTransaction
	require.NoError(t, json.Unmarshal([]byte(payload), &history))
	require.Len(t, history, 1)
	require.Equal(t, "product-003", history[0].ReferenceID)
	require.Equal(t, "product-003", history[0].Payload.SupplyChain.ProductID)
}

func TestRecordSupplyChainEventReturnsExistingValueForSameLedgerKey(t *testing.T) {
	ctx := newMockTransactionContext("tx-shipment-001", "farm2forkchannel")
	contract := &contract.Farm2ForkContract{}

	first, err := contract.RecordSupplyChainEvent(
		ctx,
		"outbox-shipment-001-product-001",
		"shipment-001",
		"Shipment",
		"product-001",
		"farmer-001",
		"shipment_assigned",
		"Lahore, Punjab",
		"transporter-001",
		"transporter",
		"2026-08-11T12:00:00Z",
	)
	require.NoError(t, err)

	ctx.GetStub().(*historyTrackingStub).MockTransactionStart("tx-shipment-002")

	second, err := contract.RecordSupplyChainEvent(
		ctx,
		"outbox-shipment-001-product-001",
		"shipment-001",
		"Shipment",
		"product-001",
		"farmer-001",
		"shipment_assigned",
		"Lahore, Punjab",
		"transporter-001",
		"transporter",
		"2026-08-11T12:00:00Z",
	)
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func TestRecordSupplyChainEventRejectsDifferentPayloadForExistingLedgerKey(t *testing.T) {
	ctx := newMockTransactionContext("tx-shipment-conflict-001", "farm2forkchannel")
	contract := &contract.Farm2ForkContract{}

	_, err := contract.RecordSupplyChainEvent(
		ctx,
		"outbox-shipment-001-product-001",
		"shipment-001",
		"Shipment",
		"product-001",
		"farmer-001",
		"shipment_assigned",
		"Lahore, Punjab",
		"transporter-001",
		"transporter",
		"2026-08-11T12:00:00Z",
	)
	require.NoError(t, err)

	_, err = contract.RecordSupplyChainEvent(
		ctx,
		"outbox-shipment-001-product-001",
		"shipment-001",
		"Shipment",
		"product-002",
		"farmer-001",
		"shipment_assigned",
		"Lahore, Punjab",
		"transporter-001",
		"transporter",
		"2026-08-11T12:00:00Z",
	)
	require.ErrorContains(t, err, "different immutable content")
}

func TestRecordSupplyChainEventRejectsInvalidEventWithoutWritingStateOrIndexes(t *testing.T) {
	ctx := newMockTransactionContext("tx-invalid-supply-001", "farm2forkchannel")
	ledgerKey := "outbox-shipment-invalid-product-001"

	_, err := (&contract.Farm2ForkContract{}).RecordSupplyChainEvent(
		ctx,
		ledgerKey,
		"shipment-001",
		"Shipment",
		"product-001",
		"farmer-001",
		"shipment_completed",
		"Lahore, Punjab",
		"transporter-001",
		"transporter",
		"2026-08-11T12:00:00Z",
	)
	require.ErrorContains(t, err, "reference model")

	stored, err := ctx.GetStub().GetState(ledgerKey)
	require.NoError(t, err)
	require.Empty(t, stored)

	referenceIndexKey, err := ctx.GetStub().CreateCompositeKey("f2f.reference", []string{"Shipment", "shipment-001", ledgerKey})
	require.NoError(t, err)
	referenceIndex, err := ctx.GetStub().GetState(referenceIndexKey)
	require.NoError(t, err)
	require.Empty(t, referenceIndex)

	productIndexKey, err := ctx.GetStub().CreateCompositeKey("f2f.product", []string{"product-001", ledgerKey})
	require.NoError(t, err)
	productIndex, err := ctx.GetStub().GetState(productIndexKey)
	require.NoError(t, err)
	require.Empty(t, productIndex)
}

func TestGetTransactionsByReferenceReturnsEveryProductForShipment(t *testing.T) {
	ctx := newMockTransactionContext("tx-shipment-index-001", "farm2forkchannel")
	contract := &contract.Farm2ForkContract{}

	for _, productID := range []string{"product-apple", "product-mango"} {
		_, err := contract.RecordSupplyChainEvent(
			ctx,
			"outbox-shipment-001-"+productID,
			"shipment-001",
			"Shipment",
			productID,
			"farmer-001",
			"shipment_assigned",
			"Lahore, Punjab",
			"transporter-001",
			"transporter",
			"2026-08-11T12:00:00Z",
		)
		require.NoError(t, err)
	}

	payload, err := contract.GetTransactionsByReference(ctx, "Shipment", "shipment-001")
	require.NoError(t, err)
	var records []contractmodel.BlockchainTransaction
	require.NoError(t, json.Unmarshal([]byte(payload), &records))
	require.Len(t, records, 2)
	require.Equal(t, "product-apple", records[0].Payload.SupplyChain.ProductID)
	require.Equal(t, "product-mango", records[1].Payload.SupplyChain.ProductID)
}

func TestGetTransactionsByProductIdReturnsOnlyMatchingShipmentEvents(t *testing.T) {
	ctx := newMockTransactionContext("tx-product-index-001", "farm2forkchannel")
	contract := &contract.Farm2ForkContract{}

	for _, event := range []struct {
		ledgerKey string
		shipment  string
		productID string
		eventType string
	}{
		{"outbox-shipment-001-apple", "shipment-001", "product-apple", "shipment_assigned"},
		{"outbox-shipment-001-mango", "shipment-001", "product-mango", "shipment_assigned"},
		{"outbox-shipment-002-apple", "shipment-002", "product-apple", "shipment_in_transit"},
	} {
		_, err := contract.RecordSupplyChainEvent(
			ctx,
			event.ledgerKey,
			event.shipment,
			"Shipment",
			event.productID,
			"farmer-001",
			event.eventType,
			"Lahore, Punjab",
			"transporter-001",
			"transporter",
			"2026-08-11T12:00:00Z",
		)
		require.NoError(t, err)
	}

	payload, err := contract.GetTransactionsByProductId(ctx, "product-apple")
	require.NoError(t, err)
	var records []contractmodel.BlockchainTransaction
	require.NoError(t, json.Unmarshal([]byte(payload), &records))
	require.Len(t, records, 2)
	for _, record := range records {
		require.Equal(t, "product-apple", record.Payload.SupplyChain.ProductID)
	}
}

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
	var tx contractmodel.BlockchainTransaction
	require.NoError(t, json.Unmarshal([]byte(payload), &tx))
	require.Equal(t, "supply_chain_event", tx.Type)
	require.Equal(t, "Product", tx.ReferenceModel)
	require.Equal(t, "product-001:event-001", tx.ReferenceID)
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
		"product-002:event-001",
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

	payload, err := (&contract.Farm2ForkContract{}).GetTransactionByReferenceId(ctx, "product-002:event-001")
	require.NoError(t, err)
	var tx contractmodel.BlockchainTransaction
	require.NoError(t, json.Unmarshal([]byte(payload), &tx))
	require.Equal(t, "product-002:event-001", tx.ReferenceID)
	require.Equal(t, "product-002", tx.Payload.SupplyChain.ProductID)
	require.Equal(t, "Multan", tx.Payload.SupplyChain.Location)
}

func TestGetHistoryForKeyReturnsEntriesForStoredKey(t *testing.T) {
	ctx := newMockTransactionContext("tx-history-001", "farm2forkchannel")

	_, err := (&contract.Farm2ForkContract{}).RecordSupplyChainEvent(
		ctx,
		"product-003:event-001",
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

	payload, err := (&contract.Farm2ForkContract{}).GetHistoryForKey(ctx, "product-003:event-001")
	require.NoError(t, err)
	var history []contractmodel.BlockchainTransaction
	require.NoError(t, json.Unmarshal([]byte(payload), &history))
	require.Len(t, history, 1)
	require.Equal(t, "product-003:event-001", history[0].ReferenceID)
	require.Equal(t, "product-003", history[0].Payload.SupplyChain.ProductID)
}

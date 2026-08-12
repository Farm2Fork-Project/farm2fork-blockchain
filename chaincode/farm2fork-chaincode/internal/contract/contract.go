package contract

import (
	"encoding/json"
	"errors"
	"reflect"

	contractapi "github.com/hyperledger/fabric-contract-api-go/contractapi"

	"farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/model"
)

var errTransactionNotFound = errors.New("transaction not found")

const (
	referenceIndexObjectType = "f2f.reference"
	productIndexObjectType   = "f2f.product"
)

type Farm2ForkContract struct {
	contractapi.Contract
}

func requireTransactionContext(ctx contractapi.TransactionContextInterface) error {
	if ctx == nil || ctx.GetStub() == nil {
		return errors.New("transaction context is required")
	}

	return nil
}

func buildBaseTransaction(
	ctx contractapi.TransactionContextInterface,
	referenceID string,
	referenceModel string,
	transactionType string,
	createdAt string,
) *model.BlockchainTransaction {
	return &model.BlockchainTransaction{
		Type:           transactionType,
		ReferenceID:    referenceID,
		ReferenceModel: referenceModel,
		TxHash:         ctx.GetStub().GetTxID(),
		BlockNumber:    0,
		ChannelName:    ctx.GetStub().GetChannelID(),
		Status:         "confirmed",
		RetryCount:     0,
		CreatedAt:      createdAt,
	}
}

func persistNewTransaction(ctx contractapi.TransactionContextInterface, ledgerKey string, tx *model.BlockchainTransaction) error {
	bytes, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	if err := ctx.GetStub().PutState(ledgerKey, bytes); err != nil {
		return err
	}

	referenceIndexKey, err := ctx.GetStub().CreateCompositeKey(
		referenceIndexObjectType,
		[]string{tx.ReferenceModel, tx.ReferenceID, ledgerKey},
	)
	if err != nil {
		return err
	}
	if err := ctx.GetStub().PutState(referenceIndexKey, []byte{1}); err != nil {
		return err
	}

	if tx.Payload.SupplyChain == nil {
		return nil
	}

	productIndexKey, err := ctx.GetStub().CreateCompositeKey(
		productIndexObjectType,
		[]string{tx.Payload.SupplyChain.ProductID, ledgerKey},
	)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(productIndexKey, []byte{1})
}

func loadTransactionByLedgerKey(ctx contractapi.TransactionContextInterface, ledgerKey string) (*model.BlockchainTransaction, error) {
	bytes, err := ctx.GetStub().GetState(ledgerKey)
	if err != nil {
		return nil, err
	}
	if len(bytes) == 0 {
		return nil, errTransactionNotFound
	}

	var tx model.BlockchainTransaction
	if err := json.Unmarshal(bytes, &tx); err != nil {
		return nil, err
	}

	return &tx, nil
}

func ensureSameImmutableTransaction(existing *model.BlockchainTransaction, requested *model.BlockchainTransaction) error {
	if existing.Type != requested.Type ||
		existing.ReferenceID != requested.ReferenceID ||
		existing.ReferenceModel != requested.ReferenceModel ||
		existing.CreatedAt != requested.CreatedAt ||
		!reflect.DeepEqual(existing.Payload, requested.Payload) {
		return errors.New("ledger key already exists with different immutable content")
	}

	return nil
}

func loadHistory(ctx contractapi.TransactionContextInterface, referenceID string) ([]*model.BlockchainTransaction, error) {
	iter, err := ctx.GetStub().GetHistoryForKey(referenceID)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	history := make([]*model.BlockchainTransaction, 0)
	for iter.HasNext() {
		response, err := iter.Next()
		if err != nil {
			return nil, err
		}
		if len(response.Value) == 0 {
			continue
		}

		var tx model.BlockchainTransaction
		if err := json.Unmarshal(response.Value, &tx); err != nil {
			return nil, err
		}
		history = append(history, &tx)
	}

	return history, nil
}

func loadTransactionsForIndex(
	ctx contractapi.TransactionContextInterface,
	objectType string,
	attributes []string,
) ([]*model.BlockchainTransaction, error) {
	iter, err := ctx.GetStub().GetStateByPartialCompositeKey(objectType, attributes)
	if err != nil {
		return nil, err
	}
	defer iter.Close()

	transactions := make([]*model.BlockchainTransaction, 0)
	for iter.HasNext() {
		response, err := iter.Next()
		if err != nil {
			return nil, err
		}

		_, compositeKeys, err := ctx.GetStub().SplitCompositeKey(response.Key)
		if err != nil {
			return nil, err
		}
		if len(compositeKeys) == 0 {
			continue
		}

		ledgerKey := compositeKeys[len(compositeKeys)-1]
		tx, err := loadTransactionByLedgerKey(ctx, ledgerKey)
		if errors.Is(err, errTransactionNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

func (c *Farm2ForkContract) RecordPayment(
	ctx contractapi.TransactionContextInterface,
	ledgerKey string,
	referenceID string,
	orderID string,
	buyerID string,
	farmerID string,
	amount float64,
	currency string,
	gateway string,
	paidAt string,
) (string, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return "", err
	}

	ledgerKey, referenceID, payment, err := validatePaymentInput(ledgerKey, referenceID, model.PaymentPayload{
		OrderID:  orderID,
		BuyerID:  buyerID,
		FarmerID: farmerID,
		Amount:   amount,
		Currency: currency,
		Gateway:  gateway,
		PaidAt:   paidAt,
	})
	if err != nil {
		return "", err
	}

	tx := buildBaseTransaction(ctx, referenceID, "Payment", "payment", payment.PaidAt)
	tx.Payload.Payment = &payment

	existing, err := loadTransactionByLedgerKey(ctx, ledgerKey)
	if err == nil {
		if err := ensureSameImmutableTransaction(existing, tx); err != nil {
			return "", err
		}

		bytes, err := json.Marshal(existing)
		if err != nil {
			return "", err
		}

		return string(bytes), nil
	}
	if !errors.Is(err, errTransactionNotFound) {
		return "", err
	}

	if err := persistNewTransaction(ctx, ledgerKey, tx); err != nil {
		return "", err
	}

	bytes, err := json.Marshal(tx)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (c *Farm2ForkContract) RecordSupplyChainEvent(
	ctx contractapi.TransactionContextInterface,
	ledgerKey string,
	referenceID string,
	referenceModel string,
	productID string,
	farmerID string,
	eventType string,
	location string,
	actorID string,
	actorRole string,
	timestamp string,
) (string, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return "", err
	}

	ledgerKey, referenceID, referenceModel, supplyChain, err := validateSupplyChainInput(ledgerKey, referenceID, referenceModel, model.SupplyChainPayload{
		ProductID: productID,
		FarmerID:  farmerID,
		EventType: eventType,
		Location:  location,
		ActorID:   actorID,
		ActorRole: actorRole,
		Timestamp: timestamp,
	})
	if err != nil {
		return "", err
	}

	tx := buildBaseTransaction(ctx, referenceID, referenceModel, "supply_chain_event", supplyChain.Timestamp)
	tx.Payload.SupplyChain = &supplyChain

	existing, err := loadTransactionByLedgerKey(ctx, ledgerKey)
	if err == nil {
		if err := ensureSameImmutableTransaction(existing, tx); err != nil {
			return "", err
		}

		bytes, err := json.Marshal(existing)
		if err != nil {
			return "", err
		}

		return string(bytes), nil
	}
	if !errors.Is(err, errTransactionNotFound) {
		return "", err
	}

	if err := persistNewTransaction(ctx, ledgerKey, tx); err != nil {
		return "", err
	}

	bytes, err := json.Marshal(tx)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (c *Farm2ForkContract) GetTransactionByLedgerKey(
	ctx contractapi.TransactionContextInterface,
	ledgerKey string,
) (string, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return "", err
	}

	tx, err := loadTransactionByLedgerKey(ctx, ledgerKey)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(tx)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (c *Farm2ForkContract) GetTransactionsByReference(
	ctx contractapi.TransactionContextInterface,
	referenceModel string,
	referenceID string,
) (string, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return "", err
	}

	transactions, err := loadTransactionsForIndex(
		ctx,
		referenceIndexObjectType,
		[]string{referenceModel, referenceID},
	)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(transactions)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (c *Farm2ForkContract) GetTransactionsByProductId(
	ctx contractapi.TransactionContextInterface,
	productID string,
) (string, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return "", err
	}

	transactions, err := loadTransactionsForIndex(
		ctx,
		productIndexObjectType,
		[]string{productID},
	)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(transactions)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

func (c *Farm2ForkContract) GetHistoryForKey(
	ctx contractapi.TransactionContextInterface,
	referenceID string,
) (string, error) {
	if err := requireTransactionContext(ctx); err != nil {
		return "", err
	}

	history, err := loadHistory(ctx, referenceID)
	if err != nil {
		return "", err
	}

	bytes, err := json.Marshal(history)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

package contract

import (
	"errors"
	"math"
	"regexp"
	"strings"
	"time"

	"farm2fork-blockchain/chaincode/farm2fork-chaincode/internal/model"
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

func validatePaymentInput(
	ledgerKey string,
	referenceID string,
	payment model.PaymentPayload,
) (string, string, model.PaymentPayload, error) {
	ledgerKey = strings.TrimSpace(ledgerKey)
	if ledgerKey == "" {
		return "", "", model.PaymentPayload{}, errors.New("ledger key is required")
	}

	referenceID = strings.TrimSpace(referenceID)
	if referenceID == "" {
		return "", "", model.PaymentPayload{}, errors.New("reference ID is required")
	}

	payment.OrderID = strings.TrimSpace(payment.OrderID)
	if payment.OrderID == "" {
		return "", "", model.PaymentPayload{}, errors.New("payment order ID is required")
	}

	payment.BuyerID = strings.TrimSpace(payment.BuyerID)
	if payment.BuyerID == "" {
		return "", "", model.PaymentPayload{}, errors.New("payment buyer ID is required")
	}

	payment.FarmerID = strings.TrimSpace(payment.FarmerID)
	if payment.FarmerID == "" {
		return "", "", model.PaymentPayload{}, errors.New("payment farmer ID is required")
	}

	if payment.Amount <= 0 || math.IsNaN(payment.Amount) || math.IsInf(payment.Amount, 0) {
		return "", "", model.PaymentPayload{}, errors.New("payment amount must be a finite positive number")
	}

	payment.Currency = strings.TrimSpace(payment.Currency)
	if !currencyCodePattern.MatchString(payment.Currency) {
		return "", "", model.PaymentPayload{}, errors.New("payment currency must be a three-letter uppercase code")
	}

	payment.Gateway = strings.TrimSpace(payment.Gateway)
	if payment.Gateway != "stripe" && payment.Gateway != "jazzcash" {
		return "", "", model.PaymentPayload{}, errors.New("payment gateway must be stripe or jazzcash")
	}

	payment.PaidAt = strings.TrimSpace(payment.PaidAt)
	if _, err := time.Parse(time.RFC3339, payment.PaidAt); err != nil {
		return "", "", model.PaymentPayload{}, errors.New("payment paidAt must be an RFC3339 timestamp")
	}

	return ledgerKey, referenceID, payment, nil
}

func validateSupplyChainInput(
	ledgerKey string,
	referenceID string,
	referenceModel string,
	supplyChain model.SupplyChainPayload,
) (string, string, string, model.SupplyChainPayload, error) {
	ledgerKey = strings.TrimSpace(ledgerKey)
	if ledgerKey == "" {
		return "", "", "", model.SupplyChainPayload{}, errors.New("ledger key is required")
	}

	referenceID = strings.TrimSpace(referenceID)
	if referenceID == "" {
		return "", "", "", model.SupplyChainPayload{}, errors.New("reference ID is required")
	}

	referenceModel = strings.TrimSpace(referenceModel)
	if referenceModel != "Product" && referenceModel != "Shipment" {
		return "", "", "", model.SupplyChainPayload{}, errors.New("reference model must be Product or Shipment")
	}

	supplyChain.ProductID = strings.TrimSpace(supplyChain.ProductID)
	if supplyChain.ProductID == "" {
		return "", "", "", model.SupplyChainPayload{}, errors.New("supply-chain product ID is required")
	}

	supplyChain.FarmerID = strings.TrimSpace(supplyChain.FarmerID)
	if supplyChain.FarmerID == "" {
		return "", "", "", model.SupplyChainPayload{}, errors.New("supply-chain farmer ID is required")
	}

	supplyChain.EventType = strings.TrimSpace(supplyChain.EventType)
	if supplyChain.EventType == "" {
		return "", "", "", model.SupplyChainPayload{}, errors.New("supply-chain event type is required")
	}

	supplyChain.Location = strings.TrimSpace(supplyChain.Location)
	if supplyChain.Location == "" {
		return "", "", "", model.SupplyChainPayload{}, errors.New("supply-chain location is required")
	}

	supplyChain.ActorID = strings.TrimSpace(supplyChain.ActorID)
	if supplyChain.ActorID == "" {
		return "", "", "", model.SupplyChainPayload{}, errors.New("supply-chain actor ID is required")
	}

	supplyChain.ActorRole = strings.TrimSpace(supplyChain.ActorRole)
	if supplyChain.ActorRole == "" {
		return "", "", "", model.SupplyChainPayload{}, errors.New("supply-chain actor role is required")
	}

	supplyChain.Timestamp = strings.TrimSpace(supplyChain.Timestamp)
	if _, err := time.Parse(time.RFC3339, supplyChain.Timestamp); err != nil {
		return "", "", "", model.SupplyChainPayload{}, errors.New("supply-chain timestamp must be an RFC3339 timestamp")
	}

	if !isAllowedSupplyChainEvent(referenceModel, supplyChain.EventType, supplyChain.ActorRole) {
		return "", "", "", model.SupplyChainPayload{}, errors.New("reference model, event type, and actor role combination is not supported")
	}

	return ledgerKey, referenceID, referenceModel, supplyChain, nil
}

func isAllowedSupplyChainEvent(referenceModel string, eventType string, actorRole string) bool {
	if referenceModel == "Product" {
		return eventType == "listed" && actorRole == "farmer"
	}

	if referenceModel == "Shipment" && actorRole == "transporter" {
		switch eventType {
		case "shipment_assigned", "shipment_picked_up", "shipment_in_transit", "shipment_delivered", "shipment_failed":
			return true
		}
	}

	return false
}

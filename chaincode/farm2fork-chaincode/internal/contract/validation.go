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

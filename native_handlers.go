package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

func GET_payments_links(w http.ResponseWriter, r *http.Request) {
	AppContext.mu.RLock()
	links := make([]*PaymentLink, 0, len(AppContext.PaymentLinkOrder))
	for _, id := range AppContext.PaymentLinkOrder {
		if pl, ok := AppContext.PaymentLinks[id]; ok {
			links = append(links, pl)
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"payment_links": links,
	})
}

func POST_payments_links(w http.ResponseWriter, r *http.Request) {
	params, err := parseRequestParams(r)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", "", "Invalid request body", "")
		return
	}

	amount, hasAmount := params.GetInt64("amount")
	currency := params.GetString("currency")
	if !hasAmount || amount <= 0 {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_invalid_integer", "", "Amount must be a positive integer", "amount")
		return
	}
	if currency == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "", "Missing required param: currency", "currency")
		return
	}

	id := params.GetString("id")
	title := params.GetString("title")
	description := params.GetString("description")
	active := true
	if b, ok := params.GetBool("active"); ok {
		active = b
	}
	callbackUrl := params.GetString("callback_url")
	successUrl := params.GetString("success_url")
	cancelUrl := params.GetString("cancel_url")
	webhookUrl := params.GetString("webhook_url")
	metadata := params.GetMetadata()

	pl := NewPaymentLinkObject(id, title, description, amount, currency, active, callbackUrl, successUrl, cancelUrl, webhookUrl, metadata)

	AppContext.mu.Lock()
	if _, exists := AppContext.PaymentLinks[pl.Id]; !exists {
		AppContext.PaymentLinkOrder = append(AppContext.PaymentLinkOrder, pl.Id)
	}
	AppContext.PaymentLinks[pl.Id] = pl
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusCreated, pl)
}

func GET_payments_links_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pl, exists := AppContext.PaymentLinks[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_link: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, pl)
}

func POST_payments_intents(w http.ResponseWriter, r *http.Request) {
	params, err := parseRequestParams(r)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", "", "Invalid request body", "")
		return
	}

	amount, hasAmount := params.GetInt64("amount")
	currency := params.GetString("currency")
	if !hasAmount || amount <= 0 {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_invalid_integer", "", "Amount must be a positive integer", "amount")
		return
	}
	if currency == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "", "Missing required param: currency", "currency")
		return
	}

	description := params.GetString("description")
	customer := params.GetString("customer_id")
	if customer == "" {
		customer = params.GetString("customer")
	}
	customerEmail := params.GetString("customer_email")
	receiptEmail := params.GetString("receipt_email")
	paymentMethod := params.GetString("payment_method")
	cardNumber := params.GetString("card_number")
	confirm, _ := params.GetBool("confirm")
	captureMethod := params.GetString("capture_method")
	callbackUrl := params.GetString("callback_url")
	successUrl := params.GetString("success_url")
	cancelUrl := params.GetString("cancel_url")
	webhookUrl := params.GetString("webhook_url")
	metadata := params.GetMetadata()

	reqId := getRequestId(r)

	pi := createPaymentIntentObject(
		amount,
		currency,
		description,
		customer,
		customerEmail,
		receiptEmail,
		paymentMethod,
		[]string{"card"},
		true,
		captureMethod,
		"automatic",
		"",
		callbackUrl,
		callbackUrl,
		successUrl,
		cancelUrl,
		webhookUrl,
		"",
		metadata,
		true,
	)

	AppContext.mu.Lock()
	AppContext.PaymentIntents[pi.Id] = pi
	AppContext.PaymentIntentOrder = append(AppContext.PaymentIntentOrder, pi.Id)
	AppContext.mu.Unlock()

	emitAndDeliverEvent("payment_intent.created", pi, pi.Id, reqId, webhookUrl)

	if confirm {
		charge, payErr := confirmPaymentIntentInternal(pi, paymentMethod, cardNumber, 12, 2030, "", customerEmail, callbackUrl, captureMethod, reqId)
		if payErr != nil {
			emitAndDeliverEvent("payment_intent.payment_failed", pi, pi.Id, reqId, webhookUrl)
			writePaymentErrorWithIntent(w, http.StatusPaymentRequired, payErr, pi)
			return
		}
		emitAndDeliverEvent("payment_intent.succeeded", pi, pi.Id, reqId, webhookUrl)
		emitAndDeliverEvent("charge.succeeded", charge, pi.Id, reqId, webhookUrl)
	}

	writeJSON(w, http.StatusCreated, pi)
}

func GET_payments_intents(w http.ResponseWriter, r *http.Request) {
	statusFilter := r.URL.Query().Get("status")
	customerFilter := r.URL.Query().Get("customer")
	linkFilter := r.URL.Query().Get("payment_link_id")

	AppContext.mu.RLock()
	intents := make([]*PaymentIntent, 0)
	for _, id := range AppContext.PaymentIntentOrder {
		pi, ok := AppContext.PaymentIntents[id]
		if !ok {
			continue
		}
		if statusFilter != "" && pi.Status != statusFilter {
			continue
		}
		if customerFilter != "" {
			cusStr, _ := pi.Customer.(string)
			if cusStr != customerFilter {
				continue
			}
		}
		if linkFilter != "" && pi.PaymentLinkId != linkFilter {
			continue
		}
		intents = append(intents, pi)
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"payment_intents": intents,
	})
}

func GET_payments_intents_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pi, exists := AppContext.PaymentIntents[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_intent: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, pi)
}

func POST_payments_intents_id_confirm(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pi, exists := AppContext.PaymentIntents[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_intent: '%s'", id), "id")
		return
	}

	if pi.Status == "succeeded" || pi.Status == "canceled" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "payment_intent_unexpected_state", "", fmt.Sprintf("This PaymentIntent cannot be confirmed because its status is %s.", pi.Status), "")
		return
	}

	params, _ := parseRequestParams(r)
	paymentMethod := params.GetString("payment_method")
	cardNumber := params.GetString("card_number")
	if cardNumber == "" {
		cardNumber = params.GetString("payment_method_data[card][number]")
	}
	expMonth := 12
	if m, ok := params.GetInt64("exp_month"); ok && m > 0 {
		expMonth = int(m)
	}
	expYear := 2030
	if y, ok := params.GetInt64("exp_year"); ok && y > 0 {
		expYear = int(y)
	}
	customerEmail := params.GetString("customer_email")
	if customerEmail == "" {
		customerEmail = params.GetString("receipt_email")
	}
	returnUrl := params.GetString("return_url")
	captureMethod := params.GetString("capture_method")

	reqId := getRequestId(r)
	charge, payErr := confirmPaymentIntentInternal(pi, paymentMethod, cardNumber, expMonth, expYear, "", customerEmail, returnUrl, captureMethod, reqId)
	if payErr != nil {
		emitAndDeliverEvent("payment_intent.payment_failed", pi, pi.Id, reqId, pi.WebhookUrl)
		writePaymentErrorWithIntent(w, http.StatusPaymentRequired, payErr, pi)
		return
	}

	emitAndDeliverEvent("payment_intent.succeeded", pi, pi.Id, reqId, pi.WebhookUrl)
	emitAndDeliverEvent("charge.succeeded", charge, pi.Id, reqId, pi.WebhookUrl)

	writeJSON(w, http.StatusOK, pi)
}

func POST_payments_intents_id_cancel(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pi, exists := AppContext.PaymentIntents[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_intent: '%s'", id), "id")
		return
	}

	if pi.Status == "succeeded" || pi.Status == "canceled" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "payment_intent_unexpected_state", "", fmt.Sprintf("This PaymentIntent cannot be canceled because its status is %s.", pi.Status), "")
		return
	}

	params, _ := parseRequestParams(r)
	reason := params.GetString("cancellation_reason")
	now := time.Now().Unix()

	AppContext.mu.Lock()
	pi.Status = "canceled"
	pi.CanceledAt = &now
	if reason != "" {
		pi.CancellationReason = &reason
	}
	AppContext.mu.Unlock()

	reqId := getRequestId(r)
	emitAndDeliverEvent("payment_intent.canceled", pi, pi.Id, reqId, pi.WebhookUrl)

	writeJSON(w, http.StatusOK, pi)
}

func createRefundInternal(pi *PaymentIntent, ch *Charge, amount int64, reason string, metadata map[string]string, reqId string) (*Refund, error) {
	remaining := pi.AmountReceived - pi.AmountRefunded
	if amount == 0 {
		amount = remaining
	}
	if amount <= 0 || amount > remaining {
		return nil, fmt.Errorf("Refund amount (%d) exceeds remaining unrefunded amount (%d)", amount, remaining)
	}

	if metadata == nil {
		metadata = make(map[string]string)
	}

	var reasonVal interface{}
	if reason != "" {
		reasonVal = reason
	}

	chId := ""
	if ch != nil {
		chId = ch.Id
	} else if lc, ok := pi.LatestCharge.(string); ok {
		chId = lc
	}

	refId := generateID("re")
	refund := &Refund{
		Id:                 refId,
		Object:             "refund",
		Amount:             amount,
		BalanceTransaction: generateID("txn"),
		Charge:             chId,
		Created:            time.Now().Unix(),
		Currency:           pi.Currency,
		DestinationDetails: map[string]interface{}{
			"card": map[string]interface{}{
				"reference":        "123456789012",
				"reference_status": "available",
				"reference_type":   "acquirer_reference_number",
				"type":             "refund",
			},
			"type": "card",
		},
		Metadata:               cloneMetadata(metadata),
		PaymentIntent:          pi.Id,
		Reason:                 reasonVal,
		ReceiptNumber:          nil,
		SourceTransferReversal: nil,
		Status:                 "succeeded",
		TransferReversal:       nil,
	}

	AppContext.mu.Lock()
	pi.AmountRefunded += amount
	if ch != nil {
		ch.AmountRefunded += amount
		if ch.AmountRefunded >= ch.AmountCaptured {
			ch.Refunded = true
		}
	} else if chId != "" {
		if storedCh, ok := AppContext.Charges[chId]; ok {
			storedCh.AmountRefunded += amount
			if storedCh.AmountRefunded >= storedCh.AmountCaptured {
				storedCh.Refunded = true
			}
			ch = storedCh
		}
	}
	AppContext.Refunds[refund.Id] = refund
	AppContext.RefundOrder = append(AppContext.RefundOrder, refund.Id)
	AppContext.mu.Unlock()

	if ch != nil {
		emitAndDeliverEvent("charge.refunded", ch, pi.Id, reqId, pi.WebhookUrl)
	} else {
		emitAndDeliverEvent("charge.refunded", refund, pi.Id, reqId, pi.WebhookUrl)
	}

	return refund, nil
}

func POST_payments_intents_id_refund(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pi, exists := AppContext.PaymentIntents[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_intent: '%s'", id), "id")
		return
	}

	if pi.Status != "succeeded" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "charge_not_refundable", "", "Cannot refund a PaymentIntent that has not succeeded", "")
		return
	}

	params, _ := parseRequestParams(r)
	amount, _ := params.GetInt64("amount")
	reason := params.GetString("reason")
	metadata := params.GetMetadata()

	reqId := getRequestId(r)
	refund, err := createRefundInternal(pi, nil, amount, reason, metadata, reqId)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "refund_amount_invalid", "", err.Error(), "amount")
		return
	}

	writeJSON(w, http.StatusCreated, refund)
}

func GET_payments_refunds(w http.ResponseWriter, r *http.Request) {
	piFilter := r.URL.Query().Get("payment_intent")

	AppContext.mu.RLock()
	refunds := make([]*Refund, 0)
	for _, id := range AppContext.RefundOrder {
		ref, ok := AppContext.Refunds[id]
		if !ok {
			continue
		}
		if piFilter != "" && ref.PaymentIntent != piFilter {
			continue
		}
		refunds = append(refunds, ref)
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"refunds": refunds,
	})
}

func GET_payments_refunds_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	ref, exists := AppContext.Refunds[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such refund: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, ref)
}

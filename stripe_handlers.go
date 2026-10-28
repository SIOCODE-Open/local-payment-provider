package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

func parseLimitQuery(r *http.Request) int {
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		return 100
	}
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		return l
	}
	return 100
}

// ============================================================================
// Payment Intents (/v1/payment_intents)
// ============================================================================

func POST_v1_payment_intents(w http.ResponseWriter, r *http.Request) {
	params, err := parseRequestParams(r)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", "", "Invalid request parameters", "")
		return
	}

	amount, hasAmount := params.GetInt64("amount")
	currency := params.GetString("currency")
	if !hasAmount || amount <= 0 {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_invalid_integer", "", "Missing or invalid required parameter: amount", "amount")
		return
	}
	if currency == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "", "Missing required param: currency.", "currency")
		return
	}

	customer := params.GetString("customer")
	description := params.GetString("description")
	receiptEmail := params.GetString("receipt_email")
	customerEmail := params.GetString("customer_email")
	paymentMethod := params.GetString("payment_method")
	cardNumber := params.GetString("payment_method_data[card][number]")
	if cardNumber == "" {
		cardNumber = params.GetString("card_number")
	}
	pmTypes := params.GetStringSlice("payment_method_types")
	autoPmEnabled := true
	if b, ok := params.GetBool("automatic_payment_methods[enabled]"); ok {
		autoPmEnabled = b
	}
	confirm, _ := params.GetBool("confirm")
	captureMethod := params.GetString("capture_method")
	confirmationMethod := params.GetString("confirmation_method")
	setupFutureUsage := params.GetString("setup_future_usage")
	returnUrl := params.GetString("return_url")
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
		pmTypes,
		autoPmEnabled,
		captureMethod,
		confirmationMethod,
		setupFutureUsage,
		returnUrl,
		callbackUrl,
		successUrl,
		cancelUrl,
		webhookUrl,
		"",
		metadata,
		false,
	)

	AppContext.mu.Lock()
	AppContext.PaymentIntents[pi.Id] = pi
	AppContext.PaymentIntentOrder = append(AppContext.PaymentIntentOrder, pi.Id)
	AppContext.mu.Unlock()

	emitAndDeliverEvent("payment_intent.created", pi, pi.Id, reqId, webhookUrl)

	if confirm {
		charge, payErr := confirmPaymentIntentInternal(pi, paymentMethod, cardNumber, 12, 2030, "", receiptEmail, returnUrl, captureMethod, reqId)
		if payErr != nil {
			emitAndDeliverEvent("payment_intent.payment_failed", pi, pi.Id, reqId, webhookUrl)
			writePaymentErrorWithIntent(w, http.StatusPaymentRequired, payErr, pi)
			return
		}
		emitAndDeliverEvent("payment_intent.succeeded", pi, pi.Id, reqId, webhookUrl)
		emitAndDeliverEvent("charge.succeeded", charge, pi.Id, reqId, webhookUrl)
	}

	writeJSON(w, http.StatusOK, pi)
}

func GET_v1_payment_intents(w http.ResponseWriter, r *http.Request) {
	customerFilter := r.URL.Query().Get("customer")
	statusFilter := r.URL.Query().Get("status")
	limit := parseLimitQuery(r)

	AppContext.mu.RLock()
	data := make([]*PaymentIntent, 0)
	for _, id := range AppContext.PaymentIntentOrder {
		pi, ok := AppContext.PaymentIntents[id]
		if !ok {
			continue
		}
		if customerFilter != "" {
			cusStr, _ := pi.Customer.(string)
			if cusStr != customerFilter {
				continue
			}
		}
		if statusFilter != "" && pi.Status != statusFilter {
			continue
		}
		data = append(data, pi)
		if len(data) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, StripeListResponse{
		Object:  "list",
		Url:     "/v1/payment_intents",
		HasMore: false,
		Data:    data,
	})
}

func GET_v1_payment_intents_id(w http.ResponseWriter, r *http.Request) {
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

func POST_v1_payment_intents_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pi, exists := AppContext.PaymentIntents[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_intent: '%s'", id), "id")
		return
	}

	params, _ := parseRequestParams(r)

	AppContext.mu.Lock()
	if amt, ok := params.GetInt64("amount"); ok && amt > 0 {
		pi.Amount = amt
	}
	if cur := params.GetString("currency"); cur != "" {
		pi.Currency = strings.ToLower(cur)
	}
	if cus := params.GetString("customer"); cus != "" {
		pi.Customer = cus
	}
	if params.Has("description") {
		pi.Description = params.GetString("description")
	}
	if em := params.GetString("receipt_email"); em != "" {
		pi.ReceiptEmail = em
		pi.CustomerEmail = em
	}
	if pm := params.GetString("payment_method"); pm != "" {
		pi.PaymentMethod = pm
		if pi.Status == "requires_payment_method" {
			pi.Status = "requires_confirmation"
		}
	}
	if sfu := params.GetString("setup_future_usage"); sfu != "" {
		pi.SetupFutureUsage = sfu
	}
	for k, v := range params.GetMetadata() {
		if pi.Metadata == nil {
			pi.Metadata = make(map[string]string)
		}
		pi.Metadata[k] = v
	}
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, pi)
}

func POST_v1_payment_intents_id_confirm(w http.ResponseWriter, r *http.Request) {
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
	cardNumber := params.GetString("payment_method_data[card][number]")
	if cardNumber == "" {
		cardNumber = params.GetString("card_number")
	}
	receiptEmail := params.GetString("receipt_email")
	returnUrl := params.GetString("return_url")
	captureMethod := params.GetString("capture_method")

	reqId := getRequestId(r)
	charge, payErr := confirmPaymentIntentInternal(pi, paymentMethod, cardNumber, 12, 2030, "", receiptEmail, returnUrl, captureMethod, reqId)
	if payErr != nil {
		emitAndDeliverEvent("payment_intent.payment_failed", pi, pi.Id, reqId, pi.WebhookUrl)
		writePaymentErrorWithIntent(w, http.StatusPaymentRequired, payErr, pi)
		return
	}

	emitAndDeliverEvent("payment_intent.succeeded", pi, pi.Id, reqId, pi.WebhookUrl)
	emitAndDeliverEvent("charge.succeeded", charge, pi.Id, reqId, pi.WebhookUrl)

	writeJSON(w, http.StatusOK, pi)
}

func POST_v1_payment_intents_id_cancel(w http.ResponseWriter, r *http.Request) {
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

func POST_v1_payment_intents_id_capture(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pi, exists := AppContext.PaymentIntents[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_intent: '%s'", id), "id")
		return
	}

	if pi.Status != "requires_capture" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "payment_intent_unexpected_state", "", fmt.Sprintf("This PaymentIntent could not be captured because it has a status of %s. Only a PaymentIntent with one of the following statuses may be captured: requires_capture.", pi.Status), "")
		return
	}

	params, _ := parseRequestParams(r)
	amountToCapture := pi.AmountCapturable
	if amt, ok := params.GetInt64("amount_to_capture"); ok {
		if amt <= 0 || amt > pi.AmountCapturable {
			writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "amount_too_large", "", "Amount to capture cannot exceed amount_capturable", "amount_to_capture")
			return
		}
		amountToCapture = amt
	}

	AppContext.mu.Lock()
	pi.Status = "succeeded"
	pi.AmountReceived = amountToCapture
	pi.AmountCapturable = 0
	var capturedCharge *Charge
	if chId, ok := pi.LatestCharge.(string); ok {
		if ch, exists := AppContext.Charges[chId]; exists {
			ch.Captured = true
			ch.AmountCaptured = amountToCapture
			capturedCharge = ch
		}
	}
	AppContext.mu.Unlock()

	reqId := getRequestId(r)
	emitAndDeliverEvent("payment_intent.succeeded", pi, pi.Id, reqId, pi.WebhookUrl)
	if capturedCharge != nil {
		emitAndDeliverEvent("charge.succeeded", capturedCharge, pi.Id, reqId, pi.WebhookUrl)
	}

	writeJSON(w, http.StatusOK, pi)
}

// ============================================================================
// Payment Links (/v1/payment_links)
// ============================================================================

func extractLineItemsTotalAndDetails(params *RequestParams) (int64, string, string) {
	unitAmount, hasUnit := params.GetInt64("line_items[0][price_data][unit_amount]")
	currency := params.GetString("line_items[0][price_data][currency]")
	productName := params.GetString("line_items[0][price_data][product_data][name]")
	quantity, hasQty := params.GetInt64("line_items[0][quantity]")
	if !hasQty || quantity <= 0 {
		quantity = 1
	}

	var amount int64
	if hasUnit && unitAmount > 0 {
		amount = unitAmount * quantity
	} else if directAmt, ok := params.GetInt64("amount"); ok && directAmt > 0 {
		amount = directAmt * quantity
	}

	if currency == "" {
		currency = params.GetString("currency")
	}
	if productName == "" {
		productName = params.GetString("title")
	}

	return amount, currency, productName
}

func POST_v1_payment_links(w http.ResponseWriter, r *http.Request) {
	params, err := parseRequestParams(r)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", "", "Invalid request parameters", "")
		return
	}

	amount, currency, title := extractLineItemsTotalAndDetails(params)
	if amount <= 0 {
		amount = 2000 // fallback if price ID was passed without unit_amount
		if !params.Has("line_items[0][price]") && !params.Has("line_items") {
			writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "", "Missing required param: line_items or amount", "line_items")
			return
		}
	}
	if currency == "" {
		currency = "usd"
	}

	description := params.GetString("description")
	callbackUrl := params.GetString("callback_url")
	successUrl := params.GetString("success_url")
	cancelUrl := params.GetString("cancel_url")
	webhookUrl := params.GetString("webhook_url")
	afterCompletionRedirect := params.GetString("after_completion[redirect][url]")
	if afterCompletionRedirect != "" && successUrl == "" {
		successUrl = afterCompletionRedirect
	}
	metadata := params.GetMetadata()

	pl := NewPaymentLinkObject(
		params.GetString("id"),
		title,
		description,
		amount,
		currency,
		true,
		callbackUrl,
		successUrl,
		cancelUrl,
		webhookUrl,
		metadata,
	)

	if b, ok := params.GetBool("allow_promotion_codes"); ok {
		pl.AllowPromotionCodes = b
	}
	if bac := params.GetString("billing_address_collection"); bac != "" {
		pl.BillingAddressCollection = bac
	}
	if cc := params.GetString("customer_creation"); cc != "" {
		pl.CustomerCreation = cc
	}
	if st := params.GetString("submit_type"); st != "" {
		pl.SubmitType = st
	}
	if acType := params.GetString("after_completion[type]"); acType != "" {
		if acType == "redirect" && afterCompletionRedirect != "" {
			pl.AfterCompletion = map[string]interface{}{
				"type": "redirect",
				"redirect": map[string]interface{}{
					"url": afterCompletionRedirect,
				},
			}
		} else {
			pl.AfterCompletion = map[string]interface{}{
				"type": acType,
			}
		}
	}

	AppContext.mu.Lock()
	if _, exists := AppContext.PaymentLinks[pl.Id]; !exists {
		AppContext.PaymentLinkOrder = append(AppContext.PaymentLinkOrder, pl.Id)
	}
	AppContext.PaymentLinks[pl.Id] = pl
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, pl)
}

func GET_v1_payment_links(w http.ResponseWriter, r *http.Request) {
	activeStr := r.URL.Query().Get("active")
	hasActiveFilter := activeStr != ""
	activeFilter, _ := strconv.ParseBool(activeStr)
	limit := parseLimitQuery(r)

	AppContext.mu.RLock()
	data := make([]*PaymentLink, 0)
	for _, id := range AppContext.PaymentLinkOrder {
		pl, ok := AppContext.PaymentLinks[id]
		if !ok {
			continue
		}
		if hasActiveFilter && pl.Active != activeFilter {
			continue
		}
		data = append(data, pl)
		if len(data) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, StripeListResponse{
		Object:  "list",
		Url:     "/v1/payment_links",
		HasMore: false,
		Data:    data,
	})
}

func GET_v1_payment_links_id(w http.ResponseWriter, r *http.Request) {
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

func POST_v1_payment_links_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pl, exists := AppContext.PaymentLinks[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_link: '%s'", id), "id")
		return
	}

	params, _ := parseRequestParams(r)

	AppContext.mu.Lock()
	if active, ok := params.GetBool("active"); ok {
		pl.Active = active
	}
	if redUrl := params.GetString("after_completion[redirect][url]"); redUrl != "" {
		pl.SuccessUrl = redUrl
		pl.AfterCompletion = map[string]interface{}{
			"type": "redirect",
			"redirect": map[string]interface{}{
				"url": redUrl,
			},
		}
	}
	if apc, ok := params.GetBool("allow_promotion_codes"); ok {
		pl.AllowPromotionCodes = apc
	}
	for k, v := range params.GetMetadata() {
		if pl.Metadata == nil {
			pl.Metadata = make(map[string]string)
		}
		pl.Metadata[k] = v
	}
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, pl)
}

// ============================================================================
// Checkout Sessions (/v1/checkout/sessions)
// ============================================================================

func POST_v1_checkout_sessions(w http.ResponseWriter, r *http.Request) {
	params, err := parseRequestParams(r)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", "", "Invalid request parameters", "")
		return
	}

	successUrl := params.GetString("success_url")
	if successUrl == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "", "Missing required param: success_url.", "success_url")
		return
	}
	cancelUrl := params.GetString("cancel_url")
	mode := params.GetString("mode")
	if mode == "" {
		mode = "payment"
	}

	amount, currency, title := extractLineItemsTotalAndDetails(params)
	if amount <= 0 {
		amount = 2000
	}
	if currency == "" {
		currency = "usd"
	} else {
		currency = strings.ToLower(currency)
	}
	if title == "" {
		title = "Checkout Session"
	}

	customer := params.GetString("customer")
	customerEmail := params.GetString("customer_email")
	clientRefId := params.GetString("client_reference_id")
	pmTypes := params.GetStringSlice("payment_method_types")
	if len(pmTypes) == 0 {
		pmTypes = []string{"card"}
	}
	submitType := params.GetString("submit_type")
	webhookUrl := params.GetString("webhook_url")
	metadata := params.GetMetadata()

	reqId := getRequestId(r)

	pi := createPaymentIntentObject(
		amount,
		currency,
		title,
		customer,
		customerEmail,
		customerEmail,
		"",
		pmTypes,
		true,
		"automatic",
		"automatic",
		"",
		successUrl,
		"",
		successUrl,
		cancelUrl,
		webhookUrl,
		"",
		metadata,
		false,
	)

	AppContext.mu.Lock()
	AppContext.PaymentIntents[pi.Id] = pi
	AppContext.PaymentIntentOrder = append(AppContext.PaymentIntentOrder, pi.Id)
	AppContext.mu.Unlock()

	emitAndDeliverEvent("payment_intent.created", pi, pi.Id, reqId, webhookUrl)

	csId := generateID("cs_test")
	now := time.Now().Unix()

	var customerVal interface{}
	if customer != "" {
		customerVal = customer
	}
	var customerEmailVal interface{}
	if customerEmail != "" {
		customerEmailVal = customerEmail
	}
	var clientRefVal interface{}
	if clientRefId != "" {
		clientRefVal = clientRefId
	}
	var submitTypeVal interface{}
	if submitType != "" {
		submitTypeVal = submitType
	}

	cs := &CheckoutSession{
		Id:                  csId,
		Object:              "checkout.session",
		AfterExpiration:     nil,
		AllowPromotionCodes: nil,
		AmountSubtotal:      amount,
		AmountTotal:         amount,
		AutomaticTax: map[string]interface{}{
			"enabled":   false,
			"liability": nil,
			"status":    nil,
		},
		BillingAddressCollection: nil,
		CancelUrl:                cancelUrl,
		ClientReferenceId:        clientRefVal,
		ClientSecret:             nil,
		Consent:                  nil,
		ConsentCollection:        nil,
		Created:                  now,
		Currency:                 currency,
		CustomFields:             []interface{}{},
		CustomText: map[string]interface{}{
			"after_submit":                nil,
			"shipping_address":            nil,
			"submit":                      nil,
			"terms_of_service_acceptance": nil,
		},
		Customer:         customerVal,
		CustomerCreation: "if_required",
		CustomerDetails: map[string]interface{}{
			"address":    nil,
			"email":      customerEmailVal,
			"name":       nil,
			"phone":      nil,
			"tax_exempt": "none",
			"tax_ids":    []interface{}{},
		},
		CustomerEmail: customerEmailVal,
		ExpiresAt:     now + 86400,
		Invoice:       nil,
		InvoiceCreation: map[string]interface{}{
			"enabled": false,
			"invoice_data": map[string]interface{}{
				"account_tax_ids":   nil,
				"custom_fields":     nil,
				"description":       nil,
				"footer":            nil,
				"issuer":            nil,
				"metadata":          map[string]string{},
				"rendering_options": nil,
			},
		},
		Livemode:                          false,
		Locale:                            nil,
		Metadata:                          cloneMetadata(metadata),
		Mode:                              mode,
		PaymentIntent:                     pi.Id,
		PaymentLink:                       nil,
		PaymentMethodCollection:           "always",
		PaymentMethodConfigurationDetails: nil,
		PaymentMethodOptions:              map[string]interface{}{},
		PaymentMethodTypes:                pmTypes,
		PaymentStatus:                     "unpaid",
		PhoneNumberCollection: map[string]interface{}{
			"enabled": false,
		},
		RecoveredFrom:             nil,
		SetupIntent:               nil,
		ShippingAddressCollection: nil,
		ShippingCost:              nil,
		ShippingDetails:           nil,
		ShippingOptions:           []interface{}{},
		Status:                    "open",
		SubmitType:                submitTypeVal,
		Subscription:              nil,
		SuccessUrl:                successUrl,
		TotalDetails: map[string]interface{}{
			"amount_discount": 0,
			"amount_shipping": 0,
			"amount_tax":      0,
		},
		UiMode:      "hosted",
		Url:         fmt.Sprintf("%s/checkout/session/%s", AppConfig.BaseUrl, csId),
		WebhookUrl:  webhookUrl,
		Title:       title,
		Description: params.GetString("description"),
	}

	AppContext.mu.Lock()
	AppContext.CheckoutSessions[cs.Id] = cs
	AppContext.SessionOrder = append(AppContext.SessionOrder, cs.Id)
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, cs)
}

func GET_v1_checkout_sessions(w http.ResponseWriter, r *http.Request) {
	customerFilter := r.URL.Query().Get("customer")
	piFilter := r.URL.Query().Get("payment_intent")
	statusFilter := r.URL.Query().Get("status")
	limit := parseLimitQuery(r)

	AppContext.mu.RLock()
	data := make([]*CheckoutSession, 0)
	for _, id := range AppContext.SessionOrder {
		cs, ok := AppContext.CheckoutSessions[id]
		if !ok {
			continue
		}
		if customerFilter != "" {
			cusStr, _ := cs.Customer.(string)
			if cusStr != customerFilter {
				continue
			}
		}
		if piFilter != "" {
			piStr, _ := cs.PaymentIntent.(string)
			if piStr != piFilter {
				continue
			}
		}
		if statusFilter != "" && cs.Status != statusFilter {
			continue
		}
		data = append(data, cs)
		if len(data) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, StripeListResponse{
		Object:  "list",
		Url:     "/v1/checkout/sessions",
		HasMore: false,
		Data:    data,
	})
}

func GET_v1_checkout_sessions_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	cs, exists := AppContext.CheckoutSessions[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such checkout.session: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, cs)
}

func POST_v1_checkout_sessions_id_expire(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	cs, exists := AppContext.CheckoutSessions[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such checkout.session: '%s'", id), "id")
		return
	}

	if cs.Status != "open" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "session_not_open", "", fmt.Sprintf("Only open Checkout Sessions can be expired. This session has status '%s'.", cs.Status), "")
		return
	}

	AppContext.mu.Lock()
	cs.Status = "expired"
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, cs)
}

// ============================================================================
// Customers (/v1/customers)
// ============================================================================

func POST_v1_customers(w http.ResponseWriter, r *http.Request) {
	params, err := parseRequestParams(r)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", "", "Invalid request parameters", "")
		return
	}

	id := params.GetString("id")
	email := params.GetString("email")
	name := params.GetString("name")
	phone := params.GetString("phone")
	description := params.GetString("description")
	balance, _ := params.GetInt64("balance")
	paymentMethod := params.GetString("payment_method")
	metadata := params.GetMetadata()

	cus := NewCustomerObject(id, email, name, phone, description, balance, metadata)
	if paymentMethod != "" {
		cus.InvoiceSettings["default_payment_method"] = paymentMethod
	}

	AppContext.mu.Lock()
	if _, exists := AppContext.Customers[cus.Id]; !exists {
		AppContext.CustomerOrder = append(AppContext.CustomerOrder, cus.Id)
	}
	AppContext.Customers[cus.Id] = cus
	if paymentMethod != "" {
		if pm, ok := AppContext.PaymentMethods[paymentMethod]; ok {
			pm.Customer = cus.Id
		}
	}
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, cus)
}

func GET_v1_customers(w http.ResponseWriter, r *http.Request) {
	emailFilter := r.URL.Query().Get("email")
	limit := parseLimitQuery(r)

	AppContext.mu.RLock()
	data := make([]*Customer, 0)
	for _, id := range AppContext.CustomerOrder {
		cus, ok := AppContext.Customers[id]
		if !ok {
			continue
		}
		if emailFilter != "" {
			emStr, _ := cus.Email.(string)
			if emStr != emailFilter {
				continue
			}
		}
		data = append(data, cus)
		if len(data) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, StripeListResponse{
		Object:  "list",
		Url:     "/v1/customers",
		HasMore: false,
		Data:    data,
	})
}

func GET_v1_customers_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	cus, exists := AppContext.Customers[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such customer: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, cus)
}

func POST_v1_customers_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	cus, exists := AppContext.Customers[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such customer: '%s'", id), "id")
		return
	}

	params, _ := parseRequestParams(r)

	AppContext.mu.Lock()
	if params.Has("email") {
		cus.Email = params.GetString("email")
	}
	if params.Has("name") {
		cus.Name = params.GetString("name")
	}
	if params.Has("phone") {
		cus.Phone = params.GetString("phone")
	}
	if params.Has("description") {
		cus.Description = params.GetString("description")
	}
	if bal, ok := params.GetInt64("balance"); ok {
		cus.Balance = bal
	}
	if ds := params.GetString("default_source"); ds != "" {
		cus.DefaultSource = ds
	}
	if dpm := params.GetString("invoice_settings[default_payment_method]"); dpm != "" {
		if cus.InvoiceSettings == nil {
			cus.InvoiceSettings = make(map[string]interface{})
		}
		cus.InvoiceSettings["default_payment_method"] = dpm
	}
	for k, v := range params.GetMetadata() {
		if cus.Metadata == nil {
			cus.Metadata = make(map[string]string)
		}
		cus.Metadata[k] = v
	}
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, cus)
}

func DELETE_v1_customers_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.Lock()
	_, exists := AppContext.Customers[id]
	if exists {
		delete(AppContext.Customers, id)
		newOrder := make([]string, 0, len(AppContext.CustomerOrder))
		for _, cid := range AppContext.CustomerOrder {
			if cid != id {
				newOrder = append(newOrder, cid)
			}
		}
		AppContext.CustomerOrder = newOrder
	}
	AppContext.mu.Unlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such customer: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"object":  "customer",
		"deleted": true,
	})
}

// ============================================================================
// Payment Methods (/v1/payment_methods)
// ============================================================================

func POST_v1_payment_methods(w http.ResponseWriter, r *http.Request) {
	params, err := parseRequestParams(r)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", "", "Invalid request parameters", "")
		return
	}

	pmType := params.GetString("type")
	if pmType == "" {
		pmType = "card"
	}

	cardNumber := params.GetString("card[number]")
	if cardNumber == "" {
		cardNumber = "4242424242424242"
	}
	expMonth := 12
	if m, ok := params.GetInt64("card[exp_month]"); ok && m > 0 {
		expMonth = int(m)
	}
	expYear := 2030
	if y, ok := params.GetInt64("card[exp_year]"); ok && y > 0 {
		expYear = int(y)
	}

	var billingName interface{}
	if n := params.GetString("billing_details[name]"); n != "" {
		billingName = n
	}
	var billingEmail interface{}
	if e := params.GetString("billing_details[email]"); e != "" {
		billingEmail = e
	}
	var billingPhone interface{}
	if p := params.GetString("billing_details[phone]"); p != "" {
		billingPhone = p
	}

	last4, brand := cardLast4AndBrand(cardNumber)
	metadata := params.GetMetadata()

	pm := &PaymentMethod{
		Id:             generateID("pm"),
		Object:         "payment_method",
		AllowRedisplay: "unspecified",
		BillingDetails: map[string]interface{}{
			"address": map[string]interface{}{
				"city":        nil,
				"country":     nil,
				"line1":       nil,
				"line2":       nil,
				"postal_code": nil,
				"state":       nil,
			},
			"email": billingEmail,
			"name":  billingName,
			"phone": billingPhone,
		},
		Card: map[string]interface{}{
			"brand": brand,
			"checks": map[string]interface{}{
				"address_line1_check":       nil,
				"address_postal_code_check": nil,
				"cvc_check":                 "pass",
			},
			"country":        "US",
			"display_brand":  brand,
			"exp_month":      expMonth,
			"exp_year":       expYear,
			"fingerprint":    "mToisGZ01V71BCos",
			"funding":        "credit",
			"generated_from": nil,
			"last4":          last4,
			"networks": map[string]interface{}{
				"available": []string{brand},
				"preferred": nil,
			},
			"three_d_secure_usage": map[string]interface{}{
				"supported": true,
			},
			"wallet": nil,
		},
		Created:    time.Now().Unix(),
		Customer:   nil,
		Livemode:   false,
		Metadata:   cloneMetadata(metadata),
		Type:       pmType,
		CardNumber: cardNumber,
	}

	AppContext.mu.Lock()
	AppContext.PaymentMethods[pm.Id] = pm
	AppContext.PaymentMethodOrder = append(AppContext.PaymentMethodOrder, pm.Id)
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, pm)
}

func GET_v1_payment_methods(w http.ResponseWriter, r *http.Request) {
	customerFilter := r.URL.Query().Get("customer")
	typeFilter := r.URL.Query().Get("type")
	limit := parseLimitQuery(r)

	AppContext.mu.RLock()
	data := make([]*PaymentMethod, 0)
	for _, id := range AppContext.PaymentMethodOrder {
		pm, ok := AppContext.PaymentMethods[id]
		if !ok {
			continue
		}
		if customerFilter != "" {
			cusStr, _ := pm.Customer.(string)
			if cusStr != customerFilter {
				continue
			}
		}
		if typeFilter != "" && pm.Type != typeFilter {
			continue
		}
		data = append(data, pm)
		if len(data) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, StripeListResponse{
		Object:  "list",
		Url:     "/v1/payment_methods",
		HasMore: false,
		Data:    data,
	})
}

func GET_v1_payment_methods_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pm, exists := AppContext.PaymentMethods[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such PaymentMethod: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, pm)
}

func POST_v1_payment_methods_id_attach(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	params, _ := parseRequestParams(r)
	customerId := params.GetString("customer")
	if customerId == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "", "Missing required param: customer.", "customer")
		return
	}

	AppContext.mu.Lock()
	pm, pmExists := AppContext.PaymentMethods[id]
	_, cusExists := AppContext.Customers[customerId]
	if pmExists && cusExists {
		pm.Customer = customerId
	}
	AppContext.mu.Unlock()

	if !pmExists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such PaymentMethod: '%s'", id), "id")
		return
	}
	if !cusExists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such customer: '%s'", customerId), "customer")
		return
	}

	writeJSON(w, http.StatusOK, pm)
}

func POST_v1_payment_methods_id_detach(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.Lock()
	pm, exists := AppContext.PaymentMethods[id]
	if exists {
		pm.Customer = nil
	}
	AppContext.mu.Unlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such PaymentMethod: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, pm)
}

// ============================================================================
// Charges (/v1/charges)
// ============================================================================

func GET_v1_charges(w http.ResponseWriter, r *http.Request) {
	piFilter := r.URL.Query().Get("payment_intent")
	customerFilter := r.URL.Query().Get("customer")
	limit := parseLimitQuery(r)

	AppContext.mu.RLock()
	data := make([]*Charge, 0)
	for _, id := range AppContext.ChargeOrder {
		ch, ok := AppContext.Charges[id]
		if !ok {
			continue
		}
		if piFilter != "" && ch.PaymentIntent != piFilter {
			continue
		}
		if customerFilter != "" {
			cusStr, _ := ch.Customer.(string)
			if cusStr != customerFilter {
				continue
			}
		}
		data = append(data, ch)
		if len(data) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, StripeListResponse{
		Object:  "list",
		Url:     "/v1/charges",
		HasMore: false,
		Data:    data,
	})
}

func GET_v1_charges_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	ch, exists := AppContext.Charges[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such charge: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, ch)
}

// ============================================================================
// Refunds (/v1/refunds)
// ============================================================================

func POST_v1_refunds(w http.ResponseWriter, r *http.Request) {
	params, err := parseRequestParams(r)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", "", "Invalid request parameters", "")
		return
	}

	piId := params.GetString("payment_intent")
	chargeId := params.GetString("charge")
	if piId == "" && chargeId == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "", "One of payment_intent or charge is required.", "payment_intent")
		return
	}

	var pi *PaymentIntent
	var ch *Charge

	AppContext.mu.RLock()
	if chargeId != "" {
		ch = AppContext.Charges[chargeId]
		if ch != nil && piId == "" {
			piId = ch.PaymentIntent
		}
	}
	if piId != "" {
		pi = AppContext.PaymentIntents[piId]
	}
	AppContext.mu.RUnlock()

	if chargeId != "" && ch == nil {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such charge: '%s'", chargeId), "charge")
		return
	}
	if pi == nil {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_intent: '%s'", piId), "payment_intent")
		return
	}

	if pi.Status != "succeeded" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "charge_not_refundable", "", "Cannot refund a PaymentIntent that has not succeeded.", "")
		return
	}

	amount, _ := params.GetInt64("amount")
	reason := params.GetString("reason")
	metadata := params.GetMetadata()

	reqId := getRequestId(r)
	refund, refErr := createRefundInternal(pi, ch, amount, reason, metadata, reqId)
	if refErr != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "refund_amount_invalid", "", refErr.Error(), "amount")
		return
	}

	writeJSON(w, http.StatusOK, refund)
}

func GET_v1_refunds(w http.ResponseWriter, r *http.Request) {
	piFilter := r.URL.Query().Get("payment_intent")
	chargeFilter := r.URL.Query().Get("charge")
	limit := parseLimitQuery(r)

	AppContext.mu.RLock()
	data := make([]*Refund, 0)
	for _, id := range AppContext.RefundOrder {
		ref, ok := AppContext.Refunds[id]
		if !ok {
			continue
		}
		if piFilter != "" && ref.PaymentIntent != piFilter {
			continue
		}
		if chargeFilter != "" && ref.Charge != chargeFilter {
			continue
		}
		data = append(data, ref)
		if len(data) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, StripeListResponse{
		Object:  "list",
		Url:     "/v1/refunds",
		HasMore: false,
		Data:    data,
	})
}

func GET_v1_refunds_id(w http.ResponseWriter, r *http.Request) {
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

// ============================================================================
// Events (/v1/events)
// ============================================================================

func GET_v1_events(w http.ResponseWriter, r *http.Request) {
	typeFilter := r.URL.Query().Get("type")
	limit := parseLimitQuery(r)

	AppContext.mu.RLock()
	data := make([]*StripeEvent, 0)
	for _, id := range AppContext.EventOrder {
		evt, ok := AppContext.Events[id]
		if !ok {
			continue
		}
		if typeFilter != "" && evt.Type != typeFilter {
			continue
		}
		data = append(data, evt)
		if len(data) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, StripeListResponse{
		Object:  "list",
		Url:     "/v1/events",
		HasMore: false,
		Data:    data,
	})
}

func GET_v1_events_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	evt, exists := AppContext.Events[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such event: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, evt)
}

// ============================================================================
// Webhook Endpoints (/v1/webhook_endpoints)
// ============================================================================

func POST_v1_webhook_endpoints(w http.ResponseWriter, r *http.Request) {
	params, err := parseRequestParams(r)
	if err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_request", "", "Invalid request parameters", "")
		return
	}

	urlStr := params.GetString("url")
	if urlStr == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "", "Missing required param: url.", "url")
		return
	}

	enabledEvents := params.GetStringSlice("enabled_events")
	description := params.GetString("description")
	secret := params.GetString("secret")
	metadata := params.GetMetadata()

	wh := NewWebhookEndpointObject("", urlStr, secret, description, enabledEvents, metadata)

	AppContext.mu.Lock()
	AppContext.WebhookEndpoints[wh.Id] = wh
	AppContext.WebhookOrder = append(AppContext.WebhookOrder, wh.Id)
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, wh)
}

func GET_v1_webhook_endpoints(w http.ResponseWriter, r *http.Request) {
	limit := parseLimitQuery(r)

	AppContext.mu.RLock()
	data := make([]*WebhookEndpoint, 0)
	for _, id := range AppContext.WebhookOrder {
		wh, ok := AppContext.WebhookEndpoints[id]
		if !ok {
			continue
		}
		data = append(data, wh)
		if len(data) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, StripeListResponse{
		Object:  "list",
		Url:     "/v1/webhook_endpoints",
		HasMore: false,
		Data:    data,
	})
}

func GET_v1_webhook_endpoints_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	wh, exists := AppContext.WebhookEndpoints[id]
	AppContext.mu.RUnlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such webhook_endpoint: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, wh)
}

func DELETE_v1_webhook_endpoints_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.Lock()
	_, exists := AppContext.WebhookEndpoints[id]
	if exists {
		delete(AppContext.WebhookEndpoints, id)
		newOrder := make([]string, 0, len(AppContext.WebhookOrder))
		for _, wid := range AppContext.WebhookOrder {
			if wid != id {
				newOrder = append(newOrder, wid)
			}
		}
		AppContext.WebhookOrder = newOrder
	}
	AppContext.mu.Unlock()

	if !exists {
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such webhook_endpoint: '%s'", id), "id")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":      id,
		"object":  "webhook_endpoint",
		"deleted": true,
	})
}

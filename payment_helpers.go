package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// evaluateCardPayment checks if the payment should fail based on testing config, payment_method token, or card number.
func evaluateCardPayment(paymentMethodInput, cardNumberInput string) *PaymentError {
	AppContext.mu.RLock()
	testingCfg := cloneTestingConfig(AppContext.Testing)
	var storedPmCardNumber string
	if paymentMethodInput != "" {
		if pm, exists := AppContext.PaymentMethods[paymentMethodInput]; exists {
			storedPmCardNumber = pm.CardNumber
		}
	}
	AppContext.mu.RUnlock()

	if testingCfg.AlwaysFail {
		code := testingCfg.AlwaysFailCode
		if code == "" {
			code = "processing_error"
		}
		msg := testingCfg.AlwaysFailMessage
		if msg == "" {
			msg = "Payment processing failed due to test configuration"
		}
		return &PaymentError{
			Type:        "card_error",
			Code:        code,
			DeclineCode: code,
			Message:     msg,
		}
	}

	// Check Stripe test tokens
	switch paymentMethodInput {
	case "pm_card_chargeDeclined":
		if rule, ok := testingCfg.DeclinedCards["4000000000000002"]; ok {
			return &PaymentError{
				Type:        "card_error",
				Code:        rule.Code,
				DeclineCode: rule.DeclineCode,
				Message:     rule.Message,
			}
		}
		return &PaymentError{
			Type:        "card_error",
			Code:        "card_declined",
			DeclineCode: "generic_decline",
			Message:     "Your card was declined.",
		}
	case "pm_card_chargeDeclinedInsufficientFunds":
		if rule, ok := testingCfg.DeclinedCards["4000000000009995"]; ok {
			return &PaymentError{
				Type:        "card_error",
				Code:        rule.Code,
				DeclineCode: rule.DeclineCode,
				Message:     rule.Message,
			}
		}
		return &PaymentError{
			Type:        "card_error",
			Code:        "card_declined",
			DeclineCode: "insufficient_funds",
			Message:     "Your card has insufficient funds.",
		}
	case "pm_card_chargeDeclinedExpiredCard":
		if rule, ok := testingCfg.DeclinedCards["4000000000000069"]; ok {
			return &PaymentError{
				Type:        "card_error",
				Code:        rule.Code,
				DeclineCode: rule.DeclineCode,
				Message:     rule.Message,
			}
		}
		return &PaymentError{
			Type:        "card_error",
			Code:        "expired_card",
			DeclineCode: "expired_card",
			Message:     "Your card has expired.",
		}
	case "pm_card_processingError":
		if rule, ok := testingCfg.DeclinedCards["4000000000000119"]; ok {
			return &PaymentError{
				Type:        "card_error",
				Code:        rule.Code,
				DeclineCode: rule.DeclineCode,
				Message:     rule.Message,
			}
		}
		return &PaymentError{
			Type:        "card_error",
			Code:        "processing_error",
			DeclineCode: "processing_error",
			Message:     "An error occurred while processing your card.",
		}
	case "pm_card_chargeDeclinedFraudulent":
		return &PaymentError{
			Type:        "card_error",
			Code:        "card_declined",
			DeclineCode: "fraudulent",
			Message:     "Your card was declined.",
		}
	}

	// Clean card number (strip spaces and dashes)
	cleanCard := strings.ReplaceAll(strings.ReplaceAll(cardNumberInput, " ", ""), "-", "")
	if cleanCard == "" && storedPmCardNumber != "" {
		cleanCard = strings.ReplaceAll(strings.ReplaceAll(storedPmCardNumber, " ", ""), "-", "")
	}

	if cleanCard != "" {
		if rule, exists := testingCfg.DeclinedCards[cleanCard]; exists {
			code := rule.Code
			if code == "" {
				code = "card_declined"
			}
			declineCode := rule.DeclineCode
			if declineCode == "" {
				declineCode = "generic_decline"
			}
			msg := rule.Message
			if msg == "" {
				msg = "Your card was declined."
			}
			return &PaymentError{
				Type:        "card_error",
				Code:        code,
				DeclineCode: declineCode,
				Message:     msg,
			}
		}
	}

	// Check if paymentMethodInput itself was configured in DeclinedCards
	if paymentMethodInput != "" {
		if rule, exists := testingCfg.DeclinedCards[paymentMethodInput]; exists {
			return &PaymentError{
				Type:        "card_error",
				Code:        rule.Code,
				DeclineCode: rule.DeclineCode,
				Message:     rule.Message,
			}
		}
	}

	return nil
}

func cardLast4AndBrand(cardNumber string) (string, string) {
	clean := strings.ReplaceAll(strings.ReplaceAll(cardNumber, " ", ""), "-", "")
	if len(clean) < 4 {
		clean = "4242424242424242"
	}
	last4 := clean[len(clean)-4:]
	brand := "visa"
	if strings.HasPrefix(clean, "5") {
		brand = "mastercard"
	} else if strings.HasPrefix(clean, "3") {
		brand = "amex"
	}
	return last4, brand
}

func createPaymentIntentObject(
	amount int64,
	currency string,
	description string,
	customer string,
	customerEmail string,
	receiptEmail string,
	paymentMethod string,
	paymentMethodTypes []string,
	autoPmEnabled bool,
	captureMethod string,
	confirmationMethod string,
	setupFutureUsage string,
	returnUrl string,
	callbackUrl string,
	successUrl string,
	cancelUrl string,
	webhookUrl string,
	paymentLinkId string,
	metadata map[string]string,
	isNative bool,
) *PaymentIntent {
	id := generateID("pi")
	secretSuffix := strings.TrimPrefix(generateID("sec"), "sec_")
	clientSecret := fmt.Sprintf("%s_secret_%s", id, secretSuffix)

	if currency == "" {
		currency = "usd"
	} else {
		currency = strings.ToLower(currency)
	}
	if len(paymentMethodTypes) == 0 {
		paymentMethodTypes = []string{"card"}
	}
	if captureMethod == "" {
		captureMethod = "automatic"
	}
	if confirmationMethod == "" {
		confirmationMethod = "automatic"
	}
	if metadata == nil {
		metadata = make(map[string]string)
	}

	if customerEmail == "" && receiptEmail != "" {
		customerEmail = receiptEmail
	}
	if receiptEmail == "" && customerEmail != "" {
		receiptEmail = customerEmail
	}

	var customerVal interface{}
	if customer != "" {
		customerVal = customer
	}
	var receiptEmailVal interface{}
	if receiptEmail != "" {
		receiptEmailVal = receiptEmail
	}
	var pmVal interface{}
	if paymentMethod != "" {
		pmVal = paymentMethod
	}
	var sfuVal interface{}
	if setupFutureUsage != "" {
		sfuVal = setupFutureUsage
	}

	status := "requires_payment_method"
	if paymentMethod != "" || isNative {
		status = "requires_confirmation"
	}

	return &PaymentIntent{
		Id:                   id,
		Object:               "payment_intent",
		Amount:               amount,
		AmountCapturable:     0,
		AmountReceived:       0,
		AmountRefunded:       0,
		Application:          nil,
		ApplicationFeeAmount: nil,
		AutomaticPaymentMethods: map[string]interface{}{
			"enabled": autoPmEnabled,
		},
		CanceledAt:         nil,
		CancellationReason: nil,
		CaptureMethod:      captureMethod,
		ClientSecret:       clientSecret,
		CheckoutUrl:        fmt.Sprintf("%s/checkout/intent/%s", AppConfig.BaseUrl, id),
		ConfirmationMethod: confirmationMethod,
		Created:            time.Now().Unix(),
		Currency:           currency,
		Customer:           customerVal,
		CustomerEmail:      customerEmail,
		Description:        description,
		Invoice:            nil,
		LastPaymentError:   nil,
		LatestCharge:       nil,
		Livemode:           false,
		Metadata:           cloneMetadata(metadata),
		NextAction:         nil,
		PaymentLinkId:      paymentLinkId,
		PaymentMethod:      pmVal,
		PaymentMethodOptions: map[string]interface{}{
			"card": map[string]interface{}{
				"request_three_d_secure": "automatic",
			},
		},
		PaymentMethodTypes:        paymentMethodTypes,
		Processing:                nil,
		ReceiptEmail:              receiptEmailVal,
		ReturnUrl:                 returnUrl,
		SetupFutureUsage:          sfuVal,
		Shipping:                  nil,
		StatementDescriptor:       nil,
		StatementDescriptorSuffix: nil,
		Status:                    status,
		TransferData:              nil,
		TransferGroup:             nil,
		CallbackUrl:               callbackUrl,
		SuccessUrl:                successUrl,
		CancelUrl:                 cancelUrl,
		WebhookUrl:                webhookUrl,
	}
}

func confirmPaymentIntentInternal(
	pi *PaymentIntent,
	paymentMethod string,
	cardNumber string,
	expMonth int,
	expYear int,
	cardholderName string,
	customerEmail string,
	returnUrl string,
	captureMethodOverride string,
	reqId string,
) (*Charge, *PaymentError) {
	if paymentMethod != "" {
		pi.PaymentMethod = paymentMethod
	}
	pmStr, _ := pi.PaymentMethod.(string)
	if pmStr == "" {
		pmStr = "pm_card_visa"
	}

	if customerEmail != "" {
		pi.CustomerEmail = customerEmail
		pi.ReceiptEmail = customerEmail
	}
	if returnUrl != "" {
		pi.ReturnUrl = returnUrl
	}
	if captureMethodOverride != "" {
		pi.CaptureMethod = captureMethodOverride
	}

	if expMonth == 0 {
		expMonth = 12
	}
	if expYear == 0 {
		expYear = 2030
	}

	payErr := evaluateCardPayment(pmStr, cardNumber)
	if payErr != nil {
		pi.Status = "requires_payment_method"
		pi.LastPaymentError = payErr
		return nil, payErr
	}

	pi.LastPaymentError = nil
	pi.PaymentMethod = pmStr

	last4, brand := cardLast4AndBrand(cardNumber)
	AppContext.mu.RLock()
	if pmObj, exists := AppContext.PaymentMethods[pmStr]; exists {
		if c, ok := pmObj.Card["last4"].(string); ok && c != "" {
			last4 = c
		}
		if b, ok := pmObj.Card["brand"].(string); ok && b != "" {
			brand = b
		}
		if em, ok := pmObj.Card["exp_month"].(int); ok && em > 0 {
			expMonth = em
		}
		if ey, ok := pmObj.Card["exp_year"].(int); ok && ey > 0 {
			expYear = ey
		}
	}
	AppContext.mu.RUnlock()

	captured := pi.CaptureMethod != "manual"
	amountCaptured := int64(0)
	if captured {
		pi.Status = "succeeded"
		pi.AmountReceived = pi.Amount
		pi.AmountCapturable = 0
		amountCaptured = pi.Amount
	} else {
		pi.Status = "requires_capture"
		pi.AmountReceived = 0
		pi.AmountCapturable = pi.Amount
	}

	chId := "ch_" + strings.TrimPrefix(pi.Id, "pi_")
	AppContext.mu.RLock()
	if _, exists := AppContext.Charges[chId]; exists {
		chId = generateID("ch")
	}
	AppContext.mu.RUnlock()

	var emailVal interface{}
	if pi.CustomerEmail != "" {
		emailVal = pi.CustomerEmail
	} else if pi.ReceiptEmail != nil {
		emailVal = pi.ReceiptEmail
	}

	var nameVal interface{}
	if cardholderName != "" {
		nameVal = cardholderName
	} else if cusId, ok := pi.Customer.(string); ok && cusId != "" {
		AppContext.mu.RLock()
		if cus, exists := AppContext.Customers[cusId]; exists && cus.Name != nil {
			nameVal = cus.Name
		}
		AppContext.mu.RUnlock()
	}

	var descVal interface{}
	if pi.Description != "" {
		descVal = pi.Description
	}

	charge := &Charge{
		Id:                   chId,
		Object:               "charge",
		Amount:               pi.Amount,
		AmountCaptured:       amountCaptured,
		AmountRefunded:       0,
		Application:          nil,
		ApplicationFee:       nil,
		ApplicationFeeAmount: nil,
		BalanceTransaction:   "txn_" + strings.TrimPrefix(chId, "ch_"),
		BillingDetails: map[string]interface{}{
			"address": map[string]interface{}{
				"city":        nil,
				"country":     nil,
				"line1":       nil,
				"line2":       nil,
				"postal_code": nil,
				"state":       nil,
			},
			"email": emailVal,
			"name":  nameVal,
			"phone": nil,
		},
		CalculatedStatementDescriptor: "LOCALPAYMENT",
		Captured:                      captured,
		Created:                       time.Now().Unix(),
		Currency:                      pi.Currency,
		Customer:                      pi.Customer,
		Description:                   descVal,
		Disputed:                      false,
		FailureBalanceTransaction:     nil,
		FailureCode:                   nil,
		FailureMessage:                nil,
		FraudDetails:                  map[string]interface{}{},
		Invoice:                       nil,
		Livemode:                      false,
		Metadata:                      cloneMetadata(pi.Metadata),
		OnBehalfOf:                    nil,
		Outcome: map[string]interface{}{
			"network_status": "approved_by_network",
			"reason":         nil,
			"risk_level":     "normal",
			"risk_score":     10,
			"seller_message": "Payment complete.",
			"type":           "authorized",
		},
		Paid:          true,
		PaymentIntent: pi.Id,
		PaymentMethod: pmStr,
		PaymentMethodDetails: map[string]interface{}{
			"card": map[string]interface{}{
				"brand": brand,
				"checks": map[string]interface{}{
					"address_line1_check":       nil,
					"address_postal_code_check": nil,
					"cvc_check":                 "pass",
				},
				"country":        "US",
				"exp_month":      expMonth,
				"exp_year":       expYear,
				"fingerprint":    "mToisGZ01V71BCos",
				"funding":        "credit",
				"installments":   nil,
				"last4":          last4,
				"mandate":        nil,
				"network":        brand,
				"three_d_secure": nil,
				"wallet":         nil,
			},
			"type": "card",
		},
		ReceiptEmail:              emailVal,
		ReceiptNumber:             nil,
		ReceiptUrl:                fmt.Sprintf("%s/receipts/%s", AppConfig.BaseUrl, chId),
		Refunded:                  false,
		Review:                    nil,
		Shipping:                  nil,
		Source:                    nil,
		SourceTransfer:            nil,
		StatementDescriptor:       nil,
		StatementDescriptorSuffix: nil,
		Status:                    "succeeded",
		TransferData:              nil,
		TransferGroup:             nil,
	}

	pi.LatestCharge = charge.Id

	AppContext.mu.Lock()
	AppContext.Charges[charge.Id] = charge
	AppContext.ChargeOrder = append(AppContext.ChargeOrder, charge.Id)
	AppContext.mu.Unlock()

	return charge, nil
}

type webhookTarget struct {
	Url    string
	Secret string
}

func matchesWebhookEvents(enabledEvents []string, eventType string) bool {
	if len(enabledEvents) == 0 {
		return true
	}
	for _, ev := range enabledEvents {
		if ev == "*" || ev == eventType {
			return true
		}
	}
	return false
}

// emitAndDeliverEvent creates a StripeEvent, stores it in AppContext, and delivers it to all matching webhook endpoints plus any extraWebhookUrls.
func emitAndDeliverEvent(eventType string, dataObject interface{}, paymentIntentId string, reqId string, extraWebhookUrls ...string) (StripeEvent, []WebhookDelivery) {
	evtId := generateID("evt")
	var reqIdVal interface{}
	if reqId != "" {
		reqIdVal = reqId
	}

	// Convert dataObject to map[string]interface{} via JSON roundtrip so it is an immutable snapshot
	var objMap map[string]interface{}
	if raw, err := json.Marshal(dataObject); err == nil {
		_ = json.Unmarshal(raw, &objMap)
	}

	evt := StripeEvent{
		Id:         evtId,
		Object:     "event",
		ApiVersion: "2023-10-16",
		Created:    time.Now().Unix(),
		Data: map[string]interface{}{
			"object": objMap,
		},
		Livemode:        false,
		PendingWebhooks: 0,
		Request: map[string]interface{}{
			"id":              reqIdVal,
			"idempotency_key": nil,
		},
		Type: eventType,
	}

	AppContext.mu.Lock()
	AppContext.Events[evt.Id] = &evt
	AppContext.EventOrder = append(AppContext.EventOrder, evt.Id)

	webhookDelayMs := AppContext.Testing.WebhookDelayMs

	var targets []webhookTarget
	seenUrls := make(map[string]bool)

	for _, whId := range AppContext.WebhookOrder {
		wh := AppContext.WebhookEndpoints[whId]
		if wh != nil && wh.Status == "enabled" && wh.Url != "" && matchesWebhookEvents(wh.EnabledEvents, eventType) {
			if !seenUrls[wh.Url] {
				seenUrls[wh.Url] = true
				sec := wh.Secret
				if sec == "" {
					sec = AppConfig.WebhookSecret
				}
				targets = append(targets, webhookTarget{Url: wh.Url, Secret: sec})
			}
		}
	}

	for _, extraUrl := range extraWebhookUrls {
		if extraUrl != "" && !seenUrls[extraUrl] {
			seenUrls[extraUrl] = true
			targets = append(targets, webhookTarget{Url: extraUrl, Secret: AppConfig.WebhookSecret})
		}
	}
	AppContext.mu.Unlock()

	if webhookDelayMs > 0 && len(targets) > 0 {
		time.Sleep(time.Duration(webhookDelayMs) * time.Millisecond)
	}

	var deliveries []WebhookDelivery
	for _, target := range targets {
		del := deliverWebhookToTarget(evt, target.Url, target.Secret, paymentIntentId)
		deliveries = append(deliveries, del)
	}

	return evt, deliveries
}

func deliverWebhookToTarget(evt StripeEvent, targetUrl, secret, paymentIntentId string) WebhookDelivery {
	if secret == "" {
		secret = AppConfig.WebhookSecret
	}

	bodyBytes, _ := json.Marshal(evt)
	ts := time.Now().Unix()

	// Stripe-Signature: t={unix_timestamp},v1={hmac_sha256_hex} over "{unix_timestamp}.{raw_body}"
	signedPayload := fmt.Sprintf("%d.%s", ts, string(bodyBytes))
	stripeMac := hmac.New(sha256.New, []byte(secret))
	stripeMac.Write([]byte(signedPayload))
	stripeSigHex := hex.EncodeToString(stripeMac.Sum(nil))
	stripeSigHeader := fmt.Sprintf("t=%d,v1=%s", ts, stripeSigHex)

	// X-Payment-Signature: sha256={hmac_sha256_hex} over "{raw_body}"
	payMac := hmac.New(sha256.New, []byte(secret))
	payMac.Write(bodyBytes)
	paySigHex := hex.EncodeToString(payMac.Sum(nil))
	paySigHeader := fmt.Sprintf("sha256=%s", paySigHex)

	reqHeaders := map[string]string{
		"Content-Type":        "application/json",
		"Stripe-Signature":    stripeSigHeader,
		"X-Payment-Signature": paySigHeader,
		"X-Webhook-Event":     evt.Type,
		"X-Webhook-Id":        evt.Id,
	}

	delivery := WebhookDelivery{
		Id:              generateID("whd"),
		EventId:         evt.Id,
		EventType:       evt.Type,
		PaymentIntentId: paymentIntentId,
		Url:             targetUrl,
		RequestHeaders:  reqHeaders,
		Payload:         evt,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
	}

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("POST", targetUrl, bytes.NewReader(bodyBytes))
	if err != nil {
		delivery.Status = "failed"
		delivery.Error = err.Error()
	} else {
		for k, v := range reqHeaders {
			req.Header.Set(k, v)
		}
		resp, doErr := client.Do(req)
		if doErr != nil {
			delivery.Status = "failed"
			delivery.Error = doErr.Error()
		} else {
			defer resp.Body.Close()
			respBytes, _ := io.ReadAll(resp.Body)
			delivery.ResponseStatusCode = resp.StatusCode
			delivery.ResponseBody = string(respBytes)
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				delivery.Status = "succeeded"
			} else {
				delivery.Status = "failed"
				delivery.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
			}
		}
	}

	AppContext.mu.Lock()
	AppContext.WebhookDeliveries = append(AppContext.WebhookDeliveries, delivery)
	AppContext.mu.Unlock()

	return delivery
}

func appendQueryParams(baseUrl string, params map[string]string, orderedKeys []string) string {
	u, err := url.Parse(baseUrl)
	if err != nil {
		sep := "?"
		if strings.Contains(baseUrl, "?") {
			sep = "&"
		}
		var parts []string
		for _, k := range orderedKeys {
			if v, ok := params[k]; ok && v != "" {
				parts = append(parts, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
			}
		}
		if len(parts) == 0 {
			return baseUrl
		}
		return baseUrl + sep + strings.Join(parts, "&")
	}

	q := u.Query()
	for _, k := range orderedKeys {
		if v, ok := params[k]; ok && v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

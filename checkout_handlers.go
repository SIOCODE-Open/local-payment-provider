package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

func GET_checkout_link_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pl, exists := AppContext.PaymentLinks[id]
	if !exists || !pl.Active {
		AppContext.mu.RUnlock()
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such active payment_link: '%s'", id), "id")
		return
	}
	title := pl.Title
	description := pl.Description
	amount := pl.Amount
	currency := pl.Currency
	callbackUrl := pl.CallbackUrl
	successUrl := pl.SuccessUrl
	cancelUrl := pl.CancelUrl
	AppContext.mu.RUnlock()

	if q := r.URL.Query().Get("callback_url"); q != "" {
		callbackUrl = q
	}
	if q := r.URL.Query().Get("success_url"); q != "" {
		successUrl = q
	}
	if q := r.URL.Query().Get("cancel_url"); q != "" {
		cancelUrl = q
	}
	state := r.URL.Query().Get("state")
	customerEmail := r.URL.Query().Get("customer_email")
	redirectOnError := r.URL.Query().Get("redirect_on_error")

	renderCheckoutForm(w, http.StatusOK, checkoutTemplateData{
		ResourceID:      id,
		Title:           title,
		Description:     description,
		Amount:          amount,
		Currency:        currency,
		FormAction:      fmt.Sprintf("/checkout/link/%s/submit", id),
		CallbackUrl:     callbackUrl,
		SuccessUrl:      successUrl,
		CancelUrl:       cancelUrl,
		State:           state,
		RedirectOnError: redirectOnError,
		CustomerEmail:   customerEmail,
	})
}

func POST_checkout_link_id_submit(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pl, exists := AppContext.PaymentLinks[id]
	if !exists || !pl.Active {
		AppContext.mu.RUnlock()
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such active payment_link: '%s'", id), "id")
		return
	}
	title := pl.Title
	description := pl.Description
	amount := pl.Amount
	currency := pl.Currency
	callbackUrl := pl.CallbackUrl
	successUrl := pl.SuccessUrl
	cancelUrl := pl.CancelUrl
	webhookUrl := pl.WebhookUrl
	metadata := cloneMetadata(pl.Metadata)
	var afterCompletionRedirectUrl string
	if pl.AfterCompletion != nil {
		if red, ok := pl.AfterCompletion["redirect"].(map[string]interface{}); ok {
			if u, ok := red["url"].(string); ok {
				afterCompletionRedirectUrl = u
			}
		}
	}
	AppContext.mu.RUnlock()

	params, _ := parseRequestParams(r)

	action := params.GetString("action")
	if action == "" {
		action = "pay"
	}
	cardNumber := params.GetString("card_number")
	expMonthStr := params.GetString("exp_month")
	expYearStr := params.GetString("exp_year")
	cvc := params.GetString("cvc")
	cardholderName := params.GetString("cardholder_name")
	customerEmail := params.GetString("customer_email")
	paymentMethod := params.GetString("payment_method")
	state := params.GetString("state")
	redirectOnError := params.GetString("redirect_on_error")

	if v := params.GetString("callback_url"); v != "" {
		callbackUrl = v
	}
	if v := params.GetString("success_url"); v != "" {
		successUrl = v
	}
	if v := params.GetString("cancel_url"); v != "" {
		cancelUrl = v
	}

	if cardNumber == "" && paymentMethod == "" {
		cardNumber = "4242424242424242"
	}

	reqId := getRequestId(r)

	pi := createPaymentIntentObject(
		amount,
		currency,
		title,
		"",
		customerEmail,
		customerEmail,
		paymentMethod,
		[]string{"card"},
		true,
		"automatic",
		"automatic",
		"",
		callbackUrl,
		callbackUrl,
		successUrl,
		cancelUrl,
		webhookUrl,
		id,
		metadata,
		true,
	)

	AppContext.mu.Lock()
	AppContext.PaymentIntents[pi.Id] = pi
	AppContext.PaymentIntentOrder = append(AppContext.PaymentIntentOrder, pi.Id)
	AppContext.mu.Unlock()

	if action == "cancel" {
		now := pi.Created
		reason := "requested_by_customer"
		pi.Status = "canceled"
		pi.CanceledAt = &now
		pi.CancellationReason = &reason

		emitAndDeliverEvent("payment_intent.canceled", pi, pi.Id, reqId, webhookUrl)

		targetUrl := cancelUrl
		if targetUrl == "" {
			targetUrl = callbackUrl
		}
		if targetUrl != "" {
			targetUrl = strings.ReplaceAll(targetUrl, "{CHECKOUT_SESSION_ID}", pi.Id)
			qParams := map[string]string{
				"status":            "canceled",
				"payment_intent_id": pi.Id,
				"payment_link_id":   id,
				"state":             state,
			}
			redirectTo := appendQueryParams(targetUrl, qParams, []string{"status", "payment_intent_id", "payment_link_id", "state"})
			http.Redirect(w, r, redirectTo, http.StatusFound)
			return
		}

		renderCheckoutResult(w, http.StatusOK, checkoutResultData{
			Heading:         "Payment Canceled",
			Status:          "canceled",
			Message:         "Your payment has been canceled.",
			PaymentIntentID: pi.Id,
			FormattedAmount: formatCurrencyAmount(amount, currency),
			UpperCurrency:   strings.ToUpper(currency),
		})
		return
	}

	expMonth, expYear := parseExpMonthYear(expMonthStr, expYearStr)
	charge, payErr := confirmPaymentIntentInternal(pi, paymentMethod, cardNumber, expMonth, expYear, cardholderName, customerEmail, callbackUrl, "", reqId)
	if payErr != nil {
		emitAndDeliverEvent("payment_intent.payment_failed", pi, pi.Id, reqId, webhookUrl)

		if redirectOnError == "true" && callbackUrl != "" {
			qParams := map[string]string{
				"status":            "failed",
				"payment_intent_id": pi.Id,
				"payment_link_id":   id,
				"error_code":        payErr.Code,
				"error_message":     payErr.Message,
				"state":             state,
			}
			redirectTo := appendQueryParams(callbackUrl, qParams, []string{"status", "payment_intent_id", "payment_link_id", "error_code", "error_message", "state"})
			http.Redirect(w, r, redirectTo, http.StatusFound)
			return
		}

		renderCheckoutForm(w, http.StatusPaymentRequired, checkoutTemplateData{
			ResourceID:      id,
			Title:           title,
			Description:     description,
			Amount:          amount,
			Currency:        currency,
			FormAction:      fmt.Sprintf("/checkout/link/%s/submit", id),
			CallbackUrl:     callbackUrl,
			SuccessUrl:      successUrl,
			CancelUrl:       cancelUrl,
			State:           state,
			RedirectOnError: redirectOnError,
			CustomerEmail:   customerEmail,
			CardholderName:  cardholderName,
			CardNumber:      cardNumber,
			ExpMonth:        expMonthStr,
			ExpYear:         expYearStr,
			Cvc:             cvc,
			PaymentMethod:   paymentMethod,
			Error:           payErr.Message,
		})
		return
	}

	emitAndDeliverEvent("payment_intent.succeeded", pi, pi.Id, reqId, webhookUrl)
	emitAndDeliverEvent("charge.succeeded", charge, pi.Id, reqId, webhookUrl)

	AppContext.mu.RLock()
	plSnapshot := AppContext.PaymentLinks[id]
	AppContext.mu.RUnlock()
	if plSnapshot != nil {
		emitAndDeliverEvent("payment_link.paid", plSnapshot, pi.Id, reqId, webhookUrl)
	}

	targetUrl := successUrl
	if targetUrl == "" {
		targetUrl = afterCompletionRedirectUrl
	}
	if targetUrl == "" {
		targetUrl = callbackUrl
	}

	if targetUrl != "" {
		targetUrl = strings.ReplaceAll(targetUrl, "{CHECKOUT_SESSION_ID}", pi.Id)
		qParams := map[string]string{
			"status":            "succeeded",
			"payment_intent_id": pi.Id,
			"payment_link_id":   id,
			"state":             state,
		}
		redirectTo := appendQueryParams(targetUrl, qParams, []string{"status", "payment_intent_id", "payment_link_id", "state"})
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}

	renderCheckoutResult(w, http.StatusOK, checkoutResultData{
		Heading:         "Payment Successful",
		Status:          "succeeded",
		Message:         "Thank you! Your payment has been processed.",
		PaymentIntentID: pi.Id,
		FormattedAmount: formatCurrencyAmount(amount, currency),
		UpperCurrency:   strings.ToUpper(currency),
	})
}

func GET_checkout_intent_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pi, exists := AppContext.PaymentIntents[id]
	if !exists {
		AppContext.mu.RUnlock()
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_intent: '%s'", id), "id")
		return
	}
	description := pi.Description
	amount := pi.Amount
	currency := pi.Currency
	callbackUrl := pi.CallbackUrl
	if callbackUrl == "" {
		callbackUrl = pi.ReturnUrl
	}
	successUrl := pi.SuccessUrl
	cancelUrl := pi.CancelUrl
	customerEmail := pi.CustomerEmail
	AppContext.mu.RUnlock()

	if q := r.URL.Query().Get("callback_url"); q != "" {
		callbackUrl = q
	}
	if q := r.URL.Query().Get("success_url"); q != "" {
		successUrl = q
	}
	if q := r.URL.Query().Get("cancel_url"); q != "" {
		cancelUrl = q
	}
	state := r.URL.Query().Get("state")
	redirectOnError := r.URL.Query().Get("redirect_on_error")
	if q := r.URL.Query().Get("customer_email"); q != "" {
		customerEmail = q
	}

	title := description
	if title == "" {
		title = "Complete Your Payment"
	}

	renderCheckoutForm(w, http.StatusOK, checkoutTemplateData{
		ResourceID:      id,
		Title:           title,
		Description:     description,
		Amount:          amount,
		Currency:        currency,
		FormAction:      fmt.Sprintf("/checkout/intent/%s/submit", id),
		CallbackUrl:     callbackUrl,
		SuccessUrl:      successUrl,
		CancelUrl:       cancelUrl,
		State:           state,
		RedirectOnError: redirectOnError,
		CustomerEmail:   customerEmail,
	})
}

func POST_checkout_intent_id_submit(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	pi, exists := AppContext.PaymentIntents[id]
	if !exists {
		AppContext.mu.RUnlock()
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such payment_intent: '%s'", id), "id")
		return
	}
	description := pi.Description
	amount := pi.Amount
	currency := pi.Currency
	callbackUrl := pi.CallbackUrl
	if callbackUrl == "" {
		callbackUrl = pi.ReturnUrl
	}
	successUrl := pi.SuccessUrl
	cancelUrl := pi.CancelUrl
	webhookUrl := pi.WebhookUrl
	AppContext.mu.RUnlock()

	params, _ := parseRequestParams(r)

	action := params.GetString("action")
	if action == "" {
		action = "pay"
	}
	cardNumber := params.GetString("card_number")
	expMonthStr := params.GetString("exp_month")
	expYearStr := params.GetString("exp_year")
	cvc := params.GetString("cvc")
	cardholderName := params.GetString("cardholder_name")
	customerEmail := params.GetString("customer_email")
	paymentMethod := params.GetString("payment_method")
	state := params.GetString("state")
	redirectOnError := params.GetString("redirect_on_error")

	if v := params.GetString("callback_url"); v != "" {
		callbackUrl = v
	}
	if v := params.GetString("success_url"); v != "" {
		successUrl = v
	}
	if v := params.GetString("cancel_url"); v != "" {
		cancelUrl = v
	}

	if cardNumber == "" && paymentMethod == "" {
		cardNumber = "4242424242424242"
	}

	reqId := getRequestId(r)
	title := description
	if title == "" {
		title = "Complete Your Payment"
	}

	if action == "cancel" {
		now := pi.Created
		reason := "requested_by_customer"
		pi.Status = "canceled"
		pi.CanceledAt = &now
		pi.CancellationReason = &reason

		emitAndDeliverEvent("payment_intent.canceled", pi, pi.Id, reqId, webhookUrl)

		targetUrl := cancelUrl
		if targetUrl == "" {
			targetUrl = callbackUrl
		}
		if targetUrl != "" {
			qParams := map[string]string{
				"status":            "canceled",
				"payment_intent_id": pi.Id,
				"state":             state,
			}
			redirectTo := appendQueryParams(targetUrl, qParams, []string{"status", "payment_intent_id", "state"})
			http.Redirect(w, r, redirectTo, http.StatusFound)
			return
		}

		renderCheckoutResult(w, http.StatusOK, checkoutResultData{
			Heading:         "Payment Canceled",
			Status:          "canceled",
			Message:         "Your payment has been canceled.",
			PaymentIntentID: pi.Id,
			FormattedAmount: formatCurrencyAmount(amount, currency),
			UpperCurrency:   strings.ToUpper(currency),
		})
		return
	}

	expMonth, expYear := parseExpMonthYear(expMonthStr, expYearStr)
	charge, payErr := confirmPaymentIntentInternal(pi, paymentMethod, cardNumber, expMonth, expYear, cardholderName, customerEmail, callbackUrl, "", reqId)
	if payErr != nil {
		emitAndDeliverEvent("payment_intent.payment_failed", pi, pi.Id, reqId, webhookUrl)

		targetUrl := callbackUrl
		if targetUrl == "" {
			targetUrl = cancelUrl
		}
		if redirectOnError == "true" && targetUrl != "" {
			qParams := map[string]string{
				"status":            "failed",
				"payment_intent_id": pi.Id,
				"error_code":        payErr.Code,
				"error_message":     payErr.Message,
				"state":             state,
			}
			redirectTo := appendQueryParams(targetUrl, qParams, []string{"status", "payment_intent_id", "error_code", "error_message", "state"})
			http.Redirect(w, r, redirectTo, http.StatusFound)
			return
		}

		renderCheckoutForm(w, http.StatusPaymentRequired, checkoutTemplateData{
			ResourceID:      id,
			Title:           title,
			Description:     description,
			Amount:          amount,
			Currency:        currency,
			FormAction:      fmt.Sprintf("/checkout/intent/%s/submit", id),
			CallbackUrl:     callbackUrl,
			SuccessUrl:      successUrl,
			CancelUrl:       cancelUrl,
			State:           state,
			RedirectOnError: redirectOnError,
			CustomerEmail:   customerEmail,
			CardholderName:  cardholderName,
			CardNumber:      cardNumber,
			ExpMonth:        expMonthStr,
			ExpYear:         expYearStr,
			Cvc:             cvc,
			PaymentMethod:   paymentMethod,
			Error:           payErr.Message,
		})
		return
	}

	emitAndDeliverEvent("payment_intent.succeeded", pi, pi.Id, reqId, webhookUrl)
	emitAndDeliverEvent("charge.succeeded", charge, pi.Id, reqId, webhookUrl)

	targetUrl := successUrl
	if targetUrl == "" {
		targetUrl = callbackUrl
	}
	if targetUrl != "" {
		qParams := map[string]string{
			"status":            "succeeded",
			"payment_intent_id": pi.Id,
			"state":             state,
		}
		redirectTo := appendQueryParams(targetUrl, qParams, []string{"status", "payment_intent_id", "state"})
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}

	renderCheckoutResult(w, http.StatusOK, checkoutResultData{
		Heading:         "Payment Successful",
		Status:          "succeeded",
		Message:         "Thank you! Your payment has been processed.",
		PaymentIntentID: pi.Id,
		FormattedAmount: formatCurrencyAmount(amount, currency),
		UpperCurrency:   strings.ToUpper(currency),
	})
}

func GET_checkout_session_id(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	cs, exists := AppContext.CheckoutSessions[id]
	if !exists {
		AppContext.mu.RUnlock()
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such checkout.session: '%s'", id), "id")
		return
	}
	title := cs.Title
	if title == "" {
		title = "Checkout Session"
	}
	description := cs.Description
	amount := cs.AmountTotal
	currency := cs.Currency
	successUrl := cs.SuccessUrl
	cancelUrl := cs.CancelUrl
	var customerEmail string
	if em, ok := cs.CustomerEmail.(string); ok {
		customerEmail = em
	}
	AppContext.mu.RUnlock()

	state := r.URL.Query().Get("state")

	renderCheckoutForm(w, http.StatusOK, checkoutTemplateData{
		ResourceID:    id,
		Title:         title,
		Description:   description,
		Amount:        amount,
		Currency:      currency,
		FormAction:    fmt.Sprintf("/checkout/session/%s/submit", id),
		SuccessUrl:    successUrl,
		CancelUrl:     cancelUrl,
		State:         state,
		CustomerEmail: customerEmail,
	})
}

func POST_checkout_session_id_submit(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	AppContext.mu.RLock()
	cs, exists := AppContext.CheckoutSessions[id]
	if !exists {
		AppContext.mu.RUnlock()
		writeStripeError(w, http.StatusNotFound, "invalid_request_error", "resource_missing", "", fmt.Sprintf("No such checkout.session: '%s'", id), "id")
		return
	}
	title := cs.Title
	if title == "" {
		title = "Checkout Session"
	}
	description := cs.Description
	amount := cs.AmountTotal
	currency := cs.Currency
	successUrl := cs.SuccessUrl
	cancelUrl := cs.CancelUrl
	webhookUrl := cs.WebhookUrl
	piId, _ := cs.PaymentIntent.(string)
	pi := AppContext.PaymentIntents[piId]
	AppContext.mu.RUnlock()

	params, _ := parseRequestParams(r)

	action := params.GetString("action")
	if action == "" {
		action = "pay"
	}
	cardNumber := params.GetString("card_number")
	expMonthStr := params.GetString("exp_month")
	expYearStr := params.GetString("exp_year")
	cvc := params.GetString("cvc")
	cardholderName := params.GetString("cardholder_name")
	customerEmail := params.GetString("customer_email")
	paymentMethod := params.GetString("payment_method")
	state := params.GetString("state")

	if cardNumber == "" && paymentMethod == "" {
		cardNumber = "4242424242424242"
	}

	reqId := getRequestId(r)

	if action == "cancel" {
		if cancelUrl != "" {
			qParams := map[string]string{
				"session_id": id,
				"status":     "canceled",
				"state":      state,
			}
			redirectTo := appendQueryParams(cancelUrl, qParams, []string{"session_id", "status", "state"})
			http.Redirect(w, r, redirectTo, http.StatusFound)
			return
		}
		renderCheckoutResult(w, http.StatusOK, checkoutResultData{
			Heading:         "Checkout Canceled",
			Status:          "canceled",
			Message:         "Your checkout session has been canceled.",
			PaymentIntentID: piId,
			FormattedAmount: formatCurrencyAmount(amount, currency),
			UpperCurrency:   strings.ToUpper(currency),
		})
		return
	}

	if pi == nil {
		pi = createPaymentIntentObject(
			amount,
			currency,
			title,
			"",
			customerEmail,
			customerEmail,
			paymentMethod,
			[]string{"card"},
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
			cs.Metadata,
			false,
		)
		piId = pi.Id
		cs.PaymentIntent = pi.Id
		AppContext.mu.Lock()
		AppContext.PaymentIntents[pi.Id] = pi
		AppContext.PaymentIntentOrder = append(AppContext.PaymentIntentOrder, pi.Id)
		AppContext.mu.Unlock()
	}

	expMonth, expYear := parseExpMonthYear(expMonthStr, expYearStr)
	charge, payErr := confirmPaymentIntentInternal(pi, paymentMethod, cardNumber, expMonth, expYear, cardholderName, customerEmail, successUrl, "", reqId)
	if payErr != nil {
		emitAndDeliverEvent("payment_intent.payment_failed", pi, pi.Id, reqId, webhookUrl)

		renderCheckoutForm(w, http.StatusPaymentRequired, checkoutTemplateData{
			ResourceID:     id,
			Title:          title,
			Description:    description,
			Amount:         amount,
			Currency:       currency,
			FormAction:     fmt.Sprintf("/checkout/session/%s/submit", id),
			SuccessUrl:     successUrl,
			CancelUrl:      cancelUrl,
			State:          state,
			CustomerEmail:  customerEmail,
			CardholderName: cardholderName,
			CardNumber:     cardNumber,
			ExpMonth:       expMonthStr,
			ExpYear:        expYearStr,
			Cvc:            cvc,
			PaymentMethod:  paymentMethod,
			Error:          payErr.Message,
		})
		return
	}

	AppContext.mu.Lock()
	cs.Status = "complete"
	cs.PaymentStatus = "paid"
	if customerEmail != "" {
		cs.CustomerEmail = customerEmail
		if cs.CustomerDetails != nil {
			cs.CustomerDetails["email"] = customerEmail
		}
	}
	if cardholderName != "" && cs.CustomerDetails != nil {
		cs.CustomerDetails["name"] = cardholderName
	}
	AppContext.mu.Unlock()

	emitAndDeliverEvent("checkout.session.completed", cs, pi.Id, reqId, webhookUrl)
	emitAndDeliverEvent("payment_intent.succeeded", pi, pi.Id, reqId, webhookUrl)
	emitAndDeliverEvent("charge.succeeded", charge, pi.Id, reqId, webhookUrl)

	if successUrl != "" {
		redirectTo := successUrl
		if strings.Contains(redirectTo, "{CHECKOUT_SESSION_ID}") {
			redirectTo = strings.ReplaceAll(redirectTo, "{CHECKOUT_SESSION_ID}", id)
			if state != "" {
				redirectTo = appendQueryParams(redirectTo, map[string]string{"state": state}, []string{"state"})
			}
		} else {
			qParams := map[string]string{
				"session_id":        id,
				"payment_intent_id": pi.Id,
				"status":            "succeeded",
				"state":             state,
			}
			redirectTo = appendQueryParams(redirectTo, qParams, []string{"session_id", "payment_intent_id", "status", "state"})
		}
		http.Redirect(w, r, redirectTo, http.StatusFound)
		return
	}

	renderCheckoutResult(w, http.StatusOK, checkoutResultData{
		Heading:         "Payment Successful",
		Status:          "succeeded",
		Message:         "Thank you! Your checkout session is complete.",
		PaymentIntentID: pi.Id,
		FormattedAmount: formatCurrencyAmount(amount, currency),
		UpperCurrency:   strings.ToUpper(currency),
	})
}

package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func GET_testing_config(w http.ResponseWriter, r *http.Request) {
	AppContext.mu.RLock()
	cfg := cloneTestingConfig(AppContext.Testing)
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, cfg)
}

func PUT_testing_config(w http.ResponseWriter, r *http.Request) {
	var update TestingConfigUpdate
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "invalid_json", "", "Invalid JSON body", "")
		return
	}

	AppContext.mu.Lock()
	if update.DelayMs != nil {
		AppContext.Testing.DelayMs = *update.DelayMs
	}
	if update.WebhookDelayMs != nil {
		AppContext.Testing.WebhookDelayMs = *update.WebhookDelayMs
	}
	if update.AlwaysFail != nil {
		AppContext.Testing.AlwaysFail = *update.AlwaysFail
	}
	if update.AlwaysFailCode != nil {
		AppContext.Testing.AlwaysFailCode = *update.AlwaysFailCode
	}
	if update.AlwaysFailMessage != nil {
		AppContext.Testing.AlwaysFailMessage = *update.AlwaysFailMessage
	}
	if update.DeclinedCards != nil {
		if AppContext.Testing.DeclinedCards == nil {
			AppContext.Testing.DeclinedCards = make(map[string]DeclinedCardRule)
		}
		for k, v := range update.DeclinedCards {
			AppContext.Testing.DeclinedCards[k] = v
		}
	}
	cfg := cloneTestingConfig(AppContext.Testing)
	AppContext.mu.Unlock()

	writeJSON(w, http.StatusOK, cfg)
}

func GET_testing_requests(w http.ResponseWriter, r *http.Request) {
	methodFilter := strings.ToUpper(r.URL.Query().Get("method"))
	pathFilter := r.URL.Query().Get("path")
	limit := 0
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}

	AppContext.mu.RLock()
	result := make([]RecordedRequest, 0)
	for _, req := range AppContext.RecordedRequests {
		if methodFilter != "" && strings.ToUpper(req.Method) != methodFilter {
			continue
		}
		if pathFilter != "" && req.Path != pathFilter && !strings.HasPrefix(req.Path, pathFilter) {
			continue
		}
		result = append(result, req)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"requests": result,
	})
}

func GET_testing_payments(w http.ResponseWriter, r *http.Request) {
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

	charges := make([]*Charge, 0, len(AppContext.ChargeOrder))
	for _, id := range AppContext.ChargeOrder {
		if ch, ok := AppContext.Charges[id]; ok {
			charges = append(charges, ch)
		}
	}

	refunds := make([]*Refund, 0, len(AppContext.RefundOrder))
	for _, id := range AppContext.RefundOrder {
		if ref, ok := AppContext.Refunds[id]; ok {
			refunds = append(refunds, ref)
		}
	}

	sessions := make([]*CheckoutSession, 0, len(AppContext.SessionOrder))
	for _, id := range AppContext.SessionOrder {
		if cs, ok := AppContext.CheckoutSessions[id]; ok {
			sessions = append(sessions, cs)
		}
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"payment_intents":   intents,
		"charges":           charges,
		"refunds":           refunds,
		"checkout_sessions": sessions,
	})
}

func GET_testing_webhooks(w http.ResponseWriter, r *http.Request) {
	eventTypeFilter := r.URL.Query().Get("event_type")
	urlFilter := r.URL.Query().Get("url")
	piFilter := r.URL.Query().Get("payment_intent_id")
	statusFilter := r.URL.Query().Get("status")

	AppContext.mu.RLock()
	deliveries := make([]WebhookDelivery, 0)
	for _, del := range AppContext.WebhookDeliveries {
		if eventTypeFilter != "" && del.EventType != eventTypeFilter {
			continue
		}
		if urlFilter != "" && del.Url != urlFilter {
			continue
		}
		if piFilter != "" && del.PaymentIntentId != piFilter {
			continue
		}
		if statusFilter != "" && del.Status != statusFilter {
			continue
		}
		deliveries = append(deliveries, del)
	}
	AppContext.mu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"deliveries": deliveries,
	})
}

type triggerWebhookRequest struct {
	Type            string                 `json:"type"`
	Url             string                 `json:"url"`
	Secret          string                 `json:"secret"`
	PaymentIntentId string                 `json:"payment_intent_id"`
	Data            map[string]interface{} `json:"data"`
}

func POST_testing_webhooks_trigger(w http.ResponseWriter, r *http.Request) {
	var req triggerWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Type == "" {
		writeStripeError(w, http.StatusBadRequest, "invalid_request_error", "parameter_missing", "", "Missing required parameter: type", "type")
		return
	}

	var dataObj interface{}
	if len(req.Data) > 0 {
		dataObj = req.Data
	} else {
		if req.PaymentIntentId != "" {
			AppContext.mu.RLock()
			if pi, ok := AppContext.PaymentIntents[req.PaymentIntentId]; ok {
				dataObj = pi
			}
			AppContext.mu.RUnlock()
		}
		if dataObj == nil {
			piId := req.PaymentIntentId
			if piId == "" {
				piId = generateID("pi")
			}
			dataObj = map[string]interface{}{
				"id":       piId,
				"object":   "payment_intent",
				"amount":   2000,
				"currency": "usd",
				"status":   "succeeded",
			}
		}
	}

	var objMap map[string]interface{}
	if raw, err := json.Marshal(dataObj); err == nil {
		_ = json.Unmarshal(raw, &objMap)
	}

	piId := req.PaymentIntentId
	if piId == "" {
		if idStr, ok := objMap["id"].(string); ok && strings.HasPrefix(idStr, "pi_") {
			piId = idStr
		}
	}

	evt := StripeEvent{
		Id:         generateID("evt"),
		Object:     "event",
		ApiVersion: "2023-10-16",
		Created:    time.Now().Unix(),
		Data: map[string]interface{}{
			"object": objMap,
		},
		Livemode:        false,
		PendingWebhooks: 0,
		Request: map[string]interface{}{
			"id":              nil,
			"idempotency_key": nil,
		},
		Type: req.Type,
	}

	AppContext.mu.Lock()
	AppContext.Events[evt.Id] = &evt
	AppContext.EventOrder = append(AppContext.EventOrder, evt.Id)

	var targets []webhookTarget
	if req.Url != "" {
		sec := req.Secret
		if sec == "" {
			sec = AppConfig.WebhookSecret
		}
		targets = append(targets, webhookTarget{Url: req.Url, Secret: sec})
	} else {
		for _, whId := range AppContext.WebhookOrder {
			wh := AppContext.WebhookEndpoints[whId]
			if wh != nil && wh.Status == "enabled" && wh.Url != "" && matchesWebhookEvents(wh.EnabledEvents, req.Type) {
				sec := req.Secret
				if sec == "" {
					sec = wh.Secret
				}
				if sec == "" {
					sec = AppConfig.WebhookSecret
				}
				targets = append(targets, webhookTarget{Url: wh.Url, Secret: sec})
			}
		}
	}
	AppContext.mu.Unlock()

	deliveries := make([]WebhookDelivery, 0, len(targets))
	for _, t := range targets {
		del := deliverWebhookToTarget(evt, t.Url, t.Secret, piId)
		deliveries = append(deliveries, del)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"event":      evt,
		"deliveries": deliveries,
	})
}

func POST_testing_reset(w http.ResponseWriter, r *http.Request) {
	AppContext.Reset()

	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "OK",
		"message": "State reset to initial configuration",
	})
}

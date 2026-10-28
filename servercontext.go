package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"
)

type AppServerContext struct {
	mu                 sync.RWMutex
	PaymentLinks       map[string]*PaymentLink
	PaymentLinkOrder   []string
	PaymentIntents     map[string]*PaymentIntent
	PaymentIntentOrder []string
	CheckoutSessions   map[string]*CheckoutSession
	SessionOrder       []string
	Customers          map[string]*Customer
	CustomerOrder      []string
	PaymentMethods     map[string]*PaymentMethod
	PaymentMethodOrder []string
	Charges            map[string]*Charge
	ChargeOrder        []string
	Refunds            map[string]*Refund
	RefundOrder        []string
	Events             map[string]*StripeEvent
	EventOrder         []string
	WebhookEndpoints   map[string]*WebhookEndpoint
	WebhookOrder       []string
	RecordedRequests   []RecordedRequest
	WebhookDeliveries  []WebhookDelivery
	Testing            TestingConfig
}

var AppContext *AppServerContext

func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}

func cloneMetadata(m map[string]string) map[string]string {
	res := make(map[string]string)
	for k, v := range m {
		res[k] = v
	}
	return res
}

func cloneTestingConfig(cfg TestingConfig) TestingConfig {
	dc := make(map[string]DeclinedCardRule)
	for k, v := range cfg.DeclinedCards {
		dc[k] = v
	}
	return TestingConfig{
		DelayMs:           cfg.DelayMs,
		WebhookDelayMs:    cfg.WebhookDelayMs,
		AlwaysFail:        cfg.AlwaysFail,
		AlwaysFailCode:    cfg.AlwaysFailCode,
		AlwaysFailMessage: cfg.AlwaysFailMessage,
		DeclinedCards:     dc,
	}
}

func NewPaymentLinkObject(id, title, description string, amount int64, currency string, active bool, callbackUrl, successUrl, cancelUrl, webhookUrl string, metadata map[string]string) *PaymentLink {
	if id == "" {
		id = generateID("plink")
	}
	if title == "" {
		title = "Payment"
	}
	if currency == "" {
		currency = "usd"
	} else {
		currency = strings.ToLower(currency)
	}
	if metadata == nil {
		metadata = make(map[string]string)
	}
	afterCompletion := map[string]interface{}{
		"type": "hosted_confirmation",
	}
	if successUrl != "" {
		afterCompletion = map[string]interface{}{
			"type": "redirect",
			"redirect": map[string]interface{}{
				"url": successUrl,
			},
		}
	}

	return &PaymentLink{
		Id:                    id,
		Object:                "payment_link",
		Active:                active,
		AfterCompletion:       afterCompletion,
		AllowPromotionCodes:   false,
		ApplicationFeeAmount:  nil,
		ApplicationFeePercent: nil,
		AutomaticTax: map[string]interface{}{
			"enabled":   false,
			"liability": nil,
		},
		BillingAddressCollection: "auto",
		Currency:                 currency,
		CustomFields:             []interface{}{},
		CustomText: map[string]interface{}{
			"after_submit":                nil,
			"shipping_address":            nil,
			"submit":                      nil,
			"terms_of_service_acceptance": nil,
		},
		CustomerCreation: "if_required",
		InactiveMessage:  nil,
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
		Livemode:                false,
		Metadata:                cloneMetadata(metadata),
		OnBehalfOf:              nil,
		PaymentIntentData:       nil,
		PaymentMethodCollection: "always",
		PaymentMethodTypes:      nil,
		PhoneNumberCollection: map[string]interface{}{
			"enabled": false,
		},
		Restrictions:              nil,
		ShippingAddressCollection: nil,
		ShippingOptions:           []interface{}{},
		SubmitType:                "auto",
		SubscriptionData:          nil,
		TaxIdCollection: map[string]interface{}{
			"enabled": false,
		},
		TransferData: nil,
		Url:          fmt.Sprintf("%s/checkout/link/%s", AppConfig.BaseUrl, id),
		Amount:       amount,
		Title:        title,
		Description:  description,
		CallbackUrl:  callbackUrl,
		SuccessUrl:   successUrl,
		CancelUrl:    cancelUrl,
		WebhookUrl:   webhookUrl,
		Created:      time.Now().Unix(),
	}
}

func NewCustomerObject(id, email, name, phone, description string, balance int64, metadata map[string]string) *Customer {
	if id == "" {
		id = generateID("cus")
	}
	if metadata == nil {
		metadata = make(map[string]string)
	}
	var emailVal interface{}
	if email != "" {
		emailVal = email
	}
	var nameVal interface{}
	if name != "" {
		nameVal = name
	}
	var phoneVal interface{}
	if phone != "" {
		phoneVal = phone
	}
	var descVal interface{}
	if description != "" {
		descVal = description
	}
	return &Customer{
		Id:            id,
		Object:        "customer",
		Address:       nil,
		Balance:       balance,
		Created:       time.Now().Unix(),
		Currency:      nil,
		DefaultSource: nil,
		Delinquent:    false,
		Description:   descVal,
		Discount:      nil,
		Email:         emailVal,
		InvoicePrefix: "0759376C",
		InvoiceSettings: map[string]interface{}{
			"custom_fields":          nil,
			"default_payment_method": nil,
			"footer":                 nil,
			"rendering_options":      nil,
		},
		Livemode:            false,
		Metadata:            cloneMetadata(metadata),
		Name:                nameVal,
		NextInvoiceSequence: 1,
		Phone:               phoneVal,
		PreferredLocales:    []string{},
		Shipping:            nil,
		TaxExempt:           "none",
		TestClock:           nil,
	}
}

func NewWebhookEndpointObject(id, url, secret, description string, events []string, metadata map[string]string) *WebhookEndpoint {
	if id == "" {
		id = generateID("we")
	}
	if secret == "" {
		secret = AppConfig.WebhookSecret
	}
	if len(events) == 0 {
		events = []string{"*"}
	}
	if metadata == nil {
		metadata = make(map[string]string)
	}
	var descVal interface{}
	if description != "" {
		descVal = description
	}
	evCopy := make([]string, len(events))
	copy(evCopy, events)
	return &WebhookEndpoint{
		Id:            id,
		Object:        "webhook_endpoint",
		ApiVersion:    nil,
		Application:   nil,
		Created:       time.Now().Unix(),
		Description:   descVal,
		EnabledEvents: evCopy,
		Livemode:      false,
		Metadata:      cloneMetadata(metadata),
		Secret:        secret,
		Status:        "enabled",
		Url:           url,
	}
}

func NewAppContext() *AppServerContext {
	ctx := &AppServerContext{}
	ctx.Reset()
	return ctx
}

func (ctx *AppServerContext) Reset() {
	ctx.mu.Lock()
	defer ctx.mu.Unlock()

	ctx.PaymentLinks = make(map[string]*PaymentLink)
	ctx.PaymentLinkOrder = []string{}
	ctx.PaymentIntents = make(map[string]*PaymentIntent)
	ctx.PaymentIntentOrder = []string{}
	ctx.CheckoutSessions = make(map[string]*CheckoutSession)
	ctx.SessionOrder = []string{}
	ctx.Customers = make(map[string]*Customer)
	ctx.CustomerOrder = []string{}
	ctx.PaymentMethods = make(map[string]*PaymentMethod)
	ctx.PaymentMethodOrder = []string{}
	ctx.Charges = make(map[string]*Charge)
	ctx.ChargeOrder = []string{}
	ctx.Refunds = make(map[string]*Refund)
	ctx.RefundOrder = []string{}
	ctx.Events = make(map[string]*StripeEvent)
	ctx.EventOrder = []string{}
	ctx.WebhookEndpoints = make(map[string]*WebhookEndpoint)
	ctx.WebhookOrder = []string{}
	ctx.RecordedRequests = []RecordedRequest{}
	ctx.WebhookDeliveries = []WebhookDelivery{}
	ctx.Testing = cloneTestingConfig(AppConfig.Testing)

	for _, plCfg := range AppConfig.PaymentLinks {
		active := true
		if plCfg.Active != nil {
			active = *plCfg.Active
		}
		pl := NewPaymentLinkObject(
			plCfg.Id,
			plCfg.Title,
			plCfg.Description,
			plCfg.Amount,
			plCfg.Currency,
			active,
			plCfg.CallbackUrl,
			plCfg.SuccessUrl,
			plCfg.CancelUrl,
			plCfg.WebhookUrl,
			plCfg.Metadata,
		)
		ctx.PaymentLinks[pl.Id] = pl
		ctx.PaymentLinkOrder = append(ctx.PaymentLinkOrder, pl.Id)
	}

	for _, cusCfg := range AppConfig.Customers {
		cus := NewCustomerObject(
			cusCfg.Id,
			cusCfg.Email,
			cusCfg.Name,
			cusCfg.Phone,
			cusCfg.Description,
			cusCfg.Balance,
			cusCfg.Metadata,
		)
		ctx.Customers[cus.Id] = cus
		ctx.CustomerOrder = append(ctx.CustomerOrder, cus.Id)
	}

	for _, whCfg := range AppConfig.Webhooks {
		events := whCfg.Events
		if len(events) == 0 {
			events = whCfg.EnabledEvents
		}
		wh := NewWebhookEndpointObject(
			whCfg.Id,
			whCfg.Url,
			whCfg.Secret,
			whCfg.Description,
			events,
			whCfg.Metadata,
		)
		ctx.WebhookEndpoints[wh.Id] = wh
		ctx.WebhookOrder = append(ctx.WebhookOrder, wh.Id)
	}
}

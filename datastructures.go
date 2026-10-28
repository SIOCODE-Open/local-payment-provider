package main

type ApiToggleConfig struct {
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

type DeclinedCardRule struct {
	Code        string `json:"code" yaml:"code"`
	DeclineCode string `json:"decline_code" yaml:"decline_code"`
	Message     string `json:"message" yaml:"message"`
}

type TestingConfig struct {
	DelayMs           int                         `json:"delay_ms" yaml:"delay_ms"`
	WebhookDelayMs    int                         `json:"webhook_delay_ms" yaml:"webhook_delay_ms"`
	AlwaysFail        bool                        `json:"always_fail" yaml:"always_fail"`
	AlwaysFailCode    string                      `json:"always_fail_code" yaml:"always_fail_code"`
	AlwaysFailMessage string                      `json:"always_fail_message" yaml:"always_fail_message"`
	DeclinedCards     map[string]DeclinedCardRule `json:"declined_cards" yaml:"declined_cards"`
}

type TestingConfigUpdate struct {
	DelayMs           *int                        `json:"delay_ms,omitempty"`
	WebhookDelayMs    *int                        `json:"webhook_delay_ms,omitempty"`
	AlwaysFail        *bool                       `json:"always_fail,omitempty"`
	AlwaysFailCode    *string                     `json:"always_fail_code,omitempty"`
	AlwaysFailMessage *string                     `json:"always_fail_message,omitempty"`
	DeclinedCards     map[string]DeclinedCardRule `json:"declined_cards,omitempty"`
}

type PaymentLinkConfig struct {
	Id          string            `json:"id" yaml:"id"`
	Title       string            `json:"title" yaml:"title"`
	Description string            `json:"description" yaml:"description"`
	Amount      int64             `json:"amount" yaml:"amount"`
	Currency    string            `json:"currency" yaml:"currency"`
	Active      *bool             `json:"active,omitempty" yaml:"active,omitempty"`
	CallbackUrl string            `json:"callback_url" yaml:"callback_url"`
	SuccessUrl  string            `json:"success_url" yaml:"success_url"`
	CancelUrl   string            `json:"cancel_url" yaml:"cancel_url"`
	WebhookUrl  string            `json:"webhook_url" yaml:"webhook_url"`
	Metadata    map[string]string `json:"metadata" yaml:"metadata"`
}

type CustomerConfig struct {
	Id          string            `json:"id" yaml:"id"`
	Email       string            `json:"email" yaml:"email"`
	Name        string            `json:"name" yaml:"name"`
	Phone       string            `json:"phone" yaml:"phone"`
	Description string            `json:"description" yaml:"description"`
	Balance     int64             `json:"balance" yaml:"balance"`
	Metadata    map[string]string `json:"metadata" yaml:"metadata"`
}

type WebhookConfig struct {
	Id            string            `json:"id" yaml:"id"`
	Url           string            `json:"url" yaml:"url"`
	Secret        string            `json:"secret" yaml:"secret"`
	Events        []string          `json:"events" yaml:"events"`
	EnabledEvents []string          `json:"enabled_events" yaml:"enabled_events"`
	Description   string            `json:"description" yaml:"description"`
	Metadata      map[string]string `json:"metadata" yaml:"metadata"`
}

type PaymentProviderConfig struct {
	Port           int                 `json:"port" yaml:"port"`
	BaseUrl        string              `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	AllowedOrigins string              `json:"allowed_origins,omitempty" yaml:"allowed_origins,omitempty"`
	WebhookSecret  string              `json:"webhook_secret,omitempty" yaml:"webhook_secret,omitempty"`
	ApiKeys        []string            `json:"api_keys,omitempty" yaml:"api_keys,omitempty"`
	NativeApi      ApiToggleConfig     `json:"native_api,omitempty" yaml:"native_api,omitempty"`
	StripeApi      ApiToggleConfig     `json:"stripe_api,omitempty" yaml:"stripe_api,omitempty"`
	PaymentLinks   []PaymentLinkConfig `json:"payment_links,omitempty" yaml:"payment_links,omitempty"`
	Customers      []CustomerConfig    `json:"customers,omitempty" yaml:"customers,omitempty"`
	Webhooks       []WebhookConfig     `json:"webhooks,omitempty" yaml:"webhooks,omitempty"`
	Testing        TestingConfig       `json:"testing,omitempty" yaml:"testing,omitempty"`
}

// PaymentLink represents both Native and Stripe PaymentLink objects
type PaymentLink struct {
	Id                        string                 `json:"id"`
	Object                    string                 `json:"object"`
	Active                    bool                   `json:"active"`
	AfterCompletion           map[string]interface{} `json:"after_completion"`
	AllowPromotionCodes       bool                   `json:"allow_promotion_codes"`
	ApplicationFeeAmount      interface{}            `json:"application_fee_amount"`
	ApplicationFeePercent     interface{}            `json:"application_fee_percent"`
	AutomaticTax              map[string]interface{} `json:"automatic_tax"`
	BillingAddressCollection  string                 `json:"billing_address_collection"`
	Currency                  string                 `json:"currency"`
	CustomFields              []interface{}          `json:"custom_fields"`
	CustomText                map[string]interface{} `json:"custom_text"`
	CustomerCreation          string                 `json:"customer_creation"`
	InactiveMessage           interface{}            `json:"inactive_message"`
	InvoiceCreation           map[string]interface{} `json:"invoice_creation"`
	Livemode                  bool                   `json:"livemode"`
	Metadata                  map[string]string      `json:"metadata"`
	OnBehalfOf                interface{}            `json:"on_behalf_of"`
	PaymentIntentData         interface{}            `json:"payment_intent_data"`
	PaymentMethodCollection   string                 `json:"payment_method_collection"`
	PaymentMethodTypes        interface{}            `json:"payment_method_types"`
	PhoneNumberCollection     map[string]interface{} `json:"phone_number_collection"`
	Restrictions              interface{}            `json:"restrictions"`
	ShippingAddressCollection interface{}            `json:"shipping_address_collection"`
	ShippingOptions           []interface{}          `json:"shipping_options"`
	SubmitType                string                 `json:"submit_type"`
	SubscriptionData          interface{}            `json:"subscription_data"`
	TaxIdCollection           map[string]interface{} `json:"tax_id_collection"`
	TransferData              interface{}            `json:"transfer_data"`
	Url                       string                 `json:"url"`
	Amount                    int64                  `json:"amount"`
	Title                     string                 `json:"title"`
	Description               string                 `json:"description"`
	CallbackUrl               string                 `json:"callback_url"`
	SuccessUrl                string                 `json:"success_url"`
	CancelUrl                 string                 `json:"cancel_url"`
	WebhookUrl                string                 `json:"webhook_url"`
	Created                   int64                  `json:"created"`
}

type PaymentError struct {
	Type        string `json:"type"`
	Code        string `json:"code"`
	DeclineCode string `json:"decline_code,omitempty"`
	Message     string `json:"message"`
	Param       string `json:"param,omitempty"`
}

// PaymentIntent represents both Native and Stripe PaymentIntent objects
type PaymentIntent struct {
	Id                        string                 `json:"id"`
	Object                    string                 `json:"object"`
	Amount                    int64                  `json:"amount"`
	AmountCapturable          int64                  `json:"amount_capturable"`
	AmountReceived            int64                  `json:"amount_received"`
	AmountRefunded            int64                  `json:"amount_refunded"`
	Application               interface{}            `json:"application"`
	ApplicationFeeAmount      interface{}            `json:"application_fee_amount"`
	AutomaticPaymentMethods   map[string]interface{} `json:"automatic_payment_methods"`
	CanceledAt                *int64                 `json:"canceled_at"`
	CancellationReason        *string                `json:"cancellation_reason"`
	CaptureMethod             string                 `json:"capture_method"`
	ClientSecret              string                 `json:"client_secret"`
	CheckoutUrl               string                 `json:"checkout_url"`
	ConfirmationMethod        string                 `json:"confirmation_method"`
	Created                   int64                  `json:"created"`
	Currency                  string                 `json:"currency"`
	Customer                  interface{}            `json:"customer"`
	CustomerEmail             string                 `json:"customer_email"`
	Description               string                 `json:"description"`
	Invoice                   interface{}            `json:"invoice"`
	LastPaymentError          *PaymentError          `json:"last_payment_error"`
	LatestCharge              interface{}            `json:"latest_charge"`
	Livemode                  bool                   `json:"livemode"`
	Metadata                  map[string]string      `json:"metadata"`
	NextAction                interface{}            `json:"next_action"`
	PaymentLinkId             string                 `json:"payment_link_id"`
	PaymentMethod             interface{}            `json:"payment_method"`
	PaymentMethodOptions      map[string]interface{} `json:"payment_method_options"`
	PaymentMethodTypes        []string               `json:"payment_method_types"`
	Processing                interface{}            `json:"processing"`
	ReceiptEmail              interface{}            `json:"receipt_email"`
	ReturnUrl                 string                 `json:"return_url,omitempty"`
	SetupFutureUsage          interface{}            `json:"setup_future_usage"`
	Shipping                  interface{}            `json:"shipping"`
	StatementDescriptor       interface{}            `json:"statement_descriptor"`
	StatementDescriptorSuffix interface{}            `json:"statement_descriptor_suffix"`
	Status                    string                 `json:"status"`
	TransferData              interface{}            `json:"transfer_data"`
	TransferGroup             interface{}            `json:"transfer_group"`
	CallbackUrl               string                 `json:"callback_url"`
	SuccessUrl                string                 `json:"success_url"`
	CancelUrl                 string                 `json:"cancel_url"`
	WebhookUrl                string                 `json:"webhook_url"`
}

type CheckoutSession struct {
	Id                                string                 `json:"id"`
	Object                            string                 `json:"object"`
	AfterExpiration                   interface{}            `json:"after_expiration"`
	AllowPromotionCodes               interface{}            `json:"allow_promotion_codes"`
	AmountSubtotal                    int64                  `json:"amount_subtotal"`
	AmountTotal                       int64                  `json:"amount_total"`
	AutomaticTax                      map[string]interface{} `json:"automatic_tax"`
	BillingAddressCollection          interface{}            `json:"billing_address_collection"`
	CancelUrl                         string                 `json:"cancel_url"`
	ClientReferenceId                 interface{}            `json:"client_reference_id"`
	ClientSecret                      interface{}            `json:"client_secret"`
	Consent                           interface{}            `json:"consent"`
	ConsentCollection                 interface{}            `json:"consent_collection"`
	Created                           int64                  `json:"created"`
	Currency                          string                 `json:"currency"`
	CustomFields                      []interface{}          `json:"custom_fields"`
	CustomText                        map[string]interface{} `json:"custom_text"`
	Customer                          interface{}            `json:"customer"`
	CustomerCreation                  string                 `json:"customer_creation"`
	CustomerDetails                   map[string]interface{} `json:"customer_details"`
	CustomerEmail                     interface{}            `json:"customer_email"`
	ExpiresAt                         int64                  `json:"expires_at"`
	Invoice                           interface{}            `json:"invoice"`
	InvoiceCreation                   map[string]interface{} `json:"invoice_creation"`
	Livemode                          bool                   `json:"livemode"`
	Locale                            interface{}            `json:"locale"`
	Metadata                          map[string]string      `json:"metadata"`
	Mode                              string                 `json:"mode"`
	PaymentIntent                     interface{}            `json:"payment_intent"`
	PaymentLink                       interface{}            `json:"payment_link"`
	PaymentMethodCollection           string                 `json:"payment_method_collection"`
	PaymentMethodConfigurationDetails interface{}            `json:"payment_method_configuration_details"`
	PaymentMethodOptions              map[string]interface{} `json:"payment_method_options"`
	PaymentMethodTypes                []string               `json:"payment_method_types"`
	PaymentStatus                     string                 `json:"payment_status"`
	PhoneNumberCollection             map[string]interface{} `json:"phone_number_collection"`
	RecoveredFrom                     interface{}            `json:"recovered_from"`
	SetupIntent                       interface{}            `json:"setup_intent"`
	ShippingAddressCollection         interface{}            `json:"shipping_address_collection"`
	ShippingCost                      interface{}            `json:"shipping_cost"`
	ShippingDetails                   interface{}            `json:"shipping_details"`
	ShippingOptions                   []interface{}          `json:"shipping_options"`
	Status                            string                 `json:"status"`
	SubmitType                        interface{}            `json:"submit_type"`
	Subscription                      interface{}            `json:"subscription"`
	SuccessUrl                        string                 `json:"success_url"`
	TotalDetails                      map[string]interface{} `json:"total_details"`
	UiMode                            string                 `json:"ui_mode"`
	Url                               string                 `json:"url"`
	WebhookUrl                        string                 `json:"webhook_url,omitempty"`
	Title                             string                 `json:"title,omitempty"`
	Description                       string                 `json:"description,omitempty"`
}

type Customer struct {
	Id                  string                 `json:"id"`
	Object              string                 `json:"object"`
	Address             interface{}            `json:"address"`
	Balance             int64                  `json:"balance"`
	Created             int64                  `json:"created"`
	Currency            interface{}            `json:"currency"`
	DefaultSource       interface{}            `json:"default_source"`
	Delinquent          bool                   `json:"delinquent"`
	Description         interface{}            `json:"description"`
	Discount            interface{}            `json:"discount"`
	Email               interface{}            `json:"email"`
	InvoicePrefix       string                 `json:"invoice_prefix"`
	InvoiceSettings     map[string]interface{} `json:"invoice_settings"`
	Livemode            bool                   `json:"livemode"`
	Metadata            map[string]string      `json:"metadata"`
	Name                interface{}            `json:"name"`
	NextInvoiceSequence int                    `json:"next_invoice_sequence"`
	Phone               interface{}            `json:"phone"`
	PreferredLocales    []string               `json:"preferred_locales"`
	Shipping            interface{}            `json:"shipping"`
	TaxExempt           string                 `json:"tax_exempt"`
	TestClock           interface{}            `json:"test_clock"`
}

type PaymentMethod struct {
	Id             string                 `json:"id"`
	Object         string                 `json:"object"`
	AllowRedisplay string                 `json:"allow_redisplay"`
	BillingDetails map[string]interface{} `json:"billing_details"`
	Card           map[string]interface{} `json:"card"`
	Created        int64                  `json:"created"`
	Customer       interface{}            `json:"customer"`
	Livemode       bool                   `json:"livemode"`
	Metadata       map[string]string      `json:"metadata"`
	Type           string                 `json:"type"`
	CardNumber     string                 `json:"-"`
}

type Charge struct {
	Id                            string                 `json:"id"`
	Object                        string                 `json:"object"`
	Amount                        int64                  `json:"amount"`
	AmountCaptured                int64                  `json:"amount_captured"`
	AmountRefunded                int64                  `json:"amount_refunded"`
	Application                   interface{}            `json:"application"`
	ApplicationFee                interface{}            `json:"application_fee"`
	ApplicationFeeAmount          interface{}            `json:"application_fee_amount"`
	BalanceTransaction            string                 `json:"balance_transaction"`
	BillingDetails                map[string]interface{} `json:"billing_details"`
	CalculatedStatementDescriptor string                 `json:"calculated_statement_descriptor"`
	Captured                      bool                   `json:"captured"`
	Created                       int64                  `json:"created"`
	Currency                      string                 `json:"currency"`
	Customer                      interface{}            `json:"customer"`
	Description                   interface{}            `json:"description"`
	Disputed                      bool                   `json:"disputed"`
	FailureBalanceTransaction     interface{}            `json:"failure_balance_transaction"`
	FailureCode                   interface{}            `json:"failure_code"`
	FailureMessage                interface{}            `json:"failure_message"`
	FraudDetails                  map[string]interface{} `json:"fraud_details"`
	Invoice                       interface{}            `json:"invoice"`
	Livemode                      bool                   `json:"livemode"`
	Metadata                      map[string]string      `json:"metadata"`
	OnBehalfOf                    interface{}            `json:"on_behalf_of"`
	Outcome                       map[string]interface{} `json:"outcome"`
	Paid                          bool                   `json:"paid"`
	PaymentIntent                 string                 `json:"payment_intent"`
	PaymentMethod                 string                 `json:"payment_method"`
	PaymentMethodDetails          map[string]interface{} `json:"payment_method_details"`
	ReceiptEmail                  interface{}            `json:"receipt_email"`
	ReceiptNumber                 interface{}            `json:"receipt_number"`
	ReceiptUrl                    string                 `json:"receipt_url"`
	Refunded                      bool                   `json:"refunded"`
	Review                        interface{}            `json:"review"`
	Shipping                      interface{}            `json:"shipping"`
	Source                        interface{}            `json:"source"`
	SourceTransfer                interface{}            `json:"source_transfer"`
	StatementDescriptor           interface{}            `json:"statement_descriptor"`
	StatementDescriptorSuffix     interface{}            `json:"statement_descriptor_suffix"`
	Status                        string                 `json:"status"`
	TransferData                  interface{}            `json:"transfer_data"`
	TransferGroup                 interface{}            `json:"transfer_group"`
}

type Refund struct {
	Id                     string                 `json:"id"`
	Object                 string                 `json:"object"`
	Amount                 int64                  `json:"amount"`
	BalanceTransaction     string                 `json:"balance_transaction"`
	Charge                 string                 `json:"charge"`
	Created                int64                  `json:"created"`
	Currency               string                 `json:"currency"`
	DestinationDetails     map[string]interface{} `json:"destination_details"`
	Metadata               map[string]string      `json:"metadata"`
	PaymentIntent          string                 `json:"payment_intent"`
	Reason                 interface{}            `json:"reason"`
	ReceiptNumber          interface{}            `json:"receipt_number"`
	SourceTransferReversal interface{}            `json:"source_transfer_reversal"`
	Status                 string                 `json:"status"`
	TransferReversal       interface{}            `json:"transfer_reversal"`
}

type StripeEvent struct {
	Id              string                 `json:"id"`
	Object          string                 `json:"object"`
	ApiVersion      string                 `json:"api_version"`
	Created         int64                  `json:"created"`
	Data            map[string]interface{} `json:"data"`
	Livemode        bool                   `json:"livemode"`
	PendingWebhooks int                    `json:"pending_webhooks"`
	Request         map[string]interface{} `json:"request"`
	Type            string                 `json:"type"`
}

type WebhookEndpoint struct {
	Id            string            `json:"id"`
	Object        string            `json:"object"`
	ApiVersion    interface{}       `json:"api_version"`
	Application   interface{}       `json:"application"`
	Created       int64             `json:"created"`
	Description   interface{}       `json:"description"`
	EnabledEvents []string          `json:"enabled_events"`
	Livemode      bool              `json:"livemode"`
	Metadata      map[string]string `json:"metadata"`
	Secret        string            `json:"secret"`
	Status        string            `json:"status"`
	Url           string            `json:"url"`
}

type RecordedRequest struct {
	Id         string            `json:"id"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Query      string            `json:"query"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
	StatusCode int               `json:"status_code"`
	Timestamp  string            `json:"timestamp"`
}

type WebhookDelivery struct {
	Id                 string            `json:"id"`
	EventId            string            `json:"event_id"`
	EventType          string            `json:"event_type"`
	PaymentIntentId    string            `json:"payment_intent_id"`
	Url                string            `json:"url"`
	Status             string            `json:"status"`
	ResponseStatusCode int               `json:"response_status_code"`
	ResponseBody       string            `json:"response_body"`
	Error              string            `json:"error"`
	RequestHeaders     map[string]string `json:"request_headers"`
	Payload            StripeEvent       `json:"payload"`
	Timestamp          string            `json:"timestamp"`
}

type StripeListResponse struct {
	Object  string      `json:"object"`
	Url     string      `json:"url"`
	HasMore bool        `json:"has_more"`
	Data    interface{} `json:"data"`
}

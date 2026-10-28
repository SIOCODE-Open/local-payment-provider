package main

import (
	"flag"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

func main() {
	flag.Parse()

	log.SetFlags(log.LstdFlags | log.LUTC)
	AppConfig = LoadConfig()
	AppContext = NewAppContext()
	port := AppConfig.Port

	log.Printf("Starting local-payment-provider on port %d", port)
	log.Printf("Base URL: %s", AppConfig.BaseUrl)
	log.Printf("Configured payment links: %d", len(AppConfig.PaymentLinks))
	log.Printf("Configured customers: %d", len(AppConfig.Customers))
	log.Printf("Configured webhooks: %d", len(AppConfig.Webhooks))

	router := mux.NewRouter()

	// Health check
	router.HandleFunc("/healthz", GET_healthz).Methods("GET")

	// Hosted Checkout UI endpoints
	router.HandleFunc("/checkout/link/{id}", GET_checkout_link_id).Methods("GET")
	router.HandleFunc("/checkout/link/{id}/submit", POST_checkout_link_id_submit).Methods("POST")
	router.HandleFunc("/checkout/intent/{id}", GET_checkout_intent_id).Methods("GET")
	router.HandleFunc("/checkout/intent/{id}/submit", POST_checkout_intent_id_submit).Methods("POST")
	router.HandleFunc("/checkout/session/{id}", GET_checkout_session_id).Methods("GET")
	router.HandleFunc("/checkout/session/{id}/submit", POST_checkout_session_id_submit).Methods("POST")

	// Native Payment API endpoints (conditional based on config)
	if *AppConfig.NativeApi.Enabled {
		router.HandleFunc("/payments/links", GET_payments_links).Methods("GET")
		router.HandleFunc("/payments/links", POST_payments_links).Methods("POST")
		router.HandleFunc("/payments/links/{id}", GET_payments_links_id).Methods("GET")
		router.HandleFunc("/payments/intents", POST_payments_intents).Methods("POST")
		router.HandleFunc("/payments/intents", GET_payments_intents).Methods("GET")
		router.HandleFunc("/payments/intents/{id}", GET_payments_intents_id).Methods("GET")
		router.HandleFunc("/payments/intents/{id}/confirm", POST_payments_intents_id_confirm).Methods("POST")
		router.HandleFunc("/payments/intents/{id}/cancel", POST_payments_intents_id_cancel).Methods("POST")
		router.HandleFunc("/payments/intents/{id}/refund", POST_payments_intents_id_refund).Methods("POST")
		router.HandleFunc("/payments/refunds", GET_payments_refunds).Methods("GET")
		router.HandleFunc("/payments/refunds/{id}", GET_payments_refunds_id).Methods("GET")
		log.Printf("Native Payment API endpoints enabled")
	} else {
		log.Printf("Native Payment API endpoints disabled")
	}

	// Stripe-Compatible API endpoints (conditional based on config)
	if *AppConfig.StripeApi.Enabled {
		router.HandleFunc("/v1/payment_intents", POST_v1_payment_intents).Methods("POST")
		router.HandleFunc("/v1/payment_intents", GET_v1_payment_intents).Methods("GET")
		router.HandleFunc("/v1/payment_intents/{id}", GET_v1_payment_intents_id).Methods("GET")
		router.HandleFunc("/v1/payment_intents/{id}", POST_v1_payment_intents_id).Methods("POST")
		router.HandleFunc("/v1/payment_intents/{id}/confirm", POST_v1_payment_intents_id_confirm).Methods("POST")
		router.HandleFunc("/v1/payment_intents/{id}/cancel", POST_v1_payment_intents_id_cancel).Methods("POST")
		router.HandleFunc("/v1/payment_intents/{id}/capture", POST_v1_payment_intents_id_capture).Methods("POST")

		router.HandleFunc("/v1/payment_links", POST_v1_payment_links).Methods("POST")
		router.HandleFunc("/v1/payment_links", GET_v1_payment_links).Methods("GET")
		router.HandleFunc("/v1/payment_links/{id}", GET_v1_payment_links_id).Methods("GET")
		router.HandleFunc("/v1/payment_links/{id}", POST_v1_payment_links_id).Methods("POST")

		router.HandleFunc("/v1/checkout/sessions", POST_v1_checkout_sessions).Methods("POST")
		router.HandleFunc("/v1/checkout/sessions", GET_v1_checkout_sessions).Methods("GET")
		router.HandleFunc("/v1/checkout/sessions/{id}", GET_v1_checkout_sessions_id).Methods("GET")
		router.HandleFunc("/v1/checkout/sessions/{id}/expire", POST_v1_checkout_sessions_id_expire).Methods("POST")

		router.HandleFunc("/v1/customers", POST_v1_customers).Methods("POST")
		router.HandleFunc("/v1/customers", GET_v1_customers).Methods("GET")
		router.HandleFunc("/v1/customers/{id}", GET_v1_customers_id).Methods("GET")
		router.HandleFunc("/v1/customers/{id}", POST_v1_customers_id).Methods("POST")
		router.HandleFunc("/v1/customers/{id}", DELETE_v1_customers_id).Methods("DELETE")

		router.HandleFunc("/v1/payment_methods", POST_v1_payment_methods).Methods("POST")
		router.HandleFunc("/v1/payment_methods", GET_v1_payment_methods).Methods("GET")
		router.HandleFunc("/v1/payment_methods/{id}", GET_v1_payment_methods_id).Methods("GET")
		router.HandleFunc("/v1/payment_methods/{id}/attach", POST_v1_payment_methods_id_attach).Methods("POST")
		router.HandleFunc("/v1/payment_methods/{id}/detach", POST_v1_payment_methods_id_detach).Methods("POST")

		router.HandleFunc("/v1/charges", GET_v1_charges).Methods("GET")
		router.HandleFunc("/v1/charges/{id}", GET_v1_charges_id).Methods("GET")

		router.HandleFunc("/v1/refunds", POST_v1_refunds).Methods("POST")
		router.HandleFunc("/v1/refunds", GET_v1_refunds).Methods("GET")
		router.HandleFunc("/v1/refunds/{id}", GET_v1_refunds_id).Methods("GET")

		router.HandleFunc("/v1/events", GET_v1_events).Methods("GET")
		router.HandleFunc("/v1/events/{id}", GET_v1_events_id).Methods("GET")

		router.HandleFunc("/v1/webhook_endpoints", POST_v1_webhook_endpoints).Methods("POST")
		router.HandleFunc("/v1/webhook_endpoints", GET_v1_webhook_endpoints).Methods("GET")
		router.HandleFunc("/v1/webhook_endpoints/{id}", GET_v1_webhook_endpoints_id).Methods("GET")
		router.HandleFunc("/v1/webhook_endpoints/{id}", DELETE_v1_webhook_endpoints_id).Methods("DELETE")

		log.Printf("Stripe-Compatible API endpoints enabled")
	} else {
		log.Printf("Stripe-Compatible API endpoints disabled")
	}

	// Testing & Spying API endpoints
	router.HandleFunc("/_testing/config", GET_testing_config).Methods("GET")
	router.HandleFunc("/_testing/config", PUT_testing_config).Methods("PUT")
	router.HandleFunc("/_testing/requests", GET_testing_requests).Methods("GET")
	router.HandleFunc("/_testing/payments", GET_testing_payments).Methods("GET")
	router.HandleFunc("/_testing/webhooks", GET_testing_webhooks).Methods("GET")
	router.HandleFunc("/_testing/webhooks/trigger", POST_testing_webhooks_trigger).Methods("POST")
	router.HandleFunc("/_testing/reset", POST_testing_reset).Methods("POST")

	corsRouter := corsMiddleware(router)
	loggedRouter := accessLogger(corsRouter)

	addr := ":" + strconv.Itoa(port)
	if err := http.ListenAndServe(addr, loggedRouter); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

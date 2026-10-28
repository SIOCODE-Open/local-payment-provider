package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

var AppConfig *PaymentProviderConfig
var ConfigPathFlag string

func init() {
	flag.StringVar(&ConfigPathFlag, "config-path", "", "Path to configuration file")
	flag.StringVar(&ConfigPathFlag, "c", "", "Path to configuration file (shorthand)")
}

func defaultDeclinedCards() map[string]DeclinedCardRule {
	return map[string]DeclinedCardRule{
		"4000000000000002": {
			Code:        "card_declined",
			DeclineCode: "generic_decline",
			Message:     "Your card was declined.",
		},
		"4000000000009995": {
			Code:        "card_declined",
			DeclineCode: "insufficient_funds",
			Message:     "Your card has insufficient funds.",
		},
		"4000000000000069": {
			Code:        "expired_card",
			DeclineCode: "expired_card",
			Message:     "Your card has expired.",
		},
		"4000000000000119": {
			Code:        "processing_error",
			DeclineCode: "processing_error",
			Message:     "An error occurred while processing your card.",
		},
	}
}

func LoadConfig() *PaymentProviderConfig {
	config := new(PaymentProviderConfig)

	configPath := ConfigPathFlag
	if configPath == "" {
		configPath = os.Getenv("CONFIG_PATH")
	}
	if configPath == "" {
		configPath = "/config.yaml"
	}

	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, config); err != nil {
			log.Printf("Error parsing config file: %v", err)
		}
	} else {
		log.Printf("Error reading config file: %v", err)
	}

	if portEnv := os.Getenv("PORT"); portEnv != "" {
		if p, err := strconv.Atoi(portEnv); err == nil {
			config.Port = p
		}
	}

	if config.Port == 0 {
		config.Port = 8080
	}

	if config.BaseUrl == "" {
		config.BaseUrl = fmt.Sprintf("http://localhost:%d", config.Port)
	}
	config.BaseUrl = strings.TrimRight(config.BaseUrl, "/")

	if config.AllowedOrigins == "" {
		config.AllowedOrigins = "*"
	}

	if config.WebhookSecret == "" {
		config.WebhookSecret = "whsec_test_secret"
	}

	if config.NativeApi.Enabled == nil {
		trueVal := true
		config.NativeApi.Enabled = &trueVal
	}

	if config.StripeApi.Enabled == nil {
		trueVal := true
		config.StripeApi.Enabled = &trueVal
	}

	if config.Testing.AlwaysFailCode == "" {
		config.Testing.AlwaysFailCode = "processing_error"
	}
	if config.Testing.AlwaysFailMessage == "" {
		config.Testing.AlwaysFailMessage = "Payment processing failed due to test configuration"
	}

	defaults := defaultDeclinedCards()
	if config.Testing.DeclinedCards == nil {
		config.Testing.DeclinedCards = defaults
	} else {
		for k, v := range defaults {
			if _, exists := config.Testing.DeclinedCards[k]; !exists {
				config.Testing.DeclinedCards[k] = v
			}
		}
	}

	return config
}

# Configuration Reference – SIOCODE Local Payment Provider

This document describes all configuration options for the SIOCODE Local Payment Provider.

---

## Configuration File

The payment provider is configured via a YAML file. By default, the server looks for `/config.yaml`, but you can specify a custom path using:

1. Command-line flag: `-c` or `--config-path`
2. Environment variable: `CONFIG_PATH`

### Configuration File Path

- **Command-Line Flag**: `-c <path>` or `--config-path <path>`
- **Environment Variable**: `CONFIG_PATH`
- **Default**: `/config.yaml`
- **Example**:
  - `local-payment-provider -c /app/my-config.yaml`
  - `local-payment-provider --config-path /app/my-config.yaml`
  - `CONFIG_PATH=/app/my-config.yaml local-payment-provider`

---

## Configuration Structure

### YAML Format

```yaml
port: 8080
base_url: http://localhost:8080
allowed_origins: "*"
webhook_secret: "whsec_test_secret"
api_keys: []

native_api:
  enabled: true

stripe_api:
  enabled: true

payment_links:
  - id: "plink_pro_plan"
    title: "Pro Plan Subscription"
    description: "Monthly subscription to Pro Plan"
    amount: 2999
    currency: "usd"
    active: true
    callback_url: "http://localhost:3000/payment/callback"
    success_url: "http://localhost:3000/payment/success"
    cancel_url: "http://localhost:3000/payment/cancel"
    webhook_url: "http://localhost:3000/webhooks/payment"
    metadata:
      plan: "pro"

customers:
  - id: "cus_default_1"
    email: "alice@example.com"
    name: "Alice Smith"
    phone: "+15551234567"
    description: "Default test customer"
    balance: 0
    metadata:
      tier: "gold"

webhooks:
  - id: "wh_default_1"
    url: "http://localhost:3000/webhooks/stripe"
    secret: "whsec_test_secret"
    events:
      - "*"
    description: "Primary local webhook listener"
    metadata:
      env: "local"

testing:
  delay_ms: 0
  webhook_delay_ms: 0
  always_fail: false
  always_fail_code: "processing_error"
  always_fail_message: "Payment processing failed due to test configuration"
  declined_cards:
    "4000000000000002":
      code: "card_declined"
      decline_code: "generic_decline"
      message: "Your card was declined."
```

---

## Configuration Options

### `port` (integer, optional)

The port on which the server will listen.

- **Type**: Integer
- **Default**: `8080`
- **Environment Variable Override**: `PORT`
- **Example**: `port: 8080`

If the `PORT` environment variable is set, it will override the value in the configuration file.

---

### `base_url` (string, optional)

The public base URL of the payment provider used when constructing hosted checkout URLs and receipt URLs.

- **Type**: String
- **Default**: `http://localhost:<port>`
- **Example**: `base_url: http://localhost:8080`

If not provided, defaults to `http://localhost:<port>` using the configured port (trailing slashes are automatically stripped). This value is used to build URLs for:
- Hosted Payment Link checkout pages (`<base_url>/checkout/link/{id}`)
- Hosted Payment Intent checkout pages (`<base_url>/checkout/intent/{id}`)
- Hosted Checkout Session pages (`<base_url>/checkout/session/{id}`)
- Charge receipt URLs (`<base_url>/receipts/{charge_id}`)

---

### `allowed_origins` (string, optional)

The allowed origins for Cross-Origin Resource Sharing (CORS) requests.

- **Type**: String
- **Default**: `*` (all origins allowed)
- **Example**: `allowed_origins: http://localhost:3000`

This value configures CORS headers for cross-origin requests. The server will:
- Set the `Access-Control-Allow-Origin` header to the configured value
- Handle preflight `OPTIONS` requests automatically (`204 No Content`)
- Allow credentials and standard/Stripe headers (`Authorization`, `Content-Type`, `Stripe-Version`, `Idempotency-Key`, `X-Api-Key`) for cross-origin requests

Common configurations:
- `*` - Allow all origins (default, convenient for local testing)
- `http://localhost:3000` - Allow a single specific origin
- `http://localhost:3000,https://example.com` - Allow multiple specific origins

---

### `webhook_secret` (string, optional)

The default signing secret used to sign outgoing webhook payloads when a webhook endpoint does not specify its own `secret`.

- **Type**: String
- **Default**: `"whsec_test_secret"`
- **Example**: `webhook_secret: "whsec_test_secret"`

All outgoing webhook `POST` requests include:
- `Stripe-Signature: t=<unix_timestamp>,v1=<hmac_sha256_hex>` (HMAC-SHA256 over `<unix_timestamp>.<raw_json_body>`, compatible with official Stripe SDKs such as `stripe.webhooks.constructEvent`)
- `X-Payment-Signature: sha256=<hmac_sha256_hex>` (HMAC-SHA256 over `<raw_json_body>`)

---

### `api_keys` (array, optional)

List of accepted API keys for authenticating requests to `/payments/...` and `/v1/...` endpoints.

- **Type**: Array of strings
- **Default**: Empty `[]` (all requests allowed without authentication)
- **Example**:
  ```yaml
  api_keys:
    - "sk_test_123456"
    - "sk_test_abcdef"
  ```

**Behavior:**
- When `api_keys` is omitted or empty (`[]`), the server accepts any API key or unauthenticated requests for frictionless local development.
- When one or more keys are configured in `api_keys`, every request to `/payments/...` and `/v1/...` must supply a matching key via one of:
  - `Authorization: Bearer <key>`
  - `Authorization: Basic <base64(key:)>` (standard Stripe Basic Auth with the API key as the username)
  - `X-Api-Key: <key>`
- Non-matching or missing credentials return `401 Unauthorized`.

---

### `native_api` (object, optional)

Configuration options for the provider-agnostic Native Payment API (`/payments/...`).

- **Type**: Object
- **Default**: `{ enabled: true }`

#### NativeApi Object Properties

##### `enabled` (boolean, optional)

Whether the Native Payment API endpoints are enabled.

- **Type**: Boolean
- **Default**: `true`
- **Example**: `enabled: true`

When enabled, the following endpoints are available:
- `GET /payments/links`
- `POST /payments/links`
- `GET /payments/links/{id}`
- `POST /payments/intents`
- `GET /payments/intents`
- `GET /payments/intents/{id}`
- `POST /payments/intents/{id}/confirm`
- `POST /payments/intents/{id}/cancel`
- `POST /payments/intents/{id}/refund`
- `GET /payments/refunds`
- `GET /payments/refunds/{id}`

When disabled (`enabled: false`), these endpoints will not be registered.

#### NativeApi Example

```yaml
native_api:
  enabled: true
```

---

### `stripe_api` (object, optional)

Configuration options for the Stripe-Compatible API (`/v1/...`).

- **Type**: Object
- **Default**: `{ enabled: true }`

#### StripeApi Object Properties

##### `enabled` (boolean, optional)

Whether the Stripe-compatible `/v1/...` API endpoints are enabled.

- **Type**: Boolean
- **Default**: `true`
- **Example**: `enabled: true`

When enabled, the following endpoints are available:
- `/v1/payment_intents` (`POST`, `GET`, `GET /{id}`, `POST /{id}`, `POST /{id}/confirm`, `POST /{id}/cancel`, `POST /{id}/capture`)
- `/v1/payment_links` (`POST`, `GET`, `GET /{id}`, `POST /{id}`)
- `/v1/checkout/sessions` (`POST`, `GET`, `GET /{id}`, `POST /{id}/expire`)
- `/v1/customers` (`POST`, `GET`, `GET /{id}`, `POST /{id}`, `DELETE /{id}`)
- `/v1/payment_methods` (`POST`, `GET`, `GET /{id}`, `POST /{id}/attach`, `POST /{id}/detach`)
- `/v1/charges` (`GET`, `GET /{id}`)
- `/v1/refunds` (`POST`, `GET`, `GET /{id}`)
- `/v1/events` (`GET`, `GET /{id}`)
- `/v1/webhook_endpoints` (`POST`, `GET`, `GET /{id}`, `DELETE /{id}`)

When disabled (`enabled: false`), these endpoints will not be registered.

#### StripeApi Example

```yaml
stripe_api:
  enabled: true
```

---

### `payment_links` (array, optional)

An array of pre-configured Payment Links available immediately when the server starts and restored whenever `POST /_testing/reset` is called.

- **Type**: Array of PaymentLink objects
- **Default**: Empty `[]`

#### PaymentLink Object Properties

Each payment link in the array supports the following properties:

##### `id` (string, optional)

Unique identifier for the payment link. Used in the hosted checkout URL (`/checkout/link/{id}`) and API lookups.

- **Type**: String
- **Default**: Auto-generated with `plink_` prefix if omitted
- **Example**: `id: "plink_pro_plan"`

##### `title` (string, optional)

Title displayed on the hosted HTML checkout page.

- **Type**: String
- **Default**: `"Payment"`
- **Example**: `title: "Pro Plan Subscription"`

##### `description` (string, optional)

Description displayed beneath the title on the hosted HTML checkout page and copied to created PaymentIntents.

- **Type**: String
- **Default**: `""`
- **Example**: `description: "Monthly subscription to Pro Plan"`

##### `amount` (integer, required)

The payment amount in the smallest currency unit (e.g., cents for USD/EUR).

- **Type**: Integer (`int64`)
- **Required**: Yes (must be `> 0`)
- **Example**: `amount: 2999` ($29.99)

##### `currency` (string, required)

Three-letter ISO 4217 currency code in lowercase.

- **Type**: String
- **Required**: Yes
- **Example**: `currency: "usd"`

##### `active` (boolean, optional)

Whether the payment link is active and can be used for checkout. Inactive payment links return `404 Not Found` on `/checkout/link/{id}`.

- **Type**: Boolean
- **Default**: `true`
- **Example**: `active: true`

##### `callback_url` (string, optional)

Fallback redirect URL used after checkout completion or cancellation when `success_url` or `cancel_url` is not set.

- **Type**: String
- **Default**: `""`
- **Example**: `callback_url: "http://localhost:3000/payment/callback"`

##### `success_url` (string, optional)

Redirect URL after a customer completes a successful payment on `/checkout/link/{id}`. Also populates Stripe's `after_completion.redirect.url`. Supports the `{CHECKOUT_SESSION_ID}` template placeholder or receives query parameters `?status=succeeded&payment_intent_id={pi_id}&payment_link_id={id}`.

- **Type**: String
- **Default**: `""`
- **Example**: `success_url: "http://localhost:3000/payment/success"`

##### `cancel_url` (string, optional)

Redirect URL when a customer cancels payment on `/checkout/link/{id}`. Receives query parameters `?status=canceled&payment_intent_id={pi_id}&payment_link_id={id}`.

- **Type**: String
- **Default**: `""`
- **Example**: `cancel_url: "http://localhost:3000/payment/cancel"`

##### `webhook_url` (string, optional)

Per-link webhook URL that receives signed event payloads (`payment_intent.succeeded`, `charge.succeeded`, `payment_link.paid`, `payment_intent.payment_failed`, `payment_intent.canceled`) whenever a payment is processed through this link.

- **Type**: String
- **Default**: `""`
- **Example**: `webhook_url: "http://localhost:3000/webhooks/payment"`

##### `metadata` (object, optional)

Arbitrary string key-value pairs attached to the payment link and propagated to PaymentIntents created from this link.

- **Type**: Object (map of string keys to string values)
- **Default**: `{}`
- **Example**:
  ```yaml
  metadata:
    plan: "pro"
    billing_cycle: "monthly"
  ```

#### PaymentLink Example

```yaml
payment_links:
  - id: "plink_pro_plan"
    title: "Pro Plan Subscription"
    description: "Monthly subscription to Pro Plan"
    amount: 2999
    currency: "usd"
    active: true
    callback_url: "http://localhost:3000/payment/callback"
    success_url: "http://localhost:3000/payment/success"
    cancel_url: "http://localhost:3000/payment/cancel"
    webhook_url: "http://localhost:3000/webhooks/payment"
    metadata:
      plan: "pro"
```

---

### `customers` (array, optional)

An array of pre-configured Customer objects seeded at startup and restored on `POST /_testing/reset`.

- **Type**: Array of Customer objects
- **Default**: Empty `[]`

#### Customer Object Properties

##### `id` (string, optional)

Unique identifier for the customer.

- **Type**: String
- **Default**: Auto-generated with `cus_` prefix if omitted
- **Example**: `id: "cus_default_1"`

##### `email` (string, optional)

The customer's email address.

- **Type**: String
- **Default**: `null`
- **Example**: `email: "alice@example.com"`

##### `name` (string, optional)

The customer's full name or business name.

- **Type**: String
- **Default**: `null`
- **Example**: `name: "Alice Smith"`

##### `phone` (string, optional)

The customer's phone number.

- **Type**: String
- **Default**: `null`
- **Example**: `phone: "+15551234567"`

##### `description` (string, optional)

An arbitrary string attached to the customer object.

- **Type**: String
- **Default**: `null`
- **Example**: `description: "Enterprise account customer"`

##### `balance` (integer, optional)

Current customer balance in cents.

- **Type**: Integer (`int64`)
- **Default**: `0`
- **Example**: `balance: 0`

##### `metadata` (object, optional)

Set of key-value pairs attached to the customer object.

- **Type**: Object (map of string keys to string values)
- **Default**: `{}`
- **Example**:
  ```yaml
  metadata:
    tier: "gold"
  ```

#### Customer Example

```yaml
customers:
  - id: "cus_default_1"
    email: "alice@example.com"
    name: "Alice Smith"
    phone: "+15551234567"
    description: "VIP Customer"
    balance: 0
    metadata:
      tier: "gold"
```

---

### `webhooks` (array, optional)

An array of pre-configured webhook endpoints that receive signed event notifications whenever matching events occur.

- **Type**: Array of Webhook objects
- **Default**: Empty `[]`

#### Webhook Object Properties

##### `id` (string, optional)

Unique identifier for the webhook endpoint.

- **Type**: String
- **Default**: Auto-generated with `we_` prefix if omitted
- **Example**: `id: "wh_default_1"`

##### `url` (string, required)

The destination URL where webhook `POST` requests are delivered.

- **Type**: String
- **Required**: Yes
- **Example**: `url: "http://localhost:3000/webhooks/stripe"`

##### `secret` (string, optional)

Signing secret for this specific webhook endpoint. If omitted, falls back to the global `webhook_secret`.

- **Type**: String
- **Default**: Value of `webhook_secret` (`"whsec_test_secret"`)
- **Example**: `secret: "whsec_custom_endpoint_secret"`

##### `events` / `enabled_events` (array of strings, optional)

List of event types this endpoint subscribes to. Both `events` and `enabled_events` are supported in YAML configuration. Use `["*"]` to receive all events.

- **Type**: Array of strings
- **Default**: `["*"]` (if both `events` and `enabled_events` are omitted)
- **Example**:
  ```yaml
  events:
    - "payment_intent.succeeded"
    - "payment_intent.payment_failed"
    - "charge.refunded"
  ```

Supported event types include:
- `payment_intent.created`
- `payment_intent.succeeded`
- `payment_intent.payment_failed`
- `payment_intent.canceled`
- `charge.succeeded`
- `charge.refunded`
- `checkout.session.completed`
- `payment_link.paid`
- `*` (wildcard matching all events)

##### `description` (string, optional)

Optional description of the webhook endpoint.

- **Type**: String
- **Default**: `null`
- **Example**: `description: "Main backend webhook receiver"`

##### `metadata` (object, optional)

Set of key-value pairs attached to the webhook endpoint.

- **Type**: Object (map of string keys to string values)
- **Default**: `{}`

#### Webhook Example

```yaml
webhooks:
  - id: "wh_default_1"
    url: "http://localhost:3000/webhooks/stripe"
    secret: "whsec_test_secret"
    events:
      - "*"
    description: "Catch-all webhook endpoint"
```

---

### `testing` (object, optional)

Configuration for simulating latency, forced payment failures, and card decline rules. Can also be inspected and modified at runtime via `GET /_testing/config` and `PUT /_testing/config`.

- **Type**: Object

#### Testing Object Properties

##### `delay_ms` (integer, optional)

Artificial delay in milliseconds applied before responding to API and checkout requests. Useful for testing client timeouts and loading states.

- **Type**: Integer
- **Default**: `0`
- **Example**: `delay_ms: 250`

##### `webhook_delay_ms` (integer, optional)

Artificial delay in milliseconds applied before dispatching outgoing webhook HTTP requests.

- **Type**: Integer
- **Default**: `0`
- **Example**: `webhook_delay_ms: 500`

##### `always_fail` (boolean, optional)

When `true`, every payment confirmation or checkout submission fails regardless of the card number or payment method used.

- **Type**: Boolean
- **Default**: `false`
- **Example**: `always_fail: false`

##### `always_fail_code` (string, optional)

Error code returned when `always_fail` is `true`.

- **Type**: String
- **Default**: `"processing_error"`
- **Example**: `always_fail_code: "card_declined"`

##### `always_fail_message` (string, optional)

Human-readable error message returned when `always_fail` is `true`.

- **Type**: String
- **Default**: `"Payment processing failed due to test configuration"`
- **Example**: `always_fail_message: "Simulated payment processor outage"`

##### `declined_cards` (object, optional)

Map of card numbers to decline rules (`code`, `decline_code`, `message`). Any custom card numbers defined here are merged with the built-in default declined cards (custom entries with the same card number override the built-in rule).

- **Type**: Object (map of card number string to `DeclinedCardRule` object)
- **Each rule has**:
  - `code` (string): Stripe error code (e.g., `"card_declined"`, `"expired_card"`, `"processing_error"`)
  - `decline_code` (string): Specific decline reason (e.g., `"generic_decline"`, `"insufficient_funds"`, `"expired_card"`, `"fraudulent"`)
  - `message` (string): Error message returned in API responses and displayed on the hosted checkout UI

#### Built-in Test Cards & PaymentMethod Tokens

The payment provider includes built-in support for standard test card numbers and Stripe-compatible `pm_card_*` tokens:

| Card Number        | PaymentMethod Token                       | Outcome / Code     | Decline Code         | Message                                          |
| ------------------ | ----------------------------------------- | ------------------ | -------------------- | ------------------------------------------------ |
| `4242424242424242` | `pm_card_visa`                            | **Succeeded**      | —                    | Payment succeeds (Visa, last4 `4242`)            |
| `5555555555554444` | `pm_card_mastercard`                      | **Succeeded**      | —                    | Payment succeeds (Mastercard, last4 `4444`)      |
| `4000000000000002` | `pm_card_chargeDeclined`                  | `card_declined`    | `generic_decline`    | `"Your card was declined."`                      |
| `4000000000009995` | `pm_card_chargeDeclinedInsufficientFunds` | `card_declined`    | `insufficient_funds` | `"Your card has insufficient funds."`            |
| `4000000000000069` | `pm_card_chargeDeclinedExpiredCard`       | `expired_card`     | `expired_card`       | `"Your card has expired."`                       |
| `4000000000000119` | `pm_card_processingError`                 | `processing_error` | `processing_error`   | `"An error occurred while processing your card."`|

#### Testing Example

```yaml
testing:
  delay_ms: 0
  webhook_delay_ms: 0
  always_fail: false
  always_fail_code: "processing_error"
  always_fail_message: "Payment processing failed due to test configuration"
  declined_cards:
    "4000000000000002":
      code: "card_declined"
      decline_code: "generic_decline"
      message: "Your card was declined."
    "4000000000009995":
      code: "card_declined"
      decline_code: "insufficient_funds"
      message: "Your card has insufficient funds."
    "4000000000000069":
      code: "expired_card"
      decline_code: "expired_card"
      message: "Your card has expired."
    "4000000000000119":
      code: "processing_error"
      decline_code: "processing_error"
      message: "An error occurred while processing your card."
    "4111111111111119":
      code: "card_declined"
      decline_code: "fraudulent"
      message: "Your card was blocked for suspected fraud."
```

---

## Environment Variables

### `CONFIG_PATH`

Path to the YAML configuration file.

- **Default**: `/config.yaml`
- **Example**: `CONFIG_PATH=/app/config/payment-provider.yaml`

### `PORT`

Server port (overrides the `port` setting in the configuration file).

- **Default**: `8080` (if not specified in config file)
- **Example**: `PORT=9000`

---

## Complete Configuration Example

```yaml
# Server configuration
port: 8080
base_url: http://localhost:8080
allowed_origins: "*"
webhook_secret: "whsec_test_secret"

# Optional API key enforcement (leave empty to allow any key)
api_keys: []

# Enable or disable API surfaces
native_api:
  enabled: true

stripe_api:
  enabled: true

# Pre-configured Payment Links
payment_links:
  - id: "plink_pro_plan"
    title: "Pro Plan Subscription"
    description: "Monthly subscription to Pro Plan"
    amount: 2999
    currency: "usd"
    active: true
    callback_url: "http://localhost:3000/payment/callback"
    success_url: "http://localhost:3000/payment/success"
    cancel_url: "http://localhost:3000/payment/cancel"
    webhook_url: "http://localhost:3000/webhooks/payment"
    metadata:
      plan: "pro"

  - id: "plink_starter_pack"
    title: "Starter Pack"
    description: "One-time starter package"
    amount: 999
    currency: "eur"
    active: true
    callback_url: "http://localhost:3000/payment/callback"
    metadata:
      plan: "starter"

# Pre-configured Customers
customers:
  - id: "cus_default_1"
    email: "alice@example.com"
    name: "Alice Smith"
    phone: "+15551234567"
    description: "Alice - Pro Plan Customer"
    balance: 0
    metadata:
      tier: "gold"

  - id: "cus_default_2"
    email: "bob@example.com"
    name: "Bob Johnson"
    metadata:
      tier: "standard"

# Pre-configured Webhook Endpoints
webhooks:
  - id: "wh_default_1"
    url: "http://localhost:3000/webhooks/stripe"
    secret: "whsec_test_secret"
    events:
      - "*"
    description: "All payment events"

# Testing & Simulation settings
testing:
  delay_ms: 0
  webhook_delay_ms: 0
  always_fail: false
  always_fail_code: "processing_error"
  always_fail_message: "Payment processing failed due to test configuration"
  declined_cards:
    "4000000000000002":
      code: "card_declined"
      decline_code: "generic_decline"
      message: "Your card was declined."
    "4000000000009995":
      code: "card_declined"
      decline_code: "insufficient_funds"
      message: "Your card has insufficient funds."
    "4000000000000069":
      code: "expired_card"
      decline_code: "expired_card"
      message: "Your card has expired."
    "4000000000000119":
      code: "processing_error"
      decline_code: "processing_error"
      message: "An error occurred while processing your card."
```

---

## Docker Configuration

When running in Docker, mount your configuration file to `/config.yaml` or set the `CONFIG_PATH` environment variable:

```yaml
services:
  local-payment-provider:
    image: siocode/local-payment-provider
    ports:
      - "8080:8080"
    volumes:
      - ./my-config.yaml:/config.yaml:ro
    environment:
      - PORT=8080
```

Or with a custom path:

```yaml
services:
  local-payment-provider:
    image: siocode/local-payment-provider
    ports:
      - "9000:9000"
    volumes:
      - ./configs/payment.yaml:/app/payment-config.yaml:ro
    environment:
      - CONFIG_PATH=/app/payment-config.yaml
      - PORT=9000
```

---

## Notes

- **All data is in-memory**: Payment links, payment intents, checkout sessions, customers, payment methods, charges, refunds, events, and logs are stored in memory and will be lost when the server restarts.
- **State reset support**: You can reset all in-memory state back to the initial YAML configuration at any time by calling `POST /_testing/reset`.
- **Runtime test mutation**: All settings under `testing` can be modified on the fly via `PUT /_testing/config` without restarting the container.
- **No real payment processing**: This provider is designed for local development, integration testing, and E2E testing only—never use real card numbers.

---

## 📄 License

MIT © 2026 SIOCODE

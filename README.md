# Local Payment Provider

A fully in-memory, configurable **local payment provider** for development and testing of payment flows, checkout sessions, payment links, and webhook integrations.

Supports:

- **Hosted Payment Links** (pre-configured in YAML or created dynamically via API)
- **Payment Intents** (created by application backend, confirmed via hosted HTML UI or direct API)
- **Native Payment API** (`/payments/...`) for simple, provider-agnostic payment integrations
- **Stripe-Compatible API** (`/v1/...`) supporting both `application/x-www-form-urlencoded` (official Stripe SDKs) and `application/json`
- **Callback Redirects & Signed Webhooks** (`Stripe-Signature` and `X-Payment-Signature`)
- **Testing & Spying API** (`/_testing/...`) for inspecting requests, payments, webhook deliveries, and mutating runtime test behaviors
- Dockerized and architecture-portable (x86_64 and arm64)

## 🚀 Purpose

This project simulates a **realistic payment provider** (like Stripe, SimplePay, or Adyen) that you can run locally without any external dependencies or cloud accounts. It's perfect for:

- Testing **Payment Link** and **Payment Intent** checkout flows end-to-end
- Using official **Stripe SDKs** (`stripe-node`, `stripe-go`, `stripe-python`) pointed at a local base URL
- Simulating **card declines**, insufficient funds, expired cards, artificial network delays, and forced failures
- Verifying **webhook delivery**, signature verification, and **redirect callback** handling
- Running deterministic E2E tests in CI/CD or air-gapped environments

All state is **in-memory** and initialized via a simple YAML config file.

## 🐳 Docker

Pre-built images available at:

```
docker pull siocode/local-payment-provider
```

Available for:
- `linux/amd64` (x86 64-bit)
- `linux/arm64` (ARM 64-bit)

## 🛠️ Configuration

YAML configuration is passed via `-c` / `--config-path` CLI flag, `CONFIG_PATH` env var, or defaults to `/config.yaml`.

See [`CONFIG.md`](./CONFIG.md) for full details.

### Example: `example/local-payment-provider.config.yaml`

```yaml
port: 8080
base_url: http://localhost:8080
allowed_origins: "*"
webhook_secret: "whsec_test_secret"

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
    metadata:
      tier: "gold"

webhooks:
  - id: "wh_default_1"
    url: "http://localhost:3000/webhooks/stripe"
    secret: "whsec_test_secret"
    events:
      - "*"

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
```

## 🔐 Endpoints

[📄 See `API.md` for a complete reference of the API endpoints](./API.md)

### 🔧 Health & Metadata

| Method | Path       | Description  |
| ------ | ---------- | ------------ |
| GET    | `/healthz` | Health check |

### 🖥️ Hosted Checkout UI (Browser Flows)

| Method | Path                            | Description                                         |
| ------ | ------------------------------- | --------------------------------------------------- |
| GET    | `/checkout/link/{id}`           | Render HTML checkout page for a Payment Link        |
| POST   | `/checkout/link/{id}/submit`    | Submit payment or cancel for a Payment Link         |
| GET    | `/checkout/intent/{id}`         | Render HTML checkout page for a Payment Intent      |
| POST   | `/checkout/intent/{id}/submit`  | Submit payment or cancel for a Payment Intent       |
| GET    | `/checkout/session/{id}`        | Render HTML checkout page for a Checkout Session    |
| POST   | `/checkout/session/{id}/submit` | Submit payment or cancel for a Checkout Session     |

### 💳 Native Payment API

Available when `native_api.enabled: true` (default).

| Method | Path                             | Description                                        |
| ------ | -------------------------------- | -------------------------------------------------- |
| GET    | `/payments/links`                | List all payment links                             |
| POST   | `/payments/links`                | Create a payment link                              |
| GET    | `/payments/links/{id}`           | Get a payment link by ID                           |
| POST   | `/payments/intents`              | Create a payment intent                            |
| GET    | `/payments/intents`              | List all payment intents                           |
| GET    | `/payments/intents/{id}`         | Get a payment intent by ID                         |
| POST   | `/payments/intents/{id}/confirm` | Confirm and pay a payment intent via API           |
| POST   | `/payments/intents/{id}/cancel`  | Cancel a payment intent                            |
| POST   | `/payments/intents/{id}/refund`  | Refund a succeeded payment intent                  |
| GET    | `/payments/refunds`              | List all refunds                                   |
| GET    | `/payments/refunds/{id}`         | Get a refund by ID                                 |

### ⚡ Stripe-Compatible API (`/v1`)

Available when `stripe_api.enabled: true` (default). Accepts both `application/x-www-form-urlencoded` and `application/json`.

| Method | Path                                    | Description                                  |
| ------ | --------------------------------------- | -------------------------------------------- |
| POST   | `/v1/payment_intents`                   | Create a Stripe PaymentIntent                |
| GET    | `/v1/payment_intents`                   | List Stripe PaymentIntents                   |
| GET    | `/v1/payment_intents/{id}`              | Retrieve a Stripe PaymentIntent              |
| POST   | `/v1/payment_intents/{id}`              | Update a Stripe PaymentIntent                |
| POST   | `/v1/payment_intents/{id}/confirm`      | Confirm a Stripe PaymentIntent               |
| POST   | `/v1/payment_intents/{id}/cancel`       | Cancel a Stripe PaymentIntent                |
| POST   | `/v1/payment_intents/{id}/capture`      | Capture an uncaptured PaymentIntent          |
| POST   | `/v1/payment_links`                     | Create a Stripe PaymentLink                  |
| GET    | `/v1/payment_links`                     | List Stripe PaymentLinks                     |
| GET    | `/v1/payment_links/{id}`                | Retrieve a Stripe PaymentLink                |
| POST   | `/v1/payment_links/{id}`                | Update a Stripe PaymentLink                  |
| POST   | `/v1/checkout/sessions`                 | Create a Stripe Checkout Session             |
| GET    | `/v1/checkout/sessions`                 | List Stripe Checkout Sessions                |
| GET    | `/v1/checkout/sessions/{id}`            | Retrieve a Stripe Checkout Session           |
| POST   | `/v1/checkout/sessions/{id}/expire`     | Expire a Stripe Checkout Session             |
| POST   | `/v1/customers`                         | Create a Stripe Customer                     |
| GET    | `/v1/customers`                         | List Stripe Customers                        |
| GET    | `/v1/customers/{id}`                    | Retrieve a Stripe Customer                   |
| POST   | `/v1/customers/{id}`                    | Update a Stripe Customer                     |
| DELETE | `/v1/customers/{id}`                    | Delete a Stripe Customer                     |
| POST   | `/v1/payment_methods`                   | Create a Stripe PaymentMethod                |
| GET    | `/v1/payment_methods`                   | List Stripe PaymentMethods                   |
| GET    | `/v1/payment_methods/{id}`              | Retrieve a Stripe PaymentMethod              |
| POST   | `/v1/payment_methods/{id}/attach`       | Attach PaymentMethod to a Customer           |
| POST   | `/v1/payment_methods/{id}/detach`       | Detach PaymentMethod from a Customer         |
| GET    | `/v1/charges`                           | List Stripe Charges                          |
| GET    | `/v1/charges/{id}`                      | Retrieve a Stripe Charge                     |
| POST   | `/v1/refunds`                           | Create a Stripe Refund                       |
| GET    | `/v1/refunds`                           | List Stripe Refunds                          |
| GET    | `/v1/refunds/{id}`                      | Retrieve a Stripe Refund                     |
| GET    | `/v1/events`                            | List Stripe Events                           |
| GET    | `/v1/events/{id}`                       | Retrieve a Stripe Event                      |
| POST   | `/v1/webhook_endpoints`                 | Create a Stripe WebhookEndpoint              |
| GET    | `/v1/webhook_endpoints`                 | List Stripe WebhookEndpoints                 |
| GET    | `/v1/webhook_endpoints/{id}`            | Retrieve a Stripe WebhookEndpoint            |
| DELETE | `/v1/webhook_endpoints/{id}`            | Delete a Stripe WebhookEndpoint              |

### 🧪 Testing & Spying API

| Method | Path                         | Description                                                 |
| ------ | ---------------------------- | ----------------------------------------------------------- |
| GET    | `/_testing/config`           | Get active runtime testing configuration                    |
| PUT    | `/_testing/config`           | Update runtime testing configuration (delays, card errors)  |
| GET    | `/_testing/requests`         | List recorded API requests for test assertions              |
| GET    | `/_testing/payments`         | List all payment intents and transactions                   |
| GET    | `/_testing/webhooks`         | List all webhook delivery attempts and responses            |
| POST   | `/_testing/webhooks/trigger` | Manually trigger and deliver a synthetic webhook event      |
| POST   | `/_testing/reset`            | Reset all in-memory state and logs back to initial config   |

## 🔔 Webhooks & Callbacks

`local-payment-provider` supports both **redirect callbacks** and **webhook notifications**:

- **Redirect Callbacks**: When a user completes or cancels a payment on the hosted checkout UI (`/checkout/...`), the server redirects (`302 Found`) to the configured `success_url`, `cancel_url`, or `callback_url` with query parameters (`payment_intent_id`, `status`, `payment_link_id`, `session_id`, `state`).
- **Signed Webhooks**: When payment state changes (e.g. `payment_intent.created`, `payment_intent.succeeded`, `payment_intent.payment_failed`, `payment_intent.canceled`, `charge.succeeded`, `charge.refunded`, `checkout.session.completed`, `payment_link.paid`), the server sends a `POST` request to all matching configured webhook endpoints as well as any per-request/per-link `webhook_url`.
- **Signatures**: Every webhook request includes both:
  - `Stripe-Signature: t=<timestamp>,v1=<hmac_sha256_hex>` (compatible with `stripe.webhooks.constructEvent`)
  - `X-Payment-Signature: sha256=<hmac_sha256_hex>`

## 🧪 Example Docker Compose

### `example/docker-compose.yaml`

```yaml
services:
  local-payment-provider:
    image: siocode/local-payment-provider
    ports:
      - "8080:8080"
    volumes:
      - ./local-payment-provider.config.yaml:/config.yaml:ro
```

Start it:

```bash
cd example
docker compose up
```

## 🏗️ Building

Build the container via the following command (builds for both X86 & ARM architectures):

```bash
docker buildx build --platform=linux/arm64,linux/amd64 -t siocode/local-payment-provider:latest .
```

## 🧪 Testing

End-to-end tests are in the `e2e/` folder using Mocha + Chai + Docker Compose. To run:

```bash
cd e2e
npm install
npm test
```

## 📜 License

MIT © 2026 SIOCODE

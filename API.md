# API Reference – SIOCODE Local Payment Provider

This document describes all API endpoints exposed by the SIOCODE Local Payment Provider.

**Note:** Some endpoints can be disabled via configuration:
- Native Payment API endpoints can be disabled by setting `native_api.enabled: false` in the configuration
- Stripe-Compatible API endpoints can be disabled by setting `stripe_api.enabled: false` in the configuration

---

## 🧠 Health Check

### `GET /healthz`

Returns a health check response.

**Response:**
```json
{
  "status": "OK"
}
```

---

## 🖥️ Hosted Checkout UI

### `GET /checkout/link/{id}`

Displays an HTML checkout form for a Payment Link.

**Path Parameters:**

| Parameter | Type   | Description         |
| --------- | ------ | ------------------- |
| `id`      | string | The Payment Link ID |

**Query Parameters:**

| Parameter        | Type   | Required | Description                                                                 |
| ---------------- | ------ | -------- | --------------------------------------------------------------------------- |
| `callback_url`   | string | No       | Overrides the Payment Link's default `callback_url`                         |
| `success_url`    | string | No       | Overrides the Payment Link's default `success_url`                          |
| `cancel_url`     | string | No       | Overrides the Payment Link's default `cancel_url`                           |
| `state`          | string | No       | Opaque value used to maintain state across form submission and redirect     |
| `customer_email` | string | No       | Pre-fills the customer email field on the checkout form                     |

**Response:**

Returns an HTML checkout page (`text/html; charset=utf-8`) containing a `<form method="POST" action="/checkout/link/{id}/submit">` displaying the payment link's `title`, `description`, `amount`, and `currency`.

**Errors:**

- `404 Not Found` - If the Payment Link does not exist or `active` is `false`

---

### `POST /checkout/link/{id}/submit`

Submits the checkout form for a Payment Link, creates a `PaymentIntent` (and `Charge` on success), dispatches webhook events (`payment_intent.succeeded`, `charge.succeeded`, `payment_link.paid` on success; `payment_intent.canceled` on cancel; `payment_intent.payment_failed` on failure), and redirects back to the client application.

**Path Parameters:**

| Parameter | Type   | Description         |
| --------- | ------ | ------------------- |
| `id`      | string | The Payment Link ID |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Form Parameters:**

| Parameter           | Type   | Required | Description                                                                                      |
| ------------------- | ------ | -------- | ------------------------------------------------------------------------------------------------ |
| `action`            | string | No       | `"pay"` (default) to process payment, or `"cancel"` to cancel checkout                           |
| `card_number`       | string | No       | Card number (defaults to `"4242424242424242"` if `payment_method` is not provided)               |
| `exp_month`         | string | No       | Card expiration month (default: `"12"`)                                                          |
| `exp_year`          | string | No       | Card expiration year (default: `"2030"`)                                                         |
| `cvc`               | string | No       | Card CVC security code (default: `"123"`)                                                        |
| `cardholder_name`   | string | No       | Cardholder name                                                                                  |
| `customer_email`    | string | No       | Customer email address                                                                           |
| `payment_method`    | string | No       | PaymentMethod ID or test token (e.g. `"pm_card_visa"`, `"pm_card_chargeDeclined"`)               |
| `callback_url`      | string | No       | Overrides the Payment Link's `callback_url`                                                      |
| `success_url`       | string | No       | Overrides the Payment Link's `success_url`                                                       |
| `cancel_url`        | string | No       | Overrides the Payment Link's `cancel_url`                                                        |
| `state`             | string | No       | Opaque state string appended to the redirect URL                                                 |
| `redirect_on_error` | string | No       | If `"true"`, redirects to `callback_url` on payment failure instead of re-rendering the HTML form |

**Response:**

- `302 Found` (when `action=pay` succeeds and a redirect URL is configured via `success_url`, `after_completion.redirect.url`, or `callback_url`) - Redirects to:
  `{target_url}?status=succeeded&payment_intent_id={pi_id}&payment_link_id={id}` (plus `&state={state}` if `state` was provided; if `{target_url}` contains `{CHECKOUT_SESSION_ID}`, it is replaced with the generated session/intent ID)
- `302 Found` (when `action=cancel` and a redirect URL is configured via `cancel_url` or `callback_url`) - Redirects to:
  `{target_url}?status=canceled&payment_intent_id={pi_id}&payment_link_id={id}` (plus `&state={state}` if `state` was provided)
- `302 Found` (when `action=pay` fails and `redirect_on_error=true` with a `callback_url`) - Redirects to:
  `{callback_url}?status=failed&payment_intent_id={pi_id}&payment_link_id={id}&error_code={code}&error_message={url_encoded_message}` (plus `&state={state}` if `state` was provided)
- `200 OK` (`text/html; charset=utf-8`) - Renders an HTML confirmation or cancellation page if no redirect URL is configured

**Errors:**

- `402 Payment Required` - Re-renders the HTML checkout form with `<div class="error">{error_message}</div>` if the card is declined (e.g. `4000000000000002` / `pm_card_chargeDeclined`, `4000000000009995` / `pm_card_chargeDeclinedInsufficientFunds`, `4000000000000069` / `pm_card_chargeDeclinedExpiredCard`, `4000000000000119` / `pm_card_processingError`, or custom `declined_cards`) or `always_fail` is enabled
- `404 Not Found` - If the Payment Link does not exist or `active` is `false`

---

### `GET /checkout/intent/{id}`

Displays an HTML checkout form for an existing Payment Intent.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The Payment Intent ID |

**Query Parameters:**

| Parameter      | Type   | Required | Description                                                               |
| -------------- | ------ | -------- | ------------------------------------------------------------------------- |
| `callback_url` | string | No       | Overrides the Payment Intent's `callback_url` / `return_url`              |
| `success_url`  | string | No       | Overrides the Payment Intent's `success_url`                              |
| `cancel_url`   | string | No       | Overrides the Payment Intent's `cancel_url`                               |
| `state`        | string | No       | Opaque value used to maintain state across form submission and redirect   |

**Response:**

Returns an HTML checkout page (`text/html; charset=utf-8`) containing a `<form method="POST" action="/checkout/intent/{id}/submit">` displaying the Payment Intent's `description`, `amount`, and `currency`.

**Errors:**

- `404 Not Found` - If the Payment Intent does not exist

---

### `POST /checkout/intent/{id}/submit`

Submits the checkout form for an existing Payment Intent, updates its status, dispatches webhook events (`payment_intent.succeeded` and `charge.succeeded` on success; `payment_intent.canceled` on cancel; `payment_intent.payment_failed` on failure), and redirects back to the client application.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The Payment Intent ID |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Form Parameters:**

| Parameter           | Type   | Required | Description                                                                                      |
| ------------------- | ------ | -------- | ------------------------------------------------------------------------------------------------ |
| `action`            | string | No       | `"pay"` (default) to confirm and pay, or `"cancel"` to cancel the Payment Intent                 |
| `card_number`       | string | No       | Card number (defaults to `"4242424242424242"` if `payment_method` is not provided)               |
| `exp_month`         | string | No       | Card expiration month (default: `"12"`)                                                          |
| `exp_year`          | string | No       | Card expiration year (default: `"2030"`)                                                         |
| `cvc`               | string | No       | Card CVC security code (default: `"123"`)                                                        |
| `cardholder_name`   | string | No       | Cardholder name                                                                                  |
| `customer_email`    | string | No       | Customer email address                                                                           |
| `payment_method`    | string | No       | PaymentMethod ID or test token (e.g. `"pm_card_visa"`, `"pm_card_chargeDeclined"`)               |
| `callback_url`      | string | No       | Overrides the Payment Intent's `callback_url` / `return_url`                                     |
| `success_url`       | string | No       | Overrides the Payment Intent's `success_url`                                                     |
| `cancel_url`        | string | No       | Overrides the Payment Intent's `cancel_url`                                                      |
| `state`             | string | No       | Opaque state string appended to the redirect URL                                                 |
| `redirect_on_error` | string | No       | If `"true"`, redirects to `callback_url` on payment failure instead of re-rendering the HTML form |

**Response:**

- `302 Found` (when `action=pay` succeeds and `success_url`, `callback_url`, or `return_url` is set) - Redirects to:
  `{target_url}?status=succeeded&payment_intent_id={id}` (plus `&state={state}` if `state` was provided)
- `302 Found` (when `action=cancel` and `cancel_url`, `callback_url`, or `return_url` is set) - Redirects to:
  `{target_url}?status=canceled&payment_intent_id={id}` (plus `&state={state}` if `state` was provided)
- `302 Found` (when `action=pay` fails and `redirect_on_error=true` with a redirect target URL) - Redirects to:
  `{target_url}?status=failed&payment_intent_id={id}&error_code={code}&error_message={url_encoded_message}` (plus `&state={state}` if `state` was provided)
- `200 OK` (`text/html; charset=utf-8`) - Renders an HTML confirmation or cancellation page if no redirect URL is configured

**Errors:**

- `402 Payment Required` - Re-renders the HTML checkout form with `<div class="error">{error_message}</div>` if the card is declined or `always_fail` is enabled
- `404 Not Found` - If the Payment Intent does not exist

---

### `GET /checkout/session/{id}`

Displays an HTML checkout form for a Stripe Checkout Session.

**Path Parameters:**

| Parameter | Type   | Description             |
| --------- | ------ | ----------------------- |
| `id`      | string | The Checkout Session ID |

**Response:**

Returns an HTML checkout page (`text/html; charset=utf-8`) containing a `<form method="POST" action="/checkout/session/{id}/submit">` displaying the session's `amount_total` and `currency`.

**Errors:**

- `404 Not Found` - If the Checkout Session does not exist

---

### `POST /checkout/session/{id}/submit`

Submits the checkout form for a Stripe Checkout Session, updates the session (`status: "complete"`, `payment_status: "paid"`) and its underlying `PaymentIntent` (`status: "succeeded"`), dispatches webhook events (`checkout.session.completed`, `payment_intent.succeeded`, `charge.succeeded`), and redirects to `success_url` or `cancel_url`.

**Path Parameters:**

| Parameter | Type   | Description             |
| --------- | ------ | ----------------------- |
| `id`      | string | The Checkout Session ID |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Form Parameters:**

| Parameter         | Type   | Required | Description                                                                        |
| ----------------- | ------ | -------- | ---------------------------------------------------------------------------------- |
| `action`          | string | No       | `"pay"` (default) to complete checkout, or `"cancel"` to cancel                    |
| `card_number`     | string | No       | Card number (defaults to `"4242424242424242"` if `payment_method` is not provided) |
| `exp_month`       | string | No       | Card expiration month (default: `"12"`)                                            |
| `exp_year`        | string | No       | Card expiration year (default: `"2030"`)                                           |
| `cvc`             | string | No       | Card CVC security code (default: `"123"`)                                          |
| `cardholder_name` | string | No       | Cardholder name                                                                    |
| `customer_email`  | string | No       | Customer email address                                                             |
| `payment_method`  | string | No       | PaymentMethod ID or test token                                                     |
| `state`           | string | No       | Opaque state string appended to the redirect URL                                   |

**Response:**

- `302 Found` (when `action=pay` succeeds) - Redirects to `success_url` (if `success_url` contains `{CHECKOUT_SESSION_ID}`, it is replaced with `{id}`; otherwise appends `?session_id={id}&payment_intent_id={pi_id}&status=succeeded`)
- `302 Found` (when `action=cancel`) - Redirects to `cancel_url` with `?session_id={id}&status=canceled`

**Errors:**

- `402 Payment Required` - Re-renders the HTML checkout form with `<div class="error">{error_message}</div>` if the card is declined or `always_fail` is enabled
- `404 Not Found` - If the Checkout Session does not exist

---

## 💳 Native Payment API

**Configuration:** These endpoints are available when `native_api.enabled: true` (default).

### `GET /payments/links`

Returns a list of all payment links.

**Response:**

```json
{
  "payment_links": [
    {
      "id": "plink_pro_plan",
      "object": "payment_link",
      "title": "Pro Plan Subscription",
      "description": "Monthly subscription to Pro Plan",
      "amount": 2999,
      "currency": "usd",
      "active": true,
      "url": "http://localhost:8080/checkout/link/plink_pro_plan",
      "callback_url": "http://localhost:3000/payment/callback",
      "success_url": "http://localhost:3000/payment/success",
      "cancel_url": "http://localhost:3000/payment/cancel",
      "webhook_url": "http://localhost:3000/webhooks/payment",
      "metadata": {
        "plan": "pro"
      },
      "created": 1700000000
    }
  ]
}
```

---

### `POST /payments/links`

Creates a new payment link.

**Content-Type:** `application/json`

**Request:**

```json
{
  "id": "plink_starter",
  "title": "Starter Pack",
  "description": "One-time starter pack purchase",
  "amount": 1500,
  "currency": "usd",
  "active": true,
  "callback_url": "http://localhost:3000/payment/callback",
  "success_url": "http://localhost:3000/payment/success",
  "cancel_url": "http://localhost:3000/payment/cancel",
  "webhook_url": "http://localhost:3000/webhooks/payment",
  "metadata": {
    "sku": "starter_01"
  }
}
```

**Request Fields:**

| Field          | Type    | Required | Description                                                              |
| -------------- | ------- | -------- | ------------------------------------------------------------------------ |
| `id`           | string  | No       | Custom ID (auto-generated with `plink_` prefix if omitted)               |
| `title`        | string  | No       | Title displayed on the hosted checkout page (default: `"Payment"`)       |
| `description`  | string  | No       | Description displayed on the hosted checkout page                        |
| `amount`       | integer | Yes      | Amount in smallest currency unit (e.g. cents, must be `> 0`)             |
| `currency`     | string  | Yes      | Three-letter ISO currency code in lowercase (e.g. `"usd"`)               |
| `active`       | boolean | No       | Whether the payment link is active (default: `true`)                     |
| `callback_url` | string  | No       | Redirect URL after checkout completion or cancellation                   |
| `success_url`  | string  | No       | Redirect URL after successful checkout                                   |
| `cancel_url`   | string  | No       | Redirect URL when checkout is canceled                                   |
| `webhook_url`  | string  | No       | Webhook URL to receive events for payments made via this link            |
| `metadata`     | object  | No       | Set of key-value pairs attached to the payment link                      |

**Response (`201 Created`):**

```json
{
  "id": "plink_starter",
  "object": "payment_link",
  "title": "Starter Pack",
  "description": "One-time starter pack purchase",
  "amount": 1500,
  "currency": "usd",
  "active": true,
  "url": "http://localhost:8080/checkout/link/plink_starter",
  "callback_url": "http://localhost:3000/payment/callback",
  "success_url": "http://localhost:3000/payment/success",
  "cancel_url": "http://localhost:3000/payment/cancel",
  "webhook_url": "http://localhost:3000/webhooks/payment",
  "metadata": {
    "sku": "starter_01"
  },
  "created": 1700000000
}
```

**Errors:**

- `400 Bad Request` - If `amount <= 0` or `currency` is missing

---

### `GET /payments/links/{id}`

Retrieves a specific payment link by ID.

**Path Parameters:**

| Parameter | Type   | Description         |
| --------- | ------ | ------------------- |
| `id`      | string | The Payment Link ID |

**Response:**

```json
{
  "id": "plink_pro_plan",
  "object": "payment_link",
  "title": "Pro Plan Subscription",
  "description": "Monthly subscription to Pro Plan",
  "amount": 2999,
  "currency": "usd",
  "active": true,
  "url": "http://localhost:8080/checkout/link/plink_pro_plan",
  "callback_url": "http://localhost:3000/payment/callback",
  "success_url": "http://localhost:3000/payment/success",
  "cancel_url": "http://localhost:3000/payment/cancel",
  "webhook_url": "http://localhost:3000/webhooks/payment",
  "metadata": {
    "plan": "pro"
  },
  "created": 1700000000
}
```

**Errors:**

- `404 Not Found` - If payment link does not exist

---

### `POST /payments/intents`

Creates a new payment intent. Emits a `payment_intent.created` webhook event. If `confirm: true` is set, immediately attempts confirmation and emits `payment_intent.succeeded` + `charge.succeeded` (or `payment_intent.payment_failed` on error).

**Content-Type:** `application/json`

**Request:**

```json
{
  "amount": 4999,
  "currency": "usd",
  "description": "Order #1042",
  "customer_id": "cus_default_1",
  "customer_email": "alice@example.com",
  "payment_method": "pm_card_visa",
  "confirm": false,
  "capture_method": "automatic",
  "callback_url": "http://localhost:3000/payment/callback",
  "success_url": "http://localhost:3000/payment/success",
  "cancel_url": "http://localhost:3000/payment/cancel",
  "webhook_url": "http://localhost:3000/webhooks/payment",
  "metadata": {
    "order_id": "1042"
  }
}
```

**Request Fields:**

| Field            | Type    | Required | Description                                                                           |
| ---------------- | ------- | -------- | ------------------------------------------------------------------------------------- |
| `amount`         | integer | Yes      | Amount in smallest currency unit (e.g. cents, must be `> 0`)                          |
| `currency`       | string  | Yes      | Three-letter ISO currency code in lowercase (e.g. `"usd"`)                            |
| `description`    | string  | No       | Arbitrary description of the payment                                                  |
| `customer_id`    | string  | No       | Customer ID (also accepted as `customer`)                                             |
| `customer_email` | string  | No       | Customer email address (also accepted as `receipt_email`)                             |
| `payment_method` | string  | No       | PaymentMethod ID or test token (e.g. `"pm_card_visa"`, `"pm_card_chargeDeclined"`)    |
| `card_number`    | string  | No       | Raw card number if confirming directly without a PaymentMethod ID                     |
| `confirm`        | boolean | No       | Whether to immediately confirm and process the payment (default: `false`)             |
| `capture_method` | string  | No       | `"automatic"` (default) or `"manual"`                                                 |
| `callback_url`   | string  | No       | Redirect callback URL for hosted checkout at `/checkout/intent/{id}`                  |
| `success_url`    | string  | No       | Redirect URL on successful hosted checkout                                            |
| `cancel_url`     | string  | No       | Redirect URL on canceled hosted checkout                                              |
| `webhook_url`    | string  | No       | Per-intent webhook URL notified on all events for this Payment Intent                 |
| `metadata`       | object  | No       | Arbitrary key-value pairs attached to the Payment Intent                              |

**Response (`201 Created`):**

```json
{
  "id": "pi_1a2b3c4d5e6f",
  "object": "payment_intent",
  "amount": 4999,
  "amount_capturable": 0,
  "amount_received": 0,
  "amount_refunded": 0,
  "automatic_payment_methods": {
    "enabled": true
  },
  "canceled_at": null,
  "cancellation_reason": null,
  "capture_method": "automatic",
  "client_secret": "pi_1a2b3c4d5e6f_secret_9z8y7x",
  "checkout_url": "http://localhost:8080/checkout/intent/pi_1a2b3c4d5e6f",
  "confirmation_method": "automatic",
  "created": 1700000000,
  "currency": "usd",
  "customer": "cus_default_1",
  "customer_email": "alice@example.com",
  "description": "Order #1042",
  "last_payment_error": null,
  "latest_charge": null,
  "livemode": false,
  "metadata": {
    "order_id": "1042"
  },
  "next_action": null,
  "payment_link_id": "",
  "payment_method": "pm_card_visa",
  "payment_method_types": [
    "card"
  ],
  "receipt_email": "alice@example.com",
  "setup_future_usage": null,
  "status": "requires_confirmation",
  "callback_url": "http://localhost:3000/payment/callback",
  "success_url": "http://localhost:3000/payment/success",
  "cancel_url": "http://localhost:3000/payment/cancel",
  "webhook_url": "http://localhost:3000/webhooks/payment"
}
```

**Errors:**

- `400 Bad Request` - If `amount <= 0` or `currency` is missing
- `402 Payment Required` - If `confirm: true` is passed and the payment fails due to a declined card or `always_fail: true`

---

### `GET /payments/intents`

Lists all payment intents.

**Query Parameters:**

| Parameter         | Type   | Required | Description                           |
| ----------------- | ------ | -------- | ------------------------------------- |
| `status`          | string | No       | Filter by `status`                    |
| `customer`        | string | No       | Filter by `customer` ID               |
| `payment_link_id` | string | No       | Filter by originating Payment Link ID |

**Response:**

```json
{
  "payment_intents": [
    {
      "id": "pi_1a2b3c4d5e6f",
      "object": "payment_intent",
      "amount": 4999,
      "amount_capturable": 0,
      "amount_received": 4999,
      "amount_refunded": 0,
      "automatic_payment_methods": {
        "enabled": true
      },
      "canceled_at": null,
      "cancellation_reason": null,
      "capture_method": "automatic",
      "client_secret": "pi_1a2b3c4d5e6f_secret_9z8y7x",
      "checkout_url": "http://localhost:8080/checkout/intent/pi_1a2b3c4d5e6f",
      "confirmation_method": "automatic",
      "created": 1700000000,
      "currency": "usd",
      "customer": "cus_default_1",
      "customer_email": "alice@example.com",
      "description": "Order #1042",
      "last_payment_error": null,
      "latest_charge": "ch_1a2b3c4d5e6f",
      "livemode": false,
      "metadata": {
        "order_id": "1042"
      },
      "next_action": null,
      "payment_link_id": "",
      "payment_method": "pm_card_visa",
      "payment_method_types": [
        "card"
      ],
      "receipt_email": "alice@example.com",
      "setup_future_usage": null,
      "status": "succeeded",
      "callback_url": "http://localhost:3000/payment/callback",
      "success_url": "http://localhost:3000/payment/success",
      "cancel_url": "http://localhost:3000/payment/cancel",
      "webhook_url": "http://localhost:3000/webhooks/payment"
    }
  ]
}
```

---

### `GET /payments/intents/{id}`

Retrieves a specific payment intent by ID.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The Payment Intent ID |

**Response:**

Returns the `PaymentIntent` object (same schema as `POST /payments/intents`).

**Errors:**

- `404 Not Found` - If the Payment Intent does not exist

---

### `POST /payments/intents/{id}/confirm`

Confirms and processes an existing payment intent via API.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The Payment Intent ID |

**Content-Type:** `application/json`

**Request:**

```json
{
  "payment_method": "pm_card_visa",
  "card_number": "4242424242424242",
  "exp_month": 12,
  "exp_year": 2030,
  "cvc": "123",
  "customer_email": "alice@example.com",
  "return_url": "http://localhost:3000/payment/callback"
}
```

**Request Fields:**

| Field            | Type    | Required | Description                                                                        |
| ---------------- | ------- | -------- | ---------------------------------------------------------------------------------- |
| `payment_method` | string  | No       | PaymentMethod ID or test token (e.g. `"pm_card_visa"`, `"pm_card_chargeDeclined"`) |
| `card_number`    | string  | No       | Card number (defaults to `"4242424242424242"` if no `payment_method` is set)       |
| `exp_month`      | integer | No       | Expiration month (default: `12`)                                                   |
| `exp_year`       | integer | No       | Expiration year (default: `2030`)                                                  |
| `cvc`            | string  | No       | Card CVC (default: `"123"`)                                                        |
| `customer_email` | string  | No       | Customer email address                                                             |
| `return_url`     | string  | No       | Optional return URL                                                                |

**Response:**

Returns the confirmed `PaymentIntent` object with `status: "succeeded"` (or `"requires_capture"` if `capture_method` is `"manual"`), `amount_received` set to `amount`, and `latest_charge` set to the created `Charge` ID. Emits `payment_intent.succeeded` and `charge.succeeded` webhook events.

**Errors:**

- `400 Bad Request` - If the Payment Intent is already `"succeeded"` or `"canceled"`
- `402 Payment Required` - If the card is declined or `always_fail` is enabled (emits `payment_intent.payment_failed`):
  ```json
  {
    "error": {
      "type": "card_error",
      "code": "card_declined",
      "decline_code": "insufficient_funds",
      "message": "Your card has insufficient funds.",
      "payment_intent": {
        "id": "pi_1a2b3c4d5e6f",
        "object": "payment_intent",
        "status": "requires_payment_method",
        "last_payment_error": {
          "type": "card_error",
          "code": "card_declined",
          "decline_code": "insufficient_funds",
          "message": "Your card has insufficient funds."
        }
      }
    }
  }
  ```
- `404 Not Found` - If the Payment Intent does not exist

---

### `POST /payments/intents/{id}/cancel`

Cancels a payment intent and emits a `payment_intent.canceled` webhook event.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The Payment Intent ID |

**Content-Type:** `application/json`

**Request:**

```json
{
  "cancellation_reason": "requested_by_customer"
}
```

**Response:**

Returns the updated `PaymentIntent` object with `status: "canceled"`, `canceled_at` set to the current Unix timestamp, and `cancellation_reason` populated.

**Errors:**

- `400 Bad Request` - If the Payment Intent is already `"succeeded"` or `"canceled"`
- `404 Not Found` - If the Payment Intent does not exist

---

### `POST /payments/intents/{id}/refund`

Refunds a succeeded payment intent (fully or partially) and emits a `charge.refunded` webhook event.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The Payment Intent ID |

**Content-Type:** `application/json`

**Request:**

```json
{
  "amount": 2000,
  "reason": "requested_by_customer",
  "metadata": {
    "note": "partial refund"
  }
}
```

**Request Fields:**

| Field      | Type    | Required | Description                                                                                      |
| ---------- | ------- | -------- | ------------------------------------------------------------------------------------------------ |
| `amount`   | integer | No       | Amount to refund in smallest currency unit (defaults to full remaining unrefunded amount)        |
| `reason`   | string  | No       | `"duplicate"`, `"fraudulent"`, or `"requested_by_customer"`                                      |
| `metadata` | object  | No       | Key-value pairs attached to the refund                                                           |

**Response (`201 Created`):**

```json
{
  "id": "re_1a2b3c4d5e",
  "object": "refund",
  "amount": 2000,
  "balance_transaction": "txn_1a2b3c4d5e",
  "charge": "ch_1a2b3c4d5e6f",
  "created": 1700000050,
  "currency": "usd",
  "metadata": {
    "note": "partial refund"
  },
  "payment_intent": "pi_1a2b3c4d5e6f",
  "reason": "requested_by_customer",
  "receipt_number": null,
  "status": "succeeded"
}
```

**Errors:**

- `400 Bad Request` - If the Payment Intent is not in `"succeeded"` status or `amount` exceeds the remaining refundable amount
- `404 Not Found` - If the Payment Intent does not exist

---

### `GET /payments/refunds`

Lists all refunds.

**Query Parameters:**

| Parameter        | Type   | Required | Description                 |
| ---------------- | ------ | -------- | --------------------------- |
| `payment_intent` | string | No       | Filter by Payment Intent ID |

**Response:**

```json
{
  "refunds": [
    {
      "id": "re_1a2b3c4d5e",
      "object": "refund",
      "amount": 2000,
      "balance_transaction": "txn_1a2b3c4d5e",
      "charge": "ch_1a2b3c4d5e6f",
      "created": 1700000050,
      "currency": "usd",
      "metadata": {
        "note": "partial refund"
      },
      "payment_intent": "pi_1a2b3c4d5e6f",
      "reason": "requested_by_customer",
      "receipt_number": null,
      "status": "succeeded"
    }
  ]
}
```

---

### `GET /payments/refunds/{id}`

Retrieves a specific refund by ID.

**Path Parameters:**

| Parameter | Type   | Description   |
| --------- | ------ | ------------- |
| `id`      | string | The Refund ID |

**Response:**

Returns the `Refund` object.

**Errors:**

- `404 Not Found` - If the Refund does not exist

---

## ⚡ Stripe-Compatible API

**Configuration:** These endpoints are available when `stripe_api.enabled: true` (default).

All `/v1/...` endpoints accept both `application/x-www-form-urlencoded` (supporting nested bracket notation such as `metadata[order_id]=1042`, `payment_method_data[card][number]=4242...`, `line_items[0][price_data][unit_amount]=2000`) and `application/json`.

Error responses on `/v1/...` follow the official Stripe error object format:
```json
{
  "error": {
    "type": "invalid_request_error",
    "code": "resource_missing",
    "decline_code": "",
    "message": "No such payment_intent: 'pi_unknown'",
    "param": "id"
  }
}
```

---

### `POST /v1/payment_intents`

Creates a Stripe `PaymentIntent` object. Emits a `payment_intent.created` event (and `payment_intent.succeeded` + `charge.succeeded` or `payment_intent.payment_failed` if `confirm: true`).

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter                              | Type             | Required | Description                                                                                      |
| -------------------------------------- | ---------------- | -------- | ------------------------------------------------------------------------------------------------ |
| `amount`                               | integer          | Yes      | Amount intended to be collected in smallest currency unit (e.g. `2000` for $20.00)               |
| `currency`                             | string           | Yes      | Three-letter ISO currency code, in lowercase (e.g. `"usd"`)                                      |
| `customer`                             | string           | No       | ID of the `Customer` this PaymentIntent belongs to                                               |
| `description`                          | string           | No       | Arbitrary string attached to the object                                                          |
| `receipt_email`                        | string           | No       | Email address that the receipt for the resulting payment will be sent to                         |
| `payment_method`                       | string           | No       | ID of the `PaymentMethod` (`pm_...`) or test token (e.g. `"pm_card_visa"`)                       |
| `payment_method_types`                 | array of strings | No       | List of payment method types (default: `["card"]`)                                               |
| `automatic_payment_methods[enabled]`   | boolean          | No       | Whether automatic payment methods are enabled (default: `true`)                                  |
| `confirm`                              | boolean          | No       | Set to `true` to attempt to confirm this PaymentIntent immediately (default: `false`)            |
| `capture_method`                       | string           | No       | Controls when funds will be captured: `"automatic"` (default) or `"manual"`                      |
| `confirmation_method`                  | string           | No       | `"automatic"` (default) or `"manual"`                                                            |
| `setup_future_usage`                   | string           | No       | `"on_session"` or `"off_session"`                                                                |
| `return_url`                           | string           | No       | URL to redirect the customer back to after they authenticate or cancel their payment             |
| `callback_url`                         | string           | No       | Redirect callback URL for hosted checkout UI                                                     |
| `success_url`                          | string           | No       | Success redirect URL for hosted checkout UI                                                      |
| `cancel_url`                           | string           | No       | Cancel redirect URL for hosted checkout UI                                                       |
| `webhook_url`                          | string           | No       | Per-intent webhook URL                                                                           |
| `metadata`                             | object           | No       | Set of key-value pairs attached to the object (`metadata[key]=value`)                            |

**Response (`200 OK`):**

```json
{
  "id": "pi_3M0EXAMPLE123456789",
  "object": "payment_intent",
  "amount": 2000,
  "amount_capturable": 0,
  "amount_received": 0,
  "amount_refunded": 0,
  "application": null,
  "application_fee_amount": null,
  "automatic_payment_methods": {
    "enabled": true
  },
  "canceled_at": null,
  "cancellation_reason": null,
  "capture_method": "automatic",
  "client_secret": "pi_3M0EXAMPLE123456789_secret_AbCdEf",
  "checkout_url": "http://localhost:8080/checkout/intent/pi_3M0EXAMPLE123456789",
  "confirmation_method": "automatic",
  "created": 1700000000,
  "currency": "usd",
  "customer": "cus_default_1",
  "description": "Software Subscription",
  "invoice": null,
  "last_payment_error": null,
  "latest_charge": null,
  "livemode": false,
  "metadata": {
    "order_id": "8765"
  },
  "next_action": null,
  "payment_method": null,
  "payment_method_options": {
    "card": {
      "request_three_d_secure": "automatic"
    }
  },
  "payment_method_types": [
    "card"
  ],
  "processing": null,
  "receipt_email": "alice@example.com",
  "setup_future_usage": null,
  "shipping": null,
  "statement_descriptor": null,
  "statement_descriptor_suffix": null,
  "status": "requires_payment_method",
  "transfer_data": null,
  "transfer_group": null
}
```

**Errors:**

- `400 Bad Request` - If required parameters (`amount`, `currency`) are missing or invalid
- `402 Payment Required` - If `confirm=true` and the card is declined or `always_fail` is enabled

---

### `GET /v1/payment_intents`

Returns a list of `PaymentIntent` objects.

**Query Parameters:**

| Parameter  | Type    | Required | Description                                                |
| ---------- | ------- | -------- | ---------------------------------------------------------- |
| `customer` | string  | No       | Only return PaymentIntents for the Customer ID specified   |
| `status`   | string  | No       | Filter PaymentIntents by `status`                          |
| `limit`    | integer | No       | Limit on the number of objects to be returned (default 100)|

**Response (`200 OK`):**

```json
{
  "object": "list",
  "url": "/v1/payment_intents",
  "has_more": false,
  "data": [
    {
      "id": "pi_3M0EXAMPLE123456789",
      "object": "payment_intent",
      "amount": 2000,
      "amount_capturable": 0,
      "amount_received": 2000,
      "amount_refunded": 0,
      "application": null,
      "application_fee_amount": null,
      "automatic_payment_methods": {
        "enabled": true
      },
      "canceled_at": null,
      "cancellation_reason": null,
      "capture_method": "automatic",
      "client_secret": "pi_3M0EXAMPLE123456789_secret_AbCdEf",
      "checkout_url": "http://localhost:8080/checkout/intent/pi_3M0EXAMPLE123456789",
      "confirmation_method": "automatic",
      "created": 1700000000,
      "currency": "usd",
      "customer": "cus_default_1",
      "description": "Software Subscription",
      "invoice": null,
      "last_payment_error": null,
      "latest_charge": "ch_3M0EXAMPLE123456789",
      "livemode": false,
      "metadata": {
        "order_id": "8765"
      },
      "next_action": null,
      "payment_method": "pm_card_visa",
      "payment_method_options": {
        "card": {
          "request_three_d_secure": "automatic"
        }
      },
      "payment_method_types": [
        "card"
      ],
      "processing": null,
      "receipt_email": "alice@example.com",
      "setup_future_usage": null,
      "shipping": null,
      "statement_descriptor": null,
      "statement_descriptor_suffix": null,
      "status": "succeeded",
      "transfer_data": null,
      "transfer_group": null
    }
  ]
}
```

---

### `GET /v1/payment_intents/{id}`

Retrieves the details of a `PaymentIntent` that has previously been created.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The PaymentIntent ID  |

**Response (`200 OK`):**

Returns the `PaymentIntent` object.

**Errors:**

- `404 Not Found` - If the PaymentIntent does not exist

---

### `POST /v1/payment_intents/{id}`

Updates properties on a `PaymentIntent` object without confirming it.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The PaymentIntent ID  |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter            | Type    | Required | Description                                        |
| -------------------- | ------- | -------- | -------------------------------------------------- |
| `amount`             | integer | No       | Updated amount in smallest currency unit           |
| `currency`           | string  | No       | Updated three-letter ISO currency code             |
| `customer`           | string  | No       | Updated Customer ID                                |
| `description`        | string  | No       | Updated description                                |
| `receipt_email`      | string  | No       | Updated receipt email                              |
| `payment_method`     | string  | No       | Updated PaymentMethod ID                           |
| `setup_future_usage` | string  | No       | `"on_session"` or `"off_session"`                  |
| `metadata`           | object  | No       | Key-value metadata to merge/update                 |

**Response (`200 OK`):**

Returns the updated `PaymentIntent` object.

**Errors:**

- `404 Not Found` - If the PaymentIntent does not exist

---

### `POST /v1/payment_intents/{id}/confirm`

Confirms that your customer intends to pay with current or provided payment method. Upon confirmation, the `PaymentIntent` will attempt to initiate a payment and create a `Charge`.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The PaymentIntent ID  |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter                           | Type   | Required | Description                                                                                      |
| ----------------------------------- | ------ | -------- | ------------------------------------------------------------------------------------------------ |
| `payment_method`                    | string | No       | ID of the PaymentMethod (`pm_...`) or test token (e.g. `"pm_card_visa"`, `"pm_card_chargeDeclined"`) |
| `payment_method_data[card][number]` | string | No       | Raw card number (also accepted as `card_number`)                                                 |
| `receipt_email`                     | string | No       | Email address that the receipt will be sent to                                                   |
| `return_url`                        | string | No       | URL to redirect the customer back to after confirmation                                          |
| `capture_method`                    | string | No       | Optional override for `"automatic"` or `"manual"`                                                |

**Response (`200 OK`):**

Returns the confirmed `PaymentIntent` object with `status: "succeeded"` (or `status: "requires_capture"` and `amount_capturable: amount` when `capture_method` is `"manual"`).

**Errors:**

- `400 Bad Request` - If the PaymentIntent is already `"succeeded"` or `"canceled"`
- `402 Payment Required` - If the payment fails (e.g. declined card or `always_fail: true`), returning a Stripe `card_error` object with `last_payment_error` populated on the PaymentIntent
- `404 Not Found` - If the PaymentIntent does not exist

---

### `POST /v1/payment_intents/{id}/cancel`

Cancels a `PaymentIntent` object when its status is one of `requires_payment_method`, `requires_capture`, `requires_confirmation`, `requires_action`, or `processing`.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The PaymentIntent ID  |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter             | Type   | Required | Description                                                                              |
| --------------------- | ------ | -------- | ---------------------------------------------------------------------------------------- |
| `cancellation_reason` | string | No       | Reason for canceling: `"duplicate"`, `"fraudulent"`, `"requested_by_customer"`, or `"abandoned"` |

**Response (`200 OK`):**

Returns the canceled `PaymentIntent` object with `status: "canceled"` and `canceled_at` populated.

**Errors:**

- `400 Bad Request` - If the PaymentIntent cannot be canceled (e.g. already `"succeeded"`)
- `404 Not Found` - If the PaymentIntent does not exist

---

### `POST /v1/payment_intents/{id}/capture`

Captures the funds of an existing uncaptured `PaymentIntent` when its status is `requires_capture`.

**Path Parameters:**

| Parameter | Type   | Description           |
| --------- | ------ | --------------------- |
| `id`      | string | The PaymentIntent ID  |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter           | Type    | Required | Description                                                                          |
| ------------------- | ------- | -------- | ------------------------------------------------------------------------------------ |
| `amount_to_capture` | integer | No       | The amount to capture, which must be less than or equal to the `amount_capturable`   |

**Response (`200 OK`):**

Returns the updated `PaymentIntent` object with `status: "succeeded"`, `amount_received` set to the captured amount, and `amount_capturable: 0`.

**Errors:**

- `400 Bad Request` - If the PaymentIntent is not in `"requires_capture"` status or `amount_to_capture > amount_capturable`
- `404 Not Found` - If the PaymentIntent does not exist

---

### `POST /v1/payment_links`

Creates a Stripe `PaymentLink` object.

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter                                       | Type    | Required | Description                                                                         |
| ----------------------------------------------- | ------- | -------- | ----------------------------------------------------------------------------------- |
| `line_items[0][price_data][unit_amount]`        | integer | Cond.    | Unit amount in cents (required unless `amount` is provided directly)                |
| `line_items[0][price_data][currency]`           | string  | Cond.    | Three-letter ISO currency code (default: `"usd"`)                                   |
| `line_items[0][price_data][product_data][name]` | string  | No       | Product name (used as `title` on the hosted checkout page)                          |
| `line_items[0][quantity]`                       | integer | No       | Quantity multiplier (default: `1`)                                                  |
| `amount`                                        | integer | Cond.    | Shorthand amount in cents if `line_items` is omitted                                |
| `currency`                                      | string  | Cond.    | Shorthand currency code if `line_items` is omitted                                  |
| `title`                                         | string  | No       | Shorthand title for hosted checkout page                                            |
| `description`                                   | string  | No       | Description for hosted checkout page                                                |
| `after_completion[type]`                        | string  | No       | `"redirect"` or `"hosted_confirmation"`                                             |
| `after_completion[redirect][url]`               | string  | No       | Redirect URL after purchase completion (also sets `success_url`)                    |
| `allow_promotion_codes`                         | boolean | No       | Whether promotion codes are enabled (default: `false`)                              |
| `billing_address_collection`                    | string  | No       | `"auto"` (default) or `"required"`                                                  |
| `customer_creation`                             | string  | No       | `"if_required"` (default) or `"always"`                                             |
| `submit_type`                                   | string  | No       | `"auto"` (default), `"pay"`, `"book"`, or `"donate"`                                |
| `callback_url`                                  | string  | No       | Optional fallback redirect callback URL                                             |
| `success_url`                                   | string  | No       | Optional success redirect URL                                                       |
| `cancel_url`                                    | string  | No       | Optional cancel redirect URL                                                        |
| `webhook_url`                                   | string  | No       | Optional per-link webhook URL                                                       |
| `metadata`                                      | object  | No       | Set of key-value pairs attached to the object                                       |

**Response (`200 OK`):**

```json
{
  "id": "plink_1MoC3ULkdIwHu7ixZjtGpVl2",
  "object": "payment_link",
  "active": true,
  "after_completion": {
    "type": "redirect",
    "redirect": {
      "url": "http://localhost:3000/payment/success"
    }
  },
  "allow_promotion_codes": false,
  "application_fee_amount": null,
  "application_fee_percent": null,
  "automatic_tax": {
    "enabled": false,
    "liability": null
  },
  "billing_address_collection": "auto",
  "currency": "usd",
  "custom_fields": [],
  "custom_text": {
    "after_submit": null,
    "shipping_address": null,
    "submit": null,
    "terms_of_service_acceptance": null
  },
  "customer_creation": "if_required",
  "inactive_message": null,
  "invoice_creation": {
    "enabled": false,
    "invoice_data": {
      "account_tax_ids": null,
      "custom_fields": null,
      "description": null,
      "footer": null,
      "issuer": null,
      "metadata": {},
      "rendering_options": null
    }
  },
  "livemode": false,
  "metadata": {
    "plan": "pro"
  },
  "on_behalf_of": null,
  "payment_intent_data": null,
  "payment_method_collection": "always",
  "payment_method_types": null,
  "phone_number_collection": {
    "enabled": false
  },
  "restrictions": null,
  "shipping_address_collection": null,
  "shipping_options": [],
  "submit_type": "auto",
  "subscription_data": null,
  "tax_id_collection": {
    "enabled": false
  },
  "transfer_data": null,
  "url": "http://localhost:8080/checkout/link/plink_1MoC3ULkdIwHu7ixZjtGpVl2",
  "amount": 2999,
  "title": "Pro Plan Subscription",
  "description": "Monthly subscription to Pro Plan",
  "callback_url": "http://localhost:3000/payment/callback",
  "success_url": "http://localhost:3000/payment/success",
  "cancel_url": "http://localhost:3000/payment/cancel",
  "webhook_url": "http://localhost:3000/webhooks/payment",
  "created": 1700000000
}
```

---

### `GET /v1/payment_links`

Returns a list of `PaymentLink` objects.

**Query Parameters:**

| Parameter | Type    | Required | Description                                                |
| --------- | ------- | -------- | ---------------------------------------------------------- |
| `active`  | boolean | No       | Only return payment links that are active or inactive      |
| `limit`   | integer | No       | Limit on the number of objects to be returned (default 100)|

**Response (`200 OK`):**

```json
{
  "object": "list",
  "url": "/v1/payment_links",
  "has_more": false,
  "data": [
    {
      "id": "plink_1MoC3ULkdIwHu7ixZjtGpVl2",
      "object": "payment_link",
      "active": true,
      "url": "http://localhost:8080/checkout/link/plink_1MoC3ULkdIwHu7ixZjtGpVl2",
      "currency": "usd",
      "amount": 2999,
      "livemode": false,
      "metadata": {}
    }
  ]
}
```

---

### `GET /v1/payment_links/{id}`

Retrieves a `PaymentLink` by ID.

**Path Parameters:**

| Parameter | Type   | Description         |
| --------- | ------ | ------------------- |
| `id`      | string | The PaymentLink ID  |

**Response (`200 OK`):**

Returns the `PaymentLink` object.

**Errors:**

- `404 Not Found` - If the PaymentLink does not exist

---

### `POST /v1/payment_links/{id}`

Updates a `PaymentLink` object.

**Path Parameters:**

| Parameter | Type   | Description         |
| --------- | ------ | ------------------- |
| `id`      | string | The PaymentLink ID  |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter                         | Type    | Required | Description                                    |
| --------------------------------- | ------- | -------- | ---------------------------------------------- |
| `active`                          | boolean | No       | Whether the payment link is active             |
| `after_completion[redirect][url]` | string  | No       | Updated completion redirect URL                |
| `allow_promotion_codes`           | boolean | No       | Whether promotion codes are enabled            |
| `metadata`                        | object  | No       | Updated metadata map                           |

**Response (`200 OK`):**

Returns the updated `PaymentLink` object.

**Errors:**

- `404 Not Found` - If the PaymentLink does not exist

---

### `POST /v1/checkout/sessions`

Creates a Stripe `CheckoutSession` object (`object: "checkout.session"`) and an associated `PaymentIntent`.

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter                                       | Type             | Required | Description                                                                                      |
| ----------------------------------------------- | ---------------- | -------- | ------------------------------------------------------------------------------------------------ |
| `success_url`                                   | string           | Yes      | The URL to which Stripe should send customers when payment or setup is complete                  |
| `cancel_url`                                    | string           | No       | The URL the customer will be directed to if they decide to cancel payment                        |
| `mode`                                          | string           | No       | `"payment"` (default), `"setup"`, or `"subscription"`                                            |
| `line_items[0][price_data][unit_amount]`        | integer          | Cond.    | Unit amount in cents (or pass `amount` directly)                                                 |
| `line_items[0][price_data][currency]`           | string           | Cond.    | Three-letter ISO currency code (default: `"usd"`)                                                |
| `line_items[0][price_data][product_data][name]` | string           | No       | Product name                                                                                     |
| `line_items[0][quantity]`                       | integer          | No       | Quantity (default: `1`)                                                                          |
| `amount`                                        | integer          | Cond.    | Shorthand amount in cents if `line_items` is omitted                                             |
| `currency`                                      | string           | Cond.    | Shorthand currency if `line_items` is omitted                                                    |
| `customer`                                      | string           | No       | ID of an existing `Customer`                                                                     |
| `customer_email`                                | string           | No       | Customer email address                                                                           |
| `client_reference_id`                           | string           | No       | A unique string to reference the Checkout Session                                                |
| `payment_method_types`                          | array of strings | No       | List of payment method types (default: `["card"]`)                                               |
| `submit_type`                                   | string           | No       | `"auto"` (default), `"pay"`, `"book"`, or `"donate"`                                             |
| `webhook_url`                                   | string           | No       | Optional per-session webhook URL                                                                 |
| `metadata`                                      | object           | No       | Set of key-value pairs attached to the session                                                   |

**Response (`200 OK`):**

```json
{
  "id": "cs_test_a1b2c3d4e5f6",
  "object": "checkout.session",
  "after_expiration": null,
  "allow_promotion_codes": null,
  "amount_subtotal": 2000,
  "amount_total": 2000,
  "automatic_tax": {
    "enabled": false,
    "liability": null,
    "status": null
  },
  "billing_address_collection": null,
  "cancel_url": "http://localhost:3000/cancel",
  "client_reference_id": "order_1042",
  "client_secret": null,
  "consent": null,
  "consent_collection": null,
  "created": 1700000000,
  "currency": "usd",
  "custom_fields": [],
  "custom_text": {
    "after_submit": null,
    "shipping_address": null,
    "submit": null,
    "terms_of_service_acceptance": null
  },
  "customer": "cus_default_1",
  "customer_creation": "if_required",
  "customer_details": {
    "address": null,
    "email": "alice@example.com",
    "name": null,
    "phone": null,
    "tax_exempt": "none",
    "tax_ids": []
  },
  "customer_email": "alice@example.com",
  "expires_at": 1700086400,
  "invoice": null,
  "invoice_creation": {
    "enabled": false,
    "invoice_data": {
      "account_tax_ids": null,
      "custom_fields": null,
      "description": null,
      "footer": null,
      "issuer": null,
      "metadata": {},
      "rendering_options": null
    }
  },
  "livemode": false,
  "locale": null,
  "metadata": {
    "order_id": "1042"
  },
  "mode": "payment",
  "payment_intent": "pi_3M0EXAMPLE123456789",
  "payment_link": null,
  "payment_method_collection": "always",
  "payment_method_configuration_details": null,
  "payment_method_options": {},
  "payment_method_types": [
    "card"
  ],
  "payment_status": "unpaid",
  "phone_number_collection": {
    "enabled": false
  },
  "recovered_from": null,
  "setup_intent": null,
  "shipping_address_collection": null,
  "shipping_cost": null,
  "shipping_details": null,
  "shipping_options": [],
  "status": "open",
  "submit_type": null,
  "subscription": null,
  "success_url": "http://localhost:3000/success?session_id={CHECKOUT_SESSION_ID}",
  "total_details": {
    "amount_discount": 0,
    "amount_shipping": 0,
    "amount_tax": 0
  },
  "ui_mode": "hosted",
  "url": "http://localhost:8080/checkout/session/cs_test_a1b2c3d4e5f6"
}
```

---

### `GET /v1/checkout/sessions`

Returns a list of `CheckoutSession` objects.

**Query Parameters:**

| Parameter        | Type    | Required | Description                                                |
| ---------------- | ------- | -------- | ---------------------------------------------------------- |
| `customer`       | string  | No       | Only return sessions for the Customer ID specified         |
| `payment_intent` | string  | No       | Only return sessions for the PaymentIntent ID specified    |
| `status`         | string  | No       | Filter by session `status` (`open`, `complete`, `expired`) |
| `limit`          | integer | No       | Limit on the number of objects to be returned (default 100)|

**Response (`200 OK`):**

Stripe list envelope (`object: "list"`, `url: "/v1/checkout/sessions"`, `has_more: false`, `data: [...]`).

---

### `GET /v1/checkout/sessions/{id}`

Retrieves a `CheckoutSession` object by ID.

**Path Parameters:**

| Parameter | Type   | Description             |
| --------- | ------ | ----------------------- |
| `id`      | string | The Checkout Session ID |

**Response (`200 OK`):**

Returns the `CheckoutSession` object.

**Errors:**

- `404 Not Found` - If the Checkout Session does not exist

---

### `POST /v1/checkout/sessions/{id}/expire`

Expires an open `CheckoutSession`.

**Path Parameters:**

| Parameter | Type   | Description             |
| --------- | ------ | ----------------------- |
| `id`      | string | The Checkout Session ID |

**Response (`200 OK`):**

Returns the updated `CheckoutSession` object with `status: "expired"`.

**Errors:**

- `400 Bad Request` - If the Checkout Session is not in `"open"` status
- `404 Not Found` - If the Checkout Session does not exist

---

### `POST /v1/customers`

Creates a new `Customer` object.

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter                | Type    | Required | Description                                           |
| ------------------------ | ------- | -------- | ----------------------------------------------------- |
| `id`                     | string  | No       | Optional custom ID (auto-generated with `cus_` prefix)|
| `email`                  | string  | No       | Customer's email address                              |
| `name`                   | string  | No       | Customer's full name or business name                 |
| `phone`                  | string  | No       | Customer's phone number                               |
| `description`            | string  | No       | Arbitrary string attached to the object               |
| `payment_method`         | string  | No       | ID of a PaymentMethod to attach to the customer       |
| `balance`                | integer | No       | Current balance in cents (default: `0`)               |
| `metadata`               | object  | No       | Set of key-value pairs attached to the object         |

**Response (`200 OK`):**

```json
{
  "id": "cus_NffrFeUfNV2Hib",
  "object": "customer",
  "address": null,
  "balance": 0,
  "created": 1700000000,
  "currency": null,
  "default_source": null,
  "delinquent": false,
  "description": "VIP Customer",
  "discount": null,
  "email": "alice@example.com",
  "invoice_prefix": "0759376C",
  "invoice_settings": {
    "custom_fields": null,
    "default_payment_method": null,
    "footer": null,
    "rendering_options": null
  },
  "livemode": false,
  "metadata": {
    "tier": "gold"
  },
  "name": "Alice Smith",
  "next_invoice_sequence": 1,
  "phone": "+15551234567",
  "preferred_locales": [],
  "shipping": null,
  "tax_exempt": "none",
  "test_clock": null
}
```

---

### `GET /v1/customers`

Returns a list of `Customer` objects.

**Query Parameters:**

| Parameter | Type    | Required | Description                                                 |
| --------- | ------- | -------- | ----------------------------------------------------------- |
| `email`   | string  | No       | Case-sensitive filter on the list based on customer `email` |
| `limit`   | integer | No       | Limit on the number of objects to be returned (default 100) |

**Response (`200 OK`):**

Stripe list envelope (`object: "list"`, `url: "/v1/customers"`, `has_more: false`, `data: [...]`).

---

### `GET /v1/customers/{id}`

Retrieves a `Customer` object by ID.

**Path Parameters:**

| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| `id`      | string | The Customer ID |

**Response (`200 OK`):**

Returns the `Customer` object.

**Errors:**

- `404 Not Found` - If the Customer does not exist

---

### `POST /v1/customers/{id}`

Updates the specified `Customer` by setting the values of the parameters passed.

**Path Parameters:**

| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| `id`      | string | The Customer ID |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:** `email`, `name`, `phone`, `description`, `balance`, `default_source`, `invoice_settings[default_payment_method]`, `metadata`.

**Response (`200 OK`):**

Returns the updated `Customer` object.

**Errors:**

- `404 Not Found` - If the Customer does not exist

---

### `DELETE /v1/customers/{id}`

Permanently deletes a `Customer`.

**Path Parameters:**

| Parameter | Type   | Description     |
| --------- | ------ | --------------- |
| `id`      | string | The Customer ID |

**Response (`200 OK`):**

```json
{
  "id": "cus_NffrFeUfNV2Hib",
  "object": "customer",
  "deleted": true
}
```

**Errors:**

- `404 Not Found` - If the Customer does not exist

---

### `POST /v1/payment_methods`

Creates a `PaymentMethod` object.

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter                | Type    | Required | Description                                                |
| ------------------------ | ------- | -------- | ---------------------------------------------------------- |
| `type`                   | string  | No       | The type of the PaymentMethod (default: `"card"`)          |
| `card[number]`           | string  | No       | Card number (default: `"4242424242424242"`)                |
| `card[exp_month]`        | integer | No       | Two-digit number representing the card's expiration month  |
| `card[exp_year]`         | integer | No       | Four-digit number representing the card's expiration year  |
| `card[cvc]`              | string  | No       | Card security code                                         |
| `billing_details[name]`  | string  | No       | Billing full name                                          |
| `billing_details[email]` | string  | No       | Billing email address                                      |
| `billing_details[phone]` | string  | No       | Billing phone number                                       |
| `metadata`               | object  | No       | Set of key-value pairs attached to the object              |

**Response (`200 OK`):**

```json
{
  "id": "pm_1MqM05LkdIwHu7ixlDxxO6Mc",
  "object": "payment_method",
  "allow_redisplay": "unspecified",
  "billing_details": {
    "address": {
      "city": null,
      "country": null,
      "line1": null,
      "line2": null,
      "postal_code": null,
      "state": null
    },
    "email": "alice@example.com",
    "name": "Alice Smith",
    "phone": null
  },
  "card": {
    "brand": "visa",
    "checks": {
      "address_line1_check": null,
      "address_postal_code_check": null,
      "cvc_check": "pass"
    },
    "country": "US",
    "display_brand": "visa",
    "exp_month": 12,
    "exp_year": 2030,
    "fingerprint": "mToisGZ01V71BCos",
    "funding": "credit",
    "generated_from": null,
    "last4": "4242",
    "networks": {
      "available": [
        "visa"
      ],
      "preferred": null
    },
    "three_d_secure_usage": {
      "supported": true
    },
    "wallet": null
  },
  "created": 1700000000,
  "customer": null,
  "livemode": false,
  "metadata": {},
  "type": "card"
}
```

---

### `GET /v1/payment_methods`

Returns a list of `PaymentMethod` objects.

**Query Parameters:**

| Parameter  | Type    | Required | Description                                                |
| ---------- | ------- | -------- | ---------------------------------------------------------- |
| `customer` | string  | No       | The ID of the customer whose PaymentMethods will be listed |
| `type`     | string  | No       | Filter by PaymentMethod `type` (e.g. `"card"`)             |
| `limit`    | integer | No       | Limit on the number of objects to be returned (default 100)|

**Response (`200 OK`):**

Stripe list envelope (`object: "list"`, `url: "/v1/payment_methods"`, `has_more: false`, `data: [...]`).

---

### `GET /v1/payment_methods/{id}`

Retrieves a `PaymentMethod` object by ID.

**Path Parameters:**

| Parameter | Type   | Description          |
| --------- | ------ | -------------------- |
| `id`      | string | The PaymentMethod ID |

**Response (`200 OK`):**

Returns the `PaymentMethod` object.

**Errors:**

- `404 Not Found` - If the PaymentMethod does not exist

---

### `POST /v1/payment_methods/{id}/attach`

Attaches a `PaymentMethod` object to a `Customer`.

**Path Parameters:**

| Parameter | Type   | Description          |
| --------- | ------ | -------------------- |
| `id`      | string | The PaymentMethod ID |

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter  | Type   | Required | Description                                         |
| ---------- | ------ | -------- | --------------------------------------------------- |
| `customer` | string | Yes      | The ID of the customer to which to attach the method|

**Response (`200 OK`):**

Returns the updated `PaymentMethod` object with `customer` set.

**Errors:**

- `400 Bad Request` - If `customer` is missing
- `404 Not Found` - If the PaymentMethod or Customer does not exist

---

### `POST /v1/payment_methods/{id}/detach`

Detaches a `PaymentMethod` object from a `Customer`.

**Path Parameters:**

| Parameter | Type   | Description          |
| --------- | ------ | -------------------- |
| `id`      | string | The PaymentMethod ID |

**Response (`200 OK`):**

Returns the updated `PaymentMethod` object with `customer: null`.

**Errors:**

- `404 Not Found` - If the PaymentMethod does not exist

---

### `GET /v1/charges`

Returns a list of `Charge` objects created when `PaymentIntent`s are confirmed.

**Query Parameters:**

| Parameter        | Type    | Required | Description                                                |
| ---------------- | ------- | -------- | ---------------------------------------------------------- |
| `payment_intent` | string  | No       | Only return charges associated with this PaymentIntent ID  |
| `customer`       | string  | No       | Only return charges associated with this Customer ID       |
| `limit`          | integer | No       | Limit on the number of objects to be returned (default 100)|

**Response (`200 OK`):**

```json
{
  "object": "list",
  "url": "/v1/charges",
  "has_more": false,
  "data": [
    {
      "id": "ch_3M0EXAMPLE123456789",
      "object": "charge",
      "amount": 2000,
      "amount_captured": 2000,
      "amount_refunded": 0,
      "application": null,
      "application_fee": null,
      "application_fee_amount": null,
      "balance_transaction": "txn_3M0EXAMPLE123456789",
      "billing_details": {
        "address": {
          "city": null,
          "country": null,
          "line1": null,
          "line2": null,
          "postal_code": null,
          "state": null
        },
        "email": "alice@example.com",
        "name": "Alice Smith",
        "phone": null
      },
      "calculated_statement_descriptor": "LOCALPAYMENT",
      "captured": true,
      "created": 1700000000,
      "currency": "usd",
      "customer": "cus_default_1",
      "description": "Software Subscription",
      "disputed": false,
      "failure_balance_transaction": null,
      "failure_code": null,
      "failure_message": null,
      "fraud_details": {},
      "invoice": null,
      "livemode": false,
      "metadata": {
        "order_id": "8765"
      },
      "on_behalf_of": null,
      "outcome": {
        "network_status": "approved_by_network",
        "reason": null,
        "risk_level": "normal",
        "risk_score": 10,
        "seller_message": "Payment complete.",
        "type": "authorized"
      },
      "paid": true,
      "payment_intent": "pi_3M0EXAMPLE123456789",
      "payment_method": "pm_card_visa",
      "payment_method_details": {
        "card": {
          "brand": "visa",
          "checks": {
            "address_line1_check": null,
            "address_postal_code_check": null,
            "cvc_check": "pass"
          },
          "country": "US",
          "exp_month": 12,
          "exp_year": 2030,
          "fingerprint": "mToisGZ01V71BCos",
          "funding": "credit",
          "installments": null,
          "last4": "4242",
          "mandate": null,
          "network": "visa",
          "three_d_secure": null,
          "wallet": null
        },
        "type": "card"
      },
      "receipt_email": "alice@example.com",
      "receipt_number": null,
      "receipt_url": "http://localhost:8080/receipts/ch_3M0EXAMPLE123456789",
      "refunded": false,
      "review": null,
      "shipping": null,
      "source": null,
      "source_transfer": null,
      "statement_descriptor": null,
      "statement_descriptor_suffix": null,
      "status": "succeeded",
      "transfer_data": null,
      "transfer_group": null
    }
  ]
}
```

---

### `GET /v1/charges/{id}`

Retrieves the details of a `Charge` that has previously been created.

**Path Parameters:**

| Parameter | Type   | Description   |
| --------- | ------ | ------------- |
| `id`      | string | The Charge ID |

**Response (`200 OK`):**

Returns the `Charge` object.

**Errors:**

- `404 Not Found` - If the Charge does not exist

---

### `POST /v1/refunds`

Creates a `Refund` on a previously created `Charge` or `PaymentIntent` and emits a `charge.refunded` webhook event.

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter        | Type    | Required | Description                                                                          |
| ---------------- | ------- | -------- | ------------------------------------------------------------------------------------ |
| `payment_intent` | string  | Cond.    | The identifier of the PaymentIntent to refund (required if `charge` is not provided) |
| `charge`         | string  | Cond.    | The identifier of the Charge to refund (required if `payment_intent` is not provided)|
| `amount`         | integer | No       | Amount in cents to refund (defaults to full remaining unrefunded amount)             |
| `reason`         | string  | No       | `"duplicate"`, `"fraudulent"`, or `"requested_by_customer"`                          |
| `metadata`       | object  | No       | Set of key-value pairs attached to the refund                                        |

**Response (`200 OK`):**

```json
{
  "id": "re_3M0EXAMPLE123456789",
  "object": "refund",
  "amount": 2000,
  "balance_transaction": "txn_3M0EXAMPLE987654321",
  "charge": "ch_3M0EXAMPLE123456789",
  "created": 1700000050,
  "currency": "usd",
  "destination_details": {
    "card": {
      "reference": "123456789012",
      "reference_status": "available",
      "reference_type": "acquirer_reference_number",
      "type": "refund"
    },
    "type": "card"
  },
  "metadata": {},
  "payment_intent": "pi_3M0EXAMPLE123456789",
  "reason": "requested_by_customer",
  "receipt_number": null,
  "source_transfer_reversal": null,
  "status": "succeeded",
  "transfer_reversal": null
}
```

**Errors:**

- `400 Bad Request` - If neither `payment_intent` nor `charge` is provided, if the payment is not succeeded, or if `amount` exceeds the remaining refundable amount
- `404 Not Found` - If the referenced `PaymentIntent` or `Charge` does not exist

---

### `GET /v1/refunds`

Returns a list of all `Refund` objects.

**Query Parameters:**

| Parameter        | Type    | Required | Description                                                |
| ---------------- | ------- | -------- | ---------------------------------------------------------- |
| `payment_intent` | string  | No       | Only return refunds for the PaymentIntent specified        |
| `charge`         | string  | No       | Only return refunds for the Charge specified               |
| `limit`          | integer | No       | Limit on the number of objects to be returned (default 100)|

**Response (`200 OK`):**

Stripe list envelope (`object: "list"`, `url: "/v1/refunds"`, `has_more: false`, `data: [...]`).

---

### `GET /v1/refunds/{id}`

Retrieves the details of an existing `Refund`.

**Path Parameters:**

| Parameter | Type   | Description   |
| --------- | ------ | ------------- |
| `id`      | string | The Refund ID |

**Response (`200 OK`):**

Returns the `Refund` object.

**Errors:**

- `404 Not Found` - If the Refund does not exist

---

### `GET /v1/events`

Lists `Event` objects generated by the provider (e.g. `payment_intent.created`, `payment_intent.succeeded`, `payment_intent.payment_failed`, `payment_intent.canceled`, `charge.succeeded`, `charge.refunded`, `checkout.session.completed`, `payment_link.paid`).

**Query Parameters:**

| Parameter | Type    | Required | Description                                                 |
| --------- | ------- | -------- | ----------------------------------------------------------- |
| `type`    | string  | No       | Filter events by a specific event `type`                    |
| `limit`   | integer | No       | Limit on the number of objects to be returned (default 100) |

**Response (`200 OK`):**

```json
{
  "object": "list",
  "url": "/v1/events",
  "has_more": false,
  "data": [
    {
      "id": "evt_1OcXYZ2eZvKYlo2C5abcdefg",
      "object": "event",
      "api_version": "2023-10-16",
      "created": 1700000000,
      "data": {
        "object": {
          "id": "pi_3M0EXAMPLE123456789",
          "object": "payment_intent",
          "amount": 2000,
          "currency": "usd",
          "status": "succeeded"
        }
      },
      "livemode": false,
      "pending_webhooks": 0,
      "request": {
        "id": "req_1a2b3c4d",
        "idempotency_key": null
      },
      "type": "payment_intent.succeeded"
    }
  ]
}
```

---

### `GET /v1/events/{id}`

Retrieves the details of an `Event` by ID.

**Path Parameters:**

| Parameter | Type   | Description  |
| --------- | ------ | ------------ |
| `id`      | string | The Event ID |

**Response (`200 OK`):**

Returns the `Event` object.

**Errors:**

- `404 Not Found` - If the Event does not exist

---

### `POST /v1/webhook_endpoints`

Configures a new `WebhookEndpoint` destination. Whenever an event matching `enabled_events` occurs, the `Event` object is `POST`ed as JSON to `url` with headers:
- `Stripe-Signature: t={unix_timestamp},v1={hmac_sha256_hex}` (HMAC-SHA256 over `"{unix_timestamp}.{raw_body}"` using `secret`)
- `X-Payment-Signature: sha256={hmac_sha256_hex}` (HMAC-SHA256 over `"{raw_body}"` using `secret`)
- `X-Webhook-Event: {event_type}`
- `X-Webhook-Id: {event_id}`

**Content-Type:** `application/x-www-form-urlencoded` or `application/json`

**Parameters:**

| Parameter        | Type             | Required | Description                                                                                      |
| ---------------- | ---------------- | -------- | ------------------------------------------------------------------------------------------------ |
| `url`            | string           | Yes      | The URL of the webhook endpoint                                                                  |
| `enabled_events` | array of strings | No       | The list of events to enable for this endpoint (e.g. `["*"]` or `["payment_intent.succeeded"]`)  |
| `description`    | string           | No       | An optional description of what the webhook is used for                                          |
| `secret`         | string           | No       | Optional custom signing secret (defaults to the server's `webhook_secret`)                       |
| `metadata`       | object           | No       | Set of key-value pairs attached to the object                                                    |

**Response (`200 OK`):**

```json
{
  "id": "we_1Mr5jULkdIwHu7ix1ibLTM0x",
  "object": "webhook_endpoint",
  "api_version": null,
  "application": null,
  "created": 1700000000,
  "description": null,
  "enabled_events": [
    "*"
  ],
  "livemode": false,
  "metadata": {},
  "secret": "whsec_test_secret",
  "status": "enabled",
  "url": "http://localhost:3000/webhooks/stripe"
}
```

**Errors:**

- `400 Bad Request` - If `url` is missing

---

### `GET /v1/webhook_endpoints`

Returns a list of `WebhookEndpoint` objects.

**Response (`200 OK`):**

Stripe list envelope (`object: "list"`, `url: "/v1/webhook_endpoints"`, `has_more: false`, `data: [...]`).

---

### `GET /v1/webhook_endpoints/{id}`

Retrieves the `WebhookEndpoint` with the given ID.

**Path Parameters:**

| Parameter | Type   | Description            |
| --------- | ------ | ---------------------- |
| `id`      | string | The WebhookEndpoint ID |

**Response (`200 OK`):**

Returns the `WebhookEndpoint` object.

**Errors:**

- `404 Not Found` - If the WebhookEndpoint does not exist

---

### `DELETE /v1/webhook_endpoints/{id}`

Deletes the `WebhookEndpoint` with the given ID.

**Path Parameters:**

| Parameter | Type   | Description            |
| --------- | ------ | ---------------------- |
| `id`      | string | The WebhookEndpoint ID |

**Response (`200 OK`):**

```json
{
  "id": "we_1Mr5jULkdIwHu7ix1ibLTM0x",
  "object": "webhook_endpoint",
  "deleted": true
}
```

**Errors:**

- `404 Not Found` - If the WebhookEndpoint does not exist

---

## 🧪 Testing & Spying API

### `GET /_testing/config`

Returns the active runtime testing configuration.

**Response:**

```json
{
  "delay_ms": 0,
  "webhook_delay_ms": 0,
  "always_fail": false,
  "always_fail_code": "processing_error",
  "always_fail_message": "Payment processing failed due to test configuration",
  "declined_cards": {
    "4000000000000002": {
      "code": "card_declined",
      "decline_code": "generic_decline",
      "message": "Your card was declined."
    },
    "4000000000009995": {
      "code": "card_declined",
      "decline_code": "insufficient_funds",
      "message": "Your card has insufficient funds."
    },
    "4000000000000069": {
      "code": "expired_card",
      "decline_code": "expired_card",
      "message": "Your card has expired."
    },
    "4000000000000119": {
      "code": "processing_error",
      "decline_code": "processing_error",
      "message": "An error occurred while processing your card."
    }
  }
}
```

---

### `PUT /_testing/config`

Updates the runtime testing configuration. All fields are optional; only provided fields are updated.

**Content-Type:** `application/json`

**Request:**

```json
{
  "delay_ms": 100,
  "webhook_delay_ms": 0,
  "always_fail": true,
  "always_fail_code": "card_declined",
  "always_fail_message": "Simulated payment failure",
  "declined_cards": {
    "4111111111111119": {
      "code": "card_declined",
      "decline_code": "fraudulent",
      "message": "Card blocked for suspected fraud."
    }
  }
}
```

**Response:**

Returns the updated runtime testing configuration object (same schema as `GET /_testing/config`).

**Errors:**

- `400 Bad Request` - If request body is invalid JSON

---

### `GET /_testing/requests`

Returns all recorded HTTP requests made to `/payments/...`, `/v1/...`, and `/checkout/...` in chronological order.

**Query Parameters:**

| Parameter | Type    | Required | Description                                                      |
| --------- | ------- | -------- | ---------------------------------------------------------------- |
| `method`  | string  | No       | Filter by HTTP method (e.g. `"POST"`, `"GET"`)                   |
| `path`    | string  | No       | Filter by exact path or path prefix (e.g. `"/v1/payment_intents"`) |
| `limit`   | integer | No       | Maximum number of requests to return                             |

**Response:**

```json
{
  "requests": [
    {
      "id": "req_1a2b3c4d",
      "method": "POST",
      "path": "/v1/payment_intents",
      "query": "",
      "headers": {
        "Content-Type": "application/x-www-form-urlencoded"
      },
      "body": "amount=2000&currency=usd",
      "status_code": 200,
      "timestamp": "2025-01-01T12:00:00Z"
    }
  ]
}
```

---

### `GET /_testing/payments`

Returns all in-memory `payment_intents`, `charges`, `refunds`, and `checkout_sessions`.

**Query Parameters:**

| Parameter         | Type   | Required | Description                           |
| ----------------- | ------ | -------- | ------------------------------------- |
| `status`          | string | No       | Filter `payment_intents` by status    |
| `customer`        | string | No       | Filter `payment_intents` by customer  |
| `payment_link_id` | string | No       | Filter `payment_intents` by link ID   |

**Response:**

```json
{
  "payment_intents": [],
  "charges": [],
  "refunds": [],
  "checkout_sessions": []
}
```

---

### `GET /_testing/webhooks`

Returns all recorded webhook delivery attempts and their HTTP response statuses/bodies in chronological order.

**Query Parameters:**

| Parameter           | Type   | Required | Description                                             |
| ------------------- | ------ | -------- | ------------------------------------------------------- |
| `event_type`        | string | No       | Filter by event type (e.g. `"payment_intent.succeeded"`)|
| `url`               | string | No       | Filter by destination webhook URL                       |
| `payment_intent_id` | string | No       | Filter by associated PaymentIntent ID                   |
| `status`            | string | No       | Filter by delivery status (`"succeeded"` or `"failed"`) |

**Response:**

```json
{
  "deliveries": [
    {
      "id": "whd_1a2b3c4d",
      "event_id": "evt_1OcXYZ2eZvKYlo2C5abcdefg",
      "event_type": "payment_intent.succeeded",
      "payment_intent_id": "pi_3M0EXAMPLE123456789",
      "url": "http://localhost:3000/webhooks/stripe",
      "status": "succeeded",
      "response_status_code": 200,
      "response_body": "{\"received\":true}",
      "error": "",
      "request_headers": {
        "Content-Type": "application/json",
        "Stripe-Signature": "t=1700000000,v1=abcdef123456...",
        "X-Payment-Signature": "sha256=987654fedcba...",
        "X-Webhook-Event": "payment_intent.succeeded",
        "X-Webhook-Id": "evt_1OcXYZ2eZvKYlo2C5abcdefg"
      },
      "payload": {
        "id": "evt_1OcXYZ2eZvKYlo2C5abcdefg",
        "object": "event",
        "api_version": "2023-10-16",
        "created": 1700000000,
        "data": {
          "object": {
            "id": "pi_3M0EXAMPLE123456789",
            "object": "payment_intent",
            "amount": 2000,
            "currency": "usd",
            "status": "succeeded"
          }
        },
        "livemode": false,
        "pending_webhooks": 0,
        "request": {
          "id": "req_1a2b3c4d",
          "idempotency_key": null
        },
        "type": "payment_intent.succeeded"
      },
      "timestamp": "2025-01-01T12:00:00Z"
    }
  ]
}
```

---

### `POST /_testing/webhooks/trigger`

Manually constructs, signs, and delivers a synthetic webhook event.

**Content-Type:** `application/json`

**Request:**

```json
{
  "type": "payment_intent.succeeded",
  "url": "http://localhost:3000/webhooks/stripe",
  "secret": "whsec_test_secret",
  "payment_intent_id": "pi_3M0EXAMPLE123456789",
  "data": {
    "id": "pi_3M0EXAMPLE123456789",
    "object": "payment_intent",
    "amount": 2000,
    "currency": "usd",
    "status": "succeeded"
  }
}
```

**Request Fields:**

| Field               | Type   | Required | Description                                                                                 |
| ------------------- | ------ | -------- | ------------------------------------------------------------------------------------------- |
| `type`              | string | Yes      | Event type to trigger (e.g. `"payment_intent.succeeded"`)                                   |
| `url`               | string | No       | Target webhook URL (if omitted, delivers to all registered webhooks matching `type`)        |
| `secret`            | string | No       | Signing secret override (defaults to the endpoint's secret or global `webhook_secret`)      |
| `payment_intent_id` | string | No       | If provided and `data` is omitted, populates `data.object` with the matching PaymentIntent  |
| `data`              | object | No       | Custom object to place in `event.data.object`                                               |

**Response:**

```json
{
  "event": {
    "id": "evt_1OcXYZ2eZvKYlo2C5abcdefg",
    "object": "event",
    "api_version": "2023-10-16",
    "created": 1700000000,
    "data": {
      "object": {
        "id": "pi_3M0EXAMPLE123456789",
        "object": "payment_intent",
        "amount": 2000,
        "currency": "usd",
        "status": "succeeded"
      }
    },
    "livemode": false,
    "pending_webhooks": 0,
    "request": {
      "id": null,
      "idempotency_key": null
    },
    "type": "payment_intent.succeeded"
  },
  "deliveries": [
    {
      "id": "whd_1a2b3c4d",
      "event_id": "evt_1OcXYZ2eZvKYlo2C5abcdefg",
      "event_type": "payment_intent.succeeded",
      "payment_intent_id": "pi_3M0EXAMPLE123456789",
      "url": "http://localhost:3000/webhooks/stripe",
      "status": "succeeded",
      "response_status_code": 200
    }
  ]
}
```

**Errors:**

- `400 Bad Request` - If `type` is missing or request body is invalid

---

### `POST /_testing/reset`

Resets all in-memory state, dynamically created resources, recorded requests, and webhook deliveries back to the initial configuration loaded from `config.yaml`.

**Response:**

```json
{
  "status": "OK",
  "message": "State reset to initial configuration"
}
```

---

## 📋 Response Codes Summary

| Code | Description                                                                 |
| ---- | --------------------------------------------------------------------------- |
| 200  | Success - Request completed successfully                                    |
| 201  | Created - Resource created successfully (Native API)                        |
| 204  | No Content - Success with no response body                                  |
| 302  | Found - Redirect (used in Hosted Checkout UI callback flow)                 |
| 400  | Bad Request - Invalid request format or parameters                          |
| 401  | Unauthorized - Authentication failed or API key invalid                     |
| 402  | Payment Required - Card declined or payment processing failed               |
| 404  | Not Found - Resource does not exist                                         |
| 500  | Internal Server Error - Server encountered an error                         |

---

## 🔒 Authentication

By default, endpoints accept any API key or no authentication header for frictionless local testing. When `api_keys` is configured in `config.yaml`, `/payments/...` and `/v1/...` endpoints require authentication via Bearer token or HTTP Basic Auth:

```http
Authorization: Bearer {api_key}
```

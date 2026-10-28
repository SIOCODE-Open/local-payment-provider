package main

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"
)

const checkoutPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} – SIOCODE Pay</title>
    <style>
        :root {
            --sioblue: #002266;
            --sioblue-hover: #001a4d;
            --sioblue-accent: #f0f4ff;
        }
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background-color: #f8fafc;
            background-image: radial-gradient(#cbd5e1 1px, transparent 1px);
            background-size: 20px 20px;
            color: #0f172a;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
        }
        .hero-header {
            background-color: #002266;
            background-image: radial-gradient(rgba(255, 255, 255, 0.15) 1px, transparent 1px);
            background-size: 20px 20px;
            color: #ffffff;
            padding: 20px 24px;
            border-bottom: 1px solid rgba(255, 255, 255, 0.1);
        }
        .hero-inner {
            max-width: 480px;
            margin: 0 auto;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .brand {
            display: flex;
            align-items: center;
            gap: 10px;
            font-weight: 700;
            font-size: 18px;
            letter-spacing: -0.02em;
        }
        .brand-badge {
            background: #ffffff;
            color: #002266;
            font-size: 11px;
            font-weight: 700;
            padding: 3px 8px;
            border-radius: 9999px;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        .main-wrapper {
            flex: 1;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 32px 16px;
        }
        .card {
            width: 100%;
            max-width: 480px;
            background: #ffffff;
            border: 1px solid #e2e8f0;
            border-radius: 12px;
            box-shadow: 0 10px 25px -5px rgba(0, 34, 102, 0.08), 0 8px 10px -6px rgba(0, 34, 102, 0.04);
            overflow: hidden;
        }
        .summary-banner {
            background: #f0f4ff;
            border-bottom: 1px solid #e2e8f0;
            padding: 24px;
        }
        .summary-label {
            font-size: 12px;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            color: #002266;
            margin-bottom: 6px;
        }
        .summary-title {
            font-size: 20px;
            font-weight: 700;
            color: #0f172a;
            margin-bottom: 4px;
        }
        .summary-description {
            font-size: 14px;
            color: #475569;
            margin-bottom: 16px;
        }
        .summary-amount {
            font-size: 32px;
            font-weight: 800;
            color: #002266;
            letter-spacing: -0.03em;
        }
        .summary-currency {
            font-size: 14px;
            font-weight: 600;
            color: #64748b;
            text-transform: uppercase;
            margin-left: 4px;
        }
        .card-body {
            padding: 24px;
        }
        .error {
            background-color: #fef2f2;
            border: 1px solid #fecaca;
            color: #dc2626;
            padding: 12px 16px;
            border-radius: 8px;
            font-size: 14px;
            font-weight: 500;
            margin-bottom: 20px;
        }
        .form-group {
            margin-bottom: 16px;
        }
        .form-row {
            display: grid;
            grid-template-columns: 1fr 1fr 1fr;
            gap: 12px;
            margin-bottom: 16px;
        }
        label {
            display: block;
            font-size: 13px;
            font-weight: 600;
            color: #334155;
            margin-bottom: 6px;
        }
        input[type="text"],
        input[type="email"] {
            width: 100%;
            padding: 10px 12px;
            font-size: 14px;
            color: #0f172a;
            background-color: #ffffff;
            border: 1px solid #cbd5e1;
            border-radius: 8px;
            outline: none;
            transition: border-color 0.15s ease, box-shadow 0.15s ease;
        }
        input[type="text"]:focus,
        input[type="email"]:focus {
            border-color: #002266;
            box-shadow: 0 0 0 3px rgba(0, 34, 102, 0.15);
        }
        .button-group {
            display: flex;
            flex-direction: column;
            gap: 10px;
            margin-top: 24px;
        }
        .btn-primary {
            width: 100%;
            padding: 12px 20px;
            background-color: #002266;
            color: #ffffff;
            font-size: 15px;
            font-weight: 600;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            transition: background-color 0.15s ease;
        }
        .btn-primary:hover {
            background-color: #001a4d;
        }
        .btn-secondary {
            width: 100%;
            padding: 10px 20px;
            background-color: #f1f5f9;
            color: #0f172a;
            font-size: 14px;
            font-weight: 600;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            cursor: pointer;
            transition: background-color 0.15s ease;
        }
        .btn-secondary:hover {
            background-color: #e2e8f0;
        }
        .footer-note {
            text-align: center;
            font-size: 12px;
            color: #64748b;
            margin-top: 20px;
            padding-top: 16px;
            border-top: 1px solid #f1f5f9;
        }
    </style>
</head>
<body>
    <header class="hero-header">
        <div class="hero-inner">
            <div class="brand">
                <span>SIOCODE Pay</span>
            </div>
            <span class="brand-badge">Local Payment Provider</span>
        </div>
    </header>
    <main class="main-wrapper">
        <div class="card">
            <div class="summary-banner">
                <div class="summary-label">Hosted Checkout</div>
                <div class="summary-title">{{.Title}}</div>
                {{if .Description}}
                <div class="summary-description">{{.Description}}</div>
                {{end}}
                <div>
                    <span class="summary-amount">{{.FormattedAmount}}</span>
                    <span class="summary-currency">{{.Currency}}</span>
                </div>
            </div>
            <div class="card-body">
                {{if .Error}}
                <div class="error">{{.Error}}</div>
                {{end}}
                <form method="POST" action="{{.FormAction}}">
                    <input type="hidden" name="callback_url" value="{{.CallbackUrl}}">
                    <input type="hidden" name="success_url" value="{{.SuccessUrl}}">
                    <input type="hidden" name="cancel_url" value="{{.CancelUrl}}">
                    <input type="hidden" name="state" value="{{.State}}">
                    <input type="hidden" name="redirect_on_error" value="{{.RedirectOnError}}">
                    <input type="hidden" name="payment_method" value="{{.PaymentMethod}}">

                    <div class="form-group">
                        <label for="customer_email">Email Address</label>
                        <input type="email" id="customer_email" name="customer_email" value="{{.CustomerEmail}}" placeholder="alice@example.com">
                    </div>

                    <div class="form-group">
                        <label for="cardholder_name">Cardholder Name</label>
                        <input type="text" id="cardholder_name" name="cardholder_name" value="{{.CardholderName}}" placeholder="Alice Smith">
                    </div>

                    <div class="form-group">
                        <label for="card_number">Card Number</label>
                        <input type="text" id="card_number" name="card_number" value="{{.CardNumber}}" placeholder="4242 4242 4242 4242">
                    </div>

                    <div class="form-row">
                        <div>
                            <label for="exp_month">Exp Month</label>
                            <input type="text" id="exp_month" name="exp_month" value="{{.ExpMonth}}" placeholder="12">
                        </div>
                        <div>
                            <label for="exp_year">Exp Year</label>
                            <input type="text" id="exp_year" name="exp_year" value="{{.ExpYear}}" placeholder="2030">
                        </div>
                        <div>
                            <label for="cvc">CVC</label>
                            <input type="text" id="cvc" name="cvc" value="{{.Cvc}}" placeholder="123">
                        </div>
                    </div>

                    <div class="button-group">
                        <button type="submit" name="action" value="pay" class="btn-primary">Pay {{.FormattedAmount}} {{.UpperCurrency}}</button>
                        <button type="submit" name="action" value="cancel" class="btn-secondary">Cancel Payment</button>
                    </div>
                </form>
                <div class="footer-note">
                    Secured by <strong>SIOCODE</strong> Local Payment Provider ({{.ResourceID}})
                </div>
            </div>
        </div>
    </main>
</body>
</html>`

const checkoutResultTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Heading}} – SIOCODE Pay</title>
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background-color: #f8fafc;
            background-image: radial-gradient(#cbd5e1 1px, transparent 1px);
            background-size: 20px 20px;
            color: #0f172a;
            min-height: 100vh;
            display: flex;
            flex-direction: column;
        }
        .hero-header {
            background-color: #002266;
            background-image: radial-gradient(rgba(255, 255, 255, 0.15) 1px, transparent 1px);
            background-size: 20px 20px;
            color: #ffffff;
            padding: 20px 24px;
        }
        .hero-inner {
            max-width: 480px;
            margin: 0 auto;
            display: flex;
            align-items: center;
            justify-content: space-between;
        }
        .brand {
            font-weight: 700;
            font-size: 18px;
        }
        .brand-badge {
            background: #ffffff;
            color: #002266;
            font-size: 11px;
            font-weight: 700;
            padding: 3px 8px;
            border-radius: 9999px;
            text-transform: uppercase;
        }
        .main-wrapper {
            flex: 1;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 32px 16px;
        }
        .card {
            width: 100%;
            max-width: 480px;
            background: #ffffff;
            border: 1px solid #e2e8f0;
            border-radius: 12px;
            box-shadow: 0 10px 25px -5px rgba(0, 34, 102, 0.08), 0 8px 10px -6px rgba(0, 34, 102, 0.04);
            padding: 32px 24px;
            text-align: center;
        }
        .status-badge {
            display: inline-block;
            padding: 6px 14px;
            border-radius: 9999px;
            font-size: 13px;
            font-weight: 700;
            text-transform: uppercase;
            margin-bottom: 16px;
            background: #f0f4ff;
            color: #002266;
        }
        h1 {
            font-size: 24px;
            font-weight: 700;
            color: #0f172a;
            margin-bottom: 8px;
        }
        p {
            font-size: 15px;
            color: #475569;
            margin-bottom: 20px;
        }
        .details {
            background: #f8fafc;
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            padding: 16px;
            text-align: left;
            font-size: 13px;
            color: #334155;
        }
        .details div {
            margin-bottom: 6px;
        }
        .details div:last-child {
            margin-bottom: 0;
        }
    </style>
</head>
<body>
    <header class="hero-header">
        <div class="hero-inner">
            <div class="brand">SIOCODE Pay</div>
            <span class="brand-badge">Local Payment Provider</span>
        </div>
    </header>
    <main class="main-wrapper">
        <div class="card">
            <div class="status-badge">{{.Status}}</div>
            <h1>{{.Heading}}</h1>
            <p>{{.Message}}</p>
            <div class="details">
                <div><strong>Payment Intent ID:</strong> {{.PaymentIntentID}}</div>
                <div><strong>Amount:</strong> {{.FormattedAmount}} {{.UpperCurrency}}</div>
                <div><strong>Status:</strong> {{.Status}}</div>
            </div>
        </div>
    </main>
</body>
</html>`

type checkoutTemplateData struct {
	ResourceID      string
	Title           string
	Description     string
	Amount          int64
	FormattedAmount string
	Currency        string
	UpperCurrency   string
	FormAction      string
	CallbackUrl     string
	SuccessUrl      string
	CancelUrl       string
	State           string
	RedirectOnError string
	CustomerEmail   string
	CardholderName  string
	CardNumber      string
	ExpMonth        string
	ExpYear         string
	Cvc             string
	PaymentMethod   string
	Error           string
}

type checkoutResultData struct {
	Heading         string
	Status          string
	Message         string
	PaymentIntentID string
	FormattedAmount string
	UpperCurrency   string
}

func formatCurrencyAmount(amount int64, currency string) string {
	cur := strings.ToLower(currency)
	symbol := "$"
	switch cur {
	case "eur":
		symbol = "€"
	case "gbp":
		symbol = "£"
	case "huf":
		return fmt.Sprintf("%d Ft", amount)
	case "jpy":
		return fmt.Sprintf("¥%d", amount)
	case "usd", "cad", "aud":
		symbol = "$"
	default:
		symbol = ""
	}
	whole := amount / 100
	cents := amount % 100
	if cents < 0 {
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", symbol, whole, cents)
}

func renderCheckoutForm(w http.ResponseWriter, status int, data checkoutTemplateData) {
	if data.CardNumber == "" {
		data.CardNumber = "4242424242424242"
	}
	if data.ExpMonth == "" {
		data.ExpMonth = "12"
	}
	if data.ExpYear == "" {
		data.ExpYear = "2030"
	}
	if data.Cvc == "" {
		data.Cvc = "123"
	}
	data.FormattedAmount = formatCurrencyAmount(data.Amount, data.Currency)
	data.UpperCurrency = strings.ToUpper(data.Currency)

	tmpl, err := template.New("checkout").Parse(checkoutPageTemplate)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = tmpl.Execute(w, data)
}

func renderCheckoutResult(w http.ResponseWriter, status int, data checkoutResultData) {
	tmpl, err := template.New("result").Parse(checkoutResultTemplate)
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_ = tmpl.Execute(w, data)
}

func parseExpMonthYear(expMonthStr, expYearStr string) (int, int) {
	expMonth := 12
	expYear := 2030
	if m, err := strconv.Atoi(expMonthStr); err == nil && m > 0 {
		expMonth = m
	}
	if y, err := strconv.Atoi(expYearStr); err == nil && y > 0 {
		expYear = y
	}
	return expMonth, expYear
}

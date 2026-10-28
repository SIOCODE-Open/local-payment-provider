package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type contextKey string

const requestIdContextKey contextKey = "request_id"

func getRequestId(r *http.Request) string {
	if r == nil {
		return ""
	}
	if v, ok := r.Context().Value(requestIdContextKey).(string); ok {
		return v
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeStripeError(w http.ResponseWriter, status int, errType, code, declineCode, message, param string) {
	errObj := map[string]interface{}{
		"type":         errType,
		"code":         code,
		"decline_code": declineCode,
		"message":      message,
	}
	if param != "" {
		errObj["param"] = param
	}
	writeJSON(w, status, map[string]interface{}{
		"error": errObj,
	})
}

func writePaymentErrorWithIntent(w http.ResponseWriter, status int, payErr *PaymentError, pi *PaymentIntent) {
	errObj := map[string]interface{}{
		"type":           payErr.Type,
		"code":           payErr.Code,
		"decline_code":   payErr.DeclineCode,
		"message":        payErr.Message,
		"payment_intent": pi,
	}
	writeJSON(w, status, map[string]interface{}{
		"error": errObj,
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func checkAuth(r *http.Request) bool {
	if len(AppConfig.ApiKeys) == 0 {
		return true
	}
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return false
	}

	var token string
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	} else if user, pass, ok := r.BasicAuth(); ok {
		if user != "" {
			token = user
		} else {
			token = pass
		}
	} else {
		token = strings.TrimSpace(authHeader)
	}

	for _, k := range AppConfig.ApiKeys {
		if k == token {
			return true
		}
	}
	return false
}

func accessLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Read body so we can record it and still allow handlers to read it
		var bodyBytes []byte
		if r.Body != nil {
			bodyBytes, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		reqId := generateID("req")
		ctx := context.WithValue(r.Context(), requestIdContextKey, reqId)
		r = r.WithContext(ctx)

		isPayments := strings.HasPrefix(r.URL.Path, "/payments")
		isStripe := strings.HasPrefix(r.URL.Path, "/v1")
		isCheckout := strings.HasPrefix(r.URL.Path, "/checkout")

		// Apply artificial delay if configured for API/checkout requests
		if isPayments || isStripe || isCheckout {
			AppContext.mu.RLock()
			delayMs := AppContext.Testing.DelayMs
			AppContext.mu.RUnlock()
			if delayMs > 0 {
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
		}

		rec := &statusRecorder{ResponseWriter: w, status: 200}

		// Check authentication for /payments/... and /v1/...
		if (isPayments || isStripe) && !checkAuth(r) {
			writeStripeError(rec, http.StatusUnauthorized, "invalid_request_error", "api_key_invalid", "", "Invalid API Key provided", "")
		} else {
			next.ServeHTTP(rec, r)
		}

		if isPayments || isStripe || isCheckout {
			headers := make(map[string]string)
			for k, vals := range r.Header {
				if len(vals) > 0 {
					headers[k] = strings.Join(vals, ", ")
				}
			}
			recorded := RecordedRequest{
				Id:         reqId,
				Method:     r.Method,
				Path:       r.URL.Path,
				Query:      r.URL.RawQuery,
				Headers:    headers,
				Body:       string(bodyBytes),
				StatusCode: rec.status,
				Timestamp:  start.UTC().Format(time.RFC3339),
			}
			AppContext.mu.Lock()
			AppContext.RecordedRequests = append(AppContext.RecordedRequests, recorded)
			AppContext.mu.Unlock()
		}

		log.Printf("%s %s %d %s", r.Method, r.URL.Path, rec.status, time.Since(start))
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowedOrigins := AppConfig.AllowedOrigins

		if allowedOrigins == "*" {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && allowedOrigins != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
			w.Header().Set("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Stripe-Signature, X-Payment-Signature, Stripe-Version, Idempotency-Key")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// RequestParams provides unified access to both JSON and form-urlencoded (including Stripe nested bracket notation) payloads.
type RequestParams struct {
	RawJSON map[string]interface{}
	Form    url.Values
	IsJSON  bool
}

func parseRequestParams(r *http.Request) (*RequestParams, error) {
	rp := &RequestParams{
		RawJSON: make(map[string]interface{}),
		Form:    make(url.Values),
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return rp, err
	}
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	trimmed := bytes.TrimSpace(bodyBytes)
	if len(trimmed) == 0 {
		_ = r.ParseForm()
		rp.Form = r.Form
		return rp, nil
	}

	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if strings.Contains(ct, "application/json") || (ct == "" && (trimmed[0] == '{' || trimmed[0] == '[')) {
		if err := json.Unmarshal(trimmed, &rp.RawJSON); err != nil {
			return rp, err
		}
		rp.IsJSON = true
		return rp, nil
	}

	vals, err := url.ParseQuery(string(bodyBytes))
	if err != nil {
		// Fallback: try JSON if form parsing fails
		if jsonErr := json.Unmarshal(trimmed, &rp.RawJSON); jsonErr == nil {
			rp.IsJSON = true
			return rp, nil
		}
		return rp, err
	}
	// Merge with query values if needed, though form body takes precedence
	rp.Form = vals
	return rp, nil
}

func (rp *RequestParams) Has(key string) bool {
	if rp.IsJSON {
		_, ok := rp.getNestedJSON(key)
		return ok
	}
	if _, ok := rp.Form[key]; ok {
		return true
	}
	prefix := key + "["
	for k := range rp.Form {
		if strings.HasPrefix(k, prefix) {
			return true
		}
	}
	return false
}

func (rp *RequestParams) getNestedJSON(key string) (interface{}, bool) {
	if val, ok := rp.RawJSON[key]; ok {
		return val, true
	}
	// Support bracket paths like "automatic_payment_methods[enabled]" or "line_items[0][price_data][unit_amount]"
	if !strings.Contains(key, "[") {
		return nil, false
	}
	parts := parseBracketPath(key)
	var curr interface{} = rp.RawJSON
	for _, p := range parts {
		switch node := curr.(type) {
		case map[string]interface{}:
			v, exists := node[p]
			if !exists {
				return nil, false
			}
			curr = v
		case []interface{}:
			idx, err := strconv.Atoi(p)
			if err != nil || idx < 0 || idx >= len(node) {
				return nil, false
			}
			curr = node[idx]
		default:
			return nil, false
		}
	}
	return curr, true
}

func parseBracketPath(key string) []string {
	var parts []string
	rawParts := strings.Split(key, "[")
	for i, p := range rawParts {
		if i == 0 {
			parts = append(parts, p)
		} else {
			parts = append(parts, strings.TrimSuffix(p, "]"))
		}
	}
	return parts
}

func (rp *RequestParams) GetString(key string) string {
	if rp.IsJSON {
		val, ok := rp.getNestedJSON(key)
		if !ok || val == nil {
			return ""
		}
		switch v := val.(type) {
		case string:
			return v
		case float64:
			return strconv.FormatInt(int64(v), 10)
		case bool:
			return strconv.FormatBool(v)
		default:
			return ""
		}
	}
	return rp.Form.Get(key)
}

func (rp *RequestParams) GetInt64(key string) (int64, bool) {
	if rp.IsJSON {
		val, ok := rp.getNestedJSON(key)
		if !ok || val == nil {
			return 0, false
		}
		switch v := val.(type) {
		case float64:
			return int64(v), true
		case int64:
			return v, true
		case int:
			return int64(v), true
		case string:
			i, err := strconv.ParseInt(v, 10, 64)
			return i, err == nil
		default:
			return 0, false
		}
	}
	s := rp.Form.Get(key)
	if s == "" {
		return 0, false
	}
	i, err := strconv.ParseInt(s, 10, 64)
	return i, err == nil
}

func (rp *RequestParams) GetBool(key string) (bool, bool) {
	if rp.IsJSON {
		val, ok := rp.getNestedJSON(key)
		if !ok || val == nil {
			return false, false
		}
		switch v := val.(type) {
		case bool:
			return v, true
		case string:
			b, err := strconv.ParseBool(v)
			return b, err == nil
		default:
			return false, false
		}
	}
	s := rp.Form.Get(key)
	if s == "" {
		return false, false
	}
	b, err := strconv.ParseBool(s)
	return b, err == nil
}

func (rp *RequestParams) GetStringSlice(key string) []string {
	if rp.IsJSON {
		val, ok := rp.getNestedJSON(key)
		if !ok || val == nil {
			return nil
		}
		switch v := val.(type) {
		case []interface{}:
			var res []string
			for _, item := range v {
				if s, ok := item.(string); ok {
					res = append(res, s)
				}
			}
			return res
		case []string:
			return v
		case string:
			return []string{v}
		}
		return nil
	}
	if vals, ok := rp.Form[key]; ok && len(vals) > 0 {
		return vals
	}
	if vals, ok := rp.Form[key+"[]"]; ok && len(vals) > 0 {
		return vals
	}
	// Check indexed keys like payment_method_types[0], payment_method_types[1]
	var res []string
	for i := 0; i < 20; i++ {
		indexedKey := key + "[" + strconv.Itoa(i) + "]"
		if v := rp.Form.Get(indexedKey); v != "" {
			res = append(res, v)
		} else if i == 0 {
			break
		}
	}
	return res
}

func (rp *RequestParams) GetMetadata() map[string]string {
	res := make(map[string]string)
	if rp.IsJSON {
		if val, ok := rp.RawJSON["metadata"]; ok {
			if m, ok := val.(map[string]interface{}); ok {
				for k, v := range m {
					switch sv := v.(type) {
					case string:
						res[k] = sv
					case float64:
						res[k] = strconv.FormatInt(int64(sv), 10)
					case bool:
						res[k] = strconv.FormatBool(sv)
					}
				}
			}
		}
		return res
	}
	for k, vals := range rp.Form {
		if strings.HasPrefix(k, "metadata[") && strings.HasSuffix(k, "]") {
			inner := k[len("metadata[") : len(k)-1]
			if inner != "" && len(vals) > 0 {
				res[inner] = vals[0]
			}
		}
	}
	return res
}

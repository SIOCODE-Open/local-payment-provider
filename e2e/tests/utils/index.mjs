import path from 'path';
import fs from 'fs';
import cp from 'child_process';
import http from 'http';
import crypto from 'crypto';

export function getBasePort() {
    return parseInt(process.env.TEST_PORT || '8080', 10);
}

export function getSnapshotPort(offset = 0) {
    return getBasePort() + offset;
}

export function getSnapshotUrl(offset = 0) {
    if (offset === 0 && process.env.BASE_URL) {
        return process.env.BASE_URL;
    }
    return `http://localhost:${getSnapshotPort(offset)}`;
}

export function getWebhookPort() {
    return parseInt(process.env.WEBHOOK_PORT || '8099', 10);
}

export async function waitAvailable(baseUrl, timeoutMs = 120000, intervalMs = 500) {
    const startTime = Date.now();
    while (true) {
        try {
            const response = await fetch(`${baseUrl}/healthz`);
            if (response.ok) {
                return;
            }
        } catch (err) {
            // Ignore errors and retry
        }
        if (Date.now() - startTime > timeoutMs) {
            throw new Error(`Service at ${baseUrl} did not become available within ${timeoutMs} ms`);
        }
        await new Promise(resolve => setTimeout(resolve, intervalMs));
    }
}

export async function launchSnapshot(snapshotName, port) {
    const snapshotsDir = path.resolve(process.cwd(), 'snapshots', snapshotName);
    if (!fs.existsSync(snapshotsDir)) {
        throw new Error(`Snapshot directory does not exist: ${snapshotsDir}`);
    }

    const snapshotDockerCompose = path.join(snapshotsDir, 'docker-compose.yaml');
    if (!fs.existsSync(snapshotDockerCompose)) {
        throw new Error(`docker-compose.yaml not found in snapshot directory: ${snapshotDockerCompose}`);
    }

    const env = { ...process.env };
    if (port) {
        env.SNAPSHOT_PORT = String(port);
    }
    const projectName = `lpp-${snapshotName}-${port || 'default'}`;

    return new Promise((resolve, reject) => {
        try {
            const dockerComposeProcess = cp.spawn(
                'docker',
                ['compose', '-p', projectName, '-f', snapshotDockerCompose, 'up', '-d', '--build'],
                {
                    cwd: snapshotsDir,
                    env,
                }
            );
            dockerComposeProcess.on('close', (code) => {
                if (code === 0) {
                    resolve();
                } else {
                    reject(new Error(`docker-compose process exited with code ${code}`));
                }
            });
        } catch (err) {
            reject(err);
        }
    });
}

export async function teardownSnapshot(snapshotName, port) {
    const snapshotsDir = path.resolve(process.cwd(), 'snapshots', snapshotName);
    const snapshotDockerCompose = path.join(snapshotsDir, 'docker-compose.yaml');
    const env = { ...process.env };
    if (port) {
        env.SNAPSHOT_PORT = String(port);
    }
    const projectName = `lpp-${snapshotName}-${port || 'default'}`;

    return new Promise((resolve, reject) => {
        try {
            const dockerComposeProcess = cp.spawn(
                'docker',
                ['compose', '-p', projectName, '-f', snapshotDockerCompose, 'down', '-v'],
                {
                    cwd: snapshotsDir,
                    env,
                }
            );
            dockerComposeProcess.on('close', (code) => {
                if (code === 0) {
                    resolve();
                } else {
                    reject(new Error(`docker-compose process exited with code ${code}`));
                }
            });
        } catch (err) {
            reject(err);
        }
    });
}

export class WebhookReceiver {
    constructor(port = getWebhookPort()) {
        this.port = port;
        this.events = [];
        this.server = null;
    }

    async start() {
        return new Promise((resolve, reject) => {
            this.server = http.createServer((req, res) => {
                const chunks = [];
                req.on('data', chunk => chunks.push(chunk));
                req.on('end', () => {
                    const rawBody = Buffer.concat(chunks).toString('utf8');
                    let parsedBody = null;
                    try {
                        parsedBody = JSON.parse(rawBody);
                    } catch (e) {
                        // Ignore JSON parse error
                    }
                    this.events.push({
                        method: req.method,
                        url: req.url,
                        headers: req.headers,
                        rawBody,
                        body: parsedBody,
                        timestamp: Date.now(),
                    });
                    res.writeHead(200, { 'Content-Type': 'application/json' });
                    res.end(JSON.stringify({ received: true }));
                });
            });
            this.server.on('error', reject);
            this.server.listen(this.port, '0.0.0.0', () => resolve());
        });
    }

    async stop() {
        if (!this.server) return;
        return new Promise((resolve) => {
            this.server.close(() => {
                this.server = null;
                resolve();
            });
        });
    }

    clear() {
        this.events = [];
    }

    static verifyStripeSignature(rawBody, signatureHeader, secret) {
        if (!signatureHeader) return false;
        const parts = signatureHeader.split(',');
        let timestamp = null;
        let v1 = null;
        for (const part of parts) {
            const [k, v] = part.trim().split('=');
            if (k === 't') timestamp = v;
            if (k === 'v1') v1 = v;
        }
        if (!timestamp || !v1) return false;
        const signedPayload = `${timestamp}.${rawBody}`;
        const expected = crypto.createHmac('sha256', secret).update(signedPayload, 'utf8').digest('hex');
        return expected === v1;
    }

    static verifyPaymentSignature(rawBody, signatureHeader, secret) {
        if (!signatureHeader || !signatureHeader.startsWith('sha256=')) return false;
        const actual = signatureHeader.slice('sha256='.length);
        const expected = crypto.createHmac('sha256', secret).update(rawBody, 'utf8').digest('hex');
        return expected === actual;
    }
}

export class PaymentProviderClient {
    constructor(baseUrl, apiKey = null) {
        this.baseUrl = baseUrl;
        this.apiKey = apiKey;
    }

    _headers(extra = {}) {
        const h = { ...extra };
        if (this.apiKey) {
            h['Authorization'] = `Bearer ${this.apiKey}`;
        }
        return h;
    }

    async request(method, reqPath, { query, json, form, headers = {}, redirect = 'follow' } = {}) {
        let url = `${this.baseUrl}${reqPath}`;
        if (query && Object.keys(query).length > 0) {
            const qs = new URLSearchParams(query).toString();
            url += `?${qs}`;
        }
        const reqHeaders = this._headers(headers);
        let body = undefined;
        if (json !== undefined) {
            reqHeaders['Content-Type'] = 'application/json';
            body = JSON.stringify(json);
        } else if (form !== undefined) {
            reqHeaders['Content-Type'] = 'application/x-www-form-urlencoded';
            body = form instanceof URLSearchParams ? form.toString() : new URLSearchParams(form).toString();
        }
        return await fetch(url, {
            method,
            headers: reqHeaders,
            body,
            redirect,
        });
    }

    // Health Check
    async healthz() {
        const res = await this.request('GET', '/healthz');
        if (!res.ok) throw new Error(`Health check failed with status ${res.status}`);
        return await res.json();
    }

    // Hosted Checkout UI
    async getCheckoutLink(id, query = {}) {
        return await this.request('GET', `/checkout/link/${id}`, { query });
    }

    async submitCheckoutLink(id, formData = {}, asJson = false) {
        return await this.request('POST', `/checkout/link/${id}/submit`, {
            ...(asJson ? { json: formData } : { form: formData }),
            redirect: 'manual',
        });
    }

    async getCheckoutIntent(id, query = {}) {
        return await this.request('GET', `/checkout/intent/${id}`, { query });
    }

    async submitCheckoutIntent(id, formData = {}, asJson = false) {
        return await this.request('POST', `/checkout/intent/${id}/submit`, {
            ...(asJson ? { json: formData } : { form: formData }),
            redirect: 'manual',
        });
    }

    async getCheckoutSessionPage(id, query = {}) {
        return await this.request('GET', `/checkout/session/${id}`, { query });
    }

    async submitCheckoutSessionPage(id, formData = {}, asJson = false) {
        return await this.request('POST', `/checkout/session/${id}/submit`, {
            ...(asJson ? { json: formData } : { form: formData }),
            redirect: 'manual',
        });
    }

    // Native Payment API
    async listNativePaymentLinks() {
        return await this.request('GET', '/payments/links');
    }

    async createNativePaymentLink(data) {
        return await this.request('POST', '/payments/links', { json: data });
    }

    async getNativePaymentLink(id) {
        return await this.request('GET', `/payments/links/${id}`);
    }

    async createNativePaymentIntent(data) {
        return await this.request('POST', '/payments/intents', { json: data });
    }

    async listNativePaymentIntents(query = {}) {
        return await this.request('GET', '/payments/intents', { query });
    }

    async getNativePaymentIntent(id) {
        return await this.request('GET', `/payments/intents/${id}`);
    }

    async confirmNativePaymentIntent(id, data = {}) {
        return await this.request('POST', `/payments/intents/${id}/confirm`, { json: data });
    }

    async cancelNativePaymentIntent(id, data = {}) {
        return await this.request('POST', `/payments/intents/${id}/cancel`, { json: data });
    }

    async refundNativePaymentIntent(id, data = {}) {
        return await this.request('POST', `/payments/intents/${id}/refund`, { json: data });
    }

    async listNativeRefunds(query = {}) {
        return await this.request('GET', '/payments/refunds', { query });
    }

    async getNativeRefund(id) {
        return await this.request('GET', `/payments/refunds/${id}`);
    }

    // Stripe-Compatible API (/v1)
    async createStripePaymentIntent(data, asForm = true) {
        return await this.request('POST', '/v1/payment_intents', asForm ? { form: data } : { json: data });
    }

    async listStripePaymentIntents(query = {}) {
        return await this.request('GET', '/v1/payment_intents', { query });
    }

    async getStripePaymentIntent(id) {
        return await this.request('GET', `/v1/payment_intents/${id}`);
    }

    async updateStripePaymentIntent(id, data, asForm = true) {
        return await this.request('POST', `/v1/payment_intents/${id}`, asForm ? { form: data } : { json: data });
    }

    async confirmStripePaymentIntent(id, data = {}, asForm = true) {
        return await this.request('POST', `/v1/payment_intents/${id}/confirm`, asForm ? { form: data } : { json: data });
    }

    async cancelStripePaymentIntent(id, data = {}, asForm = true) {
        return await this.request('POST', `/v1/payment_intents/${id}/cancel`, asForm ? { form: data } : { json: data });
    }

    async captureStripePaymentIntent(id, data = {}, asForm = true) {
        return await this.request('POST', `/v1/payment_intents/${id}/capture`, asForm ? { form: data } : { json: data });
    }

    async createStripePaymentLink(data, asForm = true) {
        return await this.request('POST', '/v1/payment_links', asForm ? { form: data } : { json: data });
    }

    async listStripePaymentLinks(query = {}) {
        return await this.request('GET', '/v1/payment_links', { query });
    }

    async getStripePaymentLink(id) {
        return await this.request('GET', `/v1/payment_links/${id}`);
    }

    async updateStripePaymentLink(id, data, asForm = true) {
        return await this.request('POST', `/v1/payment_links/${id}`, asForm ? { form: data } : { json: data });
    }

    async createStripeCheckoutSession(data, asForm = true) {
        return await this.request('POST', '/v1/checkout/sessions', asForm ? { form: data } : { json: data });
    }

    async listStripeCheckoutSessions(query = {}) {
        return await this.request('GET', '/v1/checkout/sessions', { query });
    }

    async getStripeCheckoutSession(id) {
        return await this.request('GET', `/v1/checkout/sessions/${id}`);
    }

    async expireStripeCheckoutSession(id) {
        return await this.request('POST', `/v1/checkout/sessions/${id}/expire`);
    }

    async createStripeCustomer(data, asForm = true) {
        return await this.request('POST', '/v1/customers', asForm ? { form: data } : { json: data });
    }

    async listStripeCustomers(query = {}) {
        return await this.request('GET', '/v1/customers', { query });
    }

    async getStripeCustomer(id) {
        return await this.request('GET', `/v1/customers/${id}`);
    }

    async updateStripeCustomer(id, data, asForm = true) {
        return await this.request('POST', `/v1/customers/${id}`, asForm ? { form: data } : { json: data });
    }

    async deleteStripeCustomer(id) {
        return await this.request('DELETE', `/v1/customers/${id}`);
    }

    async createStripePaymentMethod(data, asForm = true) {
        return await this.request('POST', '/v1/payment_methods', asForm ? { form: data } : { json: data });
    }

    async listStripePaymentMethods(query = {}) {
        return await this.request('GET', '/v1/payment_methods', { query });
    }

    async getStripePaymentMethod(id) {
        return await this.request('GET', `/v1/payment_methods/${id}`);
    }

    async attachStripePaymentMethod(id, data, asForm = true) {
        return await this.request('POST', `/v1/payment_methods/${id}/attach`, asForm ? { form: data } : { json: data });
    }

    async detachStripePaymentMethod(id) {
        return await this.request('POST', `/v1/payment_methods/${id}/detach`);
    }

    async listStripeCharges(query = {}) {
        return await this.request('GET', '/v1/charges', { query });
    }

    async getStripeCharge(id) {
        return await this.request('GET', `/v1/charges/${id}`);
    }

    async createStripeRefund(data, asForm = true) {
        return await this.request('POST', '/v1/refunds', asForm ? { form: data } : { json: data });
    }

    async listStripeRefunds(query = {}) {
        return await this.request('GET', '/v1/refunds', { query });
    }

    async getStripeRefund(id) {
        return await this.request('GET', `/v1/refunds/${id}`);
    }

    async listStripeEvents(query = {}) {
        return await this.request('GET', '/v1/events', { query });
    }

    async getStripeEvent(id) {
        return await this.request('GET', `/v1/events/${id}`);
    }

    async createStripeWebhookEndpoint(data, asForm = true) {
        return await this.request('POST', '/v1/webhook_endpoints', asForm ? { form: data } : { json: data });
    }

    async listStripeWebhookEndpoints() {
        return await this.request('GET', '/v1/webhook_endpoints');
    }

    async getStripeWebhookEndpoint(id) {
        return await this.request('GET', `/v1/webhook_endpoints/${id}`);
    }

    async deleteStripeWebhookEndpoint(id) {
        return await this.request('DELETE', `/v1/webhook_endpoints/${id}`);
    }

    // Testing & Spying API
    async getTestingConfig() {
        return await this.request('GET', '/_testing/config');
    }

    async updateTestingConfig(data) {
        return await this.request('PUT', '/_testing/config', { json: data });
    }

    async getTestingRequests(query = {}) {
        return await this.request('GET', '/_testing/requests', { query });
    }

    async getTestingPayments(query = {}) {
        return await this.request('GET', '/_testing/payments', { query });
    }

    async getTestingWebhooks(query = {}) {
        return await this.request('GET', '/_testing/webhooks', { query });
    }

    async triggerTestingWebhook(data) {
        return await this.request('POST', '/_testing/webhooks/trigger', { json: data });
    }

    async resetTestingState() {
        return await this.request('POST', '/_testing/reset');
    }
}

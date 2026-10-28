import { expect } from 'chai';
import {
    PaymentProviderClient,
    getSnapshotPort,
    getSnapshotUrl,
    launchSnapshot,
    teardownSnapshot,
    waitAvailable,
} from './utils/index.mjs';

describe('stripe-disabled', () => {
    const port = getSnapshotPort(2);
    const baseUrl = getSnapshotUrl(2);
    const client = new PaymentProviderClient(baseUrl);

    before(async () => {
        await launchSnapshot('stripe-disabled', port);
        await waitAvailable(baseUrl);
    });

    after(async () => {
        await teardownSnapshot('stripe-disabled', port);
    });

    describe('Stripe-Compatible API endpoints should return 404 when disabled', () => {
        it('POST /v1/payment_intents should return 404', async () => {
            const res = await client.createStripePaymentIntent({ amount: '1000', currency: 'usd' }, true);
            expect(res.status).to.equal(404);
        });

        it('GET /v1/payment_intents should return 404', async () => {
            const res = await client.listStripePaymentIntents();
            expect(res.status).to.equal(404);
        });

        it('GET /v1/customers should return 404', async () => {
            const res = await client.listStripeCustomers();
            expect(res.status).to.equal(404);
        });

        it('POST /v1/checkout/sessions should return 404', async () => {
            const res = await client.createStripeCheckoutSession({
                success_url: 'http://localhost:3000/s',
                amount: '1000',
                currency: 'usd',
            }, true);
            expect(res.status).to.equal(404);
        });

        it('GET /v1/charges and /v1/refunds and /v1/events should return 404', async () => {
            const r1 = await client.listStripeCharges();
            expect(r1.status).to.equal(404);
            const r2 = await client.listStripeRefunds();
            expect(r2.status).to.equal(404);
            const r3 = await client.listStripeEvents();
            expect(r3.status).to.equal(404);
        });
    });

    describe('Native Payment API and Hosted Checkout should still work', () => {
        it('GET /healthz should return OK', async () => {
            const health = await client.healthz();
            expect(health).to.have.property('status', 'OK');
        });

        it('POST /payments/intents should work', async () => {
            const res = await client.createNativePaymentIntent({
                amount: 1800,
                currency: 'usd',
                payment_method: 'pm_card_visa',
                confirm: true,
            });
            expect(res.status).to.equal(201);
            const pi = await res.json();
            expect(pi.status).to.equal('succeeded');
        });

        it('GET /checkout/link/plink_pro_plan should render HTML checkout form', async () => {
            const res = await client.getCheckoutLink('plink_pro_plan');
            expect(res.status).to.equal(200);
            const html = await res.text();
            expect(html).to.include('<form');
        });
    });
});

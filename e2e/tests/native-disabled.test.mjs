import { expect } from 'chai';
import {
    PaymentProviderClient,
    getSnapshotPort,
    getSnapshotUrl,
    launchSnapshot,
    teardownSnapshot,
    waitAvailable,
} from './utils/index.mjs';

describe('native-disabled', () => {
    const port = getSnapshotPort(1);
    const baseUrl = getSnapshotUrl(1);
    const client = new PaymentProviderClient(baseUrl);

    before(async () => {
        await launchSnapshot('native-disabled', port);
        await waitAvailable(baseUrl);
    });

    after(async () => {
        await teardownSnapshot('native-disabled', port);
    });

    describe('Native Payment API endpoints should return 404 when disabled', () => {
        it('GET /payments/links should return 404', async () => {
            const res = await client.listNativePaymentLinks();
            expect(res.status).to.equal(404);
        });

        it('POST /payments/links should return 404', async () => {
            const res = await client.createNativePaymentLink({ amount: 1000, currency: 'usd' });
            expect(res.status).to.equal(404);
        });

        it('POST /payments/intents should return 404', async () => {
            const res = await client.createNativePaymentIntent({ amount: 1000, currency: 'usd' });
            expect(res.status).to.equal(404);
        });

        it('GET /payments/intents should return 404', async () => {
            const res = await client.listNativePaymentIntents();
            expect(res.status).to.equal(404);
        });

        it('GET /payments/refunds should return 404', async () => {
            const res = await client.listNativeRefunds();
            expect(res.status).to.equal(404);
        });
    });

    describe('Stripe-Compatible API and Hosted Checkout should still work', () => {
        it('GET /healthz should return OK', async () => {
            const health = await client.healthz();
            expect(health).to.have.property('status', 'OK');
        });

        it('POST /v1/payment_intents should work', async () => {
            const res = await client.createStripePaymentIntent({
                amount: '2500',
                currency: 'usd',
                payment_method: 'pm_card_visa',
                confirm: 'true',
            }, true);
            expect(res.status).to.equal(200);
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

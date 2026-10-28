import { expect } from 'chai';
import {
    PaymentProviderClient,
    getSnapshotPort,
    getSnapshotUrl,
    launchSnapshot,
    teardownSnapshot,
    waitAvailable,
} from './utils/index.mjs';

describe('auth-enabled', () => {
    const port = getSnapshotPort(3);
    const baseUrl = getSnapshotUrl(3);
    const unauthClient = new PaymentProviderClient(baseUrl);
    const wrongKeyClient = new PaymentProviderClient(baseUrl, 'sk_test_wrong_key');
    const authClient = new PaymentProviderClient(baseUrl, 'sk_test_12345');

    before(async () => {
        await launchSnapshot('auth-enabled', port);
        await waitAvailable(baseUrl);
    });

    after(async () => {
        await teardownSnapshot('auth-enabled', port);
    });

    describe('API Key Authentication Enforcement', () => {
        it('Public endpoints (/healthz and /checkout/...) should not require API key', async () => {
            const health = await unauthClient.healthz();
            expect(health).to.have.property('status', 'OK');

            const checkoutRes = await unauthClient.getCheckoutLink('plink_pro_plan');
            expect(checkoutRes.status).to.equal(200);
        });

        it('Unauthenticated requests to /payments/... and /v1/... should return 401', async () => {
            const r1 = await unauthClient.listNativePaymentLinks();
            expect(r1.status).to.equal(401);

            const r2 = await unauthClient.listStripeCustomers();
            expect(r2.status).to.equal(401);
        });

        it('Requests with invalid API key should return 401', async () => {
            const r1 = await wrongKeyClient.listNativePaymentLinks();
            expect(r1.status).to.equal(401);

            const r2 = await wrongKeyClient.listStripeCustomers();
            expect(r2.status).to.equal(401);
        });

        it('Requests with valid Bearer API key should succeed', async () => {
            const r1 = await authClient.listNativePaymentLinks();
            expect(r1.status).to.equal(200);

            const r2 = await authClient.listStripeCustomers();
            expect(r2.status).to.equal(200);
        });

        it('Requests with valid HTTP Basic Auth API key (Stripe SDK style) should succeed', async () => {
            const basicToken = Buffer.from('pk_test_67890:').toString('base64');
            const res = await unauthClient.request('GET', '/v1/customers', {
                headers: {
                    Authorization: `Basic ${basicToken}`,
                },
            });
            expect(res.status).to.equal(200);
        });
    });
});

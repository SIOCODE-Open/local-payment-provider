import { expect } from 'chai';
import {
    PaymentProviderClient,
    WebhookReceiver,
    getSnapshotPort,
    getSnapshotUrl,
    getWebhookPort,
    launchSnapshot,
    teardownSnapshot,
    waitAvailable,
} from './utils/index.mjs';

describe('simple', () => {
    const port = getSnapshotPort(0);
    const baseUrl = getSnapshotUrl(0);
    const webhookPort = getWebhookPort();
    const client = new PaymentProviderClient(baseUrl);
    const webhookReceiver = new WebhookReceiver(webhookPort);

    before(async () => {
        await webhookReceiver.start();
        await launchSnapshot('simple', port);
        await waitAvailable(baseUrl);
    });

    after(async () => {
        await teardownSnapshot('simple', port);
        await webhookReceiver.stop();
    });

    describe('Health Check', () => {
        it('GET /healthz should return status OK', async () => {
            const res = await client.healthz();
            expect(res).to.have.property('status', 'OK');
        });
    });

    describe('Native Payment API (/payments/...)', () => {
        let createdIntentId;
        let createdRefundId;

        it('GET /payments/links should list pre-configured payment links', async () => {
            const res = await client.listNativePaymentLinks();
            expect(res.status).to.equal(200);
            const body = await res.json();
            expect(body).to.have.property('payment_links').that.is.an('array');
            const proLink = body.payment_links.find(l => l.id === 'plink_pro_plan');
            expect(proLink).to.exist;
            expect(proLink).to.include({
                id: 'plink_pro_plan',
                object: 'payment_link',
                title: 'Pro Plan Subscription',
                amount: 2999,
                currency: 'usd',
                active: true,
            });
            expect(proLink.metadata).to.deep.equal({ plan: 'pro' });
        });

        it('POST /payments/links should create a new payment link (201 Created)', async () => {
            const res = await client.createNativePaymentLink({
                id: 'plink_starter',
                title: 'Starter Pack',
                description: 'One-time starter pack purchase',
                amount: 1500,
                currency: 'usd',
                callback_url: 'http://localhost:3000/payment/callback',
                success_url: 'http://localhost:3000/payment/success',
                cancel_url: 'http://localhost:3000/payment/cancel',
                metadata: { sku: 'starter_01' },
            });
            expect(res.status).to.equal(201);
            const link = await res.json();
            expect(link).to.include({
                id: 'plink_starter',
                object: 'payment_link',
                title: 'Starter Pack',
                amount: 1500,
                currency: 'usd',
                active: true,
            });
            expect(link.url).to.include('/checkout/link/plink_starter');
            expect(link.metadata).to.deep.equal({ sku: 'starter_01' });
        });

        it('POST /payments/links should fail with 400 when amount <= 0 or currency missing', async () => {
            const res1 = await client.createNativePaymentLink({ amount: 0, currency: 'usd' });
            expect(res1.status).to.equal(400);

            const res2 = await client.createNativePaymentLink({ amount: 1000 });
            expect(res2.status).to.equal(400);
        });

        it('GET /payments/links/{id} should retrieve a payment link or return 404', async () => {
            const res = await client.getNativePaymentLink('plink_starter');
            expect(res.status).to.equal(200);
            const link = await res.json();
            expect(link.id).to.equal('plink_starter');

            const res404 = await client.getNativePaymentLink('plink_nonexistent');
            expect(res404.status).to.equal(404);
        });

        it('POST /payments/intents should create an unconfirmed payment intent (201 Created)', async () => {
            webhookReceiver.clear();
            const res = await client.createNativePaymentIntent({
                amount: 4999,
                currency: 'usd',
                description: 'Order #1042',
                customer_id: 'cus_default_1',
                customer_email: 'alice@example.com',
                payment_method: 'pm_card_visa',
                confirm: false,
                callback_url: 'http://localhost:3000/payment/callback',
                success_url: 'http://localhost:3000/payment/success',
                cancel_url: 'http://localhost:3000/payment/cancel',
                metadata: { order_id: '1042' },
            });
            expect(res.status).to.equal(201);
            const intent = await res.json();
            expect(intent.id).to.match(/^pi_/);
            expect(intent.object).to.equal('payment_intent');
            expect(intent.amount).to.equal(4999);
            expect(intent.amount_received).to.equal(0);
            expect(intent.currency).to.equal('usd');
            expect(intent.customer).to.equal('cus_default_1');
            expect(intent.customer_email).to.equal('alice@example.com');
            expect(intent.checkout_url).to.include(`/checkout/intent/${intent.id}`);
            expect(intent.status).to.be.oneOf(['requires_confirmation', 'requires_payment_method']);
            createdIntentId = intent.id;
        });

        it('POST /payments/intents should fail with 400 on invalid amount or missing currency', async () => {
            const res1 = await client.createNativePaymentIntent({ amount: -10, currency: 'usd' });
            expect(res1.status).to.equal(400);

            const res2 = await client.createNativePaymentIntent({ amount: 500 });
            expect(res2.status).to.equal(400);
        });

        it('POST /payments/intents with confirm=true and declined card should return 402', async () => {
            const res = await client.createNativePaymentIntent({
                amount: 2500,
                currency: 'usd',
                payment_method: 'pm_card_chargeDeclinedInsufficientFunds',
                confirm: true,
            });
            expect(res.status).to.equal(402);
            const body = await res.json();
            expect(body).to.have.property('error');
            expect(body.error).to.include({
                type: 'card_error',
                code: 'card_declined',
                decline_code: 'insufficient_funds',
            });
            expect(body.error.payment_intent).to.have.property('status', 'requires_payment_method');
        });

        it('GET /payments/intents and GET /payments/intents/{id} should list and get intents', async () => {
            const listRes = await client.listNativePaymentIntents({ customer: 'cus_default_1' });
            expect(listRes.status).to.equal(200);
            const listBody = await listRes.json();
            expect(listBody.payment_intents).to.be.an('array');
            expect(listBody.payment_intents.some(pi => pi.id === createdIntentId)).to.be.true;

            const getRes = await client.getNativePaymentIntent(createdIntentId);
            expect(getRes.status).to.equal(200);
            const pi = await getRes.json();
            expect(pi.id).to.equal(createdIntentId);

            const get404 = await client.getNativePaymentIntent('pi_nonexistent');
            expect(get404.status).to.equal(404);
        });

        it('POST /payments/intents/{id}/confirm should confirm and pay the intent', async () => {
            const res = await client.confirmNativePaymentIntent(createdIntentId, {
                payment_method: 'pm_card_visa',
                customer_email: 'alice@example.com',
            });
            expect(res.status).to.equal(200);
            const pi = await res.json();
            expect(pi.status).to.equal('succeeded');
            expect(pi.amount_received).to.equal(4999);
            expect(pi.latest_charge).to.match(/^ch_/);

            // Confirming an already succeeded intent should return 400
            const resAgain = await client.confirmNativePaymentIntent(createdIntentId, {});
            expect(resAgain.status).to.equal(400);
        });

        it('POST /payments/intents/{id}/cancel should cancel an unconfirmed intent', async () => {
            const createRes = await client.createNativePaymentIntent({
                amount: 1999,
                currency: 'usd',
            });
            const pi = await createRes.json();

            const cancelRes = await client.cancelNativePaymentIntent(pi.id, {
                cancellation_reason: 'requested_by_customer',
            });
            expect(cancelRes.status).to.equal(200);
            const canceled = await cancelRes.json();
            expect(canceled.status).to.equal('canceled');
            expect(canceled.canceled_at).to.be.a('number');
            expect(canceled.cancellation_reason).to.equal('requested_by_customer');

            // Canceling again or canceling succeeded intent should return 400
            const cancelAgain = await client.cancelNativePaymentIntent(pi.id, {});
            expect(cancelAgain.status).to.equal(400);
            const cancelSucceeded = await client.cancelNativePaymentIntent(createdIntentId, {});
            expect(cancelSucceeded.status).to.equal(400);
        });

        it('POST /payments/intents/{id}/refund should partially and fully refund a succeeded intent', async () => {
            const refundRes1 = await client.refundNativePaymentIntent(createdIntentId, {
                amount: 2000,
                reason: 'requested_by_customer',
                metadata: { note: 'partial refund' },
            });
            expect(refundRes1.status).to.equal(201);
            const refund1 = await refundRes1.json();
            expect(refund1.id).to.match(/^re_/);
            expect(refund1.object).to.equal('refund');
            expect(refund1.amount).to.equal(2000);
            expect(refund1.payment_intent).to.equal(createdIntentId);
            expect(refund1.reason).to.equal('requested_by_customer');
            expect(refund1.status).to.equal('succeeded');
            createdRefundId = refund1.id;

            // Exceeding remaining refundable amount (4999 - 2000 = 2999) should return 400
            const overRefund = await client.refundNativePaymentIntent(createdIntentId, {
                amount: 4000,
            });
            expect(overRefund.status).to.equal(400);

            // Refund remaining balance (omitting amount)
            const refundRes2 = await client.refundNativePaymentIntent(createdIntentId, {});
            expect(refundRes2.status).to.equal(201);
            const refund2 = await refundRes2.json();
            expect(refund2.amount).to.equal(2999);
        });

        it('GET /payments/refunds and GET /payments/refunds/{id} should list and retrieve refunds', async () => {
            const listRes = await client.listNativeRefunds({ payment_intent: createdIntentId });
            expect(listRes.status).to.equal(200);
            const listBody = await listRes.json();
            expect(listBody.refunds).to.be.an('array');
            expect(listBody.refunds.length).to.equal(2);

            const getRes = await client.getNativeRefund(createdRefundId);
            expect(getRes.status).to.equal(200);
            const refund = await getRes.json();
            expect(refund.id).to.equal(createdRefundId);

            const get404 = await client.getNativeRefund('re_nonexistent');
            expect(get404.status).to.equal(404);
        });
    });

    describe('Hosted Checkout UI (/checkout/...)', () => {
        describe('Payment Link Checkout', () => {
            it('GET /checkout/link/{id} should render HTML checkout form for active link', async () => {
                const res = await client.getCheckoutLink('plink_pro_plan', {
                    state: 'xyz123',
                    customer_email: 'bob@example.com',
                });
                expect(res.status).to.equal(200);
                expect(res.headers.get('content-type')).to.include('text/html');
                const html = await res.text();
                expect(html).to.include('<form');
                expect(html).to.include('/checkout/link/plink_pro_plan/submit');
                expect(html).to.include('Pro Plan Subscription');
            });

            it('GET /checkout/link/{id} should return 404 for inactive or non-existent link', async () => {
                const resInactive = await client.getCheckoutLink('plink_inactive');
                expect(resInactive.status).to.equal(404);

                const resMissing = await client.getCheckoutLink('plink_does_not_exist');
                expect(resMissing.status).to.equal(404);
            });

            it('POST /checkout/link/{id}/submit with action=pay should redirect with 302 to success_url', async () => {
                const res = await client.submitCheckoutLink('plink_pro_plan', {
                    action: 'pay',
                    card_number: '4242424242424242',
                    customer_email: 'buyer@example.com',
                    state: 'state_abc',
                });
                expect(res.status).to.equal(302);
                const location = res.headers.get('location');
                expect(location).to.include('http://localhost:3000/payment/success');
                expect(location).to.include('status=succeeded');
                expect(location).to.include('payment_link_id=plink_pro_plan');
                expect(location).to.include('payment_intent_id=pi_');
                expect(location).to.include('state=state_abc');
            });

            it('POST /checkout/link/{id}/submit should support {CHECKOUT_SESSION_ID} template in success_url', async () => {
                const res = await client.submitCheckoutLink('plink_pro_plan', {
                    action: 'pay',
                    success_url: 'http://localhost:3000/done?sid={CHECKOUT_SESSION_ID}',
                });
                expect(res.status).to.equal(302);
                const location = res.headers.get('location');
                expect(location).to.not.include('{CHECKOUT_SESSION_ID}');
                expect(location).to.match(/sid=pi_/);
            });

            it('POST /checkout/link/{id}/submit with action=cancel should redirect with 302 to cancel_url', async () => {
                const res = await client.submitCheckoutLink('plink_pro_plan', {
                    action: 'cancel',
                    state: 'cancel_state_1',
                });
                expect(res.status).to.equal(302);
                const location = res.headers.get('location');
                expect(location).to.include('http://localhost:3000/payment/cancel');
                expect(location).to.include('status=canceled');
                expect(location).to.include('payment_link_id=plink_pro_plan');
                expect(location).to.include('state=cancel_state_1');
            });

            it('POST /checkout/link/{id}/submit with declined card should return 402 HTML with error div', async () => {
                const res = await client.submitCheckoutLink('plink_pro_plan', {
                    action: 'pay',
                    card_number: '4000000000000002',
                });
                expect(res.status).to.equal(402);
                const html = await res.text();
                expect(html).to.include('<div class="error">');
                expect(html).to.include('Your card was declined.');
            });

            it('POST /checkout/link/{id}/submit with declined card and redirect_on_error=true should 302 redirect', async () => {
                const res = await client.submitCheckoutLink('plink_pro_plan', {
                    action: 'pay',
                    card_number: '4000000000009995',
                    redirect_on_error: 'true',
                    state: 'err_state',
                });
                expect(res.status).to.equal(302);
                const location = res.headers.get('location');
                expect(location).to.include('http://localhost:3000/payment/callback');
                expect(location).to.include('status=failed');
                expect(location).to.include('error_code=');
                expect(location).to.include('error_message=');
                expect(location).to.include('state=err_state');
            });

            it('POST /checkout/link/{id}/submit without redirect URLs should render 200 HTML confirmation page', async () => {
                await client.createNativePaymentLink({
                    id: 'plink_no_redirect',
                    title: 'No Redirect Link',
                    amount: 1000,
                    currency: 'usd',
                });
                const res = await client.submitCheckoutLink('plink_no_redirect', {
                    action: 'pay',
                });
                expect(res.status).to.equal(200);
                expect(res.headers.get('content-type')).to.include('text/html');
            });
        });

        describe('Payment Intent Checkout', () => {
            it('GET /checkout/intent/{id} and POST /checkout/intent/{id}/submit should handle pay, cancel, and decline', async () => {
                const createRes = await client.createNativePaymentIntent({
                    amount: 3500,
                    currency: 'usd',
                    description: 'Intent UI Test',
                    callback_url: 'http://localhost:3000/intent/callback',
                    success_url: 'http://localhost:3000/intent/success',
                    cancel_url: 'http://localhost:3000/intent/cancel',
                });
                const pi = await createRes.json();

                const pageRes = await client.getCheckoutIntent(pi.id);
                expect(pageRes.status).to.equal(200);
                const pageHtml = await pageRes.text();
                expect(pageHtml).to.include(`/checkout/intent/${pi.id}/submit`);
                expect(pageHtml).to.include('Intent UI Test');

                // Declined card -> 402 HTML
                const declineRes = await client.submitCheckoutIntent(pi.id, {
                    action: 'pay',
                    card_number: '4000000000000069',
                });
                expect(declineRes.status).to.equal(402);
                const declineHtml = await declineRes.text();
                expect(declineHtml).to.include('<div class="error">');
                expect(declineHtml).to.include('expired');

                // Declined card with redirect_on_error=true -> 302
                const declineRedir = await client.submitCheckoutIntent(pi.id, {
                    action: 'pay',
                    card_number: '4000000000000069',
                    redirect_on_error: 'true',
                    state: 'pi_err',
                });
                expect(declineRedir.status).to.equal(302);
                expect(declineRedir.headers.get('location')).to.include('status=failed');

                // Pay succeeds -> 302
                const payRes = await client.submitCheckoutIntent(pi.id, {
                    action: 'pay',
                    card_number: '4242424242424242',
                    state: 'pi_ok',
                });
                expect(payRes.status).to.equal(302);
                const payLoc = payRes.headers.get('location');
                expect(payLoc).to.include('http://localhost:3000/intent/success');
                expect(payLoc).to.include(`payment_intent_id=${pi.id}`);
                expect(payLoc).to.include('status=succeeded');
                expect(payLoc).to.include('state=pi_ok');

                // Cancel on another intent -> 302
                const createRes2 = await client.createNativePaymentIntent({
                    amount: 1200,
                    currency: 'usd',
                    cancel_url: 'http://localhost:3000/intent/cancel',
                });
                const pi2 = await createRes2.json();
                const cancelRes = await client.submitCheckoutIntent(pi2.id, {
                    action: 'cancel',
                    state: 'pi_cancel',
                });
                expect(cancelRes.status).to.equal(302);
                expect(cancelRes.headers.get('location')).to.include('status=canceled');
            });
        });

        describe('Checkout Session UI', () => {
            it('GET /checkout/session/{id} and POST /checkout/session/{id}/submit should complete or cancel session', async () => {
                const sessionRes = await client.createStripeCheckoutSession({
                    success_url: 'http://localhost:3000/session/success?session_id={CHECKOUT_SESSION_ID}',
                    cancel_url: 'http://localhost:3000/session/cancel',
                    'line_items[0][price_data][unit_amount]': '2500',
                    'line_items[0][price_data][currency]': 'usd',
                    'line_items[0][price_data][product_data][name]': 'Pro Widget',
                    'line_items[0][quantity]': '2',
                }, true);
                expect(sessionRes.status).to.equal(200);
                const session = await sessionRes.json();
                expect(session.amount_total).to.equal(5000);

                const getRes = await client.getCheckoutSessionPage(session.id);
                expect(getRes.status).to.equal(200);
                const html = await getRes.text();
                expect(html).to.include(`/checkout/session/${session.id}/submit`);

                // Declined card -> 402
                const failRes = await client.submitCheckoutSessionPage(session.id, {
                    action: 'pay',
                    card_number: '4000000000000002',
                });
                expect(failRes.status).to.equal(402);
                const failHtml = await failRes.text();
                expect(failHtml).to.include('<div class="error">');

                // Pay -> 302 with {CHECKOUT_SESSION_ID} replaced
                const payRes = await client.submitCheckoutSessionPage(session.id, {
                    action: 'pay',
                    card_number: '4242424242424242',
                });
                expect(payRes.status).to.equal(302);
                const location = payRes.headers.get('location');
                expect(location).to.include(`session_id=${session.id}`);
                expect(location).to.not.include('{CHECKOUT_SESSION_ID}');

                // Verify session is now complete and paid
                const updatedSessionRes = await client.getStripeCheckoutSession(session.id);
                const updatedSession = await updatedSessionRes.json();
                expect(updatedSession.status).to.equal('complete');
                expect(updatedSession.payment_status).to.equal('paid');

                // Cancel flow on a second session
                const sessionRes2 = await client.createStripeCheckoutSession({
                    success_url: 'http://localhost:3000/session/success',
                    cancel_url: 'http://localhost:3000/session/cancel',
                    amount: 1000,
                    currency: 'usd',
                }, false);
                const session2 = await sessionRes2.json();
                const cancelRes = await client.submitCheckoutSessionPage(session2.id, {
                    action: 'cancel',
                });
                expect(cancelRes.status).to.equal(302);
                expect(cancelRes.headers.get('location')).to.include(`session_id=${session2.id}`);
                expect(cancelRes.headers.get('location')).to.include('status=canceled');
            });
        });
    });

    describe('Stripe-Compatible API (/v1/...)', () => {
        it('Should support PaymentIntents CRUD, confirm, manual capture, and cancel in both form-urlencoded and JSON', async () => {
            // Create with form-urlencoded and nested bracket metadata
            const createRes = await client.createStripePaymentIntent({
                amount: '3000',
                currency: 'usd',
                customer: 'cus_default_1',
                description: 'Stripe Form Intent',
                receipt_email: 'alice@example.com',
                'metadata[order_id]': '8765',
                'automatic_payment_methods[enabled]': 'true',
            }, true);
            expect(createRes.status).to.equal(200);
            const pi = await createRes.json();
            expect(pi.object).to.equal('payment_intent');
            expect(pi.amount).to.equal(3000);
            expect(pi.metadata).to.deep.equal({ order_id: '8765' });

            // Update PaymentIntent
            const updateRes = await client.updateStripePaymentIntent(pi.id, {
                description: 'Updated Description',
                'metadata[extra]': 'val',
            }, true);
            expect(updateRes.status).to.equal(200);
            const updatedPi = await updateRes.json();
            expect(updatedPi.description).to.equal('Updated Description');
            expect(updatedPi.metadata).to.include({ order_id: '8765', extra: 'val' });

            // Confirm PaymentIntent with form-urlencoded nested card number
            const confirmRes = await client.confirmStripePaymentIntent(pi.id, {
                'payment_method_data[card][number]': '4242424242424242',
            }, true);
            expect(confirmRes.status).to.equal(200);
            const confirmedPi = await confirmRes.json();
            expect(confirmedPi.status).to.equal('succeeded');
            expect(confirmedPi.amount_received).to.equal(3000);
            expect(confirmedPi.latest_charge).to.match(/^ch_/);

            // Manual capture flow
            const manualRes = await client.createStripePaymentIntent({
                amount: 4000,
                currency: 'usd',
                capture_method: 'manual',
                payment_method: 'pm_card_visa',
                confirm: true,
            }, false);
            expect(manualRes.status).to.equal(200);
            const manualPi = await manualRes.json();
            expect(manualPi.status).to.equal('requires_capture');
            expect(manualPi.amount_capturable).to.equal(4000);
            expect(manualPi.amount_received).to.equal(0);

            // Capture partial amount
            const captureRes = await client.captureStripePaymentIntent(manualPi.id, {
                amount_to_capture: '3200',
            }, true);
            expect(captureRes.status).to.equal(200);
            const capturedPi = await captureRes.json();
            expect(capturedPi.status).to.equal('succeeded');
            expect(capturedPi.amount_received).to.equal(3200);
            expect(capturedPi.amount_capturable).to.equal(0);
        });

        it('Should return Stripe error envelope on 404 and 402 card declines', async () => {
            const res404 = await client.getStripePaymentIntent('pi_unknown');
            expect(res404.status).to.equal(404);
            const err404 = await res404.json();
            expect(err404).to.have.property('error');
            expect(err404.error).to.include({
                type: 'invalid_request_error',
                code: 'resource_missing',
            });

            const resDecline = await client.createStripePaymentIntent({
                amount: '1500',
                currency: 'usd',
                payment_method: 'pm_card_chargeDeclinedExpiredCard',
                confirm: 'true',
            }, true);
            expect(resDecline.status).to.equal(402);
            const errDecline = await resDecline.json();
            expect(errDecline.error).to.include({
                type: 'card_error',
                code: 'expired_card',
                decline_code: 'expired_card',
            });
            expect(errDecline.error.payment_intent.last_payment_error).to.exist;
        });

        it('Should support Stripe PaymentLinks create, list, retrieve, and update', async () => {
            const createRes = await client.createStripePaymentLink({
                'line_items[0][price_data][unit_amount]': '1999',
                'line_items[0][price_data][currency]': 'usd',
                'line_items[0][price_data][product_data][name]': 'Basic Plan',
                'after_completion[type]': 'redirect',
                'after_completion[redirect][url]': 'http://localhost:3000/basic-done',
                'metadata[tier]': 'basic',
            }, true);
            expect(createRes.status).to.equal(200);
            const plink = await createRes.json();
            expect(plink.id).to.match(/^plink_/);
            expect(plink.object).to.equal('payment_link');
            expect(plink.amount).to.equal(1999);
            expect(plink.after_completion.redirect.url).to.equal('http://localhost:3000/basic-done');

            const listRes = await client.listStripePaymentLinks({ active: 'true' });
            expect(listRes.status).to.equal(200);
            const listBody = await listRes.json();
            expect(listBody.object).to.equal('list');
            expect(listBody.data.some(l => l.id === plink.id)).to.be.true;

            const updateRes = await client.updateStripePaymentLink(plink.id, {
                active: 'false',
                'metadata[updated]': 'yes',
            }, true);
            expect(updateRes.status).to.equal(200);
            const updated = await updateRes.json();
            expect(updated.active).to.equal(false);
        });

        it('Should support Stripe Checkout Sessions list, retrieve, and expire', async () => {
            const createRes = await client.createStripeCheckoutSession({
                success_url: 'http://localhost:3000/s',
                cancel_url: 'http://localhost:3000/c',
                amount: '1200',
                currency: 'usd',
                customer: 'cus_default_1',
            }, true);
            expect(createRes.status).to.equal(200);
            const cs = await createRes.json();
            expect(cs.id).to.match(/^cs_/);
            expect(cs.status).to.equal('open');

            const listRes = await client.listStripeCheckoutSessions({ customer: 'cus_default_1' });
            expect(listRes.status).to.equal(200);
            const listBody = await listRes.json();
            expect(listBody.object).to.equal('list');
            expect(listBody.data.some(s => s.id === cs.id)).to.be.true;

            const expireRes = await client.expireStripeCheckoutSession(cs.id);
            expect(expireRes.status).to.equal(200);
            const expired = await expireRes.json();
            expect(expired.status).to.equal('expired');

            // Expiring again should return 400
            const expireAgain = await client.expireStripeCheckoutSession(cs.id);
            expect(expireAgain.status).to.equal(400);
        });

        it('Should support Stripe Customers CRUD', async () => {
            const createRes = await client.createStripeCustomer({
                email: 'charlie@example.com',
                name: 'Charlie Brown',
                phone: '+15550001111',
                description: 'Test customer',
                'metadata[role]': 'user',
            }, true);
            expect(createRes.status).to.equal(200);
            const cus = await createRes.json();
            expect(cus.id).to.match(/^cus_/);
            expect(cus.object).to.equal('customer');
            expect(cus.email).to.equal('charlie@example.com');
            expect(cus.metadata).to.deep.equal({ role: 'user' });

            const listRes = await client.listStripeCustomers({ email: 'charlie@example.com' });
            expect(listRes.status).to.equal(200);
            const listBody = await listRes.json();
            expect(listBody.data.length).to.equal(1);

            const updateRes = await client.updateStripeCustomer(cus.id, {
                name: 'Charlie B.',
            }, true);
            expect(updateRes.status).to.equal(200);
            const updated = await updateRes.json();
            expect(updated.name).to.equal('Charlie B.');

            const delRes = await client.deleteStripeCustomer(cus.id);
            expect(delRes.status).to.equal(200);
            const deleted = await delRes.json();
            expect(deleted).to.deep.equal({
                id: cus.id,
                object: 'customer',
                deleted: true,
            });

            const get404 = await client.getStripeCustomer(cus.id);
            expect(get404.status).to.equal(404);
        });

        it('Should support Stripe PaymentMethods create, retrieve, list, attach, and detach', async () => {
            const createRes = await client.createStripePaymentMethod({
                type: 'card',
                'card[number]': '4242424242424242',
                'card[exp_month]': '11',
                'card[exp_year]': '2029',
                'card[cvc]': '999',
                'billing_details[name]': 'Alice Smith',
                'billing_details[email]': 'alice@example.com',
            }, true);
            expect(createRes.status).to.equal(200);
            const pm = await createRes.json();
            expect(pm.id).to.match(/^pm_/);
            expect(pm.object).to.equal('payment_method');
            expect(pm.card.last4).to.equal('4242');
            expect(pm.card.exp_month).to.equal(11);
            expect(pm.card.exp_year).to.equal(2029);
            expect(pm.billing_details.name).to.equal('Alice Smith');

            const attachRes = await client.attachStripePaymentMethod(pm.id, {
                customer: 'cus_default_1',
            }, true);
            expect(attachRes.status).to.equal(200);
            const attached = await attachRes.json();
            expect(attached.customer).to.equal('cus_default_1');

            const listRes = await client.listStripePaymentMethods({ customer: 'cus_default_1', type: 'card' });
            expect(listRes.status).to.equal(200);
            const listBody = await listRes.json();
            expect(listBody.data.some(m => m.id === pm.id)).to.be.true;

            const detachRes = await client.detachStripePaymentMethod(pm.id);
            expect(detachRes.status).to.equal(200);
            const detached = await detachRes.json();
            expect(detached.customer).to.be.null;
        });

        it('Should support Stripe Charges and Refunds', async () => {
            const piRes = await client.createStripePaymentIntent({
                amount: '5000',
                currency: 'usd',
                customer: 'cus_default_1',
                payment_method: 'pm_card_visa',
                confirm: 'true',
            }, true);
            const pi = await piRes.json();
            const chargeId = pi.latest_charge;
            expect(chargeId).to.match(/^ch_/);

            const chargesRes = await client.listStripeCharges({ payment_intent: pi.id });
            expect(chargesRes.status).to.equal(200);
            const chargesList = await chargesRes.json();
            expect(chargesList.data.length).to.equal(1);
            expect(chargesList.data[0].id).to.equal(chargeId);

            const chargeRes = await client.getStripeCharge(chargeId);
            expect(chargeRes.status).to.equal(200);
            const charge = await chargeRes.json();
            expect(charge.amount).to.equal(5000);
            expect(charge.paid).to.be.true;

            // Create refund by charge ID
            const refundRes = await client.createStripeRefund({
                charge: chargeId,
                amount: '1500',
                reason: 'duplicate',
                'metadata[ref]': 'r1',
            }, true);
            expect(refundRes.status).to.equal(200);
            const refund = await refundRes.json();
            expect(refund.id).to.match(/^re_/);
            expect(refund.amount).to.equal(1500);
            expect(refund.charge).to.equal(chargeId);
            expect(refund.payment_intent).to.equal(pi.id);

            const getRefundRes = await client.getStripeRefund(refund.id);
            expect(getRefundRes.status).to.equal(200);

            const listRefundsRes = await client.listStripeRefunds({ charge: chargeId });
            expect(listRefundsRes.status).to.equal(200);
            const listRefunds = await listRefundsRes.json();
            expect(listRefunds.data.length).to.equal(1);
        });

        it('Should support Stripe Events and WebhookEndpoints CRUD', async () => {
            const eventsRes = await client.listStripeEvents({ type: 'payment_intent.succeeded' });
            expect(eventsRes.status).to.equal(200);
            const eventsList = await eventsRes.json();
            expect(eventsList.object).to.equal('list');
            expect(eventsList.data.length).to.be.greaterThan(0);

            const eventId = eventsList.data[0].id;
            const singleEventRes = await client.getStripeEvent(eventId);
            expect(singleEventRes.status).to.equal(200);
            const singleEvent = await singleEventRes.json();
            expect(singleEvent.id).to.equal(eventId);

            // WebhookEndpoint CRUD
            const createWhRes = await client.createStripeWebhookEndpoint({
                url: `http://host.docker.internal:${webhookPort}/webhooks/custom`,
                'enabled_events[0]': 'payment_intent.succeeded',
                description: 'Custom webhook endpoint',
                secret: 'whsec_custom_123',
            }, true);
            expect(createWhRes.status).to.equal(200);
            const wh = await createWhRes.json();
            expect(wh.id).to.match(/^(we_|wh_)/);
            expect(wh.object).to.equal('webhook_endpoint');
            expect(wh.url).to.equal(`http://host.docker.internal:${webhookPort}/webhooks/custom`);

            const listWhRes = await client.listStripeWebhookEndpoints();
            expect(listWhRes.status).to.equal(200);
            const listWh = await listWhRes.json();
            expect(listWh.data.some(w => w.id === wh.id)).to.be.true;

            const getWhRes = await client.getStripeWebhookEndpoint(wh.id);
            expect(getWhRes.status).to.equal(200);

            const delWhRes = await client.deleteStripeWebhookEndpoint(wh.id);
            expect(delWhRes.status).to.equal(200);
            const delWh = await delWhRes.json();
            expect(delWh.deleted).to.be.true;
        });
    });

    describe('Webhook Delivery & Signature Verification', () => {
        it('Should deliver signed webhooks with valid Stripe-Signature and X-Payment-Signature headers', async () => {
            webhookReceiver.clear();

            const res = await client.createStripePaymentIntent({
                amount: 2222,
                currency: 'usd',
                payment_method: 'pm_card_visa',
                confirm: true,
            }, false);
            expect(res.status).to.equal(200);
            const pi = await res.json();

            // Wait briefly if async delivery is still in-flight
            const deadline = Date.now() + 5000;
            while (
                !webhookReceiver.events.some(
                    e => e.body && e.body.type === 'payment_intent.succeeded' && e.body.data?.object?.id === pi.id
                ) &&
                Date.now() < deadline
            ) {
                await new Promise(r => setTimeout(r, 100));
            }

            const matchingEvent = webhookReceiver.events.find(
                e => e.body && e.body.type === 'payment_intent.succeeded' && e.body.data?.object?.id === pi.id
            );
            expect(matchingEvent, 'Expected payment_intent.succeeded webhook to be received').to.exist;

            const stripeSig = matchingEvent.headers['stripe-signature'];
            const paymentSig = matchingEvent.headers['x-payment-signature'];
            expect(stripeSig).to.be.a('string');
            expect(paymentSig).to.be.a('string');
            expect(matchingEvent.headers['x-webhook-event']).to.equal('payment_intent.succeeded');
            expect(matchingEvent.headers['x-webhook-id']).to.equal(matchingEvent.body.id);

            expect(
                WebhookReceiver.verifyStripeSignature(matchingEvent.rawBody, stripeSig, 'whsec_test_secret')
            ).to.be.true;
            expect(
                WebhookReceiver.verifyPaymentSignature(matchingEvent.rawBody, paymentSig, 'whsec_test_secret')
            ).to.be.true;
        });
    });

    describe('Testing & Spying API (/_testing/...)', () => {
        it('GET & PUT /_testing/config should inspect and mutate runtime test behaviors', async () => {
            const getRes = await client.getTestingConfig();
            expect(getRes.status).to.equal(200);
            const initialCfg = await getRes.json();
            expect(initialCfg).to.have.property('always_fail', false);
            expect(initialCfg).to.have.property('declined_cards');

            // Enable always_fail
            const putRes = await client.updateTestingConfig({
                always_fail: true,
                always_fail_code: 'card_declined',
                always_fail_message: 'Forced test failure',
            });
            expect(putRes.status).to.equal(200);
            const updatedCfg = await putRes.json();
            expect(updatedCfg.always_fail).to.be.true;

            // Even a valid card should now fail with 402
            const failPiRes = await client.createNativePaymentIntent({
                amount: 1000,
                currency: 'usd',
                payment_method: 'pm_card_visa',
                confirm: true,
            });
            expect(failPiRes.status).to.equal(402);
            const failBody = await failPiRes.json();
            expect(failBody.error.message).to.equal('Forced test failure');

            // Restore always_fail = false
            await client.updateTestingConfig({
                always_fail: false,
            });
        });

        it('GET /_testing/requests should list recorded API requests with filtering', async () => {
            const res = await client.getTestingRequests({
                method: 'POST',
                path: '/v1/payment_intents',
            });
            expect(res.status).to.equal(200);
            const body = await res.json();
            expect(body).to.have.property('requests').that.is.an('array');
            expect(body.requests.length).to.be.greaterThan(0);
            expect(body.requests[0]).to.include.keys('id', 'method', 'path', 'status_code', 'timestamp');
        });

        it('GET /_testing/payments should return all in-memory payment objects', async () => {
            const res = await client.getTestingPayments({ status: 'succeeded' });
            expect(res.status).to.equal(200);
            const body = await res.json();
            expect(body).to.include.keys('payment_intents', 'charges', 'refunds', 'checkout_sessions');
            expect(body.payment_intents.every(pi => pi.status === 'succeeded')).to.be.true;
        });

        it('GET /_testing/webhooks and POST /_testing/webhooks/trigger should inspect and trigger webhooks', async () => {
            webhookReceiver.clear();
            const triggerRes = await client.triggerTestingWebhook({
                type: 'payment_intent.succeeded',
                url: `http://host.docker.internal:${webhookPort}/webhooks/triggered`,
                secret: 'whsec_triggered_secret',
                data: {
                    id: 'pi_synthetic_123',
                    object: 'payment_intent',
                    amount: 7777,
                    currency: 'usd',
                    status: 'succeeded',
                },
            });
            expect(triggerRes.status).to.equal(200);
            const triggerBody = await triggerRes.json();
            expect(triggerBody).to.have.property('event');
            expect(triggerBody).to.have.property('deliveries').that.is.an('array');
            expect(triggerBody.deliveries[0].status).to.equal('succeeded');

            const received = webhookReceiver.events.find(e => e.url === '/webhooks/triggered');
            expect(received).to.exist;
            expect(
                WebhookReceiver.verifyStripeSignature(
                    received.rawBody,
                    received.headers['stripe-signature'],
                    'whsec_triggered_secret'
                )
            ).to.be.true;

            const whLogRes = await client.getTestingWebhooks({
                event_type: 'payment_intent.succeeded',
                status: 'succeeded',
            });
            expect(whLogRes.status).to.equal(200);
            const whLogBody = await whLogRes.json();
            expect(whLogBody.deliveries.length).to.be.greaterThan(0);
        });

        it('POST /_testing/reset should reset state back to initial configuration', async () => {
            const resetRes = await client.resetTestingState();
            expect(resetRes.status).to.equal(200);
            const resetBody = await resetRes.json();
            expect(resetBody).to.have.property('status', 'OK');

            // Verify dynamic payment intents and requests were cleared
            const paymentsRes = await client.getTestingPayments();
            const paymentsBody = await paymentsRes.json();
            expect(paymentsBody.payment_intents).to.be.empty;
            expect(paymentsBody.charges).to.be.empty;
            expect(paymentsBody.refunds).to.be.empty;

            // Verify initial configured payment links and customers still exist
            const linksRes = await client.listNativePaymentLinks();
            const linksBody = await linksRes.json();
            expect(linksBody.payment_links.some(l => l.id === 'plink_pro_plan')).to.be.true;
            expect(linksBody.payment_links.some(l => l.id === 'plink_starter')).to.be.false;
        });
    });
});

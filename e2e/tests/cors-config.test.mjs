import { expect } from 'chai';
import {
    getSnapshotPort,
    getSnapshotUrl,
    launchSnapshot,
    teardownSnapshot,
    waitAvailable,
} from './utils/index.mjs';

describe('cors-config', () => {
    const port = getSnapshotPort(4);
    const baseUrl = getSnapshotUrl(4);

    before(async () => {
        await launchSnapshot('cors-config', port);
        await waitAvailable(baseUrl);
    });

    after(async () => {
        await teardownSnapshot('cors-config', port);
    });

    describe('CORS headers should be properly configured', () => {
        it('Should allow requests from http://localhost:3000', async () => {
            const res = await fetch(`${baseUrl}/healthz`, {
                headers: { Origin: 'http://localhost:3000' },
            });
            expect(res.status).to.equal(200);
            expect(res.headers.get('Access-Control-Allow-Origin')).to.include('http://localhost:3000');
        });

        it('Should allow requests from http://example.com', async () => {
            const res = await fetch(`${baseUrl}/healthz`, {
                headers: { Origin: 'http://example.com' },
            });
            expect(res.status).to.equal(200);
            expect(res.headers.get('Access-Control-Allow-Origin')).to.include('http://example.com');
        });

        it('Should handle OPTIONS preflight requests from allowed origin', async () => {
            const res = await fetch(`${baseUrl}/payments/intents`, {
                method: 'OPTIONS',
                headers: {
                    Origin: 'http://localhost:3000',
                    'Access-Control-Request-Method': 'POST',
                    'Access-Control-Request-Headers': 'Content-Type, Authorization',
                },
            });
            expect(res.status).to.be.oneOf([200, 204]);
            expect(res.headers.get('Access-Control-Allow-Origin')).to.include('http://localhost:3000');
            expect(res.headers.get('Access-Control-Allow-Methods')).to.include('POST');
        });
    });
});

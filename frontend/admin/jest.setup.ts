import '@testing-library/jest-dom';
import { TextDecoder, TextEncoder } from 'util';

Object.assign(globalThis, { TextDecoder, TextEncoder });

// MSW server setup - only for integration tests
// Unit tests don't need MSW
let server: any;

try {
  const { setupServer } = require('msw/node');
  const { handlers } = require('@/mocks/handlers');
  server = setupServer(...handlers);

  beforeAll(() => server.listen({ onUnhandledRequest: 'warn' }));
  afterEach(() => server.resetHandlers());
  afterAll(() => server.close());
} catch (e) {
  // MSW not available, skip server setup
  console.log('MSW setup skipped for this test run');
}

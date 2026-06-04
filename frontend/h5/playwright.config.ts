import { defineConfig, devices } from '@playwright/test';

const e2ePort = process.env.E2E_PORT || '4173';
const e2eBaseURL = process.env.E2E_BASE_URL || `http://localhost:${e2ePort}`;

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['json', { outputFile: 'test-results.json' }],
    ['list']
  ],
  use: {
    baseURL: e2eBaseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    actionTimeout: 10000,
    navigationTimeout: 30000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },
  ],
  webServer: {
    command: `npm run dev -- --host localhost --port ${e2ePort} --strictPort`,
    url: e2eBaseURL,
    reuseExistingServer: true,
    timeout: 120000,
  },
  timeout: 60000,
  expect: {
    timeout: 10000,
  },
});

import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',

  // One worker, in order. The tests share a database, and the demo path is a
  // sequence rather than a set of independent assertions.
  fullyParallel: false,
  workers: 1,

  reporter: process.env.CI
    ? [['github'], ['html', { outputFolder: 'playwright-report', open: 'never' }]]
    : 'list',
  timeout: 30_000,

  use: {
    baseURL: process.env.BASE_URL ?? 'http://localhost:5173',
    trace: 'retain-on-failure',
  },

  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        // Wider than the xl breakpoint the three column layout needs. The
        // default 1280 sits exactly on it, so the rail would appear or vanish
        // on a one pixel change and the failure would look like anything but a
        // viewport problem.
        viewport: { width: 1440, height: 900 },
      },
    },
  ],
})

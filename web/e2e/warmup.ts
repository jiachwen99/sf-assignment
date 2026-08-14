import { chromium, request } from '@playwright/test'

import { API } from './api'

/*
 * Loads the app once before the suite runs.
 *
 * The first request to a cold dev server triggers a dependency re-optimisation
 * and a full module graph build, which takes long enough that a test racing two
 * browser contexts through it can see one of them arrive late. Both of the
 * failures this suite has had were the first run after a restart, and in CI
 * every run is the first run after a restart.
 *
 * Warming up here removes the class of failure rather than adding a retry to
 * the tests that happened to trip over it.
 */
export default async function warmup() {
  const base = process.env.BASE_URL ?? 'http://localhost:5173'

  // The API first: no point compiling the front end for a stack that is not up.
  const api = await request.newContext()
  await api.get(`${API}/healthz`, { timeout: 60_000 })
  await api.dispose()

  const browser = await chromium.launch()
  const page = await browser.newPage()
  await page.goto(base, { timeout: 60_000 })
  await page.waitForSelector('h1', { timeout: 60_000 })
  await browser.close()
}

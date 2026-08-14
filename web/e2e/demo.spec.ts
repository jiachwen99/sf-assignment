import { expect, test, type Page } from '@playwright/test'

import {
  API,
  addDependency,
  completeTask,
  createTask,
  readTask,
  subscriberCount,
  unique,
  updateTask,
} from './api'

/*
 * The demo path, in the order it gets demonstrated.
 *
 * These run against a seeded database rather than an empty one. A suite that
 * truncates first passes against four rows and proves nothing about the list at
 * twenty thousand, which is the size the brief actually asks about. Each test
 * names its tasks uniquely and reaches them through the URL, which is the same
 * mechanism a shared filtered view uses.
 */

// The list state lives in the URL, so a test can start where it needs to be
// rather than clicking its way there.
async function openList(page: Page, query: string) {
  await page.goto(`/?${query}`)
  await expect(page.getByTestId('todo-row').first()).toBeVisible()
}

async function openTaskNamed(page: Page, name: string) {
  await openList(page, `name=${encodeURIComponent(name)}`)
  await page.getByTestId('todo-row').first().click()
  await expect(page.getByLabel('Task detail')).toBeVisible()
}

test('a task can be created, edited and deleted without leaving the list', async ({ page }) => {
  const name = unique('crud')

  await page.goto('/')
  await page.getByTestId('new-todo').click()
  await page.getByTestId('todo-name').fill(name)
  await page.getByTestId('todo-priority').selectOption('high')
  await page.getByTestId('todo-save').click()

  await openTaskNamed(page, name)
  await expect(page.getByTestId('todo-priority')).toHaveValue('high')

  await page.getByTestId('todo-status').selectOption('in_progress')
  await page.getByTestId('todo-save').click()

  await openTaskNamed(page, name)
  await expect(page.getByTestId('todo-status')).toHaveValue('in_progress')

  await page.getByTestId('todo-delete').click()
  await expect(page.getByTestId('todo-row')).toHaveCount(0)
})

test('a deleted task waits in the trash and comes back whole', async ({ page, request }) => {
  const name = unique('trash')
  const task = await createTask(request, { name, priority: 'high', status: 'in_progress' })

  await openTaskNamed(page, name)
  await page.getByTestId('todo-delete').click()

  await page.getByTestId('toggle-trash').click()
  const row = page.getByText(name)
  await expect(row).toBeVisible()

  await page.getByTestId('restore-button').first().click()
  await expect(page.getByText(name)).toBeHidden()

  // Restored as it was, not as a new task.
  const restored = await readTask(request, task.id)
  expect(restored.status).toBe('in_progress')
  expect(restored.priority).toBe('high')
})

test('completing a repeating task opens the next occurrence and hands over the schedule', async ({
  page,
  request,
}) => {
  const name = unique('rent')
  await createTask(request, {
    name,
    dueDate: '2026-01-31T09:00:00Z',
    status: 'not_started',
    priority: 'high',
    recurUnit: 'month',
    recurInterval: 1,
  })

  await openTaskNamed(page, name)
  await expect(page.getByTestId('todo-repeats')).toHaveValue('month')
  await page.getByTestId('todo-complete').click()

  // Two rows now: the one just finished and the one it opened.
  await openList(page, `name=${encodeURIComponent(name)}`)
  await expect(page.getByTestId('todo-row')).toHaveCount(2)

  // Only the live occurrence carries the schedule. The completed one gave it up
  // in the same transaction, which is what stops the series forking.
  await expect(page.getByTestId('repeats-badge')).toHaveCount(1)

  // And the two ends are linked, in both directions.
  await page.getByTestId('todo-row').first().click()
  await expect(page.getByTestId('task-history')).toContainText('Created by a recurrence')
  await page.getByTestId('history-link').click()
  await expect(page.getByTestId('task-history')).toContainText('Created the next occurrence')
})

test('a blocked task refuses to start, and completing its blocker releases it', async ({
  page,
  request,
}) => {
  const label = unique('chain')
  const blocker = await createTask(request, { name: `${label} collect the data` })
  const waiting = await createTask(request, { name: `${label} write the report` })
  await addDependency(request, waiting.id, blocker.id)

  await openList(page, `name=${encodeURIComponent(label)}&sort=name&dir=asc`)
  await expect(page.getByTestId('blocked-badge')).toHaveCount(1)

  // The blocked task cannot be completed, and the chain says what by.
  await page.getByTestId('todo-row').nth(1).click()
  await expect(page.getByTestId('todo-complete')).toBeDisabled()
  await expect(page.getByTestId('dependency-chain')).toContainText('collect the data')

  // Only completing releases it. Archiving the blocker would not.
  await completeTask(request, blocker)

  await openList(page, `name=${encodeURIComponent(label)}&sort=name&dir=asc`)
  await expect(page.getByTestId('blocked-badge')).toHaveCount(0)
  await page.getByTestId('todo-row').nth(1).click()
  await expect(page.getByTestId('todo-complete')).toBeEnabled()
})

test('a link that would close a loop is refused, and the message names the loop', async ({
  page,
  request,
}) => {
  const label = unique('loop')
  const first = await createTask(request, { name: `${label} alpha` })
  const second = await createTask(request, { name: `${label} beta` })
  await addDependency(request, first.id, second.id)

  await openTaskNamed(page, `${label} beta`)
  await page.getByTestId('dependency-search').fill(`${label} alpha`)
  // Scoped to the picker: the same task is already visible in the chain below,
  // and an unscoped match would be ambiguous.
  await page.getByTestId('dependency-results').getByRole('button', { name: `${label} alpha` }).click()

  await expect(page.getByLabel('Task detail')).toContainText('would create a loop')
  await expect(page.getByLabel('Task detail')).toContainText(`${label} alpha`)
})

test('two people editing the same task cannot silently overwrite each other', async ({
  browser,
  request,
}) => {
  const name = unique('conflict')
  const task = await createTask(request, { name })

  // Two contexts, not two tabs. One context shares state, and the test would
  // pass for the wrong reason.
  const [mine, theirs] = await Promise.all([browser.newContext(), browser.newContext()])
  const [myPage, theirPage] = await Promise.all([mine.newPage(), theirs.newPage()])

  await openTaskNamed(myPage, name)
  await openTaskNamed(theirPage, name)

  // They save first.
  await theirPage.getByTestId('todo-priority').selectOption('high')
  await theirPage.getByTestId('todo-save').click()

  // My copy is now stale, and saying so is the whole point.
  await myPage.getByTestId('todo-priority').selectOption('low')
  await myPage.getByTestId('todo-save').click()
  await expect(myPage.getByTestId('conflict-banner')).toBeVisible()

  // Their write survived mine.
  expect((await readTask(request, task.id)).priority).toBe('high')

  // Loading the current version and saving again goes through.
  await myPage.getByTestId('conflict-reload').click()

  // Wait for the loaded values to arrive before touching them. Loading
  // replaces the whole form, so a selection made in the instant before it
  // commits is discarded by the render that follows, and the save would send
  // what was loaded rather than what was chosen.
  await expect(myPage.getByTestId('todo-priority')).toHaveValue('high')

  await myPage.getByTestId('todo-priority').selectOption('low')
  await myPage.getByTestId('todo-save').click()
  await expect(myPage.getByTestId('conflict-banner')).toBeHidden()
  expect((await readTask(request, task.id)).priority).toBe('low')

  await Promise.all([mine.close(), theirs.close()])
})

test('the list filters, sorts and pages at seeded scale', async ({ page }) => {
  await openList(page, '')

  // The rail's four statuses partition the list, which is the claim a reader
  // checks by eye.
  const total = await countIn(page, 'view-all')
  const statuses = await Promise.all(
    ['view-not_started', 'view-in_progress', 'view-completed', 'view-archived'].map((id) =>
      countIn(page, id),
    ),
  )
  expect(statuses.reduce((a, b) => a + b, 0)).toBe(total)
  expect(total).toBeGreaterThan(1000)

  // Sorting comes from the column header, and the URL follows it.
  await page.getByTestId('sort-name').click()
  await expect(page).toHaveURL(/sort=name&dir=asc/)

  // Paging appends rather than replacing, and nothing is repeated across the
  // boundary, which is what a keyset cursor is for.
  const before = await page.getByTestId('todo-row').count()
  await page.getByTestId('next-page').click()
  await expect(page.getByTestId('todo-row')).toHaveCount(before * 2)

  // By id, not by name. Two different tasks are allowed to share a name, so
  // comparing rendered text would fail on real data for the wrong reason.
  const ids = await page.getByTestId('todo-row').evaluateAll((rows) =>
    rows.map((r) => r.getAttribute('data-id')),
  )
  expect(new Set(ids).size).toBe(ids.length)

  // A view is a query, so it survives a reload as a shareable URL.
  await page.getByTestId('view-blocked').click()
  await expect(page).toHaveURL(/blocked=true/)
  await page.reload()
  await expect(page.getByTestId('view-blocked')).toHaveAttribute('aria-current', 'page')
  await expect(page.getByTestId('filter-blocked')).toHaveValue('true')
})

async function countIn(page: Page, testId: string): Promise<number> {
  const text = await page.getByTestId(testId).innerText()
  return Number(text.replace(/[^\d]/g, ''))
}

test('a change in one browser appears in another without a refresh', async ({
  browser,
  request,
}) => {
  const name = unique('live')

  const [watching, working] = await Promise.all([browser.newContext(), browser.newContext()])
  const [watcher, worker] = await Promise.all([watching.newPage(), working.newPage()])

  // Both looking at the same filtered view, which is empty to begin with.
  const url = `/?name=${encodeURIComponent(name)}`
  await watcher.goto(url)
  await worker.goto(url)
  await expect(watcher.getByTestId('todo-row')).toHaveCount(0)

  // Created over HTTP: what matters is that the change reaches a page nobody
  // touched, not which client made it.
  const task = await createTask(request, { name, priority: 'high' })
  await expect(watcher.getByTestId('todo-row')).toHaveCount(1)

  // And an edit reaches it too, so this is not just the first load arriving
  // late. Asserted on the row rather than on it disappearing: the filter
  // matches anywhere in the name, so a renamed task would still be listed.
  await expect(watcher.getByTestId('todo-row')).toContainText('Not started')
  await updateTask(request, task, { status: 'in_progress' })
  await expect(watcher.getByTestId('todo-row')).toContainText('In progress')

  await Promise.all([watching.close(), working.close()])
})

/*
 * A leaked subscription is invisible until the process runs out of memory,
 * which is far too late to find out, so it is asserted rather than assumed.
 *
 * The stream is opened against the API directly rather than through the page,
 * because the dev server proxies it and does not pass the client's disconnect
 * upstream: closing a browser leaves the proxy holding a live socket, and the
 * API cannot tell the difference. That is a dev-server limitation rather than
 * a property of the hub, and testing through it would assert the proxy's
 * behaviour instead of this application's.
 */
test('a client that goes away releases its subscription', async ({ request }) => {
  const before = await subscriberCount(request)

  const stream = new AbortController()
  const opened = fetch(`${API}/api/events`, { signal: stream.signal })
  await expect
    .poll(() => subscriberCount(request), { message: 'the client should have subscribed' })
    .toBe(before + 1)

  stream.abort()
  await opened.catch(() => {})

  await expect
    .poll(() => subscriberCount(request), { message: 'the subscription should be released' })
    .toBe(before)
})

test('two accounts see the same list, and the history says who did what', async ({
  browser,
  request,
}) => {
  const label = unique('shared')
  const stamp = Date.now()

  const [firstContext, secondContext] = await Promise.all([
    browser.newContext(),
    browser.newContext(),
  ])
  const [priya, marcus] = await Promise.all([firstContext.newPage(), secondContext.newPage()])

  await register(priya, `priya-${stamp}@example.com`, 'Priya')
  await register(marcus, `marcus-${stamp}@example.com`, 'Marcus')
  await expect(priya.getByTestId('current-user')).toHaveText('Priya')
  await expect(marcus.getByTestId('current-user')).toHaveText('Marcus')

  // Priya writes a task. Accounts supply identity, never separation, so it is
  // Marcus's task too: an account that could not see it would contradict the
  // requirement that users share one list.
  await priya.goto(`/?name=${encodeURIComponent(label)}`)
  await priya.getByTestId('new-todo').click()
  await priya.getByTestId('todo-name').fill(`${label} shared task`)
  await priya.getByTestId('todo-save').click()
  await expect(priya.getByTestId('todo-row')).toHaveCount(1)

  await marcus.goto(`/?name=${encodeURIComponent(label)}`)
  await expect(marcus.getByTestId('todo-row')).toHaveCount(1)

  // Marcus edits it, and the history names them both.
  await marcus.getByTestId('todo-row').first().click()
  await marcus.getByTestId('todo-priority').selectOption('high')
  await marcus.getByTestId('todo-save').click()

  await marcus.getByTestId('todo-row').first().click()
  const history = marcus.getByTestId('task-history')
  await expect(history).toContainText('Priya')
  await expect(history).toContainText('Marcus')

  await Promise.all([firstContext.close(), secondContext.close()])
})

// A change made by nobody says so, rather than leaving a blank where a name
// would be.
test('a change made while signed out is recorded as unattributed', async ({ page, request }) => {
  const name = unique('anon')
  await createTask(request, { name })

  await openTaskNamed(page, name)
  await expect(page.getByTestId('task-history')).toContainText('Not signed in')
})

async function register(page: Page, email: string, name: string) {
  await page.goto('/')
  await page.getByTestId('open-login').click()
  await page.getByTestId('tab-register').click()
  await page.getByTestId('register-name').fill(name)
  await page.getByTestId('auth-email').fill(email)
  await page.getByTestId('auth-password').fill('a long enough password')
  await page.getByTestId('register-submit').click()
}

test('a batch applies what it can and names what it could not', async ({ page, request }) => {
  const label = unique('bulk')
  const blocker = await createTask(request, { name: `${label} collect the data` })
  const waiting = await createTask(request, { name: `${label} write the report` })
  const plain = await createTask(request, { name: `${label} unrelated` })
  await addDependency(request, waiting.id, blocker.id)

  // Sorted so the blocked one is selected before its blocker, which is the
  // order that produces a refusal rather than releasing it on the way past.
  await openList(page, `name=${encodeURIComponent(label)}&sort=name&dir=desc`)
  await page.getByTestId('select-page').check()
  await expect(page.getByTestId('bulk-complete')).toBeVisible()

  await page.getByTestId('bulk-complete').click()

  // One refused, and it says which and why.
  const refused = page.getByTestId('bulk-refused')
  await expect(refused).toContainText('1 of 3 could not be done')
  await expect(refused).toContainText(`${label} write the report`)
  await expect(refused).toContainText('waiting on')

  // The other two went through, which is the whole point of per-item results.
  expect((await readTask(request, blocker.id)).status).toBe('completed')
  expect((await readTask(request, plain.id)).status).toBe('completed')
})

test('a batch that fully succeeds clears the selection', async ({ page, request }) => {
  const label = unique('bulk ok')
  await createTask(request, { name: `${label} one` })
  await createTask(request, { name: `${label} two` })

  await openList(page, `name=${encodeURIComponent(label)}`)
  await page.getByTestId('select-page').check()
  await page.getByTestId('bulk-archive').click()

  await expect(page.getByTestId('bulk-archive')).toBeHidden()
  await expect(page.getByTestId('todo-row').first()).toContainText('Archived')
})

// The save closes the panel after the write returns. Holding the response open
// makes that land after the next task is on screen, which is when it used to
// close the wrong one.
test('a save that lands late does not close the task opened after it', async ({
  page,
  request,
}) => {
  const label = unique('late save')
  const first = await createTask(request, { name: `${label} first`, priority: 'low' })
  await createTask(request, { name: `${label} second`, priority: 'low' })

  // Held open until released, so the gap is guaranteed rather than hoped for.
  let release: () => void = () => {}
  const held = new Promise<void>((resolve) => {
    release = resolve
  })
  await page.route(`**/api/todos/${first.id}`, async (route) => {
    if (route.request().method() !== 'PUT') return route.fallback()
    await held
    await route.continue()
  })

  await openList(page, `name=${encodeURIComponent(label)}&sort=name&dir=asc`)
  await page.getByTestId('todo-row').filter({ hasText: 'first' }).click()
  await expect(page.getByTestId('todo-name')).toHaveValue(`${label} first`)

  await page.getByTestId('todo-priority').selectOption('high')
  await page.getByTestId('todo-save').click()

  // The save is still in flight. Open the other task.
  await page.getByTestId('todo-row').filter({ hasText: 'second' }).click()
  await expect(page.getByTestId('todo-name')).toHaveValue(`${label} second`)

  release()

  // The panel belongs to the second task now, so the first task's save must not
  // close it. Asserted after the write has settled rather than immediately.
  await expect(page.getByTestId('todo-row').filter({ hasText: 'first' })).toContainText('High')
  await expect(page.getByLabel('Task detail')).toBeVisible()
  await expect(page.getByTestId('todo-name')).toHaveValue(`${label} second`)
})

import { test as base, expect } from '@playwright/test'

const authMeResponse = {
  id: '00000000-0000-0000-0000-000000000001',
  email: 'test@example.com',
  name: 'Test User',
  avatar_url: '',
}

base('unauthenticated user sees login page', async ({ page }) => {
  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({ status: 401, body: 'unauthorized' }),
  )

  await page.goto('/')

  await expect(page.locator('.login-title')).toHaveText('知る')
  await expect(page.getByText('Sign in to continue')).toBeVisible()
})

base('authenticated user sees app content', async ({ page }) => {
  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({ json: authMeResponse }),
  )
  await page.route('**/api/v1/topics*', (route) =>
    route.fulfill({ json: { topics: ['テスト'] } }),
  )

  await page.goto('/')

  await expect(page.getByRole('link', { name: 'Home' })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Logout' })).toBeVisible()
})

base('logout returns to login page', async ({ page }) => {
  await page.route('**/api/v1/auth/me', (route) =>
    route.fulfill({ json: authMeResponse }),
  )
  await page.route('**/api/v1/topics*', (route) =>
    route.fulfill({ json: { topics: ['テスト'] } }),
  )
  await page.route('**/api/v1/auth/logout', (route) =>
    route.fulfill({ status: 200, body: '{}' }),
  )

  await page.goto('/')
  await expect(page.getByRole('button', { name: 'Logout' })).toBeVisible()

  await page.getByRole('button', { name: 'Logout' }).click()

  await expect(page.locator('.login-title')).toHaveText('知る')
})

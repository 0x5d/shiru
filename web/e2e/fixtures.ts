import { test as base } from '@playwright/test'

const authMeResponse = {
  id: '00000000-0000-0000-0000-000000000001',
  email: 'test@example.com',
  name: 'Test User',
  avatar_url: '',
}

export const test = base.extend({
  page: async ({ page }, use) => {
    await page.route('**/api/v1/auth/me', (route) =>
      route.fulfill({ json: authMeResponse }),
    )
    await use(page)
  },
})

export { expect } from '@playwright/test'

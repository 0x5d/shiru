import { test, expect } from '@playwright/test';

test('homepage displays welcome message', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: '知るへようこそ' })).toBeVisible();
});

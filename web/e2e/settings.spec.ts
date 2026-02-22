import { test, expect } from '@playwright/test';

const defaultSettings = { jlpt_level: 'N3', story_word_target: 150 };
const defaultVocab = {
  entries: [
    {
      id: 'v1',
      surface: '花',
      normalized_surface: '花',
      source: 'manual',
      created_at: '2026-01-01T00:00:00Z',
      updated_at: '2026-01-01T00:00:00Z',
    },
  ],
  total: 1,
};

async function mockDefaults(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/settings', (route) => {
    if (route.request().method() === 'GET') {
      return route.fulfill({ json: defaultSettings });
    }
    if (route.request().method() === 'PUT') {
      return route.fulfill({ json: defaultSettings });
    }
    return route.continue();
  });
  await page.route('**/api/v1/vocab?*', (route) => {
    return route.fulfill({ json: defaultVocab });
  });
  await page.route('**/api/v1/topics/generate', (route) => {
    return route.fulfill({ json: { topics: [] } });
  });
}

test('loads and displays settings', async ({ page }) => {
  await mockDefaults(page);
  await page.goto('/settings');

  await expect(page.getByRole('heading', { name: 'Settings' })).toBeVisible();
  const slider = page.getByRole('slider');
  await expect(slider).toHaveValue('2');
  await expect(page.getByLabel(/story word target/i)).toHaveValue('150');
});

test('displays vocab entries', async ({ page }) => {
  await mockDefaults(page);
  await page.goto('/settings');

  await expect(page.getByText('花 — manual')).toBeVisible();
});

test('saves settings', async ({ page }) => {
  await mockDefaults(page);

  let putBody: unknown;
  await page.route('**/api/v1/settings', (route) => {
    if (route.request().method() === 'PUT') {
      putBody = route.request().postDataJSON();
      return route.fulfill({
        json: { jlpt_level: 'N3', story_word_target: 200 },
      });
    }
    return route.fulfill({ json: defaultSettings });
  });

  await page.goto('/settings');

  const input = page.getByLabel(/story word target/i);
  await input.fill('200');
  await page.getByRole('button', { name: 'Save Settings' }).click();

  expect(putBody).toEqual(
    expect.objectContaining({ jlpt_level: 'N3', story_word_target: 200 }),
  );
});

test('adds vocab words', async ({ page }) => {
  await mockDefaults(page);

  await page.route('**/api/v1/vocab', (route) => {
    if (route.request().method() === 'POST') {
      return route.fulfill({
        json: {
          entries: [
            {
              id: 'v2',
              surface: '走る',
              normalized_surface: '走る',
              source: 'manual',
              created_at: '2026-01-01T00:00:00Z',
              updated_at: '2026-01-01T00:00:00Z',
            },
            {
              id: 'v3',
              surface: '飲む',
              normalized_surface: '飲む',
              source: 'manual',
              created_at: '2026-01-01T00:00:00Z',
              updated_at: '2026-01-01T00:00:00Z',
            },
          ],
          total: 2,
        },
      });
    }
    return route.continue();
  });

  await page.goto('/settings');

  await page.getByRole('textbox', { name: /add words/i }).fill('走る\n飲む');
  await page.getByRole('button', { name: 'Add Words' }).click();

  await expect(page.getByText('走る — manual')).toBeVisible();
  await expect(page.getByText('飲む — manual')).toBeVisible();
});

test('imports WaniKani vocab', async ({ page }) => {
  await mockDefaults(page);

  await page.route('**/api/v1/vocab/import/wanikani', (route) => {
    return route.fulfill({ json: { imported_count: 5 } });
  });

  await page.goto('/settings');

  await page.getByRole('button', { name: /Import\/Sync WaniKani/i }).click();

  await expect(page.getByText('Imported 5 items')).toBeVisible();
});

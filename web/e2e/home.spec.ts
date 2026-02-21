import { test, expect } from '@playwright/test';

const topicsResponse = { topics: ['夏祭り', '学校生活', '日本料理'] };

const storyResponse = {
  id: 'abc-123',
  topic: '夏祭り',
  title: '夏の物語',
  tone: 'funny',
  jlpt_level: 'N5',
  target_word_count: 100,
  actual_word_count: 95,
  content: '夏祭りの話',
  used_vocab_count: 10,
  source_tag_names: ['nature'],
  created_at: '2026-01-01T00:00:00Z',
};

test('shows nav links', async ({ page }) => {
  await page.route('**/api/v1/topics/generate', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(topicsResponse) }),
  );

  await page.goto('/');

  await expect(page.getByRole('link', { name: 'Home' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Stories' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'Settings' })).toBeVisible();
});

test('loads and displays topics', async ({ page }) => {
  await page.route('**/api/v1/topics/generate', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(topicsResponse) }),
  );

  await page.goto('/');

  const cards = page.getByTestId('topic-card');
  await expect(cards).toHaveCount(3);
  await expect(cards.nth(0)).toContainText('夏祭り');
  await expect(cards.nth(1)).toContainText('学校生活');
  await expect(cards.nth(2)).toContainText('日本料理');
});

test('regenerate topics button fetches new topics', async ({ page }) => {
  let firstLoad = true;
  const secondTopics = { topics: ['桜', '温泉', '東京タワー'] };

  await page.route('**/api/v1/topics/generate', (route) => {
    const body = firstLoad ? topicsResponse : secondTopics;
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(body) });
  });

  await page.goto('/');
  await expect(page.getByTestId('topic-card')).toHaveCount(3);

  firstLoad = false;
  await page.getByRole('button', { name: /regenerate/i }).click();

  await expect(page.getByTestId('topic-card').nth(0)).toContainText('桜');
  await expect(page.getByTestId('topic-card').nth(1)).toContainText('温泉');
  await expect(page.getByTestId('topic-card').nth(2)).toContainText('東京タワー');
});

test('clicking topic creates story and navigates to reader', async ({ page }) => {
  await page.route('**/api/v1/topics/generate', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(topicsResponse) }),
  );

  await page.route('**/api/v1/stories', (route) => {
    if (route.request().method() === 'POST') {
      return route.fulfill({ status: 201, contentType: 'application/json', body: JSON.stringify(storyResponse) });
    }
    return route.continue();
  });

  await page.route('**/api/v1/stories/abc-123', (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(storyResponse) }),
  );

  await page.route('**/api/v1/stories/abc-123/tokens', (route) =>
    route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({ story_id: 'abc-123', tokens: [] }),
    }),
  );

  await page.goto('/');
  await expect(page.getByTestId('topic-card')).toHaveCount(3);

  await page.getByTestId('topic-card').nth(0).click();

  await page.waitForURL('**/stories/abc-123');
});

test('shows error when topics fail to load', async ({ page }) => {
  await page.route('**/api/v1/topics/generate', (route) =>
    route.fulfill({ status: 500, contentType: 'application/json', body: JSON.stringify({ error: 'Internal Server Error' }) }),
  );

  await page.goto('/');

  await expect(page.getByText(/error/i)).toBeVisible();
});

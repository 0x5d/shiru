import { test, expect } from './fixtures';

const storiesList = {
  stories: [
    {
      id: 's1',
      topic: '夏祭り',
      title: '夏の冒険',
      tone: 'funny',
      jlpt_level: 'N4',
      target_word_count: 100,
      actual_word_count: 95,
      content: '楽しい夏祭りの話',
      used_vocab_count: 10,
      source_tag_names: ['nature'],
      created_at: '2026-02-15T00:00:00Z',
    },
    {
      id: 's2',
      topic: '学校',
      title: '学校の秘密',
      tone: 'shocking',
      jlpt_level: 'N3',
      target_word_count: 100,
      actual_word_count: 88,
      content: '学校の不思議な話',
      used_vocab_count: 8,
      source_tag_names: ['school'],
      created_at: '2026-02-14T00:00:00Z',
    },
  ],
};

const searchResults = {
  results: [
    {
      story_id: 's1',
      topic: '夏祭り',
      tone: 'funny',
      content: '楽しい夏祭りの話',
      jlpt_level: 'N4',
      created_at: '2026-02-15T00:00:00Z',
    },
  ],
  total: 1,
};

async function mockDefaults(page: import('@playwright/test').Page) {
  await page.route('**/api/v1/topics/generate', (route) =>
    route.fulfill({ json: { topics: [] } }),
  );
}

test('loads and displays stories', async ({ page }) => {
  await mockDefaults(page);
  await page.route('**/api/v1/stories?limit=20&offset=0', (route) =>
    route.fulfill({ json: storiesList }),
  );

  await page.goto('/stories');

  await expect(page.getByText('夏の冒険')).toBeVisible();
  await expect(page.getByText('学校の秘密')).toBeVisible();
});

test('shows empty state', async ({ page }) => {
  await mockDefaults(page);
  await page.route('**/api/v1/stories?limit=20&offset=0', (route) =>
    route.fulfill({ json: { stories: [] } }),
  );

  await page.goto('/stories');

  await expect(page.getByText('No stories found.')).toBeVisible();
});

test('search filters stories', async ({ page }) => {
  await mockDefaults(page);
  await page.route('**/api/v1/stories?limit=20&offset=0', (route) =>
    route.fulfill({ json: storiesList }),
  );
  await page.route('**/api/v1/stories/search?q=*', (route) =>
    route.fulfill({ json: searchResults }),
  );

  await page.goto('/stories');
  await expect(page.getByText('夏の冒険')).toBeVisible();
  await expect(page.getByText('学校の秘密')).toBeVisible();

  const searchInput = page.getByRole('textbox');
  await searchInput.fill('夏');

  const searchResponse = page.waitForResponse('**/api/v1/stories/search?q=*');
  await searchResponse;

  await expect(page.getByText('夏祭り')).toBeVisible();
  await expect(page.getByText('学校の秘密')).not.toBeVisible();
});

test('clicking story navigates to reader', async ({ page }) => {
  await mockDefaults(page);
  await page.route('**/api/v1/stories?limit=20&offset=0', (route) =>
    route.fulfill({ json: storiesList }),
  );
  await page.route('**/api/v1/stories/s1', (route) =>
    route.fulfill({ json: storiesList.stories[0] }),
  );
  await page.route('**/api/v1/stories/s1/tokens', (route) =>
    route.fulfill({ json: { tokens: [] } }),
  );

  await page.goto('/stories');
  await expect(page.getByText('夏の冒険')).toBeVisible();

  await page.getByText('夏の冒険').click();

  await expect(page).toHaveURL(/\/stories\/s1/);
});

test('clearing search returns to full list', async ({ page }) => {
  await mockDefaults(page);
  await page.route('**/api/v1/stories?limit=20&offset=0', (route) =>
    route.fulfill({ json: storiesList }),
  );
  await page.route('**/api/v1/stories/search?q=*', (route) =>
    route.fulfill({ json: searchResults }),
  );

  await page.goto('/stories');
  await expect(page.getByText('夏の冒険')).toBeVisible();
  await expect(page.getByText('学校の秘密')).toBeVisible();

  const searchInput = page.getByRole('textbox');
  await searchInput.fill('夏');
  await page.waitForResponse('**/api/v1/stories/search?q=*');

  await expect(page.getByText('学校の秘密')).not.toBeVisible();

  await searchInput.clear();

  await expect(page.getByText('夏の冒険')).toBeVisible();
  await expect(page.getByText('学校の秘密')).toBeVisible();
});

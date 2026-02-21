import { test, expect } from '@playwright/test';

const STORY_ID = 'test-story-1';

const storyResponse = {
  id: STORY_ID,
  topic: '夏祭り',
  title: '花火の夜',
  tone: 'funny',
  jlpt_level: 'N4',
  target_word_count: 100,
  actual_word_count: 3,
  content: '花がきれい',
  used_vocab_count: 1,
  source_tag_names: ['nature'],
  created_at: '2026-01-01T00:00:00Z',
};

const tokensResponse = {
  story_id: STORY_ID,
  tokens: [
    { surface: '花', start_offset: 0, end_offset: 1, vocab_entry_id: 'vocab-1', is_vocab_match: true },
    { surface: 'が', start_offset: 1, end_offset: 2, is_vocab_match: false },
    { surface: 'きれい', start_offset: 2, end_offset: 5, is_vocab_match: false },
  ],
};

const vocabDetailsResponse = {
  id: 'vocab-1',
  surface: '花',
  meaning: 'flower',
  reading: 'はな',
};

test.beforeEach(async ({ page }) => {
  await page.route(`**/api/v1/stories/${STORY_ID}`, (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(storyResponse) }),
  );
  await page.route(`**/api/v1/stories/${STORY_ID}/tokens`, (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(tokensResponse) }),
  );
  await page.route(`**/api/v1/vocab/vocab-1/details`, (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(vocabDetailsResponse) }),
  );
  await page.route(`**/api/v1/stories/${STORY_ID}/audio`, (route) =>
    route.fulfill({ status: 200, contentType: 'audio/mpeg', body: Buffer.from('fake-audio') }),
  );

  await page.goto(`/stories/${STORY_ID}`);
});

test('displays story title and metadata', async ({ page }) => {
  await expect(page.getByRole('heading', { name: '花火の夜' })).toBeVisible();
  await expect(page.getByText('夏祭り')).toBeVisible();
  await expect(page.getByText('funny')).toBeVisible();
  await expect(page.getByText('N4')).toBeVisible();
});

test('renders tokenized text with vocab highlighting', async ({ page }) => {
  const vocabToken = page.locator('.vocab-match').first();
  await expect(vocabToken).toHaveText('花');

  const gaToken = page.getByText('が');
  await expect(gaToken).not.toHaveClass(/vocab-match/);

  const kireiToken = page.getByText('きれい');
  await expect(kireiToken).not.toHaveClass(/vocab-match/);
});

test('clicking vocab token shows furigana', async ({ page }) => {
  const vocabToken = page.locator('.vocab-match').first();
  await vocabToken.click();

  const rt = page.locator('rt');
  await expect(rt).toHaveText('はな');
});

test('clicking furigana token again hides it', async ({ page }) => {
  const vocabToken = page.locator('.vocab-match').first();
  await vocabToken.click();
  await expect(page.locator('rt')).toHaveText('はな');

  await vocabToken.click();
  await expect(page.locator('rt')).toHaveCount(0);
});

test('long press shows meaning tooltip', async ({ page }) => {
  const token = page.locator('.vocab-match').first();
  await token.dispatchEvent('mousedown');
  await page.waitForTimeout(600);

  await expect(page.getByText('flower')).toBeVisible();

  await token.dispatchEvent('mouseup');
  await expect(page.getByText('flower')).not.toBeVisible();
});

test('shows back link', async ({ page }) => {
  const backLink = page.getByRole('link', { name: '← Back' });
  await expect(backLink).toBeVisible();
  await expect(backLink).toHaveAttribute('href', '/');
});

test('play button exists', async ({ page }) => {
  await expect(page.getByRole('button', { name: 'Play' })).toBeVisible();
});

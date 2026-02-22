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
    { surface: '花', reading: 'はな', start_offset: 0, end_offset: 1, vocab_entry_id: 'vocab-1', is_vocab_match: true },
    { surface: 'が', reading: 'が', start_offset: 1, end_offset: 2, is_vocab_match: false },
    { surface: 'きれい', reading: 'きれい', start_offset: 2, end_offset: 5, is_vocab_match: false },
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

test('second click on kanji token shows meaning tooltip above the word', async ({ page }) => {
  const vocabToken = page.locator('.vocab-match').first();

  // First click shows furigana
  await vocabToken.click();
  await expect(page.locator('rt')).toHaveText('はな');

  // Second click shows meaning tooltip
  await vocabToken.click();
  const tooltip = page.locator('.tooltip');
  await expect(tooltip).toBeVisible();
  await expect(tooltip).toHaveText('flower');

  // Tooltip should be positioned above the token, not at the top of the page
  const tokenBox = await vocabToken.boundingBox();
  const tooltipBox = await tooltip.boundingBox();
  expect(tokenBox).not.toBeNull();
  expect(tooltipBox).not.toBeNull();
  // Tooltip bottom should be near the token top (within 20px)
  expect(tooltipBox!.y + tooltipBox!.height).toBeGreaterThan(tokenBox!.y - 20);
  expect(tooltipBox!.y + tooltipBox!.height).toBeLessThanOrEqual(tokenBox!.y + 5);

  // Third click hides tooltip (force: tooltip overlay affects hit-test)
  await vocabToken.click({ force: true });
  await expect(tooltip).not.toBeVisible();
});

test('clicking non-kanji vocab token toggles meaning tooltip', async ({ page }) => {
  // Add a non-kanji vocab token to test
  const tokensWithNonKanji = {
    story_id: STORY_ID,
    tokens: [
      { surface: '花', reading: 'はな', start_offset: 0, end_offset: 1, vocab_entry_id: 'vocab-1', is_vocab_match: true },
      { surface: 'が', reading: 'が', start_offset: 1, end_offset: 2, is_vocab_match: false },
      { surface: 'きれい', reading: 'きれい', start_offset: 2, end_offset: 5, vocab_entry_id: 'vocab-2', is_vocab_match: true },
    ],
  };

  await page.route(`**/api/v1/stories/${STORY_ID}/tokens`, (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify(tokensWithNonKanji) }),
  );
  await page.route(`**/api/v1/vocab/vocab-2/details`, (route) =>
    route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ id: 'vocab-2', surface: 'きれい', meaning: 'pretty; beautiful', reading: 'きれい' }) }),
  );

  await page.goto(`/stories/${STORY_ID}`);

  // きれい has reading === surface, so first click should show tooltip directly
  const kireiToken = page.locator('.vocab-match').nth(1);
  await kireiToken.click();
  const tooltip = page.locator('.tooltip');
  await expect(tooltip).toHaveText('pretty; beautiful');

  // Click again hides it (force: tooltip overlay affects hit-test)
  await kireiToken.click({ force: true });
  await expect(tooltip).not.toBeVisible();
});

test('clicking non-vocab word looks up meaning via dictionary', async ({ page }) => {
  await page.route('**/api/v1/dictionary/lookup?*', (route) => {
    const url = new URL(route.request().url());
    const word = url.searchParams.get('word');
    if (word === 'きれい') {
      route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ meaning: 'pretty; clean', reading: 'きれい' }) });
    } else {
      route.fulfill({ status: 404, body: 'not found' });
    }
  });

  const kireiToken = page.getByText('きれい');
  await kireiToken.click();
  const tooltip = page.locator('.tooltip');
  await expect(tooltip).toBeVisible();
  await expect(tooltip).toHaveText('pretty; clean');
});

test('shows back link', async ({ page }) => {
  const backLink = page.getByRole('link', { name: '← Back' });
  await expect(backLink).toBeVisible();
  await expect(backLink).toHaveAttribute('href', '/');
});

test('play button exists', async ({ page }) => {
  await expect(page.getByRole('button', { name: 'Play' })).toBeVisible();
});

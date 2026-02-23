const BASE = '/api/v1'

// ── Auth ────────────────────────────────────────────────────────────────────

export type AuthUser = {
  id: string
  email: string
  name: string
  avatar_url: string
}

export async function loginWithGoogle(credential: string): Promise<AuthUser> {
  const res = await fetch(`${BASE}/auth/google`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ credential }),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function getMe(): Promise<AuthUser> {
  const res = await fetch(`${BASE}/auth/me`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function logout(): Promise<void> {
  const res = await fetch(`${BASE}/auth/logout`, {
    method: 'POST',
    credentials: 'include',
  })
  if (!res.ok) throw new Error(await res.text())
}

// ── Settings ────────────────────────────────────────────────────────────────

export type Settings = {
  jlpt_level: string
  story_word_target: number
  wanikani_api_key?: string
  wanikani_last_synced_at?: string
}

export type UpdateSettingsRequest = {
  jlpt_level: string
  story_word_target: number
  wanikani_api_key?: string
}

export async function getSettings(): Promise<Settings> {
  const res = await fetch(`${BASE}/settings`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function updateSettings(req: UpdateSettingsRequest): Promise<Settings> {
  const res = await fetch(`${BASE}/settings`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify(req),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

// ── Vocab ───────────────────────────────────────────────────────────────────

export type VocabEntry = {
  id: string
  surface: string
  normalized_surface: string
  meaning?: string
  reading?: string
  source: string
  created_at: string
  updated_at: string
}

export type ListVocabResponse = {
  entries: VocabEntry[]
  total: number
}

export type CreateVocabResponse = {
  entries: VocabEntry[]
}

export type VocabDetails = {
  id: string
  surface: string
  meaning: string
  reading: string
}

export type ImportWaniKaniResponse = {
  imported_count: number
}

export async function listVocab(query = '', limit = 20, offset = 0): Promise<ListVocabResponse> {
  const params = new URLSearchParams({ query, limit: String(limit), offset: String(offset) })
  const res = await fetch(`${BASE}/vocab?${params}`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function createVocab(entries: string[]): Promise<CreateVocabResponse> {
  const res = await fetch(`${BASE}/vocab`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ entries }),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function getVocabDetails(vocabID: string): Promise<VocabDetails> {
  const res = await fetch(`${BASE}/vocab/${vocabID}/details`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function importWaniKani(): Promise<ImportWaniKaniResponse> {
  const res = await fetch(`${BASE}/vocab/import/wanikani`, {
    method: 'POST',
    credentials: 'include',
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export type VocabStatus = {
  total_vocab: number
  tagged_vocab_count: number
  tagging_in_progress: boolean
}

export async function deleteAllVocab(): Promise<void> {
  const res = await fetch(`${BASE}/vocab`, {
    method: 'DELETE',
    credentials: 'include',
  })
  if (!res.ok) throw new Error(await res.text())
}

export async function getVocabStatus(): Promise<VocabStatus> {
  const res = await fetch(`${BASE}/vocab/status`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

// ── Dictionary ──────────────────────────────────────────────────────────────

export type LookupWordResponse = {
  meaning: string
  reading: string
}

export async function lookupWord(word: string): Promise<LookupWordResponse> {
  const params = new URLSearchParams({ word })
  const res = await fetch(`${BASE}/dictionary/lookup?${params}`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

// ── Topics ──────────────────────────────────────────────────────────────────

export type GenerateTopicsResponse = {
  topics: string[]
}

export async function getTopics(force = false): Promise<GenerateTopicsResponse> {
  const params = force ? '?force=true' : ''
  const res = await fetch(`${BASE}/topics${params}`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

// ── Stories ─────────────────────────────────────────────────────────────────

export type Story = {
  id: string
  topic: string
  title: string
  tone: string
  jlpt_level: string
  target_word_count: number
  actual_word_count: number
  content: string
  used_vocab_count: number
  source_tag_names: string[]
  created_at: string
}

export type ListStoriesResponse = {
  stories: Story[]
}

export type SearchResult = {
  story_id: string
  topic: string
  tone: string
  content: string
  jlpt_level: string
  created_at: string
}

export type SearchStoriesResponse = {
  results: SearchResult[]
  total: number
}

export type Token = {
  surface: string
  reading?: string
  start_offset: number
  end_offset: number
  vocab_entry_id?: string
  is_vocab_match: boolean
  is_lookupable: boolean
}

export type StoryTokensResponse = {
  story_id: string
  tokens: Token[]
}

export async function createStory(topic: string): Promise<Story> {
  const res = await fetch(`${BASE}/stories`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify({ topic }),
  })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function listStories(limit = 20, offset = 0): Promise<ListStoriesResponse> {
  const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
  const res = await fetch(`${BASE}/stories?${params}`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function getStory(storyID: string): Promise<Story> {
  const res = await fetch(`${BASE}/stories/${storyID}`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function searchStories(q: string, limit = 20, offset = 0): Promise<SearchStoriesResponse> {
  const params = new URLSearchParams({ q, limit: String(limit), offset: String(offset) })
  const res = await fetch(`${BASE}/stories/search?${params}`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function getStoryTokens(storyID: string): Promise<StoryTokensResponse> {
  const res = await fetch(`${BASE}/stories/${storyID}/tokens`, { credentials: 'include' })
  if (!res.ok) throw new Error(await res.text())
  return res.json()
}

export async function createStoryAudio(storyID: string): Promise<Blob> {
  const res = await fetch(`${BASE}/stories/${storyID}/audio`, {
    method: 'POST',
    credentials: 'include',
  })
  if (!res.ok) throw new Error(await res.text())
  return res.blob()
}

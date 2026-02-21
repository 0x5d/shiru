import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { listStories, searchStories } from '../api'
import type { Story, SearchResult } from '../api'

type StoryItem = {
  id: string
  topic: string
  title?: string
  tone: string
  jlpt_level: string
  created_at: string
}

function toStoryItems(stories: Story[]): StoryItem[] {
  return stories.map((s) => ({
    id: s.id,
    topic: s.topic,
    title: s.title,
    tone: s.tone,
    jlpt_level: s.jlpt_level,
    created_at: s.created_at,
  }))
}

function fromSearchResults(results: SearchResult[]): StoryItem[] {
  return results.map((r) => ({
    id: r.story_id,
    topic: r.topic,
    tone: r.tone,
    jlpt_level: r.jlpt_level,
    created_at: r.created_at,
  }))
}

function StoriesPage() {
  const [items, setItems] = useState<StoryItem[]>([])
  const [loading, setLoading] = useState(true)
  const [query, setQuery] = useState('')
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const navigate = useNavigate()

  useEffect(() => {
    setLoading(true)
    listStories()
      .then((res) => setItems(toStoryItems(res.stories)))
      .finally(() => setLoading(false))
  }, [])

  function handleSearch(value: string) {
    setQuery(value)

    if (debounceRef.current) {
      clearTimeout(debounceRef.current)
    }

    debounceRef.current = setTimeout(() => {
      setLoading(true)
      const request = value.trim()
        ? searchStories(value.trim()).then((res) => setItems(fromSearchResults(res.results)))
        : listStories().then((res) => setItems(toStoryItems(res.stories)))

      request.finally(() => setLoading(false))
    }, 300)
  }

  return (
    <div className="container">
      <h1 className="page-title">Stories</h1>
      <input
        className="search-input"
        type="text"
        placeholder="Search stories…"
        value={query}
        onChange={(e) => handleSearch(e.target.value)}
      />
      {loading ? (
        <div className="loading">Loading…</div>
      ) : items.length === 0 ? (
        <p>No stories found.</p>
      ) : (
        <ul className="story-list">
          {items.map((item) => (
            <li key={item.id} onClick={() => navigate(`/stories/${item.id}`)}>
              <strong>{item.title || item.topic}</strong>
              <span className="tone-badge">{item.tone}</span>
              <span>{item.jlpt_level}</span>
              <span>{new Date(item.created_at).toLocaleDateString()}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}

export default StoriesPage

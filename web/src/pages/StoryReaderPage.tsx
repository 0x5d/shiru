import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import {
  getStory,
  getStoryTokens,
  getVocabDetails,
  createStoryAudio,
} from '../api'
import type { Story, Token, VocabDetails } from '../api'

export default function StoryReaderPage() {
  const { storyID } = useParams<{ storyID: string }>()
  const [story, setStory] = useState<Story | null>(null)
  const [tokens, setTokens] = useState<Token[]>([])
  const [loading, setLoading] = useState(true)
  const [showFurigana, setShowFurigana] = useState<Record<number, boolean>>({})
  const [vocabCache, setVocabCache] = useState<Record<string, VocabDetails>>({})
  const [tooltip, setTooltip] = useState<{ index: number; meaning: string } | null>(null)
  const [playing, setPlaying] = useState(false)
  const longPressTimer = useRef<ReturnType<typeof setTimeout> | null>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)

  useEffect(() => {
    if (!storyID) return
    let cancelled = false
    setLoading(true)

    Promise.all([getStory(storyID), getStoryTokens(storyID)])
      .then(([s, t]) => {
        if (cancelled) return
        setStory(s)
        setTokens(t.tokens)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => { cancelled = true }
  }, [storyID])

  const fetchVocabDetails = useCallback(
    async (vocabEntryId: string): Promise<VocabDetails> => {
      if (vocabCache[vocabEntryId]) return vocabCache[vocabEntryId]
      const details = await getVocabDetails(vocabEntryId)
      setVocabCache((prev) => ({ ...prev, [vocabEntryId]: details }))
      return details
    },
    [vocabCache],
  )

  const handleTokenClick = useCallback(
    (index: number, token: Token) => {
      if (!token.reading || token.reading === token.surface) return

      if (showFurigana[index]) {
        setShowFurigana((prev) => {
          const next = { ...prev }
          delete next[index]
          return next
        })
        return
      }

      setShowFurigana((prev) => ({ ...prev, [index]: true }))
    },
    [showFurigana],
  )

  const handleMouseDown = useCallback(
    (index: number, token: Token) => {
      if (!token.is_vocab_match || !token.vocab_entry_id) return

      longPressTimer.current = setTimeout(async () => {
        const details = await fetchVocabDetails(token.vocab_entry_id!)
        setTooltip({ index, meaning: details.meaning })
      }, 500)
    },
    [fetchVocabDetails],
  )

  const handleMouseUpOrLeave = useCallback(() => {
    if (longPressTimer.current) {
      clearTimeout(longPressTimer.current)
      longPressTimer.current = null
    }
    setTooltip(null)
  }, [])

  const handlePlay = useCallback(async () => {
    if (!storyID || playing) return
    setPlaying(true)
    try {
      const blob = await createStoryAudio(storyID)
      const url = URL.createObjectURL(blob)
      const audio = new Audio(url)
      audioRef.current = audio
      audio.onended = () => {
        setPlaying(false)
        URL.revokeObjectURL(url)
      }
      audio.play()
    } catch {
      setPlaying(false)
    }
  }, [storyID, playing])

  if (loading) return <div className="loading">Loading…</div>
  if (!story) return <div className="container">Story not found.</div>

  return (
    <div className="container story-reader">
      <Link to="/">← Back</Link>
      <h2>{story.title}</h2>
      <p>
        Topic: {story.topic} · Tone: {story.tone} · JLPT: {story.jlpt_level}
      </p>
      <button className="btn btn-primary" onClick={handlePlay} disabled={playing}>
        {playing ? 'Playing...' : 'Play'}
      </button>
      <div>
        {tokens.map((token, i) => {
          const isVocab = token.is_vocab_match
          const hasFurigana = showFurigana[i]
          const reading = token.reading
          const hasReading = !!reading && reading !== token.surface
          const className = `token${isVocab ? ' vocab-match' : ''}${hasReading ? ' has-reading' : ''}`

          return hasFurigana && reading ? (
            <ruby
              key={i}
              className={className}
              onClick={() => handleTokenClick(i, token)}
              onMouseDown={() => handleMouseDown(i, token)}
              onMouseUp={handleMouseUpOrLeave}
              onMouseLeave={handleMouseUpOrLeave}
            >
              {tooltip?.index === i && (
                <span className="tooltip">{tooltip.meaning}</span>
              )}
              {token.surface}
              <rt>{reading}</rt>
            </ruby>
          ) : (
            <span
              key={i}
              className={className}
              onClick={() => handleTokenClick(i, token)}
              onMouseDown={() => handleMouseDown(i, token)}
              onMouseUp={handleMouseUpOrLeave}
              onMouseLeave={handleMouseUpOrLeave}
            >
              {tooltip?.index === i && (
                <span className="tooltip">{tooltip.meaning}</span>
              )}
              {token.surface}
            </span>
          )
        })}
      </div>
    </div>
  )
}

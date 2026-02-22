import { useState, useEffect, useRef, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import {
  getStory,
  getStoryTokens,
  getVocabDetails,
  lookupWord,
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
  const [meaningCache, setMeaningCache] = useState<Record<string, string>>({})
  const [tooltip, setTooltip] = useState<{ index: number; meaning: string } | null>(null)
  const [playing, setPlaying] = useState(false)
  const audioRef = useRef<HTMLAudioElement | null>(null)

  useEffect(() => {
    if (!storyID) return
    let cancelled = false

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

  const fetchMeaning = useCallback(
    async (token: Token): Promise<string> => {
      if (token.is_vocab_match && token.vocab_entry_id) {
        const details = await fetchVocabDetails(token.vocab_entry_id)
        return details.meaning
      }
      const surface = token.surface
      if (meaningCache[surface]) return meaningCache[surface]
      const result = await lookupWord(surface)
      setMeaningCache((prev) => ({ ...prev, [surface]: result.meaning }))
      return result.meaning
    },
    [fetchVocabDetails, meaningCache],
  )

  const handleTokenClick = useCallback(
    async (index: number, token: Token) => {
      const hasReading = !!token.reading && token.reading !== token.surface

      if (hasReading && !showFurigana[index]) {
        setShowFurigana((prev) => ({ ...prev, [index]: true }))
        return
      }

      if (!token.is_lookupable) return

      if (tooltip?.index === index) {
        setTooltip(null)
        return
      }

      try {
        const meaning = await fetchMeaning(token)
        setTooltip({ index, meaning })
      } catch {
        // No dictionary result available
      }
    },
    [showFurigana, tooltip, fetchMeaning],
  )

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
          const isInteractive = hasReading || token.is_lookupable
          const className = `token${isVocab ? ' vocab-match' : ''}${isInteractive ? ' has-reading' : ''}`

          return (
            <span
              key={i}
              className={className}
              onClick={() => handleTokenClick(i, token)}
            >
              {tooltip?.index === i && (
                <span className="tooltip">{tooltip.meaning}</span>
              )}
              {hasFurigana && reading ? (
                <ruby>{token.surface}<rt>{reading}</rt></ruby>
              ) : (
                token.surface
              )}
            </span>
          )
        })}
      </div>
    </div>
  )
}

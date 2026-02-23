import { useState, useEffect } from 'react'
import { useNavigate, Link } from 'react-router-dom'
import { getTopics, createStory, getVocabStatus, type VocabStatus } from '../api'

function HomePage() {
  const navigate = useNavigate()
  const [topics, setTopics] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [vocabStatus, setVocabStatus] = useState<VocabStatus | null>(null)

  const fetchTopics = async (force = false) => {
    setLoading(true)
    setError('')
    try {
      const res = await getTopics(force)
      setTopics(res.topics.slice(0, 3))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load topics')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    getVocabStatus().then(setVocabStatus)
    fetchTopics()
  }, [])

  const handleTopicClick = async (topic: string) => {
    setLoading(true)
    setError('')
    try {
      const story = await createStory(topic)
      navigate(`/stories/${story.id}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create story')
      setLoading(false)
    }
  }

  if (loading) return <div className="container"><div className="loading" /></div>

  const hasNoVocab = vocabStatus && vocabStatus.total_vocab === 0
  const hasNoTags = vocabStatus && vocabStatus.total_vocab > 0 && vocabStatus.tagged_vocab_count === 0
  const isTagging = vocabStatus?.tagging_in_progress

  return (
    <div className="container">
      <h1 className="page-title">Choose a Topic</h1>
      {error && <p style={{ color: 'red' }}>{error}</p>}

      {hasNoVocab && (
        <div className="notice notice-info">
          <p>
            You haven't added any words yet. Head to{' '}
            <Link to="/settings">Settings</Link> to import from WaniKani or add
            words manually.
          </p>
        </div>
      )}

      {hasNoTags && isTagging && (
        <div className="notice notice-info">
          <p>
            Your words are being processed — this may take a moment. Refresh the
            page shortly to start generating stories.
          </p>
        </div>
      )}

      {hasNoTags && !isTagging && (
        <div className="notice notice-info">
          <p>
            Your words haven't been tagged yet. Head to{' '}
            <Link to="/settings">Settings</Link> to import from WaniKani or add
            words manually.
          </p>
        </div>
      )}

      {vocabStatus && vocabStatus.tagged_vocab_count > 0 && isTagging && (
        <div className="notice notice-warning">
          <p>
            New words are being imported. Stories generated now may not include
            your latest vocabulary.
          </p>
        </div>
      )}

      {!hasNoVocab && !hasNoTags && (
        <>
          <div className="topic-grid">
            {topics.map((topic) => (
              <div key={topic} className="topic-card" data-testid="topic-card" onClick={() => handleTopicClick(topic)}>
                {topic}
              </div>
            ))}
          </div>
          <button className="btn btn-secondary" onClick={() => fetchTopics(true)}>
            Regenerate Topics
          </button>
        </>
      )}
    </div>
  )
}

export default HomePage

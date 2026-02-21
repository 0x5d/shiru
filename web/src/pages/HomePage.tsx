import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { generateTopics, createStory } from '../api'

function HomePage() {
  const navigate = useNavigate()
  const [topics, setTopics] = useState<string[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const fetchTopics = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await generateTopics()
      setTopics(res.topics.slice(0, 3))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load topics')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
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

  return (
    <div className="container">
      <h1 className="page-title">Choose a Topic</h1>
      {error && <p style={{ color: 'red' }}>{error}</p>}
      <div className="topic-grid">
        {topics.map((topic) => (
          <div key={topic} className="topic-card" data-testid="topic-card" onClick={() => handleTopicClick(topic)}>
            {topic}
          </div>
        ))}
      </div>
      <button className="btn btn-secondary" onClick={fetchTopics}>
        Regenerate Topics
      </button>
    </div>
  )
}

export default HomePage

import { useState, useEffect } from 'react'
import {
  getSettings,
  updateSettings,
  listVocab,
  createVocab,
  importWaniKani,
  type VocabEntry,
} from '../api'

const JLPT_LEVELS = ['N5', 'N4', 'N3', 'N2', 'N1']

function SettingsPage() {
  const [jlptLevel, setJlptLevel] = useState(0)
  const [storyWordTarget, setStoryWordTarget] = useState(100)
  const [wanikaniApiKey, setWanikaniApiKey] = useState('')
  const [vocabEntries, setVocabEntries] = useState<VocabEntry[]>([])
  const [newWords, setNewWords] = useState('')
  const [importedCount, setImportedCount] = useState<number | null>(null)

  useEffect(() => {
    getSettings().then((s) => {
      const idx = JLPT_LEVELS.indexOf(s.jlpt_level)
      setJlptLevel(idx >= 0 ? idx : 0)
      setStoryWordTarget(s.story_word_target)
      setWanikaniApiKey(s.wanikani_api_key ?? '')
    })
    listVocab('', 100, 0).then((res) => setVocabEntries(res.entries))
  }, [])

  const handleSave = async () => {
    await updateSettings({
      jlpt_level: JLPT_LEVELS[jlptLevel],
      story_word_target: storyWordTarget,
      wanikani_api_key: wanikaniApiKey || undefined,
    })
  }

  const handleImport = async () => {
    const res = await importWaniKani()
    setImportedCount(res.imported_count)
  }

  const handleAddWords = async () => {
    const entries = newWords
      .split('\n')
      .map((w) => w.trim())
      .filter(Boolean)
    if (entries.length === 0) return
    const res = await createVocab(entries)
    setVocabEntries((prev) => [...res.entries, ...prev])
    setNewWords('')
  }

  return (
    <div className="container">
      <h1 className="page-title">Settings</h1>

      <div className="form-group">
        <label htmlFor="jlpt-level">JLPT Level: {JLPT_LEVELS[jlptLevel]}</label>
        <input
          id="jlpt-level"
          type="range"
          min={0}
          max={4}
          step={1}
          value={jlptLevel}
          onChange={(e) => setJlptLevel(Number(e.target.value))}
        />
      </div>

      <div className="form-group">
        <label htmlFor="story-word-target">Story Word Target</label>
        <input
          id="story-word-target"
          type="number"
          min={50}
          max={500}
          value={storyWordTarget}
          onChange={(e) => setStoryWordTarget(Number(e.target.value))}
        />
      </div>

      <div className="form-group">
        <label htmlFor="wanikani-key">WaniKani API Key</label>
        <input
          id="wanikani-key"
          type="password"
          value={wanikaniApiKey}
          onChange={(e) => setWanikaniApiKey(e.target.value)}
        />
      </div>

      <button className="btn btn-secondary" onClick={handleImport}>
        Import/Sync WaniKani
      </button>
      {importedCount !== null && <span> Imported {importedCount} items</span>}

      <button className="btn btn-primary" onClick={handleSave}>
        Save Settings
      </button>

      <h2 className="page-title">Vocab</h2>

      <div className="form-group">
        <label htmlFor="add-words">Add words (one per line)</label>
        <textarea
          id="add-words"
          value={newWords}
          onChange={(e) => setNewWords(e.target.value)}
        />
      </div>

      <button className="btn btn-primary" onClick={handleAddWords}>
        Add Words
      </button>

      <ul>
        {vocabEntries.map((entry) => (
          <li key={entry.id}>
            {entry.surface} — {entry.source}
          </li>
        ))}
      </ul>
    </div>
  )
}

export default SettingsPage

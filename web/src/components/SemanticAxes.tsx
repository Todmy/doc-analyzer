import { useState } from 'react'

interface Props {
  words: string[]
  onWordsChange: (words: string[]) => void
}

const PRESETS = [
  { name: 'Abstract ↔ Concrete', words: ['abstract', 'concrete'] },
  { name: 'Technical ↔ Simple', words: ['technical', 'simple'] },
  { name: 'Positive ↔ Negative', words: ['positive', 'negative'] },
  { name: 'Theory ↔ Practice', words: ['theory', 'practice'] },
]

export default function SemanticAxes({ words, onWordsChange }: Props) {
  const [inputValue, setInputValue] = useState('')

  const addWord = () => {
    const trimmed = inputValue.trim()
    if (trimmed && !words.includes(trimmed) && words.length < 3) {
      onWordsChange([...words, trimmed])
      setInputValue('')
    }
  }

  const removeWord = (word: string) => {
    onWordsChange(words.filter((w) => w !== word))
  }

  const applyPreset = (preset: typeof PRESETS[0]) => {
    onWordsChange(preset.words)
  }

  return (
    <div className="p-4 border-b border-gray-700">
      <div className="mb-3">
        <label className="text-sm text-gray-400 mb-2 block">
          Define semantic axes (1-3 words)
        </label>
        <div className="flex gap-2">
          <input
            type="text"
            value={inputValue}
            onChange={(e) => setInputValue(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && addWord()}
            placeholder="Enter a word (e.g., knowledge)"
            className="flex-1 px-3 py-1.5 bg-gray-700 border border-gray-600 rounded text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
            disabled={words.length >= 3}
          />
          <button
            onClick={addWord}
            disabled={!inputValue.trim() || words.length >= 3}
            className="px-3 py-1.5 bg-blue-600 hover:bg-blue-700 disabled:opacity-50 rounded text-sm"
          >
            Add
          </button>
        </div>
      </div>

      {/* Current words */}
      {words.length > 0 && (
        <div className="flex gap-2 mb-3">
          {words.map((word, i) => (
            <span
              key={word}
              className="inline-flex items-center gap-1 bg-blue-500/20 text-blue-300 px-2 py-1 rounded text-sm"
            >
              <span className="text-gray-400 text-xs">
                {i === 0 ? 'X:' : i === 1 ? 'Y:' : 'Z:'}
              </span>
              {word}
              <button
                onClick={() => removeWord(word)}
                className="ml-1 hover:text-white"
              >
                ×
              </button>
            </span>
          ))}
        </div>
      )}

      {/* Presets */}
      <div>
        <label className="text-xs text-gray-400 mb-1 block">Or use a preset:</label>
        <div className="flex flex-wrap gap-2">
          {PRESETS.map((preset) => (
            <button
              key={preset.name}
              onClick={() => applyPreset(preset)}
              className="text-xs bg-gray-700 hover:bg-gray-600 px-2 py-1 rounded"
            >
              {preset.name}
            </button>
          ))}
        </div>
      </div>
    </div>
  )
}

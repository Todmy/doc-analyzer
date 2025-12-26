import { useState, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import Visualization from '../components/Visualization'
import ClusterPanel from '../components/ClusterPanel'
import SemanticAxes from '../components/SemanticAxes'

type Tab = 'clusters' | 'similar' | 'anomalies' | 'documents'

// API response types
interface SimilarPair {
  statement1: string
  statement2: string
  file1: string
  file2: string
  similarity: number
}

interface Anomaly {
  text: string
  file: string
  line: number
  score: number
}

export default function Dashboard() {
  const { projectId } = useParams<{ projectId: string }>()
  const [activeTab, setActiveTab] = useState<Tab>('clusters')
  const [viewMode, setViewMode] = useState<'2d' | '3d'>('2d')
  const [method, setMethod] = useState<'pca' | 'semantic'>('pca')
  const [semanticWords, setSemanticWords] = useState<string[]>([])
  const [uploading, setUploading] = useState(false)
  const [selectedClusterId, setSelectedClusterId] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const queryClient = useQueryClient()

  const handleUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files
    if (!files || files.length === 0) return

    setUploading(true)
    try {
      for (const file of Array.from(files)) {
        const formData = new FormData()
        formData.append('file', file)

        const response = await fetch(`/api/v1/projects/${projectId}/documents`, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
          },
          body: formData,
        })

        if (!response.ok) {
          const error = await response.json()
          alert(`Failed to upload ${file.name}: ${error.error || 'Unknown error'}`)
        }
      }
      // Refresh data after upload (clusters are included in visualization response)
      queryClient.invalidateQueries({ queryKey: ['visualization', projectId] })
      queryClient.invalidateQueries({ queryKey: ['documents', projectId] })
    } finally {
      setUploading(false)
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const needsSemanticWords = method === 'semantic' && semanticWords.length === 0

  const { data: visualization, isLoading } = useQuery({
    queryKey: ['visualization', projectId, method, semanticWords, viewMode],
    queryFn: async () => {
      const params = new URLSearchParams({ method })
      params.set('dimensions', viewMode === '3d' ? '3' : '2')
      if (method === 'semantic') {
        semanticWords.forEach(w => params.append('words', w))
      }

      const response = await fetch(
        `/api/v1/projects/${projectId}/visualization?${params}`,
        {
          headers: {
            'Authorization': `Bearer ${localStorage.getItem('token')}`,
          },
        }
      )
      if (!response.ok) throw new Error('Failed to fetch visualization')
      return response.json()
    },
    enabled: !needsSemanticWords,
  })

  // Clusters are now included in the visualization response
  const clusters = visualization?.clusters || []

  const { data: documents } = useQuery({
    queryKey: ['documents', projectId],
    queryFn: async () => {
      const response = await fetch(`/api/v1/projects/${projectId}/documents`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch documents')
      return response.json()
    },
  })

  // Fetch similar pairs (always fetch for summary counts)
  const { data: similarPairs, isLoading: loadingSimilar } = useQuery<SimilarPair[]>({
    queryKey: ['similar-pairs', projectId],
    queryFn: async () => {
      const response = await fetch(`/api/v1/projects/${projectId}/similar-pairs`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch similar pairs')
      return response.json()
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
  })

  // Fetch anomalies (always fetch for summary counts)
  const { data: anomalies, isLoading: loadingAnomalies } = useQuery<Anomaly[]>({
    queryKey: ['anomalies', projectId],
    queryFn: async () => {
      const response = await fetch(`/api/v1/projects/${projectId}/anomalies`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch anomalies')
      return response.json()
    },
    staleTime: 5 * 60 * 1000, // 5 minutes
  })


  const handleExportReport = () => {
    const lines: string[] = []
    const date = new Date().toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    })

    lines.push('# Document Analysis Report')
    lines.push('')
    lines.push(`Generated: ${date}`)
    lines.push('')

    // Summary
    lines.push('## Summary')
    lines.push('')
    lines.push(`- **Anomalies Detected:** ${anomalies?.length || 0}`)
    lines.push(`- **Similar Pairs:** ${similarPairs?.length || 0}`)
    lines.push(`- **Documents Analyzed:** ${documents?.length || 0}`)
    lines.push('')

    // Anomalies
    if (anomalies && anomalies.length > 0) {
      lines.push('## Anomalies')
      lines.push('')
      anomalies.forEach((a, idx) => {
        const severity = a.score > 0.8 ? 'High' : a.score > 0.6 ? 'Medium' : 'Low'
        lines.push(`### ${idx + 1}. ${severity} Anomaly (${Math.round(a.score * 100)}% score)`)
        lines.push('')
        lines.push(`**Location:** ${a.file}:${a.line}`)
        lines.push('')
        lines.push(`> ${a.text}`)
        lines.push('')
        lines.push('---')
        lines.push('')
      })
    }

    // Similar Pairs
    if (similarPairs && similarPairs.length > 0) {
      lines.push('## Similar Pairs')
      lines.push('')
      lines.push('These statement pairs have high similarity and may indicate duplication or redundancy.')
      lines.push('')
      similarPairs.forEach((p, idx) => {
        lines.push(`### ${idx + 1}. ${Math.round(p.similarity * 100)}% Similar`)
        lines.push('')
        lines.push(`**From ${p.file1}:**`)
        lines.push(`> ${p.statement1}`)
        lines.push('')
        lines.push(`**From ${p.file2}:**`)
        lines.push(`> ${p.statement2}`)
        lines.push('')
        lines.push('---')
        lines.push('')
      })
    }

    // Create and download file
    const content = lines.join('\n')
    const blob = new Blob([content], { type: 'text/markdown' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `analysis-report-${new Date().toISOString().split('T')[0]}.md`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  const handleDeleteDocument = async (documentId: string) => {
    if (!confirm('Are you sure you want to delete this document?')) return

    const response = await fetch(`/api/v1/projects/${projectId}/documents/${documentId}`, {
      method: 'DELETE',
      headers: {
        'Authorization': `Bearer ${localStorage.getItem('token')}`,
      },
    })

    if (response.ok) {
      queryClient.invalidateQueries({ queryKey: ['documents', projectId] })
      queryClient.invalidateQueries({ queryKey: ['visualization', projectId] })
    } else {
      alert('Failed to delete document')
    }
  }

  const tabs: { id: Tab; label: string }[] = [
    { id: 'clusters', label: 'Clusters' },
    { id: 'documents', label: 'Documents' },
    { id: 'similar', label: 'Similar Pairs' },
    { id: 'anomalies', label: 'Anomalies' },
  ]

  return (
    <div className="min-h-screen bg-gray-900 text-white flex flex-col">
      {/* Header */}
      <header className="bg-gray-800 border-b border-gray-700 px-4 py-3">
        <div className="flex justify-between items-center">
          <div className="flex items-center gap-6">
            <h1 className="text-xl font-bold">Project Analysis</h1>
            {/* Value Summary */}
            <div className="flex gap-3">
              {anomalies && anomalies.length > 0 && (
                <button
                  onClick={() => setActiveTab('anomalies')}
                  className="flex items-center gap-1.5 px-2 py-1 rounded bg-yellow-500/20 text-yellow-400 text-sm hover:bg-yellow-500/30 transition-colors"
                >
                  <span className="font-semibold">{anomalies.length}</span>
                  <span>anomal{anomalies.length !== 1 ? 'ies' : 'y'}</span>
                </button>
              )}
              {similarPairs && similarPairs.length > 0 && (
                <button
                  onClick={() => setActiveTab('similar')}
                  className="flex items-center gap-1.5 px-2 py-1 rounded bg-blue-500/20 text-blue-400 text-sm hover:bg-blue-500/30 transition-colors"
                >
                  <span className="font-semibold">{similarPairs.length}</span>
                  <span>similar pair{similarPairs.length !== 1 ? 's' : ''}</span>
                </button>
              )}
              {(loadingAnomalies || loadingSimilar) && (
                <span className="text-gray-500 text-sm">Analyzing...</span>
              )}
            </div>
          </div>
          <div className="flex gap-2">
            <input
              ref={fileInputRef}
              type="file"
              multiple
              accept=".md,.txt,.json,.csv"
              onChange={handleUpload}
              className="hidden"
            />
            <button
              onClick={() => fileInputRef.current?.click()}
              disabled={uploading}
              className="bg-blue-600 hover:bg-blue-500 disabled:bg-blue-800 px-3 py-1 rounded text-sm"
            >
              {uploading ? 'Uploading...' : 'Upload Files'}
            </button>
            <button
              onClick={handleExportReport}
              disabled={loadingAnomalies || loadingSimilar}
              className="bg-gray-700 hover:bg-gray-600 disabled:bg-gray-800 disabled:text-gray-500 px-3 py-1 rounded text-sm"
            >
              Export Report
            </button>
          </div>
        </div>
      </header>

      <div className="flex-1 flex">
        {/* Main visualization area */}
        <div className="flex-1 p-4">
          <div className="bg-gray-800 rounded-lg h-full flex flex-col">
            {/* Visualization controls */}
            <div className="p-4 border-b border-gray-700 flex gap-4 items-center">
              <div className="flex gap-2">
                <button
                  onClick={() => setViewMode('2d')}
                  className={`px-3 py-1 rounded text-sm ${
                    viewMode === '2d' ? 'bg-blue-600' : 'bg-gray-700 hover:bg-gray-600'
                  }`}
                >
                  2D
                </button>
                <button
                  onClick={() => setViewMode('3d')}
                  className={`px-3 py-1 rounded text-sm ${
                    viewMode === '3d' ? 'bg-blue-600' : 'bg-gray-700 hover:bg-gray-600'
                  }`}
                >
                  3D
                </button>
              </div>

              <div className="flex gap-2">
                <button
                  onClick={() => setMethod('pca')}
                  className={`px-3 py-1 rounded text-sm ${
                    method === 'pca' ? 'bg-blue-600' : 'bg-gray-700 hover:bg-gray-600'
                  }`}
                >
                  Auto (PCA)
                </button>
                <button
                  onClick={() => setMethod('semantic')}
                  className={`px-3 py-1 rounded text-sm ${
                    method === 'semantic' ? 'bg-blue-600' : 'bg-gray-700 hover:bg-gray-600'
                  }`}
                >
                  Semantic Axes
                </button>
              </div>
            </div>

            {/* Semantic axes input */}
            {method === 'semantic' && (
              <SemanticAxes
                words={semanticWords}
                onWordsChange={setSemanticWords}
              />
            )}

            {/* Visualization */}
            <div className="flex-1 p-4">
              {needsSemanticWords ? (
                <div className="h-full flex items-center justify-center text-gray-400">
                  Enter semantic axis words above to visualize
                </div>
              ) : isLoading ? (
                <div className="h-full flex items-center justify-center text-gray-400">
                  Loading visualization...
                </div>
              ) : (
                <Visualization
                  points={visualization?.points || []}
                  clusters={clusters}
                  viewMode={viewMode}
                  selectedClusterId={selectedClusterId}
                />
              )}
            </div>
          </div>
        </div>

        {/* Sidebar */}
        <div className="w-80 bg-gray-800 border-l border-gray-700 flex flex-col">
          {/* Tabs */}
          <div className="flex border-b border-gray-700">
            {tabs.map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id)}
                className={`flex-1 px-2 py-3 text-sm font-medium ${
                  activeTab === tab.id
                    ? 'text-blue-400 border-b-2 border-blue-400'
                    : 'text-gray-400 hover:text-white'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>

          {/* Tab content */}
          <div className="flex-1 overflow-y-auto p-4">
            {activeTab === 'clusters' && (
              <ClusterPanel
                clusters={clusters}
                selectedClusterId={selectedClusterId}
                onClusterClick={setSelectedClusterId}
              />
            )}
            {activeTab === 'documents' && (
              <div className="space-y-2">
                {(!documents || documents.length === 0) ? (
                  <div className="text-gray-400 text-sm">No documents uploaded yet.</div>
                ) : (
                  documents.map((doc: { id: string; filename: string }) => (
                    <div
                      key={doc.id}
                      className="flex items-center justify-between bg-gray-700/50 rounded-lg p-3"
                    >
                      <span className="text-sm truncate flex-1 mr-2">{doc.filename}</span>
                      <button
                        onClick={() => handleDeleteDocument(doc.id)}
                        className="text-red-400 hover:text-red-300 text-sm px-2"
                      >
                        Delete
                      </button>
                    </div>
                  ))
                )}
              </div>
            )}
            {activeTab === 'similar' && (
              <div className="space-y-3">
                {loadingSimilar ? (
                  <div className="text-gray-400 text-sm">Loading similar pairs...</div>
                ) : !similarPairs || similarPairs.length === 0 ? (
                  <div className="text-gray-400 text-sm">No similar pairs found above 75% similarity.</div>
                ) : (
                  similarPairs.map((pair, idx) => (
                    <div key={idx} className="bg-gray-700/50 rounded-lg p-3 space-y-2">
                      <div className="flex justify-between items-center">
                        <span className="text-xs text-blue-400">{Math.round(pair.similarity * 100)}% similar</span>
                      </div>
                      <div className="text-sm text-gray-300">
                        <div className="mb-2">
                          <span className="text-xs text-gray-500">{pair.file1}</span>
                          <p className="mt-1 line-clamp-2">{pair.statement1}</p>
                        </div>
                        <div className="border-t border-gray-600 pt-2">
                          <span className="text-xs text-gray-500">{pair.file2}</span>
                          <p className="mt-1 line-clamp-2">{pair.statement2}</p>
                        </div>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}
            {activeTab === 'anomalies' && (
              <div className="space-y-3">
                {loadingAnomalies ? (
                  <div className="text-gray-400 text-sm">Loading anomalies...</div>
                ) : !anomalies || anomalies.length === 0 ? (
                  <div className="text-gray-400 text-sm">No anomalies detected.</div>
                ) : (
                  anomalies.map((anomaly, idx) => (
                    <div key={idx} className="bg-gray-700/50 rounded-lg p-3 space-y-2">
                      <div className="flex justify-between items-center">
                        <span className="text-xs text-gray-500">{anomaly.file}:{anomaly.line}</span>
                        <span className={`text-xs px-2 py-0.5 rounded ${
                          anomaly.score > 0.8 ? 'bg-red-500/20 text-red-400' :
                          anomaly.score > 0.6 ? 'bg-yellow-500/20 text-yellow-400' :
                          'bg-gray-500/20 text-gray-400'
                        }`}>
                          {Math.round(anomaly.score * 100)}% anomaly
                        </span>
                      </div>
                      <p className="text-sm text-gray-300 line-clamp-3">{anomaly.text}</p>
                    </div>
                  ))
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

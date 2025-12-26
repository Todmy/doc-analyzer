import { useState, useRef } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import Visualization from '../components/Visualization'
import ClusterPanel from '../components/ClusterPanel'
import SemanticAxes from '../components/SemanticAxes'

type Tab = 'clusters' | 'similar' | 'anomalies' | 'contradictions' | 'documents'

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
    { id: 'contradictions', label: 'Contradictions' },
  ]

  return (
    <div className="min-h-screen bg-gray-900 text-white flex flex-col">
      {/* Header */}
      <header className="bg-gray-800 border-b border-gray-700 px-4 py-3">
        <div className="flex justify-between items-center">
          <h1 className="text-xl font-bold">Project Analysis</h1>
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
            <button className="bg-gray-700 hover:bg-gray-600 px-3 py-1 rounded text-sm">
              Export
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
              <div className="text-gray-400 text-sm">Similar pairs will appear here</div>
            )}
            {activeTab === 'anomalies' && (
              <div className="text-gray-400 text-sm">Anomalies will appear here</div>
            )}
            {activeTab === 'contradictions' && (
              <div className="text-gray-400 text-sm">Contradictions will appear here</div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import Visualization from '../components/Visualization'
import ClusterPanel from '../components/ClusterPanel'
import SemanticAxes from '../components/SemanticAxes'

type Tab = 'clusters' | 'similar' | 'anomalies' | 'contradictions'

export default function Dashboard() {
  const { projectId } = useParams<{ projectId: string }>()
  const [activeTab, setActiveTab] = useState<Tab>('clusters')
  const [viewMode, setViewMode] = useState<'2d' | '3d'>('2d')
  const [method, setMethod] = useState<'pca' | 'semantic'>('pca')
  const [semanticWords, setSemanticWords] = useState<string[]>([])

  const { data: visualization, isLoading } = useQuery({
    queryKey: ['visualization', projectId, method, semanticWords],
    queryFn: async () => {
      const params = new URLSearchParams({ method })
      if (method === 'semantic' && semanticWords.length > 0) {
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
  })

  const { data: clusters } = useQuery({
    queryKey: ['clusters', projectId],
    queryFn: async () => {
      const response = await fetch(`/api/v1/projects/${projectId}/clusters`, {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch clusters')
      return response.json()
    },
  })

  const tabs: { id: Tab; label: string }[] = [
    { id: 'clusters', label: 'Clusters' },
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
              {isLoading ? (
                <div className="h-full flex items-center justify-center text-gray-400">
                  Loading visualization...
                </div>
              ) : (
                <Visualization
                  points={visualization?.points || []}
                  clusters={clusters || []}
                  viewMode={viewMode}
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
              <ClusterPanel clusters={clusters || []} />
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

interface Cluster {
  id: string
  label: number
  keywords: string[]
  size: number
  density?: number
}

interface Props {
  clusters: Cluster[]
  selectedClusterId?: string | null
  onClusterClick?: (clusterId: string | null) => void
}

const COLORS = [
  '#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6',
  '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6366f1',
]

export default function ClusterPanel({ clusters, selectedClusterId, onClusterClick }: Props) {
  if (clusters.length === 0) {
    return (
      <div className="text-gray-400 text-sm">
        No clusters found. Upload and analyze documents first.
      </div>
    )
  }

  const handleClick = (clusterId: string) => {
    if (onClusterClick) {
      // Toggle selection: if already selected, deselect
      onClusterClick(selectedClusterId === clusterId ? null : clusterId)
    }
  }

  return (
    <div className="space-y-3">
      {clusters.map((cluster, index) => {
        const isSelected = selectedClusterId === cluster.id
        const color = COLORS[index % COLORS.length]

        return (
          <div
            key={cluster.id}
            onClick={() => handleClick(cluster.id)}
            className={`rounded-lg p-3 cursor-pointer transition-all ${
              isSelected
                ? 'ring-2 ring-offset-2 ring-offset-gray-800 bg-gray-700'
                : 'bg-gray-700/50 hover:bg-gray-700'
            }`}
            style={isSelected ? { boxShadow: `0 0 0 2px ${color}` } : undefined}
          >
            <div className="flex justify-between items-start mb-2">
              <div className="flex items-center gap-2">
                <div
                  className="w-3 h-3 rounded-full"
                  style={{ backgroundColor: color }}
                />
                <span className="text-sm font-medium">Cluster {index + 1}</span>
              </div>
              <span className="text-xs text-gray-400">{cluster.size} items</span>
            </div>
            <div className="flex flex-wrap gap-1">
              {cluster.keywords.slice(0, 5).map((keyword, i) => (
                <span
                  key={i}
                  className="text-xs bg-blue-500/20 text-blue-300 px-2 py-0.5 rounded"
                >
                  {keyword}
                </span>
              ))}
            </div>
            {cluster.density !== undefined && (
              <div className="mt-2 text-xs text-gray-400">
                Density: {(cluster.density * 100).toFixed(1)}%
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

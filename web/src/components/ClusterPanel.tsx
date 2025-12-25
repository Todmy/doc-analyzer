interface Cluster {
  id: string
  label: number
  keywords: string[]
  size: number
  density?: number
}

interface Props {
  clusters: Cluster[]
}

export default function ClusterPanel({ clusters }: Props) {
  if (clusters.length === 0) {
    return (
      <div className="text-gray-400 text-sm">
        No clusters found. Upload and analyze documents first.
      </div>
    )
  }

  return (
    <div className="space-y-3">
      {clusters.map((cluster, index) => (
        <div
          key={cluster.id}
          className="bg-gray-700/50 rounded-lg p-3 hover:bg-gray-700 cursor-pointer transition-colors"
        >
          <div className="flex justify-between items-start mb-2">
            <span className="text-sm font-medium">Cluster {index + 1}</span>
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
      ))}
    </div>
  )
}

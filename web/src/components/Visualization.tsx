import Plot from 'react-plotly.js'

interface Point {
  x: number
  y: number
  z?: number
  cluster_id?: string
  preview?: string
  anomaly_score?: number
}

interface Cluster {
  id: string
  label: number
  keywords: string[]
}

interface Props {
  points: Point[]
  clusters: Cluster[]
  viewMode: '2d' | '3d'
  selectedClusterId?: string | null
}

const COLORS = [
  '#3b82f6', '#ef4444', '#22c55e', '#f59e0b', '#8b5cf6',
  '#ec4899', '#06b6d4', '#84cc16', '#f97316', '#6366f1',
]

export default function Visualization({ points, clusters, viewMode, selectedClusterId }: Props) {
  if (points.length === 0) {
    return (
      <div className="h-full flex items-center justify-center text-gray-400">
        No data to visualize. Upload documents to get started.
      </div>
    )
  }

  // Group points by cluster
  const clusterMap = new Map<string, Point[]>()
  points.forEach((point) => {
    const clusterId = point.cluster_id || 'unclustered'
    if (!clusterMap.has(clusterId)) {
      clusterMap.set(clusterId, [])
    }
    clusterMap.get(clusterId)!.push(point)
  })

  const traces: Plotly.Data[] = []

  clusterMap.forEach((clusterPoints, clusterId) => {
    const clusterIndex = clusters.findIndex((c) => c.id === clusterId)
    const color = clusterIndex >= 0 ? COLORS[clusterIndex % COLORS.length] : '#6b7280'
    const cluster = clusters[clusterIndex]
    const name = cluster ? `Cluster: ${cluster.keywords.slice(0, 3).join(', ')}` : 'Unclustered'

    // Determine opacity based on selection
    const isSelected = selectedClusterId === null || selectedClusterId === undefined || clusterId === selectedClusterId
    const opacity = isSelected ? 0.9 : 0.15
    const markerSize = viewMode === '3d' ? 6 : 10

    if (viewMode === '3d') {
      traces.push({
        type: 'scatter3d',
        mode: 'markers',
        name,
        x: clusterPoints.map((p) => p.x),
        y: clusterPoints.map((p) => p.y),
        z: clusterPoints.map((p) => p.z || 0),
        text: clusterPoints.map((p) => p.preview || ''),
        hoverinfo: 'text',
        marker: {
          size: markerSize,
          color,
          opacity,
          line: {
            color: clusterPoints.map((p) =>
              (p.anomaly_score || 0) > 0.7 ? '#ef4444' : 'transparent'
            ),
            width: 2,
          },
        },
      })
    } else {
      traces.push({
        type: 'scatter',
        mode: 'markers',
        name,
        x: clusterPoints.map((p) => p.x),
        y: clusterPoints.map((p) => p.y),
        text: clusterPoints.map((p) => p.preview || ''),
        hoverinfo: 'text',
        marker: {
          size: markerSize,
          color,
          opacity,
          line: {
            color: clusterPoints.map((p) =>
              (p.anomaly_score || 0) > 0.7 ? '#ef4444' : 'transparent'
            ),
            width: 2,
          },
        },
      })
    }
  })

  const layout: Partial<Plotly.Layout> = {
    paper_bgcolor: 'transparent',
    plot_bgcolor: 'transparent',
    font: { color: '#9ca3af' },
    margin: { l: 40, r: 40, t: 40, b: 40 },
    showlegend: true,
    dragmode: 'pan',
    legend: {
      x: 0,
      y: 1,
      bgcolor: 'rgba(31, 41, 55, 0.8)',
      bordercolor: '#374151',
      borderwidth: 1,
    },
    xaxis: {
      gridcolor: '#374151',
      zerolinecolor: '#374151',
    },
    yaxis: {
      gridcolor: '#374151',
      zerolinecolor: '#374151',
    },
  }

  if (viewMode === '3d') {
    layout.scene = {
      xaxis: { gridcolor: '#374151', color: '#9ca3af' },
      yaxis: { gridcolor: '#374151', color: '#9ca3af' },
      zaxis: { gridcolor: '#374151', color: '#9ca3af' },
      bgcolor: 'transparent',
    }
  }

  return (
    <Plot
      data={traces}
      layout={layout}
      config={{
        responsive: true,
        displayModeBar: true,
        scrollZoom: true,
        modeBarButtonsToRemove: ['lasso2d', 'select2d'],
      }}
      style={{ width: '100%', height: '100%' }}
    />
  )
}

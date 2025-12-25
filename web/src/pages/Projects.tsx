import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

interface Project {
  id: string
  name: string
  created_at: string
}

export default function Projects() {
  const queryClient = useQueryClient()
  const [newProjectName, setNewProjectName] = useState('')
  const [showCreate, setShowCreate] = useState(false)

  const { data: projects, isLoading } = useQuery<Project[]>({
    queryKey: ['projects'],
    queryFn: async () => {
      const response = await fetch('/api/v1/projects', {
        headers: {
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
      })
      if (!response.ok) throw new Error('Failed to fetch projects')
      return response.json()
    },
  })

  const createMutation = useMutation({
    mutationFn: async (name: string) => {
      const response = await fetch('/api/v1/projects', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token')}`,
        },
        body: JSON.stringify({ name }),
      })
      if (!response.ok) throw new Error('Failed to create project')
      return response.json()
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      setNewProjectName('')
      setShowCreate(false)
    },
  })

  return (
    <div className="min-h-screen bg-gray-900 text-white">
      <header className="bg-gray-800 border-b border-gray-700">
        <div className="max-w-7xl mx-auto px-4 py-4 flex justify-between items-center">
          <h1 className="text-xl font-bold">Doc Analyzer</h1>
          <button
            onClick={() => setShowCreate(true)}
            className="bg-blue-600 hover:bg-blue-700 px-4 py-2 rounded-md text-sm font-medium"
          >
            New Project
          </button>
        </div>
      </header>

      <main className="max-w-7xl mx-auto px-4 py-8">
        {showCreate && (
          <div className="mb-8 bg-gray-800 p-6 rounded-lg">
            <h2 className="text-lg font-semibold mb-4">Create New Project</h2>
            <div className="flex gap-4">
              <input
                type="text"
                value={newProjectName}
                onChange={(e) => setNewProjectName(e.target.value)}
                placeholder="Project name"
                className="flex-1 px-3 py-2 bg-gray-700 border border-gray-600 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              <button
                onClick={() => createMutation.mutate(newProjectName)}
                disabled={!newProjectName || createMutation.isPending}
                className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 px-4 py-2 rounded-md"
              >
                Create
              </button>
              <button
                onClick={() => setShowCreate(false)}
                className="bg-gray-700 hover:bg-gray-600 px-4 py-2 rounded-md"
              >
                Cancel
              </button>
            </div>
          </div>
        )}

        {isLoading ? (
          <div className="text-center py-12 text-gray-400">Loading...</div>
        ) : projects?.length === 0 ? (
          <div className="text-center py-12">
            <p className="text-gray-400 mb-4">No projects yet</p>
            <button
              onClick={() => setShowCreate(true)}
              className="bg-blue-600 hover:bg-blue-700 px-4 py-2 rounded-md"
            >
              Create your first project
            </button>
          </div>
        ) : (
          <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
            {projects?.map((project) => (
              <Link
                key={project.id}
                to={`/projects/${project.id}`}
                className="block bg-gray-800 p-6 rounded-lg hover:bg-gray-750 border border-gray-700 hover:border-blue-500 transition-colors"
              >
                <h3 className="text-lg font-semibold mb-2">{project.name}</h3>
                <p className="text-gray-400 text-sm">
                  Created {new Date(project.created_at).toLocaleDateString()}
                </p>
              </Link>
            ))}
          </div>
        )}
      </main>
    </div>
  )
}

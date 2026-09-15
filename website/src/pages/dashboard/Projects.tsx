import { useState } from 'react';
import { Link } from 'react-router-dom';
import { ExternalLink, FolderOpen, Loader2 } from 'lucide-react';
import StatusBadge from '@/components/StatusBadge';
import { useProjects } from '@/hooks/useProjects';

const Projects = () => {
  const { projects, loading, deletingId, deleteProject } = useProjects();
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const handleDeleteProject = async (id: number, name: string) => {
    const confirmed = window.confirm(
      `Are you sure you want to delete "${name}"? This will permanently delete the deployment and its files.`
    );
    if (!confirmed) return;

    setDeleteError(null);
    try {
      await deleteProject(id);
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete project');
    }
  };

  return (
    <div className='space-y-6'>
      <div className='flex items-center justify-between'>
        <div>
          <h2 className='text-xl font-semibold'>Your Projects</h2>
          <p className='text-sm text-white/40'>Manage your deployed applications and drafts</p>
        </div>
        <Link
          to='/dashboard/import'
          className='rounded-lg bg-white px-4 py-2 text-xs font-semibold text-black hover:bg-white/90 transition-all'
        >
          Import Project
        </Link>
      </div>

      {deleteError && (
        <div className='rounded-lg border border-red-500/20 bg-red-500/5 px-4 py-3 text-xs text-red-400'>
          {deleteError}
        </div>
      )}

      {loading ? (
        <div className='flex h-48 flex-col items-center justify-center gap-3 rounded-xl border border-white/10 bg-white/5 p-6'>
          <Loader2 className='size-8 animate-spin text-emerald-400' />
          <p className='text-sm text-white/40'>Loading projects...</p>
        </div>
      ) : projects.length === 0 ? (
        <div className='flex h-48 flex-col items-center justify-center gap-3 rounded-xl border border-white/10 bg-white/5 p-6 text-center'>
          <FolderOpen className='size-10 text-white/20' />
          <p className='text-sm text-white/60'>No projects imported yet.</p>
          <Link
            to='/dashboard/import'
            className='mt-2 rounded-lg bg-white px-4 py-2 text-xs font-semibold text-black hover:bg-white/90 transition-all'
          >
            Import First Project
          </Link>
        </div>
      ) : (
        <div className='divide-y divide-white/5 rounded-xl border border-white/10 bg-white/5 overflow-hidden'>
          {projects.map((project) => (
            <div
              key={project.id}
              className='flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 hover:bg-white/[0.02] transition-colors'
            >
              <div className='space-y-1 min-w-0'>
                <div className='flex items-center gap-2.5'>
                  <Link
                    to={`/project/${project.id}`}
                    className='text-base font-semibold text-white/90 truncate hover:underline'
                  >
                    {project.name}
                  </Link>
                  <StatusBadge status={project.status} className='px-2 py-0.5 text-[10px]' />
                </div>
                <p className='text-xs text-white/40 truncate max-w-md' title={project.repo_url}>
                  Source: {project.repo_url}
                </p>
                {project.status === 'deployed' && project.url && (
                  <a
                    href={project.url}
                    target='_blank'
                    rel='noopener noreferrer'
                    className='inline-flex items-center gap-1 text-[11px] text-emerald-400/80 hover:text-emerald-300 truncate max-w-md'
                    title={project.url}
                  >
                    <ExternalLink className='size-3 shrink-0' />
                    {project.url}
                  </a>
                )}
              </div>

              <div className='flex items-center gap-3'>
                <button
                  onClick={() => handleDeleteProject(project.id, project.name)}
                  disabled={deletingId === project.id}
                  className='flex items-center gap-1.5 rounded-lg border border-red-500/20 bg-red-500/5 hover:bg-red-500/10 px-4 py-2 text-xs font-semibold text-red-400 hover:text-red-300 transition-all active:scale-95 disabled:opacity-50 disabled:pointer-events-none'
                >
                  {deletingId === project.id && <Loader2 className='size-3 animate-spin' />}
                  {deletingId === project.id ? 'Deleting...' : 'Delete'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default Projects;

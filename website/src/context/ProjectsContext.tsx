import { createContext, useCallback, useEffect, useState, type ReactNode } from 'react';
import { del, get } from '@/utils/api';

export interface Project {
  id: number;
  name: string;
  stack: string;
  status: 'queued' | 'pending' | 'deployed' | 'failed';
  url: string | null;
  repo_url: string;
  created_at: string;
}

interface ProjectsContextType {
  projects: Project[];
  loading: boolean;
  error: string | null;
  deletingId: number | null;
  refresh: () => Promise<void>;
  deleteProject: (id: number) => Promise<void>;
}

export const ProjectsContext = createContext<ProjectsContextType | undefined>(undefined);

export const ProjectsProvider = ({ children }: { children: ReactNode }) => {
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deletingId, setDeletingId] = useState<number | null>(null);

  const refresh = useCallback(async () => {
    try {
      const data = await get<Project[]>('deployments');
      setProjects(data || []);
      setError(null);
    } catch (err) {
      console.error('Failed to fetch projects', err);
      setError(err instanceof Error ? err.message : 'Failed to fetch projects');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const deleteProject = useCallback(async (id: number) => {
    setDeletingId(id);
    try {
      await del(`deployments/${id}`);
      // Drop it locally straight away so the UI reacts even before the refetch lands
      setProjects((prev) => prev.filter((project) => project.id !== id));
      setError(null);
      await refresh();
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete project';
      setError(message);
      throw new Error(message);
    } finally {
      setDeletingId(null);
    }
  }, [refresh]);

  return (
    <ProjectsContext.Provider value={{ projects, loading, error, deletingId, refresh, deleteProject }}>
      {children}
    </ProjectsContext.Provider>
  );
};

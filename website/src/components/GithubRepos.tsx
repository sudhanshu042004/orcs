import { useEffect, useState } from 'react';
import { get, post } from '@/utils/api';
import { Loader2, FolderGit2, Star, GitFork, ExternalLink, Code } from 'lucide-react';

interface Repo {
  id: number;
  name: string;
  description: string | null;
  html_url: string;
  stargazers_count: number;
  forks_count: number;
  language: string | null;
  updated_at: string;
}

const languageColors: Record<string, string> = {
  JavaScript: 'bg-yellow-400',
  TypeScript: 'bg-blue-500',
  Go: 'bg-cyan-500',
  HTML: 'bg-orange-500',
  CSS: 'bg-purple-500',
  Python: 'bg-blue-400',
  Rust: 'bg-orange-600',
  Java: 'bg-red-500',
};

interface GithubReposProps {
  onDeploySuccess?: () => void;
}

const GithubRepos = ({ onDeploySuccess }: GithubReposProps) => {
  const [repos, setRepos] = useState<Repo[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [cloningRepoId, setCloningRepoId] = useState<number | null>(null);
  const [cloneStatus, setCloneStatus] = useState<Record<number, { success?: boolean; error?: string; path?: string }>>({});

  useEffect(() => {
    const fetchRepos = async () => {
      try {
        const response = await get<Repo[]>('repos');
        setRepos(response);
      } catch (err: any) {
        setError(err.message || 'Failed to fetch repositories');
      } finally {
        setLoading(false);
      }
    };

    fetchRepos();
  }, []);

  const handleDeploy = async (repoId: number, name: string, cloneUrl: string) => {
    setCloningRepoId(repoId);
    setCloneStatus((prev) => ({ ...prev, [repoId]: {} }));
    try {
      const res = await post<{ message: string; path: string }>('projects/deploy', {
        name,
        clone_url: cloneUrl,
      });
      setCloneStatus((prev) => ({
        ...prev,
        [repoId]: { success: true, path: res.path },
      }));
      if (onDeploySuccess) {
        onDeploySuccess();
      }
    } catch (err: any) {
      setCloneStatus((prev) => ({
        ...prev,
        [repoId]: { error: err.message || 'Failed to deploy project' },
      }));
    } finally {
      setCloningRepoId(null);
    }
  };

  return (
    <div className="w-full space-y-6">
      {loading ? (
        <div className="flex h-48 flex-col items-center justify-center gap-3 rounded-xl border border-white/10 bg-white/5 p-6">
          <Loader2 className="size-8 animate-spin text-emerald-400" />
          <p className="text-sm text-white/40">Fetching your GitHub repositories...</p>
        </div>
      ) : error ? (
        <div className="flex h-48 flex-col items-center justify-center gap-2 rounded-xl border border-red-500/20 bg-red-500/5 p-6 text-center">
          <p className="text-sm font-medium text-red-400">Failed to load repositories</p>
          <p className="text-xs text-red-400/60 max-w-md">{error}</p>
        </div>
      ) : !repos || repos.length === 0 ? (
        <div className="flex h-48 flex-col items-center justify-center gap-3 rounded-xl border border-white/10 bg-white/5 p-6 text-center">
          <FolderGit2 className="size-10 text-white/20" />
          <p className="text-sm text-white/60">No repositories found for this user.</p>
        </div>
      ) : (
        <div className="divide-y divide-white/5 rounded-xl border border-white/10 bg-white/5 overflow-hidden">
          {repos.map((repo) => (
            <div
              key={repo.id}
              className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-4 transition-all duration-200 hover:bg-white/5"
            >
              {/* Repository details */}
              <div className="flex-1 min-w-0 space-y-1">
                <div className="flex items-center gap-2">
                  <h4 className="text-sm font-semibold text-white/90 truncate">
                    {repo.name}
                  </h4>
                  <a
                    href={repo.html_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-white/40 hover:text-white transition-colors animate-pulse-hover"
                    title="View on GitHub"
                  >
                    <ExternalLink className="size-3.5" />
                  </a>
                </div>
                {repo.description && (
                  <p className="text-xs text-white/40 truncate max-w-lg">
                    {repo.description}
                  </p>
                )}
              </div>

              {/* Stats & Language */}
              <div className="flex items-center gap-6 text-xs text-white/40">
                <div className="flex items-center gap-1.5 min-w-[90px]">
                  {repo.language ? (
                    <>
                      <span className={`size-2 rounded-full ${languageColors[repo.language] || 'bg-white/30'}`} />
                      <span>{repo.language}</span>
                    </>
                  ) : (
                    <>
                      <Code className="size-3 text-white/30" />
                      <span>Unknown</span>
                    </>
                  )}
                </div>

                <div className="flex items-center gap-3">
                  <span className="flex items-center gap-1" title="Stars">
                    <Star className="size-3 text-yellow-500/80 fill-yellow-500/20" />
                    {repo.stargazers_count}
                  </span>
                  <span className="flex items-center gap-1" title="Forks">
                    <GitFork className="size-3 text-blue-400/80" />
                    {repo.forks_count}
                  </span>
                </div>
              </div>

              {/* Action / Deployment status */}
              <div className="sm:min-w-[180px] text-left sm:text-right">
                {cloningRepoId === repo.id ? (
                  <div className="inline-flex items-center gap-1.5 rounded-lg border border-emerald-500/20 bg-emerald-500/5 px-3 py-1.5 text-xs text-emerald-400">
                    <Loader2 className="size-3 animate-spin" />
                    <span>Deploying...</span>
                  </div>
                ) : cloneStatus[repo.id]?.success ? (
                  <div>
                    <span className="inline-block rounded-md bg-emerald-500/10 border border-emerald-500/20 px-2.5 py-0.5 text-[10px] font-medium text-emerald-400">
                      Success
                    </span>
                    <p className="mt-1 text-[9px] text-white/30 truncate max-w-[180px]" title={cloneStatus[repo.id]?.path}>
                      {cloneStatus[repo.id]?.path}
                    </p>
                  </div>
                ) : (
                  <div className="flex flex-col items-start sm:items-end gap-1">
                    {cloneStatus[repo.id]?.error && (
                      <p className="text-[10px] text-red-400 max-w-[180px] truncate leading-tight" title={cloneStatus[repo.id]?.error}>
                        {cloneStatus[repo.id]?.error}
                      </p>
                    )}
                    <button
                      onClick={() => handleDeploy(repo.id, repo.name, repo.html_url)}
                      disabled={cloningRepoId !== null}
                      className="rounded-lg bg-white/10 px-4 py-1.5 text-xs font-medium text-white/80 transition-all hover:bg-white/20 active:scale-95 disabled:opacity-50 disabled:pointer-events-none"
                    >
                      Deploy
                    </button>
                  </div>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default GithubRepos;

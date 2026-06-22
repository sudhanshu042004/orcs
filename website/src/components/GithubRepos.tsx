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

const GithubRepos = () => {
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

  const handleClone = async (repoId: number, name: string, cloneUrl: string) => {
    setCloningRepoId(repoId);
    setCloneStatus((prev) => ({ ...prev, [repoId]: {} }));
    try {
      const res = await post<{ message: string; path: string }>('projects/clone', {
        name,
        clone_url: cloneUrl,
      });
      setCloneStatus((prev) => ({
        ...prev,
        [repoId]: { success: true, path: res.path },
      }));
    } catch (err: any) {
      setCloneStatus((prev) => ({
        ...prev,
        [repoId]: { error: err.message || 'Failed to clone repository' },
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
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {repos.map((repo) => (
            <div
              key={repo.id}
              className="group relative flex flex-col justify-between rounded-xl border border-white/10 bg-white/5 p-5 transition-all duration-300 hover:border-white/20 hover:bg-white/10 hover:shadow-lg hover:shadow-emerald-950/10"
            >
              <div className="space-y-2">
                <div className="flex items-start justify-between gap-2">
                  <h4 className="font-semibold text-white/90 group-hover:text-white transition-colors truncate">
                    {repo.name}
                  </h4>
                  <a
                    href={repo.html_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="flex size-7 items-center justify-center rounded-lg bg-white/5 text-white/60 opacity-0 group-hover:opacity-100 hover:bg-white/10 hover:text-white transition-all duration-200"
                    title="View on GitHub"
                  >
                    <ExternalLink className="size-3.5" />
                  </a>
                </div>
                
                <p className="line-clamp-2 text-xs leading-relaxed text-white/50 min-h-[2rem]">
                  {repo.description || "No description provided."}
                </p>
              </div>

              <div className="mt-4">
                {cloningRepoId === repo.id ? (
                  <div className="flex items-center justify-center gap-2 rounded-lg border border-emerald-500/20 bg-emerald-500/5 py-2 text-xs text-emerald-400">
                    <Loader2 className="size-3.5 animate-spin" />
                    <span>Cloning into namespace...</span>
                  </div>
                ) : cloneStatus[repo.id]?.success ? (
                  <div className="rounded-lg bg-emerald-500/10 border border-emerald-500/20 p-2.5 text-center">
                    <p className="text-[11px] font-medium text-emerald-400">Cloned successfully!</p>
                    <p className="mt-0.5 text-[9px] text-white/40 truncate" title={cloneStatus[repo.id]?.path}>
                      {cloneStatus[repo.id]?.path}
                    </p>
                  </div>
                ) : (
                  <div className="flex flex-col gap-1.5">
                    {cloneStatus[repo.id]?.error && (
                      <p className="text-[10px] text-red-400 line-clamp-2 leading-tight">
                        Error: {cloneStatus[repo.id]?.error}
                      </p>
                    )}
                    <button
                      onClick={() => handleClone(repo.id, repo.name, repo.html_url)}
                      disabled={cloningRepoId !== null}
                      className="w-full rounded-lg bg-white/10 py-2 text-center text-xs font-medium text-white/80 transition-all hover:bg-white/20 active:scale-98 disabled:opacity-50 disabled:pointer-events-none"
                    >
                      Clone & Isolate
                    </button>
                  </div>
                )}
              </div>

              <div className="mt-4 flex items-center justify-between text-xs text-white/40 border-t border-white/5 pt-3">
                <div className="flex items-center gap-1.5">
                  {repo.language ? (
                    <>
                      <span className={`size-2.5 rounded-full ${languageColors[repo.language] || 'bg-white/30'}`} />
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
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default GithubRepos;

import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { get } from '@/utils/api';
import { Loader2, FolderGit2, Star, GitFork, ExternalLink, Code, ArrowRight } from 'lucide-react';

interface Repo {
  id: number;
  name: string;
  description: string | null;
  html_url: string;
  clone_url: string;
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
  const navigate = useNavigate();
  const [repos, setRepos] = useState<Repo[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchRepos = async () => {
      try {
        const response = await get<Repo[]>('repos');
        setRepos(response);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch repositories');
      } finally {
        setLoading(false);
      }
    };

    fetchRepos();
  }, []);

  // Picking a repo never deploys straight away - it opens the deploy configuration form
  const handleSelect = (repo: Repo) => {
    const params = new URLSearchParams({
      name: repo.name,
      repo_url: repo.clone_url || repo.html_url,
    });
    navigate(`/dashboard/deploy?${params.toString()}`);
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
                    className="text-white/40 hover:text-white transition-colors"
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

              {/* Action */}
              <div className="sm:min-w-[150px] text-left sm:text-right">
                <button
                  onClick={() => handleSelect(repo)}
                  className="inline-flex items-center gap-1.5 rounded-lg bg-white/10 px-4 py-1.5 text-xs font-medium text-white/80 transition-all hover:bg-white/20 active:scale-95"
                >
                  Configure
                  <ArrowRight className="size-3" />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
};

export default GithubRepos;

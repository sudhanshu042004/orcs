import { Link, useSearchParams } from 'react-router-dom';
import { FolderGit2, FolderUp } from 'lucide-react';
import GithubRepos from '@/components/GithubRepos';

const sources = [
  { key: 'github', label: 'GitHub Repository', icon: FolderGit2, enabled: true, hint: 'Cloned and built in a container' },
  { key: 'folder', label: 'Local Folder', icon: FolderUp, enabled: false, hint: 'Coming soon' },
] as const;

const Import = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const source = searchParams.get('source') === 'folder' ? 'folder' : 'github';

  return (
    <div className='space-y-6'>
      <div className='flex items-start justify-between gap-4'>
        <div>
          <h2 className='text-xl font-semibold'>Import Project</h2>
          <p className='text-sm text-white/40'>
            Pick a repository, then configure how it should be built.
          </p>
        </div>
        <Link to='/dashboard' className='text-xs font-medium text-white/40 hover:text-white transition-colors'>
          Cancel
        </Link>
      </div>

      {/* Source - reflected in the URL so each source is linkable */}
      <div className='grid gap-3 sm:grid-cols-2'>
        {sources.map((item) => {
          const Icon = item.icon;
          const isSelected = source === item.key;
          return (
            <button
              key={item.key}
              type='button'
              disabled={!item.enabled}
              onClick={() => setSearchParams({ source: item.key })}
              className={`flex items-center gap-3 rounded-xl border px-5 py-4 text-left transition-all ${
                isSelected
                  ? 'border-indigo-500/60 bg-indigo-500/10'
                  : 'border-white/10 bg-white/5 hover:border-white/20'
              } ${item.enabled ? '' : 'cursor-not-allowed opacity-40 hover:border-white/10'}`}
            >
              <Icon className='size-5 text-white/60' />
              <div>
                <p className='text-sm font-semibold'>{item.label}</p>
                <p className='text-[11px] text-white/40'>{item.hint}</p>
              </div>
            </button>
          );
        })}
      </div>

      <GithubRepos />
    </div>
  );
};

export default Import;

import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { ArrowLeft, Loader2, Rocket } from 'lucide-react';
import { get, post } from '@/utils/api';
import { useProjects } from '@/hooks/useProjects';

interface Stack {
  key: string;
  kind: 'static' | 'dynamic';
  label: string;
  image: string;
  enabled: boolean;
  locked: boolean;
  install_cmd: string;
  build_cmd: string;
  run_cmd: string;
  port: number;
}

const isValidRepoUrl = (value: string) => /^https?:\/\/.+\/.+/.test(value.trim());

const Deploy = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { refresh } = useProjects();

  const [stacks, setStacks] = useState<Stack[]>([]);
  const [loadingStacks, setLoadingStacks] = useState(true);
  const [selectedStack, setSelectedStack] = useState<string>('react');

  const [name, setName] = useState(searchParams.get('name') ?? '');
  const [repoUrl, setRepoUrl] = useState(searchParams.get('repo_url') ?? '');
  const [installCmd, setInstallCmd] = useState('npm i');
  const [buildCmd, setBuildCmd] = useState('npm run build');
  const [runCmd, setRunCmd] = useState('');

  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchStacks = async () => {
      try {
        const data = await get<Stack[]>('stacks');
        setStacks(data || []);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load project types');
      } finally {
        setLoadingStacks(false);
      }
    };
    fetchStacks();
  }, []);

  const stack = useMemo(() => stacks.find((s) => s.key === selectedStack), [stacks, selectedStack]);

  // A locked stack (React today) always deploys with its own commands
  useEffect(() => {
    if (stack?.locked) {
      setInstallCmd(stack.install_cmd);
      setBuildCmd(stack.build_cmd);
      setRunCmd(stack.run_cmd);
    }
  }, [stack]);

  const locked = stack?.locked ?? false;
  // A dynamic stack keeps running after it is built, so it has to know how to start
  const dynamic = stack?.kind === 'dynamic';
  const canDeploy =
    !!stack?.enabled &&
    name.trim().length > 0 &&
    isValidRepoUrl(repoUrl) &&
    installCmd.trim().length > 0 &&
    buildCmd.trim().length > 0 &&
    (!dynamic || runCmd.trim().length > 0) &&
    !submitting;

  const handleDeploy = async () => {
    if (!canDeploy) return;
    setSubmitting(true);
    setError(null);

    try {
      const res = await post<{ deployment_id: number; status: string }>('deployments', {
        name: name.trim(),
        stack: selectedStack,
        repo_url: repoUrl.trim(),
        install_cmd: installCmd.trim(),
        build_cmd: buildCmd.trim(),
        run_cmd: runCmd.trim(),
      });
      refresh();
      navigate(`/project/${res.deployment_id}`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to queue deployment');
      setSubmitting(false);
    }
  };

  const fieldClass =
    'w-full rounded-lg border border-white/10 bg-black/40 px-3.5 py-2 text-sm text-white placeholder-white/20 focus:border-indigo-500 focus:outline-none disabled:cursor-not-allowed disabled:text-white/40';

  return (
    <div className='max-w-3xl space-y-6'>
      <Link
        to='/dashboard/import'
        className='group inline-flex items-center gap-2 text-xs font-medium text-white/40 hover:text-white transition-colors'
      >
        <ArrowLeft className='size-3.5 group-hover:-translate-x-0.5 transition-transform' />
        Back to import
      </Link>

      <div>
        <h2 className='text-xl font-semibold'>Configure deployment</h2>
        <p className='text-sm text-white/40'>
          Tell us about the project. The build runs in a container; a site is published as static files, an app
          keeps running and is served from its container.
        </p>
      </div>

      {error && (
        <div className='rounded-lg border border-red-500/20 bg-red-500/5 px-4 py-3 text-xs text-red-400'>{error}</div>
      )}

      {/* Stack */}
      <section className='space-y-3 rounded-xl border border-white/10 bg-white/5 p-5'>
        <div>
          <h3 className='text-sm font-semibold'>Project type</h3>
          <p className='text-xs text-white/40'>What is this codebase built with?</p>
        </div>

        {loadingStacks ? (
          <div className='flex items-center gap-2 text-xs text-white/30'>
            <Loader2 className='size-3 animate-spin' /> Loading project types...
          </div>
        ) : (
          <div className='grid grid-cols-2 gap-3 sm:grid-cols-4'>
            {stacks.map((item) => {
              const isSelected = selectedStack === item.key;
              return (
                <button
                  key={item.key}
                  type='button'
                  disabled={!item.enabled}
                  onClick={() => setSelectedStack(item.key)}
                  title={item.enabled ? item.image : 'Supported in a future release'}
                  className={`flex flex-col items-start gap-1 rounded-lg border px-4 py-3 text-left transition-all ${
                    isSelected
                      ? 'border-indigo-500/60 bg-indigo-500/10'
                      : 'border-white/10 bg-black/30 hover:border-white/20'
                  } ${item.enabled ? '' : 'cursor-not-allowed opacity-40 hover:border-white/10'}`}
                >
                  <span className='text-sm font-semibold'>{item.label}</span>
                  <span className='text-[10px] uppercase tracking-wider text-white/35'>
                    {item.enabled ? (item.kind === 'dynamic' ? 'Runs live' : 'Static site') : 'Coming soon'}
                  </span>
                </button>
              );
            })}
          </div>
        )}
      </section>

      {/* Details */}
      <section className='space-y-4 rounded-xl border border-white/10 bg-white/5 p-5'>
        <div>
          <h3 className='text-sm font-semibold'>Deployment details</h3>
          <p className='text-xs text-white/40'>Name it and point us at the repository to clone.</p>
        </div>

        <div className='space-y-1.5'>
          <label className='text-xs font-medium text-white/50' htmlFor='deployment-name'>
            Deployment name
          </label>
          <input
            id='deployment-name'
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder='my-portfolio'
            className={fieldClass}
          />
        </div>

        <div className='space-y-1.5'>
          <label className='text-xs font-medium text-white/50' htmlFor='repo-url'>
            Repository URL
          </label>
          <input
            id='repo-url'
            value={repoUrl}
            onChange={(e) => setRepoUrl(e.target.value)}
            placeholder='https://github.com/you/your-repo'
            className={fieldClass}
          />
          {repoUrl.length > 0 && !isValidRepoUrl(repoUrl) && (
            <p className='text-[11px] text-red-400'>Enter a full https:// repository URL</p>
          )}
        </div>
      </section>

      {/* Commands */}
      <section className='space-y-4 rounded-xl border border-white/10 bg-white/5 p-5'>
        <div>
          <h3 className='text-sm font-semibold'>Commands</h3>
          <p className='text-xs text-white/40'>
            {locked
              ? `${stack?.label} projects always use their own commands.`
              : 'Commands run inside the build container, from the repository root.'}
          </p>
        </div>

        <div className='grid gap-4 sm:grid-cols-2'>
          <div className='space-y-1.5'>
            <label className='text-xs font-medium text-white/50' htmlFor='install-cmd'>
              Install command
            </label>
            <input
              id='install-cmd'
              value={installCmd}
              disabled={locked}
              onChange={(e) => setInstallCmd(e.target.value)}
              className={`${fieldClass} font-mono`}
            />
          </div>

          <div className='space-y-1.5'>
            <label className='text-xs font-medium text-white/50' htmlFor='build-cmd'>
              Build command
            </label>
            <input
              id='build-cmd'
              value={buildCmd}
              disabled={locked}
              onChange={(e) => setBuildCmd(e.target.value)}
              className={`${fieldClass} font-mono`}
            />
          </div>
        </div>

        <div className='space-y-1.5'>
          <label className='text-xs font-medium text-white/50' htmlFor='run-cmd'>
            Run command{' '}
            <span className='text-white/25'>{dynamic ? '(starts the app)' : '(not used by static builds)'}</span>
          </label>
          <input
            id='run-cmd'
            value={runCmd}
            disabled={locked}
            placeholder={dynamic ? 'npm start' : 'Not used - the built site is served as static files'}
            onChange={(e) => setRunCmd(e.target.value)}
            className={`${fieldClass} font-mono`}
          />
          {dynamic && (
            <p className='text-[11px] text-white/35'>
              The container stays up and traffic is proxied to it. Listen on the port in{' '}
              <code className='font-mono text-white/50'>$PORT</code> ({stack?.port ?? 3000}).
            </p>
          )}
        </div>
      </section>

      <div className='flex items-center justify-between gap-4'>
        <p className='text-xs text-white/35'>
          {canDeploy
            ? 'Ready to deploy. The job goes to the build queue.'
            : 'Fill in every field above to enable deploying.'}
        </p>
        <button
          onClick={handleDeploy}
          disabled={!canDeploy}
          className='inline-flex items-center gap-2 rounded-lg bg-white px-5 py-2.5 text-sm font-semibold text-black transition-all hover:bg-white/90 active:scale-95 disabled:cursor-not-allowed disabled:bg-white/20 disabled:text-white/40 disabled:active:scale-100'
        >
          {submitting ? <Loader2 className='size-4 animate-spin' /> : <Rocket className='size-4' />}
          {submitting ? 'Queueing...' : 'Deploy'}
        </button>
      </div>
    </div>
  );
};

export default Deploy;

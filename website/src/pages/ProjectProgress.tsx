import { useCallback, useEffect, useRef, useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import {
  ArrowLeft,
  Check,
  CheckCircle2,
  ChevronRight,
  Copy,
  ExternalLink,
  Loader2,
  RotateCcw,
  Terminal,
  XCircle,
} from 'lucide-react';
import { apiUrl } from '@/utils/contants';

type Status = 'queued' | 'pending' | 'deployed' | 'failed';

interface StreamEvent {
  type: 'status' | 'log';
  text?: string;
  status?: Status;
  url?: string;
  name?: string;
  repo_url?: string;
}

const steps: { status: Status; label: string }[] = [
  { status: 'queued', label: 'Queued' },
  { status: 'pending', label: 'Building' },
  { status: 'deployed', label: 'Live' },
];

// How far the build has got, as an index into steps. A failed build stopped at Building.
const stepIndex = (status: Status) => (status === 'failed' ? 1 : steps.findIndex((s) => s.status === status));

const formatElapsed = (seconds: number) => {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${String(s).padStart(2, '0')}`;
};

// The worker prefixes the commands it runs with "> ", and those lines are the ones worth
// picking out of a few hundred lines of install output.
const lineTone = (line: string) => {
  if (line.startsWith('> ')) return 'text-indigo-300';
  if (/\b(error|failed|fatal|ERR!)\b/i.test(line)) return 'text-red-300';
  if (/\b(success|succeeded|built in|live at)\b/i.test(line)) return 'text-emerald-300';
  return 'text-zinc-400';
};

const ProjectProgress = () => {
  const { id } = useParams<{ id: string }>();

  const [status, setStatus] = useState<Status>('queued');
  const [name, setName] = useState('');
  const [repoUrl, setRepoUrl] = useState('');
  const [deployedUrl, setDeployedUrl] = useState('');
  const [logs, setLogs] = useState<string[]>([]);
  const [streamError, setStreamError] = useState<string | null>(null);
  const [elapsed, setElapsed] = useState(0);
  const [copied, setCopied] = useState(false);
  const [showLogs, setShowLogs] = useState(false);

  const logBoxRef = useRef<HTMLDivElement>(null);
  // Only pull the log along if the user is already at the bottom, so scrolling back
  // through the output is not fought by every new line
  const followRef = useRef(true);

  const building = status === 'queued' || status === 'pending';
  const logsOpen = status === 'deployed' ? showLogs : true;

  useEffect(() => {
    if (!building) return;
    const timer = setInterval(() => setElapsed((e) => e + 1), 1000);
    return () => clearInterval(timer);
  }, [building]);

  useEffect(() => {
    const box = logBoxRef.current;
    if (box && followRef.current) box.scrollTop = box.scrollHeight;
  }, [logs, logsOpen]);

  const onLogScroll = useCallback(() => {
    const box = logBoxRef.current;
    if (!box) return;
    followRef.current = box.scrollHeight - box.scrollTop - box.clientHeight < 40;
  }, []);

  useEffect(() => {
    let active = true;
    let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;
    let retry: ReturnType<typeof setTimeout> | undefined;
    let done = false;

    async function connect() {
      try {
        const response = await fetch(`${apiUrl}deployments/${id}/stream`, {
          method: 'GET',
          credentials: 'include',
        });

        if (response.status === 404 || response.status === 401) {
          done = true;
          setStreamError(
            response.status === 404 ? 'This deployment no longer exists.' : 'Please sign in again to follow this build.'
          );
          return;
        }
        if (!response.ok || !response.body) {
          throw new Error(`the server answered ${response.status}`);
        }

        // Every connection replays the build log from the start, so the view is rebuilt
        // rather than appended to - a reconnect cannot double up lines
        if (!active) return;
        setLogs([]);
        setStreamError(null);
        followRef.current = true;

        reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';

        while (active) {
          const { value, done: finished } = await reader.read();
          if (finished) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split('\n');
          buffer = lines.pop() ?? '';

          const batch: string[] = [];
          for (const line of lines) {
            if (!line.trim()) continue;
            let event: StreamEvent;
            try {
              event = JSON.parse(line);
            } catch {
              continue;
            }

            if (event.type === 'status' && event.status) {
              setStatus(event.status);
              if (event.url) setDeployedUrl(event.url);
              if (event.name) setName(event.name);
              if (event.repo_url) setRepoUrl(event.repo_url);
              if (event.status === 'deployed' || event.status === 'failed') done = true;
            } else if (event.type === 'log' && event.text !== undefined) {
              batch.push(event.text);
            }
          }
          // One state update per chunk rather than per line
          if (batch.length) setLogs((prev) => [...prev, ...batch]);
        }
      } catch (err) {
        if (!active) return;
        setStreamError(err instanceof Error ? err.message : 'the connection dropped');
      }

      // The stream only ends on its own once the build is over. Anything else is a
      // dropped connection while the build is still running, so pick it back up.
      if (active && !done) {
        retry = setTimeout(connect, 2000);
      }
    }

    connect();

    return () => {
      active = false;
      if (retry) clearTimeout(retry);
      reader?.cancel().catch(() => {});
    };
  }, [id]);

  const copyUrl = async () => {
    try {
      await navigator.clipboard.writeText(deployedUrl);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  };

  const title = name || `Deployment #${id}`;
  const current = stepIndex(status);

  return (
    <div className="min-h-screen bg-black text-white">
      <div className="mx-auto w-full max-w-3xl px-5 py-10 space-y-6">
        <Link
          to="/dashboard"
          className="group inline-flex items-center gap-2 text-sm font-medium text-white/50 transition-colors hover:text-white"
        >
          <ArrowLeft className="size-4 transition-transform group-hover:-translate-x-0.5" />
          Back to dashboard
        </Link>

        {status === 'deployed' ? (
          /* Live: the build is history, the URL is the point of the page */
          <section className="rounded-2xl border border-emerald-500/20 bg-emerald-500/[0.06] p-8 text-center">
            <div className="mx-auto flex size-12 items-center justify-center rounded-full border border-emerald-500/30 bg-emerald-500/10">
              <CheckCircle2 className="size-6 text-emerald-400" />
            </div>
            <h1 className="mt-5 text-2xl font-bold tracking-tight">{title} is live</h1>
            <p className="mt-1.5 text-sm text-white/50">Your site is deployed and serving traffic.</p>

            <div className="mt-6 flex items-center gap-2 rounded-xl border border-white/10 bg-black/40 p-2 pl-4 text-left">
              <a
                href={deployedUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="flex-1 truncate font-mono text-sm text-emerald-300 hover:underline"
              >
                {deployedUrl}
              </a>
              <button
                onClick={copyUrl}
                title="Copy URL"
                className="flex items-center gap-1.5 rounded-lg border border-white/10 px-2.5 py-1.5 text-xs font-medium text-white/60 transition-colors hover:bg-white/5 hover:text-white"
              >
                {copied ? <Check className="size-3.5 text-emerald-400" /> : <Copy className="size-3.5" />}
                {copied ? 'Copied' : 'Copy'}
              </button>
            </div>

            <div className="mt-5 flex flex-col gap-2 sm:flex-row sm:justify-center">
              <a
                href={deployedUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center justify-center gap-2 rounded-lg bg-white px-5 py-2.5 text-sm font-bold text-black transition-all hover:bg-zinc-200 active:scale-[0.98]"
              >
                <ExternalLink className="size-4" />
                Visit site
              </a>
              <Link
                to="/dashboard/projects"
                className="inline-flex items-center justify-center gap-2 rounded-lg border border-white/10 px-5 py-2.5 text-sm font-semibold text-white/70 transition-colors hover:bg-white/5 hover:text-white"
              >
                All projects
              </Link>
            </div>
          </section>
        ) : status === 'failed' ? (
          <section className="rounded-2xl border border-red-500/20 bg-red-500/[0.06] p-6">
            <div className="flex items-start gap-4">
              <div className="flex size-10 shrink-0 items-center justify-center rounded-full border border-red-500/30 bg-red-500/10">
                <XCircle className="size-5 text-red-400" />
              </div>
              <div className="min-w-0 flex-1">
                <h1 className="text-lg font-bold tracking-tight">Build failed</h1>
                <p className="mt-1 text-sm text-white/50">
                  {title} could not be built. The build log below ends with the reason.
                </p>
                <Link
                  to={`/dashboard/deploy?name=${encodeURIComponent(name)}&repo_url=${encodeURIComponent(repoUrl)}`}
                  className="mt-4 inline-flex items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm font-semibold text-white/80 transition-colors hover:bg-white/10 hover:text-white"
                >
                  <RotateCcw className="size-3.5" />
                  Deploy again
                </Link>
              </div>
            </div>
          </section>
        ) : (
          /* Building: what is happening, how long it has taken */
          <section className="rounded-2xl border border-white/10 bg-white/[0.03] p-6">
            <div className="flex items-center justify-between gap-4">
              <div className="flex min-w-0 items-center gap-3">
                <Loader2 className="size-5 shrink-0 animate-spin text-indigo-400" />
                <div className="min-w-0">
                  <h1 className="truncate text-lg font-bold tracking-tight">
                    {status === 'queued' ? `${title} is queued` : `Building ${title}`}
                  </h1>
                  <p className="text-sm text-white/50">
                    {status === 'queued'
                      ? 'Waiting for a free build worker.'
                      : 'Cloning the repository, installing dependencies, running the build.'}
                  </p>
                </div>
              </div>
              <span className="shrink-0 font-mono text-sm tabular-nums text-white/40">{formatElapsed(elapsed)}</span>
            </div>

            <ol className="mt-6 flex items-center gap-2">
              {steps.map((step, i) => (
                <li key={step.status} className="flex flex-1 items-center gap-2">
                  <div className="flex-1">
                    <div
                      className={`h-1 rounded-full transition-colors ${
                        i < current ? 'bg-indigo-400' : i === current ? 'bg-indigo-400/40' : 'bg-white/10'
                      }`}
                    />
                    <span
                      className={`mt-2 block text-xs font-medium transition-colors ${
                        i <= current ? 'text-white/70' : 'text-white/30'
                      }`}
                    >
                      {step.label}
                    </span>
                  </div>
                </li>
              ))}
            </ol>
          </section>
        )}

        {streamError && (
          <p className="rounded-lg border border-yellow-500/20 bg-yellow-500/[0.06] px-4 py-3 text-sm text-yellow-300/80">
            {streamError}
            {building && ' Reconnecting...'}
          </p>
        )}

        {/* A finished deployment keeps its log out of the way until it is asked for */}
        <section>
          {status === 'deployed' && (
            <button
              onClick={() => setShowLogs((v) => !v)}
              className="flex w-full items-center gap-2 rounded-xl border border-white/10 bg-white/[0.03] px-4 py-3 text-left text-sm font-medium text-white/60 transition-colors hover:bg-white/[0.06] hover:text-white"
            >
              <ChevronRight className={`size-4 transition-transform ${showLogs ? 'rotate-90' : ''}`} />
              <Terminal className="size-4" />
              Build log
              <span className="ml-auto font-mono text-xs text-white/30">{logs.length} lines</span>
            </button>
          )}

          {logsOpen && (
            <div className={status === 'deployed' ? 'mt-2' : ''}>
              <div className="flex items-center justify-between rounded-t-xl border-x border-t border-white/10 bg-white/[0.04] px-4 py-2.5">
                <div className="flex items-center gap-2 text-white/60">
                  <Terminal className="size-3.5" />
                  <span className="font-mono text-xs">build log</span>
                </div>
                <span className="font-mono text-xs text-white/30">{logs.length} lines</span>
              </div>
              <div
                ref={logBoxRef}
                onScroll={onLogScroll}
                className="h-[420px] overflow-y-auto rounded-b-xl border-x border-b border-white/10 bg-[#070707] px-4 py-3 font-mono text-xs leading-6"
              >
                {logs.length === 0 ? (
                  <p className="text-zinc-600">
                    {building ? 'Waiting for the build to start...' : 'No build output was recorded.'}
                  </p>
                ) : (
                  logs.map((line, i) => (
                    <div key={i} className={`whitespace-pre-wrap break-all ${lineTone(line)}`}>
                      {line || ' '}
                    </div>
                  ))
                )}
              </div>
            </div>
          )}
        </section>
      </div>
    </div>
  );
};

export default ProjectProgress;

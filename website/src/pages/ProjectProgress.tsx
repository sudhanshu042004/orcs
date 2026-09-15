import { useEffect, useRef, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { apiUrl } from "../utils/contants";
import { Loader2, Terminal, CheckCircle2, XCircle, ArrowLeft, Clock, ExternalLink } from "lucide-react";

interface LogMessage {
  type: string;
  text?: string;
  status?: string;
  url?: string;
}

const ProjectProgress = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [status, setStatus] = useState<string>("queued");
  const [deployedUrl, setDeployedUrl] = useState<string>("");
  const [logs, setLogs] = useState<string[]>([]);
  const terminalEndRef = useRef<HTMLDivElement>(null);
  const logContainerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight;
    }
  }, [logs]);

  useEffect(() => {
    let active = true;
    let reader: ReadableStreamDefaultReader<Uint8Array> | undefined;

    async function streamLogs() {
      try {
        const response = await fetch(`${apiUrl}deployments/${id}/stream`, {
          method: "GET",
          credentials: "include",
        });

        if (!response.ok) {
          throw new Error(`Failed to connect to stream: ${response.statusText}`);
        }

        if (!response.body) {
          throw new Error("No response body available for streaming");
        }

        reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = "";

        while (active) {
          const { value, done } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const lines = buffer.split("\n");
          buffer = lines.pop() || "";

          for (const line of lines) {
            if (!line.trim()) continue;
            try {
              const msg: LogMessage = JSON.parse(line);
              if (!active) break;

              if (msg.type === "status" && msg.status) {
                setStatus(msg.status);
                if (msg.url) setDeployedUrl(msg.url);
              } else if (msg.type === "log" && msg.text !== undefined) {
                setLogs((prev) => [...prev, msg.text as string]);
              }
            } catch (err) {
              console.error("Error parsing NDJSON line:", err);
            }
          }
        }
      } catch (err: any) {
        console.error("Stream error:", err);
        setLogs((prev) => [...prev, `[System Error] Connection lost: ${err.message}`]);
        setStatus("failed");
      }
    }

    streamLogs();

    return () => {
      active = false;
      if (reader) {
        reader.cancel().catch(() => {});
      }
    };
  }, [id]);

  const getStatusDisplay = () => {
    switch (status) {
      case "deployed":
        return {
          icon: <CheckCircle2 className="size-8 text-emerald-400 animate-bounce" />,
          title: "Deployment Successful",
          desc: "Your project has been compiled and published.",
          bg: "bg-emerald-500/10 border-emerald-500/20 text-emerald-400",
        };
      case "failed":
        return {
          icon: <XCircle className="size-8 text-red-400" />,
          title: "Build Failed",
          desc: "An error occurred during cloning or compiling the repository.",
          bg: "bg-red-500/10 border-red-500/20 text-red-400",
        };
      case "queued":
        return {
          icon: <Clock className="size-8 text-sky-400 animate-pulse" />,
          title: "Waiting in the build queue",
          desc: "The job is queued. A worker will pick it up shortly.",
          bg: "bg-sky-500/10 border-sky-500/20 text-sky-400",
        };
      default:
        return {
          icon: <Loader2 className="size-8 text-indigo-400 animate-spin" />,
          title: "Building Project...",
          desc: "Spinning up the container, cloning the repository, and installing dependencies.",
          bg: "bg-indigo-500/10 border-indigo-500/20 text-indigo-400",
        };
    }
  };

  const currentStatus = getStatusDisplay();

  return (
    <div className="min-h-screen bg-black text-white p-6 font-sans flex flex-col items-center justify-center">
      {/* Container */}
      <div className="w-full max-w-4xl space-y-6">
        
        {/* Back Button */}
        <button
          onClick={() => navigate("/dashboard")}
          className="group flex items-center gap-2 rounded-lg border border-white/10 bg-white/5 px-4 py-2 text-sm font-medium text-white/60 transition-all hover:bg-white/10 hover:text-white"
        >
          <ArrowLeft className="size-4 group-hover:-translate-x-0.5 transition-transform" />
          Back to Dashboard
        </button>

        {/* Status Card */}
        <div className={`flex flex-col sm:flex-row items-center gap-4 rounded-2xl border p-6 backdrop-blur-md shadow-2xl ${currentStatus.bg}`}>
          <div>{currentStatus.icon}</div>
          <div className="text-center sm:text-left">
            <h2 className="text-xl font-bold tracking-tight">{currentStatus.title}</h2>
            <p className="mt-1 text-sm opacity-70">{currentStatus.desc}</p>
            {status === "deployed" && deployedUrl && (
              <a
                href={deployedUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="mt-2 inline-flex items-center gap-1.5 text-xs font-semibold underline underline-offset-4 hover:opacity-80"
              >
                <ExternalLink className="size-3.5" />
                {deployedUrl}
              </a>
            )}
          </div>
        </div>

        {/* Terminal Header */}
        <div className="rounded-t-2xl border-t border-x border-white/10 bg-zinc-900/60 p-4 flex items-center justify-between shadow-lg">
          <div className="flex items-center gap-2 text-white/80">
            <Terminal className="size-4 text-indigo-400" />
            <span className="text-xs font-mono font-semibold">build-stream-{id}.log</span>
          </div>
          <div className="flex items-center gap-1.5">
            <span className="size-2.5 rounded-full bg-red-500/60" />
            <span className="size-2.5 rounded-full bg-yellow-500/60" />
            <span className="size-2.5 rounded-full bg-emerald-500/60" />
          </div>
        </div>

        {/* Terminal Console */}
        <div
          ref={logContainerRef}
          className="h-[480px] overflow-y-auto rounded-b-2xl border-b border-x border-white/10 bg-[#060606] p-6 shadow-2xl font-mono text-xs leading-relaxed text-zinc-400 select-text scrollbar-thin scrollbar-thumb-white/10 scrollbar-track-transparent"
        >
          {logs.length === 0 ? (
            <p className="text-zinc-600 italic animate-pulse">Initializing log listener...</p>
          ) : (
            <div className="space-y-1">
              {logs.map((log, idx) => (
                <div key={idx} className="whitespace-pre-wrap break-all hover:bg-white/[0.02] px-1 rounded">
                  <span className="text-zinc-600 mr-2 select-none">[{idx + 1}]</span>
                  <span>{log}</span>
                </div>
              ))}
              <div ref={terminalEndRef} />
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ProjectProgress;

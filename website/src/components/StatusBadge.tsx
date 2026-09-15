import type { Project } from '@/context/ProjectsContext';

const statusStyles: Record<Project['status'], string> = {
  deployed: 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400',
  pending: 'bg-yellow-500/10 border-yellow-500/20 text-yellow-400',
  queued: 'bg-sky-500/10 border-sky-500/20 text-sky-400',
  failed: 'bg-red-500/10 border-red-500/20 text-red-400',
};

const StatusBadge = ({ status, className = '' }: { status: Project['status']; className?: string }) => (
  <span className={`rounded-full border font-semibold ${statusStyles[status] ?? statusStyles.queued} ${className}`}>
    {status}
  </span>
);

export default StatusBadge;

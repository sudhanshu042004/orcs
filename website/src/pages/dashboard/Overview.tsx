import { Link } from 'react-router-dom';
import { FolderOpen } from 'lucide-react';
import { useProjects } from '@/hooks/useProjects';

const Overview = () => {
  const { projects } = useProjects();

  const stats = [
    { label: 'Total Projects', value: String(projects.length), sub: 'Imported applications' },
    { label: 'Deployed', value: String(projects.filter((p) => p.status === 'deployed').length), sub: 'Live environments' },
    { label: 'In progress', value: String(projects.filter((p) => p.status === 'queued' || p.status === 'pending').length), sub: 'Queued or building' },
  ];

  return (
    <>
      {/* Stats row */}
      <div className='mb-8 grid grid-cols-3 gap-4'>
        {stats.map((stat) => (
          <div key={stat.label} className='rounded-xl border border-white/10 bg-white/5 px-5 py-4'>
            <div className='text-2xl font-bold'>{stat.value}</div>
            <div className='mt-1 text-sm font-medium'>{stat.label}</div>
            <div className='mt-0.5 text-xs text-white/40'>{stat.sub}</div>
          </div>
        ))}
      </div>

      {/* Import call to action */}
      <div className='flex flex-col items-center justify-center rounded-2xl border border-dashed border-white/20 bg-white/5 p-12 text-center transition-all hover:bg-white/10'>
        <div className='mb-4 flex size-12 items-center justify-center rounded-full bg-white/10 text-white/60'>
          <FolderOpen className='size-6 text-white/40' />
        </div>
        <h3 className='text-lg font-semibold text-white/90'>
          {projects.length === 0 ? 'No projects imported yet' : 'Import another project'}
        </h3>
        <p className='mt-2 text-sm text-white/40 max-w-sm'>
          Start by importing a repository from your GitHub account or uploading a local folder.
        </p>
        <Link
          to='/dashboard/import'
          className='mt-6 rounded-lg bg-white px-5 py-2.5 text-sm font-semibold text-black transition-all hover:bg-white/90 active:scale-95'
        >
          Import Project
        </Link>
      </div>
    </>
  );
};

export default Overview;

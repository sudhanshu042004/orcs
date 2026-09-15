const steps = [
  {
    title: 'Import a project',
    body: 'Pick a repository from your GitHub account or upload a local folder from the Import page.',
  },
  {
    title: 'Watch the build',
    body: 'Every import opens a live build log at /project/<id>, streamed straight from the build container.',
  },
  {
    title: 'Manage deployments',
    body: 'The Projects page lists every deployment. Deleting one removes its build container, local files and uploaded assets.',
  },
];

const Help = () => (
  <div className='space-y-6'>
    <div>
      <h2 className='text-xl font-semibold'>Help</h2>
      <p className='text-sm text-white/40'>How deploying with ORCS works</p>
    </div>

    <div className='divide-y divide-white/5 rounded-xl border border-white/10 bg-white/5 overflow-hidden'>
      {steps.map((step, index) => (
        <div key={step.title} className='flex gap-4 px-5 py-4'>
          <span className='flex size-6 shrink-0 items-center justify-center rounded-full bg-white/10 text-[11px] font-semibold'>
            {index + 1}
          </span>
          <div>
            <h3 className='text-sm font-semibold text-white/90'>{step.title}</h3>
            <p className='mt-1 text-xs text-white/40'>{step.body}</p>
          </div>
        </div>
      ))}
    </div>
  </div>
);

export default Help;

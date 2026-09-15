import { useAuth } from '@/hooks/useAuth';

const Settings = () => {
  const { user } = useAuth();

  return (
    <div className='space-y-6'>
      <div>
        <h2 className='text-xl font-semibold'>Settings</h2>
        <p className='text-sm text-white/40'>Your account details</p>
      </div>

      <div className='divide-y divide-white/5 rounded-xl border border-white/10 bg-white/5 overflow-hidden'>
        {[
          { label: 'Name', value: user?.name ?? '-' },
          { label: 'Email', value: user?.email ?? '-' },
        ].map((row) => (
          <div key={row.label} className='flex items-center justify-between gap-4 px-5 py-4'>
            <span className='text-sm text-white/50'>{row.label}</span>
            <span className='text-sm font-medium text-white/90 truncate'>{row.value}</span>
          </div>
        ))}
      </div>
    </div>
  );
};

export default Settings;

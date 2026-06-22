import { useState } from 'react';
import { UserAvatar } from '@/components/UserAvatar';
import { useAuth } from '../hooks/useAuth';
import GetFile from '@/components/GetFile';
import GithubRepos from '@/components/GithubRepos';
import {
  LayoutDashboard,
  FolderOpen,
  Settings,
  HelpCircle,
  LogOut,
  BookOpen,
} from 'lucide-react';

const navItems = [
  { icon: LayoutDashboard, label: 'Dashboard', active: true },
  { icon: FolderOpen, label: 'Projects', active: false },
  { icon: Settings, label: 'Settings', active: false },
  { icon: HelpCircle, label: 'Help', active: false },
];

const Dashboard = () => {
  const { user } = useAuth();
  const [activeNav, setActiveNav] = useState('Dashboard');
  const [uploadMode, setUploadMode] = useState<'github' | 'folder'>('github');
  const [showImport, setShowImport] = useState(false);

  return (
    <div className='flex h-screen overflow-hidden bg-black text-white font-sans'>
      {/* Sidebar */}
      <aside className='flex w-64 flex-col border-r border-white/10 bg-black'>
        {/* Logo */}
        <div className='flex h-16 items-center gap-2 border-b border-white/10 px-6'>
          <div className='flex size-8 items-center justify-center rounded-lg bg-white'>
            <span className='text-sm font-bold text-black'>OR</span>
          </div>
          <span className='text-lg font-semibold tracking-tight'>ORCS</span>
        </div>

        {/* Nav */}
        <nav className='flex-1 space-y-1 px-3 py-4'>
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = activeNav === item.label;
            return (
              <button
                key={item.label}
                onClick={() => setActiveNav(item.label)}
                className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all ${
                  isActive
                    ? 'bg-white/10 text-white'
                    : 'text-white/50 hover:bg-white/5 hover:text-white/80'
                }`}
              >
                <Icon className='size-4' />
                {item.label}
              </button>
            );
          })}
        </nav>

        {/* Bottom section */}
        <div className='border-t border-white/10 px-3 py-4'>
          <button className='flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-white/50 hover:bg-white/5 hover:text-white/80 transition-all'>
            <LogOut className='size-4' />
            Sign Out
          </button>
        </div>
      </aside>

      {/* Main content */}
      <div className='flex flex-1 flex-col overflow-hidden'>
        {/* Header */}
        <header className='flex h-16 items-center justify-between border-b border-white/10 bg-black px-8'>
          <div>
            <h1 className='text-xl font-semibold'>Welcome back, {user?.name}</h1>
            <p className='text-sm text-white/40'>Manage your React projects below</p>
          </div>
          <div className='flex items-center gap-4'>
            <a
              href='https://github.com'
              target='_blank'
              rel='noopener noreferrer'
              className='text-white/40 hover:text-white transition-colors'
            >
              <BookOpen className='size-5' />
            </a>
            <UserAvatar />
          </div>
        </header>

        {/* Page content */}
        <main className='flex-1 overflow-y-auto p-8'>
          {/* Stats row */}
          <div className='mb-8 grid grid-cols-3 gap-4'>
            {[
              { label: 'Total Projects', value: '0', sub: 'Upload your first project' },
              { label: 'Analyzed', value: '0', sub: 'Files processed' },
              { label: 'Components', value: '0', sub: 'Components detected' },
            ].map((stat) => (
              <div
                key={stat.label}
                className='rounded-xl border border-white/10 bg-white/5 px-5 py-4'
              >
                <div className='text-2xl font-bold'>{stat.value}</div>
                <div className='mt-1 text-sm font-medium'>{stat.label}</div>
                <div className='mt-0.5 text-xs text-white/40'>{stat.sub}</div>
              </div>
            ))}
          </div>

          {/* Upload section */}
          {!showImport ? (
            <div className='flex flex-col items-center justify-center rounded-2xl border border-dashed border-white/20 bg-white/5 p-12 text-center transition-all hover:bg-white/10'>
              <div className='mb-4 flex size-12 items-center justify-center rounded-full bg-white/10 text-white/60'>
                <FolderOpen className='size-6 text-white/40' />
              </div>
              <h3 className='text-lg font-semibold text-white/90'>No projects imported yet</h3>
              <p className='mt-2 text-sm text-white/40 max-w-sm'>
                Start by importing a repository from your GitHub account or uploading a local folder.
              </p>
              <button
                onClick={() => setShowImport(true)}
                className='mt-6 rounded-lg bg-white px-5 py-2.5 text-sm font-semibold text-black transition-all hover:bg-white/90 active:scale-95'
              >
                Import Project
              </button>
            </div>
          ) : (
            <div className='rounded-2xl border border-white/10 bg-white/5 p-8 relative'>
              <button 
                onClick={() => setShowImport(false)}
                className="absolute top-8 right-8 text-xs font-medium text-white/40 hover:text-white transition-colors"
              >
                Cancel
              </button>
              <div className='mb-6 flex items-center justify-between'>
                <div>
                  <h2 className='text-lg font-semibold'>Import Project</h2>
                  <p className='mt-1 text-sm text-white/40'>
                    Select a GitHub repository or upload a local folder
                  </p>
                </div>
                
                {/* Toggle */}
                <div className="flex rounded-lg bg-black/50 p-1 border border-white/10 mr-16">
                  <button
                    onClick={() => setUploadMode('github')}
                    className={`rounded-md px-4 py-2 text-sm font-medium transition-all ${
                      uploadMode === 'github'
                        ? 'bg-white/10 text-white shadow-sm'
                        : 'text-white/40 hover:text-white/80'
                    }`}
                  >
                    GitHub Repos
                  </button>
                  <button
                    onClick={() => setUploadMode('folder')}
                    className={`rounded-md px-4 py-2 text-sm font-medium transition-all ${
                      uploadMode === 'folder'
                        ? 'bg-white/10 text-white shadow-sm'
                        : 'text-white/40 hover:text-white/80'
                    }`}
                  >
                    Local Folder
                  </button>
                </div>
              </div>
              
              {uploadMode === 'github' ? <GithubRepos /> : <GetFile />}
            </div>
          )}
        </main>
      </div>
    </div>
  );
};

export default Dashboard;
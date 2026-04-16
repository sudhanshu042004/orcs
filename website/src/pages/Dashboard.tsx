import { useState } from 'react';
import { UserAvatar } from '@/components/UserAvatar';
import { useAuth } from '../hooks/useAuth';
import GetFile from '@/components/GetFile';
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
          <div className='rounded-2xl border border-white/10 bg-white/5 p-8'>
            <div className='mb-6'>
              <h2 className='text-lg font-semibold'>Upload Project</h2>
              <p className='mt-1 text-sm text-white/40'>
                Select a React project folder to get started
              </p>
            </div>
            <GetFile />
          </div>
        </main>
      </div>
    </div>
  );
};

export default Dashboard;
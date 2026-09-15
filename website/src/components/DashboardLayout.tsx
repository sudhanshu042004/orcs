import { Link, NavLink, Outlet } from 'react-router-dom';
import { UserAvatar } from '@/components/UserAvatar';
import StatusBadge from '@/components/StatusBadge';
import { useAuth } from '@/hooks/useAuth';
import { useProjects } from '@/hooks/useProjects';
import { ProjectsProvider } from '@/context/ProjectsContext';
import {
  LayoutDashboard,
  FolderOpen,
  Settings,
  HelpCircle,
  LogOut,
  BookOpen,
  Loader2,
} from 'lucide-react';

const navItems = [
  { icon: LayoutDashboard, label: 'Dashboard', to: '/dashboard', end: true },
  { icon: FolderOpen, label: 'Projects', to: '/dashboard/projects', end: false },
  { icon: Settings, label: 'Settings', to: '/dashboard/settings', end: false },
  { icon: HelpCircle, label: 'Help', to: '/dashboard/help', end: false },
];

const DashboardShell = () => {
  const { user, logout } = useAuth();
  const { projects, loading } = useProjects();

  return (
    <div className='flex h-screen overflow-hidden bg-black text-white font-sans'>
      {/* Sidebar */}
      <aside className='flex w-64 flex-col border-r border-white/10 bg-black'>
        {/* Logo */}
        <Link to='/dashboard' className='flex h-16 items-center gap-2 border-b border-white/10 px-6'>
          <div className='flex size-8 items-center justify-center rounded-lg bg-white'>
            <span className='text-sm font-bold text-black'>OR</span>
          </div>
          <span className='text-lg font-semibold tracking-tight'>ORCS</span>
        </Link>

        {/* Nav & Projects list */}
        <div className='flex-1 overflow-y-auto px-3 py-4 space-y-6'>
          <div className='space-y-1'>
            <p className='px-3 text-[10px] font-semibold uppercase tracking-wider text-white/35'>Menu</p>
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.label}
                  to={item.to}
                  end={item.end}
                  className={({ isActive }) =>
                    `flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-all ${
                      isActive
                        ? 'bg-white/10 text-white'
                        : 'text-white/50 hover:bg-white/5 hover:text-white/80'
                    }`
                  }
                >
                  <Icon className='size-4' />
                  {item.label}
                </NavLink>
              );
            })}
          </div>

          <div className='pt-4 border-t border-white/10 space-y-2'>
            <p className='px-3 text-[10px] font-semibold uppercase tracking-wider text-white/35'>Projects</p>
            {loading ? (
              <div className='flex items-center gap-2 px-3 py-1.5 text-xs text-white/30'>
                <Loader2 className='size-3 animate-spin' />
                <span>Loading...</span>
              </div>
            ) : projects.length === 0 ? (
              <p className='px-3 py-1.5 text-xs italic text-white/30'>No projects yet</p>
            ) : (
              <div className='space-y-1 max-h-[220px] overflow-y-auto'>
                {projects.map((project) => (
                  <Link
                    key={project.id}
                    to={`/project/${project.id}`}
                    className='group flex items-center justify-between rounded-lg px-3 py-1.5 text-xs hover:bg-white/5 transition-all'
                  >
                    <span className='font-medium text-white/70 truncate max-w-[130px]' title={project.name}>
                      {project.name}
                    </span>
                    <StatusBadge status={project.status} className='px-1.5 py-0.5 text-[9px]' />
                  </Link>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Bottom section */}
        <div className='border-t border-white/10 px-3 py-4'>
          <button
            onClick={logout}
            className='flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-white/50 hover:bg-white/5 hover:text-white/80 transition-all'
          >
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
          <Outlet />
        </main>
      </div>
    </div>
  );
};

const DashboardLayout = () => (
  <ProjectsProvider>
    <DashboardShell />
  </ProjectsProvider>
);

export default DashboardLayout;

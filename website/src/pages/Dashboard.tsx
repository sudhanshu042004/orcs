import { useEffect, useState } from 'react';
import { UserAvatar } from '@/components/UserAvatar';
import { useAuth } from '../hooks/useAuth';
import GetFile from '@/components/GetFile';
import GithubRepos from '@/components/GithubRepos';
import { get, del } from '@/utils/api';
import {
  LayoutDashboard,
  FolderOpen,
  Settings,
  HelpCircle,
  LogOut,
  BookOpen,
  Loader2,
} from 'lucide-react';

interface Project {
  id: number;
  name: string;
  status: 'draft' | 'building' | 'failed' | 'success';
  url: string | null;
  repo_url: string;
  created_at: string;
}

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
  const [projects, setProjects] = useState<Project[]>([]);
  const [loadingProjects, setLoadingProjects] = useState(true);

  const fetchProjects = async () => {
    try {
      const data = await get<Project[]>('deployments');
      setProjects(data || []);
    } catch (err) {
      console.error('Failed to fetch projects', err);
    } finally {
      setLoadingProjects(false);
    }
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  const handleDeleteProject = async (id: number) => {
    if (!window.confirm("Are you sure you want to delete this project? This will permanently delete the deployment and its cloned files.")) {
      return;
    }

    try {
      await del(`deployments/${id}`);
      fetchProjects();
    } catch (err: any) {
      alert(err.message || "Failed to delete project");
    }
  };

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

        {/* Nav & Projects list */}
        <div className="flex-1 overflow-y-auto px-3 py-4 space-y-6">
          <div className="space-y-1">
            <p className="px-3 text-[10px] font-semibold uppercase tracking-wider text-white/35">Menu</p>
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
          </div>

          <div className="pt-4 border-t border-white/10 space-y-2">
            <p className="px-3 text-[10px] font-semibold uppercase tracking-wider text-white/35">Projects</p>
            {loadingProjects ? (
              <div className="flex items-center gap-2 px-3 py-1.5 text-xs text-white/30">
                <Loader2 className="size-3 animate-spin" />
                <span>Loading...</span>
              </div>
            ) : projects.length === 0 ? (
              <p className="px-3 py-1.5 text-xs italic text-white/30">No projects yet</p>
            ) : (
              <div className="space-y-1 max-h-[220px] overflow-y-auto">
                {projects.map((project) => (
                  <div
                    key={project.id}
                    className="group flex items-center justify-between rounded-lg px-3 py-1.5 text-xs hover:bg-white/5 transition-all"
                  >
                    <span className="font-medium text-white/70 truncate max-w-[130px]" title={project.name}>
                      {project.name}
                    </span>
                    <span className={`rounded-full px-1.5 py-0.5 text-[9px] font-semibold border ${
                      project.status === 'success'
                        ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400'
                        : project.status === 'building'
                          ? 'bg-yellow-500/10 border-yellow-500/20 text-yellow-400'
                          : project.status === 'failed'
                            ? 'bg-red-500/10 border-red-500/20 text-red-400'
                            : 'bg-white/5 border-white/10 text-white/40' // draft
                    }`}>
                      {project.status}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

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
          {activeNav === 'Projects' ? (
            <div className="space-y-6">
              <div className="flex items-center justify-between">
                <div>
                  <h2 className="text-xl font-semibold">Your Projects</h2>
                  <p className="text-sm text-white/40">Manage your deployed applications and drafts</p>
                </div>
              </div>

              {loadingProjects ? (
                <div className="flex h-48 flex-col items-center justify-center gap-3 rounded-xl border border-white/10 bg-white/5 p-6">
                  <Loader2 className="size-8 animate-spin text-emerald-400" />
                  <p className="text-sm text-white/40">Loading projects...</p>
                </div>
              ) : projects.length === 0 ? (
                <div className="flex h-48 flex-col items-center justify-center gap-3 rounded-xl border border-white/10 bg-white/5 p-6 text-center">
                  <FolderOpen className="size-10 text-white/20" />
                  <p className="text-sm text-white/60">No projects imported yet.</p>
                  <button
                    onClick={() => {
                      setActiveNav('Dashboard');
                      setShowImport(true);
                    }}
                    className="mt-2 rounded-lg bg-white px-4 py-2 text-xs font-semibold text-black hover:bg-white/90 transition-all"
                  >
                    Import First Project
                  </button>
                </div>
              ) : (
                <div className="divide-y divide-white/5 rounded-xl border border-white/10 bg-white/5 overflow-hidden">
                  {projects.map((project) => (
                    <div
                      key={project.id}
                      className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 p-5 hover:bg-white/[0.02] transition-colors"
                    >
                      <div className="space-y-1 min-w-0">
                        <div className="flex items-center gap-2.5">
                          <h4 className="text-base font-semibold text-white/90 truncate">{project.name}</h4>
                          <span className={`rounded-full px-2 py-0.5 text-[10px] font-semibold border ${
                            project.status === 'success'
                              ? 'bg-emerald-500/10 border-emerald-500/20 text-emerald-400'
                              : project.status === 'building'
                                ? 'bg-yellow-500/10 border-yellow-500/20 text-yellow-400'
                                : project.status === 'failed'
                                  ? 'bg-red-500/10 border-red-500/20 text-red-400'
                                  : 'bg-white/5 border-white/10 text-white/40' // draft
                          }`}>
                            {project.status}
                          </span>
                        </div>
                        <p className="text-xs text-white/40 truncate max-w-md" title={project.repo_url}>
                          Source: {project.repo_url}
                        </p>
                        {project.url && (
                          <p className="text-[11px] text-white/30 truncate max-w-md" title={project.url}>
                            Path: {project.url}
                          </p>
                        )}
                      </div>

                      <div className="flex items-center gap-3">
                        <button
                          onClick={() => handleDeleteProject(project.id)}
                          className="rounded-lg border border-red-500/20 bg-red-500/5 hover:bg-red-500/10 px-4 py-2 text-xs font-semibold text-red-400 hover:text-red-300 transition-all active:scale-95"
                        >
                          Delete
                        </button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <>
              {/* Stats row */}
              <div className='mb-8 grid grid-cols-3 gap-4'>
                {[
                  { label: 'Total Projects', value: String(projects.length), sub: 'Imported applications' },
                  { label: 'Deployed', value: String(projects.filter(p => p.status === 'success').length), sub: 'Live environments' },
                  { label: 'Drafts', value: String(projects.filter(p => p.status === 'draft').length), sub: 'Folders uploaded' },
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
                  
                  {uploadMode === 'github' ? (
                    <GithubRepos onDeploySuccess={fetchProjects} />
                  ) : (
                    <GetFile onUploadSuccess={fetchProjects} />
                  )}
                </div>
              )}
            </>
          )}
        </main>
      </div>
    </div>
  );
};

export default Dashboard;
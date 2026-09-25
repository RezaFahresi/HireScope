import { NavLink, useNavigate } from 'react-router-dom'
import {
  LayoutDashboard,
  BarChart3,
  Briefcase,
  Users,
  Calendar,
  Settings,
  LogOut,
  ChevronRight,
} from 'lucide-react'
import { useAuth } from '@/features/auth/AuthProvider'
import { cn } from '@/lib/utils'
import { toast } from 'sonner'

const navItems = [
  { to: '/dashboard', icon: LayoutDashboard, label: 'Dashboard' },
  { to: '/analytics', icon: BarChart3, label: 'Analytics' },
  { to: '/jobs', icon: Briefcase, label: 'Jobs' },
  { to: '/candidates', icon: Users, label: 'Candidates' },
  { to: '/interviews', icon: Calendar, label: 'Interviews' },
  { to: '/settings/integrations', icon: Settings, label: 'Settings' },
]

export function Sidebar() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()

  const handleLogout = async () => {
    await logout()
    toast.success('Signed out successfully')
    navigate('/login')
  }

  return (
    <aside className="w-60 flex-shrink-0 bg-white border-r border-slate-200 flex flex-col h-screen">
      {/* Logo */}
      <div className="h-14 flex items-center px-5 border-b border-slate-200 flex-shrink-0">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 bg-primary rounded-md flex items-center justify-center">
            <ChevronRight className="w-4 h-4 text-white" />
          </div>
          <span className="font-mono font-bold text-base text-primary tracking-tight">
            HireScope
          </span>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto py-4 px-3 space-y-1">
        {navItems.map(({ to, icon: Icon, label }) => (
          <NavLink
            key={to}
            to={to}
            className={({ isActive }) =>
              cn(
                'sidebar-link',
                isActive && 'active'
              )
            }
          >
            <Icon className="w-4 h-4 flex-shrink-0" />
            {label}
          </NavLink>
        ))}
      </nav>

      {/* User Footer */}
      <div className="border-t border-slate-200 p-3 flex-shrink-0">
        {(() => {
          const displayName = user?.name?.trim() || user?.email || 'User'
          const initial = (displayName.charAt(0) || 'U').toUpperCase()
          return (
            <div className="flex items-center gap-3 px-2 py-2 rounded-lg mb-1">
              <div className="w-7 h-7 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                <span className="text-xs font-bold text-primary">
                  {initial}
                </span>
              </div>
              <div className="flex-1 min-w-0">
                <p className="text-xs font-semibold text-slate-800 truncate">{displayName}</p>
                <p className="text-xs text-slate-400 truncate">{user?.role || 'Recruiter'}</p>
              </div>
            </div>
          )
        })()}
        <button
          onClick={handleLogout}
          className="sidebar-link w-full text-slate-500 hover:text-red-600 hover:bg-red-50"
        >
          <LogOut className="w-4 h-4" />
          Sign out
        </button>
      </div>
    </aside>
  )
}

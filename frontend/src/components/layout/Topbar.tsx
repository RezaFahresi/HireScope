import { useLocation, Link } from 'react-router-dom'
import { ChevronRight } from 'lucide-react'
import { useAuth } from '@/features/auth/AuthProvider'

const breadcrumbMap: Record<string, string> = {
  dashboard: 'Dashboard',
  jobs: 'Jobs',
  candidates: 'Candidates',
  new: 'New',
  edit: 'Edit',
}

function useBreadcrumbs() {
  const location = useLocation()
  const segments = location.pathname.split('/').filter(Boolean)

  return segments.map((seg, idx) => {
    const path = '/' + segments.slice(0, idx + 1).join('/')
    const isLast = idx === segments.length - 1
    // If segment is a UUID-like string, replace with "Detail"
    const isId = /^[0-9a-f-]{36}$/.test(seg)
    const label = isId ? 'Detail' : (breadcrumbMap[seg] ?? seg)
    return { label, path, isLast }
  })
}

export function Topbar() {
  const { user } = useAuth()
  const breadcrumbs = useBreadcrumbs()

  return (
    <header className="h-14 flex-shrink-0 bg-white border-b border-slate-200 flex items-center justify-between px-6">
      {/* Breadcrumbs */}
      <nav className="flex items-center gap-1 text-sm">
        {breadcrumbs.map((crumb, idx) => (
          <span key={crumb.path} className="flex items-center gap-1">
            {idx > 0 && <ChevronRight className="w-3.5 h-3.5 text-slate-400" />}
            {crumb.isLast ? (
              <span className="font-semibold text-primary">{crumb.label}</span>
            ) : (
              <Link
                to={crumb.path}
                className="text-slate-500 hover:text-primary transition-colors duration-150"
              >
                {crumb.label}
              </Link>
            )}
          </span>
        ))}
      </nav>

      {/* Right side */}
      <div className="flex items-center gap-3">
        <span className="text-xs text-slate-400 hidden sm:block">
          {user?.email}
        </span>
        {(() => {
          const displayName = user?.name?.trim() || user?.email || 'User'
          const initial = (displayName.charAt(0) || 'U').toUpperCase()
          return (
            <div className="w-7 h-7 rounded-full bg-primary/10 flex items-center justify-center">
              <span className="text-xs font-bold text-primary">
                {initial}
              </span>
            </div>
          )
        })()}
      </div>
    </header>
  )
}

import { Link } from 'react-router-dom'
import { Briefcase, ArrowUpRight } from 'lucide-react'

interface JobStatusBreakdown {
  draft: number
  open: number
  closed: number
  archived: number
}

interface JobStatusCardProps {
  statusCounts: JobStatusBreakdown
  totalJobs: number
}

export function JobStatusCard({ statusCounts, totalJobs }: JobStatusCardProps) {
  const statuses = [
    {
      key: 'OPEN',
      label: 'Open / Published',
      count: statusCounts.open,
      color: 'bg-emerald-500',
      badgeBg: 'bg-emerald-50 text-emerald-700 border-emerald-200/60',
      href: '/jobs?status=OPEN',
    },
    {
      key: 'DRAFT',
      label: 'Draft',
      count: statusCounts.draft,
      color: 'bg-amber-500',
      badgeBg: 'bg-amber-50 text-amber-700 border-amber-200/60',
      href: '/jobs?status=DRAFT',
    },
    {
      key: 'CLOSED',
      label: 'Closed',
      count: statusCounts.closed,
      color: 'bg-slate-400',
      badgeBg: 'bg-slate-100 text-slate-700 border-slate-200/60',
      href: '/jobs?status=CLOSED',
    },
    {
      key: 'ARCHIVED',
      label: 'Archived',
      count: statusCounts.archived,
      color: 'bg-zinc-400',
      badgeBg: 'bg-zinc-100 text-zinc-600 border-zinc-200/60',
      href: '/jobs?status=ARCHIVED',
    },
  ]

  return (
    <div className="rounded-lg border border-border/80 bg-card p-5 shadow-xs flex flex-col justify-between">
      <div>
        <div className="flex items-center justify-between pb-3 border-b border-border/50">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-md bg-secondary text-foreground">
              <Briefcase className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <h3 className="text-sm font-semibold text-foreground">Job Status Overview</h3>
              <p className="text-xs text-muted-foreground">Portfolio breakdown across stages</p>
            </div>
          </div>
          <div className="text-right">
            <span className="text-lg font-bold font-mono text-foreground">{totalJobs}</span>
            <span className="text-xs text-muted-foreground ml-1">total</span>
          </div>
        </div>

        <div className="mt-4 space-y-2.5">
          {statuses.map((item) => {
            const pct = totalJobs > 0 ? Math.round((item.count / totalJobs) * 100) : 0

            return (
              <Link
                key={item.key}
                to={item.href}
                className="group flex items-center justify-between p-2 rounded-md hover:bg-muted/40 transition-colors border border-transparent hover:border-border/60"
              >
                <div className="flex items-center gap-2.5 min-w-0">
                  <span className={`h-2 w-2 rounded-full ${item.color} shrink-0`} />
                  <span className="text-xs font-medium text-foreground truncate">{item.label}</span>
                </div>
                <div className="flex items-center gap-2 shrink-0">
                  <div className="w-16 h-1.5 bg-secondary rounded-full overflow-hidden hidden sm:block">
                    <div
                      className={`h-full ${item.color} transition-all duration-300`}
                      style={{ width: `${pct}%` }}
                    />
                  </div>
                  <span className="text-xs font-mono font-medium text-foreground w-6 text-right">
                    {item.count}
                  </span>
                  <span className="text-[11px] text-muted-foreground w-8 text-right font-mono hidden sm:inline-block">
                    {pct}%
                  </span>
                  <ArrowUpRight className="h-3 w-3 text-muted-foreground/60 opacity-0 group-hover:opacity-100 transition-opacity" />
                </div>
              </Link>
            )
          })}
        </div>
      </div>

      <div className="mt-4 pt-3 border-t border-border/50 flex justify-end">
        <Link
          to="/jobs"
          className="text-xs font-medium text-primary hover:underline flex items-center gap-1"
        >
          View all jobs &rarr;
        </Link>
      </div>
    </div>
  )
}

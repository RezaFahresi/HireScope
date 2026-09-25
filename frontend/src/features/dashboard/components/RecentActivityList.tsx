import { Link } from 'react-router-dom'
import {
  Activity,
  UserCheck,
  MessageSquare,
  Briefcase,
  Clock,
  CheckCircle2,
  FileText,
} from 'lucide-react'
import type { DashboardActivity } from '@/types'
import { formatDate } from '@/lib/utils'

interface RecentActivityListProps {
  activities: DashboardActivity[]
  isLoading?: boolean
}

export function RecentActivityList({ activities, isLoading }: RecentActivityListProps) {
  if (isLoading) {
    return (
      <div className="rounded-lg border border-border/80 bg-card p-5 shadow-xs">
        <div className="h-5 w-40 bg-muted rounded animate-pulse mb-4" />
        <div className="space-y-3">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="h-12 bg-muted/50 rounded animate-pulse" />
          ))}
        </div>
      </div>
    )
  }

  const getActionDetails = (act: DashboardActivity) => {
    const action = act.action.toUpperCase()

    if (action.includes('WORKFLOW') || action.includes('STATUS_CHANGE')) {
      return {
        icon: UserCheck,
        iconBg: 'bg-emerald-100 text-emerald-700',
        label: 'Candidate stage updated',
      }
    }
    if (action.includes('NOTE')) {
      return {
        icon: MessageSquare,
        iconBg: 'bg-blue-100 text-blue-700',
        label: 'Recruiter evaluation note',
      }
    }
    if (action.includes('JOB')) {
      return {
        icon: Briefcase,
        iconBg: 'bg-indigo-100 text-indigo-700',
        label: 'Requisition activity',
      }
    }
    if (action.includes('SCREEN') || action.includes('MATCH')) {
      return {
        icon: CheckCircle2,
        iconBg: 'bg-amber-100 text-amber-700',
        label: 'Screening evaluation',
      }
    }
    if (action.includes('CV') || action.includes('DOC')) {
      return {
        icon: FileText,
        iconBg: 'bg-purple-100 text-purple-700',
        label: 'Candidate CV processed',
      }
    }

    return {
      icon: Activity,
      iconBg: 'bg-slate-100 text-slate-700',
      label: act.action.replace(/_/g, ' ').toLowerCase(),
    }
  }

  // Parse metadata if JSON
  const formatMetadata = (raw?: string) => {
    if (!raw) return null
    try {
      const parsed = JSON.parse(raw)
      if (typeof parsed === 'object' && parsed !== null) {
        if (parsed.old_status && parsed.new_status) {
          return `${parsed.old_status} → ${parsed.new_status}`
        }
        if (parsed.note_preview) {
          return `"${parsed.note_preview}"`
        }
        if (parsed.stage) {
          return `Stage: ${parsed.stage}`
        }
        return Object.entries(parsed)
          .map(([k, v]) => `${k}: ${v}`)
          .slice(0, 2)
          .join(', ')
      }
      return String(raw)
    } catch {
      return raw
    }
  }

  return (
    <div className="rounded-lg border border-border/80 bg-card shadow-xs overflow-hidden">
      <div className="flex items-center justify-between p-4 border-b border-border/60">
        <div className="flex items-center gap-2">
          <div className="p-1.5 rounded-md bg-secondary text-foreground">
            <Activity className="h-4 w-4 text-muted-foreground" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-foreground">Recent Audit & Recruiter Activity</h3>
            <p className="text-xs text-muted-foreground">Chronological audit stream of pipeline operations</p>
          </div>
        </div>
        <div className="flex items-center gap-1 text-xs text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>Real-time</span>
        </div>
      </div>

      {activities.length === 0 ? (
        <div className="py-12 px-4 text-center">
          <Activity className="h-8 w-8 text-muted-foreground/40 mx-auto mb-2" />
          <p className="text-xs font-medium text-muted-foreground">No recent activity logged</p>
          <p className="text-[11px] text-muted-foreground/70 mt-0.5">
            Actions such as candidate status updates, notes, and requisition edits will appear here.
          </p>
        </div>
      ) : (
        <div className="divide-y divide-border/40">
          {activities.map((act) => {
            const details = getActionDetails(act)
            const meta = formatMetadata(act.metadata)
            const Icon = details.icon

            return (
              <div
                key={act.id}
                className="p-3.5 px-4 flex items-start gap-3 hover:bg-muted/20 transition-colors"
              >
                <div className={`p-1.5 rounded-md ${details.iconBg} shrink-0 mt-0.5`}>
                  <Icon className="h-3.5 w-3.5" />
                </div>

                <div className="min-w-0 flex-1 text-xs">
                  <div className="flex flex-wrap items-center gap-1.5 text-foreground leading-snug">
                    <span className="font-semibold text-foreground">
                      {act.actor_name || 'System / Recruiter'}
                    </span>
                    <span className="text-muted-foreground">{details.label}</span>
                    {act.candidate_name && (
                      <span className="font-medium text-foreground">
                        for{' '}
                        {act.candidate_id ? (
                          <Link
                            to={`/candidates/${act.candidate_id}`}
                            className="text-primary hover:underline"
                          >
                            {act.candidate_name}
                          </Link>
                        ) : (
                          act.candidate_name
                        )}
                      </span>
                    )}
                    {act.job_title && (
                      <span className="text-muted-foreground">
                        in{' '}
                        {act.job_id ? (
                          <Link
                            to={`/jobs/${act.job_id}`}
                            className="text-foreground hover:text-primary hover:underline font-medium"
                          >
                            {act.job_title}
                          </Link>
                        ) : (
                          act.job_title
                        )}
                      </span>
                    )}
                  </div>

                  {meta && (
                    <div className="mt-1 text-[11px] font-mono text-muted-foreground bg-muted/40 px-2 py-0.5 rounded inline-block max-w-full truncate">
                      {meta}
                    </div>
                  )}
                </div>

                <div className="text-[11px] text-muted-foreground shrink-0 text-right">
                  {act.created_at ? formatDate(act.created_at) : '—'}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

import { Link } from 'react-router-dom'
import { Users, ArrowRight, ExternalLink } from 'lucide-react'
import type { DashboardRecentCandidate, WorkflowStatus } from '@/types'
import { WorkflowStatusBadge, StatusBadge } from '@/components/data-display/StatusBadge'
import { formatDate } from '@/lib/utils'

interface RecentCandidatesTableProps {
  candidates: DashboardRecentCandidate[]
  isLoading?: boolean
}

export function RecentCandidatesTable({ candidates, isLoading }: RecentCandidatesTableProps) {
  if (isLoading) {
    return (
      <div className="rounded-lg border border-border/80 bg-card p-5 shadow-xs">
        <div className="h-5 w-40 bg-muted rounded animate-pulse mb-4" />
        <div className="space-y-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="h-10 bg-muted/50 rounded animate-pulse" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-border/80 bg-card shadow-xs overflow-hidden flex flex-col justify-between">
      <div>
        <div className="flex items-center justify-between p-4 border-b border-border/60">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-md bg-secondary text-foreground">
              <Users className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <h3 className="text-sm font-semibold text-foreground">Recent Candidate Pipeline</h3>
              <p className="text-xs text-muted-foreground">Latest candidates assigned or active</p>
            </div>
          </div>
          <Link
            to="/candidates"
            className="text-xs font-medium text-primary hover:underline flex items-center gap-1"
          >
            View all &rarr;
          </Link>
        </div>

        {candidates.length === 0 ? (
          <div className="py-12 px-4 text-center">
            <Users className="h-8 w-8 text-muted-foreground/40 mx-auto mb-2" />
            <p className="text-xs font-medium text-muted-foreground">No recent candidates found</p>
            <p className="text-[11px] text-muted-foreground/70 mt-0.5">
              Add candidates to job pipelines to see their progress here.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-border/50 bg-muted/20 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  <th className="py-2.5 px-4">Candidate</th>
                  <th className="py-2.5 px-4">Job Role</th>
                  <th className="py-2.5 px-4">Status</th>
                  <th className="py-2.5 px-4 hidden sm:table-cell">Assigned Date</th>
                  <th className="py-2.5 px-4 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/40 text-xs">
                {candidates.map((c) => {
                  const displayName = (c.full_name || 'Candidate').trim()
                  const initial = displayName.charAt(0).toUpperCase() || 'C'
                  const isWorkflowStatus =
                    c.status === 'REVIEW' || c.status === 'SHORTLISTED' || c.status === 'REJECTED'

                  const reviewUrl =
                    c.job_id && c.candidate_id
                      ? `/jobs/${c.job_id}/candidates/${c.candidate_id}/review`
                      : `/candidates/${c.candidate_id || c.id}`

                  return (
                    <tr
                      key={c.id}
                      className="hover:bg-muted/30 transition-colors group"
                    >
                      <td className="py-3 px-4">
                        <div className="flex items-center gap-2.5">
                          <div className="w-7 h-7 rounded-full bg-primary/10 text-primary font-semibold text-xs flex items-center justify-center shrink-0">
                            {initial}
                          </div>
                          <div className="min-w-0">
                            <Link
                              to={`/candidates/${c.candidate_id || c.id}`}
                              className="font-medium text-foreground hover:text-primary hover:underline truncate block"
                            >
                              {displayName}
                            </Link>
                            {c.email && (
                              <span className="text-[11px] text-muted-foreground truncate block">
                                {c.email}
                              </span>
                            )}
                          </div>
                        </div>
                      </td>

                      <td className="py-3 px-4">
                        {c.job_id ? (
                          <Link
                            to={`/jobs/${c.job_id}`}
                            className="text-foreground hover:text-primary hover:underline font-medium truncate block max-w-[180px]"
                            title={c.job_title}
                          >
                            {c.job_title || 'General Pool'}
                          </Link>
                        ) : (
                          <span className="text-muted-foreground">{c.job_title || 'General Pool'}</span>
                        )}
                      </td>

                      <td className="py-3 px-4">
                        {isWorkflowStatus ? (
                          <WorkflowStatusBadge status={c.status as WorkflowStatus} />
                        ) : (
                          <StatusBadge variant="default">{c.status || 'Active'}</StatusBadge>
                        )}
                      </td>

                      <td className="py-3 px-4 text-muted-foreground text-[11px] hidden sm:table-cell">
                        {c.assigned_at ? formatDate(c.assigned_at) : '—'}
                      </td>

                      <td className="py-3 px-4 text-right">
                        <Link
                          to={reviewUrl}
                          className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
                        >
                          Review
                          <ArrowRight className="h-3 w-3" />
                        </Link>
                      </td>
                    </tr>
                  )
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="p-3 border-t border-border/50 bg-muted/10 flex justify-between items-center text-xs text-muted-foreground px-4">
        <span>Showing up to {candidates.length} recent candidates</span>
        <Link to="/candidates" className="hover:text-foreground transition-colors flex items-center gap-1">
          Candidate directory <ExternalLink className="h-3 w-3" />
        </Link>
      </div>
    </div>
  )
}

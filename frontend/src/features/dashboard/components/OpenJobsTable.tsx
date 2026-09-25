import { Link } from 'react-router-dom'
import { Briefcase, ArrowRight, Plus, ExternalLink, MapPin } from 'lucide-react'
import type { DashboardOpenJob } from '@/types'
import { JobStatusBadge } from '@/components/data-display/StatusBadge'
import { formatEmploymentType } from '@/lib/utils'

interface OpenJobsTableProps {
  jobs: DashboardOpenJob[]
  isLoading?: boolean
}

export function OpenJobsTable({ jobs, isLoading }: OpenJobsTableProps) {
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
              <Briefcase className="h-4 w-4 text-muted-foreground" />
            </div>
            <div>
              <h3 className="text-sm font-semibold text-foreground">Active Open Positions</h3>
              <p className="text-xs text-muted-foreground">Open requisitions with candidate pipelines</p>
            </div>
          </div>
          <Link
            to="/jobs?status=OPEN"
            className="text-xs font-medium text-primary hover:underline flex items-center gap-1"
          >
            All active &rarr;
          </Link>
        </div>

        {jobs.length === 0 ? (
          <div className="py-12 px-4 text-center">
            <Briefcase className="h-8 w-8 text-muted-foreground/40 mx-auto mb-2" />
            <p className="text-xs font-medium text-muted-foreground">No open jobs found</p>
            <p className="text-[11px] text-muted-foreground/70 mt-0.5 mb-3">
              Publish draft requisitions to start receiving candidate applications.
            </p>
            <Link
              to="/jobs/new"
              className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-medium rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              Create Requisition
            </Link>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left border-collapse">
              <thead>
                <tr className="border-b border-border/50 bg-muted/20 text-[11px] font-semibold uppercase tracking-wider text-muted-foreground">
                  <th className="py-2.5 px-4">Position</th>
                  <th className="py-2.5 px-4">Department & Type</th>
                  <th className="py-2.5 px-4 text-center">Candidates</th>
                  <th className="py-2.5 px-4">Status</th>
                  <th className="py-2.5 px-4 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-border/40 text-xs">
                {jobs.map((job) => (
                  <tr
                    key={job.id}
                    className="hover:bg-muted/30 transition-colors group"
                  >
                    <td className="py-3 px-4">
                      <div className="min-w-0">
                        <Link
                          to={`/jobs/${job.id}`}
                          className="font-medium text-foreground hover:text-primary hover:underline truncate block"
                        >
                          {job.title}
                        </Link>
                        <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                          <span className="font-mono">{job.code}</span>
                          {job.location && (
                            <span className="flex items-center gap-0.5">
                              &bull; <MapPin className="h-2.5 w-2.5 inline" /> {job.location}
                            </span>
                          )}
                        </div>
                      </div>
                    </td>

                    <td className="py-3 px-4">
                      <div className="text-foreground">{job.department || '—'}</div>
                      <div className="text-[11px] text-muted-foreground">
                        {formatEmploymentType(job.employment_type)}
                      </div>
                    </td>

                    <td className="py-3 px-4 text-center">
                      <Link
                        to={`/jobs/${job.id}/candidates`}
                        className={`inline-flex items-center px-2 py-0.5 rounded-full font-mono text-xs font-semibold ${
                          job.candidates_count > 0
                            ? 'bg-blue-50 text-blue-700 border border-blue-200 hover:bg-blue-100'
                            : 'bg-muted text-muted-foreground hover:bg-muted/80'
                        } transition-colors`}
                        title="View pipeline candidates"
                      >
                        {job.candidates_count}
                      </Link>
                    </td>

                    <td className="py-3 px-4">
                      <JobStatusBadge status={job.status} />
                    </td>

                    <td className="py-3 px-4 text-right">
                      <Link
                        to={`/jobs/${job.id}/candidates`}
                        className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
                      >
                        Pipeline
                        <ArrowRight className="h-3 w-3" />
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div className="p-3 border-t border-border/50 bg-muted/10 flex justify-between items-center text-xs text-muted-foreground px-4">
        <span>Showing {jobs.length} active positions</span>
        <Link to="/jobs" className="hover:text-foreground transition-colors flex items-center gap-1">
          Requisition directory <ExternalLink className="h-3 w-3" />
        </Link>
      </div>
    </div>
  )
}

import { Link } from 'react-router-dom'
import { AlertCircle, Clock, CheckCircle2, ChevronRight, UserCheck } from 'lucide-react'

interface NeedsAttentionProps {
  awaitingReviewCount: number
  openJobsNoCandidates: number
}

export function NeedsAttentionCard({
  awaitingReviewCount,
  openJobsNoCandidates,
}: NeedsAttentionProps) {
  const hasIssues = awaitingReviewCount > 0 || openJobsNoCandidates > 0

  if (!hasIssues) {
    return (
      <div className="rounded-lg border border-emerald-200/80 bg-emerald-50/40 p-4 shadow-xs">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-full bg-emerald-100 text-emerald-700 shrink-0">
            <CheckCircle2 className="h-5 w-5" />
          </div>
          <div>
            <h4 className="text-sm font-semibold text-emerald-950">Pipelines Operational</h4>
            <p className="text-xs text-emerald-800/80 mt-0.5">
              No candidates awaiting review and all open jobs have active applicant pipelines.
            </p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-amber-200/90 bg-amber-50/30 p-4 shadow-xs">
      <div className="flex items-center gap-2 mb-3 pb-2 border-b border-amber-200/60">
        <AlertCircle className="h-4 w-4 text-amber-600 shrink-0" />
        <h4 className="text-xs font-semibold uppercase tracking-wider text-amber-900">
          Action Required
        </h4>
        <span className="text-[11px] text-amber-700/80 ml-auto">Recruiter priority items</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
        {awaitingReviewCount > 0 && (
          <div className="flex items-center justify-between p-3 rounded-md bg-white border border-amber-200/70 shadow-2xs">
            <div className="flex items-center gap-3 min-w-0">
              <div className="p-2 rounded-md bg-amber-100/70 text-amber-800 shrink-0">
                <Clock className="h-4 w-4" />
              </div>
              <div className="min-w-0">
                <div className="flex items-baseline gap-1.5">
                  <span className="text-base font-bold font-mono text-amber-950">
                    {awaitingReviewCount}
                  </span>
                  <span className="text-xs font-medium text-amber-900">
                    {awaitingReviewCount === 1 ? 'candidate' : 'candidates'} awaiting review
                  </span>
                </div>
                <p className="text-[11px] text-muted-foreground truncate">
                  Unprocessed in active job pipelines
                </p>
              </div>
            </div>
            <Link
              to="/jobs"
              className="text-xs font-medium text-amber-800 hover:text-amber-950 flex items-center gap-0.5 ml-2 shrink-0 px-2 py-1 rounded bg-amber-50 hover:bg-amber-100 transition-colors"
            >
              Review
              <ChevronRight className="h-3 w-3" />
            </Link>
          </div>
        )}

        {openJobsNoCandidates > 0 && (
          <div className="flex items-center justify-between p-3 rounded-md bg-white border border-amber-200/70 shadow-2xs">
            <div className="flex items-center gap-3 min-w-0">
              <div className="p-2 rounded-md bg-orange-100/70 text-orange-800 shrink-0">
                <UserCheck className="h-4 w-4" />
              </div>
              <div className="min-w-0">
                <div className="flex items-baseline gap-1.5">
                  <span className="text-base font-bold font-mono text-orange-950">
                    {openJobsNoCandidates}
                  </span>
                  <span className="text-xs font-medium text-orange-900">
                    {openJobsNoCandidates === 1 ? 'open job' : 'open jobs'} with zero applicants
                  </span>
                </div>
                <p className="text-[11px] text-muted-foreground truncate">
                  Need candidate sourcing or publishing
                </p>
              </div>
            </div>
            <Link
              to="/jobs?status=OPEN"
              className="text-xs font-medium text-orange-800 hover:text-orange-950 flex items-center gap-0.5 ml-2 shrink-0 px-2 py-1 rounded bg-orange-50 hover:bg-orange-100 transition-colors"
            >
              View Jobs
              <ChevronRight className="h-3 w-3" />
            </Link>
          </div>
        )}
      </div>
    </div>
  )
}

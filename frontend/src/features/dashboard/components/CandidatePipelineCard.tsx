import { WorkflowStatusBadge } from '@/components/data-display/StatusBadge'

interface CandidatePipelineCardProps {
  pipeline?: {
    review: number
    shortlisted: number
    rejected: number
  }
  isLoading?: boolean
}

export function CandidatePipelineCard({
  pipeline = { review: 0, shortlisted: 0, rejected: 0 },
  isLoading = false,
}: CandidatePipelineCardProps) {
  const { review, shortlisted, rejected } = pipeline
  const total = review + shortlisted + rejected

  const reviewPct = total > 0 ? Math.round((review / total) * 100) : 0
  const shortlistedPct = total > 0 ? Math.round((shortlisted / total) * 100) : 0
  const rejectedPct = total > 0 ? Math.max(0, 100 - reviewPct - shortlistedPct) : 0

  return (
    <div className="card space-y-4">
      <div className="flex items-center justify-between border-b border-slate-100 pb-3">
        <div>
          <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide">Candidate Pipeline</h2>
          <p className="text-xs text-slate-500 mt-0.5">Workflow distribution of active applicants</p>
        </div>
        <span className="text-xs font-semibold text-slate-500 font-mono">
          {total} total
        </span>
      </div>

      {isLoading ? (
        <div className="space-y-3 py-2">
          <div className="h-4 w-full bg-slate-200 animate-pulse rounded" />
          <div className="h-12 w-full bg-slate-100 animate-pulse rounded" />
        </div>
      ) : total === 0 ? (
        <div className="text-center py-6 text-xs text-slate-400">
          No candidates in pipeline yet.
        </div>
      ) : (
        <div className="space-y-4">
          {/* Proportional Segmented Bar */}
          <div className="h-3 w-full bg-slate-100 rounded-full overflow-hidden flex">
            {review > 0 && (
              <div
                style={{ width: `${reviewPct}%` }}
                className="bg-blue-500 h-full transition-all duration-300"
                title={`Review: ${review} (${reviewPct}%)`}
              />
            )}
            {shortlisted > 0 && (
              <div
                style={{ width: `${shortlistedPct}%` }}
                className="bg-green-500 h-full transition-all duration-300"
                title={`Shortlisted: ${shortlisted} (${shortlistedPct}%)`}
              />
            )}
            {rejected > 0 && (
              <div
                style={{ width: `${rejectedPct}%` }}
                className="bg-red-400 h-full transition-all duration-300"
                title={`Rejected: ${rejected} (${rejectedPct}%)`}
              />
            )}
          </div>

          {/* Breakdown Items */}
          <div className="grid grid-cols-3 gap-3 pt-1">
            <div className="p-3 bg-blue-50/50 rounded-lg border border-blue-100">
              <div className="flex items-center justify-between">
                <WorkflowStatusBadge status="REVIEW" />
                <span className="text-xs text-blue-600 font-medium">{reviewPct}%</span>
              </div>
              <p className="text-xl font-mono font-bold text-slate-800 mt-2">{review}</p>
              <p className="text-[11px] text-slate-500 mt-0.5">Awaiting decision</p>
            </div>

            <div className="p-3 bg-green-50/50 rounded-lg border border-green-100">
              <div className="flex items-center justify-between">
                <WorkflowStatusBadge status="SHORTLISTED" />
                <span className="text-xs text-green-600 font-medium">{shortlistedPct}%</span>
              </div>
              <p className="text-xl font-mono font-bold text-slate-800 mt-2">{shortlisted}</p>
              <p className="text-[11px] text-slate-500 mt-0.5">Advanced to next</p>
            </div>

            <div className="p-3 bg-red-50/50 rounded-lg border border-red-100">
              <div className="flex items-center justify-between">
                <WorkflowStatusBadge status="REJECTED" />
                <span className="text-xs text-red-600 font-medium">{rejectedPct}%</span>
              </div>
              <p className="text-xl font-mono font-bold text-slate-800 mt-2">{rejected}</p>
              <p className="text-[11px] text-slate-500 mt-0.5">Not moving forward</p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

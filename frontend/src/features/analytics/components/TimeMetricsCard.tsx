import React from 'react'
import { Clock, Timer, CheckCircle } from 'lucide-react'
import type { TimeMetrics } from '@/api/analytics'

interface TimeMetricsCardProps {
  metrics?: TimeMetrics
  isLoading?: boolean
}

export const TimeMetricsCard: React.FC<TimeMetricsCardProps> = ({ metrics, isLoading }) => {
  if (isLoading) {
    return (
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm animate-pulse space-y-3">
        <div className="w-36 h-5 bg-slate-200 rounded" />
        <div className="grid grid-cols-3 gap-4">
          <div className="h-16 bg-slate-100 rounded" />
          <div className="h-16 bg-slate-100 rounded" />
          <div className="h-16 bg-slate-100 rounded" />
        </div>
      </div>
    )
  }

  const formatHours = (hours?: number | null) => {
    if (hours === undefined || hours === null) {
      return 'N/A'
    }
    if (hours < 1) {
      return `${Math.round(hours * 60)} mins`
    }
    if (hours > 48) {
      const days = (hours / 24).toFixed(1)
      return `${days} days`
    }
    return `${hours.toFixed(1)} hrs`
  }

  return (
    <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm space-y-4">
      <div className="flex items-center justify-between border-b border-slate-100 pb-2.5">
        <div className="flex items-center gap-2">
          <Clock className="w-4 h-4 text-primary" />
          <h4 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
            Operational Velocity & Throughput
          </h4>
        </div>
        <span className="text-[11px] text-slate-400">
          Sample size: {metrics?.sample_size ?? 0} records
        </span>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 pt-1">
        {/* Metric 1 */}
        <div className="bg-slate-50 border border-slate-200/60 rounded-lg p-3.5 space-y-1">
          <div className="flex items-center gap-1.5 text-slate-500 text-xs font-medium">
            <Timer className="w-3.5 h-3.5 text-blue-600" />
            <span>Assignment → Review</span>
          </div>
          <div className="text-xl font-bold font-mono text-slate-900">
            {formatHours(metrics?.avg_assignment_to_review_hours)}
          </div>
          <p className="text-[11px] text-slate-400">
            Average response time on incoming candidate files
          </p>
        </div>

        {/* Metric 2 */}
        <div className="bg-slate-50 border border-slate-200/60 rounded-lg p-3.5 space-y-1">
          <div className="flex items-center gap-1.5 text-slate-500 text-xs font-medium">
            <Timer className="w-3.5 h-3.5 text-indigo-600" />
            <span>Review → Interview</span>
          </div>
          <div className="text-xl font-bold font-mono text-slate-900">
            {formatHours(metrics?.avg_review_to_interview_hours)}
          </div>
          <p className="text-[11px] text-slate-400">
            Average scheduling velocity for shortlisted candidates
          </p>
        </div>

        {/* Metric 3 */}
        <div className="bg-slate-50 border border-slate-200/60 rounded-lg p-3.5 space-y-1">
          <div className="flex items-center gap-1.5 text-slate-500 text-xs font-medium">
            <CheckCircle className="w-3.5 h-3.5 text-emerald-600" />
            <span>Interview → Completion</span>
          </div>
          <div className="text-xl font-bold font-mono text-slate-900">
            {formatHours(metrics?.avg_interview_to_completion_hours)}
          </div>
          <p className="text-[11px] text-slate-400">
            Average duration to feedback logging & completion
          </p>
        </div>
      </div>
    </div>
  )
}

import React from 'react'
import { Users, Briefcase, CalendarCheck, UserCheck } from 'lucide-react'
import type { AnalyticsSummaryCounts, FunnelData } from '@/api/analytics'

interface KPIRowProps {
  summary: AnalyticsSummaryCounts
  funnel?: FunnelData
  isLoading?: boolean
}

export const KPIRow: React.FC<KPIRowProps> = ({ summary, funnel, isLoading }) => {
  if (isLoading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm animate-pulse space-y-3">
            <div className="w-8 h-8 bg-slate-200 rounded-lg" />
            <div className="w-24 h-4 bg-slate-200 rounded" />
            <div className="w-16 h-7 bg-slate-200 rounded" />
          </div>
        ))}
      </div>
    )
  }

  const reviewRate = funnel?.review_rate ?? 0
  const shortlistRate = funnel?.shortlist_rate ?? 0

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
      {/* 1. Total Candidates */}
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm hover:border-slate-300 transition-colors">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
            Total Candidates
          </span>
          <div className="w-8 h-8 rounded-lg bg-blue-50 text-blue-600 flex items-center justify-center">
            <Users className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline gap-2">
          <span className="text-2xl font-bold font-mono text-slate-900 tracking-tight">
            {summary.total_candidates.toLocaleString()}
          </span>
          <span className="text-xs text-slate-500">distinct talent</span>
        </div>
        <p className="mt-1 text-xs text-slate-500">
          {reviewRate}% reviewed across active pipelines
        </p>
      </div>

      {/* 2. Active Jobs */}
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm hover:border-slate-300 transition-colors">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
            Active Jobs
          </span>
          <div className="w-8 h-8 rounded-lg bg-emerald-50 text-emerald-600 flex items-center justify-center">
            <Briefcase className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline gap-2">
          <span className="text-2xl font-bold font-mono text-slate-900 tracking-tight">
            {summary.active_jobs.toLocaleString()}
          </span>
          <span className="text-xs font-medium text-emerald-700 bg-emerald-50 px-1.5 py-0.5 rounded">
            OPEN Status
          </span>
        </div>
        <p className="mt-1 text-xs text-slate-500">
          Live job requisitions accepting candidates
        </p>
      </div>

      {/* 3. Total Interviews */}
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm hover:border-slate-300 transition-colors">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
            Total Interviews
          </span>
          <div className="w-8 h-8 rounded-lg bg-indigo-50 text-indigo-600 flex items-center justify-center">
            <CalendarCheck className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline gap-2">
          <span className="text-2xl font-bold font-mono text-slate-900 tracking-tight">
            {summary.total_interviews.toLocaleString()}
          </span>
          <span className="text-xs text-slate-500">sessions</span>
        </div>
        <div className="mt-1.5 flex items-center gap-1.5 flex-wrap">
          <span className="inline-flex items-center text-[11px] font-medium text-blue-700 bg-blue-50 px-1.5 py-0.5 rounded">
            {summary.scheduled_interviews} Scheduled
          </span>
          <span className="inline-flex items-center text-[11px] font-medium text-emerald-700 bg-emerald-50 px-1.5 py-0.5 rounded">
            {summary.completed_interviews} Completed
          </span>
          {summary.cancelled_interviews > 0 && (
            <span className="inline-flex items-center text-[11px] font-medium text-slate-500 bg-slate-100 px-1.5 py-0.5 rounded">
              {summary.cancelled_interviews} Cancelled
            </span>
          )}
        </div>
      </div>

      {/* 4. Shortlisted Applications */}
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm hover:border-slate-300 transition-colors">
        <div className="flex items-center justify-between">
          <span className="text-xs font-semibold text-slate-500 uppercase tracking-wider">
            Shortlisted
          </span>
          <div className="w-8 h-8 rounded-lg bg-amber-50 text-amber-600 flex items-center justify-center">
            <UserCheck className="w-4 h-4" />
          </div>
        </div>
        <div className="mt-2 flex items-baseline gap-2">
          <span className="text-2xl font-bold font-mono text-slate-900 tracking-tight">
            {summary.shortlisted_applications.toLocaleString()}
          </span>
          <span className="text-xs text-slate-500">applications</span>
        </div>
        <p className="mt-1 text-xs text-slate-500">
          {shortlistRate}% pass rate from reviewed pool
        </p>
      </div>
    </div>
  )
}

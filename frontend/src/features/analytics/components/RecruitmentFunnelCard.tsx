import React from 'react'
import { Filter, ArrowDownRight, CheckCircle2 } from 'lucide-react'
import type { FunnelData } from '@/api/analytics'

interface RecruitmentFunnelCardProps {
  funnel: FunnelData
  isLoading?: boolean
}

export const RecruitmentFunnelCard: React.FC<RecruitmentFunnelCardProps> = ({ funnel, isLoading }) => {
  if (isLoading) {
    return (
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm animate-pulse space-y-4">
        <div className="w-48 h-5 bg-slate-200 rounded" />
        <div className="space-y-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className="h-10 bg-slate-100 rounded" />
          ))}
        </div>
      </div>
    )
  }

  const stages = [
    {
      name: '1. Registered Talent',
      count: funnel.total_candidates,
      conversion: 100,
      overall: 100,
      color: 'bg-blue-600',
      lightColor: 'bg-blue-50 text-blue-700',
    },
    {
      name: '2. Recruiter Reviewed',
      count: funnel.reviewed_candidates,
      conversion: funnel.review_rate,
      overall: funnel.review_rate,
      color: 'bg-indigo-600',
      lightColor: 'bg-indigo-50 text-indigo-700',
    },
    {
      name: '3. Shortlisted',
      count: funnel.shortlisted_applications,
      conversion: funnel.shortlist_rate,
      overall: funnel.total_candidates > 0
        ? Math.round((funnel.shortlisted_applications / funnel.total_candidates) * 1000) / 10
        : 0,
      color: 'bg-violet-600',
      lightColor: 'bg-violet-50 text-violet-700',
    },
    {
      name: '4. Interviewed',
      count: funnel.interviewed_candidates,
      conversion: funnel.interview_rate,
      overall: funnel.total_candidates > 0
        ? Math.round((funnel.interviewed_candidates / funnel.total_candidates) * 1000) / 10
        : 0,
      color: 'bg-teal-600',
      lightColor: 'bg-teal-50 text-teal-700',
    },
    {
      name: '5. Interview Completed',
      count: funnel.completed_interview_candidates,
      conversion: funnel.completion_rate,
      overall: funnel.total_candidates > 0
        ? Math.round((funnel.completed_interview_candidates / funnel.total_candidates) * 1000) / 10
        : 0,
      color: 'bg-emerald-600',
      lightColor: 'bg-emerald-50 text-emerald-700',
    },
  ]

  const maxCount = Math.max(funnel.total_candidates, 1)

  return (
    <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm space-y-4">
      <div className="flex items-center justify-between border-b border-slate-100 pb-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-900 flex items-center gap-2">
            <Filter className="w-4 h-4 text-primary" />
            <span>Recruitment Pipeline Funnel</span>
          </h3>
          <p className="text-xs text-slate-500 mt-0.5">
            Stage-by-stage progression from initial applicant pool to interview completion
          </p>
        </div>
        <div className="flex items-center gap-1 text-xs font-medium text-emerald-700 bg-emerald-50 px-2 py-1 rounded-md">
          <CheckCircle2 className="w-3.5 h-3.5" />
          <span>Deterministic Metrics</span>
        </div>
      </div>

      <div className="space-y-4 pt-1">
        {stages.map((st, index) => {
          const widthPct = Math.max(Math.round((st.count / maxCount) * 100), 4)

          return (
            <div key={st.name} className="space-y-1.5">
              <div className="flex items-center justify-between text-xs">
                <span className="font-medium text-slate-800">{st.name}</span>
                <div className="flex items-center gap-3">
                  <span className="font-mono font-bold text-slate-900">
                    {st.count.toLocaleString()}
                  </span>
                  <span className={`text-[11px] font-medium px-1.5 py-0.5 rounded ${st.lightColor}`}>
                    {index === 0 ? '100%' : `${st.conversion}% step`}
                  </span>
                  <span className="text-[11px] font-mono text-slate-400 w-14 text-right">
                    ({st.overall}% pool)
                  </span>
                </div>
              </div>

              {/* Funnel Progress Track */}
              <div className="w-full bg-slate-100 rounded-full h-3 overflow-hidden flex items-center">
                <div
                  className={`h-full rounded-full transition-all duration-500 ${st.color}`}
                  style={{ width: `${widthPct}%` }}
                />
              </div>

              {/* Conversion Drop-off descriptor */}
              {index > 0 && (
                <div className="flex items-center gap-1 text-[11px] text-slate-400 pl-1">
                  <ArrowDownRight className="w-3 h-3 text-slate-300" />
                  <span>
                    {st.conversion}% conversion from previous stage ({stages[index - 1].count - st.count} dropped off)
                  </span>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

import React from 'react'
import { GitBranch, ShieldCheck, CheckCircle, XCircle } from 'lucide-react'
import type { DistributionItem, ScreeningAnalytics } from '@/api/analytics'

interface PipelineScreeningCardProps {
  pipeline: DistributionItem[]
  screening: ScreeningAnalytics
  isLoading?: boolean
}

const statusBadgeColors: Record<string, { bg: string; text: string; bar: string }> = {
  REVIEW: { bg: 'bg-amber-50', text: 'text-amber-700', bar: 'bg-amber-500' },
  SHORTLISTED: { bg: 'bg-emerald-50', text: 'text-emerald-700', bar: 'bg-emerald-500' },
  REJECTED: { bg: 'bg-rose-50', text: 'text-rose-700', bar: 'bg-rose-500' },
  QUALIFIED: { bg: 'bg-emerald-50', text: 'text-emerald-700', bar: 'bg-emerald-500' },
  NOT_QUALIFIED: { bg: 'bg-rose-50', text: 'text-rose-700', bar: 'bg-rose-500' },
}

export const PipelineScreeningCard: React.FC<PipelineScreeningCardProps> = ({
  pipeline,
  screening,
  isLoading,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm animate-pulse space-y-4">
        <div className="w-40 h-5 bg-slate-200 rounded" />
        <div className="h-32 bg-slate-100 rounded" />
      </div>
    )
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
      {/* 1. Pipeline Status Distribution */}
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm space-y-3">
        <div className="flex items-center justify-between border-b border-slate-100 pb-2.5">
          <div className="flex items-center gap-2">
            <GitBranch className="w-4 h-4 text-primary" />
            <h4 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
              Pipeline Distribution
            </h4>
          </div>
          <span className="text-[11px] text-slate-400">Application status</span>
        </div>

        <div className="space-y-3 pt-1">
          {pipeline.length === 0 ? (
            <p className="text-xs text-slate-400 py-3 text-center">No applications found</p>
          ) : (
            pipeline.map((item) => {
              const theme = statusBadgeColors[item.key] || {
                bg: 'bg-slate-50',
                text: 'text-slate-700',
                bar: 'bg-slate-500',
              }

              return (
                <div key={item.key} className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className={`font-semibold px-2 py-0.5 rounded text-[11px] ${theme.bg} ${theme.text}`}>
                      {item.key}
                    </span>
                    <div className="flex items-center gap-2">
                      <span className="font-mono font-bold text-slate-900">{item.count}</span>
                      <span className="text-slate-500 text-[11px] w-12 text-right">
                        ({item.percentage}%)
                      </span>
                    </div>
                  </div>
                  <div className="w-full bg-slate-100 rounded-full h-2 overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-300 ${theme.bar}`}
                      style={{ width: `${Math.max(item.percentage, 2)}%` }}
                    />
                  </div>
                </div>
              )
            })
          )}
        </div>
      </div>

      {/* 2. Deterministic Screening Distribution */}
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm space-y-3">
        <div className="flex items-center justify-between border-b border-slate-100 pb-2.5">
          <div className="flex items-center gap-2">
            <ShieldCheck className="w-4 h-4 text-emerald-600" />
            <h4 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
              Deterministic Screening
            </h4>
          </div>
          <span className="text-[11px] text-slate-400">Total: {screening.total}</span>
        </div>

        <div className="space-y-3 pt-1">
          {screening.by_status.length === 0 ? (
            <p className="text-xs text-slate-400 py-3 text-center">No screening results recorded</p>
          ) : (
            screening.by_status.map((item) => {
              const theme = statusBadgeColors[item.key] || {
                bg: 'bg-slate-50',
                text: 'text-slate-700',
                bar: 'bg-slate-500',
              }

              return (
                <div key={item.key} className="space-y-1">
                  <div className="flex items-center justify-between text-xs">
                    <span className={`font-semibold px-2 py-0.5 rounded text-[11px] ${theme.bg} ${theme.text}`}>
                      {item.key}
                    </span>
                    <div className="flex items-center gap-2">
                      <span className="font-mono font-bold text-slate-900">{item.count}</span>
                      <span className="text-slate-500 text-[11px] w-12 text-right">
                        ({item.percentage}%)
                      </span>
                    </div>
                  </div>
                  <div className="w-full bg-slate-100 rounded-full h-2 overflow-hidden">
                    <div
                      className={`h-full rounded-full transition-all duration-300 ${theme.bar}`}
                      style={{ width: `${Math.max(item.percentage, 2)}%` }}
                    />
                  </div>
                </div>
              )
            })
          )}

          {/* Requirement Match Overview */}
          {screening.requirement_matches.length > 0 && (
            <div className="pt-2 border-t border-slate-100 mt-2">
              <span className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider block mb-1.5">
                Requirement Matches
              </span>
              <div className="grid grid-cols-2 gap-2 text-xs">
                {screening.requirement_matches.map((rm, idx) => (
                  <div
                    key={idx}
                    className="flex items-center justify-between bg-slate-50 px-2 py-1 rounded border border-slate-100 text-[11px]"
                  >
                    <span className="flex items-center gap-1 text-slate-600 font-medium">
                      {rm.match_status === 'PASSED' ? (
                        <CheckCircle className="w-3 h-3 text-emerald-600" />
                      ) : (
                        <XCircle className="w-3 h-3 text-rose-500" />
                      )}
                      {rm.importance}
                    </span>
                    <span className="font-mono font-bold text-slate-800">{rm.count}</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

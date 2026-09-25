import React from 'react'
import { CalendarCheck, Video, MapPin, Phone, Layers } from 'lucide-react'
import type { InterviewAnalytics } from '@/api/analytics'

interface InterviewAnalyticsCardProps {
  interviews: InterviewAnalytics
  isLoading?: boolean
}

export const InterviewAnalyticsCard: React.FC<InterviewAnalyticsCardProps> = ({
  interviews,
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

  const getTypeIcon = (type: string) => {
    switch (type.toUpperCase()) {
      case 'ONLINE':
        return <Video className="w-3.5 h-3.5 text-blue-600" />
      case 'ONSITE':
        return <MapPin className="w-3.5 h-3.5 text-emerald-600" />
      case 'PHONE':
        return <Phone className="w-3.5 h-3.5 text-indigo-600" />
      default:
        return null
    }
  }

  return (
    <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm space-y-4">
      <div className="flex items-center justify-between border-b border-slate-100 pb-2.5">
        <div className="flex items-center gap-2">
          <CalendarCheck className="w-4 h-4 text-indigo-600" />
          <h4 className="text-xs font-bold text-slate-800 uppercase tracking-wider">
            Interview Operations Breakdown
          </h4>
        </div>
        <span className="text-[11px] text-slate-400">Total sessions: {interviews.total}</span>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-6 pt-1">
        {/* 1. Status Breakdown */}
        <div className="space-y-2.5">
          <span className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider block">
            By Status
          </span>
          <div className="space-y-2">
            {interviews.by_status.map((item) => (
              <div key={item.key} className="space-y-1">
                <div className="flex justify-between text-xs">
                  <span className="font-medium text-slate-700">{item.key}</span>
                  <span className="font-mono text-slate-900 font-semibold">
                    {item.count} <span className="text-[11px] text-slate-400 font-normal">({item.percentage}%)</span>
                  </span>
                </div>
                <div className="w-full bg-slate-100 rounded-full h-1.5">
                  <div
                    className={`h-full rounded-full ${
                      item.key === 'COMPLETED'
                        ? 'bg-emerald-500'
                        : item.key === 'SCHEDULED'
                        ? 'bg-blue-500'
                        : 'bg-slate-400'
                    }`}
                    style={{ width: `${Math.max(item.percentage, 2)}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* 2. Modality / Type Breakdown */}
        <div className="space-y-2.5">
          <span className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider block">
            By Modality
          </span>
          <div className="space-y-2">
            {interviews.by_type.map((item) => (
              <div key={item.key} className="space-y-1">
                <div className="flex justify-between text-xs items-center">
                  <span className="font-medium text-slate-700 flex items-center gap-1.5">
                    {getTypeIcon(item.key)}
                    {item.key}
                  </span>
                  <span className="font-mono text-slate-900 font-semibold">
                    {item.count} <span className="text-[11px] text-slate-400 font-normal">({item.percentage}%)</span>
                  </span>
                </div>
                <div className="w-full bg-slate-100 rounded-full h-1.5">
                  <div
                    className="h-full rounded-full bg-indigo-500"
                    style={{ width: `${Math.max(item.percentage, 2)}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* 3. Stage Breakdown */}
        <div className="space-y-2.5">
          <span className="text-[11px] font-semibold text-slate-500 uppercase tracking-wider block flex items-center gap-1">
            <Layers className="w-3 h-3 text-slate-400" />
            By Stage
          </span>
          <div className="space-y-2">
            {interviews.by_stage.map((item) => (
              <div key={item.key} className="space-y-1">
                <div className="flex justify-between text-xs">
                  <span className="font-medium text-slate-700 truncate max-w-[130px]" title={item.key}>
                    {item.key.replace(/_/g, ' ')}
                  </span>
                  <span className="font-mono text-slate-900 font-semibold">
                    {item.count} <span className="text-[11px] text-slate-400 font-normal">({item.percentage}%)</span>
                  </span>
                </div>
                <div className="w-full bg-slate-100 rounded-full h-1.5">
                  <div
                    className="h-full rounded-full bg-teal-500"
                    style={{ width: `${Math.max(item.percentage, 2)}%` }}
                  />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}

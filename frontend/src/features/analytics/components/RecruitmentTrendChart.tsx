import React from 'react'
import {
  ResponsiveContainer,
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
} from 'recharts'
import { TrendingUp } from 'lucide-react'
import type { TrendDataPoint } from '@/api/analytics'

interface RecruitmentTrendChartProps {
  data: TrendDataPoint[]
  interval: 'day' | 'week' | 'month'
  onIntervalChange: (interval: 'day' | 'week' | 'month') => void
  isLoading?: boolean
}

export const RecruitmentTrendChart: React.FC<RecruitmentTrendChartProps> = ({
  data,
  interval,
  onIntervalChange,
  isLoading,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm animate-pulse space-y-4">
        <div className="flex justify-between">
          <div className="w-48 h-5 bg-slate-200 rounded" />
          <div className="w-32 h-7 bg-slate-200 rounded" />
        </div>
        <div className="h-64 bg-slate-100 rounded-lg" />
      </div>
    )
  }

  const chartData = data.map((d) => ({
    date: d.period || d.date,
    'Candidates Added': Number(d.candidates_count),
    'Interviews Scheduled': Number(d.interviews_count),
  }))

  return (
    <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm space-y-4">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-100 pb-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-900 flex items-center gap-2">
            <TrendingUp className="w-4 h-4 text-primary" />
            <span>Recruitment Activity Trends</span>
          </h3>
          <p className="text-xs text-slate-500 mt-0.5">
            Temporal distribution of candidates assigned vs interviews conducted
          </p>
        </div>

        {/* Interval toggle buttons */}
        <div className="flex items-center rounded-lg border border-slate-200 p-0.5 bg-slate-50 self-start sm:self-auto">
          {(['day', 'week', 'month'] as const).map((int) => (
            <button
              key={int}
              type="button"
              onClick={() => onIntervalChange(int)}
              className={`px-3 py-1 text-xs font-medium rounded-md transition-all capitalize ${
                interval === int
                  ? 'bg-white text-primary shadow-xs font-semibold'
                  : 'text-slate-600 hover:text-slate-900'
              }`}
            >
              {int}
            </button>
          ))}
        </div>
      </div>

      {chartData.length === 0 ? (
        <div className="h-64 flex flex-col items-center justify-center text-slate-400 text-xs">
          <span>No activity data recorded within this date boundary</span>
        </div>
      ) : (
        <div className="h-64 w-full">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
              <defs>
                <linearGradient id="colorCand" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#2563eb" stopOpacity={0.25} />
                  <stop offset="95%" stopColor="#2563eb" stopOpacity={0.0} />
                </linearGradient>
                <linearGradient id="colorItw" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#10b981" stopOpacity={0.25} />
                  <stop offset="95%" stopColor="#10b981" stopOpacity={0.0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="#f1f5f9" />
              <XAxis
                dataKey="date"
                tick={{ fontSize: 11, fill: '#64748b' }}
                stroke="#cbd5e1"
                tickLine={false}
              />
              <YAxis
                allowDecimals={false}
                tick={{ fontSize: 11, fill: '#64748b' }}
                stroke="#cbd5e1"
                tickLine={false}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: '#ffffff',
                  borderRadius: '8px',
                  border: '1px solid #e2e8f0',
                  boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)',
                  fontSize: '12px',
                }}
              />
              <Legend
                verticalAlign="top"
                align="right"
                iconType="circle"
                wrapperStyle={{ fontSize: '11px', paddingBottom: '10px' }}
              />
              <Area
                type="monotone"
                dataKey="Candidates Added"
                stroke="#2563eb"
                strokeWidth={2}
                fillOpacity={1}
                fill="url(#colorCand)"
              />
              <Area
                type="monotone"
                dataKey="Interviews Scheduled"
                stroke="#10b981"
                strokeWidth={2}
                fillOpacity={1}
                fill="url(#colorItw)"
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  )
}

import React, { useState } from 'react'
import { Filter, Calendar, RotateCcw, AlertCircle, Building2, Briefcase, UserCheck } from 'lucide-react'
import type { AnalyticsFilterParams } from '@/api/analytics'
import type { Job, Interviewer } from '@/types'

interface AnalyticsFilterBarProps {
  filters: AnalyticsFilterParams
  onFiltersChange: (newFilters: AnalyticsFilterParams) => void
  jobs: Job[]
  departments: string[]
  recruiters: Interviewer[]
  isLoading?: boolean
}

type DatePreset = 'last_7_days' | 'last_14_days' | 'last_30_days' | 'last_90_days' | 'this_month' | 'last_month' | 'custom'

function formatDate(d: Date): string {
  return d.toISOString().split('T')[0]
}

function calculatePresetDates(preset: DatePreset): { from: string; to: string } {
  const now = new Date()
  const todayStr = formatDate(now)

  switch (preset) {
    case 'last_7_days': {
      const past = new Date(now)
      past.setDate(past.getDate() - 7)
      return { from: formatDate(past), to: todayStr }
    }
    case 'last_14_days': {
      const past = new Date(now)
      past.setDate(past.getDate() - 14)
      return { from: formatDate(past), to: todayStr }
    }
    case 'last_30_days': {
      const past = new Date(now)
      past.setDate(past.getDate() - 30)
      return { from: formatDate(past), to: todayStr }
    }
    case 'last_90_days': {
      const past = new Date(now)
      past.setDate(past.getDate() - 90)
      return { from: formatDate(past), to: todayStr }
    }
    case 'this_month': {
      const start = new Date(now.getFullYear(), now.getMonth(), 1)
      return { from: formatDate(start), to: todayStr }
    }
    case 'last_month': {
      const start = new Date(now.getFullYear(), now.getMonth() - 1, 1)
      const end = new Date(now.getFullYear(), now.getMonth(), 0)
      return { from: formatDate(start), to: formatDate(end) }
    }
    case 'custom':
    default:
      return { from: '', to: '' }
  }
}

export const AnalyticsFilterBar: React.FC<AnalyticsFilterBarProps> = ({
  filters,
  onFiltersChange,
  jobs,
  departments,
  recruiters,
  isLoading,
}) => {
  const [preset, setPreset] = useState<DatePreset>('last_30_days')
  const initialDates = calculatePresetDates('last_30_days')
  const [customFrom, setCustomFrom] = useState(filters.from || initialDates.from)
  const [customTo, setCustomTo] = useState(filters.to || initialDates.to)
  const [dateError, setDateError] = useState<string | null>(null)

  const validateDates = (from: string, to: string): boolean => {
    if (!from || !to) {
      setDateError(null)
      return true
    }
    const dFrom = new Date(from)
    const dTo = new Date(to)
    if (dFrom > dTo) {
      setDateError("'From' date must be before or equal to 'To' date")
      return false
    }
    const diffDays = Math.ceil((dTo.getTime() - dFrom.getTime()) / (1000 * 60 * 60 * 24))
    if (diffDays > 366) {
      setDateError('Date range cannot exceed 366 days')
      return false
    }
    setDateError(null)
    return true
  }

  const handlePresetChange = (newPreset: DatePreset) => {
    setPreset(newPreset)
    if (newPreset === 'custom') {
      return
    }
    const { from, to } = calculatePresetDates(newPreset)
    setCustomFrom(from)
    setCustomTo(to)
    setDateError(null)
    onFiltersChange({
      ...filters,
      from,
      to,
    })
  }

  const handleCustomDateChange = (from: string, to: string) => {
    setCustomFrom(from)
    setCustomTo(to)
    if (validateDates(from, to)) {
      onFiltersChange({
        ...filters,
        from: from || undefined,
        to: to || undefined,
      })
    }
  }

  const handleReset = () => {
    const { from, to } = calculatePresetDates('last_30_days')
    setPreset('last_30_days')
    setCustomFrom(from)
    setCustomTo(to)
    setDateError(null)
    onFiltersChange({
      from,
      to,
      job_id: undefined,
      department: undefined,
      job_status: undefined,
      recruiter_id: undefined,
      interval: 'day',
    })
  }

  const hasActiveFilters = Boolean(
    filters.job_id ||
    filters.department ||
    filters.job_status ||
    filters.recruiter_id ||
    preset !== 'last_30_days'
  )

  return (
    <div className="bg-white border border-slate-200 rounded-xl p-4 shadow-sm space-y-3">
      {/* Top Filter Controls */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="flex items-center gap-1.5 text-xs font-semibold text-slate-700 uppercase tracking-wider mr-1">
          <Filter className="w-3.5 h-3.5 text-primary" />
          <span>Filters</span>
        </div>

        {/* Date Preset Selector */}
        <div className="flex items-center gap-1.5">
          <Calendar className="w-3.5 h-3.5 text-slate-400" />
          <select
            value={preset}
            onChange={(e) => handlePresetChange(e.target.value as DatePreset)}
            disabled={isLoading}
            className="text-xs border border-slate-200 rounded-lg px-2.5 py-1.5 bg-slate-50 text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary cursor-pointer"
          >
            <option value="last_7_days">Last 7 Days</option>
            <option value="last_14_days">Last 14 Days</option>
            <option value="last_30_days">Last 30 Days</option>
            <option value="last_90_days">Last 90 Days</option>
            <option value="this_month">This Month</option>
            <option value="last_month">Last Month</option>
            <option value="custom">Custom Range...</option>
          </select>
        </div>

        {/* Custom Date Pickers (Shown if Custom or for visual precision) */}
        {preset === 'custom' && (
          <div className="flex items-center gap-2 bg-slate-50 px-2.5 py-1 rounded-lg border border-slate-200">
            <input
              type="date"
              value={customFrom}
              onChange={(e) => handleCustomDateChange(e.target.value, customTo)}
              disabled={isLoading}
              className="text-xs bg-transparent border-0 text-slate-700 focus:outline-none cursor-pointer"
            />
            <span className="text-slate-400 text-xs font-mono">→</span>
            <input
              type="date"
              value={customTo}
              onChange={(e) => handleCustomDateChange(customFrom, e.target.value)}
              disabled={isLoading}
              className="text-xs bg-transparent border-0 text-slate-700 focus:outline-none cursor-pointer"
            />
          </div>
        )}

        {/* Job Selector */}
        <div className="flex items-center gap-1.5">
          <Briefcase className="w-3.5 h-3.5 text-slate-400" />
          <select
            value={filters.job_id || ''}
            onChange={(e) => onFiltersChange({ ...filters, job_id: e.target.value || undefined })}
            disabled={isLoading}
            className="text-xs border border-slate-200 rounded-lg px-2.5 py-1.5 bg-slate-50 text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary cursor-pointer max-w-[200px] truncate"
          >
            <option value="">All Jobs</option>
            {jobs.map((j) => (
              <option key={j.id} value={j.id}>
                {j.code}: {j.title}
              </option>
            ))}
          </select>
        </div>

        {/* Department Selector */}
        <div className="flex items-center gap-1.5">
          <Building2 className="w-3.5 h-3.5 text-slate-400" />
          <select
            value={filters.department || ''}
            onChange={(e) => onFiltersChange({ ...filters, department: e.target.value || undefined })}
            disabled={isLoading}
            className="text-xs border border-slate-200 rounded-lg px-2.5 py-1.5 bg-slate-50 text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary cursor-pointer"
          >
            <option value="">All Departments</option>
            {departments.map((d) => (
              <option key={d} value={d}>
                {d}
              </option>
            ))}
          </select>
        </div>

        {/* Job Status Selector */}
        <select
          value={filters.job_status || ''}
          onChange={(e) => onFiltersChange({ ...filters, job_status: e.target.value || undefined })}
          disabled={isLoading}
          className="text-xs border border-slate-200 rounded-lg px-2.5 py-1.5 bg-slate-50 text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary cursor-pointer"
        >
          <option value="">All Statuses</option>
          <option value="OPEN">Open (Active)</option>
          <option value="DRAFT">Draft</option>
          <option value="CLOSED">Closed</option>
          <option value="ARCHIVED">Archived</option>
        </select>

        {/* Recruiter Selector */}
        <div className="flex items-center gap-1.5">
          <UserCheck className="w-3.5 h-3.5 text-slate-400" />
          <select
            value={filters.recruiter_id || ''}
            onChange={(e) => onFiltersChange({ ...filters, recruiter_id: e.target.value || undefined })}
            disabled={isLoading}
            className="text-xs border border-slate-200 rounded-lg px-2.5 py-1.5 bg-slate-50 text-slate-700 font-medium focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary cursor-pointer max-w-[180px] truncate"
          >
            <option value="">All Recruiters</option>
            {recruiters.map((r) => (
              <option key={r.id} value={r.id}>
                {r.name}
              </option>
            ))}
          </select>
        </div>

        {/* Reset Action */}
        {hasActiveFilters && (
          <button
            type="button"
            onClick={handleReset}
            disabled={isLoading}
            className="ml-auto inline-flex items-center gap-1 text-xs text-slate-500 hover:text-slate-800 transition-colors px-2 py-1 rounded hover:bg-slate-100"
          >
            <RotateCcw className="w-3 h-3" />
            <span>Reset</span>
          </button>
        )}
      </div>

      {/* Date Validation Inline Alert */}
      {dateError && (
        <div className="flex items-center gap-2 text-xs text-rose-600 bg-rose-50 border border-rose-200 px-3 py-1.5 rounded-lg">
          <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
          <span>{dateError}</span>
        </div>
      )}
    </div>
  )
}

import { useState, useMemo } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams, useNavigate } from 'react-router-dom'
import {
  Calendar as CalendarIcon,
  List as ListIcon,
  CalendarDays,
  Plus,
  ChevronRight,
  Filter,
  Clock,
  RotateCcw,
} from 'lucide-react'
import { interviewsApi } from '@/api/interviews'
import { jobsApi } from '@/api/jobs'
import {
  InterviewStatusBadge,
  InterviewStageBadge,
  InterviewTypeBadge,
} from '@/components/data-display/StatusBadge'
import { EmptyState, ErrorState, LoadingState } from '@/components/feedback'
import { formatTimeRange, getLocalDateString } from '@/lib/utils'
import { InterviewCalendarView } from './components/InterviewCalendarView'
import { InterviewAgendaView } from './components/InterviewAgendaView'
import type { InterviewStatus, InterviewStage, InterviewType } from '@/types'

type DisplayMode = 'calendar' | 'agenda' | 'list'
type TabView = 'upcoming' | 'today' | 'past' | 'all'

export function InterviewListPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()

  // View mode
  const modeParam = (searchParams.get('mode') as DisplayMode) || 'calendar'
  const [viewMode, setViewMode] = useState<DisplayMode>(modeParam)

  // Current calendar month navigation
  const [calendarDate, setCalendarDate] = useState<Date>(() => new Date())
  const [selectedDateStr, setSelectedDateStr] = useState<string>(() =>
    getLocalDateString(new Date())
  )

  // Filters
  const activeTab = (searchParams.get('tab') as TabView) || 'upcoming'
  const page = parseInt(searchParams.get('page') || '1', 10)
  const jobId = searchParams.get('job_id') || ''
  const status = (searchParams.get('status') as InterviewStatus) || ''
  const stage = (searchParams.get('stage') as InterviewStage) || ''
  const interviewType = (searchParams.get('type') as InterviewType) || ''

  const setMode = (mode: DisplayMode) => {
    setViewMode(mode)
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('mode', mode)
      return next
    })
  }

  const setTab = (tab: TabView) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      next.set('tab', tab)
      next.delete('page')
      return next
    })
  }

  const updateFilter = (key: string, value: string) => {
    setSearchParams((prev) => {
      const next = new URLSearchParams(prev)
      if (value) {
        next.set(key, value)
      } else {
        next.delete(key)
      }
      next.delete('page')
      return next
    })
  }

  const clearFilters = () => {
    setSearchParams((prev) => {
      const next = new URLSearchParams()
      if (prev.get('mode')) next.set('mode', prev.get('mode')!)
      if (prev.get('tab')) next.set('tab', prev.get('tab')!)
      return next
    })
  }

  const hasActiveFilters = !!(jobId || status || stage || interviewType)

  // Determine query parameters based on viewMode
  const queryParams = useMemo(() => {
    const now = new Date()

    if (viewMode === 'calendar' || viewMode === 'agenda') {
      // Month range bounded for current calendar view
      const y = calendarDate.getFullYear()
      const m = calendarDate.getMonth()
      // Extend 7 days before and after month to cover grid overlap
      const fromDate = new Date(y, m, -6, 0, 0, 0)
      const toDate = new Date(y, m + 1, 7, 23, 59, 59)

      return {
        limit: 200,
        page: 1,
        job_id: jobId || undefined,
        status: status || undefined,
        stage: stage || undefined,
        interview_type: interviewType || undefined,
        from: fromDate.toISOString(),
        to: toDate.toISOString(),
        sort: 'scheduled_start',
        order: 'ASC',
      }
    }

    // Table List mode using tabs
    let fromParam: string | undefined
    let toParam: string | undefined
    let effectiveStatus: InterviewStatus | '' = status

    if (activeTab === 'upcoming') {
      fromParam = now.toISOString()
      if (!status) effectiveStatus = 'SCHEDULED'
    } else if (activeTab === 'today') {
      const startOfDay = new Date(now)
      startOfDay.setHours(0, 0, 0, 0)
      const endOfDay = new Date(now)
      endOfDay.setHours(23, 59, 59, 999)
      fromParam = startOfDay.toISOString()
      toParam = endOfDay.toISOString()
    } else if (activeTab === 'past') {
      toParam = now.toISOString()
    }

    return {
      limit: 15,
      page,
      job_id: jobId || undefined,
      status: effectiveStatus || undefined,
      stage: stage || undefined,
      interview_type: interviewType || undefined,
      from: fromParam,
      to: toParam,
      sort: 'scheduled_start',
      order: activeTab === 'past' ? 'DESC' : 'ASC',
    }
  }, [viewMode, calendarDate, jobId, status, stage, interviewType, activeTab, page])

  // Queries
  const { data: jobsData } = useQuery({
    queryKey: ['jobs', { limit: 100 }],
    queryFn: () => jobsApi.list({ limit: 100 }),
  })

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: [
      'interviews',
      {
        mode: viewMode,
        month: `${calendarDate.getFullYear()}-${calendarDate.getMonth() + 1}`,
        tab: activeTab,
        ...queryParams,
      },
    ],
    queryFn: () => interviewsApi.list(queryParams),
  })

  const interviewsList = data?.items || []

  const handleDateClick = (dateStr: string) => {
    setSelectedDateStr(dateStr)
    // If on calendar mode, user can stay or toggle to agenda
  }

  const handleJumpToday = () => {
    const today = new Date()
    setCalendarDate(today)
    setSelectedDateStr(getLocalDateString(today))
  }

  return (
    <div className="max-w-dashboard mx-auto space-y-5">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4">
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-mono font-bold text-primary">Interviews</h1>
            <span className="text-xs font-mono font-semibold px-2 py-0.5 bg-blue-50 text-blue-700 border border-blue-200 rounded-full">
              {data?.total ?? 0}
            </span>
          </div>
          <p className="text-xs text-slate-500 mt-0.5">
            Manage scheduled and completed candidate interviews.
          </p>
        </div>

        <div className="flex items-center gap-2">
          {/* Today Button */}
          <button
            onClick={handleJumpToday}
            className="px-3 py-1.5 text-xs font-semibold font-mono text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
          >
            Today
          </button>

          {/* Schedule Interview CTA */}
          <Link
            to="/interviews/new"
            className="flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-semibold bg-cta text-white rounded-lg hover:opacity-90 transition-all cursor-pointer shadow-xs focus:outline-none focus:ring-2 focus:ring-cta focus:ring-offset-1"
          >
            <Plus className="w-4 h-4" />
            Schedule Interview
          </Link>
        </div>
      </div>

      {/* Control Bar: View Toggle + Filters */}
      <div className="bg-white rounded-card shadow-sm border border-slate-200 p-3 space-y-3">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-3">
          {/* View Mode Segmented Controls */}
          <div className="inline-flex items-center p-1 bg-slate-100 rounded-lg border border-slate-200 self-start">
            <button
              onClick={() => setMode('calendar')}
              className={`flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-md transition-all cursor-pointer focus:outline-none ${
                viewMode === 'calendar'
                  ? 'bg-white text-primary shadow-xs font-bold'
                  : 'text-slate-600 hover:text-slate-900'
              }`}
            >
              <CalendarDays className="w-3.5 h-3.5" />
              Calendar
            </button>
            <button
              onClick={() => setMode('agenda')}
              className={`flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-md transition-all cursor-pointer focus:outline-none ${
                viewMode === 'agenda'
                  ? 'bg-white text-primary shadow-xs font-bold'
                  : 'text-slate-600 hover:text-slate-900'
              }`}
            >
              <Clock className="w-3.5 h-3.5" />
              Agenda
            </button>
            <button
              onClick={() => setMode('list')}
              className={`flex items-center gap-1.5 px-3 py-1 text-xs font-semibold rounded-md transition-all cursor-pointer focus:outline-none ${
                viewMode === 'list'
                  ? 'bg-white text-primary shadow-xs font-bold'
                  : 'text-slate-600 hover:text-slate-900'
              }`}
            >
              <ListIcon className="w-3.5 h-3.5" />
              Table List
            </button>
          </div>

          {/* Sub-Tabs for Table List Mode */}
          {viewMode === 'list' && (
            <div className="flex items-center gap-1">
              {(['upcoming', 'today', 'past', 'all'] as TabView[]).map((tab) => (
                <button
                  key={tab}
                  onClick={() => setTab(tab)}
                  className={`px-2.5 py-1 text-xs font-medium rounded transition-colors uppercase font-mono cursor-pointer ${
                    activeTab === tab
                      ? 'bg-primary text-white font-bold'
                      : 'text-slate-600 hover:bg-slate-100'
                  }`}
                >
                  {tab}
                </button>
              ))}
            </div>
          )}
        </div>

        {/* Filters Row */}
        <div className="pt-2 border-t border-slate-100 flex flex-wrap items-center gap-2 text-xs">
          <div className="flex items-center gap-1 text-slate-500 font-semibold mr-1">
            <Filter className="w-3.5 h-3.5" />
            <span>Filters:</span>
          </div>

          {/* Job Filter */}
          <select
            value={jobId}
            onChange={(e) => updateFilter('job_id', e.target.value)}
            className="px-2.5 py-1 bg-white border border-slate-300 rounded text-slate-700 text-xs focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
          >
            <option value="">All Jobs</option>
            {jobsData?.items.map((j) => (
              <option key={j.id} value={j.id}>
                {j.title} {j.code ? `(${j.code})` : ''}
              </option>
            ))}
          </select>

          {/* Stage Filter */}
          <select
            value={stage}
            onChange={(e) => updateFilter('stage', e.target.value)}
            className="px-2.5 py-1 bg-white border border-slate-300 rounded text-slate-700 text-xs focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
          >
            <option value="">All Stages</option>
            <option value="PHONE_SCREEN">Phone Screen</option>
            <option value="HR_INTERVIEW">HR Interview</option>
            <option value="TECHNICAL_INTERVIEW">Technical Interview</option>
            <option value="MANAGER_INTERVIEW">Manager Interview</option>
            <option value="FINAL_INTERVIEW">Final Interview</option>
            <option value="OTHER">Other</option>
          </select>

          {/* Status Filter */}
          <select
            value={status}
            onChange={(e) => updateFilter('status', e.target.value)}
            className="px-2.5 py-1 bg-white border border-slate-300 rounded text-slate-700 text-xs focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
          >
            <option value="">All Statuses</option>
            <option value="SCHEDULED">Scheduled</option>
            <option value="COMPLETED">Completed</option>
            <option value="CANCELLED">Cancelled</option>
          </select>

          {/* Type Filter */}
          <select
            value={interviewType}
            onChange={(e) => updateFilter('type', e.target.value)}
            className="px-2.5 py-1 bg-white border border-slate-300 rounded text-slate-700 text-xs focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
          >
            <option value="">All Modalities</option>
            <option value="ONLINE">Online Video</option>
            <option value="ONSITE">Onsite Office</option>
            <option value="PHONE">Phone Call</option>
          </select>

          {/* Reset Filters */}
          {hasActiveFilters && (
            <button
              onClick={clearFilters}
              className="inline-flex items-center gap-1 text-primary hover:underline font-semibold ml-auto cursor-pointer"
            >
              <RotateCcw className="w-3 h-3" />
              Reset filters
            </button>
          )}
        </div>
      </div>

      {/* Main View Area */}
      {isLoading ? (
        <div className="bg-white rounded-card shadow-sm border border-slate-200 p-8">
          <LoadingState message="Loading interviews…" />
        </div>
      ) : isError ? (
        <div className="bg-white rounded-card shadow-sm border border-slate-200 p-8">
          <ErrorState message="Unable to load interviews." onRetry={refetch} />
        </div>
      ) : viewMode === 'calendar' ? (
        <div className="space-y-4">
          <InterviewCalendarView
            currentDate={calendarDate}
            onDateChange={setCalendarDate}
            interviews={interviewsList}
            onSelectDate={handleDateClick}
            selectedDateStr={selectedDateStr}
          />

          {/* Quick Agenda Preview for Selected Day */}
          {selectedDateStr && (
            <div className="pt-2">
              <InterviewAgendaView
                interviews={interviewsList.filter(
                  (iv) => getLocalDateString(iv.scheduled_start) === selectedDateStr
                )}
                selectedDateStr={selectedDateStr}
              />
            </div>
          )}
        </div>
      ) : viewMode === 'agenda' ? (
        <InterviewAgendaView interviews={interviewsList} selectedDateStr={selectedDateStr} />
      ) : (
        /* Table List View */
        <div className="bg-white rounded-card shadow-sm border border-slate-200 overflow-hidden">
          <div className="overflow-x-auto">
            <table className="w-full text-left text-xs border-collapse">
              <thead>
                <tr className="bg-slate-50 border-b border-slate-200 text-slate-500 font-mono font-semibold uppercase tracking-wider">
                  <th className="py-3 px-4">Interview Title</th>
                  <th className="py-3 px-4">Candidate</th>
                  <th className="py-3 px-4">Job Role</th>
                  <th className="py-3 px-4">Stage</th>
                  <th className="py-3 px-4">Date & Time</th>
                  <th className="py-3 px-4">Type</th>
                  <th className="py-3 px-4">Status</th>
                  <th className="py-3 px-4 text-right">Action</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {interviewsList.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="py-12 text-center text-slate-400">
                      <EmptyState
                        icon={CalendarIcon}
                        title="No interviews found"
                        description="Try adjusting your filters or schedule a new interview."
                        action={
                          <Link
                            to="/interviews/new"
                            className="mt-3 inline-block px-3 py-1.5 text-xs font-semibold bg-primary text-white rounded-lg hover:opacity-90 transition-all cursor-pointer"
                          >
                            + Schedule Interview
                          </Link>
                        }
                      />
                    </td>
                  </tr>
                ) : (
                  interviewsList.map((iv) => {
                    const candidateName = iv.candidate?.full_name || 'Candidate'
                    const jobTitle = iv.job?.title || 'Job'

                    return (
                      <tr
                        key={iv.id}
                        onClick={() => navigate(`/interviews/${iv.id}`)}
                        className="hover:bg-slate-50/80 transition-colors cursor-pointer"
                      >
                        <td className="py-3 px-4 font-semibold text-slate-900">{iv.title}</td>
                        <td className="py-3 px-4 font-medium text-slate-800">{candidateName}</td>
                        <td className="py-3 px-4 text-slate-600">{jobTitle}</td>
                        <td className="py-3 px-4">
                          <InterviewStageBadge stage={iv.stage} />
                        </td>
                        <td className="py-3 px-4 font-mono text-slate-600 whitespace-nowrap">
                          {formatTimeRange(iv.scheduled_start, iv.scheduled_end)}
                        </td>
                        <td className="py-3 px-4">
                          <InterviewTypeBadge type={iv.interview_type} />
                        </td>
                        <td className="py-3 px-4">
                          <InterviewStatusBadge status={iv.status} />
                        </td>
                        <td className="py-3 px-4 text-right">
                          <Link
                            to={`/interviews/${iv.id}`}
                            onClick={(e) => e.stopPropagation()}
                            className="inline-flex items-center text-primary font-semibold hover:underline"
                          >
                            View
                            <ChevronRight className="w-3.5 h-3.5" />
                          </Link>
                        </td>
                      </tr>
                    )
                  })
                )}
              </tbody>
            </table>
          </div>

          {/* Pagination */}
          {data && data.total_pages > 1 && (
            <div className="px-4 py-3 border-t border-slate-200 flex items-center justify-between text-xs text-slate-600 bg-slate-50">
              <div>
                Showing page <span className="font-semibold">{data.page}</span> of{' '}
                <span className="font-semibold">{data.total_pages}</span> ({data.total} total)
              </div>
              <div className="flex gap-1">
                <button
                  disabled={data.page <= 1}
                  onClick={() => updateFilter('page', String(data.page - 1))}
                  className="px-2.5 py-1 bg-white border border-slate-300 rounded hover:bg-slate-100 disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed font-medium"
                >
                  Previous
                </button>
                <button
                  disabled={data.page >= data.total_pages}
                  onClick={() => updateFilter('page', String(data.page + 1))}
                  className="px-2.5 py-1 bg-white border border-slate-300 rounded hover:bg-slate-100 disabled:opacity-40 cursor-pointer disabled:cursor-not-allowed font-medium"
                >
                  Next
                </button>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

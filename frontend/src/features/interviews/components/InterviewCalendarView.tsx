import { useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ChevronLeft,
  ChevronRight,
  Video,
  MapPin,
  Phone,
  Calendar as CalendarIcon,
} from 'lucide-react'
import { formatTime, getLocalDateString, formatMonthYear } from '@/lib/utils'
import type { Interview } from '@/types'

interface InterviewCalendarViewProps {
  currentDate: Date
  onDateChange: (date: Date) => void
  interviews: Interview[]
  onSelectDate: (dateStr: string) => void
  selectedDateStr?: string
}

const WEEKDAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

export function InterviewCalendarView({
  currentDate,
  onDateChange,
  interviews,
  onSelectDate,
  selectedDateStr,
}: InterviewCalendarViewProps) {
  const navigate = useNavigate()

  // Navigation handlers
  const handlePrevMonth = () => {
    const next = new Date(currentDate.getFullYear(), currentDate.getMonth() - 1, 1)
    onDateChange(next)
  }

  const handleNextMonth = () => {
    const next = new Date(currentDate.getFullYear(), currentDate.getMonth() + 1, 1)
    onDateChange(next)
  }

  const handleToday = () => {
    onDateChange(new Date())
    onSelectDate(getLocalDateString(new Date()))
  }

  // Pre-index interviews by local Jakarta date YYYY-MM-DD
  const interviewsByDate = useMemo(() => {
    const map = new Map<string, Interview[]>()
    for (const iv of interviews) {
      if (!iv.scheduled_start) continue
      const dateKey = getLocalDateString(iv.scheduled_start)
      const list = map.get(dateKey) || []
      list.push(iv)
      map.set(dateKey, list)
    }
    // Sort each day's interviews by start time
    map.forEach((list) => {
      list.sort((a, b) => new Date(a.scheduled_start).getTime() - new Date(b.scheduled_start).getTime())
    })
    return map
  }, [interviews])

  // Compute calendar grid days
  const calendarDays = useMemo(() => {
    const year = currentDate.getFullYear()
    const month = currentDate.getMonth() // 0-indexed

    const firstDayOfMonth = new Date(year, month, 1)
    const lastDayOfMonth = new Date(year, month + 1, 0)

    // Monday = 0, ..., Sunday = 6
    const startDayOfWeek = (firstDayOfMonth.getDay() + 6) % 7
    const daysInMonth = lastDayOfMonth.getDate()

    const days: {
      date: Date
      dateStr: string
      dayNum: number
      isCurrentMonth: boolean
      isToday: boolean
    }[] = []

    const todayStr = getLocalDateString(new Date())

    // Leading days from previous month
    const prevMonthLastDay = new Date(year, month, 0).getDate()
    for (let i = startDayOfWeek - 1; i >= 0; i--) {
      const dayNum = prevMonthLastDay - i
      const d = new Date(year, month - 1, dayNum)
      const dateStr = getLocalDateString(d)
      days.push({
        date: d,
        dateStr,
        dayNum,
        isCurrentMonth: false,
        isToday: dateStr === todayStr,
      })
    }

    // Days in current month
    for (let i = 1; i <= daysInMonth; i++) {
      const d = new Date(year, month, i)
      const dateStr = getLocalDateString(d)
      days.push({
        date: d,
        dateStr,
        dayNum: i,
        isCurrentMonth: true,
        isToday: dateStr === todayStr,
      })
    }

    // Trailing days to complete weeks (multiples of 7)
    const remaining = (7 - (days.length % 7)) % 7
    for (let i = 1; i <= remaining; i++) {
      const d = new Date(year, month + 1, i)
      const dateStr = getLocalDateString(d)
      days.push({
        date: d,
        dateStr,
        dayNum: i,
        isCurrentMonth: false,
        isToday: dateStr === todayStr,
      })
    }

    return days
  }, [currentDate])

  return (
    <div className="bg-white rounded-card shadow-sm border border-slate-200 overflow-hidden flex flex-col">
      {/* Calendar Header */}
      <div className="px-5 py-4 border-b border-slate-200 flex flex-wrap items-center justify-between gap-3 bg-white">
        <div className="flex items-center gap-2">
          <CalendarIcon className="w-5 h-5 text-primary" />
          <h2 className="text-base font-mono font-bold text-slate-900">
            {formatMonthYear(currentDate)}
          </h2>
        </div>

        <div className="flex items-center gap-1.5">
          <button
            onClick={handleToday}
            className="px-2.5 py-1 text-xs font-semibold font-mono text-slate-700 bg-slate-100 hover:bg-slate-200 rounded border border-slate-300 transition-colors focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer mr-1"
            title="Jump to today"
          >
            Today
          </button>
          <div className="flex items-center border border-slate-300 rounded bg-white overflow-hidden">
            <button
              onClick={handlePrevMonth}
              aria-label="Previous month"
              className="p-1.5 text-slate-600 hover:bg-slate-100 hover:text-slate-900 transition-colors focus:outline-none cursor-pointer border-r border-slate-200"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <button
              onClick={handleNextMonth}
              aria-label="Next month"
              className="p-1.5 text-slate-600 hover:bg-slate-100 hover:text-slate-900 transition-colors focus:outline-none cursor-pointer"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      {/* Weekday column headers */}
      <div className="grid grid-cols-7 border-b border-slate-200 bg-slate-50 text-xs font-mono font-semibold text-slate-500 uppercase tracking-wider text-center py-2.5">
        {WEEKDAYS.map((day) => (
          <div key={day} className="truncate">
            {day}
          </div>
        ))}
      </div>

      {/* Calendar Month Grid */}
      <div className="grid grid-cols-7 divide-x divide-y divide-slate-200 bg-slate-200">
        {calendarDays.map((day) => {
          const dayInterviews = interviewsByDate.get(day.dateStr) || []
          const isSelected = selectedDateStr === day.dateStr
          const isToday = day.isToday
          const maxVisible = 3
          const overflowCount = dayInterviews.length - maxVisible

          return (
            <div
              key={day.dateStr}
              tabIndex={0}
              role="button"
              onClick={() => onSelectDate(day.dateStr)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' || e.key === ' ') {
                  e.preventDefault()
                  onSelectDate(day.dateStr)
                }
              }}
              className={`min-h-[105px] p-1.5 transition-colors flex flex-col justify-between group focus:outline-none focus:ring-2 focus:ring-primary focus:ring-inset cursor-pointer ${
                day.isCurrentMonth ? 'bg-white' : 'bg-slate-50/70 text-slate-400'
              } ${isSelected ? 'ring-2 ring-primary ring-inset z-10 bg-blue-50/20' : ''}`}
            >
              {/* Day Number Header */}
              <div className="flex items-center justify-between mb-1">
                <span
                  className={`text-xs font-mono font-semibold inline-flex items-center justify-center w-6 h-6 rounded-full transition-colors ${
                    isToday
                      ? 'bg-primary text-white font-bold shadow-xs'
                      : day.isCurrentMonth
                      ? 'text-slate-700 group-hover:text-primary'
                      : 'text-slate-400'
                  }`}
                >
                  {day.dayNum}
                </span>

                {dayInterviews.length > 0 && (
                  <span className="text-[10px] font-mono font-semibold text-slate-400 px-1">
                    {dayInterviews.length} {dayInterviews.length === 1 ? 'event' : 'events'}
                  </span>
                )}
              </div>

              {/* Event Pills inside Day Cell */}
              <div className="space-y-1 flex-1">
                {dayInterviews.slice(0, maxVisible).map((iv) => {
                  const candidateName = iv.candidate?.full_name || 'Candidate'
                  const timeText = formatTime(iv.scheduled_start)
                  const stageText = iv.stage_label || iv.stage
                  const isCompleted = iv.status === 'COMPLETED'
                  const isCancelled = iv.status === 'CANCELLED'

                  // Accessible label
                  const ariaLabel = `${timeText} ${stageText} with ${candidateName}, ${iv.status.toLowerCase()}`

                  return (
                    <button
                      key={iv.id}
                      type="button"
                      aria-label={ariaLabel}
                      onClick={(e) => {
                        e.stopPropagation()
                        navigate(`/interviews/${iv.id}`)
                      }}
                      className={`w-full text-left p-1 rounded text-[11px] leading-tight border transition-all truncate flex items-center gap-1 group/pill focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer ${
                        isCancelled
                          ? 'bg-slate-100 text-slate-400 border-slate-200 line-through'
                          : isCompleted
                          ? 'bg-emerald-50 text-emerald-800 border-emerald-200 hover:border-emerald-300'
                          : 'bg-blue-50 text-blue-900 border-blue-200 hover:border-blue-300 hover:shadow-xs'
                      }`}
                      title={`${timeText} — ${candidateName} (${stageText})`}
                    >
                      {/* Modality icon */}
                      {iv.interview_type === 'ONLINE' ? (
                        <Video className="w-2.5 h-2.5 shrink-0 opacity-70" />
                      ) : iv.interview_type === 'PHONE' ? (
                        <Phone className="w-2.5 h-2.5 shrink-0 opacity-70" />
                      ) : (
                        <MapPin className="w-2.5 h-2.5 shrink-0 opacity-70" />
                      )}

                      <span className="font-mono font-semibold shrink-0 text-[10px]">
                        {timeText}
                      </span>
                      <span className="truncate font-medium">{candidateName}</span>
                    </button>
                  )
                })}

                {/* Overflow count */}
                {overflowCount > 0 && (
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation()
                      onSelectDate(day.dateStr)
                    }}
                    className="w-full text-left px-1 text-[10px] font-semibold text-primary hover:underline focus:outline-none cursor-pointer"
                  >
                    +{overflowCount} more
                  </button>
                )}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

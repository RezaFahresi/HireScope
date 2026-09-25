import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import {
  Calendar,
  Clock,
  Video,
  MapPin,
  Phone,
  Users,
  Briefcase,
  Download,
  ExternalLink,
  ChevronRight,
} from 'lucide-react'
import {
  InterviewStatusBadge,
  InterviewStageBadge,
  InterviewTypeBadge,
  InterviewResultBadge,
} from '@/components/data-display/StatusBadge'
import { formatTimeRange, getLocalDateString, formatDayHeader } from '@/lib/utils'
import { interviewsApi } from '@/api/interviews'
import { toast } from 'sonner'
import type { Interview } from '@/types'

interface InterviewAgendaViewProps {
  interviews: Interview[]
  selectedDateStr?: string
}

export function InterviewAgendaView({
  interviews,
  selectedDateStr,
}: InterviewAgendaViewProps) {
  const [downloadingId, setDownloadingId] = useState<string | null>(null)

  // Group interviews by date string YYYY-MM-DD
  const groupedInterviews = useMemo(() => {
    const map = new Map<string, Interview[]>()
    for (const iv of interviews) {
      if (!iv.scheduled_start) continue
      const dateKey = getLocalDateString(iv.scheduled_start)
      const list = map.get(dateKey) || []
      list.push(iv)
      map.set(dateKey, list)
    }

    // Sort dates chronologically
    const sortedDateKeys = Array.from(map.keys()).sort()
    return sortedDateKeys.map((dateKey) => {
      const dayInterviews = map.get(dateKey) || []
      dayInterviews.sort(
        (a, b) => new Date(a.scheduled_start).getTime() - new Date(b.scheduled_start).getTime()
      )
      return {
        dateStr: dateKey,
        header: formatDayHeader(dateKey),
        items: dayInterviews,
      }
    })
  }, [interviews])

  const handleDownloadICS = async (e: React.MouseEvent, interview: Interview) => {
    e.preventDefault()
    e.stopPropagation()
    setDownloadingId(interview.id)
    try {
      await interviewsApi.downloadIcs(interview.id)
      toast.success('Calendar file (.ics) downloaded.')
    } catch {
      toast.error('Unable to generate calendar file.')
    } finally {
      setDownloadingId(null)
    }
  }

  if (groupedInterviews.length === 0) {
    return (
      <div className="bg-white rounded-card shadow-sm border border-slate-200 p-12 text-center">
        <Calendar className="w-10 h-10 text-slate-300 mx-auto mb-3" />
        <h3 className="text-sm font-bold text-slate-700 uppercase tracking-wide">
          No Interviews Scheduled
        </h3>
        <p className="text-xs text-slate-500 mt-1 max-w-sm mx-auto">
          There are no interviews matching the current filters or date range.
        </p>
        <Link
          to="/interviews/new"
          className="mt-4 inline-flex items-center gap-1.5 px-3.5 py-1.5 text-xs font-semibold bg-cta text-white rounded-lg hover:opacity-90 transition-all cursor-pointer"
        >
          + Schedule Interview
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      {groupedInterviews.map((group) => {
        const isSelectedDate = selectedDateStr === group.dateStr

        return (
          <div
            key={group.dateStr}
            className={`space-y-3 transition-colors ${
              isSelectedDate ? 'p-3 bg-blue-50/50 rounded-card border border-blue-200' : ''
            }`}
          >
            {/* Date Group Header */}
            <div className="flex items-center gap-2 border-b border-slate-200 pb-2">
              <Calendar className="w-4 h-4 text-primary" />
              <h3 className="text-xs font-mono font-bold text-slate-800 uppercase tracking-wide">
                {group.header}
              </h3>
              <span className="text-[11px] font-mono text-slate-400">
                ({group.items.length} {group.items.length === 1 ? 'interview' : 'interviews'})
              </span>
            </div>

            {/* List of Interview Cards for the Day */}
            <div className="space-y-2.5">
              {group.items.map((iv) => {
                const candidateName = iv.candidate?.full_name || 'Candidate'
                const jobTitle = iv.job?.title || 'Job'
                const jobCode = iv.job?.code

                return (
                  <div
                    key={iv.id}
                    className="bg-white rounded-card border border-slate-200 p-4 hover:border-slate-300 hover:shadow-xs transition-all flex flex-col md:flex-row md:items-center justify-between gap-4"
                  >
                    {/* Left details */}
                    <div className="space-y-2 flex-1 min-w-0">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="font-semibold text-sm text-slate-900 truncate">
                          {iv.title}
                        </span>
                        <InterviewStageBadge stage={iv.stage} />
                        <InterviewStatusBadge status={iv.status} />
                        <InterviewTypeBadge type={iv.interview_type} />
                        {iv.result && <InterviewResultBadge result={iv.result} />}
                      </div>

                      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-600 font-mono">
                        <span className="flex items-center gap-1.5 text-slate-700 font-semibold">
                          <Clock className="w-3.5 h-3.5 text-slate-400" />
                          {formatTimeRange(iv.scheduled_start, iv.scheduled_end)}
                        </span>

                        <span className="flex items-center gap-1 text-slate-600">
                          <span className="text-slate-400">Candidate:</span>
                          <span className="font-medium text-slate-800">{candidateName}</span>
                        </span>

                        <span className="flex items-center gap-1 text-slate-600">
                          <Briefcase className="w-3.5 h-3.5 text-slate-400" />
                          <span className="font-medium text-slate-800">{jobTitle}</span>
                          {jobCode && <span className="text-slate-400">({jobCode})</span>}
                        </span>
                      </div>

                      {/* Location / Meeting URL / Interviewers */}
                      <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-[11px] text-slate-500">
                        {iv.interview_type === 'ONLINE' && iv.meeting_url && (
                          <a
                            href={iv.meeting_url}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="inline-flex items-center gap-1 text-primary hover:underline font-medium"
                          >
                            <Video className="w-3 h-3 text-primary" />
                            Join Video Meeting
                            <ExternalLink className="w-2.5 h-2.5" />
                          </a>
                        )}

                        {iv.interview_type === 'ONSITE' && iv.location && (
                          <span className="inline-flex items-center gap-1 text-slate-600">
                            <MapPin className="w-3 h-3 text-slate-400" />
                            {iv.location}
                          </span>
                        )}

                        {iv.interview_type === 'PHONE' && (
                          <span className="inline-flex items-center gap-1 text-slate-600">
                            <Phone className="w-3 h-3 text-slate-400" />
                            Phone Interview
                          </span>
                        )}

                        {iv.interviewers?.length > 0 && (
                          <span className="inline-flex items-center gap-1 text-slate-500">
                            <Users className="w-3 h-3 text-slate-400" />
                            {iv.interviewers.map((i) => i.name).join(', ')}
                          </span>
                        )}
                      </div>
                    </div>

                    {/* Right actions */}
                    <div className="flex items-center gap-2 shrink-0 self-end md:self-center">
                      <button
                        type="button"
                        onClick={(e) => handleDownloadICS(e, iv)}
                        disabled={downloadingId === iv.id}
                        className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold font-mono text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer disabled:opacity-50"
                        title="Download .ics calendar file"
                      >
                        <Download className="w-3.5 h-3.5 text-slate-500" />
                        {downloadingId === iv.id ? 'Exporting…' : 'Add to Calendar'}
                      </button>

                      <Link
                        to={`/interviews/${iv.id}`}
                        className="inline-flex items-center gap-1 px-3 py-1.5 text-xs font-semibold bg-primary text-white rounded-lg hover:opacity-90 transition-all focus:outline-none focus:ring-1 focus:ring-primary cursor-pointer"
                      >
                        View Details
                        <ChevronRight className="w-3.5 h-3.5" />
                      </Link>
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )
      })}
    </div>
  )
}

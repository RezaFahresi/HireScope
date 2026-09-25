import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { useParams, Link } from 'react-router-dom'
import {
  Calendar,
  Video,
  MapPin,
  Phone,
  Briefcase,
  User,
  ArrowLeft,
  ExternalLink,
  CheckCircle2,
  AlertTriangle,
  RotateCcw,
  FileText,
  Mail,
  Download,
} from 'lucide-react'
import { interviewsApi } from '@/api/interviews'
import { toast } from 'sonner'
import {
  InterviewStatusBadge,
  InterviewStageBadge,
  InterviewTypeBadge,
  InterviewResultBadge,
} from '@/components/data-display/StatusBadge'
import { LoadingState, ErrorState } from '@/components/feedback'
import { formatTimeRange, formatDateTime, formatEmploymentType } from '@/lib/utils'
import { RescheduleDialog } from './components/RescheduleDialog'
import { CancelDialog } from './components/CancelDialog'
import { CompleteDialog } from './components/CompleteDialog'
import { EmailCommunicationSection } from './components/EmailCommunicationSection'
import { ExternalCalendarSection } from './components/ExternalCalendarSection'

export function InterviewDetailPage() {
  const { id } = useParams<{ id: string }>()

  const [isRescheduleOpen, setIsRescheduleOpen] = useState(false)
  const [isCancelOpen, setIsCancelOpen] = useState(false)
  const [isCompleteOpen, setIsCompleteOpen] = useState(false)
  const [isDownloadingIcs, setIsDownloadingIcs] = useState(false)

  const { data: interview, isLoading, isError, refetch } = useQuery({
    queryKey: ['interview', id],
    queryFn: () => interviewsApi.get(id!),
    enabled: !!id,
  })

  const handleDownloadICS = async () => {
    if (!interview) return
    setIsDownloadingIcs(true)
    try {
      await interviewsApi.downloadIcs(interview.id)
      toast.success('Calendar file (.ics) downloaded.')
    } catch {
      toast.error('Unable to generate calendar file.')
    } finally {
      setIsDownloadingIcs(false)
    }
  }

  if (isLoading) {
    return <LoadingState message="Loading interview details…" className="py-24" />
  }

  if (isError || !interview) {
    return (
      <ErrorState
        message="Interview session could not be found or failed to load."
        onRetry={refetch}
        className="py-24"
      />
    )
  }

  const isScheduled = interview.status === 'SCHEDULED'
  const isCompleted = interview.status === 'COMPLETED'
  const isCancelled = interview.status === 'CANCELLED'

  const candidateName = interview.candidate?.full_name || 'Candidate'
  const candidateInitial = candidateName.trim().charAt(0).toUpperCase() || 'C'

  return (
    <div className="max-w-5xl mx-auto space-y-6 pb-16">
      {/* Back Link */}
      <Link
        to="/interviews"
        className="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground transition-colors"
      >
        <ArrowLeft className="w-3.5 h-3.5" />
        Back to Interviews
      </Link>

      {/* Header Banner */}
      <div className="rounded-lg border border-border bg-card p-5 shadow-xs">
        <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
          <div className="space-y-1.5 min-w-0">
            <div className="flex flex-wrap items-center gap-2">
              <InterviewStatusBadge status={interview.status} />
              <InterviewStageBadge stage={interview.stage} />
              <InterviewTypeBadge type={interview.interview_type} />
            </div>
            <h1 className="text-xl sm:text-2xl font-bold font-mono text-foreground truncate">
              {interview.title}
            </h1>
            <p className="text-xs text-muted-foreground flex items-center gap-1.5">
              <span>Candidate:</span>
              <Link
                to={`/candidates/${interview.candidate_id}`}
                className="font-medium text-foreground hover:text-primary hover:underline"
              >
                {candidateName}
              </Link>
              <span>&bull;</span>
              <span>Requisition:</span>
              <Link
                to={`/jobs/${interview.job_id}`}
                className="font-medium text-foreground hover:text-primary hover:underline"
              >
                {interview.job?.title || 'Job Requisition'}
              </Link>
            </p>
          </div>

          {/* Action Buttons */}
          <div className="flex flex-wrap items-center gap-2 shrink-0">
            {/* Add to Calendar (.ics) download */}
            <button
              type="button"
              onClick={handleDownloadICS}
              disabled={isDownloadingIcs}
              className="inline-flex items-center gap-1.5 px-3 py-2 text-xs font-medium rounded-md border border-border bg-background text-foreground hover:bg-muted transition-colors cursor-pointer disabled:opacity-50"
              title="Export to iCalendar (.ics)"
            >
              <Download className="w-3.5 h-3.5 text-muted-foreground" />
              {isDownloadingIcs ? 'Exporting…' : 'Add to Calendar'}
            </button>

            {isScheduled && (
              <>
                {interview.interview_type === 'ONLINE' && interview.meeting_url && (
                  <a
                    href={interview.meeting_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold rounded-md bg-blue-600 text-white hover:bg-blue-700 transition-colors shadow-2xs"
                  >
                    <Video className="w-3.5 h-3.5" />
                    Join Meeting
                    <ExternalLink className="w-3 h-3 ml-0.5" />
                  </a>
                )}

                <button
                  onClick={() => setIsRescheduleOpen(true)}
                  className="inline-flex items-center gap-1.5 px-3 py-2 text-xs font-medium rounded-md border border-border bg-background text-foreground hover:bg-muted transition-colors cursor-pointer"
                >
                  <RotateCcw className="w-3.5 h-3.5 text-muted-foreground" />
                  Reschedule
                </button>

                <button
                  onClick={() => setIsCancelOpen(true)}
                  className="inline-flex items-center gap-1.5 px-3 py-2 text-xs font-medium rounded-md border border-red-200 bg-red-50 text-red-700 hover:bg-red-100 transition-colors cursor-pointer"
                >
                  <AlertTriangle className="w-3.5 h-3.5" />
                  Cancel
                </button>

                <button
                  onClick={() => setIsCompleteOpen(true)}
                  className="inline-flex items-center gap-1.5 px-3.5 py-2 text-xs font-semibold rounded-md bg-emerald-600 text-white hover:bg-emerald-700 transition-colors shadow-2xs cursor-pointer"
                >
                  <CheckCircle2 className="w-3.5 h-3.5" />
                  Mark Completed
                </button>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Outcome / Cancellation Card if applicable */}
      {isCompleted && (
        <div className="rounded-lg border border-emerald-200/80 bg-emerald-50/40 p-5 shadow-xs space-y-3">
          <div className="flex items-center justify-between pb-2 border-b border-emerald-200/60">
            <div className="flex items-center gap-2">
              <CheckCircle2 className="w-4 h-4 text-emerald-700" />
              <h3 className="text-sm font-semibold text-emerald-950">Interview Completed & Assessed</h3>
            </div>
            {interview.result && <InterviewResultBadge result={interview.result} />}
          </div>

          <div className="text-xs text-emerald-900/90 space-y-2">
            {interview.feedback ? (
              <div className="bg-white border border-emerald-200/70 p-3.5 rounded-md text-foreground whitespace-pre-wrap leading-relaxed shadow-2xs font-sans">
                {interview.feedback}
              </div>
            ) : (
              <p className="italic text-muted-foreground">No written evaluation notes recorded.</p>
            )}

            <div className="flex items-center justify-between text-[11px] text-emerald-800 pt-1">
              <span>Completed by: {interview.completed_name || 'Recruiter'}</span>
              <span>At: {formatDateTime(interview.completed_at)}</span>
            </div>
          </div>
        </div>
      )}

      {isCancelled && (
        <div className="rounded-lg border border-red-200 bg-red-50/40 p-5 shadow-xs space-y-3">
          <div className="flex items-center gap-2 pb-2 border-b border-red-200">
            <AlertTriangle className="w-4 h-4 text-red-600" />
            <h3 className="text-sm font-semibold text-red-950">Interview Session Cancelled</h3>
          </div>

          <div className="text-xs text-red-900 space-y-1.5">
            <p className="font-medium text-slate-700">Reason for cancellation:</p>
            <div className="bg-white border border-red-200 p-3 rounded-md text-foreground shadow-2xs">
              {interview.cancellation_reason || 'No cancellation reason specified.'}
            </div>

            <div className="flex items-center justify-between text-[11px] text-red-700/80 pt-1">
              <span>Cancelled by: {interview.cancelled_name || 'Recruiter'}</span>
              <span>At: {formatDateTime(interview.cancelled_at)}</span>
            </div>
          </div>
        </div>
      )}

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-5">
        {/* Left Column (2 cols): Schedule & Modality & Notes */}
        <div className="md:col-span-2 space-y-5">
          {/* Scheduling Details Card */}
          <div className="rounded-lg border border-border bg-card p-5 space-y-4 shadow-xs text-xs">
            <div className="flex items-center gap-2 pb-3 border-b border-border/60">
              <Calendar className="w-4 h-4 text-primary" />
              <h2 className="text-sm font-semibold text-foreground">Schedule Details</h2>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <span className="text-[11px] text-muted-foreground uppercase font-medium block">
                  Date & Time
                </span>
                <span className="font-semibold text-foreground font-mono text-sm block mt-0.5">
                  {formatTimeRange(interview.scheduled_start, interview.scheduled_end)}
                </span>
                <span className="text-[11px] text-muted-foreground font-mono">
                  {interview.timezone || 'Asia/Jakarta (GMT+7)'}
                </span>
              </div>

              <div>
                <span className="text-[11px] text-muted-foreground uppercase font-medium block">
                  Modality & Access
                </span>
                {interview.interview_type === 'ONLINE' && (
                  <div className="mt-1">
                    {interview.meeting_url ? (
                      <a
                        href={interview.meeting_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-primary hover:underline font-mono break-all inline-flex items-center gap-1"
                      >
                        {interview.meeting_url}
                        <ExternalLink className="w-3 h-3 shrink-0" />
                      </a>
                    ) : (
                      <span className="text-muted-foreground">Online (No link provided)</span>
                    )}
                  </div>
                )}
                {interview.interview_type === 'ONSITE' && (
                  <p className="mt-1 font-medium text-foreground flex items-center gap-1">
                    <MapPin className="w-3.5 h-3.5 text-muted-foreground" />
                    {interview.location || 'Onsite location'}
                  </p>
                )}
                {interview.interview_type === 'PHONE' && (
                  <p className="mt-1 font-medium text-foreground flex items-center gap-1">
                    <Phone className="w-3.5 h-3.5 text-muted-foreground" />
                    Phone Screening Call
                  </p>
                )}
              </div>
            </div>

            {/* Assigned Interviewers */}
            <div className="pt-3 border-t border-border/60">
              <span className="text-[11px] text-muted-foreground uppercase font-medium block mb-2">
                Assigned Interviewer(s)
              </span>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                {interview.interviewers?.map((iv) => (
                  <div
                    key={iv.id}
                    className="flex items-center gap-2 p-2 rounded-md bg-muted/40 border border-border/60"
                  >
                    <div className="w-6 h-6 rounded-full bg-primary/10 text-primary font-semibold text-[10px] flex items-center justify-center shrink-0">
                      {(iv.name?.charAt(0) || 'U').toUpperCase()}
                    </div>
                    <div className="min-w-0">
                      <p className="font-semibold text-foreground truncate">{iv.name}</p>
                      <p className="text-[10px] text-muted-foreground truncate">{iv.email}</p>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          {/* Notes Card */}
          <div className="rounded-lg border border-border bg-card p-5 space-y-3 shadow-xs text-xs">
            <div className="flex items-center gap-2 pb-2 border-b border-border/60">
              <FileText className="w-4 h-4 text-primary" />
              <h2 className="text-sm font-semibold text-foreground">Preparation Notes</h2>
            </div>
            {interview.notes ? (
              <p className="text-foreground whitespace-pre-wrap leading-relaxed">
                {interview.notes}
              </p>
            ) : (
              <p className="text-muted-foreground italic">No preparation notes recorded.</p>
            )}
          </div>

          {/* Email Communication Section (Step 12B-2) */}
          <EmailCommunicationSection interview={interview} />

          {/* External Calendar Synchronization (Step 12B-3) */}
          <ExternalCalendarSection
            interviewId={interview.id}
            interviewStatus={interview.status}
          />
        </div>

        {/* Right Column (1 col): Candidate & Job Cards */}
        <div className="space-y-5 text-xs">
          {/* Candidate Card */}
          <div className="rounded-lg border border-border bg-card p-5 space-y-3 shadow-xs">
            <div className="flex items-center gap-2 pb-2 border-b border-border/60">
              <User className="w-4 h-4 text-primary" />
              <h3 className="text-sm font-semibold text-foreground">Candidate</h3>
            </div>

            <div className="flex items-center gap-2.5">
              <div className="w-9 h-9 rounded-full bg-primary/10 text-primary font-bold text-sm flex items-center justify-center shrink-0">
                {candidateInitial}
              </div>
              <div className="min-w-0">
                <Link
                  to={`/candidates/${interview.candidate_id}`}
                  className="font-bold text-sm text-foreground hover:text-primary hover:underline truncate block"
                >
                  {candidateName}
                </Link>
                {interview.candidate?.headline && (
                  <p className="text-[11px] text-muted-foreground truncate">
                    {interview.candidate.headline}
                  </p>
                )}
              </div>
            </div>

            <div className="space-y-1.5 pt-2 border-t border-border/40 text-muted-foreground text-[11px]">
              {interview.candidate?.email && (
                <div className="flex items-center gap-1.5 truncate">
                  <Mail className="w-3 h-3 text-muted-foreground shrink-0" />
                  <span className="truncate">{interview.candidate.email}</span>
                </div>
              )}
              {interview.candidate?.phone && (
                <div className="flex items-center gap-1.5 truncate">
                  <Phone className="w-3 h-3 text-muted-foreground shrink-0" />
                  <span>{interview.candidate.phone}</span>
                </div>
              )}
            </div>

            <div className="pt-2">
              <Link
                to={`/jobs/${interview.job_id}/candidates/${interview.candidate_id}/review`}
                className="w-full inline-flex items-center justify-center gap-1 py-1.5 px-3 rounded border border-border bg-muted/30 text-foreground hover:bg-muted text-xs font-medium transition-colors"
              >
                Open Review Workflow &rarr;
              </Link>
            </div>
          </div>

          {/* Job Card */}
          <div className="rounded-lg border border-border bg-card p-5 space-y-3 shadow-xs">
            <div className="flex items-center gap-2 pb-2 border-b border-border/60">
              <Briefcase className="w-4 h-4 text-primary" />
              <h3 className="text-sm font-semibold text-foreground">Job Requisition</h3>
            </div>

            <div>
              <Link
                to={`/jobs/${interview.job_id}`}
                className="font-bold text-foreground hover:text-primary hover:underline block"
              >
                {interview.job?.title || 'Requisition'}
              </Link>
              <div className="flex items-center gap-2 text-[11px] text-muted-foreground mt-0.5">
                <span className="font-mono">{interview.job?.code}</span>
                <span>&bull;</span>
                <span>{interview.job?.department || 'General'}</span>
              </div>
            </div>

            <div className="text-[11px] text-muted-foreground space-y-1 pt-2 border-t border-border/40">
              {interview.job?.employment_type && (
                <p>Type: {formatEmploymentType(interview.job.employment_type)}</p>
              )}
              {interview.job?.location && <p>Location: {interview.job.location}</p>}
            </div>

            <div className="pt-2">
              <Link
                to={`/jobs/${interview.job_id}/candidates`}
                className="w-full inline-flex items-center justify-center gap-1 py-1.5 px-3 rounded border border-border bg-muted/30 text-foreground hover:bg-muted text-xs font-medium transition-colors"
              >
                View Job Pipeline &rarr;
              </Link>
            </div>
          </div>
        </div>
      </div>

      {/* Dialogs */}
      <RescheduleDialog
        isOpen={isRescheduleOpen}
        onClose={() => setIsRescheduleOpen(false)}
        interview={interview}
      />
      <CancelDialog
        isOpen={isCancelOpen}
        onClose={() => setIsCancelOpen(false)}
        interview={interview}
      />
      <CompleteDialog
        isOpen={isCompleteOpen}
        onClose={() => setIsCompleteOpen(false)}
        interview={interview}
      />
    </div>
  )
}

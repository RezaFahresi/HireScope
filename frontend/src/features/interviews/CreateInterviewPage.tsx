import { useState, useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useSearchParams, Link } from 'react-router-dom'
import {
  Calendar,
  Clock,
  Video,
  MapPin,
  Phone,
  Users,
  Briefcase,
  ArrowLeft,
  AlertCircle,
  Mail,
} from 'lucide-react'
import { interviewsApi } from '@/api/interviews'
import { jobsApi } from '@/api/jobs'
import { reviewsApi } from '@/api/reviews'
import { getApiError } from '@/api/client'
import { toast } from 'sonner'
import type { InterviewStage, InterviewType } from '@/types'

export function CreateInterviewPage() {
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const queryClient = useQueryClient()

  const prefillJobId = searchParams.get('job_id') || ''
  const prefillCandidateId = searchParams.get('candidate_id') || ''

  // Form state
  const [jobId, setJobId] = useState(prefillJobId)
  const [candidateId, setCandidateId] = useState(prefillCandidateId)
  const [title, setTitle] = useState('')
  const [stage, setStage] = useState<InterviewStage>('TECHNICAL_INTERVIEW')
  const [interviewType, setInterviewType] = useState<InterviewType>('ONLINE')
  const [date, setDate] = useState(() => {
    const tomorrow = new Date()
    tomorrow.setDate(tomorrow.getDate() + 1)
    const yyyy = tomorrow.getFullYear()
    const mm = String(tomorrow.getMonth() + 1).padStart(2, '0')
    const dd = String(tomorrow.getDate()).padStart(2, '0')
    return `${yyyy}-${mm}-${dd}`
  })
  const [startTime, setStartTime] = useState('10:00')
  const [endTime, setEndTime] = useState('11:00')
  const [meetingUrl, setMeetingUrl] = useState('')
  const [location, setLocation] = useState('')
  const [selectedInterviewers, setSelectedInterviewers] = useState<string[]>([])
  const [notes, setNotes] = useState('')
  const [sendCandidateInvitation, setSendCandidateInvitation] = useState(true)
  const [sendInterviewerInvitation, setSendInterviewerInvitation] = useState(true)
  const [formError, setFormError] = useState('')

  // Queries
  const { data: jobsData, isLoading: jobsLoading } = useQuery({
    queryKey: ['jobs', { limit: 100 }],
    queryFn: () => jobsApi.list({ limit: 100 }),
  })

  const { data: jobCandidatesData, isLoading: candidatesLoading } = useQuery({
    queryKey: ['jobCandidates', jobId],
    queryFn: () => reviewsApi.listCandidatesByJob(jobId, { limit: 100 }),
    enabled: !!jobId,
  })

  const { data: usersData, isLoading: usersLoading } = useQuery({
    queryKey: ['users'],
    queryFn: interviewsApi.getUsers,
  })

  // When job changes, reset candidate unless prefilled
  const handleJobChange = (newJobId: string) => {
    setJobId(newJobId)
    if (newJobId !== prefillJobId) {
      setCandidateId('')
    }
  }

  // Prepopulate title when candidate or stage changes if title is empty or generic
  useEffect(() => {
    if (jobCandidatesData?.items && candidateId) {
      const selectedCand = jobCandidatesData.items.find((jc) => jc.candidate_id === candidateId)
      const candName = selectedCand?.candidate?.full_name || 'Candidate'
      const stageLabelMap: Record<InterviewStage, string> = {
        PHONE_SCREEN: 'Phone Screening',
        HR_INTERVIEW: 'HR Interview',
        TECHNICAL_INTERVIEW: 'Technical Assessment',
        MANAGER_INTERVIEW: 'Manager Interview',
        FINAL_INTERVIEW: 'Final Interview',
        OTHER: 'Interview Session',
      }
      const defaultTitle = `${stageLabelMap[stage] || 'Interview'} - ${candName}`
      if (!title || title.includes('Interview') || title.includes('Assessment')) {
        setTitle(defaultTitle)
      }
    }
  }, [candidateId, stage, jobCandidatesData])

  // Select current recruiter as interviewer automatically once users loaded
  useEffect(() => {
    if (usersData?.length && selectedInterviewers.length === 0) {
      setSelectedInterviewers([usersData[0].id])
    }
  }, [usersData])

  const toggleInterviewer = (userId: string) => {
    setSelectedInterviewers((prev) =>
      prev.includes(userId) ? prev.filter((id) => id !== userId) : [...prev, userId]
    )
  }

  const createMutation = useMutation({
    mutationFn: async () => {
      setFormError('')

      if (!jobId) throw new Error('Please select a job requisition.')
      if (!candidateId) throw new Error('Please select an associated candidate.')
      if (!title.trim()) throw new Error('Please enter an interview title.')
      if (!date || !startTime || !endTime) {
        throw new Error('Please enter complete date and time range.')
      }

      const scheduledStart = new Date(`${date}T${startTime}:00`).toISOString()
      const scheduledEnd = new Date(`${date}T${endTime}:00`).toISOString()

      if (new Date(scheduledEnd) <= new Date(scheduledStart)) {
        throw new Error('Scheduled end time must be after scheduled start time.')
      }

      if (interviewType === 'ONLINE' && !meetingUrl.trim()) {
        throw new Error('Meeting URL is required for online interviews.')
      }
      if (interviewType === 'ONSITE' && !location.trim()) {
        throw new Error('Physical location is required for onsite interviews.')
      }
      if (selectedInterviewers.length === 0) {
        throw new Error('Please select at least one interviewer.')
      }

      return interviewsApi.create({
        job_id: jobId,
        candidate_id: candidateId,
        title: title.trim(),
        stage,
        interview_type: interviewType,
        scheduled_start: scheduledStart,
        scheduled_end: scheduledEnd,
        timezone: 'Asia/Jakarta',
        meeting_url: interviewType === 'ONLINE' ? meetingUrl.trim() : undefined,
        location: interviewType === 'ONSITE' ? location.trim() : undefined,
        notes: notes.trim() || undefined,
        interviewer_ids: selectedInterviewers,
        send_candidate_invitation: sendCandidateInvitation,
        send_interviewer_invitation: sendInterviewerInvitation,
      })
    },
    onSuccess: (data) => {
      toast.success('Interview scheduled successfully!')
      queryClient.invalidateQueries({ queryKey: ['interviews'] })
      queryClient.invalidateQueries({ queryKey: ['candidate-interviews', candidateId] })
      queryClient.invalidateQueries({ queryKey: ['job-interviews', jobId] })
      navigate(`/interviews/${data.id}`)
    },
    onError: (err) => {
      const msg = getApiError(err)
      setFormError(msg)
      toast.error(msg)
    },
  })

  const candidateOptions = jobCandidatesData?.items || []

  return (
    <div className="max-w-4xl mx-auto space-y-6 pb-16">
      {/* Header */}
      <div>
        <Link
          to="/interviews"
          className="inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground mb-3 transition-colors"
        >
          <ArrowLeft className="w-3.5 h-3.5" />
          Back to Interviews
        </Link>
        <div className="flex items-center gap-2">
          <Calendar className="h-5 w-5 text-primary" />
          <h1 className="text-xl sm:text-2xl font-bold font-mono tracking-tight text-foreground">
            Schedule Candidate Interview
          </h1>
        </div>
        <p className="text-xs sm:text-sm text-muted-foreground mt-1">
          Coordinate an internal evaluation session with interviewer availability.
        </p>
      </div>

      {formError && (
        <div className="p-3.5 bg-red-50 border border-red-200 rounded-lg text-xs text-red-700 flex items-start gap-2.5">
          <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
          <div>
            <p className="font-semibold">Scheduling Error</p>
            <p className="mt-0.5">{formError}</p>
          </div>
        </div>
      )}

      <form
        onSubmit={(e) => {
          e.preventDefault()
          createMutation.mutate()
        }}
        className="space-y-6"
      >
        {/* Section 1: Candidate & Requisition Context */}
        <div className="rounded-lg border border-border bg-card p-5 space-y-4 shadow-xs">
          <div className="flex items-center gap-2 pb-3 border-b border-border/60">
            <Briefcase className="h-4 w-4 text-primary" />
            <h2 className="text-sm font-semibold text-foreground">Requisition & Candidate</h2>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4 text-xs">
            {/* Job Requisition */}
            <div className="space-y-1.5">
              <label className="font-medium text-foreground block">
                Job Requisition <span className="text-red-500">*</span>
              </label>
              <select
                value={jobId}
                onChange={(e) => handleJobChange(e.target.value)}
                disabled={jobsLoading}
                className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs"
                required
              >
                <option value="">Select a Job Vacancy...</option>
                {jobsData?.items?.map((j) => (
                  <option key={j.id} value={j.id}>
                    {j.title} ({j.code}) — {j.department || 'General'}
                  </option>
                ))}
              </select>
            </div>

            {/* Candidate Selector (Dependent on Job) */}
            <div className="space-y-1.5">
              <label className="font-medium text-foreground block">
                Candidate <span className="text-red-500">*</span>
              </label>
              <select
                value={candidateId}
                onChange={(e) => setCandidateId(e.target.value)}
                disabled={!jobId || candidatesLoading}
                className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs disabled:opacity-50"
                required
              >
                <option value="">
                  {!jobId
                    ? 'Select a job first...'
                    : candidatesLoading
                    ? 'Loading associated candidates...'
                    : candidateOptions.length === 0
                    ? 'No candidates in this job pipeline'
                    : 'Select Candidate...'}
                </option>
                {candidateOptions.map((jc) => {
                  const name = jc.candidate?.full_name || 'Candidate'
                  return (
                    <option key={jc.candidate_id} value={jc.candidate_id}>
                      {name} ({jc.status})
                    </option>
                  )
                })}
              </select>
              {jobId && candidateOptions.length === 0 && !candidatesLoading && (
                <p className="text-[11px] text-amber-600">
                  This job currently has no active candidates in its pipeline.
                </p>
              )}
            </div>

            {/* Title */}
            <div className="md:col-span-2 space-y-1.5">
              <label className="font-medium text-foreground block">
                Interview Title <span className="text-red-500">*</span>
              </label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="e.g. Technical Interview - Maya Lin"
                className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs"
                required
              />
            </div>

            {/* Stage */}
            <div className="space-y-1.5">
              <label className="font-medium text-foreground block">
                Interview Stage <span className="text-red-500">*</span>
              </label>
              <select
                value={stage}
                onChange={(e) => setStage(e.target.value as InterviewStage)}
                className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs"
                required
              >
                <option value="PHONE_SCREEN">Phone Screen</option>
                <option value="HR_INTERVIEW">HR Interview</option>
                <option value="TECHNICAL_INTERVIEW">Technical Interview</option>
                <option value="MANAGER_INTERVIEW">Manager Interview</option>
                <option value="FINAL_INTERVIEW">Final Interview</option>
                <option value="OTHER">Other</option>
              </select>
            </div>

            {/* Interview Modality */}
            <div className="space-y-1.5">
              <label className="font-medium text-foreground block">
                Interview Type <span className="text-red-500">*</span>
              </label>
              <div className="grid grid-cols-3 gap-2">
                {[
                  { value: 'ONLINE', label: 'Online', icon: Video },
                  { value: 'ONSITE', label: 'Onsite', icon: MapPin },
                  { value: 'PHONE', label: 'Phone', icon: Phone },
                ].map((typeOption) => {
                  const Icon = typeOption.icon
                  const isSelected = interviewType === typeOption.value
                  return (
                    <button
                      type="button"
                      key={typeOption.value}
                      onClick={() => setInterviewType(typeOption.value as InterviewType)}
                      className={`flex items-center justify-center gap-1.5 py-2 px-3 rounded-md border text-xs font-medium cursor-pointer transition-colors ${
                        isSelected
                          ? 'border-primary bg-primary/10 text-primary font-semibold'
                          : 'border-border bg-background text-muted-foreground hover:text-foreground'
                      }`}
                    >
                      <Icon className="w-3.5 h-3.5" />
                      {typeOption.label}
                    </button>
                  )
                })}
              </div>
            </div>

            {/* Conditional Modality Fields */}
            {interviewType === 'ONLINE' && (
              <div className="md:col-span-2 space-y-1.5">
                <label className="font-medium text-foreground block">
                  Meeting URL <span className="text-red-500">*</span>
                </label>
                <input
                  type="url"
                  value={meetingUrl}
                  onChange={(e) => setMeetingUrl(e.target.value)}
                  placeholder="https://meet.google.com/xxx-yyyy-zzz or Zoom link"
                  className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs font-mono"
                  required
                />
              </div>
            )}

            {interviewType === 'ONSITE' && (
              <div className="md:col-span-2 space-y-1.5">
                <label className="font-medium text-foreground block">
                  Physical Location / Office Room <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={location}
                  onChange={(e) => setLocation(e.target.value)}
                  placeholder="e.g. Jakarta HQ, 5th Floor - Meeting Room Beta"
                  className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs"
                  required
                />
              </div>
            )}
          </div>
        </div>

        {/* Section 2: Date & Time */}
        <div className="rounded-lg border border-border bg-card p-5 space-y-4 shadow-xs">
          <div className="flex items-center justify-between pb-3 border-b border-border/60">
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-primary" />
              <h2 className="text-sm font-semibold text-foreground">Schedule & Timing</h2>
            </div>
            <span className="text-xs font-mono text-muted-foreground bg-muted px-2 py-0.5 rounded">
              Timezone: Asia/Jakarta (GMT+7)
            </span>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 text-xs">
            <div className="space-y-1.5">
              <label className="font-medium text-foreground block">
                Date <span className="text-red-500">*</span>
              </label>
              <input
                type="date"
                value={date}
                onChange={(e) => setDate(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs"
                required
              />
            </div>

            <div className="space-y-1.5">
              <label className="font-medium text-foreground block">
                Start Time <span className="text-red-500">*</span>
              </label>
              <input
                type="time"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs"
                required
              />
            </div>

            <div className="space-y-1.5">
              <label className="font-medium text-foreground block">
                End Time <span className="text-red-500">*</span>
              </label>
              <input
                type="time"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
                className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs"
                required
              />
            </div>
          </div>
        </div>

        {/* Section 3: Interviewers Assignment */}
        <div className="rounded-lg border border-border bg-card p-5 space-y-4 shadow-xs">
          <div className="flex items-center justify-between pb-3 border-b border-border/60">
            <div className="flex items-center gap-2">
              <Users className="h-4 w-4 text-primary" />
              <h2 className="text-sm font-semibold text-foreground">Assigned Interviewer(s)</h2>
            </div>
            <span className="text-xs text-muted-foreground">Select one or more team members</span>
          </div>

          {usersLoading ? (
            <div className="py-4 text-center text-xs text-muted-foreground animate-pulse">
              Loading team members…
            </div>
          ) : (
            <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-2.5">
              {usersData?.map((u) => {
                const isChecked = selectedInterviewers.includes(u.id)
                return (
                  <label
                    key={u.id}
                    className={`flex items-center gap-2.5 p-2.5 rounded-md border text-xs cursor-pointer transition-colors ${
                      isChecked
                        ? 'border-primary bg-primary/5 text-foreground'
                        : 'border-border bg-background text-muted-foreground hover:bg-muted/40'
                    }`}
                  >
                    <input
                      type="checkbox"
                      checked={isChecked}
                      onChange={() => toggleInterviewer(u.id)}
                      className="accent-primary rounded"
                    />
                    <div className="min-w-0">
                      <p className="font-semibold text-foreground truncate">{u.name}</p>
                      <p className="text-[11px] text-muted-foreground truncate">{u.email}</p>
                    </div>
                  </label>
                )
              })}
            </div>
          )}
        </div>

        {/* Section 4: Internal Notes */}
        <div className="rounded-lg border border-border bg-card p-5 space-y-2 shadow-xs">
          <label className="text-sm font-semibold text-foreground block">
            Preparation & Interview Notes{' '}
            <span className="text-xs font-normal text-muted-foreground">(optional)</span>
          </label>
          <textarea
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            rows={3}
            placeholder="Add internal agenda, questions to explore, or focus areas for the interviewers..."
            className="w-full px-3 py-2 border border-border rounded-md bg-background text-foreground focus:outline-none focus:ring-1 focus:ring-primary text-xs resize-none"
            maxLength={2000}
          />
        </div>

        {/* Section 5: Email Notifications */}
        <div className="rounded-lg border border-border bg-card p-5 space-y-3 shadow-xs">
          <div className="flex items-center gap-2">
            <Mail className="w-4 h-4 text-muted-foreground" />
            <h3 className="text-sm font-semibold text-foreground">Email Notifications</h3>
          </div>
          <p className="text-xs text-muted-foreground">
            Configure automated interview invitations to be sent upon scheduling.
          </p>
          <div className="space-y-2 pt-1">
            <label className="flex items-center gap-2.5 text-xs text-foreground cursor-pointer select-none">
              <input
                type="checkbox"
                checked={sendCandidateInvitation}
                onChange={(e) => setSendCandidateInvitation(e.target.checked)}
                className="accent-primary rounded"
              />
              <span>Send interview invitation email to candidate</span>
            </label>
            <label className="flex items-center gap-2.5 text-xs text-foreground cursor-pointer select-none">
              <input
                type="checkbox"
                checked={sendInterviewerInvitation}
                onChange={(e) => setSendInterviewerInvitation(e.target.checked)}
                className="accent-primary rounded"
              />
              <span>Send notification email to assigned interviewer(s)</span>
            </label>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center justify-end gap-3 pt-2">
          <Link
            to="/interviews"
            className="px-4 py-2 text-xs font-semibold rounded-md border border-border bg-background text-foreground hover:bg-muted transition-colors"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={createMutation.isPending}
            className="px-5 py-2 text-xs font-semibold rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors shadow-2xs disabled:opacity-50 cursor-pointer"
          >
            {createMutation.isPending ? 'Scheduling Interview…' : 'Schedule Interview'}
          </button>
        </div>
      </form>
    </div>
  )
}

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, MapPin, Mail, Phone, Briefcase, BookOpen, Star, Edit, Trash2, Calendar, ChevronRight, Download } from 'lucide-react'
import { useState } from 'react'
import { reviewsApi } from '@/api/reviews'
import { jobsApi } from '@/api/jobs'
import { interviewsApi } from '@/api/interviews'
import {
  WorkflowStatusBadge,
  ScreeningStatusBadge,
  MatchStatusBadge,
  InterviewStatusBadge,
  InterviewStageBadge,
  InterviewResultBadge,
} from '@/components/data-display/StatusBadge'
import { LoadingState, ErrorState, EmptyState } from '@/components/feedback'
import { formatDate, formatRelativeDate, formatTimeRange } from '@/lib/utils'
import { getApiError } from '@/api/client'
import { toast } from 'sonner'
import { CandidateDocumentList } from '@/features/candidates/components/CandidateDocumentList'
import type { WorkflowStatus, CandidateNote, ScreeningMatch, CandidateDocument } from '@/types'

export function CandidateReviewPage() {
  const { id: jobId, candidateId } = useParams<{ id: string; candidateId: string }>()
  const queryClient = useQueryClient()

  // Modals state
  const [isStatusDialogOpen, setIsStatusDialogOpen] = useState(false)
  const [statusToChange, setStatusToChange] = useState<WorkflowStatus>('REVIEW')
  const [isNoteDialogOpen, setIsNoteDialogOpen] = useState(false)
  const [editingNote, setEditingNote] = useState<CandidateNote | null>(null)
  const [noteContent, setNoteContent] = useState('')
  const [downloadingIcsId, setDownloadingIcsId] = useState<string | null>(null)

  const handleDownloadICS = async (e: React.MouseEvent, interviewId: string) => {
    e.preventDefault()
    e.stopPropagation()
    setDownloadingIcsId(interviewId)
    try {
      await interviewsApi.downloadIcs(interviewId)
      toast.success('Calendar file (.ics) downloaded.')
    } catch {
      toast.error('Unable to generate calendar file.')
    } finally {
      setDownloadingIcsId(null)
    }
  }

  // Queries
  const { data: job } = useQuery({
    queryKey: ['job', jobId],
    queryFn: () => jobsApi.get(jobId!),
    enabled: !!jobId,
  })

  const { data: review, isLoading: reviewLoading, isError: reviewError, error: rError } = useQuery({
    queryKey: ['candidateReview', jobId, candidateId],
    queryFn: () => reviewsApi.getReviewDetail(jobId!, candidateId!),
    enabled: !!jobId && !!candidateId,
  })

  const { data: notesData, refetch: refetchNotes } = useQuery({
    queryKey: ['candidateNotes', jobId, candidateId],
    queryFn: () => reviewsApi.listNotes(jobId!, candidateId!, { limit: 50 }),
    enabled: !!jobId && !!candidateId,
  })

  const { data: interviewsData } = useQuery({
    queryKey: ['candidate-interviews', candidateId, jobId],
    queryFn: () => interviewsApi.getCandidateInterviews(candidateId!, jobId),
    enabled: !!candidateId,
  })

  // Mutations
  const updateStatus = useMutation({
    mutationFn: (status: WorkflowStatus) => reviewsApi.updateStatus(jobId!, candidateId!, status),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['candidateReview', jobId, candidateId] })
      queryClient.invalidateQueries({ queryKey: ['jobCandidates', jobId] })
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      queryClient.invalidateQueries({ queryKey: ['analytics-overview'] })
      setIsStatusDialogOpen(false)
      toast.success('Status updated successfully')
    },
    onError: (err) => toast.error(getApiError(err))
  })

  const createNote = useMutation({
    mutationFn: (content: string) => reviewsApi.createNote(jobId!, candidateId!, content),
    onSuccess: () => {
      refetchNotes()
      setIsNoteDialogOpen(false)
      setNoteContent('')
      toast.success('Note added')
    },
    onError: (err) => toast.error(getApiError(err))
  })

  const updateNote = useMutation({
    mutationFn: (args: { noteId: string; content: string }) => reviewsApi.updateNote(jobId!, candidateId!, args.noteId, args.content),
    onSuccess: () => {
      refetchNotes()
      setIsNoteDialogOpen(false)
      setEditingNote(null)
      setNoteContent('')
      toast.success('Note updated')
    },
    onError: (err) => toast.error(getApiError(err))
  })

  const deleteNote = useMutation({
    mutationFn: (noteId: string) => reviewsApi.deleteNote(jobId!, candidateId!, noteId),
    onSuccess: () => {
      refetchNotes()
      toast.success('Note deleted')
    },
    onError: (err) => toast.error(getApiError(err))
  })

  if (reviewLoading) return <LoadingState message="Loading review details…" />
  if (reviewError || !review) return <ErrorState message={getApiError(rError)} />

  const c = review.candidate
  const candidateName = c?.full_name?.trim() || c?.name?.trim() || 'Candidate'
  const currentStatus = review.workflow?.status || 'REVIEW'
  const experiences = review.experience || []
  const educations = review.education || []
  const skills = review.skills || []
  const documents = review.documents || []

  const screeningResult = review.screening?.screening_result
  const screeningMatches = review.screening?.matches || []
  const screeningStatus = screeningResult?.status
  const screenedDate = screeningResult?.evaluated_at

  const notesList = notesData?.items || review.notes || []

  const handleOpenNoteDialog = (note?: CandidateNote) => {
    if (note) {
      setEditingNote(note)
      setNoteContent(note.content)
    } else {
      setEditingNote(null)
      setNoteContent('')
    }
    setIsNoteDialogOpen(true)
  }

  const handleSaveNote = () => {
    const trimmed = noteContent.trim()
    if (!trimmed) {
      toast.error('Note cannot be empty')
      return
    }
    if (trimmed.length > 5000) {
      toast.error('Note cannot exceed 5000 characters')
      return
    }

    if (editingNote) {
      updateNote.mutate({ noteId: editingNote.id, content: trimmed })
    } else {
      createNote.mutate(trimmed)
    }
  }

  const handleDeleteNote = (id: string) => {
    if (confirm('Are you sure you want to delete this note?')) {
      deleteNote.mutate(id)
    }
  }

  return (
    <div className="max-w-7xl mx-auto space-y-6">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <Link
            to={`/jobs/${jobId}/candidates`}
            className="mt-1 text-slate-400 hover:text-primary transition-colors cursor-pointer focus:outline-none rounded"
            aria-label="Back to candidates"
          >
            <ArrowLeft className="w-5 h-5" />
          </Link>
          <div>
            <div className="flex items-center gap-3 mb-1">
              <h1 className="text-2xl font-mono font-bold text-primary">{candidateName}</h1>
              <WorkflowStatusBadge status={currentStatus} />
            </div>
            <div className="flex flex-wrap items-center gap-3 text-sm text-slate-500">
              <span className="font-semibold">{job?.title}</span>
              {c?.email && (
                <span className="flex items-center gap-1">
                  <Mail className="w-3.5 h-3.5" /> {c.email}
                </span>
              )}
              {c?.phone && (
                <span className="flex items-center gap-1">
                  <Phone className="w-3.5 h-3.5" /> {c.phone}
                </span>
              )}
              {c?.location && (
                <span className="flex items-center gap-1">
                  <MapPin className="w-3.5 h-3.5" /> {c.location}
                </span>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <button
            onClick={() => {
              setStatusToChange(currentStatus)
              setIsStatusDialogOpen(true)
            }}
            className="px-4 py-2 text-sm font-semibold bg-primary text-white rounded-lg hover:bg-primary-700 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
          >
            Change Status
          </button>
        </div>
      </div>

      {/* Main Content Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 items-start">
        {/* Left / Main Column */}
        <div className="lg:col-span-2 space-y-6">
          
          {/* Summary */}
          {c?.summary && (
            <div className="bg-white rounded-card shadow-sm border border-slate-200 p-6">
              <h2 className="text-sm font-bold text-slate-800 mb-3 uppercase tracking-wide">Candidate Summary</h2>
              <p className="text-sm text-slate-600 leading-relaxed">{c.summary}</p>
            </div>
          )}

          {/* Experience */}
          <div className="bg-white rounded-card shadow-sm border border-slate-200 p-6">
            <div className="flex items-center gap-2 mb-4 pb-3 border-b border-slate-100">
              <Briefcase className="w-4 h-4 text-primary" />
              <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide">Experience</h2>
            </div>
            {!experiences.length ? (
              <EmptyState title="No experience recorded" />
            ) : (
              <div className="space-y-4">
                {experiences.map((exp) => (
                  <div key={exp.id} className="relative pl-4 border-l-2 border-primary/20">
                    <div className="flex justify-between items-start">
                      <div>
                        <h3 className="text-sm font-semibold text-slate-800">{exp.title}</h3>
                        <p className="text-sm text-slate-600">{exp.company}</p>
                      </div>
                      <div className="text-xs text-slate-400 text-right">
                        {formatDate(exp.start_date)} — {exp.is_current ? 'Present' : formatDate(exp.end_date)}
                      </div>
                    </div>
                    {exp.description && (
                      <p className="text-xs text-slate-500 mt-2 leading-relaxed">{exp.description}</p>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Education */}
          <div className="bg-white rounded-card shadow-sm border border-slate-200 p-6">
            <div className="flex items-center gap-2 mb-4 pb-3 border-b border-slate-100">
              <BookOpen className="w-4 h-4 text-primary" />
              <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide">Education</h2>
            </div>
            {!educations.length ? (
              <EmptyState title="No education recorded" />
            ) : (
              <div className="space-y-4">
                {educations.map((edu) => (
                  <div key={edu.id} className="relative pl-4 border-l-2 border-primary/20">
                    <div className="flex justify-between items-start">
                      <div>
                        <h3 className="text-sm font-semibold text-slate-800">{edu.degree} in {edu.field_of_study}</h3>
                        <p className="text-sm text-slate-600">{edu.institution}</p>
                      </div>
                      <div className="text-xs text-slate-400">
                        {formatDate(edu.start_date)} — {edu.is_current ? 'Present' : formatDate(edu.end_date)}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Skills */}
          <div className="bg-white rounded-card shadow-sm border border-slate-200 p-6">
            <div className="flex items-center gap-2 mb-4 pb-3 border-b border-slate-100">
              <Star className="w-4 h-4 text-primary" />
              <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide">Skills</h2>
            </div>
            {!skills.length ? (
              <EmptyState title="No skills recorded" />
            ) : (
              <div className="flex flex-wrap gap-1.5">
                {skills.map((skill) => (
                  <span key={skill.id} className="inline-flex items-center px-2 py-1 rounded bg-slate-100 text-xs font-medium text-slate-700 border border-slate-200">
                    {skill.name}
                    {skill.years_of_experience && <span className="ml-1 opacity-60">· {skill.years_of_experience}y</span>}
                  </span>
                ))}
              </div>
            )}
          </div>

          {/* Documents */}
          <div className="bg-white rounded-card shadow-sm border border-slate-200 p-6">
            <CandidateDocumentList
              candidateId={candidateId!}
              initialDocuments={documents as CandidateDocument[]}
              onDocumentProcessed={() => {
                queryClient.invalidateQueries({ queryKey: ['candidateReview', jobId, candidateId] })
                queryClient.invalidateQueries({ queryKey: ['candidate', candidateId] })
              }}
              onDocumentDeleted={() => {
                queryClient.invalidateQueries({ queryKey: ['candidateReview', jobId, candidateId] })
                queryClient.invalidateQueries({ queryKey: ['candidate', candidateId] })
              }}
            />
          </div>
        </div>

        {/* Right / Supporting Column */}
        <div className="space-y-6">
          
          {/* Screening Results */}
          <div className="bg-white rounded-card shadow-sm border border-slate-200 overflow-hidden">
            <div className="bg-slate-50 px-5 py-4 border-b border-slate-200">
              <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide mb-2">Screening Result</h2>
              {screeningStatus ? (
                <div className="flex items-center gap-2">
                  <ScreeningStatusBadge status={screeningStatus} />
                  {screenedDate && (
                    <span className="text-xs text-slate-500">
                      {formatRelativeDate(screenedDate)}
                    </span>
                  )}
                </div>
              ) : (
                <span className="text-xs text-slate-500">Not screened</span>
              )}
            </div>

            {screeningMatches.length > 0 && (
              <div className="p-5 space-y-4">
                {screeningResult?.summary && (
                  <p className="text-sm text-slate-600 leading-relaxed border-b border-slate-100 pb-4">
                    {screeningResult.summary}
                  </p>
                )}

                <div className="space-y-3">
                  <h3 className="text-xs font-semibold text-slate-700 uppercase">Requirement Matches</h3>
                  <div className="space-y-3">
                    {screeningMatches.map((m: ScreeningMatch, idx: number) => {
                      const matchRequirement = m.requirement || m.requirement_description || 'Requirement'
                      const matchStatus = m.status || m.match_status || 'UNKNOWN'
                      const matchEvidence = m.evidence || m.explanation || m.reason || '—'

                      return (
                        <div key={m.requirement_id || m.id || idx} className="text-sm border border-slate-100 rounded-lg p-3 bg-slate-50/50">
                          <div className="flex justify-between items-start mb-2">
                            <span className="font-medium text-slate-800 text-xs">{matchRequirement}</span>
                            <MatchStatusBadge status={matchStatus} />
                          </div>
                          <div className="text-xs text-slate-600 mt-1">
                            <span className="font-semibold text-slate-500">Evidence:</span> {matchEvidence}
                          </div>
                        </div>
                      )
                    })}
                  </div>
                </div>
              </div>
            )}
          </div>

          {/* Notes */}
          <div className="bg-white rounded-card shadow-sm border border-slate-200 flex flex-col max-h-[600px]">
            <div className="px-5 py-4 border-b border-slate-200 flex justify-between items-center">
              <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide">Recruiter Notes</h2>
              <button
                onClick={() => handleOpenNoteDialog()}
                className="text-xs font-semibold text-primary hover:underline focus:outline-none cursor-pointer"
              >
                + Add
              </button>
            </div>
            
            <div className="p-5 overflow-y-auto flex-1 space-y-4">
              {!notesList.length ? (
                <EmptyState title="No notes" description="Add private notes about this candidate." />
              ) : (
                notesList.map((note) => {
                  const authorName = note.author?.name || 'Recruiter'
                  const authorInitial = (authorName.charAt(0) || 'R').toUpperCase()

                  return (
                    <div key={note.id} className="bg-slate-50 border border-slate-200 rounded-lg p-3 group">
                      <div className="flex justify-between items-start mb-1.5">
                        <div className="flex items-center gap-1.5">
                          <div className="w-5 h-5 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                            <span className="text-[10px] font-bold text-primary">{authorInitial}</span>
                          </div>
                          <span className="text-xs font-semibold text-slate-700">{authorName}</span>
                        </div>
                        <div className="flex items-center gap-2">
                          <span className="text-[10px] text-slate-400">{formatRelativeDate(note.created_at)}</span>
                          <div className="opacity-0 group-hover:opacity-100 transition-opacity flex items-center gap-1">
                            <button onClick={() => handleOpenNoteDialog(note)} className="text-slate-400 hover:text-primary cursor-pointer focus:outline-none">
                              <Edit className="w-3 h-3" />
                            </button>
                            <button onClick={() => handleDeleteNote(note.id)} className="text-slate-400 hover:text-red-600 cursor-pointer focus:outline-none">
                              <Trash2 className="w-3 h-3" />
                            </button>
                          </div>
                        </div>
                      </div>
                      <p className="text-sm text-slate-600 whitespace-pre-wrap">{note.content}</p>
                    </div>
                  )
                })
              )}
            </div>
          </div>

          {/* Interview History (Step 12A) */}
          <div className="bg-white rounded-card shadow-sm border border-slate-200 flex flex-col">
            <div className="px-5 py-4 border-b border-slate-200 flex justify-between items-center">
              <div className="flex items-center gap-2">
                <Calendar className="w-4 h-4 text-primary" />
                <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide">
                  Interview History
                </h2>
              </div>
              <Link
                to={`/interviews/new?job_id=${jobId}&candidate_id=${candidateId}`}
                className="text-xs font-semibold text-primary hover:underline focus:outline-none cursor-pointer"
              >
                + Schedule
              </Link>
            </div>

            <div className="p-4 space-y-3">
              {!interviewsData || interviewsData.length === 0 ? (
                <div className="py-6 text-center text-xs text-slate-400">
                  <p>No interviews scheduled yet.</p>
                  <Link
                    to={`/interviews/new?job_id=${jobId}&candidate_id=${candidateId}`}
                    className="mt-2 inline-block text-primary font-semibold hover:underline"
                  >
                    Schedule an interview
                  </Link>
                </div>
              ) : (
                interviewsData.map((iv) => (
                  <div
                    key={iv.id}
                    className="p-3 bg-slate-50 border border-slate-200 rounded-lg hover:border-slate-300 transition-colors"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div>
                        <div className="flex items-center gap-1.5 flex-wrap">
                          <span className="font-semibold text-xs text-slate-800">{iv.title}</span>
                          <InterviewStageBadge stage={iv.stage} />
                          <InterviewStatusBadge status={iv.status} />
                          {iv.result && <InterviewResultBadge result={iv.result} />}
                        </div>
                        <p className="text-[11px] font-mono text-slate-500 mt-1">
                          {formatTimeRange(iv.scheduled_start, iv.scheduled_end)}
                        </p>
                        {iv.interviewers?.length > 0 && (
                          <p className="text-[11px] text-slate-400 mt-0.5">
                            Interviewers: {iv.interviewers.map((i) => i.name).join(', ')}
                          </p>
                        )}
                      </div>
                      <div className="flex items-center gap-1.5 shrink-0 mt-0.5">
                        <button
                          type="button"
                          onClick={(e) => handleDownloadICS(e, iv.id)}
                          disabled={downloadingIcsId === iv.id}
                          className="p-1 text-slate-400 hover:text-slate-700 hover:bg-slate-200/60 rounded transition-colors cursor-pointer disabled:opacity-40"
                          title="Add to Calendar (.ics)"
                        >
                          <Download className="w-3.5 h-3.5" />
                        </button>
                        <Link
                          to={`/interviews/${iv.id}`}
                          className="text-xs font-semibold text-primary hover:underline flex items-center"
                        >
                          View
                          <ChevronRight className="w-3 h-3 ml-0.5" />
                        </Link>
                      </div>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>

        </div>
      </div>

      {/* Status Dialog */}
      {isStatusDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-sm overflow-hidden flex flex-col">
            <div className="px-5 py-4 border-b border-slate-100">
              <h3 className="text-base font-bold text-slate-800">Change candidate status</h3>
            </div>
            <div className="p-5 space-y-4">
              <div className="space-y-1">
                <p className="text-xs text-slate-500">Current status</p>
                <div><WorkflowStatusBadge status={currentStatus} /></div>
              </div>
              <div className="space-y-1">
                <label className="text-xs text-slate-500" htmlFor="new_status">New status</label>
                <select
                  id="new_status"
                  value={statusToChange}
                  onChange={(e) => setStatusToChange(e.target.value as WorkflowStatus)}
                  className="w-full input-field bg-white cursor-pointer"
                >
                  <option value="REVIEW">Review</option>
                  <option value="SHORTLISTED">Shortlisted</option>
                  <option value="REJECTED">Rejected</option>
                </select>
              </div>
            </div>
            <div className="px-5 py-4 bg-slate-50 flex justify-end gap-2 border-t border-slate-100">
              <button
                onClick={() => setIsStatusDialogOpen(false)}
                disabled={updateStatus.isPending}
                className="px-4 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-200 bg-slate-100 rounded-lg transition-colors cursor-pointer focus:outline-none"
              >
                Cancel
              </button>
              <button
                onClick={() => updateStatus.mutate(statusToChange)}
                disabled={updateStatus.isPending || statusToChange === currentStatus}
                className="px-4 py-2 text-sm font-semibold text-white bg-primary hover:bg-primary-700 rounded-lg transition-colors cursor-pointer focus:outline-none disabled:opacity-50"
              >
                {updateStatus.isPending ? 'Saving…' : 'Confirm'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Note Dialog */}
      {isNoteDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-md overflow-hidden flex flex-col">
            <div className="px-5 py-4 border-b border-slate-100">
              <h3 className="text-base font-bold text-slate-800">{editingNote ? 'Edit Note' : 'Add Note'}</h3>
            </div>
            <div className="p-5">
              <label className="block text-xs font-semibold text-slate-600 mb-1.5" htmlFor="note_content">
                Note Content
              </label>
              <textarea
                id="note_content"
                rows={5}
                value={noteContent}
                onChange={(e) => setNoteContent(e.target.value)}
                placeholder="Type your note here..."
                className="input-field resize-y"
                autoFocus
              />
              <p className="text-xs text-slate-400 mt-1 flex justify-between">
                <span>Markdown is not supported.</span>
                <span className={noteContent.length > 5000 ? 'text-red-500' : ''}>
                  {noteContent.length} / 5000
                </span>
              </p>
            </div>
            <div className="px-5 py-4 bg-slate-50 flex justify-end gap-2 border-t border-slate-100">
              <button
                onClick={() => setIsNoteDialogOpen(false)}
                disabled={createNote.isPending || updateNote.isPending}
                className="px-4 py-2 text-sm font-semibold text-slate-600 hover:bg-slate-200 bg-slate-100 rounded-lg transition-colors cursor-pointer focus:outline-none"
              >
                Cancel
              </button>
              <button
                onClick={handleSaveNote}
                disabled={createNote.isPending || updateNote.isPending}
                className="px-4 py-2 text-sm font-semibold text-white bg-primary hover:bg-primary-700 rounded-lg transition-colors cursor-pointer focus:outline-none disabled:opacity-50"
              >
                {createNote.isPending || updateNote.isPending ? 'Saving…' : 'Save Note'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Edit, ChevronDown, ChevronUp, MapPin, Calendar, Briefcase, Building2 } from 'lucide-react'
import { useState } from 'react'
import { jobsApi } from '@/api/jobs'
import { JobStatusBadge } from '@/components/data-display/StatusBadge'
import { LoadingState, EmptyState, ErrorState } from '@/components/feedback'
import { formatDate, formatEmploymentType, cn } from '@/lib/utils'
import { getApiError } from '@/api/client'
import { toast } from 'sonner'
import type { JobStatus, RequirementCategory } from '@/types'

// ─── Status Transitions ────────────────────────────────────────

const NEXT_STATUSES: Record<JobStatus, JobStatus[]> = {
  DRAFT: ['OPEN'],
  OPEN: ['CLOSED'],
  CLOSED: ['OPEN', 'ARCHIVED'],
  ARCHIVED: [],
}

const STATUS_LABELS: Record<JobStatus, string> = {
  DRAFT: 'Publish → Open',
  OPEN: 'Close Hiring',
  CLOSED: 'Reopen',
  ARCHIVED: '',
}

// ─── Requirement Category Color ────────────────────────────────

const CATEGORY_COLORS: Record<RequirementCategory, string> = {
  SKILL: 'bg-blue-50 text-blue-700 border-blue-200',
  EXPERIENCE: 'bg-purple-50 text-purple-700 border-purple-200',
  EDUCATION: 'bg-green-50 text-green-700 border-green-200',
  CERTIFICATION: 'bg-amber-50 text-amber-700 border-amber-200',
  LANGUAGE: 'bg-pink-50 text-pink-700 border-pink-200',
  OTHER: 'bg-slate-50 text-slate-600 border-slate-200',
}

// ─── Job Detail Page ───────────────────────────────────────────

export function JobDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [showDescription, setShowDescription] = useState(true)

  const { data: job, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['job', id],
    queryFn: () => jobsApi.get(id!),
    enabled: !!id,
  })

  const updateStatus = useMutation({
    mutationFn: (status: JobStatus) => jobsApi.updateStatus(id!, { status }),
    onSuccess: (updated) => {
      queryClient.setQueryData(['job', id], updated)
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
      toast.success(`Job status updated to ${updated.status}`)
    },
    onError: (err) => toast.error(getApiError(err)),
  })

  if (isLoading) return <LoadingState message="Loading job…" />
  if (isError) return <ErrorState message={getApiError(error)} onRetry={refetch} />
  if (!job) return null

  const nextStatuses = NEXT_STATUSES[job.status]
  const primaryNext = nextStatuses[0]

  // Group requirements by importance
  const required = job.requirements?.filter((r) => r.importance === 'REQUIRED') ?? []
  const preferred = job.requirements?.filter((r) => r.importance === 'PREFERRED') ?? []

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-start justify-between gap-4">
        <div className="flex items-start gap-3">
          <button
            onClick={() => navigate('/jobs')}
            className="mt-1 text-slate-400 hover:text-primary transition-colors cursor-pointer focus:outline-none rounded"
            aria-label="Back to jobs"
          >
            <ArrowLeft className="w-4 h-4" />
          </button>
          <div>
            <div className="flex items-center gap-2 mb-1">
              <h1 className="text-xl font-mono font-bold text-primary">{job.title}</h1>
              <JobStatusBadge status={job.status} />
            </div>
            <p className="text-xs text-slate-400 font-mono">{job.code}</p>
          </div>
        </div>
        <div className="flex items-center gap-2 flex-shrink-0">
          {primaryNext && (
            <button
              onClick={() => updateStatus.mutate(primaryNext)}
              disabled={updateStatus.isPending}
              className="px-3 py-1.5 text-xs font-semibold bg-cta text-white rounded-lg hover:opacity-90 transition-all cursor-pointer focus:outline-none focus:ring-2 focus:ring-cta focus:ring-offset-1 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {updateStatus.isPending
                ? 'Updating…'
                : primaryNext === 'OPEN'
                ? 'Publish'
                : primaryNext === 'CLOSED'
                ? 'Close'
                : STATUS_LABELS[job.status]}
            </button>
          )}
          {nextStatuses.length > 1 && (
            <button
              onClick={() => updateStatus.mutate(nextStatuses[1])}
              disabled={updateStatus.isPending}
              className="px-3 py-1.5 text-xs font-semibold text-slate-600 border border-slate-200 rounded-lg hover:bg-slate-50 transition-all cursor-pointer focus:outline-none focus:ring-2 focus:ring-slate-300 disabled:opacity-50"
            >
              Archive
            </button>
          )}
          <Link
            to={`/jobs/${job.id}/candidates`}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-slate-700 bg-white border border-slate-200 rounded-lg hover:bg-slate-50 transition-all cursor-pointer focus:outline-none focus:ring-2 focus:ring-slate-200"
          >
            Candidates
          </Link>
          <Link
            to={`/jobs/${job.id}/edit`}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold text-primary border border-primary/30 rounded-lg hover:bg-primary/5 transition-all cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1"
          >
            <Edit className="w-3 h-3" />
            Edit
          </Link>
        </div>
      </div>

      {/* Meta Info */}
      <div className="bg-white rounded-card shadow-md p-6">
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="flex items-center gap-2 text-sm">
            <Building2 className="w-4 h-4 text-slate-400 flex-shrink-0" />
            <div>
              <p className="text-xs text-slate-400">Department</p>
              <p className="font-medium text-slate-700">{job.department || '—'}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <MapPin className="w-4 h-4 text-slate-400 flex-shrink-0" />
            <div>
              <p className="text-xs text-slate-400">Location</p>
              <p className="font-medium text-slate-700">{job.location || '—'}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <Briefcase className="w-4 h-4 text-slate-400 flex-shrink-0" />
            <div>
              <p className="text-xs text-slate-400">Employment Type</p>
              <p className="font-medium text-slate-700">{formatEmploymentType(job.employment_type)}</p>
            </div>
          </div>
          <div className="flex items-center gap-2 text-sm">
            <Calendar className="w-4 h-4 text-slate-400 flex-shrink-0" />
            <div>
              <p className="text-xs text-slate-400">Closing Date</p>
              <p className="font-medium text-slate-700">{formatDate(job.closing_date)}</p>
            </div>
          </div>
        </div>

        <div className="mt-4 pt-4 border-t border-slate-100 grid grid-cols-2 gap-4 text-sm text-slate-500">
          <div>Created: {formatDate(job.created_at)}</div>
          <div>Updated: {formatDate(job.updated_at)}</div>
        </div>
      </div>

      {/* Description */}
      <div className="bg-white rounded-card shadow-md overflow-hidden">
        <button
          type="button"
          onClick={() => setShowDescription((v) => !v)}
          className="w-full flex items-center justify-between px-6 py-4 text-sm font-semibold text-slate-700 hover:bg-slate-50 transition-colors cursor-pointer focus:outline-none"
        >
          Description
          {showDescription ? <ChevronUp className="w-4 h-4 text-slate-400" /> : <ChevronDown className="w-4 h-4 text-slate-400" />}
        </button>
        {showDescription && (
          <div className="px-6 pb-6 border-t border-slate-100">
            <p className="text-sm text-slate-600 leading-relaxed whitespace-pre-wrap mt-4">
              {job.description}
            </p>
          </div>
        )}
      </div>

      {/* Requirements */}
      <div className="bg-white rounded-card shadow-md p-6 space-y-4">
        <h2 className="text-sm font-semibold text-slate-700 pb-2 border-b border-slate-100">
          Requirements ({job.requirements?.length ?? 0})
        </h2>

        {!job.requirements?.length ? (
          <EmptyState
            title="No requirements"
            description="Add requirements to help screen candidates effectively."
            action={
              <Link
                to={`/jobs/${job.id}/edit`}
                className="text-sm text-primary hover:underline cursor-pointer"
              >
                Edit job to add requirements
              </Link>
            }
          />
        ) : (
          <div className="space-y-4">
            {required.length > 0 && (
              <div>
                <h3 className="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">
                  Required ({required.length})
                </h3>
                <div className="space-y-2">
                  {required.map((req) => (
                    <div key={req.id} className="flex items-start gap-3 p-3 rounded-lg border border-slate-100 bg-slate-50/50">
                      <span
                        className={cn(
                          'inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold border flex-shrink-0 mt-0.5',
                          CATEGORY_COLORS[req.category]
                        )}
                      >
                        {req.category}
                      </span>
                      <p className="text-sm text-slate-700 leading-relaxed">{req.description}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}

            {preferred.length > 0 && (
              <div>
                <h3 className="text-xs font-bold text-slate-500 uppercase tracking-wider mb-2">
                  Preferred ({preferred.length})
                </h3>
                <div className="space-y-2">
                  {preferred.map((req) => (
                    <div key={req.id} className="flex items-start gap-3 p-3 rounded-lg border border-slate-100 bg-slate-50/30">
                      <span
                        className={cn(
                          'inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold border flex-shrink-0 mt-0.5 opacity-75',
                          CATEGORY_COLORS[req.category]
                        )}
                      >
                        {req.category}
                      </span>
                      <p className="text-sm text-slate-600 leading-relaxed">{req.description}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}

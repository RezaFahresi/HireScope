import { useQuery } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import { ArrowLeft, Search, Users, Clock } from 'lucide-react'
import { useState } from 'react'
import { reviewsApi } from '@/api/reviews'
import { jobsApi } from '@/api/jobs'
import { interviewsApi } from '@/api/interviews'
import { EmptyState, ErrorState, TableRowSkeleton } from '@/components/feedback'
import { WorkflowStatusBadge, ScreeningStatusBadge } from '@/components/data-display/StatusBadge'
import { formatDate, formatDateTime } from '@/lib/utils'
import { getApiError } from '@/api/client'
import type { WorkflowStatus } from '@/types'

// ─── Pagination ───────────────────────────────────────────────

interface PaginationProps {
  page: number
  totalPages: number
  onPageChange: (page: number) => void
}

function Pagination({ page, totalPages, onPageChange }: PaginationProps) {
  if (totalPages <= 1) return null
  return (
    <div className="flex items-center gap-1 px-4 py-3 border-t border-slate-100">
      <button
        onClick={() => onPageChange(page - 1)}
        disabled={page <= 1}
        className="px-3 py-1.5 text-xs rounded-md border border-slate-200 text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer transition-colors"
      >
        Previous
      </button>
      <span className="px-3 text-xs text-slate-500">
        Page {page} of {totalPages}
      </span>
      <button
        onClick={() => onPageChange(page + 1)}
        disabled={page >= totalPages}
        className="px-3 py-1.5 text-xs rounded-md border border-slate-200 text-slate-600 hover:bg-slate-50 disabled:opacity-40 disabled:cursor-not-allowed cursor-pointer transition-colors"
      >
        Next
      </button>
    </div>
  )
}

function CandidateNextInterviewCell({
  candidateId,
  jobId,
}: {
  candidateId: string
  jobId?: string
}) {
  const { data: interview, isLoading } = useQuery({
    queryKey: ['candidate-next-interview', candidateId, jobId],
    queryFn: () => interviewsApi.getCandidateNextInterview(candidateId, jobId),
    staleTime: 30_000,
  })

  if (isLoading) {
    return <span className="text-slate-300 font-mono text-xs animate-pulse">…</span>
  }

  if (!interview) {
    return <span className="text-slate-400">—</span>
  }

  return (
    <Link
      to={`/interviews/${interview.id}`}
      className="inline-flex items-center gap-1 font-mono text-xs text-primary hover:underline"
      title={`${interview.title} (${interview.stage_label})`}
    >
      <Clock className="w-3 h-3 text-slate-400" />
      {formatDateTime(interview.scheduled_start)}
    </Link>
  )
}

// ─── Job Candidates Page ──────────────────────────────────────

export function JobCandidatesPage() {
  const { id: jobId } = useParams<{ id: string }>()
  
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [activeSearch, setActiveSearch] = useState('')
  const [statusFilter, setStatusFilter] = useState<WorkflowStatus | ''>('')

  // Fetch job details to display title
  const { data: job } = useQuery({
    queryKey: ['job', jobId],
    queryFn: () => jobsApi.get(jobId!),
    enabled: !!jobId,
  })

  // Fetch candidates for this job
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['jobCandidates', jobId, { page, search: activeSearch, status: statusFilter }],
    queryFn: () => reviewsApi.listCandidatesByJob(jobId!, { 
      page, 
      limit: 20, 
      ...(activeSearch ? { search: activeSearch } : {}),
      ...(statusFilter ? { status: statusFilter } : {})
    }),
    enabled: !!jobId,
  })

  const handleSearch = () => {
    setActiveSearch(searchInput)
    setPage(1)
  }

  const clearSearch = () => {
    setSearchInput('')
    setActiveSearch('')
    setPage(1)
  }

  const SKELETON_COLS = 7

  return (
    <div className="max-w-dashboard mx-auto space-y-4">
      {/* Header */}
      <div className="flex items-start gap-3">
        <Link
          to={`/jobs/${jobId}`}
          className="mt-1 text-slate-400 hover:text-primary transition-colors cursor-pointer focus:outline-none rounded"
          aria-label="Back to job"
        >
          <ArrowLeft className="w-4 h-4" />
        </Link>
        <div>
          <h1 className="text-xl font-mono font-bold text-primary">Job Candidates</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            {job?.title ? `${job.title}` : 'Loading job...'}
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg border border-slate-200 px-4 py-3 flex flex-wrap gap-3 items-center">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400 pointer-events-none" />
          <input
            type="search"
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
            placeholder="Search candidates…"
            className="w-full pl-8 pr-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 transition-all"
          />
        </div>
        
        <select 
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value as WorkflowStatus | '')
            setPage(1)
          }}
          className="px-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 bg-white cursor-pointer"
        >
          <option value="">All Statuses</option>
          <option value="REVIEW">Review</option>
          <option value="SHORTLISTED">Shortlisted</option>
          <option value="REJECTED">Rejected</option>
        </select>

        <button
          onClick={handleSearch}
          className="px-3 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary-700 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1"
        >
          Search
        </button>
        {(activeSearch || statusFilter) && (
          <button
            onClick={() => {
              clearSearch()
              setStatusFilter('')
            }}
            className="text-sm text-slate-500 hover:text-red-600 transition-colors cursor-pointer"
          >
            Clear filters
          </button>
        )}
      </div>

      {/* Table */}
      <div className="bg-white rounded-card shadow-md overflow-hidden">
        {isError ? (
          <ErrorState message={getApiError(error)} onRetry={refetch} />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-slate-50">
                  <th className="table-header text-left">Candidate</th>
                  <th className="table-header text-left">Workflow Status</th>
                  <th className="table-header text-left hidden md:table-cell">Screening</th>
                  <th className="table-header text-left hidden lg:table-cell">Next Interview</th>
                  <th className="table-header text-left hidden lg:table-cell">Location</th>
                  <th className="table-header text-left hidden md:table-cell">Added</th>
                  <th className="table-header text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  Array.from({ length: 8 }).map((_, i) => <TableRowSkeleton key={i} cols={SKELETON_COLS} rowIndex={i} />)
                ) : !data?.items?.length ? (
                  <tr>
                    <td colSpan={SKELETON_COLS}>
                      <EmptyState
                        icon={Users}
                        title={(activeSearch || statusFilter) ? 'No candidates match filters' : 'No candidates assigned'}
                        description={
                          (activeSearch || statusFilter)
                            ? 'Try a different search term or status.'
                            : 'No candidates have been assigned to this job yet.'
                        }
                        action={
                          (activeSearch || statusFilter) ? (
                            <button
                              onClick={() => {
                                clearSearch()
                                setStatusFilter('')
                              }}
                              className="text-sm text-primary hover:underline cursor-pointer"
                            >
                              Clear filters
                            </button>
                          ) : undefined
                        }
                      />
                    </td>
                  </tr>
                ) : (
                  data.items.map((jc) => {
                    const candidateName = jc.candidate?.full_name || jc.candidate?.name || jc.full_name || 'Unknown Candidate'
                    const candidateEmail = jc.candidate?.email || jc.email
                    const candidateLocation = jc.candidate?.location || jc.location || '—'
                    const candidateId = jc.candidate?.id || jc.candidate_id || jc.id
                    const status = jc.status || jc.workflow_status || 'REVIEW'
                    const screeningStatus = jc.screening?.status || jc.screening_result?.status || jc.screening_result?.screening_status

                    return (
                      <tr key={jc.id} className="table-row hover:bg-slate-50/50">
                        <td className="table-cell">
                          <div className="font-medium text-slate-800">
                            {candidateName}
                          </div>
                          {candidateEmail && (
                            <div className="text-xs text-slate-500 truncate max-w-[200px]">
                              {candidateEmail}
                            </div>
                          )}
                        </td>
                        <td className="table-cell">
                          <WorkflowStatusBadge status={status} />
                        </td>
                        <td className="table-cell hidden md:table-cell">
                          {screeningStatus ? (
                            <ScreeningStatusBadge status={screeningStatus} />
                          ) : (
                            <span className="text-xs text-slate-400 italic">Not screened</span>
                          )}
                        </td>
                        <td className="table-cell hidden lg:table-cell">
                          <CandidateNextInterviewCell candidateId={candidateId} jobId={jobId} />
                        </td>
                        <td className="table-cell hidden lg:table-cell text-slate-600">
                          {candidateLocation}
                        </td>
                        <td className="table-cell hidden md:table-cell text-slate-500">
                          {formatDate(jc.assigned_at || jc.reviewed_at)}
                        </td>
                        <td className="table-cell text-right">
                          <Link
                            to={`/jobs/${jobId}/candidates/${candidateId}/review`}
                            className="text-xs font-semibold text-primary hover:underline cursor-pointer"
                          >
                            Review
                          </Link>
                        </td>
                      </tr>
                    )
                  })
                )}
              </tbody>
            </table>
          </div>
        )}

        {data && (
          <Pagination
            page={page}
            totalPages={data.total_pages}
            onPageChange={setPage}
          />
        )}
      </div>
    </div>
  )
}

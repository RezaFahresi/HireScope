import { useState, useCallback } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link, useSearchParams } from 'react-router-dom'
import { Plus, Search, X, Briefcase } from 'lucide-react'
import { jobsApi } from '@/api/jobs'
import { JobStatusBadge } from '@/components/data-display/StatusBadge'
import { EmptyState, ErrorState, TableRowSkeleton } from '@/components/feedback'
import { formatDate, formatEmploymentType, cn } from '@/lib/utils'
import type { JobStatus, EmploymentType } from '@/types'

// ─── Filter Bar ───────────────────────────────────────────────

const STATUS_OPTIONS: { value: JobStatus; label: string }[] = [
  { value: 'OPEN', label: 'Open' },
  { value: 'DRAFT', label: 'Draft' },
  { value: 'CLOSED', label: 'Closed' },
  { value: 'ARCHIVED', label: 'Archived' },
]

const TYPE_OPTIONS: { value: EmploymentType; label: string }[] = [
  { value: 'FULL_TIME', label: 'Full Time' },
  { value: 'PART_TIME', label: 'Part Time' },
  { value: 'CONTRACT', label: 'Contract' },
  { value: 'INTERNSHIP', label: 'Internship' },
  { value: 'FREELANCE', label: 'Freelance' },
]

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

// ─── Jobs List Page ───────────────────────────────────────────

export function JobsListPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [searchInput, setSearchInput] = useState(searchParams.get('search') ?? '')

  const page = parseInt(searchParams.get('page') ?? '1', 10)
  const status = (searchParams.get('status') ?? '') as JobStatus | ''
  const employmentType = (searchParams.get('employment_type') ?? '') as EmploymentType | ''
  const search = searchParams.get('search') ?? ''

  const updateParam = useCallback(
    (key: string, value: string) => {
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
    },
    [setSearchParams]
  )

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['jobs', { page, status, employment_type: employmentType, search }],
    queryFn: () =>
      jobsApi.list({
        page,
        limit: 20,
        ...(status ? { status } : {}),
        ...(employmentType ? { employment_type: employmentType } : {}),
        ...(search ? { search } : {}),
      }),
  })

  const handleSearch = () => {
    updateParam('search', searchInput)
  }

  const clearFilters = () => {
    setSearchInput('')
    setSearchParams({})
  }

  const hasFilters = !!(status || employmentType || search)

  return (
    <div className="max-w-dashboard mx-auto space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-mono font-bold text-primary">Jobs</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            {data ? `${data.total} position${data.total !== 1 ? 's' : ''}` : 'Manage job postings'}
          </p>
        </div>
        <Link
          to="/jobs/new"
          className="flex items-center gap-2 bg-primary text-white px-4 py-2 rounded-lg text-sm font-semibold hover:bg-primary-700 transition-colors duration-150 cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
        >
          <Plus className="w-4 h-4" />
          New Job
        </Link>
      </div>

      {/* Filters */}
      <div className="bg-white rounded-lg border border-slate-200 px-4 py-3 flex flex-wrap gap-3 items-center">
        {/* Search */}
        <div className="flex-1 min-w-48 flex gap-2">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-slate-400 pointer-events-none" />
            <input
              type="search"
              value={searchInput}
              onChange={(e) => setSearchInput(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              placeholder="Search jobs…"
              className="w-full pl-8 pr-3 py-2 text-sm border border-slate-200 rounded-lg focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 transition-all"
            />
          </div>
          <button
            onClick={handleSearch}
            className="px-3 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary-700 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1"
          >
            Search
          </button>
        </div>

        {/* Status Filter */}
        <select
          value={status}
          onChange={(e) => updateParam('status', e.target.value)}
          className="text-sm border border-slate-200 rounded-lg px-3 py-2 text-slate-700 focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 cursor-pointer transition-all bg-white"
        >
          <option value="">All statuses</option>
          {STATUS_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>

        {/* Type Filter */}
        <select
          value={employmentType}
          onChange={(e) => updateParam('employment_type', e.target.value)}
          className="text-sm border border-slate-200 rounded-lg px-3 py-2 text-slate-700 focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 cursor-pointer transition-all bg-white"
        >
          <option value="">All types</option>
          {TYPE_OPTIONS.map((opt) => (
            <option key={opt.value} value={opt.value}>{opt.label}</option>
          ))}
        </select>

        {hasFilters && (
          <button
            onClick={clearFilters}
            className="flex items-center gap-1 text-sm text-slate-500 hover:text-red-600 transition-colors cursor-pointer px-2 py-1 rounded"
          >
            <X className="w-3.5 h-3.5" />
            Clear
          </button>
        )}
      </div>

      {/* Table */}
      <div className="bg-white rounded-card shadow-md overflow-hidden">
        {isError ? (
          <ErrorState message="Failed to load jobs." onRetry={refetch} />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-slate-50">
                  <th className="table-header text-left">Job</th>
                  <th className="table-header text-left hidden md:table-cell">Department</th>
                  <th className="table-header text-left hidden lg:table-cell">Location</th>
                  <th className="table-header text-left hidden md:table-cell">Type</th>
                  <th className="table-header text-left">Status</th>
                  <th className="table-header text-left hidden lg:table-cell">Closing Date</th>
                  <th className="table-header text-left hidden lg:table-cell">Created</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  Array.from({ length: 8 }).map((_, i) => <TableRowSkeleton key={i} cols={7} rowIndex={i} />)
                ) : !data?.items?.length ? (
                  <tr>
                    <td colSpan={7}>
                      <EmptyState
                        icon={Briefcase}
                        title={hasFilters ? 'No jobs match your filters' : 'No jobs yet'}
                        description={
                          hasFilters
                            ? 'Try adjusting your filters or clearing them.'
                            : 'Create your first job posting to get started.'
                        }
                        action={
                          hasFilters ? (
                            <button
                              onClick={clearFilters}
                              className="text-sm text-primary hover:underline cursor-pointer"
                            >
                              Clear filters
                            </button>
                          ) : (
                            <Link
                              to="/jobs/new"
                              className="text-sm text-primary hover:underline cursor-pointer"
                            >
                              Create a job
                            </Link>
                          )
                        }
                      />
                    </td>
                  </tr>
                ) : (
                  data.items.map((job) => (
                    <tr key={job.id} className={cn('table-row')}>
                      <td className="table-cell">
                        <Link
                          to={`/jobs/${job.id}`}
                          className="font-medium text-primary hover:underline cursor-pointer"
                        >
                          {job.title}
                        </Link>
                        <p className="text-xs text-slate-400 font-mono">{job.code}</p>
                      </td>
                      <td className="table-cell hidden md:table-cell text-slate-600">
                        {job.department || '—'}
                      </td>
                      <td className="table-cell hidden lg:table-cell text-slate-600">
                        {job.location || '—'}
                      </td>
                      <td className="table-cell hidden md:table-cell text-slate-600">
                        {formatEmploymentType(job.employment_type)}
                      </td>
                      <td className="table-cell">
                        <JobStatusBadge status={job.status} />
                      </td>
                      <td className="table-cell hidden lg:table-cell text-slate-500">
                        {formatDate(job.closing_date)}
                      </td>
                      <td className="table-cell hidden lg:table-cell text-slate-500">
                        {formatDate(job.created_at)}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        )}

        {data && (
          <Pagination
            page={page}
            totalPages={data.total_pages}
            onPageChange={(p) =>
              setSearchParams((prev) => {
                const next = new URLSearchParams(prev)
                next.set('page', String(p))
                return next
              })
            }
          />
        )}
      </div>
    </div>
  )
}

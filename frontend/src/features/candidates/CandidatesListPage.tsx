import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Search, Users, Mail, Phone, MapPin, UserPlus } from 'lucide-react'
import { useState } from 'react'
import { candidatesApi } from '@/api/candidates'
import { EmptyState, ErrorState, TableRowSkeleton } from '@/components/feedback'
import { formatDate } from '@/lib/utils'

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

// ─── Candidates List Page ─────────────────────────────────────

export function CandidatesListPage() {
  const [page, setPage] = useState(1)
  const [searchInput, setSearchInput] = useState('')
  const [activeSearch, setActiveSearch] = useState('')

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['candidates', { page, search: activeSearch }],
    queryFn: () => candidatesApi.list({ page, limit: 20, ...(activeSearch ? { search: activeSearch } : {}) }),
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

  return (
    <div className="max-w-dashboard mx-auto space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-mono font-bold text-primary">Candidates</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            {data ? `${data.total} candidate${data.total !== 1 ? 's' : ''}` : 'All candidate profiles'}
          </p>
        </div>
        <Link
          to="/candidates/new"
          className="flex items-center gap-2 bg-primary text-white px-4 py-2 rounded-lg text-sm font-semibold hover:bg-primary-700 transition-colors duration-150 cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2"
        >
          <UserPlus className="w-4 h-4" />
          <span>Add Candidate</span>
        </Link>
      </div>

      {/* Search */}
      <div className="bg-white rounded-lg border border-slate-200 px-4 py-3 flex gap-3 items-center">
        <div className="relative flex-1 max-w-sm">
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
        <button
          onClick={handleSearch}
          className="px-3 py-2 text-sm bg-primary text-white rounded-lg hover:bg-primary-700 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1"
        >
          Search
        </button>
        {activeSearch && (
          <button
            onClick={clearSearch}
            className="text-sm text-slate-500 hover:text-red-600 transition-colors cursor-pointer"
          >
            Clear
          </button>
        )}
      </div>

      {/* Table */}
      <div className="bg-white rounded-card shadow-md overflow-hidden">
        {isError ? (
          <ErrorState message="Failed to load candidates." onRetry={refetch} />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="bg-slate-50">
                  <th className="table-header text-left">Name</th>
                  <th className="table-header text-left hidden md:table-cell">Email</th>
                  <th className="table-header text-left hidden lg:table-cell">Phone</th>
                  <th className="table-header text-left hidden lg:table-cell">Location</th>
                  <th className="table-header text-left hidden md:table-cell">Added</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  Array.from({ length: 8 }).map((_, i) => <TableRowSkeleton key={i} cols={5} rowIndex={i} />)
                ) : !data?.items?.length ? (
                  <tr>
                    <td colSpan={5}>
                      <EmptyState
                        icon={Users}
                        title={activeSearch ? 'No candidates match your search' : 'No candidates yet'}
                        description={
                          activeSearch
                            ? 'Try a different search term.'
                            : 'Candidates will appear here once added to the system.'
                        }
                        action={
                          activeSearch ? (
                            <button
                              onClick={clearSearch}
                              className="text-sm text-primary hover:underline cursor-pointer"
                            >
                              Clear search
                            </button>
                          ) : undefined
                        }
                      />
                    </td>
                  </tr>
                ) : (
                  data.items.map((candidate) => {
                    const displayName = candidate.full_name?.trim() || candidate.name?.trim() || candidate.email || 'Candidate'
                    const initial = (displayName.charAt(0) || 'C').toUpperCase()
                    return (
                      <tr key={candidate.id} className="table-row">
                        <td className="table-cell">
                          <div className="flex items-center gap-2">
                            <div className="w-7 h-7 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0">
                              <span className="text-xs font-bold text-primary">
                                {initial}
                              </span>
                            </div>
                            <div>
                              <Link
                                to={`/candidates/${candidate.id}`}
                                className="font-medium text-primary hover:underline cursor-pointer"
                              >
                                {displayName}
                              </Link>
                            </div>
                          </div>
                        </td>
                      <td className="table-cell hidden md:table-cell">
                        <div className="flex items-center gap-1.5 text-slate-600">
                          <Mail className="w-3 h-3 text-slate-400" />
                          <a href={`mailto:${candidate.email}`} className="hover:text-primary transition-colors cursor-pointer">
                            {candidate.email}
                          </a>
                        </div>
                      </td>
                      <td className="table-cell hidden lg:table-cell text-slate-600">
                        {candidate.phone ? (
                          <div className="flex items-center gap-1.5">
                            <Phone className="w-3 h-3 text-slate-400" />
                            {candidate.phone}
                          </div>
                        ) : '—'}
                      </td>
                      <td className="table-cell hidden lg:table-cell text-slate-600">
                        {candidate.location ? (
                          <div className="flex items-center gap-1.5">
                            <MapPin className="w-3 h-3 text-slate-400" />
                            {candidate.location}
                          </div>
                        ) : '—'}
                      </td>
                      <td className="table-cell hidden md:table-cell text-slate-500">
                        {formatDate(candidate.created_at)}
                      </td>
                    </tr>
                  )
                }))}
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

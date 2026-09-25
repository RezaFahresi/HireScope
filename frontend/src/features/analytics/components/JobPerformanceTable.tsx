import React, { useState } from 'react'
import { Briefcase, Search, ArrowUpDown, ChevronDown, ChevronUp } from 'lucide-react'
import type { JobPerformanceItem } from '@/api/analytics'

interface JobPerformanceTableProps {
  jobs: JobPerformanceItem[]
  isLoading?: boolean
}

type SortField = 'job_title' | 'total_candidates' | 'reviewed' | 'shortlisted' | 'interviews' | 'completed_interviews'
type SortOrder = 'asc' | 'desc'

export const JobPerformanceTable: React.FC<JobPerformanceTableProps> = ({ jobs, isLoading }) => {
  const [searchTerm, setSearchTerm] = useState('')
  const [sortField, setSortField] = useState<SortField>('total_candidates')
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc')

  if (isLoading) {
    return (
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm animate-pulse space-y-4">
        <div className="h-6 w-48 bg-slate-200 rounded" />
        <div className="h-40 bg-slate-100 rounded-lg" />
      </div>
    )
  }

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortOrder('desc')
    }
  }

  const filteredJobs = jobs.filter((j) => {
    const term = searchTerm.toLowerCase()
    return (
      j.job_title.toLowerCase().includes(term) ||
      j.job_code.toLowerCase().includes(term) ||
      j.department.toLowerCase().includes(term)
    )
  })

  const sortedJobs = [...filteredJobs].sort((a, b) => {
    let aVal = a[sortField]
    let bVal = b[sortField]

    if (typeof aVal === 'string') {
      return sortOrder === 'asc'
        ? (aVal as string).localeCompare(bVal as string)
        : (bVal as string).localeCompare(aVal as string)
    }

    return sortOrder === 'asc' ? (aVal as number) - (bVal as number) : (bVal as number) - (aVal as number)
  })

  const renderSortIndicator = (field: SortField) => {
    if (sortField !== field) {
      return <ArrowUpDown className="w-3 h-3 text-slate-300 ml-1 inline" />
    }
    return sortOrder === 'asc' ? (
      <ChevronUp className="w-3 h-3 text-primary ml-1 inline" />
    ) : (
      <ChevronDown className="w-3 h-3 text-primary ml-1 inline" />
    )
  }

  return (
    <div className="bg-white border border-slate-200 rounded-xl shadow-sm overflow-hidden space-y-3 p-5">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 border-b border-slate-100 pb-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-900 flex items-center gap-2">
            <Briefcase className="w-4 h-4 text-primary" />
            <span>Job Requisition Performance</span>
          </h3>
          <p className="text-xs text-slate-500 mt-0.5">
            Conversion metrics and volume across open and archived positions ({jobs.length} total)
          </p>
        </div>

        {/* Search input */}
        <div className="relative w-full sm:w-64">
          <Search className="w-3.5 h-3.5 text-slate-400 absolute left-3 top-1/2 -translate-y-1/2" />
          <input
            type="text"
            placeholder="Search requisition or code..."
            value={searchTerm}
            onChange={(e) => setSearchTerm(e.target.value)}
            className="w-full text-xs pl-8 pr-3 py-1.5 border border-slate-200 rounded-lg bg-slate-50 text-slate-700 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-primary/20 focus:border-primary"
          />
        </div>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs border-collapse">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50/75 text-slate-500 font-semibold uppercase tracking-wider text-[11px]">
              <th
                onClick={() => handleSort('job_title')}
                className="py-3 px-3 cursor-pointer hover:text-slate-900 transition-colors"
              >
                Requisition {renderSortIndicator('job_title')}
              </th>
              <th className="py-3 px-3">Department</th>
              <th className="py-3 px-3">Status</th>
              <th
                onClick={() => handleSort('total_candidates')}
                className="py-3 px-3 text-right cursor-pointer hover:text-slate-900 transition-colors"
              >
                Candidates {renderSortIndicator('total_candidates')}
              </th>
              <th
                onClick={() => handleSort('reviewed')}
                className="py-3 px-3 text-right cursor-pointer hover:text-slate-900 transition-colors"
              >
                Reviewed {renderSortIndicator('reviewed')}
              </th>
              <th
                onClick={() => handleSort('shortlisted')}
                className="py-3 px-3 text-right cursor-pointer hover:text-slate-900 transition-colors"
              >
                Shortlisted {renderSortIndicator('shortlisted')}
              </th>
              <th
                onClick={() => handleSort('interviews')}
                className="py-3 px-3 text-right cursor-pointer hover:text-slate-900 transition-colors"
              >
                Interviews {renderSortIndicator('interviews')}
              </th>
              <th
                onClick={() => handleSort('completed_interviews')}
                className="py-3 px-3 text-right cursor-pointer hover:text-slate-900 transition-colors"
              >
                Completed {renderSortIndicator('completed_interviews')}
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {sortedJobs.length === 0 ? (
              <tr>
                <td colSpan={8} className="py-8 text-center text-slate-400">
                  No job requisitions found matching the query
                </td>
              </tr>
            ) : (
              sortedJobs.map((j) => (
                <tr key={j.job_id} className="hover:bg-slate-50/60 transition-colors">
                  <td className="py-3 px-3">
                    <div className="font-semibold text-slate-900">{j.job_title}</div>
                    <div className="font-mono text-[11px] text-slate-400">{j.job_code}</div>
                  </td>
                  <td className="py-3 px-3 text-slate-600 font-medium">{j.department || '—'}</td>
                  <td className="py-3 px-3">
                    <span
                      className={`inline-flex items-center px-2 py-0.5 rounded text-[11px] font-semibold ${
                        j.status === 'OPEN'
                          ? 'bg-emerald-50 text-emerald-700'
                          : j.status === 'DRAFT'
                          ? 'bg-slate-100 text-slate-700'
                          : 'bg-amber-50 text-amber-700'
                      }`}
                    >
                      {j.status}
                    </span>
                  </td>
                  <td className="py-3 px-3 text-right font-mono font-bold text-slate-900">
                    {j.total_candidates}
                  </td>
                  <td className="py-3 px-3 text-right font-mono">
                    <span className="font-semibold text-slate-800">{j.reviewed}</span>
                    <span className="text-[11px] text-slate-400 ml-1">({j.review_rate}%)</span>
                  </td>
                  <td className="py-3 px-3 text-right font-mono">
                    <span className="font-semibold text-slate-800">{j.shortlisted}</span>
                    <span className="text-[11px] text-slate-400 ml-1">({j.shortlist_rate}%)</span>
                  </td>
                  <td className="py-3 px-3 text-right font-mono">
                    <span className="font-semibold text-slate-800">{j.interviews}</span>
                    <span className="text-[11px] text-slate-400 ml-1">({j.interview_rate}%)</span>
                  </td>
                  <td className="py-3 px-3 text-right font-mono">
                    <span className="font-bold text-emerald-700">{j.completed_interviews}</span>
                    {j.interviews > 0 && (
                      <span className="text-[11px] text-slate-400 ml-1">({j.completion_rate}%)</span>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

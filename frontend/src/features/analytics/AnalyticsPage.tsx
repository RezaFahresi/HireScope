import React, { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import {
  Download,
  BarChart3,
  RefreshCw,
  AlertCircle,
  FileSpreadsheet,
} from 'lucide-react'
import { toast } from 'sonner'

import { analyticsApi, type AnalyticsFilterParams } from '@/api/analytics'
import { jobsApi } from '@/api/jobs'
import { interviewsApi } from '@/api/interviews'

import { AnalyticsFilterBar } from './components/AnalyticsFilterBar'
import { KPIRow } from './components/KPIRow'
import { RecruitmentFunnelCard } from './components/RecruitmentFunnelCard'
import { RecruitmentTrendChart } from './components/RecruitmentTrendChart'
import { PipelineScreeningCard } from './components/PipelineScreeningCard'
import { InterviewAnalyticsCard } from './components/InterviewAnalyticsCard'
import { TimeMetricsCard } from './components/TimeMetricsCard'
import { JobPerformanceTable } from './components/JobPerformanceTable'
import { RecruiterActivityTable } from './components/RecruiterActivityTable'

export const AnalyticsPage: React.FC = () => {
  const [filters, setFilters] = useState<AnalyticsFilterParams>({
    interval: 'day',
  })
  const [isExporting, setIsExporting] = useState(false)
  const [activeTableTab, setActiveTableTab] = useState<'jobs' | 'recruiters'>('jobs')

  // 1. Fetch metadata dropdowns
  const { data: departments = [] } = useQuery({
    queryKey: ['analytics-departments'],
    queryFn: () => analyticsApi.getDepartments(),
  })

  const { data: jobsPaginated } = useQuery({
    queryKey: ['jobs-dropdown-list'],
    queryFn: () => jobsApi.list({ limit: 100 }),
  })
  const jobs = jobsPaginated?.items || []

  const { data: recruiters = [] } = useQuery({
    queryKey: ['recruiters-dropdown-list'],
    queryFn: () => interviewsApi.getUsers(),
  })

  // 2. Fetch analytics datasets
  const overviewQuery = useQuery({
    queryKey: ['analytics-overview', filters],
    queryFn: () => analyticsApi.getOverview(filters),
  })

  const jobsQuery = useQuery({
    queryKey: ['analytics-jobs', filters],
    queryFn: () => analyticsApi.getJobPerformance(filters),
  })

  const recruitersQuery = useQuery({
    queryKey: ['analytics-recruiters', filters],
    queryFn: () => analyticsApi.getRecruiterActivity(filters),
  })

  const handleExportCSV = async () => {
    setIsExporting(true)
    try {
      await analyticsApi.exportCSV(filters)
      toast.success('Analytics CSV report downloaded successfully')
    } catch (err: any) {
      toast.error(err?.response?.data?.error?.message || 'Failed to download CSV report')
    } finally {
      setIsExporting(false)
    }
  }

  const handleIntervalChange = (newInterval: 'day' | 'week' | 'month') => {
    setFilters((prev) => ({
      ...prev,
      interval: newInterval,
    }))
  }

  const isLoading = overviewQuery.isLoading || jobsQuery.isLoading || recruitersQuery.isLoading
  const isError = overviewQuery.isError || jobsQuery.isError || recruitersQuery.isError
  const errorMessage =
    overviewQuery.error?.message ||
    jobsQuery.error?.message ||
    recruitersQuery.error?.message ||
    'Failed to load analytics data'

  const overview = overviewQuery.data

  return (
    <div className="space-y-6 pb-12 max-w-7xl mx-auto">
      {/* Top Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-slate-200/80 pb-5">
        <div>
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded-lg bg-primary/10 text-primary flex items-center justify-center">
              <BarChart3 className="w-4 h-4" />
            </div>
            <h1 className="text-xl font-bold text-slate-900 tracking-tight">
              Recruitment Analytics & Reporting
            </h1>
          </div>
          <p className="text-xs text-slate-500 mt-1">
            Deterministic pipeline intelligence, throughput velocity, and requisition metrics derived strictly from operational records
          </p>
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => {
              overviewQuery.refetch()
              jobsQuery.refetch()
              recruitersQuery.refetch()
            }}
            disabled={isLoading}
            className="inline-flex items-center gap-1.5 px-3 py-2 border border-slate-200 rounded-lg text-xs font-semibold text-slate-700 bg-white hover:bg-slate-50 shadow-xs transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-3.5 h-3.5 text-slate-500 ${isLoading ? 'animate-spin' : ''}`} />
            <span>Refresh</span>
          </button>

          <button
            type="button"
            onClick={handleExportCSV}
            disabled={isExporting}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-lg text-xs font-semibold text-white bg-primary hover:bg-primary/90 shadow-xs transition-colors disabled:opacity-50"
          >
            {isExporting ? (
              <RefreshCw className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Download className="w-3.5 h-3.5" />
            )}
            <span>Export CSV</span>
          </button>
        </div>
      </div>

      {/* Global Filter Bar */}
      <AnalyticsFilterBar
        filters={filters}
        onFiltersChange={setFilters}
        jobs={jobs}
        departments={departments}
        recruiters={recruiters}
        isLoading={isLoading}
      />

      {/* Error Banner */}
      {isError && (
        <div className="p-4 bg-rose-50 border border-rose-200 rounded-xl flex items-start gap-3">
          <AlertCircle className="w-5 h-5 text-rose-600 flex-shrink-0 mt-0.5" />
          <div className="flex-1">
            <h4 className="text-xs font-bold text-rose-800 uppercase tracking-wider">
              Failed to load analytics
            </h4>
            <p className="text-xs text-rose-600 mt-0.5">{errorMessage}</p>
          </div>
          <button
            type="button"
            onClick={() => {
              overviewQuery.refetch()
              jobsQuery.refetch()
              recruitersQuery.refetch()
            }}
            className="text-xs font-medium text-rose-700 underline hover:text-rose-900"
          >
            Try Again
          </button>
        </div>
      )}

      {/* Top Headline KPI Row */}
      {overview && (
        <KPIRow
          summary={overview.summary}
          funnel={overview.funnel}
          isLoading={overviewQuery.isLoading}
        />
      )}

      {/* Recruitment Activity Trend Chart */}
      {overview && (
        <RecruitmentTrendChart
          data={overview.trends}
          interval={(filters.interval as 'day' | 'week' | 'month') || 'day'}
          onIntervalChange={handleIntervalChange}
          isLoading={overviewQuery.isLoading}
        />
      )}

      {/* Funnel & Velocity Grid */}
      {overview && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <RecruitmentFunnelCard
            funnel={overview.funnel}
            isLoading={overviewQuery.isLoading}
          />
          <div className="space-y-6">
            <TimeMetricsCard
              metrics={overview.time_metrics}
              isLoading={overviewQuery.isLoading}
            />
            <InterviewAnalyticsCard
              interviews={overview.interviews}
              isLoading={overviewQuery.isLoading}
            />
          </div>
        </div>
      )}

      {/* Pipeline & Deterministic Screening Breakdown */}
      {overview && (
        <PipelineScreeningCard
          pipeline={overview.pipeline}
          screening={overview.screening}
          isLoading={overviewQuery.isLoading}
        />
      )}

      {/* Tables Section with Tabs */}
      <div className="space-y-4 pt-2">
        <div className="flex items-center justify-between border-b border-slate-200">
          <div className="flex items-center gap-1 -mb-px">
            <button
              type="button"
              onClick={() => setActiveTableTab('jobs')}
              className={`px-4 py-2.5 text-xs font-semibold border-b-2 transition-colors flex items-center gap-2 ${
                activeTableTab === 'jobs'
                  ? 'border-primary text-primary font-bold'
                  : 'border-transparent text-slate-500 hover:text-slate-800'
              }`}
            >
              <FileSpreadsheet className="w-4 h-4" />
              <span>Job Requisitions Performance ({jobsQuery.data?.total ?? 0})</span>
            </button>
            <button
              type="button"
              onClick={() => setActiveTableTab('recruiters')}
              className={`px-4 py-2.5 text-xs font-semibold border-b-2 transition-colors flex items-center gap-2 ${
                activeTableTab === 'recruiters'
                  ? 'border-primary text-primary font-bold'
                  : 'border-transparent text-slate-500 hover:text-slate-800'
              }`}
            >
              <BarChart3 className="w-4 h-4" />
              <span>Recruiter Operational Workload ({recruitersQuery.data?.total ?? 0})</span>
            </button>
          </div>
        </div>

        {activeTableTab === 'jobs' ? (
          <JobPerformanceTable
            jobs={jobsQuery.data?.jobs ?? []}
            isLoading={jobsQuery.isLoading}
          />
        ) : (
          <RecruiterActivityTable
            recruiters={recruitersQuery.data?.recruiters ?? []}
            isLoading={recruitersQuery.isLoading}
          />
        )}
      </div>
    </div>
  )
}

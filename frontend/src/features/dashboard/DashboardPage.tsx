import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import {
  Briefcase,
  Users,
  UserCheck,
  Clock,
  Plus,
  RefreshCw,
} from 'lucide-react'
import { useAuth } from '@/features/auth/AuthProvider'
import { dashboardApi } from '@/api/dashboard'
import { ErrorState } from '@/components/feedback'
import { DashboardMetricCard } from './components/DashboardMetricCard'
import { CandidatePipelineCard } from './components/CandidatePipelineCard'
import { JobStatusCard } from './components/JobStatusCard'
import { NeedsAttentionCard } from './components/NeedsAttentionCard'
import { RecentCandidatesTable } from './components/RecentCandidatesTable'
import { OpenJobsTable } from './components/OpenJobsTable'
import { RecentActivityList } from './components/RecentActivityList'

export function DashboardPage() {
  const { user } = useAuth()
  const [isRefreshing, setIsRefreshing] = useState(false)

  // 1. Dashboard KPI summary
  const summaryQuery = useQuery({
    queryKey: ['dashboard', 'summary'],
    queryFn: dashboardApi.getSummary,
  })

  // 2. Recent candidate pipeline
  const recentCandidatesQuery = useQuery({
    queryKey: ['dashboard', 'recent-candidates'],
    queryFn: () => dashboardApi.getRecentCandidates(5),
  })

  // 3. Open jobs with candidate pipeline count
  const openJobsQuery = useQuery({
    queryKey: ['dashboard', 'open-jobs'],
    queryFn: () => dashboardApi.getOpenJobs(5),
  })

  // 4. Audit activity log stream
  const activityQuery = useQuery({
    queryKey: ['dashboard', 'activity'],
    queryFn: () => dashboardApi.getActivity(8),
  })

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await Promise.allSettled([
      summaryQuery.refetch(),
      recentCandidatesQuery.refetch(),
      openJobsQuery.refetch(),
      activityQuery.refetch(),
    ])
    setIsRefreshing(false)
  }

  const summary = summaryQuery.data

  return (
    <div className="max-w-7xl mx-auto space-y-6 pb-12">
      {/* Page Header */}
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-4 border-b border-border/60 pb-5">
        <div>
          <h1 className="text-xl sm:text-2xl font-bold font-mono tracking-tight text-foreground">
            Welcome back, {user?.name?.split(' ')[0] || 'Recruiter'}
          </h1>
          <p className="text-xs sm:text-sm text-muted-foreground mt-1">
            HireScope Pipeline Intelligence & Recruiter Operations Overview
          </p>
        </div>

        <div className="flex items-center gap-2.5">
          <button
            onClick={handleRefresh}
            disabled={isRefreshing}
            className="inline-flex items-center gap-1.5 px-3 py-2 text-xs font-medium rounded-md border border-border bg-card text-foreground hover:bg-muted/60 disabled:opacity-50 transition-colors cursor-pointer"
            title="Refresh dashboard metrics"
          >
            <RefreshCw className={`h-3.5 w-3.5 ${isRefreshing ? 'animate-spin' : ''}`} />
            <span className="hidden sm:inline">Refresh</span>
          </button>

          <Link
            to="/jobs/new"
            className="inline-flex items-center gap-1.5 bg-primary text-primary-foreground px-4 py-2 rounded-md text-xs font-semibold hover:bg-primary/90 transition-colors shadow-2xs"
          >
            <Plus className="h-4 w-4" />
            <span>New Job Requisition</span>
          </Link>
        </div>
      </div>

      {/* Section Error State if Summary fails */}
      {summaryQuery.isError && (
        <ErrorState
          message="Unable to load dashboard intelligence summary."
          onRetry={summaryQuery.refetch}
          className="py-8 bg-card border border-destructive/20 rounded-lg"
        />
      )}

      {/* Needs Attention Alert Bar */}
      {summary && (
        <NeedsAttentionCard
          awaitingReviewCount={summary.needs_attention.awaiting_review_count}
          openJobsNoCandidates={summary.needs_attention.open_jobs_no_candidates}
        />
      )}

      {/* Top Metric Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <DashboardMetricCard
          label="Active Jobs"
          value={summary?.active_jobs ?? 0}
          icon={Briefcase}
          description="Open & actively hiring"
          linkTo="/jobs?status=OPEN"
          isLoading={summaryQuery.isLoading}
        />
        <DashboardMetricCard
          label="Total Candidates"
          value={summary?.total_candidates ?? 0}
          icon={Users}
          description="Across all requisitions"
          linkTo="/candidates"
          isLoading={summaryQuery.isLoading}
        />
        <DashboardMetricCard
          label="Awaiting Review"
          value={summary?.review_candidates ?? 0}
          icon={Clock}
          description="Pending evaluation"
          linkTo="/jobs"
          isLoading={summaryQuery.isLoading}
        />
        <DashboardMetricCard
          label="Shortlisted"
          value={summary?.shortlisted_candidates ?? 0}
          icon={UserCheck}
          description="Advancing through pipeline"
          linkTo="/candidates"
          isLoading={summaryQuery.isLoading}
        />
      </div>

      {/* Visual Pipeline Breakdown & Job Status Cards */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
        <CandidatePipelineCard
          pipeline={
            summary?.pipeline ?? {
              review: 0,
              shortlisted: 0,
              rejected: 0,
            }
          }
          isLoading={summaryQuery.isLoading}
        />

        <JobStatusCard
          statusCounts={
            summary?.job_status ?? {
              open: 0,
              draft: 0,
              closed: 0,
              archived: 0,
            }
          }
          totalJobs={
            (summary?.job_status.open ?? 0) +
            (summary?.job_status.draft ?? 0) +
            (summary?.job_status.closed ?? 0) +
            (summary?.job_status.archived ?? 0)
          }
        />
      </div>

      {/* Tables Row: Recent Candidates & Open Requisitions with Pipeline Counts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-5">
        <RecentCandidatesTable
          candidates={recentCandidatesQuery.data ?? []}
          isLoading={recentCandidatesQuery.isLoading}
        />

        <OpenJobsTable
          jobs={openJobsQuery.data ?? []}
          isLoading={openJobsQuery.isLoading}
        />
      </div>

      {/* Recruiter & System Audit Activity Stream */}
      <RecentActivityList
        activities={activityQuery.data ?? []}
        isLoading={activityQuery.isLoading}
      />
    </div>
  )
}

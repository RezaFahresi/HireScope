import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { calendarApi } from '@/api/calendar'
import type { CalendarConnectionDTO, CalendarProviderType, InterviewCalendarEventDTO } from '@/types'
import {
  Calendar,
  CheckCircle2,
  AlertCircle,
  ExternalLink,
  Loader2,
  RefreshCw,
  PlusCircle,
} from 'lucide-react'
import { toast } from 'sonner'

interface ExternalCalendarSectionProps {
  interviewId: string
  interviewStatus: string
}

export function ExternalCalendarSection({
  interviewId,
  interviewStatus,
}: ExternalCalendarSectionProps) {
  const queryClient = useQueryClient()

  // 1. Fetch user's calendar connections
  const { data: connections, isLoading: loadingConns } = useQuery<CalendarConnectionDTO[]>({
    queryKey: ['calendar-connections'],
    queryFn: calendarApi.getConnections,
  })

  // 2. Fetch external calendar events for this interview
  const { data: events, isLoading: loadingEvents } = useQuery<InterviewCalendarEventDTO[]>({
    queryKey: ['interview-calendar-events', interviewId],
    queryFn: () => calendarApi.getInterviewEvents(interviewId),
    enabled: Boolean(interviewId),
  })

  const syncMutation = useMutation({
    mutationFn: ({ provider }: { provider: CalendarProviderType }) =>
      calendarApi.syncInterview(interviewId, provider),
    onSuccess: (data) => {
      const label = data.provider === 'GOOGLE' ? 'Google Calendar' : 'Microsoft Outlook'
      toast.success(`Interview synced to ${label}!`)
      queryClient.invalidateQueries({ queryKey: ['interview-calendar-events', interviewId] })
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.message || 'Failed to sync to external calendar')
    },
  })

  const retryMutation = useMutation({
    mutationFn: ({ provider }: { provider: CalendarProviderType }) =>
      calendarApi.retrySync(interviewId, provider),
    onSuccess: (data) => {
      const label = data.provider === 'GOOGLE' ? 'Google Calendar' : 'Microsoft Outlook'
      toast.success(`Re-synced with ${label}!`)
      queryClient.invalidateQueries({ queryKey: ['interview-calendar-events', interviewId] })
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.message || 'Failed to retry calendar synchronization')
    },
  })

  const googleConn = connections?.find((c) => c.provider === 'GOOGLE')
  const microsoftConn = connections?.find((c) => c.provider === 'MICROSOFT')

  const googleEvent = events?.find((e) => e.provider === 'GOOGLE')
  const microsoftEvent = events?.find((e) => e.provider === 'MICROSOFT')

  const isLoading = loadingConns || loadingEvents
  const isCancelled = interviewStatus === 'CANCELLED'

  const renderProviderRow = (
    provider: CalendarProviderType,
    label: string,
    conn?: CalendarConnectionDTO,
    event?: InterviewCalendarEventDTO
  ) => {
    const isConnected = conn?.status === 'CONNECTED'
    const isConfigured = conn?.is_configured !== false
    const isSynced = event?.sync_status === 'SYNCED'
    const isFailed = event?.sync_status === 'FAILED'
    const isDeleted = event?.sync_status === 'DELETED'
    const isPendingSync = syncMutation.isPending && syncMutation.variables?.provider === provider
    const isPendingRetry = retryMutation.isPending && retryMutation.variables?.provider === provider

    return (
      <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 p-3.5 rounded-lg border border-slate-200 bg-white">
        <div className="flex items-start gap-3">
          <div className="w-8 h-8 rounded-md bg-slate-100 flex items-center justify-center flex-shrink-0 mt-0.5">
            <Calendar className="w-4 h-4 text-slate-700" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="text-sm font-semibold text-slate-900">{label}</span>

              {/* Status Pills */}
              {isSynced && (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700 border border-emerald-200">
                  <CheckCircle2 className="w-3 h-3" />
                  Synced
                </span>
              )}
              {isFailed && (
                <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-700 border border-red-200">
                  <AlertCircle className="w-3 h-3" />
                  Sync failed
                </span>
              )}
              {isDeleted && (
                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-500 border border-slate-200">
                  Cancelled / Deleted
                </span>
              )}
              {!event && isConnected && (
                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600 border border-slate-200">
                  Not synced
                </span>
              )}
              {!isConnected && isConfigured && (
                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-500 border border-slate-200">
                  Not connected
                </span>
              )}
              {!isConfigured && (
                <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-400 border border-slate-200">
                  Not configured
                </span>
              )}
            </div>

            <div className="text-xs text-slate-500 mt-1">
              {isSynced && event?.last_synced_at && (
                <span>
                  Last synced {new Date(event.last_synced_at).toLocaleString('en-US', {
                    month: 'short',
                    day: 'numeric',
                    year: 'numeric',
                    hour: 'numeric',
                    minute: '2-digit',
                  })}
                </span>
              )}
              {isFailed && (
                <span className="text-red-600">
                  {event?.last_error || 'Synchronization failed'}
                </span>
              )}
              {!event && isConnected && (
                <span>Connected as {conn?.provider_email}</span>
              )}
              {!isConnected && isConfigured && (
                <span>Connect your account in Settings to sync with {label}</span>
              )}
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2 self-end sm:self-center">
          {!isConnected ? (
            <Link
              to="/settings/integrations"
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-slate-200 text-slate-700 hover:bg-slate-50 transition-colors"
            >
              <ExternalLink className="w-3.5 h-3.5" />
              Connect {label}
            </Link>
          ) : isFailed ? (
            <button
              type="button"
              onClick={() => retryMutation.mutate({ provider })}
              disabled={isPendingRetry || isCancelled}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-amber-50 text-amber-800 border border-amber-300 hover:bg-amber-100 transition-colors disabled:opacity-50"
            >
              {isPendingRetry ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <RefreshCw className="w-3.5 h-3.5" />
              )}
              Retry Sync
            </button>
          ) : isSynced ? (
            <button
              type="button"
              onClick={() => syncMutation.mutate({ provider })}
              disabled={isPendingSync || isCancelled}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-slate-200 text-slate-700 hover:bg-slate-50 transition-colors disabled:opacity-50"
              title="Push interview details to external calendar"
            >
              {isPendingSync ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <RefreshCw className="w-3.5 h-3.5" />
              )}
              Re-sync
            </button>
          ) : (
            <button
              type="button"
              onClick={() => syncMutation.mutate({ provider })}
              disabled={isPendingSync || isCancelled}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-primary text-white hover:bg-primary/90 transition-colors shadow-sm disabled:opacity-50"
            >
              {isPendingSync ? (
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <PlusCircle className="w-3.5 h-3.5" />
              )}
              Add to {label}
            </button>
          )}
        </div>
      </div>
    )
  }

  return (
    <div className="bg-white border border-slate-200 rounded-lg p-5 space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-base font-semibold text-slate-900 flex items-center gap-2">
            <Calendar className="w-4 h-4 text-primary" />
            External Calendar Synchronization
          </h3>
          <p className="text-xs text-slate-500 mt-0.5">
            Optionally sync this interview into your connected Google Calendar or Microsoft Outlook.
          </p>
        </div>
        <Link
          to="/settings/integrations"
          className="text-xs text-primary font-medium hover:underline flex items-center gap-1"
        >
          Manage Integrations
          <ExternalLink className="w-3 h-3" />
        </Link>
      </div>

      {isLoading ? (
        <div className="py-6 text-center">
          <Loader2 className="w-5 h-5 animate-spin mx-auto text-slate-400" />
          <p className="text-xs text-slate-500 mt-2">Checking external calendar status...</p>
        </div>
      ) : (
        <div className="space-y-2.5">
          {renderProviderRow('GOOGLE', 'Google Calendar', googleConn, googleEvent)}
          {renderProviderRow('MICROSOFT', 'Microsoft Outlook', microsoftConn, microsoftEvent)}
        </div>
      )}
    </div>
  )
}

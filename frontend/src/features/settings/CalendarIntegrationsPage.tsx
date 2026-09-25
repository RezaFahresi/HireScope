import { useEffect } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import { calendarApi } from '@/api/calendar'
import type { CalendarConnectionDTO, CalendarProviderType } from '@/types'
import {
  Calendar,
  CheckCircle2,
  AlertCircle,
  ExternalLink,
  Loader2,
  Unplug,
  ShieldCheck,
  RefreshCw,
} from 'lucide-react'
import { toast } from 'sonner'

export function CalendarIntegrationsPage() {
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()

  const { data: connections, isLoading, error } = useQuery<CalendarConnectionDTO[]>({
    queryKey: ['calendar-connections'],
    queryFn: calendarApi.getConnections,
  })

  // Handle OAuth callback parameters in URL
  useEffect(() => {
    const calendarParam = searchParams.get('calendar')
    const statusParam = searchParams.get('status')
    const messageParam = searchParams.get('message')

    if (calendarParam && statusParam) {
      const providerLabel = calendarParam === 'google' ? 'Google Calendar' : 'Microsoft Outlook'
      if (statusParam === 'success') {
        toast.success(`${providerLabel} connected successfully!`)
        queryClient.invalidateQueries({ queryKey: ['calendar-connections'] })
      } else {
        toast.error(`Failed to connect ${providerLabel}: ${messageParam || 'Authorization denied'}`)
      }
      // Clean query params from URL
      setSearchParams({}, { replace: true })
    }
  }, [searchParams, setSearchParams, queryClient])

  const connectMutation = useMutation({
    mutationFn: async (provider: CalendarProviderType) => {
      if (provider === 'GOOGLE') {
        return await calendarApi.connectGoogle()
      } else {
        return await calendarApi.connectMicrosoft()
      }
    },
    onSuccess: (data) => {
      if (data?.auth_url) {
        window.location.href = data.auth_url
      }
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.message || 'Failed to initiate authorization flow')
    },
  })

  const disconnectMutation = useMutation({
    mutationFn: async (provider: CalendarProviderType) => {
      if (provider === 'GOOGLE') {
        await calendarApi.disconnectGoogle()
      } else {
        await calendarApi.disconnectMicrosoft()
      }
    },
    onSuccess: (_, provider) => {
      const label = provider === 'GOOGLE' ? 'Google Calendar' : 'Microsoft Outlook'
      toast.success(`${label} disconnected`)
      queryClient.invalidateQueries({ queryKey: ['calendar-connections'] })
    },
    onError: (err: any) => {
      toast.error(err?.response?.data?.message || 'Failed to disconnect calendar')
    },
  })

  const googleConn = connections?.find((c) => c.provider === 'GOOGLE')
  const microsoftConn = connections?.find((c) => c.provider === 'MICROSOFT')

  return (
    <div className="max-w-4xl mx-auto space-y-8">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold font-mono tracking-tight text-slate-900">
          Calendar Integrations
        </h1>
        <p className="text-sm text-slate-500 mt-1">
          Connect your work calendar to sync HireScope interview schedules directly to Google
          Calendar or Microsoft Outlook.
        </p>
      </div>

      {/* Info Notice */}
      <div className="bg-slate-50 border border-slate-200 rounded-lg p-4 flex items-start gap-3">
        <ShieldCheck className="w-5 h-5 text-primary flex-shrink-0 mt-0.5" />
        <div className="text-xs text-slate-600 space-y-1">
          <p className="font-semibold text-slate-800">HireScope Source of Truth</p>
          <p>
            HireScope retains primary authority over all interview schedules. External calendars
            are integrated for visibility; events are created and updated upon explicit recruiter
            actions or when rescheduling.
          </p>
        </div>
      </div>

      {/* Loading & Error States */}
      {isLoading ? (
        <div className="bg-white border border-slate-200 rounded-lg p-12 text-center">
          <Loader2 className="w-6 h-6 animate-spin mx-auto text-slate-400 mb-2" />
          <p className="text-sm text-slate-500">Loading calendar integrations...</p>
        </div>
      ) : error ? (
        <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center text-red-700">
          <AlertCircle className="w-6 h-6 mx-auto mb-2 text-red-600" />
          <p className="text-sm font-semibold">Failed to load calendar connections</p>
          <p className="text-xs mt-1 text-red-500">{(error as any)?.message}</p>
        </div>
      ) : (
        <div className="space-y-4">
          {/* Google Calendar Card */}
          <div className="bg-white border border-slate-200 rounded-lg p-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
            <div className="flex items-start gap-4">
              <div className="w-10 h-10 rounded-lg bg-blue-50 border border-blue-100 flex items-center justify-center flex-shrink-0">
                <Calendar className="w-5 h-5 text-blue-600" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-slate-900">Google Calendar</h3>
                  {googleConn?.status === 'CONNECTED' && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700 border border-emerald-200">
                      <CheckCircle2 className="w-3 h-3" />
                      Connected
                    </span>
                  )}
                  {googleConn?.status === 'DISCONNECTED' && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600 border border-slate-200">
                      Not connected
                    </span>
                  )}
                  {(googleConn?.status === 'EXPIRED' || googleConn?.status === 'REVOKED') && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-50 text-amber-700 border border-amber-200">
                      <AlertCircle className="w-3 h-3" />
                      Auth expired
                    </span>
                  )}
                  {googleConn?.status === 'NOT_CONFIGURED' && (
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-500 border border-slate-200">
                      Not configured
                    </span>
                  )}
                </div>

                <p className="text-xs text-slate-500 mt-1">
                  {googleConn?.status === 'CONNECTED' && googleConn.provider_email ? (
                    <span>Connected as <strong className="text-slate-700">{googleConn.provider_email}</strong></span>
                  ) : googleConn?.status === 'NOT_CONFIGURED' ? (
                    <span>Google OAuth credentials are not configured on this server.</span>
                  ) : (
                    <span>Sync scheduled interviews directly to your Google primary calendar.</span>
                  )}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
              {googleConn?.status === 'CONNECTED' ? (
                <button
                  type="button"
                  onClick={() => disconnectMutation.mutate('GOOGLE')}
                  disabled={disconnectMutation.isPending}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-slate-200 text-slate-700 hover:bg-red-50 hover:text-red-700 hover:border-red-200 transition-colors"
                >
                  {disconnectMutation.isPending ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Unplug className="w-3.5 h-3.5" />
                  )}
                  Disconnect
                </button>
              ) : googleConn?.status === 'NOT_CONFIGURED' ? (
                <button
                  disabled
                  className="px-3 py-1.5 rounded-md text-xs font-medium bg-slate-100 text-slate-400 cursor-not-allowed border border-slate-200"
                >
                  Unavailable
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => connectMutation.mutate('GOOGLE')}
                  disabled={connectMutation.isPending}
                  className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-md text-xs font-medium bg-primary text-white hover:bg-primary/90 transition-colors shadow-sm"
                >
                  {connectMutation.isPending ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : googleConn?.status === 'EXPIRED' || googleConn?.status === 'REVOKED' ? (
                    <RefreshCw className="w-3.5 h-3.5" />
                  ) : (
                    <ExternalLink className="w-3.5 h-3.5" />
                  )}
                  {googleConn?.status === 'EXPIRED' || googleConn?.status === 'REVOKED'
                    ? 'Reconnect Google'
                    : 'Connect Google Calendar'}
                </button>
              )}
            </div>
          </div>

          {/* Microsoft Outlook Card */}
          <div className="bg-white border border-slate-200 rounded-lg p-6 flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
            <div className="flex items-start gap-4">
              <div className="w-10 h-10 rounded-lg bg-sky-50 border border-sky-100 flex items-center justify-center flex-shrink-0">
                <Calendar className="w-5 h-5 text-sky-600" />
              </div>
              <div>
                <div className="flex items-center gap-2">
                  <h3 className="font-semibold text-slate-900">Microsoft Outlook</h3>
                  {microsoftConn?.status === 'CONNECTED' && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-emerald-50 text-emerald-700 border border-emerald-200">
                      <CheckCircle2 className="w-3 h-3" />
                      Connected
                    </span>
                  )}
                  {microsoftConn?.status === 'DISCONNECTED' && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600 border border-slate-200">
                      Not connected
                    </span>
                  )}
                  {(microsoftConn?.status === 'EXPIRED' || microsoftConn?.status === 'REVOKED') && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-amber-50 text-amber-700 border border-amber-200">
                      <AlertCircle className="w-3 h-3" />
                      Auth expired
                    </span>
                  )}
                  {microsoftConn?.status === 'NOT_CONFIGURED' && (
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-500 border border-slate-200">
                      Not configured
                    </span>
                  )}
                </div>

                <p className="text-xs text-slate-500 mt-1">
                  {microsoftConn?.status === 'CONNECTED' && microsoftConn.provider_email ? (
                    <span>Connected as <strong className="text-slate-700">{microsoftConn.provider_email}</strong></span>
                  ) : microsoftConn?.status === 'NOT_CONFIGURED' ? (
                    <span>Microsoft OAuth credentials are not configured on this server.</span>
                  ) : (
                    <span>Sync scheduled interviews directly to your Microsoft 365 or Outlook calendar.</span>
                  )}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2 w-full sm:w-auto justify-end">
              {microsoftConn?.status === 'CONNECTED' ? (
                <button
                  type="button"
                  onClick={() => disconnectMutation.mutate('MICROSOFT')}
                  disabled={disconnectMutation.isPending}
                  className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium border border-slate-200 text-slate-700 hover:bg-red-50 hover:text-red-700 hover:border-red-200 transition-colors"
                >
                  {disconnectMutation.isPending ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : (
                    <Unplug className="w-3.5 h-3.5" />
                  )}
                  Disconnect
                </button>
              ) : microsoftConn?.status === 'NOT_CONFIGURED' ? (
                <button
                  disabled
                  className="px-3 py-1.5 rounded-md text-xs font-medium bg-slate-100 text-slate-400 cursor-not-allowed border border-slate-200"
                >
                  Unavailable
                </button>
              ) : (
                <button
                  type="button"
                  onClick={() => connectMutation.mutate('MICROSOFT')}
                  disabled={connectMutation.isPending}
                  className="inline-flex items-center gap-1.5 px-3.5 py-1.5 rounded-md text-xs font-medium bg-primary text-white hover:bg-primary/90 transition-colors shadow-sm"
                >
                  {connectMutation.isPending ? (
                    <Loader2 className="w-3.5 h-3.5 animate-spin" />
                  ) : microsoftConn?.status === 'EXPIRED' || microsoftConn?.status === 'REVOKED' ? (
                    <RefreshCw className="w-3.5 h-3.5" />
                  ) : (
                    <ExternalLink className="w-3.5 h-3.5" />
                  )}
                  {microsoftConn?.status === 'EXPIRED' || microsoftConn?.status === 'REVOKED'
                    ? 'Reconnect Outlook'
                    : 'Connect Outlook'}
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

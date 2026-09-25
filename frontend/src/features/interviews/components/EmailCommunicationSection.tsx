import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  Mail,
  Send,
  RotateCw,
  CheckCircle2,
  Clock,
  AlertCircle,
  AlertTriangle,
  User,
  Users,
  X,
} from 'lucide-react'
import { interviewsApi } from '@/api/interviews'
import { getApiError } from '@/api/client'
import { formatDateTime } from '@/lib/utils'
import { toast } from 'sonner'
import type { Interview, DeliveryStatus } from '@/types'

interface EmailCommunicationSectionProps {
  interview: Interview
}

export function EmailCommunicationSection({ interview }: EmailCommunicationSectionProps) {
  const queryClient = useQueryClient()
  const [isSendDialogOpen, setIsSendDialogOpen] = useState(false)
  const [sendCandidate, setSendCandidate] = useState(true)
  const [sendInterviewers, setSendInterviewers] = useState(true)

  const {
    data: deliveries = [],
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['interview-email-deliveries', interview.id],
    queryFn: () => interviewsApi.getEmailDeliveries(interview.id),
    enabled: !!interview.id,
  })

  const sendMutation = useMutation({
    mutationFn: () =>
      interviewsApi.sendInvitations(interview.id, {
        send_candidate: sendCandidate,
        send_interviewers: sendInterviewers,
      }),
    onSuccess: (data) => {
      queryClient.invalidateQueries({
        queryKey: ['interview-email-deliveries', interview.id],
      })
      toast.success(
        `Dispatched ${data.length} email communication record(s).`
      )
      setIsSendDialogOpen(false)
    },
    onError: (err) => {
      toast.error(getApiError(err))
    },
  })

  const retryMutation = useMutation({
    mutationFn: (deliveryId: string) =>
      interviewsApi.retryEmailDelivery(interview.id, deliveryId),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: ['interview-email-deliveries', interview.id],
      })
      toast.success('Email delivery attempt retried successfully.')
    },
    onError: (err) => {
      toast.error(getApiError(err))
    },
  })

  const renderStatusBadge = (status: DeliveryStatus) => {
    switch (status) {
      case 'SENT':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200">
            <CheckCircle2 className="w-3 h-3" />
            Sent
          </span>
        )
      case 'PENDING':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-amber-50 text-amber-700 border border-amber-200">
            <Clock className="w-3 h-3" />
            Scheduled
          </span>
        )
      case 'SENDING':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-blue-50 text-blue-700 border border-blue-200 animate-pulse">
            <RotateCw className="w-3 h-3 animate-spin" />
            Sending…
          </span>
        )
      case 'FAILED':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-red-50 text-red-700 border border-red-200">
            <AlertCircle className="w-3 h-3" />
            Failed
          </span>
        )
      case 'SKIPPED':
        return (
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-semibold bg-slate-100 text-slate-600 border border-slate-200">
            <AlertTriangle className="w-3 h-3 text-slate-400" />
            Skipped
          </span>
        )
      default:
        return (
          <span className="px-2 py-0.5 rounded text-[11px] font-semibold bg-slate-100 text-slate-600 border border-slate-200">
            {status}
          </span>
        )
    }
  }

  const candidateName = interview.candidate?.full_name || 'Candidate'
  const candidateEmail = interview.candidate?.email || ''
  const interviewerCount = interview.interviewers?.length || 0

  return (
    <div className="rounded-lg border border-border bg-card p-5 space-y-4 shadow-xs text-xs">
      {/* Header */}
      <div className="flex items-center justify-between pb-3 border-b border-border/60">
        <div className="flex items-center gap-2">
          <Mail className="w-4 h-4 text-primary" />
          <h2 className="text-sm font-semibold text-foreground">Email Communication</h2>
          {deliveries.length > 0 && (
            <span className="px-1.5 py-0.5 text-[10px] font-semibold rounded bg-muted text-muted-foreground border border-border">
              {deliveries.length}
            </span>
          )}
        </div>

        <button
          type="button"
          onClick={() => setIsSendDialogOpen(true)}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-md border border-border bg-background text-foreground hover:bg-muted transition-colors cursor-pointer"
        >
          <Send className="w-3 h-3 text-primary" />
          Send Invitation
        </button>
      </div>

      {/* Deliveries list */}
      {isLoading ? (
        <div className="py-6 text-center text-muted-foreground animate-pulse">
          Loading email communications…
        </div>
      ) : isError ? (
        <div className="py-4 text-center text-red-600 space-y-2">
          <p>Failed to load email delivery logs.</p>
          <button
            onClick={() => refetch()}
            className="text-xs text-primary underline font-medium cursor-pointer"
          >
            Retry
          </button>
        </div>
      ) : deliveries.length === 0 ? (
        <div className="py-6 text-center space-y-2 border border-dashed border-border rounded-md bg-muted/20">
          <Mail className="w-8 h-8 text-muted-foreground mx-auto opacity-50" />
          <p className="font-medium text-foreground text-xs">No email communications yet</p>
          <p className="text-[11px] text-muted-foreground max-w-sm mx-auto">
            Invitations and automated reminders scheduled for this interview will appear here with delivery timestamps.
          </p>
          <button
            type="button"
            onClick={() => setIsSendDialogOpen(true)}
            className="mt-2 inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors shadow-2xs cursor-pointer"
          >
            <Send className="w-3 h-3" />
            Send Interview Invitation
          </button>
        </div>
      ) : (
        <div className="space-y-2.5">
          {deliveries.map((delivery) => {
            const isRetrying =
              retryMutation.isPending && retryMutation.variables === delivery.id

            return (
              <div
                key={delivery.id}
                className="p-3.5 rounded-md border border-border bg-background space-y-2 hover:border-border/80 transition-colors"
              >
                {/* Row Header */}
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="flex items-center gap-2 min-w-0">
                    {delivery.recipient_type === 'CANDIDATE' ? (
                      <span className="p-1 rounded bg-blue-50 text-blue-700">
                        <User className="w-3.5 h-3.5" />
                      </span>
                    ) : (
                      <span className="p-1 rounded bg-purple-50 text-purple-700">
                        <Users className="w-3.5 h-3.5" />
                      </span>
                    )}
                    <div className="min-w-0">
                      <div className="flex items-center gap-1.5">
                        <span className="font-semibold text-foreground truncate">
                          {delivery.recipient_name}
                        </span>
                        <span className="text-[10px] text-muted-foreground uppercase font-mono px-1 rounded bg-muted/60">
                          {delivery.recipient_type}
                        </span>
                      </div>
                      <p className="text-[11px] text-muted-foreground font-mono truncate">
                        {delivery.recipient_email}
                      </p>
                    </div>
                  </div>

                  <div className="flex items-center gap-2 shrink-0">
                    <span className="px-2 py-0.5 rounded text-[10px] font-medium border border-border bg-muted/40 text-muted-foreground uppercase">
                      {delivery.email_type === 'INTERVIEW_INVITATION' ? 'Invitation' : 'Reminder'}
                    </span>
                    {renderStatusBadge(delivery.status)}
                  </div>
                </div>

                {/* Row Meta & Error */}
                <div className="flex flex-wrap items-center justify-between gap-2 pt-1 border-t border-border/40 text-[11px] text-muted-foreground">
                  <div className="space-x-3">
                    {delivery.sent_at && (
                      <span>Sent: {formatDateTime(delivery.sent_at)}</span>
                    )}
                    {!delivery.sent_at && delivery.scheduled_for && (
                      <span>Scheduled for: {formatDateTime(delivery.scheduled_for)}</span>
                    )}
                    {delivery.provider && (
                      <span className="font-mono text-[10px] text-muted-foreground/80">
                        via {delivery.provider}
                      </span>
                    )}
                    {delivery.attempt_count > 1 && (
                      <span className="font-medium text-amber-700">
                        (Attempt {delivery.attempt_count}/3)
                      </span>
                    )}
                  </div>

                  {delivery.status === 'FAILED' && (
                    <button
                      type="button"
                      onClick={() => retryMutation.mutate(delivery.id)}
                      disabled={isRetrying || delivery.attempt_count >= 3}
                      className="inline-flex items-center gap-1 px-2.5 py-1 rounded text-xs font-semibold bg-red-600 text-white hover:bg-red-700 transition-colors disabled:opacity-50 cursor-pointer shadow-2xs"
                      title={
                        delivery.attempt_count >= 3
                          ? 'Max retry attempts reached'
                          : 'Retry sending email'
                      }
                    >
                      <RotateCw className={`w-3 h-3 ${isRetrying ? 'animate-spin' : ''}`} />
                      {isRetrying ? 'Retrying…' : 'Retry'}
                    </button>
                  )}
                </div>

                {/* Error Banner if Failed or Skipped */}
                {delivery.last_error && (
                  <div
                    className={`mt-1.5 p-2 rounded text-[11px] font-mono ${
                      delivery.status === 'FAILED'
                        ? 'bg-red-50 text-red-800 border border-red-200'
                        : 'bg-slate-50 text-slate-700 border border-slate-200'
                    }`}
                  >
                    <span className="font-semibold block mb-0.5">
                      {delivery.status === 'FAILED' ? 'Delivery Error:' : 'Status Detail:'}
                    </span>
                    {delivery.last_error}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      )}

      {/* Manual Send Invitation Modal */}
      {isSendDialogOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
          <div className="bg-card border border-border rounded-lg shadow-xl w-full max-w-md overflow-hidden flex flex-col animate-in fade-in zoom-in-95 duration-100">
            <div className="px-5 py-4 border-b border-border flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Mail className="w-4 h-4 text-primary" />
                <h3 className="text-sm font-bold text-foreground">Send Interview Invitation</h3>
              </div>
              <button
                type="button"
                onClick={() => setIsSendDialogOpen(false)}
                className="text-muted-foreground hover:text-foreground cursor-pointer"
              >
                <X className="w-4 h-4" />
              </button>
            </div>

            <div className="p-5 space-y-4 text-xs">
              <p className="text-muted-foreground">
                Select the recipients to receive formal interview invitations for this session:
              </p>

              <div className="space-y-3 p-3 rounded-md border border-border bg-muted/20">
                <label className="flex items-start gap-2.5 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={sendCandidate}
                    onChange={(e) => setSendCandidate(e.target.checked)}
                    className="mt-0.5 accent-primary rounded"
                  />
                  <div>
                    <span className="font-semibold text-foreground block">
                      Candidate: {candidateName}
                    </span>
                    <span className="text-muted-foreground text-[11px] font-mono">
                      {candidateEmail || 'No candidate email available'}
                    </span>
                  </div>
                </label>

                <div className="border-t border-border/40 pt-2.5">
                  <label className="flex items-start gap-2.5 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={sendInterviewers}
                      onChange={(e) => setSendInterviewers(e.target.checked)}
                      className="mt-0.5 accent-primary rounded"
                    />
                    <div>
                      <span className="font-semibold text-foreground block">
                        Assigned Interviewer(s) ({interviewerCount})
                      </span>
                      <div className="text-muted-foreground text-[11px] mt-0.5 space-y-0.5">
                        {interview.interviewers?.map((ii) => (
                          <div key={ii.id} className="truncate">
                            &bull; {ii.name} ({ii.email})
                          </div>
                        ))}
                      </div>
                    </div>
                  </label>
                </div>
              </div>

              <div className="p-2.5 rounded bg-blue-50 border border-blue-200 text-blue-900 text-[11px] leading-relaxed">
                ℹ️ Duplicate sends are safely protected by idempotency keys. If already delivered, invitations will not be re-sent unless explicitly retried.
              </div>
            </div>

            <div className="px-5 py-3.5 bg-muted/30 border-t border-border flex items-center justify-end gap-2.5">
              <button
                type="button"
                onClick={() => setIsSendDialogOpen(false)}
                disabled={sendMutation.isPending}
                className="px-3.5 py-1.5 text-xs font-semibold rounded-md border border-border bg-background text-foreground hover:bg-muted transition-colors cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => sendMutation.mutate()}
                disabled={sendMutation.isPending || (!sendCandidate && !sendInterviewers)}
                className="inline-flex items-center gap-1.5 px-4 py-1.5 text-xs font-semibold rounded-md bg-primary text-primary-foreground hover:bg-primary/90 transition-colors shadow-2xs disabled:opacity-50 cursor-pointer"
              >
                <Send className="w-3.5 h-3.5" />
                {sendMutation.isPending ? 'Sending Invitations…' : 'Send Invitations'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

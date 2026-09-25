import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { AlertTriangle } from 'lucide-react'
import { interviewsApi } from '@/api/interviews'
import { getApiError } from '@/api/client'
import { toast } from 'sonner'
import type { Interview } from '@/types'

interface CancelDialogProps {
  isOpen: boolean
  onClose: () => void
  interview: Interview
  onSuccess?: () => void
}

export function CancelDialog({
  isOpen,
  onClose,
  interview,
  onSuccess,
}: CancelDialogProps) {
  const queryClient = useQueryClient()
  const [reason, setReason] = useState('')
  const [errorMsg, setErrorMsg] = useState('')

  const cancelMutation = useMutation({
    mutationFn: async () => {
      setErrorMsg('')
      const trimmed = reason.trim()
      if (!trimmed) {
        throw new Error('Please enter a cancellation reason.')
      }
      return interviewsApi.cancel(interview.id, { reason: trimmed })
    },
    onSuccess: () => {
      toast.success('Interview cancelled')
      queryClient.invalidateQueries({ queryKey: ['interview', interview.id] })
      queryClient.invalidateQueries({ queryKey: ['interviews'] })
      queryClient.invalidateQueries({ queryKey: ['candidate-interviews', interview.candidate_id] })
      queryClient.invalidateQueries({ queryKey: ['job-interviews', interview.job_id] })
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      queryClient.invalidateQueries({ queryKey: ['analytics-overview'] })
      onClose()
      onSuccess?.()
    },
    onError: (err) => {
      const msg = getApiError(err)
      setErrorMsg(msg)
      toast.error(msg)
    },
  })

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-in fade-in duration-150">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md overflow-hidden flex flex-col border border-slate-200">
        <div className="px-5 py-4 border-b border-slate-100 flex items-center gap-2">
          <AlertTriangle className="w-4 h-4 text-red-600" />
          <h3 className="text-base font-bold text-slate-800">Cancel Interview</h3>
        </div>

        <div className="p-5 space-y-4 text-xs">
          <p className="text-slate-600">
            Are you sure you want to cancel the interview{' '}
            <strong className="text-slate-800">"{interview.title}"</strong>?
          </p>
          <div className="p-3 bg-amber-50 border border-amber-200 rounded-md text-amber-800">
            This interview will remain recorded in candidate history as cancelled and will not be
            deleted.
          </div>

          {errorMsg && (
            <div className="p-2.5 bg-red-50 border border-red-200 rounded-md text-red-700">
              {errorMsg}
            </div>
          )}

          <div className="space-y-1">
            <label className="font-medium text-slate-700 block">
              Cancellation Reason <span className="text-red-500">*</span>
            </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={3}
              placeholder="e.g. Candidate withdrew application / Position on hold"
              className="w-full px-3 py-2 border border-slate-200 rounded-md text-slate-800 focus:outline-none focus:ring-1 focus:ring-primary text-xs resize-none"
              maxLength={1000}
              required
            />
            <div className="text-[11px] text-slate-400 text-right">{reason.length} / 1000</div>
          </div>
        </div>

        <div className="px-5 py-3.5 bg-slate-50 flex justify-end gap-2 border-t border-slate-100">
          <button
            type="button"
            onClick={onClose}
            disabled={cancelMutation.isPending}
            className="px-4 py-2 text-xs font-semibold text-slate-600 hover:bg-slate-200 bg-slate-100 rounded-md transition-colors cursor-pointer"
          >
            Keep Interview
          </button>
          <button
            type="button"
            onClick={() => cancelMutation.mutate()}
            disabled={cancelMutation.isPending || !reason.trim()}
            className="px-4 py-2 text-xs font-semibold text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors cursor-pointer disabled:opacity-50"
          >
            {cancelMutation.isPending ? 'Cancelling…' : 'Cancel Interview'}
          </button>
        </div>
      </div>
    </div>
  )
}

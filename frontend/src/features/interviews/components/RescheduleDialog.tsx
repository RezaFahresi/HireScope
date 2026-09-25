import { useState, useEffect } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Calendar, Clock, AlertCircle } from 'lucide-react'
import { interviewsApi } from '@/api/interviews'
import { getApiError } from '@/api/client'
import { formatTimeRange } from '@/lib/utils'
import { toast } from 'sonner'
import type { Interview } from '@/types'

interface RescheduleDialogProps {
  isOpen: boolean
  onClose: () => void
  interview: Interview
  onSuccess?: () => void
}

export function RescheduleDialog({
  isOpen,
  onClose,
  interview,
  onSuccess,
}: RescheduleDialogProps) {
  const queryClient = useQueryClient()

  // Prepopulate with interview start date
  const [date, setDate] = useState('')
  const [startTime, setStartTime] = useState('')
  const [endTime, setEndTime] = useState('')
  const [reason, setReason] = useState('')
  const [formError, setFormError] = useState('')

  useEffect(() => {
    if (interview && isOpen) {
      const s = new Date(interview.scheduled_start)
      const e = new Date(interview.scheduled_end)

      const yyyy = s.getFullYear()
      const mm = String(s.getMonth() + 1).padStart(2, '0')
      const dd = String(s.getDate()).padStart(2, '0')
      setDate(`${yyyy}-${mm}-${dd}`)

      const sHour = String(s.getHours()).padStart(2, '0')
      const sMin = String(s.getMinutes()).padStart(2, '0')
      setStartTime(`${sHour}:${sMin}`)

      const eHour = String(e.getHours()).padStart(2, '0')
      const eMin = String(e.getMinutes()).padStart(2, '0')
      setEndTime(`${eHour}:${eMin}`)

      setReason('')
      setFormError('')
    }
  }, [interview, isOpen])

  const rescheduleMutation = useMutation({
    mutationFn: async () => {
      setFormError('')
      if (!date || !startTime || !endTime) {
        throw new Error('Please specify date, start time, and end time.')
      }

      const scheduledStart = new Date(`${date}T${startTime}:00`).toISOString()
      const scheduledEnd = new Date(`${date}T${endTime}:00`).toISOString()

      if (new Date(scheduledEnd) <= new Date(scheduledStart)) {
        throw new Error('End time must be strictly after start time.')
      }

      return interviewsApi.reschedule(interview.id, {
        scheduled_start: scheduledStart,
        scheduled_end: scheduledEnd,
        reason: reason.trim() || undefined,
      })
    },
    onSuccess: () => {
      toast.success('Interview rescheduled successfully')
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
      setFormError(msg)
      toast.error(msg)
    },
  })

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4 animate-in fade-in duration-150">
      <div className="bg-white rounded-lg shadow-xl w-full max-w-md overflow-hidden flex flex-col border border-slate-200">
        <div className="px-5 py-4 border-b border-slate-100 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Calendar className="w-4 h-4 text-primary" />
            <h3 className="text-base font-bold text-slate-800">Reschedule Interview</h3>
          </div>
          <span className="text-xs text-slate-400 font-mono">Asia/Jakarta (GMT+7)</span>
        </div>

        <div className="p-5 space-y-4 text-xs">
          {/* Current schedule info */}
          <div className="bg-slate-50 border border-slate-200 rounded-md p-3">
            <p className="text-[11px] text-slate-400 uppercase tracking-wide font-medium">
              Current Time
            </p>
            <p className="text-slate-800 font-medium mt-0.5">
              {formatTimeRange(interview.scheduled_start, interview.scheduled_end)}
            </p>
          </div>

          {formError && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-md text-red-700 flex items-start gap-2">
              <AlertCircle className="w-4 h-4 shrink-0 mt-0.5" />
              <span>{formError}</span>
            </div>
          )}

          {/* New Date */}
          <div className="space-y-1">
            <label className="font-medium text-slate-700 block">
              New Date <span className="text-red-500">*</span>
            </label>
            <input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
              className="w-full px-3 py-2 border border-slate-200 rounded-md text-slate-800 focus:outline-none focus:ring-1 focus:ring-primary text-xs"
              required
            />
          </div>

          {/* Start and End Times */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1">
              <label className="font-medium text-slate-700 flex items-center gap-1">
                <Clock className="w-3 h-3 text-slate-400" /> Start Time{' '}
                <span className="text-red-500">*</span>
              </label>
              <input
                type="time"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
                className="w-full px-3 py-2 border border-slate-200 rounded-md text-slate-800 focus:outline-none focus:ring-1 focus:ring-primary text-xs"
                required
              />
            </div>
            <div className="space-y-1">
              <label className="font-medium text-slate-700 flex items-center gap-1">
                <Clock className="w-3 h-3 text-slate-400" /> End Time{' '}
                <span className="text-red-500">*</span>
              </label>
              <input
                type="time"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
                className="w-full px-3 py-2 border border-slate-200 rounded-md text-slate-800 focus:outline-none focus:ring-1 focus:ring-primary text-xs"
                required
              />
            </div>
          </div>

          {/* Reschedule Reason */}
          <div className="space-y-1">
            <label className="font-medium text-slate-700 block">
              Reschedule Reason <span className="text-slate-400 font-normal">(optional)</span>
            </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              rows={2}
              placeholder="e.g. Candidate requested different time slot"
              className="w-full px-3 py-2 border border-slate-200 rounded-md text-slate-800 focus:outline-none focus:ring-1 focus:ring-primary text-xs resize-none"
              maxLength={500}
            />
          </div>
        </div>

        <div className="px-5 py-3.5 bg-slate-50 flex justify-end gap-2 border-t border-slate-100">
          <button
            type="button"
            onClick={onClose}
            disabled={rescheduleMutation.isPending}
            className="px-4 py-2 text-xs font-semibold text-slate-600 hover:bg-slate-200 bg-slate-100 rounded-md transition-colors cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => rescheduleMutation.mutate()}
            disabled={rescheduleMutation.isPending}
            className="px-4 py-2 text-xs font-semibold text-white bg-primary hover:bg-primary-700 rounded-md transition-colors cursor-pointer disabled:opacity-50"
          >
            {rescheduleMutation.isPending ? 'Rescheduling…' : 'Confirm Reschedule'}
          </button>
        </div>
      </div>
    </div>
  )
}

import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2 } from 'lucide-react'
import { interviewsApi } from '@/api/interviews'
import { getApiError } from '@/api/client'
import { toast } from 'sonner'
import type { Interview, InterviewResult } from '@/types'

interface CompleteDialogProps {
  isOpen: boolean
  onClose: () => void
  interview: Interview
  onSuccess?: () => void
}

const RESULT_OPTIONS: { value: InterviewResult; label: string; desc: string }[] = [
  { value: 'PASSED', label: 'Passed', desc: 'Candidate met assessment expectations' },
  { value: 'FAILED', label: 'Failed', desc: 'Candidate did not meet stage requirements' },
  { value: 'ON_HOLD', label: 'On Hold', desc: 'Evaluation pending team calibration' },
  { value: 'NO_SHOW', label: 'No Show', desc: 'Candidate did not attend session' },
]

export function CompleteDialog({
  isOpen,
  onClose,
  interview,
  onSuccess,
}: CompleteDialogProps) {
  const queryClient = useQueryClient()
  const [result, setResult] = useState<InterviewResult>('PASSED')
  const [feedback, setFeedback] = useState('')
  const [errorMsg, setErrorMsg] = useState('')

  const completeMutation = useMutation({
    mutationFn: async () => {
      setErrorMsg('')
      return interviewsApi.complete(interview.id, {
        result,
        feedback: feedback.trim() || undefined,
      })
    },
    onSuccess: () => {
      toast.success('Interview marked as completed')
      queryClient.invalidateQueries({ queryKey: ['interview', interview.id] })
      queryClient.invalidateQueries({ queryKey: ['interviews'] })
      queryClient.invalidateQueries({ queryKey: ['candidate-interviews', interview.candidate_id] })
      queryClient.invalidateQueries({ queryKey: ['job-interviews', interview.job_id] })
      queryClient.invalidateQueries({ queryKey: ['dashboard'] })
      queryClient.invalidateQueries({ queryKey: ['analytics-overview'] })
      queryClient.invalidateQueries({ queryKey: ['analytics-jobs'] })
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
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg overflow-hidden flex flex-col border border-slate-200">
        <div className="px-5 py-4 border-b border-slate-100 flex items-center gap-2">
          <CheckCircle2 className="w-4 h-4 text-emerald-600" />
          <h3 className="text-base font-bold text-slate-800">Complete Interview & Record Outcome</h3>
        </div>

        <div className="p-5 space-y-4 text-xs">
          <div className="p-3 bg-slate-50 border border-slate-200 rounded-md">
            <span className="font-semibold text-slate-800">{interview.title}</span>
            <span className="text-slate-500 ml-1.5 font-normal">
              for {interview.candidate?.full_name || 'Candidate'} ({interview.job?.title || 'Job'})
            </span>
          </div>

          {errorMsg && (
            <div className="p-2.5 bg-red-50 border border-red-200 rounded-md text-red-700">
              {errorMsg}
            </div>
          )}

          {/* Assessment Result */}
          <div className="space-y-2">
            <label className="font-medium text-slate-700 block">
              Interview Result <span className="text-red-500">*</span>
            </label>
            <div className="grid grid-cols-2 gap-2">
              {RESULT_OPTIONS.map((opt) => (
                <label
                  key={opt.value}
                  className={`flex flex-col p-2.5 rounded-md border cursor-pointer transition-colors ${
                    result === opt.value
                      ? 'border-primary bg-primary/5 text-primary'
                      : 'border-slate-200 hover:bg-slate-50 text-slate-700'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <input
                      type="radio"
                      name="interview_result"
                      value={opt.value}
                      checked={result === opt.value}
                      onChange={() => setResult(opt.value)}
                      className="accent-primary"
                    />
                    <span className="font-semibold text-xs text-slate-900">{opt.label}</span>
                  </div>
                  <span className="text-[11px] text-slate-500 mt-1 pl-5">{opt.desc}</span>
                </label>
              ))}
            </div>
          </div>

          {/* Detailed Feedback */}
          <div className="space-y-1">
            <label className="font-medium text-slate-700 block">
              Feedback & Evaluation Notes <span className="text-slate-400 font-normal">(optional)</span>
            </label>
            <textarea
              value={feedback}
              onChange={(e) => setFeedback(e.target.value)}
              rows={4}
              placeholder="Record candidate strengths, key observations, technical assessment notes..."
              className="w-full px-3 py-2 border border-slate-200 rounded-md text-slate-800 focus:outline-none focus:ring-1 focus:ring-primary text-xs resize-none"
              maxLength={5000}
            />
            <div className="flex justify-between items-center text-[11px] text-slate-400">
              <span>Recruiter outcome notes are strictly internal.</span>
              <span>{feedback.length} / 5000</span>
            </div>
          </div>
        </div>

        <div className="px-5 py-3.5 bg-slate-50 flex justify-end gap-2 border-t border-slate-100">
          <button
            type="button"
            onClick={onClose}
            disabled={completeMutation.isPending}
            className="px-4 py-2 text-xs font-semibold text-slate-600 hover:bg-slate-200 bg-slate-100 rounded-md transition-colors cursor-pointer"
          >
            Cancel
          </button>
          <button
            type="button"
            onClick={() => completeMutation.mutate()}
            disabled={completeMutation.isPending}
            className="px-4 py-2 text-xs font-semibold text-white bg-emerald-600 hover:bg-emerald-700 rounded-md transition-colors cursor-pointer disabled:opacity-50"
          >
            {completeMutation.isPending ? 'Saving…' : 'Mark Completed'}
          </button>
        </div>
      </div>
    </div>
  )
}

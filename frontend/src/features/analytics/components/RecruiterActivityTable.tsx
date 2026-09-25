import React from 'react'
import { UserCheck, Shield, Clock } from 'lucide-react'
import type { RecruiterActivityRow } from '@/api/analytics'

interface RecruiterActivityTableProps {
  recruiters: RecruiterActivityRow[]
  isLoading?: boolean
}

export const RecruiterActivityTable: React.FC<RecruiterActivityTableProps> = ({
  recruiters,
  isLoading,
}) => {
  if (isLoading) {
    return (
      <div className="bg-white border border-slate-200 rounded-xl p-5 shadow-sm animate-pulse space-y-4">
        <div className="h-6 w-48 bg-slate-200 rounded" />
        <div className="h-32 bg-slate-100 rounded-lg" />
      </div>
    )
  }

  const formatActivityTime = (iso?: string) => {
    if (!iso) return 'No recorded activity'
    const date = new Date(iso)
    if (isNaN(date.getTime())) return '—'
    return new Intl.DateTimeFormat('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    }).format(date)
  }

  return (
    <div className="bg-white border border-slate-200 rounded-xl shadow-sm overflow-hidden space-y-3 p-5">
      <div className="flex items-center justify-between border-b border-slate-100 pb-3">
        <div>
          <h3 className="text-sm font-semibold text-slate-900 flex items-center gap-2">
            <UserCheck className="w-4 h-4 text-primary" />
            <span>Recruiter Operational Activity</span>
          </h3>
          <p className="text-xs text-slate-500 mt-0.5">
            Operational throughput, review volumes, note logs, and scheduled interviews across team members
          </p>
        </div>
        <span className="text-[11px] text-slate-400 font-mono">
          {recruiters.length} active recruiters
        </span>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs border-collapse">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50/75 text-slate-500 font-semibold uppercase tracking-wider text-[11px]">
              <th className="py-3 px-3">Team Member</th>
              <th className="py-3 px-3">Role</th>
              <th className="py-3 px-3 text-right">Candidates Reviewed</th>
              <th className="py-3 px-3 text-right">Notes Created</th>
              <th className="py-3 px-3 text-right">Interviews Scheduled</th>
              <th className="py-3 px-3 text-right">Interviews Completed</th>
              <th className="py-3 px-3 text-right">Last Action</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {recruiters.length === 0 ? (
              <tr>
                <td colSpan={7} className="py-8 text-center text-slate-400">
                  No recruiter activity records found
                </td>
              </tr>
            ) : (
              recruiters.map((r) => {
                const initial = (r.recruiter_name?.charAt(0) || 'U').toUpperCase()

                return (
                  <tr key={r.recruiter_id} className="hover:bg-slate-50/60 transition-colors">
                    <td className="py-3 px-3">
                      <div className="flex items-center gap-2.5">
                        <div className="w-7 h-7 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0 text-primary font-bold text-xs">
                          {initial}
                        </div>
                        <div>
                          <div className="font-semibold text-slate-900">{r.recruiter_name}</div>
                          <div className="text-[11px] text-slate-400">{r.recruiter_email}</div>
                        </div>
                      </div>
                    </td>
                    <td className="py-3 px-3">
                      <span className="inline-flex items-center gap-1 text-[11px] font-medium text-slate-600 bg-slate-100 px-2 py-0.5 rounded">
                        <Shield className="w-3 h-3 text-slate-400" />
                        {r.role}
                      </span>
                    </td>
                    <td className="py-3 px-3 text-right font-mono font-bold text-slate-900">
                      {r.candidates_reviewed}
                    </td>
                    <td className="py-3 px-3 text-right font-mono font-medium text-slate-700">
                      {r.notes_created}
                    </td>
                    <td className="py-3 px-3 text-right font-mono font-medium text-indigo-700">
                      {r.interviews_scheduled}
                    </td>
                    <td className="py-3 px-3 text-right font-mono font-bold text-emerald-700">
                      {r.interviews_completed}
                    </td>
                    <td className="py-3 px-3 text-right text-slate-500 font-mono text-[11px]">
                      <span className="inline-flex items-center gap-1">
                        <Clock className="w-3 h-3 text-slate-300" />
                        {formatActivityTime(r.last_activity_at)}
                      </span>
                    </td>
                  </tr>
                )
              })
            )}
          </tbody>
        </table>
      </div>
    </div>
  )
}

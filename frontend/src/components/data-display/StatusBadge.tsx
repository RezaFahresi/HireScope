import { cn } from '@/lib/utils'
import { Video, MapPin, Phone } from 'lucide-react'
import type {
  JobStatus,
  ScreeningStatus,
  WorkflowStatus,
  MatchStatus,
  InterviewStatus,
  InterviewStage,
  InterviewType,
  InterviewResult,
} from '@/types'

// ─── StatusBadge ──────────────────────────────────────────────

type BadgeVariant = 'default' | 'success' | 'warning' | 'danger' | 'info' | 'muted'

const variantStyles: Record<BadgeVariant, string> = {
  default: 'bg-slate-100 text-slate-700',
  success: 'bg-green-50 text-green-700 border border-green-200',
  warning: 'bg-amber-50 text-amber-700 border border-amber-200',
  danger: 'bg-red-50 text-red-700 border border-red-200',
  info: 'bg-blue-50 text-blue-700 border border-blue-200',
  muted: 'bg-slate-100 text-slate-500',
}

interface StatusBadgeProps {
  children: React.ReactNode
  variant?: BadgeVariant
  className?: string
}

export function StatusBadge({ children, variant = 'default', className }: StatusBadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2 py-0.5 rounded text-xs font-semibold',
        variantStyles[variant],
        className
      )}
    >
      {children}
    </span>
  )
}

// ─── Job Status Badge ─────────────────────────────────────────



const jobStatusVariant: Record<JobStatus, BadgeVariant> = {
  DRAFT: 'muted',
  OPEN: 'success',
  CLOSED: 'warning',
  ARCHIVED: 'danger',
}

const jobStatusLabel: Record<JobStatus, string> = {
  DRAFT: 'Draft',
  OPEN: 'Open',
  CLOSED: 'Closed',
  ARCHIVED: 'Archived',
}

export function JobStatusBadge({ status }: { status: JobStatus }) {
  return (
    <StatusBadge variant={jobStatusVariant[status]}>
      {jobStatusLabel[status]}
    </StatusBadge>
  )
}

// ─── Screening Status Badge ────────────────────────────────────



const screeningVariant: Record<ScreeningStatus, BadgeVariant> = {
  QUALIFIED: 'success',
  REVIEW: 'warning',
  NOT_QUALIFIED: 'danger',
}

const screeningLabel: Record<ScreeningStatus, string> = {
  QUALIFIED: 'Qualified',
  REVIEW: 'Review',
  NOT_QUALIFIED: 'Not Qualified',
}

export function ScreeningStatusBadge({ status }: { status: ScreeningStatus }) {
  return (
    <StatusBadge variant={screeningVariant[status]}>
      {screeningLabel[status]}
    </StatusBadge>
  )
}

// ─── Workflow Status Badge ─────────────────────────────────────



const workflowVariant: Record<WorkflowStatus, BadgeVariant> = {
  REVIEW: 'info',
  SHORTLISTED: 'success',
  REJECTED: 'danger',
}

const workflowLabel: Record<WorkflowStatus, string> = {
  REVIEW: 'Review',
  SHORTLISTED: 'Shortlisted',
  REJECTED: 'Rejected',
}

export function WorkflowStatusBadge({ status }: { status: WorkflowStatus }) {
  return (
    <StatusBadge variant={workflowVariant[status]}>
      {workflowLabel[status]}
    </StatusBadge>
  )
}

// ─── Match Status Badge ────────────────────────────────────────

const matchVariant: Record<MatchStatus, BadgeVariant> = {
  MATCH: 'success',
  PARTIAL: 'warning',
  MISMATCH: 'danger',
  UNKNOWN: 'muted',
  NOT_APPLICABLE: 'muted',
}

const matchLabel: Record<MatchStatus, string> = {
  MATCH: 'Match',
  PARTIAL: 'Partial',
  MISMATCH: 'Mismatch',
  UNKNOWN: 'Unknown',
  NOT_APPLICABLE: 'N/A',
}

export function MatchStatusBadge({ status }: { status: MatchStatus }) {
  return (
    <StatusBadge variant={matchVariant[status]}>
      {matchLabel[status]}
    </StatusBadge>
  )
}

// ─── Interview Status Badge (Step 12A) ─────────────────────────

const interviewStatusVariant: Record<InterviewStatus, BadgeVariant> = {
  SCHEDULED: 'info',
  COMPLETED: 'success',
  CANCELLED: 'danger',
}

const interviewStatusLabel: Record<InterviewStatus, string> = {
  SCHEDULED: 'Scheduled',
  COMPLETED: 'Completed',
  CANCELLED: 'Cancelled',
}

export function InterviewStatusBadge({ status }: { status: InterviewStatus }) {
  return (
    <StatusBadge variant={interviewStatusVariant[status]}>
      {interviewStatusLabel[status]}
    </StatusBadge>
  )
}

// ─── Interview Stage Badge ─────────────────────────────────────

const stageLabels: Record<InterviewStage, string> = {
  PHONE_SCREEN: 'Phone Screen',
  HR_INTERVIEW: 'HR Interview',
  TECHNICAL_INTERVIEW: 'Technical Interview',
  MANAGER_INTERVIEW: 'Manager Interview',
  FINAL_INTERVIEW: 'Final Interview',
  OTHER: 'Other',
}

export function InterviewStageBadge({ stage }: { stage: InterviewStage }) {
  return (
    <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 text-slate-700 border border-slate-200/60">
      {stageLabels[stage] || stage}
    </span>
  )
}

// ─── Interview Type Badge ──────────────────────────────────────

export function InterviewTypeBadge({ type }: { type: InterviewType }) {
  if (type === 'ONLINE') {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-blue-50 text-blue-700 border border-blue-200/60">
        <Video className="w-3 h-3 text-blue-600" />
        Online
      </span>
    )
  }
  if (type === 'ONSITE') {
    return (
      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-purple-50 text-purple-700 border border-purple-200/60">
        <MapPin className="w-3 h-3 text-purple-600" />
        Onsite
      </span>
    )
  }
  return (
    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded text-[11px] font-medium bg-amber-50 text-amber-700 border border-amber-200/60">
      <Phone className="w-3 h-3 text-amber-600" />
      Phone
    </span>
  )
}

// ─── Interview Result Badge ────────────────────────────────────

const resultVariant: Record<InterviewResult, BadgeVariant> = {
  PASSED: 'success',
  FAILED: 'danger',
  ON_HOLD: 'warning',
  NO_SHOW: 'muted',
}

const resultLabels: Record<InterviewResult, string> = {
  PASSED: 'Passed',
  FAILED: 'Failed',
  ON_HOLD: 'On Hold',
  NO_SHOW: 'No Show',
}

export function InterviewResultBadge({ result }: { result: InterviewResult }) {
  return (
    <StatusBadge variant={resultVariant[result]}>
      {resultLabels[result] || result}
    </StatusBadge>
  )
}


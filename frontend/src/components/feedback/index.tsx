import { Loader2, type LucideIcon } from 'lucide-react'
import { cn } from '@/lib/utils'

// ─── Loading Spinner ──────────────────────────────────────────

interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg'
  className?: string
}

const spinnerSizes = { sm: 'w-4 h-4', md: 'w-6 h-6', lg: 'w-8 h-8' }

export function LoadingSpinner({ size = 'md', className }: LoadingSpinnerProps) {
  return (
    <Loader2
      className={cn('animate-spin text-primary', spinnerSizes[size], className)}
    />
  )
}

// ─── Loading State (full area) ────────────────────────────────

interface LoadingStateProps {
  message?: string
  className?: string
}

export function LoadingState({ message = 'Loading…', className }: LoadingStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center py-20 gap-3', className)}>
      <LoadingSpinner size="lg" />
      <p className="text-sm text-slate-400">{message}</p>
    </div>
  )
}

// ─── Skeleton ─────────────────────────────────────────────────

interface SkeletonProps {
  className?: string
  style?: React.CSSProperties
}

export function Skeleton({ className, style }: SkeletonProps) {
  return <div className={cn('skeleton', className)} style={style} />
}

// ─── Table Row Skeleton ───────────────────────────────────────

const SKELETON_WIDTHS = ['75%', '60%', '80%', '65%', '70%', '55%', '85%', '72%']

export function TableRowSkeleton({ cols = 5, rowIndex = 0 }: { cols?: number; rowIndex?: number }) {
  return (
    <tr>
      {Array.from({ length: cols }).map((_, i) => (
        <td key={i} className="px-4 py-3 border-b border-slate-50">
          <Skeleton className="h-4 rounded" style={{ width: SKELETON_WIDTHS[(rowIndex + i) % SKELETON_WIDTHS.length] }} />
        </td>
      ))}
    </tr>
  )
}

// ─── Empty State ──────────────────────────────────────────────

interface EmptyStateProps {
  icon?: LucideIcon
  title: string
  description?: string
  action?: React.ReactNode
  className?: string
}

export function EmptyState({ icon: Icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center py-20 px-4 text-center', className)}>
      {Icon && (
        <div className="w-12 h-12 rounded-xl bg-slate-100 flex items-center justify-center mb-4">
          <Icon className="w-6 h-6 text-slate-400" />
        </div>
      )}
      <h3 className="text-sm font-semibold text-slate-700 mb-1">{title}</h3>
      {description && (
        <p className="text-sm text-slate-400 max-w-xs mb-4">{description}</p>
      )}
      {action}
    </div>
  )
}

// ─── Error State ──────────────────────────────────────────────

interface ErrorStateProps {
  message?: string
  onRetry?: () => void
  className?: string
}

export function ErrorState({
  message = 'Something went wrong.',
  onRetry,
  className,
}: ErrorStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center py-20 gap-3', className)}>
      <div className="w-12 h-12 rounded-xl bg-red-50 flex items-center justify-center">
        <span className="text-red-500 text-xl">!</span>
      </div>
      <p className="text-sm text-slate-600">{message}</p>
      {onRetry && (
        <button
          onClick={onRetry}
          className="text-sm text-primary hover:underline cursor-pointer transition-all"
        >
          Try again
        </button>
      )}
    </div>
  )
}

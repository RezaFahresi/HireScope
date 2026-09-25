import { Link } from 'react-router-dom'
import { ArrowRight } from 'lucide-react'

interface DashboardMetricCardProps {
  label: string
  value: number | string
  icon: React.ElementType
  description: string
  linkTo?: string
  isLoading?: boolean
}

export function DashboardMetricCard({
  label,
  value,
  icon: Icon,
  description,
  linkTo,
  isLoading = false,
}: DashboardMetricCardProps) {
  const content = (
    <div className="card flex items-start gap-4 group hover:border-slate-300 transition-colors">
      <div className="w-10 h-10 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0 text-primary">
        <Icon className="w-5 h-5" />
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-xs text-slate-500 font-medium mb-1 uppercase tracking-wide">{label}</p>
        {isLoading ? (
          <div className="h-8 w-16 bg-slate-200 animate-pulse rounded" />
        ) : (
          <p className="text-2xl font-mono font-bold text-primary">{value}</p>
        )}
        <p className="text-xs text-slate-400 mt-1">{description}</p>
      </div>
      {linkTo && (
        <ArrowRight className="w-4 h-4 text-slate-300 group-hover:text-primary transition-colors flex-shrink-0 mt-1" />
      )}
    </div>
  )

  if (linkTo) {
    return (
      <Link
        to={linkTo}
        className="cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 rounded-card block"
      >
        {content}
      </Link>
    )
  }
  return content
}

import { useQuery } from '@tanstack/react-query'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Mail, Phone, MapPin, BookOpen, Briefcase, Star } from 'lucide-react'
import { candidatesApi } from '@/api/candidates'
import { LoadingState, ErrorState, EmptyState } from '@/components/feedback'
import { StatusBadge } from '@/components/data-display/StatusBadge'
import { formatDate } from '@/lib/utils'
import { getApiError } from '@/api/client'
import { CandidateDocumentList } from './components/CandidateDocumentList'

// ─── Section Card ─────────────────────────────────────────────

function Section({ title, icon: Icon, children }: {
  title: string
  icon: React.ElementType
  children: React.ReactNode
}) {
  return (
    <div className="bg-white rounded-card shadow-md p-6">
      <div className="flex items-center gap-2 pb-3 mb-4 border-b border-slate-100">
        <Icon className="w-4 h-4 text-primary" />
        <h2 className="text-sm font-semibold text-slate-700">{title}</h2>
      </div>
      {children}
    </div>
  )
}

// ─── Candidate Detail Page ────────────────────────────────────

export function CandidateDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data: candidate, isLoading, isError, error, refetch } = useQuery({
    queryKey: ['candidate', id],
    queryFn: () => candidatesApi.get(id!),
    enabled: !!id,
  })

  if (isLoading) return <LoadingState message="Loading candidate…" />
  if (isError) return <ErrorState message={getApiError(error)} onRetry={refetch} />
  if (!candidate) return null

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-start gap-3">
        <button
          onClick={() => navigate('/candidates')}
          className="mt-1 text-slate-400 hover:text-primary transition-colors cursor-pointer focus:outline-none rounded"
          aria-label="Back to candidates"
        >
          <ArrowLeft className="w-4 h-4" />
        </button>
        <div>
          <h1 className="text-xl font-mono font-bold text-primary">
            {candidate.full_name || candidate.name || 'Candidate'}
          </h1>
          <div className="flex items-center gap-3 mt-1 text-sm text-slate-500">
            {candidate.email && (
              <a href={`mailto:${candidate.email}`} className="flex items-center gap-1 hover:text-primary transition-colors cursor-pointer">
                <Mail className="w-3.5 h-3.5" />
                {candidate.email}
              </a>
            )}
            {candidate.phone && (
              <span className="flex items-center gap-1">
                <Phone className="w-3.5 h-3.5" />
                {candidate.phone}
              </span>
            )}
            {candidate.location && (
              <span className="flex items-center gap-1">
                <MapPin className="w-3.5 h-3.5" />
                {candidate.location}
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Summary */}
      {candidate.summary && (
        <div className="bg-white rounded-card shadow-md p-6">
          <h2 className="text-sm font-semibold text-slate-700 mb-3">Summary</h2>
          <p className="text-sm text-slate-600 leading-relaxed">{candidate.summary}</p>
        </div>
      )}

      {/* Skills */}
      <Section title="Skills" icon={Star}>
        {!candidate.skills?.length ? (
          <EmptyState title="No skills recorded" />
        ) : (
          <div className="flex flex-wrap gap-2">
            {candidate.skills.map((skill) => (
              <StatusBadge key={skill.id} variant="info">
                {skill.name}
                {skill.level && <span className="ml-1 opacity-70">· {skill.level}</span>}
                {skill.years_of_experience && (
                  <span className="ml-1 opacity-70">· {skill.years_of_experience}y</span>
                )}
              </StatusBadge>
            ))}
          </div>
        )}
      </Section>

      {/* Experience */}
      <Section title="Work Experience" icon={Briefcase}>
        {!(candidate.experiences || candidate.experience)?.length ? (
          <EmptyState title="No experience recorded" />
        ) : (
          <div className="space-y-4">
            {(candidate.experiences || candidate.experience)!.map((exp) => (
              <div key={exp.id} className="border-l-2 border-primary/20 pl-4">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="text-sm font-semibold text-slate-800">{exp.title}</p>
                    <p className="text-sm text-slate-600">{exp.company}</p>
                  </div>
                  <div className="text-xs text-slate-400 text-right">
                    <p>{formatDate(exp.start_date)} — {exp.is_current ? 'Present' : formatDate(exp.end_date)}</p>
                  </div>
                </div>
                {exp.description && (
                  <p className="text-xs text-slate-500 mt-2 leading-relaxed">{exp.description}</p>
                )}
              </div>
            ))}
          </div>
        )}
      </Section>

      {/* Education */}
      <Section title="Education" icon={BookOpen}>
        {!(candidate.educations || candidate.education)?.length ? (
          <EmptyState title="No education recorded" />
        ) : (
          <div className="space-y-4">
            {(candidate.educations || candidate.education)!.map((edu) => (
              <div key={edu.id} className="border-l-2 border-primary/20 pl-4">
                <div className="flex items-start justify-between">
                  <div>
                    <p className="text-sm font-semibold text-slate-800">{edu.degree} in {edu.field_of_study}</p>
                    <p className="text-sm text-slate-600">{edu.institution}</p>
                  </div>
                  <div className="text-xs text-slate-400">
                    <p>{formatDate(edu.start_date)} — {edu.is_current ? 'Present' : formatDate(edu.end_date)}</p>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </Section>

      {/* Documents */}
      <div className="bg-white rounded-card shadow-md p-6">
        <CandidateDocumentList
          candidateId={id!}
          initialDocuments={candidate.documents}
          onDocumentProcessed={() => refetch()}
          onDocumentDeleted={() => refetch()}
        />
      </div>

      <p className="text-xs text-slate-400 text-center pb-4">
        Added {formatDate(candidate.created_at)} · Updated {formatDate(candidate.updated_at)}
      </p>
    </div>
  )
}

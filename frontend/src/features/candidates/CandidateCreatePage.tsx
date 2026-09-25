import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod/v4'
import { useNavigate, Link } from 'react-router-dom'
import { ArrowLeft, Loader2, UserPlus } from 'lucide-react'
import { candidatesApi } from '@/api/candidates'
import { documentsApi } from '@/api/documents'
import { getApiError } from '@/api/client'
import { toast } from 'sonner'
import { CandidateCVUpload } from './components/CandidateCVUpload'

const candidateSchema = z.object({
  full_name: z.string().min(1, 'Full name is required').max(255),
  email: z.string().email('Invalid email address').optional().or(z.literal('')),
  phone: z.string().max(50).optional().or(z.literal('')),
  location: z.string().max(100).optional().or(z.literal('')),
  headline: z.string().max(255).optional().or(z.literal('')),
  summary: z.string().optional().or(z.literal('')),
})

type CandidateFormValues = z.infer<typeof candidateSchema>

function FieldError({ message }: { message?: string }) {
  if (!message) return null
  return <p className="mt-1 text-xs text-red-600">{message}</p>
}

export function CandidateCreatePage() {
  const navigate = useNavigate()
  const [selectedCV, setSelectedCV] = useState<File | null>(null)
  const [uploadStep, setUploadStep] = useState<string>('')
  const [isProcessingCV, setIsProcessingCV] = useState(false)

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<CandidateFormValues>({
    resolver: zodResolver(candidateSchema),
    defaultValues: {
      full_name: '',
      email: '',
      phone: '',
      location: '',
      headline: '',
      summary: '',
    },
  })

  const onSubmit = async (values: CandidateFormValues) => {
    try {
      // 1. Create candidate profile first
      setUploadStep('Creating candidate profile...')
      const newCandidate = await candidatesApi.create({
        full_name: values.full_name.trim(),
        email: values.email?.trim() || undefined,
        phone: values.phone?.trim() || undefined,
        location: values.location?.trim() || undefined,
        headline: values.headline?.trim() || undefined,
        summary: values.summary?.trim() || undefined,
      })

      // 2. If CV was selected, upload and process it
      if (selectedCV) {
        setIsProcessingCV(true)
        try {
          setUploadStep('Uploading CV document...')
          const uploadedDoc = await documentsApi.uploadCandidateDocument(newCandidate.id, selectedCV)

          setUploadStep('Processing CV and extracting profile information...')
          await documentsApi.processCandidateDocument(newCandidate.id, uploadedDoc.id)

          toast.success('Candidate created and CV processed successfully')
        } catch (uploadErr) {
          // If CV upload fails, candidate is still created safely
          console.error('CV upload/process error:', uploadErr)
          toast.warning(
            'Candidate was created, but CV could not be processed. You can upload the CV from the candidate profile.'
          )
        } finally {
          setIsProcessingCV(false)
        }
      } else {
        toast.success('Candidate created successfully')
      }

      navigate(`/candidates/${newCandidate.id}`)
    } catch (err) {
      toast.error(getApiError(err))
    } finally {
      setUploadStep('')
    }
  }

  const isBusy = isSubmitting || isProcessingCV

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Link
          to="/candidates"
          className="text-slate-400 hover:text-primary transition-colors cursor-pointer focus:outline-none rounded"
          aria-label="Back to candidates"
        >
          <ArrowLeft className="w-5 h-5" />
        </Link>
        <div>
          <h1 className="text-xl font-mono font-bold text-primary">Add Candidate</h1>
          <p className="text-sm text-slate-500 mt-0.5">
            Add a new candidate profile and optionally upload a CV
          </p>
        </div>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
        {/* Section 1: Candidate Information */}
        <div className="bg-white rounded-card shadow-sm border border-slate-200 p-6 space-y-4">
          <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide border-b border-slate-100 pb-3">
            Candidate Information
          </h2>

          <div className="space-y-4">
            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1" htmlFor="full_name">
                Full Name <span className="text-red-500">*</span>
              </label>
              <input
                id="full_name"
                type="text"
                disabled={isBusy}
                placeholder="e.g. Jane Doe"
                className="w-full input-field"
                {...register('full_name')}
              />
              <FieldError message={errors.full_name?.message} />
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1" htmlFor="email">
                  Email Address
                </label>
                <input
                  id="email"
                  type="email"
                  disabled={isBusy}
                  placeholder="e.g. jane.doe@example.com"
                  className="w-full input-field"
                  {...register('email')}
                />
                <FieldError message={errors.email?.message} />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1" htmlFor="phone">
                  Phone Number
                </label>
                <input
                  id="phone"
                  type="tel"
                  disabled={isBusy}
                  placeholder="e.g. +1 (555) 234-5678"
                  className="w-full input-field"
                  {...register('phone')}
                />
                <FieldError message={errors.phone?.message} />
              </div>
            </div>

            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1" htmlFor="location">
                  Location
                </label>
                <input
                  id="location"
                  type="text"
                  disabled={isBusy}
                  placeholder="e.g. San Francisco, CA"
                  className="w-full input-field"
                  {...register('location')}
                />
                <FieldError message={errors.location?.message} />
              </div>

              <div>
                <label className="block text-xs font-semibold text-slate-700 mb-1" htmlFor="headline">
                  Headline / Role
                </label>
                <input
                  id="headline"
                  type="text"
                  disabled={isBusy}
                  placeholder="e.g. Senior Backend Engineer"
                  className="w-full input-field"
                  {...register('headline')}
                />
                <FieldError message={errors.headline?.message} />
              </div>
            </div>

            <div>
              <label className="block text-xs font-semibold text-slate-700 mb-1" htmlFor="summary">
                Summary
              </label>
              <textarea
                id="summary"
                rows={3}
                disabled={isBusy}
                placeholder="Brief candidate introduction or summary notes..."
                className="w-full input-field resize-y"
                {...register('summary')}
              />
              <FieldError message={errors.summary?.message} />
            </div>
          </div>
        </div>

        {/* Section 2: CV Document (Optional) */}
        <div className="bg-white rounded-card shadow-sm border border-slate-200 p-6 space-y-3">
          <div className="border-b border-slate-100 pb-3">
            <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide">
              CV Document <span className="text-slate-400 font-normal lowercase">(optional)</span>
            </h2>
            <p className="text-xs text-slate-500 mt-0.5">
              Upload a PDF or DOCX file to automatically extract education, experience, and skills upon creation.
            </p>
          </div>

          <CandidateCVUpload
            selectedFile={selectedCV}
            onFileSelect={setSelectedCV}
            disabled={isBusy}
            status={isBusy && selectedCV ? 'uploading' : 'idle'}
            statusMessage={uploadStep}
            hideUploadButton={true}
          />
        </div>

        {/* Submit Actions */}
        <div className="flex items-center justify-between pt-2">
          <Link
            to="/candidates"
            className="text-xs font-semibold text-slate-600 hover:text-slate-900 transition-colors"
          >
            Cancel
          </Link>

          <div className="flex items-center gap-3">
            {uploadStep && (
              <span className="text-xs text-slate-500 flex items-center gap-1.5 animate-pulse">
                <Loader2 className="w-3.5 h-3.5 animate-spin text-primary" />
                {uploadStep}
              </span>
            )}

            <button
              type="submit"
              disabled={isBusy}
              className="flex items-center gap-2 bg-primary text-white px-5 py-2 rounded-lg text-sm font-semibold hover:bg-primary-700 transition-colors duration-150 cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isBusy ? (
                <>
                  <Loader2 className="w-4 h-4 animate-spin" />
                  <span>Processing...</span>
                </>
              ) : (
                <>
                  <UserPlus className="w-4 h-4" />
                  <span>Create Candidate</span>
                </>
              )}
            </button>
          </div>
        </div>
      </form>
    </div>
  )
}

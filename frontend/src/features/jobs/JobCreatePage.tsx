import { useForm, useFieldArray } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod/v4'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Plus, Trash2 } from 'lucide-react'
import { jobsApi } from '@/api/jobs'
import { getApiError } from '@/api/client'
import { toast } from 'sonner'
import type { EmploymentType, RequirementCategory, RequirementImportance } from '@/types'

// ─── Schema ───────────────────────────────────────────────────

const requirementSchema = z.object({
  category: z.enum(['SKILL', 'EXPERIENCE', 'EDUCATION', 'CERTIFICATION', 'LANGUAGE', 'OTHER']),
  description: z.string().min(1, 'Description is required'),
  importance: z.enum(['REQUIRED', 'PREFERRED']),
})

const jobFormSchema = z.object({
  title: z.string().min(1, 'Title is required').max(200),
  description: z.string().min(1, 'Description is required'),
  department: z.string().min(1, 'Department is required'),
  location: z.string().min(1, 'Location is required'),
  employment_type: z.enum(['FULL_TIME', 'PART_TIME', 'CONTRACT', 'INTERNSHIP', 'FREELANCE']),
  closing_date: z.string().optional(),
  status: z.enum(['DRAFT', 'OPEN']),
  requirements: z.array(requirementSchema),
})

type JobFormValues = z.infer<typeof jobFormSchema>

// ─── Field Helpers ────────────────────────────────────────────

function FieldError({ message }: { message?: string }) {
  if (!message) return null
  return <p className="mt-1 text-xs text-red-600">{message}</p>
}

const CATEGORIES: { value: RequirementCategory; label: string }[] = [
  { value: 'SKILL', label: 'Skill' },
  { value: 'EXPERIENCE', label: 'Experience' },
  { value: 'EDUCATION', label: 'Education' },
  { value: 'CERTIFICATION', label: 'Certification' },
  { value: 'LANGUAGE', label: 'Language' },
  { value: 'OTHER', label: 'Other' },
]

// ─── Job Create Page ──────────────────────────────────────────

export function JobCreatePage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const {
    register,
    handleSubmit,
    control,
    formState: { errors, isSubmitting },
  } = useForm<JobFormValues>({
    resolver: zodResolver(jobFormSchema),
    defaultValues: {
      status: 'DRAFT',
      employment_type: 'FULL_TIME',
      requirements: [],
    },
  })

  const { fields, append, remove } = useFieldArray({ control, name: 'requirements' })

  const createJob = useMutation({
    mutationFn: async (values: JobFormValues) => {
      const job = await jobsApi.create({
        title: values.title,
        description: values.description,
        department: values.department,
        location: values.location,
        employment_type: values.employment_type as EmploymentType,
        status: values.status,
        ...(values.closing_date ? { closing_date: values.closing_date } : {}),
      })

      // Add requirements sequentially
      for (const req of values.requirements) {
        await jobsApi.addRequirement(job.id, {
          category: req.category as RequirementCategory,
          description: req.description,
          importance: req.importance as RequirementImportance,
        })
      }

      return job
    },
    onSuccess: (job) => {
      queryClient.invalidateQueries({ queryKey: ['jobs'] })
      toast.success('Job created successfully')
      navigate(`/jobs/${job.id}`)
    },
    onError: (err) => {
      toast.error(getApiError(err))
    },
  })

  const onSubmit = (values: JobFormValues) => {
    createJob.mutate(values)
  }

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div>
        <h1 className="text-xl font-mono font-bold text-primary">Create Job</h1>
        <p className="text-sm text-slate-500 mt-0.5">Fill in the details to create a new job posting.</p>
      </div>

      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-6">
        {/* Basic Info Card */}
        <div className="bg-white rounded-card shadow-md p-6 space-y-4">
          <h2 className="text-sm font-semibold text-slate-700 pb-2 border-b border-slate-100">
            Basic Information
          </h2>

          <div>
            <label className="block text-xs font-semibold text-slate-600 mb-1.5" htmlFor="title">
              Job Title <span className="text-red-500">*</span>
            </label>
            <input {...register('title')} id="title" className="input-field" placeholder="e.g. Senior Software Engineer" />
            <FieldError message={errors.title?.message} />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-600 mb-1.5" htmlFor="department">
                Department <span className="text-red-500">*</span>
              </label>
              <input {...register('department')} id="department" className="input-field" placeholder="e.g. Engineering" />
              <FieldError message={errors.department?.message} />
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-600 mb-1.5" htmlFor="location">
                Location <span className="text-red-500">*</span>
              </label>
              <input {...register('location')} id="location" className="input-field" placeholder="e.g. Jakarta, Remote" />
              <FieldError message={errors.location?.message} />
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-slate-600 mb-1.5" htmlFor="employment_type">
                Employment Type <span className="text-red-500">*</span>
              </label>
              <select {...register('employment_type')} id="employment_type" className="input-field bg-white cursor-pointer">
                <option value="FULL_TIME">Full Time</option>
                <option value="PART_TIME">Part Time</option>
                <option value="CONTRACT">Contract</option>
                <option value="INTERNSHIP">Internship</option>
                <option value="FREELANCE">Freelance</option>
              </select>
              <FieldError message={errors.employment_type?.message} />
            </div>
            <div>
              <label className="block text-xs font-semibold text-slate-600 mb-1.5" htmlFor="closing_date">
                Closing Date
              </label>
              <input {...register('closing_date')} id="closing_date" type="date" className="input-field" />
              <FieldError message={errors.closing_date?.message} />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-600 mb-1.5" htmlFor="status">
              Initial Status
            </label>
            <select {...register('status')} id="status" className="input-field bg-white cursor-pointer">
              <option value="DRAFT">Draft — save for later</option>
              <option value="OPEN">Open — publish immediately</option>
            </select>
            <FieldError message={errors.status?.message} />
          </div>

          <div>
            <label className="block text-xs font-semibold text-slate-600 mb-1.5" htmlFor="description">
              Description <span className="text-red-500">*</span>
            </label>
            <textarea
              {...register('description')}
              id="description"
              rows={6}
              className="input-field resize-y"
              placeholder="Describe the role, responsibilities, and what you're looking for…"
            />
            <FieldError message={errors.description?.message} />
          </div>
        </div>

        {/* Requirements Card */}
        <div className="bg-white rounded-card shadow-md p-6 space-y-4">
          <div className="flex items-center justify-between pb-2 border-b border-slate-100">
            <h2 className="text-sm font-semibold text-slate-700">Requirements</h2>
            <button
              type="button"
              onClick={() => append({ category: 'SKILL', description: '', importance: 'REQUIRED' })}
              className="flex items-center gap-1.5 text-xs font-semibold text-primary hover:bg-primary/5 px-2.5 py-1.5 rounded-lg transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1"
            >
              <Plus className="w-3.5 h-3.5" />
              Add requirement
            </button>
          </div>

          {fields.length === 0 ? (
            <p className="text-sm text-slate-400 text-center py-6">
              No requirements added yet. Click "Add requirement" to start.
            </p>
          ) : (
            <div className="space-y-3">
              {fields.map((field, idx) => (
                <div key={field.id} className="border border-slate-200 rounded-lg p-3 space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-xs font-semibold text-slate-500">Requirement {idx + 1}</span>
                    <button
                      type="button"
                      onClick={() => remove(idx)}
                      className="text-slate-400 hover:text-red-600 transition-colors cursor-pointer focus:outline-none rounded"
                      aria-label="Remove requirement"
                    >
                      <Trash2 className="w-3.5 h-3.5" />
                    </button>
                  </div>

                  <div className="grid grid-cols-2 gap-2">
                    <div>
                      <label className="block text-xs text-slate-500 mb-1">Category</label>
                      <select
                        {...register(`requirements.${idx}.category`)}
                        className="w-full text-sm border border-slate-200 rounded px-2 py-1.5 focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/20 bg-white cursor-pointer"
                      >
                        {CATEGORIES.map((c) => (
                          <option key={c.value} value={c.value}>{c.label}</option>
                        ))}
                      </select>
                    </div>
                    <div>
                      <label className="block text-xs text-slate-500 mb-1">Importance</label>
                      <select
                        {...register(`requirements.${idx}.importance`)}
                        className="w-full text-sm border border-slate-200 rounded px-2 py-1.5 focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/20 bg-white cursor-pointer"
                      >
                        <option value="REQUIRED">Required</option>
                        <option value="PREFERRED">Preferred</option>
                      </select>
                    </div>
                  </div>

                  <div>
                    <label className="block text-xs text-slate-500 mb-1">Description</label>
                    <input
                      {...register(`requirements.${idx}.description`)}
                      className="w-full text-sm border border-slate-200 rounded px-2 py-1.5 focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/20"
                      placeholder="e.g. 3+ years of Go experience"
                    />
                    <FieldError message={errors.requirements?.[idx]?.description?.message} />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex items-center justify-between">
          <button
            type="button"
            onClick={() => navigate('/jobs')}
            className="px-4 py-2 text-sm text-slate-600 hover:text-slate-800 border border-slate-200 rounded-lg hover:bg-slate-50 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-slate-300"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isSubmitting || createJob.isPending}
            className="px-6 py-2 text-sm font-semibold bg-primary text-white rounded-lg hover:bg-primary-700 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {isSubmitting || createJob.isPending ? 'Creating…' : 'Create Job'}
          </button>
        </div>
      </form>
    </div>
  )
}

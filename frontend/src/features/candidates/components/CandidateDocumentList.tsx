import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { FileText, Plus, AlertCircle, Loader2 } from 'lucide-react'
import { documentsApi } from '@/api/documents'
import { getApiError } from '@/api/client'
import { toast } from 'sonner'
import { CandidateCVUpload, CVUploadStatus } from './CandidateCVUpload'
import { CandidateDocumentCard } from './CandidateDocumentCard'
import type { CandidateDocument } from '@/types'

interface CandidateDocumentListProps {
  candidateId: string
  initialDocuments?: CandidateDocument[]
  onDocumentProcessed?: () => void
  onDocumentDeleted?: () => void
  className?: string
}

export function CandidateDocumentList({
  candidateId,
  initialDocuments,
  onDocumentProcessed,
  onDocumentDeleted,
  className = '',
}: CandidateDocumentListProps) {
  const queryClient = useQueryClient()
  const [isUploadOpen, setIsUploadOpen] = useState(false)
  const [uploadStatus, setUploadStatus] = useState<CVUploadStatus>('idle')
  const [uploadStatusMessage, setUploadStatusMessage] = useState<string>('')
  const [uploadError, setUploadError] = useState<string | undefined>()
  const [documentToDelete, setDocumentToDelete] = useState<string | null>(null)
  const [processingDocId, setProcessingDocId] = useState<string | null>(null)

  // Fetch documents for this candidate
  const { data: documents = initialDocuments || [], isLoading } = useQuery({
    queryKey: ['candidateDocuments', candidateId],
    queryFn: () => documentsApi.getCandidateDocuments(candidateId),
    enabled: !!candidateId,
    initialData: initialDocuments,
  })

  const invalidateQueries = () => {
    queryClient.invalidateQueries({ queryKey: ['candidateDocuments', candidateId] })
    queryClient.invalidateQueries({ queryKey: ['candidate', candidateId] })
    queryClient.invalidateQueries({ queryKey: ['candidateReview'] })
    queryClient.invalidateQueries({ queryKey: ['candidates'] })
    queryClient.invalidateQueries({ queryKey: ['jobCandidates'] })
  }

  // Upload and immediately trigger processing
  const handleUploadAndProcess = async (file: File) => {
    try {
      setUploadError(undefined)
      setUploadStatus('uploading')
      setUploadStatusMessage('Uploading CV...')

      // 1. Upload
      const uploadedDoc = await documentsApi.uploadCandidateDocument(candidateId, file)

      // 2. Process
      setUploadStatus('processing')
      setUploadStatusMessage('Processing CV and extracting profile information...')

      await documentsApi.processCandidateDocument(candidateId, uploadedDoc.id)

      setUploadStatus('processed')
      setUploadStatusMessage('CV uploaded and processed successfully!')
      toast.success('CV uploaded and processed successfully')

      invalidateQueries()
      onDocumentProcessed?.()

      setTimeout(() => {
        setIsUploadOpen(false)
        setUploadStatus('idle')
        setUploadStatusMessage('')
      }, 1500)
    } catch (err) {
      const msg = getApiError(err)
      setUploadStatus('error')
      setUploadError(msg)
      toast.error(msg)
    }
  }

  // Process existing document
  const processMutation = useMutation({
    mutationFn: (docId: string) => documentsApi.processCandidateDocument(candidateId, docId),
    onMutate: (docId) => {
      setProcessingDocId(docId)
    },
    onSuccess: () => {
      toast.success('CV processed and candidate profile updated')
      invalidateQueries()
      onDocumentProcessed?.()
    },
    onError: (err) => {
      toast.error(getApiError(err))
    },
    onSettled: () => {
      setProcessingDocId(null)
    },
  })

  // Delete document mutation
  const deleteMutation = useMutation({
    mutationFn: (docId: string) => documentsApi.deleteCandidateDocument(candidateId, docId),
    onSuccess: () => {
      toast.success('CV document deleted')
      invalidateQueries()
      setDocumentToDelete(null)
      onDocumentDeleted?.()
    },
    onError: (err) => {
      toast.error(getApiError(err))
    },
  })

  return (
    <div className={`space-y-4 ${className}`}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-sm font-bold text-slate-800 uppercase tracking-wide">Documents</h2>
          <p className="text-xs text-slate-500 mt-0.5">CV and resume files</p>
        </div>

        <button
          type="button"
          onClick={() => {
            setIsUploadOpen((prev) => !prev)
            setUploadStatus('idle')
            setUploadError(undefined)
          }}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-lg bg-primary text-white hover:bg-primary-700 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1"
        >
          <Plus className="w-3.5 h-3.5" />
          <span>Upload CV</span>
        </button>
      </div>

      {/* Inline Upload Dropzone when toggled */}
      {isUploadOpen && (
        <div className="bg-slate-50 border border-slate-200 rounded-lg p-4 space-y-3">
          <div className="flex items-center justify-between">
            <span className="text-xs font-semibold text-slate-700">Upload new CV</span>
            <button
              type="button"
              onClick={() => setIsUploadOpen(false)}
              className="text-xs text-slate-400 hover:text-slate-600 cursor-pointer"
            >
              Cancel
            </button>
          </div>

          <CandidateCVUpload
            onUploadAndProcess={handleUploadAndProcess}
            status={uploadStatus}
            statusMessage={uploadStatusMessage}
            errorMessage={uploadError}
            onRetry={() => {
              setUploadStatus('idle')
              setUploadError(undefined)
            }}
          />
        </div>
      )}

      {/* Document List */}
      {isLoading ? (
        <div className="flex items-center justify-center py-8 gap-2 text-xs text-slate-400">
          <Loader2 className="w-4 h-4 animate-spin text-primary" />
          <span>Loading documents...</span>
        </div>
      ) : documents.length === 0 ? (
        <div className="border border-dashed border-slate-200 rounded-lg p-8 text-center bg-slate-50/50">
          <div className="w-10 h-10 rounded-full bg-slate-100 flex items-center justify-center mx-auto mb-2 text-slate-400">
            <FileText className="w-5 h-5" />
          </div>
          <p className="text-sm font-semibold text-slate-700">No CV uploaded yet</p>
          <p className="text-xs text-slate-400 max-w-sm mx-auto mt-1 mb-4">
            Upload a PDF or DOCX resume to automatically extract candidate education, experience, and skills.
          </p>
          <button
            type="button"
            onClick={() => setIsUploadOpen(true)}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-md border border-slate-300 text-slate-700 hover:bg-slate-100 transition-colors cursor-pointer"
          >
            <Plus className="w-3.5 h-3.5" />
            <span>Upload CV</span>
          </button>
        </div>
      ) : (
        <div className="space-y-3">
          {documents.map((doc) => (
            <CandidateDocumentCard
              key={doc.id}
              document={doc}
              onProcess={async (docId) => {
                await processMutation.mutateAsync(docId)
              }}
              onDelete={(docId) => setDocumentToDelete(docId)}
              isProcessing={processingDocId === doc.id}
              isDeleting={deleteMutation.isPending && documentToDelete === doc.id}
            />
          ))}
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      {documentToDelete && (
        <div
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-dialog-title"
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
        >
          <div className="bg-white rounded-lg shadow-xl w-full max-w-sm overflow-hidden flex flex-col">
            <div className="px-5 py-4 border-b border-slate-100 flex items-center gap-2">
              <div className="w-8 h-8 rounded-full bg-red-50 text-red-600 flex items-center justify-center flex-shrink-0">
                <AlertCircle className="w-4 h-4" />
              </div>
              <h3 id="delete-dialog-title" className="text-sm font-bold text-slate-800">
                Delete CV?
              </h3>
            </div>
            <div className="p-5">
              <p className="text-xs text-slate-600 leading-relaxed">
                This will permanently remove this CV document from the candidate profile.
              </p>
            </div>
            <div className="px-5 py-3 bg-slate-50 flex justify-end gap-2 border-t border-slate-100">
              <button
                type="button"
                onClick={() => setDocumentToDelete(null)}
                disabled={deleteMutation.isPending}
                className="px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-200 bg-slate-100 rounded-md transition-colors cursor-pointer focus:outline-none"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={() => deleteMutation.mutate(documentToDelete)}
                disabled={deleteMutation.isPending}
                className="px-3 py-1.5 text-xs font-semibold text-white bg-red-600 hover:bg-red-700 rounded-md transition-colors cursor-pointer focus:outline-none disabled:opacity-50"
              >
                {deleteMutation.isPending ? 'Deleting...' : 'Delete CV'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

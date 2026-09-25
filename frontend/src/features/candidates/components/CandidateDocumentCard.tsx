import { useState } from 'react'
import { FileText, Loader2, Trash2, Cpu, CheckCircle2 } from 'lucide-react'
import { formatDate, formatFileSize, truncate } from '@/lib/utils'
import type { CandidateDocument } from '@/types'

interface CandidateDocumentCardProps {
  document: CandidateDocument
  onProcess: (documentId: string) => Promise<void>
  onDelete: (documentId: string) => void
  isProcessing?: boolean
  isDeleting?: boolean
}

export function CandidateDocumentCard({
  document,
  onProcess,
  onDelete,
  isProcessing = false,
  isDeleting = false,
}: CandidateDocumentCardProps) {
  const [internalProcessing, setInternalProcessing] = useState(false)

  const filename = document.original_filename || document.file_name || 'CV Document'
  const isDocx = filename.toLowerCase().endsWith('.docx') || document.source_type === 'DOCX'
  const fileTypeLabel = isDocx ? 'DOCX' : 'PDF'
  const isProcessed = Boolean(document.raw_text && document.raw_text.trim().length > 0)

  const handleProcessClick = async () => {
    try {
      setInternalProcessing(true)
      await onProcess(document.id)
    } finally {
      setInternalProcessing(false)
    }
  }

  const processingActive = isProcessing || internalProcessing

  return (
    <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-xs hover:border-slate-300 transition-colors">
      <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
        {/* Document Info */}
        <div className="flex items-start gap-3 min-w-0">
          <div className="w-9 h-9 rounded-lg bg-primary/10 text-primary flex items-center justify-center flex-shrink-0 mt-0.5">
            <FileText className="w-5 h-5" />
          </div>
          <div className="min-w-0">
            <p className="text-sm font-semibold text-slate-800 truncate" title={filename}>
              {truncate(filename, 45)}
            </p>
            <div className="flex flex-wrap items-center gap-2 mt-1 text-xs text-slate-500">
              <span className="font-medium text-slate-600">{fileTypeLabel}</span>
              <span>·</span>
              <span>{formatFileSize(document.file_size)}</span>
              <span>·</span>
              <span>Uploaded {formatDate(document.created_at)}</span>
            </div>

            {/* Status indicator */}
            <div className="mt-2 flex items-center gap-1.5 text-xs">
              {processingActive ? (
                <span className="inline-flex items-center gap-1 text-amber-700 bg-amber-50 px-2 py-0.5 rounded border border-amber-200 font-medium">
                  <Loader2 className="w-3 h-3 animate-spin" />
                  Processing CV...
                </span>
              ) : isProcessed ? (
                <span className="inline-flex items-center gap-1 text-green-700 bg-green-50 px-2 py-0.5 rounded border border-green-200 font-medium">
                  <CheckCircle2 className="w-3 h-3" />
                  Processed
                </span>
              ) : (
                <span className="inline-flex items-center gap-1 text-slate-600 bg-slate-100 px-2 py-0.5 rounded border border-slate-200 font-medium">
                  Uploaded
                </span>
              )}
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex items-center gap-2 self-end sm:self-center">
          <button
            type="button"
            onClick={handleProcessClick}
            disabled={processingActive || isDeleting}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 text-xs font-semibold rounded-md border border-slate-200 text-slate-700 hover:bg-slate-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-1"
          >
            {processingActive ? (
              <>
                <Loader2 className="w-3 h-3 animate-spin text-primary" />
                <span>Processing...</span>
              </>
            ) : (
              <>
                <Cpu className="w-3.5 h-3.5 text-slate-500" />
                <span>{isProcessed ? 'Reprocess' : 'Process CV'}</span>
              </>
            )}
          </button>

          <button
            type="button"
            onClick={() => onDelete(document.id)}
            disabled={processingActive || isDeleting}
            className="p-1.5 text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-md transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-red-300 disabled:opacity-50"
            aria-label={`Delete ${filename}`}
            title="Delete CV"
          >
            {isDeleting ? (
              <Loader2 className="w-4 h-4 animate-spin text-red-500" />
            ) : (
              <Trash2 className="w-4 h-4" />
            )}
          </button>
        </div>
      </div>
    </div>
  )
}

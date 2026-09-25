import { useState, useRef, DragEvent, ChangeEvent, KeyboardEvent } from 'react'
import { UploadCloud, FileText, X, AlertCircle, CheckCircle2, Loader2, RefreshCw } from 'lucide-react'
import { formatFileSize } from '@/lib/utils'

const MAX_FILE_SIZE = 20 * 1024 * 1024 // 20 MB
const ALLOWED_EXTENSIONS = ['.pdf', '.docx']
const ACCEPT_STRING = '.pdf,.docx,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document'

export type CVUploadStatus = 'idle' | 'uploading' | 'processing' | 'processed' | 'error'

export interface CandidateCVUploadProps {
  // Mode A: Controlled file selection (e.g. in Candidate Create form)
  selectedFile?: File | null
  onFileSelect?: (file: File | null) => void

  // Mode B: Direct upload & process (e.g. in Candidate Detail or Review modal)
  onUploadAndProcess?: (file: File) => Promise<void>
  status?: CVUploadStatus
  statusMessage?: string
  errorMessage?: string
  onRetry?: () => void
  disabled?: boolean
  className?: string
  hideUploadButton?: boolean
}

function validateCVFile(file: File): { valid: boolean; error?: string } {
  if (!file) {
    return { valid: false, error: 'No file selected.' }
  }

  if (file.size === 0) {
    return { valid: false, error: 'File is empty. Please select a valid CV document.' }
  }

  if (file.size > MAX_FILE_SIZE) {
    return { valid: false, error: 'File is too large. Please upload a file smaller than 20 MB.' }
  }

  const name = file.name.toLowerCase()
  const hasValidExt = ALLOWED_EXTENSIONS.some((ext) => name.endsWith(ext))
  if (!hasValidExt) {
    return { valid: false, error: "This file type isn't supported. Upload a PDF or DOCX." }
  }

  return { valid: true }
}

function formatTruncatedFileName(filename: string, maxBaseLength = 28): string {
  const lastDot = filename.lastIndexOf('.')
  if (lastDot === -1) return filename

  const ext = filename.substring(lastDot)
  const base = filename.substring(0, lastDot)

  if (base.length <= maxBaseLength) return filename
  return `${base.substring(0, maxBaseLength)}...${ext}`
}

export function CandidateCVUpload({
  selectedFile: controlledFile,
  onFileSelect,
  onUploadAndProcess,
  status = 'idle',
  statusMessage,
  errorMessage: externalError,
  onRetry,
  disabled = false,
  className = '',
  hideUploadButton = false,
}: CandidateCVUploadProps) {
  const [internalFile, setInternalFile] = useState<File | null>(null)
  const [validationError, setValidationError] = useState<string | null>(null)
  const [isDragging, setIsDragging] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const activeFile = controlledFile !== undefined ? controlledFile : internalFile

  const handleFileChange = (file: File | null) => {
    setValidationError(null)
    if (!file) {
      if (controlledFile === undefined) setInternalFile(null)
      onFileSelect?.(null)
      return
    }

    const { valid, error } = validateCVFile(file)
    if (!valid) {
      setValidationError(error || 'Invalid file')
      return
    }

    if (controlledFile === undefined) setInternalFile(file)
    onFileSelect?.(file)
  }

  const onInputChange = (e: ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0] || null
    handleFileChange(file)
    // Reset input value so re-selecting same file triggers change
    if (e.target) e.target.value = ''
  }

  const handleDragOver = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    if (disabled || status === 'uploading' || status === 'processing') return
    setIsDragging(true)
  }

  const handleDragLeave = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
  }

  const handleDrop = (e: DragEvent<HTMLDivElement>) => {
    e.preventDefault()
    e.stopPropagation()
    setIsDragging(false)
    if (disabled || status === 'uploading' || status === 'processing') return

    const file = e.dataTransfer.files?.[0] || null
    handleFileChange(file)
  }

  const handleRemove = (e: React.MouseEvent) => {
    e.stopPropagation()
    if (disabled || status === 'uploading' || status === 'processing') return
    handleFileChange(null)
  }

  const handleKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      if (!disabled && status !== 'uploading' && status !== 'processing') {
        fileInputRef.current?.click()
      }
    }
  }

  const isBusy = status === 'uploading' || status === 'processing'
  const displayError = validationError || externalError

  const fileTypeLabel = activeFile?.name.toLowerCase().endsWith('.docx') ? 'DOCX' : 'PDF'

  return (
    <div className={`space-y-3 ${className}`}>
      {/* Hidden native input */}
      <input
        ref={fileInputRef}
        type="file"
        accept={ACCEPT_STRING}
        onChange={onInputChange}
        disabled={disabled || isBusy}
        className="sr-only"
        id="cv-file-upload-input"
        aria-describedby={displayError ? 'cv-upload-error' : 'cv-upload-hint'}
      />

      {/* Upload Zone / Selected File Card */}
      {!activeFile ? (
        <div
          role="button"
          tabIndex={disabled || isBusy ? -1 : 0}
          onClick={() => !disabled && !isBusy && fileInputRef.current?.click()}
          onKeyDown={handleKeyDown}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
          aria-label="Upload CV document"
          className={`border-2 border-dashed rounded-lg p-6 text-center transition-all cursor-pointer select-none focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2 ${
            isDragging
              ? 'border-primary bg-primary/5 text-primary'
              : 'border-slate-200 hover:border-slate-300 bg-slate-50/50 hover:bg-slate-50 text-slate-600'
          } ${disabled || isBusy ? 'opacity-50 cursor-not-allowed' : ''}`}
        >
          <div className="flex flex-col items-center justify-center gap-2">
            <div className="w-10 h-10 rounded-full bg-white shadow-xs border border-slate-200 flex items-center justify-center text-slate-500">
              <UploadCloud className="w-5 h-5 text-primary" />
            </div>
            <div>
              <p className="text-sm font-semibold text-slate-700">
                {isDragging ? 'Drop CV here' : 'Drag & drop your CV here'}
              </p>
              <p className="text-xs text-slate-500 mt-0.5">
                or <span className="text-primary font-medium hover:underline">browse files</span>
              </p>
            </div>
            <p id="cv-upload-hint" className="text-[11px] text-slate-400 mt-1">
              PDF or DOCX · Maximum file size: 20 MB
            </p>
          </div>
        </div>
      ) : (
        <div className="bg-white border border-slate-200 rounded-lg p-4 shadow-xs">
          <div className="flex items-start justify-between gap-3">
            <div className="flex items-center gap-3 min-w-0">
              <div className="w-9 h-9 rounded-lg bg-primary/10 flex items-center justify-center flex-shrink-0 text-primary">
                <FileText className="w-5 h-5" />
              </div>
              <div className="min-w-0">
                <p className="text-sm font-semibold text-slate-800 truncate" title={activeFile.name}>
                  {formatTruncatedFileName(activeFile.name)}
                </p>
                <p className="text-xs text-slate-500 mt-0.5">
                  <span className="font-medium text-slate-600">{fileTypeLabel}</span> · {formatFileSize(activeFile.size)}
                </p>
              </div>
            </div>

            {!isBusy && status !== 'processed' && (
              <button
                type="button"
                onClick={handleRemove}
                disabled={disabled}
                className="text-slate-400 hover:text-slate-600 p-1 rounded-md hover:bg-slate-100 transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-slate-300"
                aria-label="Remove selected CV file"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>

          {/* Status feedback indicator */}
          <div className="mt-3 pt-3 border-t border-slate-100 flex items-center justify-between text-xs" aria-live="polite">
            <div className="flex items-center gap-1.5">
              {status === 'uploading' && (
                <>
                  <Loader2 className="w-3.5 h-3.5 text-primary animate-spin" />
                  <span className="text-slate-600 font-medium">Uploading...</span>
                </>
              )}
              {status === 'processing' && (
                <>
                  <Loader2 className="w-3.5 h-3.5 text-amber-600 animate-spin" />
                  <span className="text-amber-700 font-medium">Processing CV...</span>
                </>
              )}
              {status === 'processed' && (
                <>
                  <CheckCircle2 className="w-3.5 h-3.5 text-green-600" />
                  <span className="text-green-700 font-medium">CV processed</span>
                </>
              )}
              {status === 'error' && (
                <>
                  <AlertCircle className="w-3.5 h-3.5 text-red-500" />
                  <span className="text-red-600 font-medium">{displayError || 'Processing failed'}</span>
                </>
              )}
              {status === 'idle' && (
                <span className="text-slate-500">Ready to upload</span>
              )}
            </div>

            {/* Actions for standalone mode */}
            {onUploadAndProcess && !hideUploadButton && (
              <div className="flex items-center gap-2">
                {status === 'error' && onRetry && (
                  <button
                    type="button"
                    onClick={onRetry}
                    className="flex items-center gap-1 text-xs text-primary font-semibold hover:underline cursor-pointer"
                  >
                    <RefreshCw className="w-3 h-3" />
                    Retry
                  </button>
                )}
                {status === 'idle' && (
                  <button
                    type="button"
                    onClick={() => onUploadAndProcess(activeFile)}
                    disabled={disabled || isBusy}
                    className="px-3 py-1.5 bg-primary text-white text-xs font-semibold rounded-md hover:bg-primary-700 transition-colors cursor-pointer disabled:opacity-50"
                  >
                    Upload CV
                  </button>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Validation or API Error message */}
      {displayError && status !== 'error' && (
        <div
          id="cv-upload-error"
          role="alert"
          className="flex items-center gap-1.5 text-xs text-red-600 bg-red-50/80 border border-red-200 rounded-md p-2.5"
        >
          <AlertCircle className="w-3.5 h-3.5 flex-shrink-0" />
          <span>{displayError}</span>
        </div>
      )}

      {statusMessage && status !== 'error' && (
        <p className="text-xs text-slate-500">{statusMessage}</p>
      )}
    </div>
  )
}

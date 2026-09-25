import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function formatDate(date: string | undefined | null): string {
  if (!date) return '—'
  return new Intl.DateTimeFormat('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    timeZone: 'Asia/Jakarta',
  }).format(new Date(date))
}

export function formatDateTime(date: string | undefined | null): string {
  if (!date) return '—'
  return new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Jakarta',
  }).format(new Date(date))
}

export function formatTime(date: string | undefined | null): string {
  if (!date) return '—'
  return new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Jakarta',
  }).format(new Date(date))
}

export function formatTimeRange(startStr: string, endStr: string): string {
  if (!startStr) return '—'
  const startDate = new Date(startStr)
  const endDate = new Date(endStr)

  const datePart = new Intl.DateTimeFormat('en-US', {
    month: 'short',
    day: 'numeric',
    timeZone: 'Asia/Jakarta',
  }).format(startDate)

  const startTime = new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Jakarta',
  }).format(startDate)

  const endTime = new Intl.DateTimeFormat('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'Asia/Jakarta',
  }).format(endDate)

  return `${datePart}, ${startTime}–${endTime}`
}

export function formatRelativeDate(date: string): string {
  const now = new Date()
  const d = new Date(date)
  const diffMs = now.getTime() - d.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays === 0) return 'Today'
  if (diffDays === 1) return 'Yesterday'
  if (diffDays < 7) return `${diffDays} days ago`
  if (diffDays < 30) return `${Math.floor(diffDays / 7)} weeks ago`
  if (diffDays < 365) return `${Math.floor(diffDays / 30)} months ago`
  return `${Math.floor(diffDays / 365)} years ago`
}

export function formatEmploymentType(type: string): string {
  const map: Record<string, string> = {
    FULL_TIME: 'Full Time',
    PART_TIME: 'Part Time',
    CONTRACT: 'Contract',
    INTERNSHIP: 'Internship',
    FREELANCE: 'Freelance',
  }
  return map[type] ?? type
}

export function formatJobStatus(status: string): string {
  const map: Record<string, string> = {
    DRAFT: 'Draft',
    OPEN: 'Open',
    CLOSED: 'Closed',
    ARCHIVED: 'Archived',
  }
  return map[status] ?? status
}

export function truncate(str: string, length: number): string {
  if (str.length <= length) return str
  return str.slice(0, length) + '…'
}

export function formatFileSize(bytes?: number | null): string {
  if (bytes == null || isNaN(bytes)) return '—'
  if (bytes < 1024) return `${bytes} B`
  const kb = bytes / 1024
  if (kb < 1024) return `${kb.toFixed(1)} KB`
  const mb = kb / 1024
  return `${mb.toFixed(2)} MB`
}

export function getLocalDateString(dateInput: Date | string): string {
  const d = typeof dateInput === 'string' ? new Date(dateInput) : dateInput
  const formatter = new Intl.DateTimeFormat('en-CA', {
    timeZone: 'Asia/Jakarta',
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })
  return formatter.format(d)
}

export function formatDayHeader(dateStr: string): string {
  const todayStr = getLocalDateString(new Date())
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  const tomorrowStr = getLocalDateString(tomorrow)

  const [y, m, d] = dateStr.split('-').map(Number)
  const dateObj = new Date(y, m - 1, d, 12, 0, 0)

  const formattedDate = new Intl.DateTimeFormat('en-US', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  }).format(dateObj)

  if (dateStr === todayStr) {
    return `TODAY — ${formattedDate}`
  } else if (dateStr === tomorrowStr) {
    return `TOMORROW — ${formattedDate}`
  }
  return formattedDate
}

export function formatMonthYear(d: Date): string {
  return new Intl.DateTimeFormat('en-US', {
    month: 'long',
    year: 'numeric',
  }).format(d)
}


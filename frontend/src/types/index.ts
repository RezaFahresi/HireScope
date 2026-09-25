// ============================================================
// HireScope — TypeScript Types
// Matches backend API contracts exactly.
// ============================================================

// ─── Auth ────────────────────────────────────────────────────

export interface User {
  id: string
  name: string
  email: string
  role: UserRole
  created_at: string
  updated_at: string
}

export type UserRole = 'ADMIN' | 'RECRUITER'

export interface LoginRequest {
  email: string
  password: string
}

export interface LoginResponse {
  access_token: string
  token_type: 'Bearer'
  expires_in: number
  user: User
}

// ─── Jobs ────────────────────────────────────────────────────

export type JobStatus = 'DRAFT' | 'OPEN' | 'CLOSED' | 'ARCHIVED'
export type EmploymentType = 'FULL_TIME' | 'PART_TIME' | 'CONTRACT' | 'INTERNSHIP' | 'FREELANCE'
export type RequirementCategory = 'SKILL' | 'EXPERIENCE' | 'EDUCATION' | 'CERTIFICATION' | 'LANGUAGE' | 'OTHER'
export type RequirementImportance = 'REQUIRED' | 'PREFERRED'

export interface JobRequirement {
  id: string
  job_id: string
  category: RequirementCategory
  description: string
  importance: RequirementImportance
  created_at: string
  updated_at: string
}

export interface Job {
  id: string
  code: string
  title: string
  description: string
  department: string
  location: string
  employment_type: EmploymentType
  status: JobStatus
  closing_date?: string
  created_by: string
  created_at: string
  updated_at: string
  requirements?: JobRequirement[]
}

export interface CreateJobRequest {
  title: string
  description: string
  department: string
  location: string
  employment_type: EmploymentType
  status?: JobStatus
  closing_date?: string
}

export interface UpdateJobRequest {
  title?: string
  description?: string
  department?: string
  location?: string
  employment_type?: EmploymentType
  closing_date?: string
}

export interface UpdateJobStatusRequest {
  status: JobStatus
}

export interface CreateRequirementRequest {
  category: RequirementCategory
  description: string
  importance: RequirementImportance
}

export interface JobListParams {
  page?: number
  limit?: number
  search?: string
  status?: JobStatus
  department?: string
  employment_type?: EmploymentType
}

// ─── Candidates ──────────────────────────────────────────────

export type CvSource = 'FILE' | 'TEXT' | 'NONE'
export type DocumentSourceType = 'TEXT' | 'PDF' | 'DOCX'
export type ScreeningStatus = 'QUALIFIED' | 'REVIEW' | 'NOT_QUALIFIED'
export type WorkflowStatus = 'REVIEW' | 'SHORTLISTED' | 'REJECTED'

export interface CandidateSkill {
  id: string
  candidate_id: string
  skill?: string
  name?: string
  level?: string
  years_of_experience?: number
}

export interface CandidateEducation {
  id: string
  candidate_id: string
  institution: string
  degree: string
  field_of_study: string
  start_date?: string
  end_date?: string
  is_current?: boolean
}

export interface CandidateExperience {
  id: string
  candidate_id: string
  company: string
  position?: string
  title?: string
  location?: string
  employment_type?: string
  description?: string
  start_date?: string
  end_date?: string
  is_current: boolean
}

export interface CandidateDocument {
  id: string
  candidate_id: string
  source_type: DocumentSourceType
  original_filename?: string
  mime_type?: string
  file_size?: number
  storage_path?: string
  raw_text?: string
  created_at: string
  updated_at?: string
  // Optional legacy/alias fallbacks
  cv_source?: string
  file_name?: string
  file_path?: string
  cv_text?: string
  parsed_at?: string
}

export interface CreateCandidateRequest {
  full_name: string
  email?: string
  phone?: string
  location?: string
  headline?: string
  summary?: string
}

export interface ProcessDocumentResult {
  document_id: string
  source_type: string
  extraction: {
    status: string
    characters: number
  }
  parsing: {
    status: string
    fields_detected: string[]
  }
  candidate: Candidate
}

export interface Candidate {
  id: string
  full_name: string
  name?: string
  email?: string
  phone?: string
  location?: string
  headline?: string
  summary?: string
  skills?: CandidateSkill[]
  educations?: CandidateEducation[]
  education?: CandidateEducation[]
  experiences?: CandidateExperience[]
  experience?: CandidateExperience[]
  documents?: CandidateDocument[]
  created_at: string
  updated_at: string
}

export interface CandidateListParams {
  page?: number
  limit?: number
  search?: string
}

// ─── Screening ───────────────────────────────────────────────

export type MatchStatus = 'MATCH' | 'PARTIAL' | 'MISMATCH' | 'UNKNOWN' | 'NOT_APPLICABLE'

export interface ScreeningMatch {
  requirement_id: string
  requirement: string
  category: RequirementCategory
  importance: RequirementImportance
  status: MatchStatus
  evidence: string
  reason: string
  confidence?: string
  source?: string
  matched_value?: string
  // Legacy / fallback fields
  id?: string
  requirement_description?: string
  match_status?: MatchStatus
  match_score?: number
  explanation?: string
}

export interface ScreeningResult {
  id: string
  job_id: string
  candidate_id: string
  status: ScreeningStatus
  required_match_count?: number
  required_partial_count?: number
  required_mismatch_count?: number
  required_unknown_count?: number
  preferred_match_count?: number
  preferred_partial_count?: number
  preferred_mismatch_count?: number
  preferred_unknown_count?: number
  evaluated_at: string
  // Legacy / fallback fields
  screening_status?: ScreeningStatus
  overall_score?: number
  required_score?: number
  preferred_score?: number
  summary?: string
  matches?: ScreeningMatch[]
  screened_at?: string
}

export interface ScreeningResponse {
  screening_result: ScreeningResult
  matches: ScreeningMatch[]
}

// ─── Recruiter Workflow ──────────────────────────────────────

export interface UserSummary {
  id: string
  name: string
  email: string
  role?: string
}

export interface DocumentSummary {
  id: string
  source_type?: string
  cv_source?: string
  file_name?: string
  original_filename?: string
  mime_type?: string
  file_size?: number
  created_at: string
}

export interface JobCandidate {
  id: string
  full_name?: string
  email?: string
  phone?: string
  location?: string
  headline?: string
  candidate?: {
    id: string
    full_name: string
    name?: string
    email?: string
    phone?: string
    location?: string
    headline?: string
  }
  status: WorkflowStatus
  workflow_status?: WorkflowStatus
  reviewed_at?: string
  reviewed_by?: UserSummary
  screening?: {
    status: ScreeningStatus
    required: { match: number; partial: number; mismatch: number; unknown: number }
    preferred: { match: number; partial: number; mismatch: number; unknown: number }
    evaluated_at: string
  }
  screening_result?: ScreeningResult
  // Legacy / fallback fields
  job_id?: string
  candidate_id?: string
  assigned_at?: string
  updated_at?: string
}

export interface ReviewDetailResponse {
  job: {
    id: string
    title: string
    status: JobStatus
    department: string
    location: string
    employment_type: EmploymentType
  }
  candidate: {
    id: string
    full_name: string
    name?: string
    email?: string
    phone?: string
    location?: string
    headline?: string
    summary?: string
  }
  workflow: {
    status: WorkflowStatus
    reviewed_at?: string
    reviewed_by?: UserSummary
  }
  documents: DocumentSummary[]
  education: CandidateEducation[]
  experience: CandidateExperience[]
  skills: CandidateSkill[]
  screening?: ScreeningResponse
  notes: CandidateNote[]
}

export interface CandidateNote {
  id: string
  job_id?: string
  candidate_id?: string
  job_candidate_id?: string
  author?: UserSummary
  author_id?: string
  content: string
  created_at: string
  updated_at: string
}

// ─── API Response Wrappers ───────────────────────────────────

export interface ApiSuccess<T> {
  success: true
  data: T
}

export interface ApiError {
  error: {
    code: string
    message: string
  }
}

export interface PaginatedData<T> {
  items: T[]
  total: number
  page: number
  limit: number
  total_pages: number
}

export interface PaginatedResponse<T> {
  data: PaginatedData<T>
}

// ─── Dashboard Intelligence (Step 11) ─────────────────────────

export interface DashboardSummary {
  active_jobs: number
  total_candidates: number
  review_candidates: number
  shortlisted_candidates: number
  rejected_candidates: number
  job_status: {
    draft: number
    open: number
    closed: number
    archived: number
  }
  pipeline: {
    review: number
    shortlisted: number
    rejected: number
  }
  needs_attention: {
    awaiting_review_count: number
    open_jobs_no_candidates: number
  }
}

export interface DashboardRecentCandidate {
  id: string
  candidate_id: string
  full_name: string
  email?: string
  job_id?: string
  job_title: string
  status: string
  assigned_at: string
}

export interface DashboardOpenJob {
  id: string
  code: string
  title: string
  department: string
  location: string
  employment_type: EmploymentType
  status: JobStatus
  candidates_count: number
  updated_at: string
}

export interface DashboardActivity {
  id: string
  action: string
  actor_name: string
  job_id?: string
  job_title?: string
  candidate_id?: string
  candidate_name?: string
  metadata?: string
  created_at: string
}

// ─── Interviews (Step 12A) ────────────────────────────────────

export type InterviewStage =
  | 'PHONE_SCREEN'
  | 'HR_INTERVIEW'
  | 'TECHNICAL_INTERVIEW'
  | 'MANAGER_INTERVIEW'
  | 'FINAL_INTERVIEW'
  | 'OTHER'

export type InterviewType = 'ONLINE' | 'ONSITE' | 'PHONE'
export type InterviewStatus = 'SCHEDULED' | 'COMPLETED' | 'CANCELLED'
export type InterviewResult = 'PASSED' | 'FAILED' | 'ON_HOLD' | 'NO_SHOW'

export interface Interviewer {
  id: string
  name: string
  email: string
  role: string
}

export interface InterviewJob {
  id: string
  code: string
  title: string
  department: string
  location: string
  employment_type: EmploymentType
  status: JobStatus
}

export interface InterviewCandidate {
  id: string
  full_name: string
  email?: string
  phone?: string
  location?: string
  headline?: string
}

export interface Interview {
  id: string
  job_id: string
  candidate_id: string
  title: string
  stage: InterviewStage
  stage_label: string
  interview_type: InterviewType
  scheduled_start: string
  scheduled_end: string
  timezone: string
  meeting_url?: string
  location?: string
  notes?: string
  status: InterviewStatus
  created_by: string
  completed_at?: string
  completed_by?: string
  result?: InterviewResult
  feedback?: string
  cancelled_at?: string
  cancelled_by?: string
  cancellation_reason?: string
  created_at: string
  updated_at: string
  job?: InterviewJob
  candidate?: InterviewCandidate
  creator_name?: string
  completed_name?: string
  cancelled_name?: string
  interviewers: Interviewer[]
}

export interface InterviewListParams {
  job_id?: string
  candidate_id?: string
  status?: InterviewStatus | ''
  stage?: InterviewStage | ''
  interview_type?: InterviewType | ''
  from?: string
  to?: string
  page?: number
  limit?: number
  sort?: string
  order?: string
}

export interface CreateInterviewRequest {
  job_id: string
  candidate_id: string
  title: string
  stage: InterviewStage
  interview_type: InterviewType
  scheduled_start: string
  scheduled_end: string
  timezone?: string
  meeting_url?: string
  location?: string
  notes?: string
  interviewer_ids: string[]
  send_candidate_invitation?: boolean
  send_interviewer_invitation?: boolean
}

export interface UpdateInterviewRequest {
  title: string
  stage: InterviewStage
  interview_type: InterviewType
  scheduled_start: string
  scheduled_end: string
  timezone?: string
  meeting_url?: string
  location?: string
  notes?: string
  interviewer_ids: string[]
}

export interface RescheduleInterviewRequest {
  scheduled_start: string
  scheduled_end: string
  reason?: string
}

export interface CancelInterviewRequest {
  reason: string
}

export interface CompleteInterviewRequest {
  result: InterviewResult
  feedback?: string
}

// ─── Interview Email Deliveries (Step 12B-2) ───────────────────────────

export type RecipientType = 'CANDIDATE' | 'INTERVIEWER'
export type EmailType = 'INTERVIEW_INVITATION' | 'INTERVIEW_REMINDER'
export type DeliveryStatus = 'PENDING' | 'SENDING' | 'SENT' | 'FAILED' | 'SKIPPED'

export interface InterviewEmailDelivery {
  id: string
  interview_id: string
  recipient_type: RecipientType
  recipient_user_id?: string
  recipient_candidate_id?: string
  recipient_email: string
  recipient_name: string
  email_type: EmailType
  status: DeliveryStatus
  provider: string
  provider_message_id?: string
  idempotency_key: string
  attempt_count: number
  last_error?: string
  sent_at?: string
  scheduled_for?: string
  created_at: string
  updated_at: string
}

export interface SendInvitationRequest {
  send_candidate: boolean
  send_interviewers: boolean
}

// ─── External Calendar Integrations (Step 12B-3) ───────────────────────

export type CalendarProviderType = 'GOOGLE' | 'MICROSOFT'
export type CalendarConnectionStatus =
  | 'CONNECTED'
  | 'EXPIRED'
  | 'REVOKED'
  | 'ERROR'
  | 'DISCONNECTED'
  | 'NOT_CONFIGURED'

export type CalendarSyncStatus =
  | 'PENDING'
  | 'SYNCED'
  | 'FAILED'
  | 'DELETED'
  | 'SKIPPED'

export interface CalendarConnectionDTO {
  id?: string
  provider: CalendarProviderType
  provider_email?: string
  status: CalendarConnectionStatus
  is_configured: boolean
  connected_at?: string
  updated_at?: string
}

export interface InterviewCalendarEventDTO {
  id: string
  interview_id: string
  provider: CalendarProviderType
  external_event_id: string
  external_calendar_id?: string
  sync_status: CalendarSyncStatus
  last_synced_at?: string
  last_error?: string
  created_at: string
}



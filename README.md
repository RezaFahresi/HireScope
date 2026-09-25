# HireScope

**Candidate Screening & Recruitment Intelligence Platform**

A modern, transparent, and enterprise-grade recruitment platform designed to assist recruiters in evaluating candidates against specific job requirements with verifiable evidence.

---

## Architecture Overview

HireScope follows a strict layered architecture:

```text
Handler (HTTP / Gin)
   ↓
Service (Business Logic)
   ↓
Repository (Data Access)
   ↓
Database (PostgreSQL 18 / GORM)
```

Authentication and authorization follow an isolated service and middleware flow:

```text
HTTP Request (Bearer Token)
   ↓
AuthMiddleware
   ↓
JWTService (validation & algorithm enforcement via github.com/golang-jwt/jwt/v5)
   ↓
RevokedTokenRepository (server-side logout check)
   ↓
RoleMiddleware (ADMIN / RECRUITER authorization check)
   ↓
Handler
```

### Future Unified CV Input Pipeline (Section 20A)
For future Candidate and CV stages, HireScope is designed to support multiple input sources (`PDF`, `DOCX`, and `TEXT`), converging into a single unified downstream processing pipeline:

```text
PDF ────────┐
            │
DOCX ───────┼──→ Raw CV Text
            │          │
Pasted Text ┘          ↓
                 Text Normalization
                        ↓
                    CV Parser
                        ↓
              Structured Candidate Profile
                        ↓
                Screening Engine
```

- **File Sources (`PDF`, `DOCX`)**: Extracted into raw text via document text extractors.
- **Pasted Text (`TEXT`)**: Stored directly as safe raw CV text without requiring a physical dummy file or fake paths.
- **Unified Downstream Processing**: Both converge at the normalization and structured parser stage.

---

## Prerequisites

- **Go**: 1.22+ (tested with Go 1.26.0 / 1.25.0)
- **PostgreSQL**: PostgreSQL 18
- **Node.js**: v20+ / npm v10+ (for frontend)

---

## PostgreSQL Setup

1. Verify PostgreSQL service is running on your machine:
   ```powershell
   Get-Service *postgres*
   ```

2. Ensure the database `hirescope` exists:
   ```bash
   psql -U postgres -c "CREATE DATABASE hirescope;"
   ```

---

## Backend Setup

1. Navigate to the backend directory:
   ```bash
   cd hirescope/backend
   ```

2. Copy the environment template:
   ```bash
   cp .env.example .env
   ```

3. Configure your local environment variables in `.env`:
   ```env
   APP_ENV=development
   APP_PORT=8080

   DB_HOST=localhost
   DB_PORT=5432
   DB_NAME=hirescope
   DB_USER=postgres
   DB_PASSWORD=
   DB_SSLMODE=disable

   JWT_SECRET=your_secure_random_jwt_secret_here_at_least_32_bytes
   JWT_EXPIRATION_HOURS=24

   UPLOAD_DIR=storage/uploads
   ```

4. Install dependencies:
   ```bash
   go mod download
   ```

---

## Environment Variables Reference

| Variable | Description | Default / Example | Required |
| :--- | :--- | :--- | :--- |
| `APP_ENV` | Application environment (`development`, `production`) | `development` | No |
| `APP_PORT` | Port for the HTTP server | `8080` | No |
| `DB_HOST` | PostgreSQL host address | `localhost` | **Yes** |
| `DB_PORT` | PostgreSQL port | `5432` | **Yes** |
| `DB_NAME` | PostgreSQL database name | `hirescope` | **Yes** |
| `DB_USER` | PostgreSQL user | `postgres` | **Yes** |
| `DB_PASSWORD` | PostgreSQL user password | (empty if trust/local) | No |
| `DB_SSLMODE` | SSL mode (`disable`, `require`, etc.) | `disable` | No |
| `JWT_SECRET` | Secret key for HMAC-SHA256 JWT signing | - | **Yes** |
| `JWT_EXPIRATION_HOURS` | Expiration window for JWT tokens in hours | `24` | No (default: 24) |
| `UPLOAD_DIR` | Directory to store uploaded candidate documents | `storage/uploads` | In upload step |

*Note: Missing required database environment variables or `JWT_SECRET` will cause the backend to immediately fail at startup with a clear configuration validation error.*

---

## Authentication & Security

### Real JWT Implementation
HireScope uses the official, standard Go JWT library:
- **Library**: `github.com/golang-jwt/jwt/v5`
- **Signing Algorithm**: `HS256` (HMAC with SHA-256)
- **Token Format**: Standard RFC 7519 three-segment compact serialization (`header.payload.signature`)

### JWT Claims
- `sub`: Authenticated User ID (UUID)
- `email`: Normalized user email address
- `role`: Role enum (`ADMIN` or `RECRUITER`)
- `iss`: Issuer (`hirescope`)
- `iat`: Issued-at Unix timestamp
- `exp`: Expiration Unix timestamp

*Security note: Passwords, password hashes, and sensitive application secrets are never stored in JWT claims.*

### JWT Validation & Algorithm Enforcement
The `AuthMiddleware` verifies every incoming token using `github.com/golang-jwt/jwt/v5`:
1. Checks that the Authorization header matches `Bearer <token>`.
2. Validates that the token has not been revoked on logout.
3. Explicitly verifies that `token.Method.Alg() == "HS256"`. Tokens specifying `alg: none`, `RS256`, or any unexpected algorithm are rejected immediately.
4. Verifies the cryptographic HMAC signature against `JWT_SECRET`.
5. Verifies token expiration (`exp`) and active lifetime (`nbf`).
6. Attaches the authenticated user context (`userID`, `userEmail`, `userRole`) to the Gin context.

### Password Security
- Passwords are encrypted using `golang.org/x/crypto/bcrypt` with `bcrypt.DefaultCost`.
- Plaintext passwords are never stored in the database.
- The `PasswordHash` field has the GORM/JSON tag `json:"-"`, ensuring password hashes are never leaked into API responses or logs.

### Server-Side Logout Strategy
Stateless JWT tokens are rendered invalid on logout through a dedicated `revoked_tokens` table in PostgreSQL:
- On `POST /api/v1/auth/logout`, the server computes the SHA-256 hash of the token string and stores it with the token's expiration timestamp.
- On every protected request, `AuthMiddleware` verifies that the incoming token's hash is not present in `revoked_tokens`.
- Expired revocation entries are pruned automatically, maintaining high database efficiency without accumulating stale data.

---

## Default Development Credentials

When started in `APP_ENV=development` with an empty users table, HireScope automatically seeds two default accounts:

| Role | Email | Password |
| :--- | :--- | :--- |
| `ADMIN` | `admin@hirescope.local` | `AdminSecure2026!` |
| `RECRUITER` | `recruiter@hirescope.local` | `RecruiterSecure2026!` |

---

## API Endpoints (v1)

### Health
- `GET /api/v1/health` (Public)

### Authentication
- `POST /api/v1/auth/login` (Public)
  - **Request**:
    ```json
    {
      "email": "recruiter@hirescope.local",
      "password": "RecruiterSecure2026!"
    }
    ```
  - **Response (200 OK)**:
    ```json
    {
      "data": {
        "access_token": "<REAL_SIGNED_JWT>",
        "token_type": "Bearer",
        "expires_in": 86400,
        "user": {
          "id": "c1f7899b-...",
          "name": "Sarah Recruiter",
          "email": "recruiter@hirescope.local",
          "role": "RECRUITER",
          "created_at": "...",
          "updated_at": "..."
        }
      }
    }
    ```
- `POST /api/v1/auth/logout` (Protected: requires Bearer token)
  - **Response (200 OK)**:
    ```json
    {
      "data": {
        "message": "successfully logged out"
      }
    }
    ```
- `GET /api/v1/auth/me` (Protected: requires Bearer token)
  - **Response (200 OK)**:
    ```json
    {
      "data": {
        "id": "c1f7899b-...",
        "name": "Sarah Recruiter",
        "email": "recruiter@hirescope.local",
        "role": "RECRUITER",
        "created_at": "...",
        "updated_at": "..."
      }
    }
    ```

### Jobs
- `POST /api/v1/jobs` (Protected: `ADMIN`, `RECRUITER`)
  - Creates a new vacancy. `created_by` is always derived from authenticated JWT.
- `GET /api/v1/jobs` (Protected: `ADMIN`, `RECRUITER`)
  - Supports query parameters: `page`, `limit`, `search`, `status`, `department`, `employment_type`, `sort`, `order`.
  - Returns paginated list with metadata (`total`, `page`, `limit`, `total_pages`).
- `GET /api/v1/jobs/:id` (Protected: `ADMIN`, `RECRUITER`)
  - Returns job details, creator profile, and requirements list.
- `PUT /api/v1/jobs/:id` (Protected: `ADMIN`, `RECRUITER`)
  - Updates job fields. Enforces valid status transitions.
- `POST /api/v1/jobs/:id/archive` (Protected: `ADMIN`, `RECRUITER`)
  - Sets job status to `ARCHIVED`.

### Job Requirements
- `GET /api/v1/jobs/:id/requirements` (Protected: `ADMIN`, `RECRUITER`)
  - Returns all structured requirements for the specified job.
- `POST /api/v1/jobs/:id/requirements` (Protected: `ADMIN`, `RECRUITER`)
  - Adds a requirement (`category`: `SKILL`, `EXPERIENCE`, `EDUCATION`, `CERTIFICATION`, `LANGUAGE`, `OTHER`; `importance`: `REQUIRED`, `PREFERRED`).
- `PUT /api/v1/jobs/:id/requirements/:requirementId` (Protected: `ADMIN`, `RECRUITER`)
  - Updates requirement. Cross-job tampering is rejected.
- `DELETE /api/v1/jobs/:id/requirements/:requirementId` (Protected: `ADMIN`, `RECRUITER`)
  - Deletes requirement belonging strictly to the specified job.

### Job ↔ Candidate Associations
- `POST /api/v1/jobs/:id/candidates` (Protected: `ADMIN`, `RECRUITER`)
  - Links a candidate to a job. Body: `{"candidate_id": "uuid"}`.
  - Returns `201 Created` on success, `409 Conflict` if candidate is already linked to the job.
- `GET /api/v1/jobs/:id/candidates` (Protected: `ADMIN`, `RECRUITER`)
  - Returns paginated list of candidates linked to this job vacancy (`page`, `limit`).
- `DELETE /api/v1/jobs/:id/candidates/:candidateId` (Protected: `ADMIN`, `RECRUITER`)
  - Unlinks candidate from the job. Does NOT delete the candidate from the system.
- `GET /api/v1/candidates/:id/jobs` (Protected: `ADMIN`, `RECRUITER`)
  - Returns all job vacancies associated with the specified candidate.

### Candidate Management
- `POST /api/v1/candidates` (Protected: `ADMIN`, `RECRUITER`)
  - Creates candidate profile (`full_name`, `email`, `phone`, `location`, `headline`, `summary`).
  - `created_by` is strictly populated from the authenticated JWT token.
- `GET /api/v1/candidates` (Protected: `ADMIN`, `RECRUITER`)
  - Paginated list with search (`search`, `location`, `page`, `limit`, `sort`, `order`).
- `GET /api/v1/candidates/:id` (Protected: `ADMIN`, `RECRUITER`)
  - Returns complete candidate profile including nested educations, work experiences, skills, and documents.
- `PUT /api/v1/candidates/:id` (Protected: `ADMIN`, `RECRUITER`)
  - Updates candidate details. `id` and `created_by` cannot be overwritten.

### Candidate Sub-resources
- **Education**:
  - `POST /api/v1/candidates/:id/educations` — Add education (`institution`, `degree`, `field_of_study`, `start_date`, `end_date`, `description`). Enforces `end_date >= start_date`.
  - `GET /api/v1/candidates/:id/educations` — List educations for candidate.
  - `PUT /api/v1/candidates/:id/educations/:educationId` — Update education record.
  - `DELETE /api/v1/candidates/:id/educations/:educationId` — Delete education record.
- **Experience**:
  - `POST /api/v1/candidates/:id/experiences` — Add work experience (`company`, `position`, `location`, `employment_type`, `start_date`, `end_date`, `is_current`, `description`).
  - `GET /api/v1/candidates/:id/experiences` — List experiences for candidate.
  - `PUT /api/v1/candidates/:id/experiences/:experienceId` — Update experience record.
  - `DELETE /api/v1/candidates/:id/experiences/:experienceId` — Delete experience record.
- **Skills**:
  - `POST /api/v1/candidates/:id/skills` — Add skill. Enforces case-insensitive uniqueness per candidate (`409 Conflict` on duplicate).
  - `GET /api/v1/candidates/:id/skills` — List candidate skills.
  - `PUT /api/v1/candidates/:id/skills/:skillId` — Update candidate skill.
  - `DELETE /api/v1/candidates/:id/skills/:skillId` — Delete candidate skill.
- **Documents / CV Processing (Step 6)**:
  - `POST /api/v1/candidates/:id/documents` — Adds raw CV text (`source_type: "TEXT"`, `raw_text: "..."`, max 500 KB).
  - `POST /api/v1/candidates/:id/documents/upload` — Uploads PDF or DOCX file (`multipart/form-data`, file key: `file`, max 20 MB). Validates extension and file signatures (`%PDF-` for PDF, OpenXML ZIP package structure for DOCX). Saves file securely to isolated storage with UUID filename.
  - `POST /api/v1/candidates/:id/documents/:documentId/process` — Executes extraction, normalization, and deterministic parsing pipeline. Converges PDF, DOCX, and TEXT into standardized UTF-8 text (max 5 MB limit), extracts candidate profile, skills, educations, and work experiences. Enforces idempotency (no duplicate entries) and manual data protection (never overwrites recruiter-edited fields).
  - `GET /api/v1/candidates/:id/documents` — List documents for candidate.
  - `GET /api/v1/candidates/:id/documents/:documentId` — Get document details, metadata, and raw text.
  - `DELETE /api/v1/candidates/:id/documents/:documentId` — Deletes document database record and removes physical file from storage if present.

### Candidate Screening (Step 7)
- `POST /api/v1/jobs/:id/candidates/:candidateId/screen` (Protected: `ADMIN`, `RECRUITER`)
  - Evaluates all job requirements against candidate profile using deterministic evaluators.
  - Enforces relationship security: candidate must be associated with the job.
  - Atomically saves/updates screening result and replaces screening matches in a single transaction.
  - Returns overall screening status (`QUALIFIED`, `REVIEW`, `NOT_QUALIFIED`), aggregated counts, and requirement-by-requirement explainable evidence.
- `GET /api/v1/jobs/:id/candidates/:candidateId/screening` (Protected: `ADMIN`, `RECRUITER`)
  - Retrieves the latest screening result and detailed requirement matches for the candidate and job.
  - Returns structured `404 Not Found` with `code: "SCREENING_NOT_FOUND"` if no screening has been executed yet.

---

## CV Processing & Parsing Pipeline (Step 6)

HireScope implements a deterministic, multi-stage CV processing architecture designed for reliability, security, and reproducibility without external AI dependencies:

```text
       ┌──────────┐       ┌──────────┐       ┌──────────┐
       │   PDF    │       │   DOCX   │       │   TEXT   │
       │ (Upload) │       │ (Upload) │       │ (Pasted) │
       └────┬─────┘       └────┬─────┘       └────┬─────┘
            │                  │                  │
   Signature Check    ZIP Structure Check         │
            │                  │                  │
    PDF Extractor       DOCX Extractor            │
            │                  │                  │
            └──────────┬───────┘                  │
                       ▼                          │
                  Raw CV Text ◄───────────────────┘
                       │
                       ▼
              Text Normalization
           (CRLF→LF, strip non-printables,
             collapse spaces, normalize)
                       │
                       ▼
             Deterministic Parser
          (Regex & Heuristic Extractors:
        Email, Phone, Name, Summary, Skills,
              Education, Experience)
                       │
                       ▼
        Idempotent Profile Enrichment
     (Recruiter Manual Data Protection:
         never overwrites existing data;
         deduplicates skills & history)
```

### Key Components

1. **Storage Abstraction (`internal/storage`)**:
   - `StorageService` interface decoupling business logic from underlying filesystem.
   - `LocalStorageService` generates safe UUID filenames, rejects path traversal (`..` and separator tricks), and ensures secure directory creation.
2. **Document Extractors (`internal/extractor`)**:
   - `PDFTextExtractor`: Robust text extraction using `github.com/ledongthuc/pdf` with panic recovery and extraction guards.
   - `DOCXTextExtractor`: Pure Go XML parser (`archive/zip` + `encoding/xml`) parsing paragraph (`<w:p>`) and table (`<w:tr>`) elements while strictly preventing macro/script execution.
   - `Validator`: Deep file inspection enforcing magic signatures (`%PDF-` for PDFs; OpenXML `[Content_Types].xml` and `word/document.xml` validation for DOCX).
3. **Text Normalizer (`internal/normalizer`)**:
   - Normalizes CRLF/CR to LF, cleans non-printable control characters (except `\n` and `\t`), collapses runs of horizontal whitespace, and reduces excessive blank lines to a maximum of 2.
4. **Deterministic Parser (`internal/parser`)**:
   - Extracts contact info (email, international/domestic phone numbers).
   - Identifies candidate name and headline from introductory sections.
   - Segments CV by standard section headers (`EXPERIENCE`, `EDUCATION`, `SKILLS`, `SUMMARY`, etc.).
   - Parses dates, institutions, degrees, companies, and roles.
   - Tokenizes and extracts technical skills.
5. **Idempotency & Recruiter Protection (`internal/service`)**:
   - Re-processing a document is idempotent and will not duplicate skills, educations, or experiences.
   - Candidate fields entered or edited by recruiters take precedence and are never overwritten by parser output.

---

## Candidate Screening & Requirement Matching (Step 7)

> [!NOTE]
> **Deterministic & Explainable Principle**:
> HireScope Step 7 uses deterministic, explainable requirement matching. It does **not** use AI/LLM, semantic embeddings, machine learning, or candidate ranking. No numeric scores or percentages are computed. Every decision is verifiable with concrete evidence extracted from candidate data.

### Screening Architecture

```text
ScreeningHandler (HTTP / Gin)
      ↓
ScreeningService (Validation & Orchestration)
      ↓
ScreeningEngine
      ↓
Requirement Evaluator Registry
   ├── SkillEvaluator (Exact, aliases, multi-skill breakdown)
   ├── ExperienceEvaluator (Durations, non-overlapping intervals, roles)
   ├── EducationEvaluator (Degree ranks, field matching, word-boundary checks)
   ├── CertificationEvaluator (Explicit certification verification)
   ├── LanguageEvaluator (Explicit language proficiency checks)
   └── GenericEvaluator (Profile & background token matching)
      ↓
ScreeningRepository (PostgreSQL 18 / GORM Transaction)
      ↓
Explainable Response (Screening Result + Verifiable Matches)
```

### Supported Requirement Categories

1. **`SKILL`**:
   - Deterministic matching using `candidate_skill.skill` and `candidate_skill.normalized_skill`.
   - Conservative normalization: lowercase, trim, multi-space collapse.
   - Deterministic aliases: `postgres` ↔ `postgresql`, `golang` ↔ `go`, `js` ↔ `javascript`, `ts` ↔ `typescript`, `reactjs` ↔ `react`, `k8s` ↔ `kubernetes`, `py` ↔ `python`, `cs` ↔ `c#`, `cpp` ↔ `c++`, `aws` ↔ `amazon web services`, `gcp` ↔ `google cloud platform`.
   - **Multi-Skill Requirements** (e.g. `"Go, PostgreSQL, Docker"`):
     - Evaluates each skill individually.
     - All present → `MATCH`.
     - Some present → `PARTIAL` (Evidence: `"Matched: Go, PostgreSQL. Missing: Docker."`).
     - None present → `UNKNOWN`.
2. **`EXPERIENCE`**:
   - Parses duration criteria (e.g. `"3 years experience"`, `"5+ years of experience"`, `"minimum 2 years"`).
   - Merges overlapping candidate employment periods chronologically to prevent double-counting.
   - Supports role/domain targeting (e.g. `"3 years experience as Backend Engineer"`).
   - Candidate duration ≥ requirement → `MATCH`.
   - Candidate has relevant experience but duration < requirement → `PARTIAL`.
   - Qualitative experience requirements (e.g. `"Experience with PostgreSQL"`) match explicit company, role, or description text.
   - Missing or unusable date intervals → `UNKNOWN`.
3. **`EDUCATION`**:
   - Academic degree rank hierarchy:
     - Doctorate / PhD / S3 (`RankDoctorate`)
     - Master's / MSc / S2 / Magister / MBA (`RankMaster`)
     - Bachelor's / BSc / S1 / Sarjana (`RankBachelor`)
     - Diploma / Associate / D3 / D4 (`RankDiploma`)
   - Candidate degree rank < required degree rank → `MISMATCH`.
   - Degree rank matches and field matches → `MATCH`.
   - Degree rank matches but field explicitly conflicts → `MISMATCH`.
   - Candidate has institution but no degree or field → `UNKNOWN`.
4. **`CERTIFICATION`**:
   - Checks candidate skills, profile summary, headline, and experience descriptions for explicit certification claims.
   - Found → `MATCH` (`source: CERTIFICATION`).
   - Not found → `UNKNOWN` (`source: NONE`). Absence is never assumed to be MISMATCH.
5. **`LANGUAGE`**:
   - Checks candidate skills, summary, and headline for explicit language proficiency claims (e.g. `"English — Professional Working Proficiency"`).
   - Found → `MATCH` (`source: LANGUAGE`).
   - Not found → `UNKNOWN` (`source: NONE`). Never inferred from nationality, location, or university.
6. **`OTHER`**:
   - Conservative token/phrase matching against candidate profile headline, summary, skills, and descriptions.
   - Found → `MATCH` (`source: PROFILE` / `EXPERIENCE` / `EDUCATION`).
   - Not found → `UNKNOWN`.

### Status Model

| Status | Meaning |
|---|---|
| **`MATCH`** | Candidate explicitly satisfies the requirement with verifiable evidence. |
| **`PARTIAL`** | Candidate partially satisfies the requirement (e.g. some skills in multi-skill set, or shorter experience duration). |
| **`MISMATCH`** | Candidate information explicitly conflicts with or fails the requirement (e.g. lower degree level, mismatched field of study). |
| **`UNKNOWN`** | Insufficient information in candidate profile. Absence of evidence is **never** collapsed into MISMATCH. |
| **`NOT_APPLICABLE`** | Requirement cannot reasonably be evaluated against candidate data. |

### Overall Screening Status

Derived deterministically from requirement matches:
- **`QUALIFIED`**: All `REQUIRED` requirements are `MATCH`.
- **`REVIEW`**: One or more `REQUIRED` requirements are `PARTIAL` or `UNKNOWN`, and there is no explicit `REQUIRED` `MISMATCH`.
- **`NOT_QUALIFIED`**: At least one `REQUIRED` requirement is `MISMATCH`.

> [!IMPORTANT]
> **Preferred Requirements**: `PREFERRED` requirements are recorded independently with their own counts (`preferred_match_count`, `preferred_partial_count`, `preferred_mismatch_count`, `preferred_unknown_count`). A mismatch or unknown on a `PREFERRED` requirement does **not** disqualify a candidate.

### Idempotency & Database Transactions

- Re-screening a candidate against a job is idempotent: it updates the existing `screening_results` record atomically, clears old matches, inserts fresh matches, and updates `evaluated_at` within a single database transaction.
- Prevents duplicate screening rows or orphaned matches.

---

## Step 8 — Candidate Review & Recruiter Workflow

HireScope empowers human recruiters to make transparent, verifiable decisions. Automated screening (Step 7) acts strictly as a decision-support tool; **the recruiter remains the sole and final decision maker**.

```text
Deterministic Screening (Step 7)
         ↓
Recruiter Review Workspace (Step 8)
  ├── Full Candidate Context (Profile, Docs, Education, Experience, Skills)
  ├── Explainable Requirement-by-Requirement Evidence
  ├── Job-Specific Candidate Notes (Collaboration & History)
  └── Recruiter Manual Decision: [SHORTLIST] or [REJECT]
         ↓
Structured Audit Trail (Privacy-Preserving)
```

### 1. Recruiter as the Final Decision Maker
- The system **never** automatically rejects or shortlists a candidate based on screening results.
- Screening outputs (`QUALIFIED`, `REVIEW`, `NOT_QUALIFIED`) are advisory evidence for recruiters.
- Recruiters can shortlist a candidate who had missing or partial requirements if they find compensating factors, or reject a qualified candidate based on team fit or interview performance.

### 2. Candidate Workflow Status Model
Candidates associated with a job vacancy move through distinct workflow states on `job_candidates`:

| Workflow Status | Description |
|---|---|
| **`REVIEW`** | Default initial state when candidate is linked to a job vacancy. Awaiting recruiter evaluation. |
| **`SHORTLISTED`** | Candidate manually approved by recruiter for interviews or next hiring stages. |
| **`REJECTED`** | Candidate manually rejected by recruiter for this specific vacancy. |

#### Workflow Status Transitions:
- `REVIEW` ➔ `SHORTLISTED`
- `REVIEW` ➔ `REJECTED`
- `SHORTLISTED` ➔ `REVIEW` (re-open review)
- `REJECTED` ➔ `REVIEW` (re-open review)
- Invalid transitions or unauthorized statuses are rejected with `400 Bad Request`.
- Status updates automatically record `reviewed_at` timestamp and `reviewed_by` user ID from authenticated JWT claims.

### 3. Job-Specific Candidate Notes
- Recruiters can add contextual review notes, interview impressions, and evaluation comments.
- **Vacancy Scoping**: Notes belong to a specific `(job_id, candidate_id)` association and are completely isolated across jobs.
- **Permission Guards**: Only the note's original author or an `ADMIN` can update (`PUT`) or delete (`DELETE`) a note. Other recruiters receive `403 Forbidden`.
- **Validation**: Note content must be 1 to 5000 characters (whitespace trimmed). Empty notes are rejected with `400 Bad Request`.
- **Ordering**: Notes are retrieved newest first (`created_at DESC`) with pagination support (`page`, `limit`).

### 4. Recruiter Review Detail Aggregation Endpoint
`GET /api/v1/jobs/:id/candidates/:candidateId/review` returns a unified, consolidated recruiter review workspace:
- **`job`**: Vacancy ID, title, status, department, location, employment type.
- **`candidate`**: Full candidate profile (name, email, phone, location, headline, summary).
- **`workflow`**: Current status (`REVIEW`, `SHORTLISTED`, `REJECTED`), `reviewed_at`, `reviewed_by` (ID, name, email, role).
- **`documents`**: Safe document metadata (`id`, `source_type`, `file_name`, `mime_type`, `file_size`, `created_at`). Internal disk paths and raw binary bytes are never exposed.
- **`education`**: Candidate education entries (institution, degree, field of study, dates).
- **`experience`**: Candidate work experience entries (company, position, description, dates).
- **`skills`**: Candidate skills list.
- **`screening`**: Step 7 deterministic screening result (`status`, required/preferred match counts, timestamp) and requirement-by-requirement matches with verifiable evidence (or `null` if un-screened).
- **`notes`**: Complete chronological list of recruiter notes for this candidate under this job vacancy.

### 5. Enhanced Job Candidate List
`GET /api/v1/jobs/:id/candidates` provides recruiter review views:
- **Filtering**:
  - `status`: Filter by `REVIEW`, `SHORTLISTED`, or `REJECTED`.
  - `search`: Case-insensitive search across candidate full name, email, and phone.
- **Pagination & Sorting**: `page`, `limit`, `sort`, `order`.
- **Items**: Returns candidate identity details, current workflow status, reviewer metadata, and Step 7 screening summary.

### 6. Recruiter Audit Trail
All critical recruiter actions generate an immutable audit log entry in PostgreSQL (`audit_logs` table):
- Actions: `CANDIDATE_STATUS_CHANGED`, `CANDIDATE_NOTE_CREATED`, `CANDIDATE_NOTE_UPDATED`, `CANDIDATE_NOTE_DELETED`, `INTERVIEW_SCHEDULED`, `INTERVIEW_RESCHEDULED`, `INTERVIEW_CANCELLED`, `INTERVIEW_COMPLETED`, `CALENDAR_SYNCED`, `REPORT_EXPORTED`.
- Attributes: `id`, `action`, `actor_id`, `job_id`, `candidate_id`, `metadata` (JSON), `created_at`.
- **Privacy Assurance**: Audit log metadata records only structural identifiers (status transitions, note IDs, affected entities). No PII, passwords, JWT tokens, note contents, or raw CV text are recorded in audit logs.

---

## Capabilities & Feature Modules (Steps 1–14)

### 1. Core Platform & Security
- **Authentication**: JWT token-based auth with server-side revocation on logout (`revoked_tokens`).
- **Security Headers**: Standard headers injected on every response (`X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`, `X-XSS-Protection: 1; mode=block`, `Permissions-Policy`).
- **Rate Limiting**: In-memory sliding-window IP rate limiter on sensitive endpoints (e.g. `POST /api/v1/auth/login` capped to 20 req/min).
- **Strict Data Validation & SQL Injection Prevention**: Parameterized queries via GORM, strict field allowlists for sorting/ordering, bounded pagination (limit capped at 100), and CSV formula injection neutralization (`=`, `+`, `-`, `@`, `\t`, `\r` escaped).
- **Zero AI / Deterministic Guarantee**: Pure deterministic pattern matching for CV extraction, scoring, and requirement screening. No external AI APIs, LLMs, or non-deterministic algorithms.

### 2. Job Vacancies & Requirements (Step 3)
- Job lifecycle state machine: `DRAFT` ➔ `OPEN` ➔ `CLOSED` ➔ `ARCHIVED`.
- Configurable deterministic requirements: Experience years, degrees, required skills, and certifications with weights and must-have flags.

### 3. Candidate Profile & CV Extraction (Steps 4, 5, 10)
- Support for `PDF`, `DOCX`, and pasted `TEXT` formats.
- Magic byte validation for uploads (max 20MB), filename sanitization, and isolated storage.
- Deterministic extraction for education, work experience, and tech skills.

### 4. Deterministic Screening & Recruiter Review (Steps 6, 7, 8, 9)
- Deterministic match scoring against job requirements with clear evidence snippets.
- Recruiter workflow stages: `APPLIED` ➔ `SCREENING` ➔ `REVIEW` ➔ `INTERVIEW` ➔ `OFFER` ➔ `HIRED` / `REJECTED`.
- Recruiter notes with rich history and audit trail.

### 5. Recruitment Operations Dashboard (Step 11)
- Real-time pipeline KPI metrics, recent candidates, active jobs, and activity timeline.
- TanStack Query auto-refresh with manual sync and stale-while-revalidate caching.

### 6. Interview Management & Operations (Step 12A, 12B-1, 12B-2, 12B-3)
- **Lifecycle Management**: Schedule, reschedule, cancel, and complete interviews across stages (`PHONE_SCREEN`, `HR_INTERVIEW`, `TECHNICAL_INTERVIEW`, `MANAGER_INTERVIEW`, `FINAL_INTERVIEW`).
- **Calendar & Agenda View**: Month, week, day, and agenda views with stage and status filters.
- **ICS Calendar Export**: Standard RFC 5545 `.ics` export with candidate and interviewer details.
- **Email Invitations & Reminders (Step 12B-2)**: Modular email notification service with Resend integration and audit log delivery records.
- **External Calendar Sync (Step 12B-3)**: Two-way sync state for Google Calendar and Microsoft Outlook 365, with encrypted OAuth token storage (AES-GCM-256).

### 7. Recruitment Analytics & Reporting (Step 13)
- Comprehensive analytics overview: Time-to-hire, pipeline conversion rates, stage drop-offs, recruiter productivity, and department breakdown.
- Export to sanitized CSV with audit logging.

---

## Job Lifecycle & Status Transitions

```text
       ┌──────────┐
       │  DRAFT   │
       └────┬─────┘
            │
            ├───────────────────────┐
            ▼                       ▼
       ┌──────────┐            ┌──────────┐
       │   OPEN   │◄───────────┤  CLOSED  │
       └────┬─────┘            └────┬─────┘
            │                       │
            ├───────────────┬───────┘
            ▼               ▼
       ┌──────────┐   ┌──────────┐
       │  CLOSED  │   │ ARCHIVED │ (Terminal)
       └──────────┘   └──────────┘
```

---

## How to Run Tests & Database Test Safety

### Database Test Safety Guarantee
All unit and integration tests run strictly against in-memory mock repositories and state engines. Tests **never connect to, drop, truncate, or alter** the live PostgreSQL database.

```powershell
# Run all backend tests (using isolated temp dir on Windows)
cd hirescope/backend
$env:GOTMPDIR = "C:\Users\fahre\hirescope\backend\tmp"
go test -v -count=1 ./...

# Run static analysis
go vet ./...
```

### Frontend Lint & Build
```powershell
cd hirescope/frontend
cmd.exe /c "npm run lint"
cmd.exe /c "npm run build"
```

---

## How to Run the Application

### 1. Backend Server
```powershell
cd hirescope/backend
go run cmd/api/main.go
# Server listens on http://localhost:8080
```

### 2. Frontend Application
```powershell
cd hirescope/frontend
npm run dev
# Frontend runs on http://localhost:5173
```

### Default Development Credentials
- **Admin**: `admin@hirescope.local` / `AdminSecure2026!`
- **Recruiter**: `recruiter@hirescope.local` / `RecruiterSecure2026!`

---

## Production Deployment Guide

### 1. Database Deployment & Migration Procedure
HireScope uses GORM automated schema migrations on startup.
1. Provision a production PostgreSQL 18 instance (e.g., AWS RDS, Supabase, Neon, or self-hosted).
2. Create the production database:
   ```sql
   CREATE DATABASE hirescope;
   ```
3. Set the database connection environment variables (`DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_SSLMODE=require`).
4. On backend service launch, database migrations execute automatically before the HTTP listener binds to the port.

### 2. Backend Production Environment Variables
Set the following environment variables in your production container / hosting environment:

| Variable | Description | Example / Recommended Value |
|---|---|---|
| `APP_ENV` | Application environment mode | `production` (disables debug logging and dev seeds) |
| `APP_PORT` | HTTP port for the Gin web service | `8080` |
| `DB_HOST` | Production PostgreSQL host | `postgres.internal` |
| `DB_PORT` | Production PostgreSQL port | `5432` |
| `DB_NAME` | PostgreSQL database name | `hirescope` |
| `DB_USER` | Database username | `hirescope_user` |
| `DB_PASSWORD` | Strong database user password | `(generated high-entropy secret)` |
| `DB_SSLMODE` | SSL connection mode | `require` |
| `JWT_SECRET` | Cryptographic secret for signing JWTs | `(at least 32-character random string)` |
| `JWT_EXPIRATION_HOURS` | Token validity duration | `24` |
| `CV_STORAGE_DIR` | Filesystem path for uploaded CV documents | `/var/data/hirescope/storage` |
| `CALENDAR_TOKEN_ENCRYPTION_KEY` | 32-byte hex key for AES-256-GCM OAuth token encryption | `(64-char hex string)` |
| `EMAIL_ENABLED` | Toggle email delivery worker | `true` or `false` |
| `EMAIL_PROVIDER` | Email provider integration | `resend` |
| `RESEND_API_KEY` | Resend API key for outbound email | `re_...` |
| `EMAIL_FROM_ADDRESS` | Verified sending email address | `recruitment@yourdomain.com` |
| `EMAIL_BASE_URL` | Production frontend domain for email links | `https://hirescope.yourdomain.com` |

### 3. Production CORS Configuration
- In `backend/cmd/api/main.go`, `corsConfig.AllowOrigins` specifies permitted client origins.
- In production, configure this to match your production domain:
  ```go
  corsConfig.AllowOrigins = []string{"https://hirescope.yourdomain.com"}
  ```

### 4. Frontend Production Build & API URL Configuration
1. Configure `frontend/.env.production`:
   ```env
   VITE_API_BASE_URL=https://api.hirescope.yourdomain.com/api/v1
   ```
2. Build optimized static assets:
   ```bash
   cd hirescope/frontend
   npm run build
   ```
3. Deploy the resulting `frontend/dist/` directory to your static web host (Nginx, Caddy, Cloudflare Pages, AWS S3/CloudFront, or Vercel). Ensure the server is configured for Single Page Application (SPA) fallback:
   ```nginx
   location / {
       try_files $uri $uri/ /index.html;
   }
   ```

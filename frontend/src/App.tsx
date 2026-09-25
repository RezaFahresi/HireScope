import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Toaster } from 'sonner'
import { AuthProvider } from '@/features/auth/AuthProvider'
import { ProtectedRoute } from '@/features/auth/ProtectedRoute'
import { LoginPage } from '@/features/auth/LoginPage'
import { AppShell } from '@/components/layout/AppShell'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { JobsListPage } from '@/features/jobs/JobsListPage'
import { JobCreatePage } from '@/features/jobs/JobCreatePage'
import { JobDetailPage } from '@/features/jobs/JobDetailPage'
import { JobEditPage } from '@/features/jobs/JobEditPage'
import { CandidatesListPage } from '@/features/candidates/CandidatesListPage'
import { CandidateDetailPage } from '@/features/candidates/CandidateDetailPage'
import { CandidateCreatePage } from '@/features/candidates/CandidateCreatePage'
import { JobCandidatesPage } from '@/features/reviews/JobCandidatesPage'
import { CandidateReviewPage } from '@/features/reviews/CandidateReviewPage'
import { InterviewListPage } from '@/features/interviews/InterviewListPage'
import { CreateInterviewPage } from '@/features/interviews/CreateInterviewPage'
import { InterviewDetailPage } from '@/features/interviews/InterviewDetailPage'
import { CalendarIntegrationsPage } from '@/features/settings/CalendarIntegrationsPage'
import { AnalyticsPage } from '@/features/analytics/AnalyticsPage'

import { ErrorBoundary } from '@/components/ErrorBoundary'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
  },
})

export default function App() {
  return (
    <ErrorBoundary>
      <QueryClientProvider client={queryClient}>
        <AuthProvider>
          <BrowserRouter>
            <Routes>
              {/* Public */}
              <Route path="/login" element={<LoginPage />} />

              {/* Protected app shell */}
              <Route
                path="/"
                element={
                  <ProtectedRoute>
                    <AppShell />
                  </ProtectedRoute>
                }
              >
                <Route index element={<Navigate to="/dashboard" replace />} />
                <Route path="dashboard" element={<DashboardPage />} />
                <Route path="analytics" element={<AnalyticsPage />} />

                {/* Jobs */}
                <Route path="jobs" element={<JobsListPage />} />
                <Route path="jobs/new" element={<JobCreatePage />} />
                <Route path="jobs/:id" element={<JobDetailPage />} />
                <Route path="jobs/:id/edit" element={<JobEditPage />} />
                <Route path="jobs/:id/candidates" element={<JobCandidatesPage />} />
                <Route path="jobs/:id/candidates/:candidateId/review" element={<CandidateReviewPage />} />

                {/* Candidates */}
                <Route path="candidates" element={<CandidatesListPage />} />
                <Route path="candidates/new" element={<CandidateCreatePage />} />
                <Route path="candidates/:id" element={<CandidateDetailPage />} />

                {/* Interviews (Step 12A) */}
                <Route path="interviews" element={<InterviewListPage />} />
                <Route path="interviews/new" element={<CreateInterviewPage />} />
                <Route path="interviews/:id" element={<InterviewDetailPage />} />

                {/* Settings & External Calendar Integrations (Step 12B-3) */}
                <Route path="settings" element={<Navigate to="/settings/integrations" replace />} />
                <Route path="settings/integrations" element={<CalendarIntegrationsPage />} />

                {/* Catch-all */}
                <Route path="*" element={<Navigate to="/dashboard" replace />} />
              </Route>
            </Routes>

            <Toaster
              position="top-right"
              richColors
              toastOptions={{
                duration: 4000,
                style: {
                  fontFamily: '"Fira Sans", sans-serif',
                  fontSize: '13px',
                },
              }}
            />
          </BrowserRouter>
        </AuthProvider>
      </QueryClientProvider>
    </ErrorBoundary>
  )
}

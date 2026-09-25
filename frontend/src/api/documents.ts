import { apiClient } from './client'
import type {
  CandidateDocument,
  ProcessDocumentResult,
  ApiSuccess,
} from '@/types'

export const documentsApi = {
  // Upload a CV document (PDF or DOCX) for a candidate
  uploadCandidateDocument: async (
    candidateId: string,
    file: File
  ): Promise<CandidateDocument> => {
    const formData = new FormData()
    formData.append('file', file)

    // Note: Do NOT set Content-Type header manually; let the browser and Axios
    // generate the multipart/form-data boundary automatically.
    const res = await apiClient.post<ApiSuccess<CandidateDocument>>(
      `/candidates/${candidateId}/documents/upload`,
      formData
    )
    return res.data.data
  },

  // Process and extract structured CV data
  processCandidateDocument: async (
    candidateId: string,
    documentId: string
  ): Promise<ProcessDocumentResult> => {
    const res = await apiClient.post<ApiSuccess<ProcessDocumentResult>>(
      `/candidates/${candidateId}/documents/${documentId}/process`
    )
    return res.data.data
  },

  // List all uploaded CV documents for a candidate
  getCandidateDocuments: async (
    candidateId: string
  ): Promise<CandidateDocument[]> => {
    const res = await apiClient.get<ApiSuccess<CandidateDocument[]>>(
      `/candidates/${candidateId}/documents`
    )
    return res.data.data
  },

  // Get metadata for a specific document
  getCandidateDocument: async (
    candidateId: string,
    documentId: string
  ): Promise<CandidateDocument> => {
    const res = await apiClient.get<ApiSuccess<CandidateDocument>>(
      `/candidates/${candidateId}/documents/${documentId}`
    )
    return res.data.data
  },

  // Permanently delete a CV document
  deleteCandidateDocument: async (
    candidateId: string,
    documentId: string
  ): Promise<void> => {
    await apiClient.delete(
      `/candidates/${candidateId}/documents/${documentId}`
    )
  },
}

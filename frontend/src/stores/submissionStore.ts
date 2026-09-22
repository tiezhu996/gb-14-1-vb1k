// 提交评测状态
import { create } from 'zustand'
import type { Submission } from '../types'
import * as submissionApi from '../api/submission'

interface SubmissionState {
  submissions: Submission[]
  total: number
  loading: boolean
  fetchSubmissions: (params?: { page?: number; page_size?: number; problem_id?: string; status?: string }) => Promise<void>
  fetchSubmission: (id: string) => Promise<Submission>
  submit: (problemId: string, payload: { language: string; code: string }) => Promise<Submission>
  rejudge: (id: string) => Promise<Submission>
}

export const useSubmissionStore = create<SubmissionState>((set) => ({
  submissions: [],
  total: 0,
  loading: false,
  fetchSubmissions: async (params) => {
    set({ loading: true })
    try {
      const data = await submissionApi.listSubmissions(params || {})
      set({ submissions: data.list, total: data.total })
    } finally {
      set({ loading: false })
    }
  },
  fetchSubmission: async (id) => {
    return submissionApi.getSubmission(id)
  },
  submit: async (problemId, payload) => {
    const submission = await submissionApi.submitCode(problemId, payload)
    set((s) => ({ submissions: [submission, ...s.submissions] }))
    return submission
  },
  rejudge: async (id) => {
    const submission = await submissionApi.rejudgeSubmission(id)
    set((s) => ({ submissions: s.submissions.map((it) => (it.id === id ? submission : it)) }))
    return submission
  },
}))

// 提交重判状态：发起重判 + 历次结果历史
import { create } from 'zustand'
import type { RejudgeRecord } from '../types'
import { listRejudges, rejudgeSubmission } from '../api/submission'

interface RejudgeState {
  history: RejudgeRecord[]
  loading: boolean
  acting: boolean
  fetchHistory: (submissionId: string) => Promise<void>
  rejudge: (submissionId: string) => Promise<RejudgeRecord>
  reset: () => void
}

export const useRejudgeStore = create<RejudgeState>((set) => ({
  history: [],
  loading: false,
  acting: false,
  fetchHistory: async (submissionId) => {
    set({ loading: true })
    try {
      const list = await listRejudges(submissionId)
      set({ history: list })
    } finally {
      set({ loading: false })
    }
  },
  rejudge: async (submissionId) => {
    set({ acting: true })
    try {
      const record = await rejudgeSubmission(submissionId)
      // 回读最新历史（刷新后亦可回读，独立记录追加）
      const list = await listRejudges(submissionId)
      set({ history: list })
      return record
    } finally {
      set({ acting: false })
    }
  },
  reset: () => set({ history: [], loading: false, acting: false }),
}))

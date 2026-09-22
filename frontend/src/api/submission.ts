// 提交评测 API
import request from '../utils/request'
import type { PageData, Submission, RejudgeRecord } from '../types'

export function submitCode(problemId: string, payload: { language: string; code: string }) {
  return request.post<unknown, Submission>(`/problems/${problemId}/submit`, payload)
}

export function getSubmission(id: string) {
  return request.get<unknown, Submission>(`/submissions/${id}`)
}

export function listSubmissions(params: { page?: number; page_size?: number; problem_id?: string; status?: string }) {
  return request.get<unknown, PageData<Submission>>('/submissions', { params })
}

// 管理员发起重判（每次生成独立记录，原代码与首次结果不变）
export function rejudgeSubmission(id: string) {
  return request.post<unknown, RejudgeRecord>(`/submissions/${id}/rejudge`)
}

// 查询某条提交的历次重判记录（本人或管理员）
export function listRejudges(id: string) {
  return request.get<unknown, RejudgeRecord[]>(`/submissions/${id}/rejudges`)
}

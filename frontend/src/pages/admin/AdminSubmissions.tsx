// 提交重判管理（管理员）：查看全部提交，按最新状态过滤，对“已完成但未通过”的提交发起重判
import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { listSubmissions, rejudgeSubmission } from '../../api/submission'
import type { Submission } from '../../types'
import StatusBadge from '../../components/StatusBadge'
import Pagination from '../../components/Pagination'
import ConfirmDialog from '../../components/ConfirmDialog'
import { usePagination } from '../../hooks/usePagination'
import { LANGUAGE_LABELS, SUBMISSION_STATUS_LABELS, REJUDGE_ELIGIBLE_STATUSES } from '../../constants'

const STATUS_OPTIONS = [
  { value: '', label: '全部状态' },
  { value: 'accepted', label: '通过' },
  { value: 'partial', label: '部分通过' },
  { value: 'runtime_error', label: '运行错误' },
  { value: 'timeout', label: '超时' },
]

export default function AdminSubmissions() {
  const navigate = useNavigate()
  const { page, pageSize, total, setTotal, onPageChange } = usePagination(10)
  const [submissions, setSubmissions] = useState<Submission[]>([])
  const [loading, setLoading] = useState(false)
  const [status, setStatus] = useState('')
  const [confirm, setConfirm] = useState<Submission | null>(null)
  const [acting, setActing] = useState(false)
  const [msg, setMsg] = useState('')

  const load = (p = page) => {
    setLoading(true)
    listSubmissions({ page: p, page_size: pageSize, status })
      .then((data) => {
        setSubmissions(data.list)
        setTotal(data.total)
      })
      .catch((e) => setMsg((e as Error).message))
      .finally(() => setLoading(false))
  }

  useEffect(() => {
    load(page)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page, pageSize, status])

  const doRejudge = async () => {
    if (!confirm) return
    setActing(true)
    setMsg('')
    try {
      const rec = await rejudgeSubmission(confirm.id)
      setConfirm(null)
      setMsg(
        rec.status === 'accepted'
          ? `重判通过（第 ${rec.round} 次），奖励${rec.rewards_granted ? '已补发' : '此前已补发'}`
          : `第 ${rec.round} 次重判完成：${SUBMISSION_STATUS_LABELS[rec.status] || rec.status}，统计不扣减`,
      )
      load()
    } catch (e) {
      setMsg((e as Error).message)
    } finally {
      setActing(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-800">提交重判管理</h1>
          <p className="mt-1 text-sm text-gray-500">仅对“已完成但未通过”的提交发起重判；每次重判独立记录，原代码与首次结果不变</p>
        </div>
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          className="rounded-lg border border-gray-300 px-3 py-2 text-sm"
        >
          {STATUS_OPTIONS.map((o) => (
            <option key={o.value} value={o.value}>{o.label}</option>
          ))}
        </select>
      </div>

      {msg && <div className="rounded-lg bg-gray-50 px-4 py-2 text-sm text-gray-600">{msg}</div>}

      {loading && submissions.length === 0 ? (
        <div className="py-20 text-center text-gray-400">加载中...</div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">提交人</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">题目</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">语言</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">首次状态</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">最新状态</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">重判次数</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">得分</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">时间</th>
                <th className="px-4 py-3 text-left font-semibold text-gray-600">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {submissions.map((s) => {
                const latest = s.latest_status || s.status
                const eligible = REJUDGE_ELIGIBLE_STATUSES.includes(latest as (typeof REJUDGE_ELIGIBLE_STATUSES)[number])
                return (
                  <tr key={s.id} className="hover:bg-gray-50">
                    <td className="px-4 py-3 text-gray-600">{s.username}</td>
                    <td className="px-4 py-3 font-medium text-gray-800">{s.problem_title}</td>
                    <td className="px-4 py-3 text-gray-600">{LANGUAGE_LABELS[s.language] || s.language}</td>
                    <td className="px-4 py-3"><StatusBadge value={s.status} kind="submission" /></td>
                    <td className="px-4 py-3"><StatusBadge value={latest} kind="submission" /></td>
                    <td className="px-4 py-3">
                      {s.rejudge_count > 0 ? (
                        <span className="rounded-full bg-violet-100 px-2 py-0.5 text-xs font-medium text-violet-700">{s.rejudge_count} 次</span>
                      ) : (
                        <span className="text-xs text-gray-400">0</span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-gray-700">{s.score}%</td>
                    <td className="px-4 py-3 text-gray-500">{s.created_at}</td>
                    <td className="px-4 py-3">
                      <div className="flex gap-2">
                        <button
                          onClick={() => navigate(`/submissions/${s.id}`)}
                          className="rounded bg-gray-100 px-2.5 py-1 text-xs text-gray-700 hover:bg-gray-200"
                        >
                          详情/历史
                        </button>
                        <button
                          onClick={() => setConfirm(s)}
                          disabled={!eligible || acting}
                          title={eligible ? '发起重判' : '仅已完成但未通过可重判'}
                          className="rounded bg-violet-600 px-2.5 py-1 text-xs text-white hover:bg-violet-700 disabled:cursor-not-allowed disabled:opacity-40"
                        >
                          重判
                        </button>
                      </div>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      <Pagination page={page} pageSize={pageSize} total={total} onChange={onPageChange} />

      <ConfirmDialog
        open={!!confirm}
        title="发起重判"
        message={`确定对「${confirm?.problem_title}」（提交人 ${confirm?.username}）发起第 ${(confirm?.rejudge_count ?? 0) + 1} 次重判？原代码与首次评测结果不变，重判通过仅补发一次奖励。`}
        confirmText="发起重判"
        onConfirm={doRejudge}
        onCancel={() => setConfirm(null)}
      />
    </div>
  )
}

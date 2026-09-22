// 提交详情：原代码 + 首次评测结果（不可变）+ 重判次数/最新状态 + 历次重判结果与差异
// 学生只能查看本人记录（后端鉴权），管理员可查看全部并对“已完成但未通过”的提交发起重判
import { useEffect, useState, useCallback } from 'react'
import { Link, useParams } from 'react-router-dom'
import { getSubmission } from '../../api/submission'
import { useRejudgeStore } from '../../stores/rejudgeStore'
import { useAuth } from '../../hooks/useAuth'
import type { Submission } from '../../types'
import StatusBadge from '../../components/StatusBadge'
import CodeBlock from '../../components/CodeBlock'
import JudgeResultsTable from '../../components/JudgeResultsTable'
import ConfirmDialog from '../../components/ConfirmDialog'
import { LANGUAGE_LABELS, SUBMISSION_STATUS_LABELS, formatRejudgeRound, REJUDGE_ELIGIBLE_STATUSES } from '../../constants'

export default function SubmissionDetail() {
  const { id } = useParams()
  const { isAdmin } = useAuth()
  const [submission, setSubmission] = useState<Submission | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [confirmOpen, setConfirmOpen] = useState(false)
  const [actionMsg, setActionMsg] = useState('')
  const { history, loading: historyLoading, acting, fetchHistory, rejudge } = useRejudgeStore()

  const load = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError('')
    try {
      const sub = await getSubmission(id)
      setSubmission(sub)
      if (sub.rejudge_count > 0) {
        await fetchHistory(id)
      }
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [id, fetchHistory])

  useEffect(() => {
    load()
  }, [load])

  const doRejudge = async () => {
    if (!id) return
    setConfirmOpen(false)
    setActionMsg('')
    try {
      const rec = await rejudge(id)
      const sub = await getSubmission(id)
      setSubmission(sub)
      setActionMsg(
        rec.status === 'accepted'
          ? rec.rewards_granted
            ? '重判通过，已补发积分/解决数/通过数'
            : '重判通过（奖励此前已补发，不重复累计）'
          : `重判完成：${SUBMISSION_STATUS_LABELS[rec.status] || rec.status}，未通过不扣减统计`,
      )
    } catch (e) {
      setActionMsg((e as Error).message)
    }
  }

  if (loading) {
    return <div className="py-20 text-center text-gray-400">加载中...</div>
  }
  if (error || !submission) {
    return (
      <div className="space-y-4">
        <div className="py-20 text-center text-rose-500">{error || '提交记录不存在'}</div>
        <div className="text-center">
          <Link to="/submissions" className="text-brand-600 hover:underline">← 返回提交列表</Link>
        </div>
      </div>
    )
  }

  const latestStatus = submission.latest_status || submission.status
  const canRejudge = isAdmin && REJUDGE_ELIGIBLE_STATUSES.includes(latestStatus as (typeof REJUDGE_ELIGIBLE_STATUSES)[number])

  return (
    <div className="space-y-6">
      <div>
        <Link to="/submissions" className="text-sm text-brand-600 hover:underline">← 返回提交列表</Link>
      </div>

      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="text-xl font-bold text-gray-800">{submission.problem_title}</h1>
          <StatusBadge value={latestStatus} kind="submission" />
          {submission.rejudge_count > 0 && (
            <span className="rounded-full bg-violet-100 px-2.5 py-0.5 text-xs font-medium text-violet-700">
              重判 {submission.rejudge_count} 次
            </span>
          )}
          {submission.rewards_granted && (
            <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-700">已补发奖励</span>
          )}
        </div>
        <div className="mt-2 flex flex-wrap gap-4 text-xs text-gray-500">
          <span>提交人：{submission.username}</span>
          <span>语言：{LANGUAGE_LABELS[submission.language] || submission.language}</span>
          <span>提交时间：{submission.created_at}</span>
          <span>首次状态：{SUBMISSION_STATUS_LABELS[submission.status] || submission.status}</span>
        </div>
        <div className="mt-4 flex items-center gap-3">
          {canRejudge && (
            <button
              onClick={() => setConfirmOpen(true)}
              disabled={acting}
              className="rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 disabled:opacity-60"
            >
              {acting ? '重判中...' : '🔁 发起重判'}
            </button>
          )}
          {actionMsg && <span className="text-sm text-gray-600">{actionMsg}</span>}
        </div>
      </div>

      {/* 原代码（只读，重判不改写） */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <h2 className="mb-3 font-semibold text-gray-800">原提交代码（只读，重判不修改）</h2>
        <CodeBlock code={submission.code} language={submission.language} />
      </div>

      {/* 首次评测结果（不可变） */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center gap-3">
          <h2 className="font-semibold text-gray-800">首次评测结果</h2>
          <StatusBadge value={submission.status} kind="submission" />
          <span className="text-sm text-gray-500">
            得分 {submission.score}% · 耗时 {submission.runtime_ms}ms
          </span>
          {submission.points_awarded > 0 && (
            <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-700">
              +{submission.points_awarded} 分
            </span>
          )}
        </div>
        {submission.error_message && (
          <div className="mt-3 rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-700">{submission.error_message}</div>
        )}
        <div className="mt-4">
          <JudgeResultsTable results={submission.results} />
        </div>
      </div>

      {/* 历次重判结果与差异 */}
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex items-center gap-3">
          <h2 className="font-semibold text-gray-800">重判历史</h2>
          <span className="text-sm text-gray-500">共 {submission.rejudge_count} 次</span>
        </div>
        {historyLoading ? (
          <div className="py-8 text-center text-gray-400">加载中...</div>
        ) : history.length === 0 ? (
          <p className="mt-4 text-sm text-gray-400">暂无重判记录。仅管理员可对“已完成但未通过”的提交发起重判。</p>
        ) : (
          <div className="mt-4 space-y-6">
            {history.map((rj) => (
              <div key={rj.id} className="rounded-lg border border-gray-200 p-4">
                <div className="flex flex-wrap items-center gap-3">
                  <span className="rounded-full bg-violet-50 px-2.5 py-0.5 text-xs font-medium text-violet-700">
                    {formatRejudgeRound(rj.round)}
                  </span>
                  <StatusBadge value={rj.status} kind="submission" />
                  <span className="text-sm text-gray-500">
                    得分 {rj.score}% · 耗时 {rj.runtime_ms}ms · {rj.created_at}
                  </span>
                  <span className="text-xs text-gray-400">操作人：{rj.operator_name || '管理员'}</span>
                  {rj.status === 'accepted' && (
                    <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${rj.rewards_granted ? 'bg-amber-100 text-amber-700' : 'bg-gray-100 text-gray-500'}`}>
                      {rj.rewards_granted ? `已补发 +${rj.points_awarded} 分` : '奖励此前已补发'}
                    </span>
                  )}
                </div>

                {/* 与上一轮差异 */}
                <div className="mt-3 rounded-lg bg-gray-50 px-4 py-3 text-xs text-gray-600">
                  <span className="font-semibold text-gray-700">与上一轮差异：</span>
                  {rj.diff.status_changed ? (
                    <span className="ml-1">
                      状态 {SUBMISSION_STATUS_LABELS[rj.diff.from_status] || rj.diff.from_status} →{' '}
                      {SUBMISSION_STATUS_LABELS[rj.diff.to_status] || rj.diff.to_status}
                    </span>
                  ) : (
                    <span className="ml-1">状态无变化</span>
                  )}
                  <span className="ml-3">得分差 {rj.diff.score_delta >= 0 ? `+${rj.diff.score_delta}` : rj.diff.score_delta}</span>
                  <span className="ml-3">
                    耗时差 {rj.diff.runtime_delta_ms >= 0 ? `+${rj.diff.runtime_delta_ms}` : rj.diff.runtime_delta_ms}ms
                  </span>
                  {rj.diff.changed_cases.length > 0 && (
                    <span className="ml-3 text-amber-600">
                      翻转用例：{rj.diff.changed_cases.map((c) => `#${c + 1}`).join('、')}
                    </span>
                  )}
                </div>

                {rj.error_message && (
                  <div className="mt-3 rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-700">{rj.error_message}</div>
                )}
                <div className="mt-3">
                  <JudgeResultsTable results={rj.results} highlightCases={rj.diff.changed_cases} />
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      <ConfirmDialog
        open={confirmOpen}
        title="发起重判"
        message="将使用原提交代码与题目当前测试用例重新评测。原代码和首次评测结果不会被修改，重判通过时仅补发一次积分/解决数/通过数。是否继续？"
        confirmText="发起重判"
        onConfirm={doRejudge}
        onCancel={() => setConfirmOpen(false)}
      />
    </div>
  )
}

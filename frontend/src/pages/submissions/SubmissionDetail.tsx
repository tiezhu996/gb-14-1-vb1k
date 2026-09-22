// 提交详情：首次评测结果 + 历次重判记录及差异（学生仅本人可见，管理员可发起重判）
import { useCallback, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useSubmissionStore } from '../../stores/submissionStore'
import { useAuthStore } from '../../stores/authStore'
import type { JudgeResult, Submission } from '../../types'
import StatusBadge from '../../components/StatusBadge'
import CodeBlock from '../../components/CodeBlock'
import ConfirmDialog from '../../components/ConfirmDialog'
import { LANGUAGE_LABELS, ROLES, SUBMISSION_STATUS_LABELS, canRejudgeSubmission } from '../../constants'

function ResultTable({ results }: { results: JudgeResult[] }) {
  if (!results || results.length === 0) return null
  return (
    <div className="mt-3 overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200 text-sm">
        <thead className="bg-gray-50">
          <tr>
            <th className="px-4 py-2 text-left font-semibold text-gray-600">用例</th>
            <th className="px-4 py-2 text-left font-semibold text-gray-600">输入</th>
            <th className="px-4 py-2 text-left font-semibold text-gray-600">期望输出</th>
            <th className="px-4 py-2 text-left font-semibold text-gray-600">实际输出</th>
            <th className="px-4 py-2 text-left font-semibold text-gray-600">结果</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-gray-100">
          {results.map((r) => (
            <tr key={r.test_case_index}>
              <td className="px-4 py-2">#{r.test_case_index + 1}</td>
              <td className="px-4 py-2 font-mono text-xs">{r.input || '(空)'}</td>
              <td className="px-4 py-2 font-mono text-xs">{r.expected || '(空)'}</td>
              <td className="px-4 py-2 font-mono text-xs">{r.actual || '(空)'}</td>
              <td className="px-4 py-2">
                {r.passed ? <span className="text-emerald-600">✅ 通过</span> : <span className="text-rose-600">❌ 未通过</span>}
                {r.error_message && <span className="ml-1 text-xs text-rose-500">{r.error_message}</span>}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ScoreDelta({ delta }: { delta: number }) {
  if (delta === 0) return <span className="text-gray-400">±0</span>
  if (delta > 0) return <span className="font-medium text-emerald-600">+{delta}</span>
  return <span className="font-medium text-rose-600">{delta}</span>
}

export default function SubmissionDetail() {
  const { id } = useParams()
  const { fetchSubmission, rejudge } = useSubmissionStore()
  const { user } = useAuthStore()
  const [submission, setSubmission] = useState<Submission | null>(null)
  const [error, setError] = useState('')
  const [confirming, setConfirming] = useState(false)
  const [rejudging, setRejudging] = useState(false)

  const load = useCallback(() => {
    if (!id) return
    fetchSubmission(id)
      .then(setSubmission)
      .catch((e) => setError((e as Error).message))
  }, [id, fetchSubmission])

  useEffect(() => {
    load()
  }, [load])

  const doRejudge = async () => {
    if (!id) return
    setConfirming(false)
    setRejudging(true)
    setError('')
    try {
      const updated = await rejudge(id)
      setSubmission(updated)
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setRejudging(false)
    }
  }

  if (!submission) {
    return <div className="py-20 text-center text-gray-400">{error || '加载中...'}</div>
  }

  const isAdmin = user?.role === ROLES.admin
  const latestStatus = submission.latest_status || submission.status
  const latestScore = submission.latest_score ?? submission.score

  return (
    <div className="space-y-6">
      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="mr-2 text-2xl font-bold text-gray-800">提交详情</h1>
          <StatusBadge value={latestStatus} kind="submission" />
          <span className="text-sm text-gray-500">
            最新状态：{SUBMISSION_STATUS_LABELS[latestStatus]} · 得分 {latestScore}% · 重判 {submission.rejudge_count} 次
          </span>
          {submission.rejudge_points_awarded > 0 && (
            <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-700">
              重判补发 +{submission.rejudge_points_awarded} 分
            </span>
          )}
          {isAdmin && canRejudgeSubmission(latestStatus) && (
            <button
              onClick={() => setConfirming(true)}
              disabled={rejudging}
              className="ml-auto rounded-lg bg-violet-600 px-4 py-2 text-sm font-semibold text-white hover:bg-violet-700 disabled:opacity-60"
            >
              {rejudging ? '重判中...' : '🔁 发起重判'}
            </button>
          )}
        </div>
        <div className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm text-gray-500">
          <span>
            题目：
            <Link to={`/problems/${submission.problem_id}`} className="text-brand-600 hover:underline">
              {submission.problem_title}
            </Link>
          </span>
          <span>提交人：{submission.username}</span>
          <span>语言：{LANGUAGE_LABELS[submission.language] || submission.language}</span>
          <span>提交时间：{submission.created_at}</span>
        </div>
        {error && <div className="mt-3 rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-700">{error}</div>}
      </div>

      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <h2 className="mb-3 font-semibold text-gray-800">提交代码（原始代码，重判不改写）</h2>
        <CodeBlock code={submission.code} language={submission.language} />
      </div>

      <div className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
        <div className="flex items-center gap-3">
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
        <ResultTable results={submission.results} />
      </div>

      {submission.rejudges?.length > 0 && (
        <div className="space-y-4">
          <h2 className="font-semibold text-gray-800">重判历史（{submission.rejudge_count} 次）</h2>
          {submission.rejudges.map((r) => (
            <div key={r.id} className="rounded-xl border border-gray-200 bg-white p-6 shadow-sm">
              <div className="flex flex-wrap items-center gap-3">
                <span className="rounded-full bg-violet-100 px-2.5 py-0.5 text-xs font-medium text-violet-700">
                  第 {r.seq} 次重判
                </span>
                <StatusBadge value={r.prev_status} kind="submission" />
                <span className="text-gray-400">→</span>
                <StatusBadge value={r.status} kind="submission" />
                <span className="text-sm text-gray-500">
                  得分 {r.prev_score}% → {r.score}%（<ScoreDelta delta={r.score_delta} />
                  ）· 耗时 {r.runtime_ms}ms
                </span>
                {r.points_awarded > 0 && (
                  <span className="rounded-full bg-amber-100 px-2.5 py-0.5 text-xs font-medium text-amber-700">
                    补发 +{r.points_awarded} 分
                  </span>
                )}
              </div>
              <div className="mt-2 text-xs text-gray-400">
                操作人：{r.operator_name} · 重判时间：{r.created_at}
              </div>
              {r.error_message && (
                <div className="mt-3 rounded-lg bg-rose-50 px-4 py-3 text-sm text-rose-700">{r.error_message}</div>
              )}
              <ResultTable results={r.results} />
            </div>
          ))}
        </div>
      )}

      <ConfirmDialog
        open={confirming}
        title="发起重判"
        message="将使用提交时的原始代码，按题目当前测试用例重新评测。重判生成独立记录，不会改写首次评测结果。确认继续？"
        confirmText="确认重判"
        onConfirm={doRejudge}
        onCancel={() => setConfirming(false)}
      />
    </div>
  )
}

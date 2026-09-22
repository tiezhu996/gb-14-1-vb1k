// 我的提交记录（管理员可查看全部）
import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useSubmissionStore } from '../../stores/submissionStore'
import StatusBadge from '../../components/StatusBadge'
import Pagination from '../../components/Pagination'
import { usePagination } from '../../hooks/usePagination'
import { useAuth } from '../../hooks/useAuth'
import { LANGUAGE_LABELS } from '../../constants'

export default function Submissions() {
  const { submissions, total, loading, fetchSubmissions } = useSubmissionStore()
  const { page, pageSize, setTotal, onPageChange } = usePagination(10)
  const { isAdmin, user } = useAuth()

  useEffect(() => {
    fetchSubmissions({ page, page_size: pageSize }).then(() => {})
  }, [fetchSubmissions, page, pageSize])

  useEffect(() => {
    setTotal(total)
  }, [total, setTotal])

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-800">{isAdmin ? '全部提交记录' : '我的提交'}</h1>
        <p className="mt-1 text-sm text-gray-500">
          查看每次提交的评测结果、重判次数与最新状态{isAdmin ? '（管理员可见全部学生记录）' : ''}
        </p>
      </div>

      {loading && submissions.length === 0 ? (
        <div className="py-20 text-center text-gray-400">加载中...</div>
      ) : (
        <div className="overflow-x-auto rounded-xl border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200 text-sm">
            <thead className="bg-gray-50">
              <tr>
                {isAdmin && <th className="px-5 py-3 text-left font-semibold text-gray-600">提交人</th>}
                <th className="px-5 py-3 text-left font-semibold text-gray-600">题目</th>
                <th className="px-5 py-3 text-left font-semibold text-gray-600">语言</th>
                <th className="px-5 py-3 text-left font-semibold text-gray-600">首次状态</th>
                <th className="px-5 py-3 text-left font-semibold text-gray-600">最新状态</th>
                <th className="px-5 py-3 text-left font-semibold text-gray-600">重判</th>
                <th className="px-5 py-3 text-left font-semibold text-gray-600">得分</th>
                <th className="px-5 py-3 text-left font-semibold text-gray-600">耗时</th>
                <th className="px-5 py-3 text-left font-semibold text-gray-600">提交时间</th>
                <th className="px-5 py-3 text-left font-semibold text-gray-600">操作</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {submissions.map((s) => {
                const latest = s.latest_status || s.status
                const rejudged = s.rejudge_count > 0
                return (
                  <tr key={s.id} className="hover:bg-gray-50">
                    {isAdmin && <td className="px-5 py-3 text-gray-600">{s.username}</td>}
                    <td className="px-5 py-3">
                      <Link to={`/problems/${s.problem_id}`} className="font-medium text-brand-600 hover:underline">
                        {s.problem_title}
                      </Link>
                    </td>
                    <td className="px-5 py-3 text-gray-600">{LANGUAGE_LABELS[s.language] || s.language}</td>
                    <td className="px-5 py-3">
                      <StatusBadge value={s.status} kind="submission" />
                    </td>
                    <td className="px-5 py-3">
                      <StatusBadge value={latest} kind="submission" />
                      {rejudged && latest !== s.status && (
                        <span className="ml-1 text-xs text-violet-600">（重判后）</span>
                      )}
                    </td>
                    <td className="px-5 py-3">
                      {rejudged ? (
                        <span className="rounded-full bg-violet-100 px-2 py-0.5 text-xs font-medium text-violet-700">
                          {s.rejudge_count} 次
                        </span>
                      ) : (
                        <span className="text-xs text-gray-400">0</span>
                      )}
                    </td>
                    <td className="px-5 py-3 font-medium text-gray-700">{s.score}%</td>
                    <td className="px-5 py-3 text-gray-600">{s.runtime_ms} ms</td>
                    <td className="px-5 py-3 text-gray-500">{s.created_at}</td>
                    <td className="px-5 py-3">
                      <Link
                        to={`/submissions/${s.id}`}
                        className={`text-sm font-medium hover:underline ${s.user_id === user?.id || isAdmin ? 'text-brand-600' : 'text-gray-400'}`}
                      >
                        详情
                      </Link>
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}

      <Pagination page={page} pageSize={pageSize} total={total} onChange={onPageChange} />
    </div>
  )
}

// 评测用例结果表（首次评测 / 重判历史复用）
import type { JudgeResult } from '../types'

interface Props {
  results: JudgeResult[]
  // 需要高亮差异的用例下标（重判相对上一轮翻转的用例）
  highlightCases?: number[]
}

export default function JudgeResultsTable({ results, highlightCases = [] }: Props) {
  if (!results || results.length === 0) return null
  const highlight = new Set(highlightCases)
  return (
    <div className="overflow-x-auto">
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
            <tr key={r.test_case_index} className={highlight.has(r.test_case_index) ? 'bg-amber-50' : ''}>
              <td className="px-4 py-2">
                #{r.test_case_index + 1}
                {highlight.has(r.test_case_index) && <span className="ml-1 text-xs font-medium text-amber-600">差异</span>}
              </td>
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

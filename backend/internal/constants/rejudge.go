package constants

// 重判业务规则常量（与 model.Rejudge / rejudge_service 多处耦合）。

// RejudgeEligibleStatus 判断提交当前状态是否允许发起重判（已完成但未通过）。
func RejudgeEligibleStatus(status string) bool {
	switch status {
	case SubmissionPartial, SubmissionRuntimeError, SubmissionTimeout:
		return true
	}
	return false
}

// TerminalSubmissionStatus 判断评测状态是否为终态（已完成）。
func TerminalSubmissionStatus(status string) bool {
	switch status {
	case SubmissionAccepted, SubmissionPartial, SubmissionRuntimeError, SubmissionTimeout:
		return true
	}
	return false
}

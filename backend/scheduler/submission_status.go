package scheduler

type SubmissionStatus string

const (
	StatusAccepted            SubmissionStatus = "ACCEPTED"
	StatusWrongAnswer         SubmissionStatus = "WRONG ANSWER"
	StatusCompileError        SubmissionStatus = "COMPILE ERROR"
	StatusRuntimeError        SubmissionStatus = "RUNTIME ERROR"
	StatusTimeLimitExceeded   SubmissionStatus = "TIME LIMIT EXCEEDED"
	StatusMemoryLimitExceeded SubmissionStatus = "MEMORY LIMIT EXCEEDED"
	StatusTimeout             SubmissionStatus = "TIMEOUT"
	StatusInternalError       SubmissionStatus = "INTERNAL ERROR"
	StatusPending             SubmissionStatus = "PENDING"
	StatusInProgress          SubmissionStatus = "IN_PROGRESS"
)

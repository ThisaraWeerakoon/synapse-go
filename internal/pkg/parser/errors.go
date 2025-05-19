package parser

import "fmt"

// ErrUnsupportedExpression is returned when the expression type is not supported.
type ErrUnsupportedExpression struct {
	Expression string
}

func (e *ErrUnsupportedExpression) Error() string {
	return fmt.Sprintf("unsupported expression: %s", e.Expression)
}

// ErrEvaluationFailed is returned when expression evaluation fails.
type ErrEvaluationFailed struct {
	Expression string
	Reason     string
	InnerError error
}

func (e *ErrEvaluationFailed) Error() string {
	if e.InnerError != nil {
		return fmt.Sprintf("evaluation failed for expression '%s': %s (Caused by: %v)", e.Expression, e.Reason, e.InnerError)
	}
	return fmt.Sprintf("evaluation failed for expression '%s': %s", e.Expression, e.Reason)
}
func (e *ErrEvaluationFailed) Unwrap() error { return e.InnerError }


// ErrInvalidPayloadForOperation is returned when an operation is attempted on an unsuitable payload.
type ErrInvalidPayloadForOperation struct {
	Operation   string
	PayloadType string
	Reason      string
}

func (e *ErrInvalidPayloadForOperation) Error() string {
	return fmt.Sprintf("invalid payload for operation '%s': payload type '%s'. Reason: %s", e.Operation, e.PayloadType, e.Reason)
}

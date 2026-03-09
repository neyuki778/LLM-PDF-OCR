package task

import "fmt"

type PageLimitExceededError struct {
	TotalPages int
	MaxPages   int
}

func (e *PageLimitExceededError) Error() string {
	return fmt.Sprintf("total pages %d exceeds max pages %d", e.TotalPages, e.MaxPages)
}

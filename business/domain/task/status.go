package task

import (
	"fmt"
	"strings"
)

// Status represents the status of the task in the system.
type Status int

const (
	StatusPending Status = iota
	StatusFailed
	StatusCompleted
)

var statusNames = []string{"pending", "failed", "completed"}

func (s Status) String() string {
	if s < StatusPending || s > StatusCompleted {
		return "UNKNOWN"
	}
	return statusNames[s]
}

// ParseStatus creates a status off of single string or return error if status is invalid.
func ParseStatus(s string) (Status, error) {
	for i, status := range statusNames {
		if strings.ToLower(s) == status {
			return Status(i), nil
		}
	}
	return Status(-1), fmt.Errorf("%q is invalid status", s)
}

package worker

import (
	"strings"
)

type WorkerStatus int

const (
	//WorkerBusy worker exeuting command.
	WorkerBusy WorkerStatus = iota
	//WorkerIdle worker ready to accept command
	WorkerIdle
)

var names = [...]string{"busy", "idle"}

// String implements the stringer interface.
func (w WorkerStatus) String() string {
	if w < WorkerBusy || w > WorkerIdle {
		return "UNKNOWN"
	}
	return names[w]
}

// ParseStatus creates a status from string.
func ParseStatus(s string) WorkerStatus {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)

	for i, name := range names {
		if s == name {
			return WorkerStatus(i)
		}
	}
	return WorkerStatus(-1)
}

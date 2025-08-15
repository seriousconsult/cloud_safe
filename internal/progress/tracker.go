package progress

import (
	"time"
)

// Tracker interface for progress tracking
type Tracker interface {
        Update(bytes int64)
        GetProgress() (transferred, total int64, percentage float64)
        GetSpeed() float64
        GetETA() time.Duration
        Finish()
        SetTotal(total int64)
        SetTransferred(transferred int64)
}

// NewTracker creates a new progress tracker
func NewTracker(totalSize int64) Tracker {
        return NewSimpleTracker(totalSize)
}
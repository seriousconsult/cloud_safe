package progress

import (
	"fmt"
	"sync"
	"time"

	"github.com/cheggaaa/pb/v3"
)

// Tracker tracks upload progress and provides ETA calculations
type Tracker struct {
	mu           sync.RWMutex
	totalSize    int64
	transferred  int64
	startTime    time.Time
	progressBar  *pb.ProgressBar
	lastUpdate   time.Time
	updateCount  int64
}

// NewTracker creates a new progress tracker
func NewTracker(totalSize int64) *Tracker {
	bar := pb.Full.Start64(totalSize)
	bar.Set(pb.Bytes, true)
	bar.Set(pb.SIBytesPrefix, true)
	
	return &Tracker{
		totalSize:   totalSize,
		startTime:   time.Now(),
		progressBar: bar,
		lastUpdate:  time.Now(),
	}
}

// Update updates the progress with the number of bytes transferred
func (t *Tracker) Update(bytes int64) {
	t.mu.Lock()
	t.transferred += bytes
	t.updateCount++
	t.lastUpdate = time.Now()
	
	// Update progress bar
	t.progressBar.SetCurrent(t.transferred)
	
	// Update ETA and speed every 10 updates to avoid too frequent calculations
	if t.updateCount%10 == 0 {
		t.updateStats()
	}
	t.mu.Unlock()
}

// updateStats updates transfer statistics
func (t *Tracker) updateStats() {
	elapsed := time.Since(t.startTime)
	if elapsed.Seconds() > 0 {
		speed := float64(t.transferred) / elapsed.Seconds()
		remaining := t.totalSize - t.transferred
		
		if speed > 0 {
			eta := time.Duration(float64(remaining)/speed) * time.Second
			t.progressBar.Set("eta", eta.Round(time.Second))
		}
		
		t.progressBar.Set("speed", fmt.Sprintf("%.2f MB/s", speed/(1024*1024)))
	}
}

// GetProgress returns current progress information
func (t *Tracker) GetProgress() (transferred, total int64, percentage float64) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	percentage = 0
	if t.totalSize > 0 {
		percentage = float64(t.transferred) / float64(t.totalSize) * 100
	}
	
	return t.transferred, t.totalSize, percentage
}

// GetSpeed returns current transfer speed in bytes per second
func (t *Tracker) GetSpeed() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	elapsed := time.Since(t.startTime)
	if elapsed.Seconds() > 0 {
		return float64(t.transferred) / elapsed.Seconds()
	}
	return 0
}

// GetETA returns estimated time to completion
func (t *Tracker) GetETA() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	speed := t.GetSpeed()
	if speed > 0 {
		remaining := t.totalSize - t.transferred
		return time.Duration(float64(remaining)/speed) * time.Second
	}
	return 0
}

// Finish completes the progress tracking
func (t *Tracker) Finish() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.progressBar.Finish()
	
	elapsed := time.Since(t.startTime)
	avgSpeed := float64(t.totalSize) / elapsed.Seconds()
	
	fmt.Printf("\nUpload completed in %v (average speed: %.2f MB/s)\n", 
		elapsed.Round(time.Second), avgSpeed/(1024*1024))
}

// SetTotal updates the total size (useful for resumable uploads)
func (t *Tracker) SetTotal(total int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.totalSize = total
	t.progressBar.SetTotal(total)
}

// SetTransferred sets the current transferred amount (useful for resumable uploads)
func (t *Tracker) SetTransferred(transferred int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.transferred = transferred
	t.progressBar.SetCurrent(transferred)
}

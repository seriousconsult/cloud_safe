package progress

import (
	"fmt"
	"sync"
	"time"
)

// SimpleTracker provides basic progress tracking without external dependencies
type SimpleTracker struct {
	mu          sync.RWMutex
	totalSize   int64
	transferred int64
	startTime   time.Time
	lastUpdate  time.Time
	updateCount int64
}

// NewSimpleTracker creates a new simple progress tracker
func NewSimpleTracker(totalSize int64) *SimpleTracker {
	return &SimpleTracker{
		totalSize:  totalSize,
		startTime:  time.Now(),
		lastUpdate: time.Now(),
	}
}

// Update updates the progress with the number of bytes transferred
func (t *SimpleTracker) Update(bytes int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.transferred += bytes
	t.updateCount++
	t.lastUpdate = time.Now()
	
	// Print progress every 50 updates to avoid spam
	if t.updateCount%50 == 0 {
		t.printProgress()
	}
}

// printProgress prints current progress to console
func (t *SimpleTracker) printProgress() {
	percentage := float64(t.transferred) / float64(t.totalSize) * 100
	elapsed := time.Since(t.startTime)
	speed := float64(t.transferred) / elapsed.Seconds() / (1024 * 1024) // MB/s
	
	fmt.Printf("\rProgress: %.1f%% (%.2f MB/%.2f MB) Speed: %.2f MB/s", 
		percentage, 
		float64(t.transferred)/(1024*1024), 
		float64(t.totalSize)/(1024*1024), 
		speed)
}

// GetProgress returns current progress information
func (t *SimpleTracker) GetProgress() (transferred, total int64, percentage float64) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	percentage = 0
	if t.totalSize > 0 {
		percentage = float64(t.transferred) / float64(t.totalSize) * 100
	}
	
	return t.transferred, t.totalSize, percentage
}

// GetSpeed returns current transfer speed in bytes per second
func (t *SimpleTracker) GetSpeed() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	elapsed := time.Since(t.startTime)
	if elapsed.Seconds() > 0 {
		return float64(t.transferred) / elapsed.Seconds()
	}
	return 0
}

// GetETA returns estimated time to completion
func (t *SimpleTracker) GetETA() time.Duration {
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
func (t *SimpleTracker) Finish() {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	elapsed := time.Since(t.startTime)
	avgSpeed := float64(t.totalSize) / elapsed.Seconds() / (1024 * 1024)
	
	fmt.Printf("\n\nUpload completed in %v (average speed: %.2f MB/s)\n", 
		elapsed.Round(time.Second), avgSpeed)
}

// SetTotal updates the total size (useful for resumable uploads)
func (t *SimpleTracker) SetTotal(total int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.totalSize = total
}

// SetTransferred sets the current transferred amount (useful for resumable uploads)
func (t *SimpleTracker) SetTransferred(transferred int64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.transferred = transferred
}
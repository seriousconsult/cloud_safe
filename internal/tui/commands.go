
package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbletea"
)

// Message types
type progressMsg struct {
	transferred int64
	total       int64
	speed       float64
}

type completeMsg struct{}

type errorMsg struct {
	error error
}

// startProcessing starts the upload process in a goroutine
func (m Model) startProcessing() tea.Cmd {
	return func() tea.Msg {
		err := m.processor.ProcessWithProgress(m.ctx, func(transferred, total int64, speed float64) {
			// This callback is called from the processor
			// We'll use a different approach to get progress updates
		})
		
		if err != nil {
			if err == context.Canceled {
				return errorMsg{error: err}
			}
			return errorMsg{error: err}
		}
		
		return completeMsg{}
	}
}

// waitForProgress returns a command that waits for progress updates
func (m Model) waitForProgress() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		// In a real implementation, you'd get actual progress from the processor
		// For now, we'll simulate progress
		if m.state == StateProgress {
			// This is where you'd get real progress data
			// For demonstration, we'll just return a progress message
			return progressMsg{
				transferred: m.transferred,
				total:       m.total,
				speed:       m.speed,
			}
		}
		return nil
	})
}

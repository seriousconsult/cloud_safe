
package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"cloudarchiver/internal/config"
	"cloudarchiver/internal/logger"
	"cloudarchiver/internal/pipeline"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// State represents the current state of the TUI
type State int

const (
	StateSetup State = iota
	StateProgress
	StateComplete
	StateError
)

// Model holds the state for the TUI
type Model struct {
	state       State
	config      *config.Config
	logger      *logger.Logger
	processor   *pipeline.Processor
	
	// Setup form fields
	sourcePath  string
	s3Bucket    string
	s3Key       string
	workers     int
	encrypt     bool
	resume      bool
	
	// Current field being edited
	currentField int
	
	// Progress tracking
	transferred int64
	total       int64
	speed       float64
	eta         time.Duration
	
	// Status
	status      string
	error       error
	
	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)
	
	fieldStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("246"))
	
	activeFieldStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("205")).
		Bold(true)
	
	progressStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))
	
	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	
	statusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("33"))
)

// NewModel creates a new TUI model
func NewModel() Model {
	ctx, cancel := context.WithCancel(context.Background())
	
	return Model{
		state:        StateSetup,
		workers:      4,
		encrypt:      true,
		resume:       true,
		currentField: 0,
		ctx:          ctx,
		cancel:       cancel,
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)
	case progressMsg:
		m.transferred = msg.transferred
		m.total = msg.total
		m.speed = msg.speed
		return m, m.waitForProgress()
	case completeMsg:
		m.state = StateComplete
		m.status = "Upload completed successfully!"
		return m, tea.Quit
	case errorMsg:
		m.state = StateError
		m.error = msg.error
		return m, tea.Quit
	}
	
	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case StateSetup:
		return m.handleSetupKeys(msg)
	case StateProgress:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			m.cancel()
			return m, tea.Quit
		}
	}
	
	return m, nil
}

// handleSetupKeys handles keys during setup phase
func (m Model) handleSetupKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up":
		if m.currentField > 0 {
			m.currentField--
		}
	case "down":
		if m.currentField < 5 {
			m.currentField++
		}
	case "tab":
		m.currentField = (m.currentField + 1) % 6
	case "enter":
		if m.currentField == 5 { // Start button
			return m.startUpload()
		}
		// For boolean fields, toggle them
		if m.currentField == 3 { // encrypt
			m.encrypt = !m.encrypt
		} else if m.currentField == 4 { // resume
			m.resume = !m.resume
		}
	case "backspace":
		return m.handleBackspace(), nil
	default:
		return m.handleTextInput(msg.String()), nil
	}
	
	return m, nil
}

// handleBackspace removes last character from current field
func (m Model) handleBackspace() Model {
	switch m.currentField {
	case 0:
		if len(m.sourcePath) > 0 {
			m.sourcePath = m.sourcePath[:len(m.sourcePath)-1]
		}
	case 1:
		if len(m.s3Bucket) > 0 {
			m.s3Bucket = m.s3Bucket[:len(m.s3Bucket)-1]
		}
	case 2:
		if len(m.s3Key) > 0 {
			m.s3Key = m.s3Key[:len(m.s3Key)-1]
		}
	}
	return m
}

// handleTextInput adds text to current field
func (m Model) handleTextInput(input string) Model {
	// Only handle printable characters
	if len(input) != 1 {
		return m
	}
	
	switch m.currentField {
	case 0:
		m.sourcePath += input
	case 1:
		m.s3Bucket += input
	case 2:
		m.s3Key += input
	}
	return m
}

// startUpload validates inputs and starts the upload process
func (m Model) startUpload() (tea.Model, tea.Cmd) {
	// Validate required fields
	if m.sourcePath == "" || m.s3Bucket == "" || m.s3Key == "" {
		m.status = "Please fill in all required fields"
		return m, nil
	}
	
	// Check if source path exists
	if _, err := os.Stat(m.sourcePath); os.IsNotExist(err) {
		m.status = "Source path does not exist"
		return m, nil
	}
	
	// Create config
	m.config = &config.Config{
		SourcePath: m.sourcePath,
		S3Bucket:   m.s3Bucket,
		S3Key:      m.s3Key,
		Workers:    m.workers,
		ChunkSize:  100 * 1024 * 1024, // 100MB
		BufferSize: 64 * 1024,         // 64KB
		Encrypt:    m.encrypt,
		Resume:     m.resume,
		AWSRegion:  os.Getenv("AWS_REGION"),
		AWSProfile: os.Getenv("AWS_PROFILE"),
	}
	
	if m.config.AWSRegion == "" {
		m.config.AWSRegion = "us-east-1"
	}
	
	// Create logger and processor
	m.logger = logger.New(false)
	processor, err := pipeline.NewProcessor(m.config, m.logger)
	if err != nil {
		m.status = fmt.Sprintf("Failed to create processor: %v", err)
		return m, nil
	}
	
	m.processor = processor
	m.state = StateProgress
	m.status = "Starting upload..."
	
	// Start the upload process
	return m, tea.Batch(
		m.startProcessing(),
		m.waitForProgress(),
	)
}

// View renders the current state
func (m Model) View() string {
	switch m.state {
	case StateSetup:
		return m.setupView()
	case StateProgress:
		return m.progressView()
	case StateComplete:
		return m.completeView()
	case StateError:
		return m.errorView()
	}
	return ""
}

// setupView renders the setup form
func (m Model) setupView() string {
	var b strings.Builder
	
	b.WriteString(titleStyle.Render("CloudArchiver - Setup"))
	b.WriteString("\n\n")
	
	// Source path
	label := "Source Path:"
	value := m.sourcePath
	if value == "" {
		value = "<enter path>"
	}
	if m.currentField == 0 {
		b.WriteString(activeFieldStyle.Render(fmt.Sprintf("→ %s %s", label, value)))
	} else {
		b.WriteString(fieldStyle.Render(fmt.Sprintf("  %s %s", label, value)))
	}
	b.WriteString("\n")
	
	// S3 Bucket
	label = "S3 Bucket:"
	value = m.s3Bucket
	if value == "" {
		value = "<enter bucket>"
	}
	if m.currentField == 1 {
		b.WriteString(activeFieldStyle.Render(fmt.Sprintf("→ %s %s", label, value)))
	} else {
		b.WriteString(fieldStyle.Render(fmt.Sprintf("  %s %s", label, value)))
	}
	b.WriteString("\n")
	
	// S3 Key
	label = "S3 Key:"
	value = m.s3Key
	if value == "" {
		value = "<enter key>"
	}
	if m.currentField == 2 {
		b.WriteString(activeFieldStyle.Render(fmt.Sprintf("→ %s %s", label, value)))
	} else {
		b.WriteString(fieldStyle.Render(fmt.Sprintf("  %s %s", label, value)))
	}
	b.WriteString("\n")
	
	// Encryption
	label = "Encryption:"
	value = fmt.Sprintf("%t", m.encrypt)
	if m.currentField == 3 {
		b.WriteString(activeFieldStyle.Render(fmt.Sprintf("→ %s %s (press Enter to toggle)", label, value)))
	} else {
		b.WriteString(fieldStyle.Render(fmt.Sprintf("  %s %s", label, value)))
	}
	b.WriteString("\n")
	
	// Resume
	label = "Resume:"
	value = fmt.Sprintf("%t", m.resume)
	if m.currentField == 4 {
		b.WriteString(activeFieldStyle.Render(fmt.Sprintf("→ %s %s (press Enter to toggle)", label, value)))
	} else {
		b.WriteString(fieldStyle.Render(fmt.Sprintf("  %s %s", label, value)))
	}
	b.WriteString("\n\n")
	
	// Start button
	if m.currentField == 5 {
		b.WriteString(activeFieldStyle.Render("→ [Start Upload]"))
	} else {
		b.WriteString(fieldStyle.Render("  [Start Upload]"))
	}
	b.WriteString("\n\n")
	
	if m.status != "" {
		b.WriteString(statusStyle.Render(m.status))
		b.WriteString("\n")
	}
	
	b.WriteString(fieldStyle.Render("Use arrow keys/tab to navigate, Enter to select/toggle, Ctrl+C to quit"))
	
	return b.String()
}

// progressView renders the progress display
func (m Model) progressView() string {
	var b strings.Builder
	
	b.WriteString(titleStyle.Render("CloudArchiver - Upload Progress"))
	b.WriteString("\n\n")
	
	// Progress bar
	if m.total > 0 {
		percentage := float64(m.transferred) / float64(m.total) * 100
		barWidth := 40
		filled := int(percentage / 100 * float64(barWidth))
		
		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
		b.WriteString(progressStyle.Render(fmt.Sprintf("[%s] %.1f%%", bar, percentage)))
		b.WriteString("\n")
		
		// Transfer stats
		b.WriteString(fmt.Sprintf("Transferred: %d / %d bytes\n", m.transferred, m.total))
		if m.speed > 0 {
			b.WriteString(fmt.Sprintf("Speed: %.2f MB/s\n", m.speed/(1024*1024)))
		}
	}
	
	if m.status != "" {
		b.WriteString(statusStyle.Render(m.status))
		b.WriteString("\n")
	}
	
	b.WriteString("\n")
	b.WriteString(fieldStyle.Render("Press Ctrl+C or 'q' to cancel"))
	
	return b.String()
}

// completeView renders the completion screen
func (m Model) completeView() string {
	var b strings.Builder
	
	b.WriteString(titleStyle.Render("CloudArchiver - Complete"))
	b.WriteString("\n\n")
	b.WriteString(progressStyle.Render(m.status))
	b.WriteString("\n\n")
	b.WriteString(fieldStyle.Render("Press any key to exit"))
	
	return b.String()
}

// errorView renders the error screen
func (m Model) errorView() string {
	var b strings.Builder
	
	b.WriteString(titleStyle.Render("CloudArchiver - Error"))
	b.WriteString("\n\n")
	if m.error != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.error)))
	}
	b.WriteString("\n\n")
	b.WriteString(fieldStyle.Render("Press any key to exit"))
	
	return b.String()
}

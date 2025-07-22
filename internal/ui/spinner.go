package ui

import (
	"fmt"
	"strings"
	"time"
)

// SpinnerDisplay wraps any display with spinner functionality
type SpinnerDisplay struct {
	base    ToolDisplayInterface
	spinner *ProgressSpinner
}

// NewSpinnerDisplay creates a display wrapper with spinner support
func NewSpinnerDisplay(base ToolDisplayInterface) *SpinnerDisplay {
	return &SpinnerDisplay{
		base: base,
	}
}

// ShowToolCall displays a tool call, potentially with spinner
func (s *SpinnerDisplay) ShowToolCall(serverName, toolName string, arguments map[string]interface{}) error {
	return s.base.ShowToolCall(serverName, toolName, arguments)
}

// ShowToolResult displays the result and stops any active spinner
func (s *SpinnerDisplay) ShowToolResult(result interface{}, duration time.Duration) error {
	if s.spinner != nil {
		s.spinner.Stop()
		s.spinner = nil
	}
	return s.base.ShowToolResult(result, duration)
}

// ShowError displays an error and stops any active spinner
func (s *SpinnerDisplay) ShowError(err error) error {
	if s.spinner != nil {
		s.spinner.Stop()
		s.spinner = nil
	}
	return s.base.ShowError(err)
}

// ShowProgress displays progress with spinner animation
func (s *SpinnerDisplay) ShowProgress(message string) error {
	if s.spinner != nil {
		s.spinner.Stop()
		s.spinner = nil
	}
	
	// Start spinner for operations that might take time
	shouldSpin := strings.Contains(message, "Processing") || 
	             strings.Contains(message, "Finding") ||
	             strings.Contains(message, "Initializing") ||
	             strings.Contains(strings.ToLower(message), "loading") ||
	             strings.Contains(strings.ToLower(message), "connecting") ||
	             strings.Contains(strings.ToLower(message), "ai agent") ||
	             strings.Contains(strings.ToLower(message), "llm")
	
	if shouldSpin {
		s.spinner = NewProgressSpinner(message)
		s.spinner.Start()
		// Give spinner time to show before returning
		time.Sleep(100 * time.Millisecond)
		return nil
	}
	
	return s.base.ShowProgress(message)
}

// ShowSummary displays execution summary and stops any spinner
func (s *SpinnerDisplay) ShowSummary(summary ExecutionSummary) error {
	if s.spinner != nil {
		s.spinner.Stop()
		s.spinner = nil
	}
	return s.base.ShowSummary(summary)
}

// PromptToolApproval prompts for approval and stops spinner
func (s *SpinnerDisplay) PromptToolApproval(serverName, toolName string, arguments map[string]interface{}) (bool, error) {
	if s.spinner != nil {
		s.spinner.Stop()
		s.spinner = nil
	}
	return s.base.PromptToolApproval(serverName, toolName, arguments)
}

// Enhanced progress display with time tracking
type TimedProgressDisplay struct {
	base            ToolDisplayInterface
	sessionStartTime time.Time
	lastEventTime   time.Time
}

// NewTimedProgressDisplay creates a display that tracks operation timing
func NewTimedProgressDisplay(base ToolDisplayInterface) *TimedProgressDisplay {
	now := time.Now()
	return &TimedProgressDisplay{
		base:             base,
		sessionStartTime: now,
		lastEventTime:    now,
	}
}

// ShowToolCall displays a tool call with timing
func (t *TimedProgressDisplay) ShowToolCall(serverName, toolName string, arguments map[string]interface{}) error {
	now := time.Now()
	elapsed := now.Sub(t.lastEventTime)
	t.lastEventTime = now
	fmt.Printf("⏱️  %s ", ColorGray(fmt.Sprintf("[+%.1fs]", elapsed.Seconds())))
	return t.base.ShowToolCall(serverName, toolName, arguments)
}

// ShowToolResult displays result with timing
func (t *TimedProgressDisplay) ShowToolResult(result interface{}, duration time.Duration) error {
	return t.base.ShowToolResult(result, duration)
}

// ShowError displays error with timing
func (t *TimedProgressDisplay) ShowError(err error) error {
	now := time.Now()
	elapsed := now.Sub(t.lastEventTime)
	t.lastEventTime = now
	fmt.Printf("⏱️  %s ", ColorGray(fmt.Sprintf("[+%.1fs]", elapsed.Seconds())))
	return t.base.ShowError(err)
}

// ShowProgress displays progress with timing
func (t *TimedProgressDisplay) ShowProgress(message string) error {
	now := time.Now()
	elapsed := now.Sub(t.lastEventTime)
	t.lastEventTime = now
	
	// For spinner-worthy messages, don't add timing immediately - let spinner handle it
	shouldSpin := strings.Contains(message, "Processing") || 
	             strings.Contains(message, "Finding") ||
	             strings.Contains(message, "Initializing") ||
	             strings.Contains(strings.ToLower(message), "loading") ||
	             strings.Contains(strings.ToLower(message), "connecting") ||
	             strings.Contains(strings.ToLower(message), "ai agent") ||
	             strings.Contains(strings.ToLower(message), "llm")
	
	if shouldSpin {
		// Let the base display (which includes spinner) handle this
		return t.base.ShowProgress(message)
	}
	
	// For regular messages, show timing
	fmt.Printf("⏱️  %s %s\n", 
		ColorGray(fmt.Sprintf("[+%.1fs]", elapsed.Seconds())), 
		ColorYellow(message))
	return nil
}

// ShowSummary displays summary with total time
func (t *TimedProgressDisplay) ShowSummary(summary ExecutionSummary) error {
	totalElapsed := time.Since(t.sessionStartTime)
	fmt.Printf("\n⏱️  %s Total session time: %s\n", 
		ColorGray("Session:"), 
		ColorWhite(totalElapsed.String()))
	return t.base.ShowSummary(summary)
}

// PromptToolApproval prompts with timing context
func (t *TimedProgressDisplay) PromptToolApproval(serverName, toolName string, arguments map[string]interface{}) (bool, error) {
	now := time.Now()
	elapsed := now.Sub(t.lastEventTime)
	t.lastEventTime = now
	fmt.Printf("⏱️  %s ", ColorGray(fmt.Sprintf("[+%.1fs]", elapsed.Seconds())))
	return t.base.PromptToolApproval(serverName, toolName, arguments)
}
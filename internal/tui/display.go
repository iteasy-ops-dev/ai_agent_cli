package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/iteasy-ops-dev/syseng-agent/internal/llm"
	"github.com/iteasy-ops-dev/syseng-agent/internal/ui"
)

// TUIDisplay implements the ToolDisplayInterface for Bubble Tea TUI
type TUIDisplay struct {
	program *tea.Program
	model   *ChatModel
}

// NewTUIDisplay creates a new TUI display
func NewTUIDisplay(program *tea.Program, model *ChatModel) *TUIDisplay {
	return &TUIDisplay{
		program: program,
		model:   model,
	}
}

// ToolCallMsg represents a tool call display message
type ToolCallMsg struct {
	ServerName string
	ToolName   string
	Arguments  map[string]interface{}
}

// ToolResultMsg represents a tool result display message
type ToolResultMsg struct {
	Result   interface{}
	Duration time.Duration
}

// ToolErrorMsg represents a tool error display message
type ToolErrorMsg struct {
	Error error
}

// ProgressMsg represents a progress display message
type ProgressMsg struct {
	Message string
}

// SummaryMsg represents an execution summary display message
type SummaryMsg struct {
	Summary ui.ExecutionSummary
}

// ToolApprovalPromptMsg represents a tool approval prompt
type ToolApprovalPromptMsg struct {
	ServerName string
	ToolName   string
	Arguments  map[string]interface{}
	Response   chan bool
}

func (d *TUIDisplay) ShowToolCall(serverName, toolName string, arguments map[string]interface{}) error {
	if d.program != nil {
		d.program.Send(ToolCallMsg{
			ServerName: serverName,
			ToolName:   toolName,
			Arguments:  arguments,
		})
	}
	return nil
}

func (d *TUIDisplay) ShowToolResult(result interface{}, duration time.Duration) error {
	if d.program != nil {
		d.program.Send(ToolResultMsg{
			Result:   result,
			Duration: duration,
		})
	}
	return nil
}

func (d *TUIDisplay) ShowError(err error) error {
	if d.program != nil {
		d.program.Send(ToolErrorMsg{
			Error: err,
		})
	}
	return nil
}

func (d *TUIDisplay) ShowProgress(message string) error {
	if d.program != nil {
		d.program.Send(ProgressMsg{
			Message: message,
		})
	}
	return nil
}

func (d *TUIDisplay) ShowSummary(summary ui.ExecutionSummary) error {
	if d.program != nil {
		d.program.Send(SummaryMsg{
			Summary: summary,
		})
	}
	return nil
}

func (d *TUIDisplay) PromptToolApproval(serverName, toolName string, arguments map[string]interface{}) (bool, error) {
	// For TUI, we can implement a modal dialog or fallback to the existing interactive display
	// For now, let's use a simplified approach with a channel
	response := make(chan bool, 1)
	
	if d.program != nil {
		d.program.Send(ToolApprovalPromptMsg{
			ServerName: serverName,
			ToolName:   toolName,
			Arguments:  arguments,
			Response:   response,
		})
		
		// Wait for response
		approved := <-response
		return approved, nil
	}
	
	// Fallback: auto-approve for TUI mode (can be changed later)
	return true, nil
}

// Helper functions to format tool information for display
func formatToolCall(serverName, toolName string, arguments map[string]interface{}) string {
	var argStr strings.Builder
	for key, value := range arguments {
		if argStr.Len() > 0 {
			argStr.WriteString(", ")
		}
		argStr.WriteString(fmt.Sprintf("%s=%v", key, value))
	}
	
	toolCallStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("32")).
		Bold(true)
	
	argStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("242"))
	
	return fmt.Sprintf("%s %s.%s(%s)",
		"🔧",
		toolCallStyle.Render(serverName),
		toolCallStyle.Render(toolName),
		argStyle.Render(argStr.String()),
	)
}

func formatToolResult(result interface{}, duration time.Duration) string {
	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("82")).
		Bold(true)
	
	durationStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))
	
	resultStr := fmt.Sprintf("%v", result)
	if len(resultStr) > llm.LocalContentTruncateLimit {
		resultStr = resultStr[:llm.LocalContentTruncateLimit] + "..."
	}
	
	return fmt.Sprintf("%s %s %s\n%s",
		"✅",
		successStyle.Render("Success"),
		durationStyle.Render(fmt.Sprintf("(%.2fs)", duration.Seconds())),
		resultStr,
	)
}

func formatToolError(err error) string {
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)
	
	return fmt.Sprintf("%s %s: %v",
		"❌",
		errorStyle.Render("Error"),
		err,
	)
}

func formatProgress(message string) string {
	progressStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214"))
	
	return fmt.Sprintf("%s %s",
		"⏳",
		progressStyle.Render(message),
	)
}

func formatSummary(summary ui.ExecutionSummary) string {
	summaryStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)
	
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("%s %s\n",
		"📊",
		summaryStyle.Render("Execution Summary"),
	))
	
	builder.WriteString(fmt.Sprintf("   Total tools called: %d\n", summary.TotalTools))
	builder.WriteString(fmt.Sprintf("   Successful: %d, Failed: %d\n", summary.SuccessfulCalls, summary.FailedCalls))
	builder.WriteString(fmt.Sprintf("   Total duration: %.2fs\n", summary.TotalDuration.Seconds()))
	
	if len(summary.ToolCalls) > 0 {
		builder.WriteString("   Tool execution order:\n")
		for i, call := range summary.ToolCalls {
			status := "✅"
			if !call.Success {
				status = "❌"
			}
			builder.WriteString(fmt.Sprintf("     %d. %s %s.%s (%.2fs)\n",
				i+1,
				status,
				call.ServerName,
				call.ToolName,
				call.Duration.Seconds(),
			))
		}
	}
	
	return builder.String()
}

// SimpleTUIDisplay is a minimal display implementation for TUI background processing
type SimpleTUIDisplay struct{}

// NewSimpleTUIDisplay creates a simple display that doesn't interfere with TUI
func NewSimpleTUIDisplay() *SimpleTUIDisplay {
	return &SimpleTUIDisplay{}
}

func (d *SimpleTUIDisplay) ShowToolCall(serverName, toolName string, arguments map[string]interface{}) error {
	// No-op for TUI - the TUI will handle display through its message system
	return nil
}

func (d *SimpleTUIDisplay) ShowToolResult(result interface{}, duration time.Duration) error {
	// No-op for TUI
	return nil
}

func (d *SimpleTUIDisplay) ShowError(err error) error {
	// No-op for TUI
	return nil
}

func (d *SimpleTUIDisplay) ShowProgress(message string) error {
	// No-op for TUI
	return nil
}

func (d *SimpleTUIDisplay) ShowSummary(summary ui.ExecutionSummary) error {
	// No-op for TUI
	return nil
}

func (d *SimpleTUIDisplay) PromptToolApproval(serverName, toolName string, arguments map[string]interface{}) (bool, error) {
	// For TUI, we'll auto-approve for now since the TUI doesn't have interactive approval yet
	// TODO: Implement TUI-based approval dialog
	return true, nil
}
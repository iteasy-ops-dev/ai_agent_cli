package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// ToolDisplayInterface defines how tool execution is displayed to users
type ToolDisplayInterface interface {
	ShowToolCall(serverName, toolName string, arguments map[string]interface{}) error
	ShowToolResult(result interface{}, duration time.Duration) error
	ShowError(err error) error
	ShowProgress(message string) error
	ShowSummary(summary ExecutionSummary) error
	PromptToolApproval(serverName, toolName string, arguments map[string]interface{}) (bool, error)
}

// ExecutionSummary contains statistics about tool execution
type ExecutionSummary struct {
	TotalTools     int
	SuccessfulCalls int
	FailedCalls    int
	TotalDuration  time.Duration
	ToolCalls      []ToolCallRecord
}

// ToolCallRecord tracks individual tool call details
type ToolCallRecord struct {
	ServerName string
	ToolName   string
	Duration   time.Duration
	Success    bool
	Error      string
}

// NonInteractiveDisplay shows tool execution without user interaction
type NonInteractiveDisplay struct{}

// InteractiveDisplay allows user control over tool execution
type InteractiveDisplay struct {
	reader *bufio.Reader
}

// NewNonInteractiveDisplay creates a display that shows progress without interaction
func NewNonInteractiveDisplay() *NonInteractiveDisplay {
	return &NonInteractiveDisplay{}
}

// NewInteractiveDisplay creates a display that prompts user for tool approval
func NewInteractiveDisplay() *InteractiveDisplay {
	return &InteractiveDisplay{
		reader: bufio.NewReader(os.Stdin),
	}
}

// ShowToolCall displays a tool call without interaction
func (d *NonInteractiveDisplay) ShowToolCall(serverName, toolName string, arguments map[string]interface{}) error {
	fmt.Printf("🔧 %s Calling tool: %s%s\n", 
		ColorBlue("Tool Call"), 
		ColorCyan(fmt.Sprintf("%s.%s", serverName, toolName)),
		ColorGray(formatArguments(arguments)))
	return nil
}

// ShowToolResult displays the result of a tool call
func (d *NonInteractiveDisplay) ShowToolResult(result interface{}, duration time.Duration) error {
	fmt.Printf("✅ %s %s\n", 
		ColorGreen("Success"), 
		ColorGray(fmt.Sprintf("(%.2fs)", duration.Seconds())))
	
	// Show abbreviated result
	resultStr := fmt.Sprintf("%v", result)
	if len(resultStr) > 200 {
		resultStr = resultStr[:200] + "..."
	}
	fmt.Printf("   %s\n", ColorWhite(resultStr))
	return nil
}

// ShowError displays an error
func (d *NonInteractiveDisplay) ShowError(err error) error {
	fmt.Printf("❌ %s %s\n", ColorRed("Error"), err.Error())
	return nil
}

// ShowProgress displays a progress message
func (d *NonInteractiveDisplay) ShowProgress(message string) error {
	fmt.Printf("⏳ %s\n", ColorYellow(message))
	return nil
}

// ShowSummary displays execution summary
func (d *NonInteractiveDisplay) ShowSummary(summary ExecutionSummary) error {
	fmt.Println(ColorCyan("\n📊 Execution Summary"))
	fmt.Printf("   Total tools called: %d\n", summary.TotalTools)
	fmt.Printf("   Successful: %s, Failed: %s\n", 
		ColorGreen(fmt.Sprintf("%d", summary.SuccessfulCalls)),
		ColorRed(fmt.Sprintf("%d", summary.FailedCalls)))
	fmt.Printf("   Total duration: %s\n", ColorWhite(summary.TotalDuration.String()))
	
	if len(summary.ToolCalls) > 0 {
		fmt.Println(ColorGray("   Tool execution order:"))
		for i, call := range summary.ToolCalls {
			status := "✅"
			if !call.Success {
				status = "❌"
			}
			fmt.Printf("     %d. %s %s.%s (%.2fs)\n", 
				i+1, status, call.ServerName, call.ToolName, call.Duration.Seconds())
		}
	}
	return nil
}

// PromptToolApproval for non-interactive mode always returns true
func (d *NonInteractiveDisplay) PromptToolApproval(serverName, toolName string, arguments map[string]interface{}) (bool, error) {
	return true, nil
}

// ShowToolCall displays a tool call and prompts for approval in interactive mode
func (d *InteractiveDisplay) ShowToolCall(serverName, toolName string, arguments map[string]interface{}) error {
	fmt.Printf("🔧 %s: %s%s\n", 
		ColorBlue("Proposed Tool Call"), 
		ColorCyan(fmt.Sprintf("%s.%s", serverName, toolName)),
		ColorGray(formatArguments(arguments)))
	return nil
}

// ShowToolResult displays the result of a tool call
func (d *InteractiveDisplay) ShowToolResult(result interface{}, duration time.Duration) error {
	return (&NonInteractiveDisplay{}).ShowToolResult(result, duration)
}

// ShowError displays an error
func (d *InteractiveDisplay) ShowError(err error) error {
	return (&NonInteractiveDisplay{}).ShowError(err)
}

// ShowProgress displays a progress message
func (d *InteractiveDisplay) ShowProgress(message string) error {
	return (&NonInteractiveDisplay{}).ShowProgress(message)
}

// ShowSummary displays execution summary
func (d *InteractiveDisplay) ShowSummary(summary ExecutionSummary) error {
	return (&NonInteractiveDisplay{}).ShowSummary(summary)
}

// PromptToolApproval prompts user for tool execution approval
func (d *InteractiveDisplay) PromptToolApproval(serverName, toolName string, arguments map[string]interface{}) (bool, error) {
	fmt.Printf("\n%s\n", ColorYellow("🤔 TOOL EXECUTION APPROVAL REQUIRED"))
	fmt.Printf("Tool: %s\n", ColorCyan(fmt.Sprintf("%s.%s", serverName, toolName)))
	if len(arguments) > 0 {
		fmt.Printf("Arguments: %s\n", formatArguments(arguments))
	}
	fmt.Printf("\n%s\n", ColorWhite("Options:"))
	fmt.Printf("  %s - Execute this tool\n", ColorGreen("[y]es"))
	fmt.Printf("  %s - Skip this tool only\n", ColorYellow("[n]o"))
	fmt.Printf("  %s - Auto-approve all remaining tools\n", ColorGreen("[s]kip prompts"))
	fmt.Printf("  %s - Abort entire session\n", ColorRed("[a]bort"))
	fmt.Printf("\n%s ", ColorBold("Your choice:"))
	
	// Ensure output is flushed before reading input
	os.Stdout.Sync()
	
	response, err := d.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read user input: %w", err)
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	switch response {
	case "y", "yes", "":
		fmt.Printf("%s\n", ColorGreen("✅ Tool approved - executing..."))
		return true, nil
	case "n", "no":
		fmt.Printf("%s\n", ColorYellow("⏭️ Tool skipped"))
		return false, nil
	case "s", "skip", "skip all":
		fmt.Printf("%s\n", ColorGreen("✅ Auto-approve mode activated - all remaining tools will be executed"))
		return false, fmt.Errorf("AUTO_APPROVE_ALL")
	case "a", "abort":
		fmt.Printf("%s\n", ColorRed("🛑 Session aborted by user"))
		return false, fmt.Errorf("ABORT")
	default:
		fmt.Printf("%s Please enter y, n, s, or a.\n", ColorRed("❌ Invalid response."))
		return d.PromptToolApproval(serverName, toolName, arguments)
	}
}

// formatArguments formats tool arguments for display
func formatArguments(arguments map[string]interface{}) string {
	if len(arguments) == 0 {
		return ""
	}
	
	var parts []string
	for key, value := range arguments {
		valueStr := fmt.Sprintf("%v", value)
		if len(valueStr) > 50 {
			valueStr = valueStr[:50] + "..."
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, valueStr))
	}
	
	return fmt.Sprintf(" (%s)", strings.Join(parts, ", "))
}
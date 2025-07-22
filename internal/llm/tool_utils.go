package llm

import (
	"encoding/json"
	"fmt"
	"github.com/iteasy-ops-dev/syseng-agent/internal/ui"
)

// ToolProcessor handles common tool processing logic
type ToolProcessor struct {
	tools      []Tool
	toolCaller ToolCaller
}

// NewToolProcessor creates a new tool processor
func NewToolProcessor(tools []Tool, toolCaller ToolCaller) *ToolProcessor {
	return &ToolProcessor{
		tools:      tools,
		toolCaller: toolCaller,
	}
}

// FindTool finds a tool by name
func (tp *ToolProcessor) FindTool(name string) *Tool {
	for _, tool := range tp.tools {
		if tool.Function.Name == name {
			return &tool
		}
	}
	return nil
}

// ValidateToolCall validates if a tool call is valid
func (tp *ToolProcessor) ValidateToolCall(name string, args map[string]interface{}) error {
	tool := tp.FindTool(name)
	if tool == nil {
		return fmt.Errorf("tool %s not found", name)
	}
	
	// Basic validation - could be enhanced with schema validation
	if tool.Function.Parameters != nil {
		required, ok := tool.Function.Parameters["required"].([]interface{})
		if ok {
			for _, req := range required {
				if reqStr, ok := req.(string); ok {
					if _, exists := args[reqStr]; !exists {
						return fmt.Errorf("required parameter %s missing for tool %s", reqStr, name)
					}
				}
			}
		}
	}
	
	return nil
}

// ExecuteTool executes a tool call with validation
func (tp *ToolProcessor) ExecuteTool(name string, args map[string]interface{}) (interface{}, error) {
	if err := tp.ValidateToolCall(name, args); err != nil {
		return nil, err
	}
	
	return tp.toolCaller(name, args)
}

// ProcessToolCalls processes multiple tool calls
func (tp *ToolProcessor) ProcessToolCalls(calls []ToolCall) ([]ToolCallResult, error) {
	results := make([]ToolCallResult, len(calls))
	
	for i, call := range calls {
		// Parse arguments string to map
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err != nil {
			results[i] = ToolCallResult{
				ID:    call.ID,
				Error: fmt.Errorf("failed to parse arguments: %v", err),
			}
			continue
		}
		
		result, err := tp.ExecuteTool(call.Function.Name, args)
		
		results[i] = ToolCallResult{
			ID:     call.ID,
			Result: result,
			Error:  err,
		}
	}
	
	return results, nil
}

// FormatToolResult formats tool result for LLM consumption
func FormatToolResult(result interface{}) string {
	if result == nil {
		return "null"
	}
	
	// Try to convert to JSON for structured data
	if jsonBytes, err := json.Marshal(result); err == nil {
		return string(jsonBytes)
	}
	
	// Fallback to string representation
	return fmt.Sprintf("%v", result)
}

// WrapToolCallerWithUI wraps a tool caller to add UI feedback
func WrapToolCallerWithUI(toolCaller ToolCaller, display ui.ToolDisplayInterface) ToolCaller {
	return func(name string, args map[string]interface{}) (interface{}, error) {
		// Show tool call in UI
		display.ShowToolCall("MCP", name, args)
		
		// Execute the tool
		result, err := toolCaller(name, args)
		
		if err != nil {
			display.ShowError(fmt.Errorf("Tool execution failed: %v", err))
			return nil, err
		}
		
		display.ShowToolResult(result, 0) // Duration will be calculated elsewhere
		return result, nil
	}
}

// Tool call result for processing multiple tool calls
type ToolCallResult struct {
	ID     string
	Result interface{}
	Error  error
}

// EnhanceMessageWithTools adds tool context to a message
func EnhanceMessageWithTools(message string, toolCount int) string {
	if toolCount == 0 {
		return message
	}
	
	enhanced := message + "\n\n"
	enhanced += fmt.Sprintf("Note: You have access to %d tools for system operations. ", toolCount)
	enhanced += "Use them when appropriate to help with the user's request."
	
	return enhanced
}